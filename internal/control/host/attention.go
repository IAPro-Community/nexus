package host

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

const (
	PromptKindNone     = "none"
	PromptKindYN       = "yn"
	PromptKindChoice   = "choice"
	PromptKindFreeText = "free_text"

	AttentionNeedsUser = "needs_user"
	AttentionWorking   = "working"
	AttentionCompleted = "completed"
	AttentionError     = "error"
	AttentionIdle      = "idle"
)

var (
	oscTitleRegex = regexp.MustCompile(`\x1b\][02];([^\x07\x1b]+)(?:\x07|\x1b\\)`)

	chromeLinePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^\?\s*for\s+shortcuts\s*$`),
		regexp.MustCompile(`(?i)^press\s+.+\s+for\s+shortcuts\s*$`),
		regexp.MustCompile(`(?i)^\[?[yn]/?[yn]\]?\s*$`),
		regexp.MustCompile(`(?i)^\([yn]/[yn]\)\s*$`),
	}

	ynTokenPattern = regexp.MustCompile(`(?i)(\[[yY]/[nN]\]|\([yY]/[nN]\)|\byes\s*/\s*no\b|\by\s*/\s*n\b)`)

	questionPhrasePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)do you want to proceed\?`),
		regexp.MustCompile(`(?i)por favor,?\s*revise o plano`),
		regexp.MustCompile(`(?i)artifact\s+to\s+review`),
		regexp.MustCompile(`(?i)select an option|escolha uma op[çc][ãa]o`),
		regexp.MustCompile(`(?i)waiting for (?:user )?input|aguardando (?:resposta|confirma[çc][ãa]o)`),
		regexp.MustCompile(`(?i)are you sure\?|tem certeza\?`),
		regexp.MustCompile(`(?i)approve\?|aprovar\?`),
		regexp.MustCompile(`(?i)deseja\s+.+\?`),
		regexp.MustCompile(`(?i)would you like\s+.+\?`),
		regexp.MustCompile(`(?i)continue\?\s*$`),
	}

	choicePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)choice\s*\[\d+-\d+\]:`),
		regexp.MustCompile(`(?i)enter\s+(?:a\s+)?(?:number|option)\s*[:=]`),
	}

	// AGY/Antigravity ask_question TUI (questionnaires), not generic shell prompts.
	questionnairePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bask_question\b`),
		regexp.MustCompile(`(?i)question\s+\d+\s+of\s+\d+`),
		regexp.MustCompile(`(?i)pergunta\s+\d+\s+de\s+\d+`),
		regexp.MustCompile(`(?i)question[aá]rio`),
		regexp.MustCompile(`(?i)select all that apply`),
		regexp.MustCompile(`(?i)press\s+(?:space|espaço).{0,40}toggle`),
		regexp.MustCompile(`(?i)space to (?:select|toggle)`),
	}

	// Numbered bullets alone are agent narration, not a choice prompt.
	numberedOptionLine = regexp.MustCompile(`(?i)^\s*\d+[\).]\s+\S+`)

	taskCompletedPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^task\s+(?:completed|finished|conclu[ií]da)`),
		regexp.MustCompile(`(?i)^successfully\s+(?:created|updated|built|installed|completed|generated)\b`),
		regexp.MustCompile(`(?i)^all\s+\d+\s+tests?\s+pass`),
		regexp.MustCompile(`(?i)^tudo pronto!|^sucesso!|^finalizado com sucesso`),
	}

	workingPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)thinking\.\.\.|pensando\.\.\.`),
		regexp.MustCompile(`(?i)running command|executando ação`),
		regexp.MustCompile(`(?i)analyzing|analisando`),
		regexp.MustCompile(`(?i)generating|gerando`),
		regexp.MustCompile(`(?i)searching|pesquisando`),
	}

	// Live process failures only: anchored at start of the useful line so
	// agent narration quoting "CreateProcessW failed: …" or checklist rows
	// (FC-01 | … | STILL_PRESENT | CI run …) does not OS-notify.
	errorPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^error:\s+\S+`),
		regexp.MustCompile(`(?i)^failed:\s+\S+`),
		regexp.MustCompile(`(?i)^exited with code [1-9]`),
	}

	errorCitationNoise = []*regexp.Regexp{
		regexp.MustCompile("`"),
		regexp.MustCompile(`\|`),
		regexp.MustCompile(`(?i)\bFC-\d+\b`),
		regexp.MustCompile(`(?i)\bSTILL_PRESENT\b`),
		regexp.MustCompile(`(?i)\bCI\s+run\b`),
	}
)

// ExtractOSCTitle finds the newest OSC 0/2 window title sequence in the raw chunk.
func ExtractOSCTitle(raw string) string {
	matches := oscTitleRegex.FindAllStringSubmatch(raw, -1)
	if len(matches) == 0 {
		return ""
	}
	last := matches[len(matches)-1]
	if len(last) > 1 {
		return strings.TrimSpace(last[1])
	}
	return ""
}

// AttentionDetector monitors stream chunks for a runtime session and emits dynamic updates.
// PTY regex is only a TERMINAL-mode fallback: when ControlLevel is EVENTS/CONTROL_API
// (structured provider events), ProcessChunk is a no-op for classification.
type AttentionDetector struct {
	mu               sync.Mutex
	runtimeID        string
	providerID       string
	profileID        string
	workspace        string
	projectID        string
	projectName      string
	agentID          string
	controlLevel     registry.ControlLevel
	structuredEvents bool
	buffer           strings.Builder
	lastState        registry.RuntimeState
	lastReason       string
	lastKind         string
	lastPrompt       string
	lastTitle        string
	lastOSCTitle     string
	lastFingerprint  string
	lastUpdateAt     time.Time
	registry         *registry.Registry
	onAttention      func(reason, context, dynamicTitle string, state registry.RuntimeState)
}

// NewAttentionDetector creates an attention detector for a session.
func NewAttentionDetector(runtimeID, providerID, profileID, workspace string, onAttention func(reason, context, dynamicTitle string, state registry.RuntimeState)) *AttentionDetector {
	return NewAttentionDetectorWithProject(runtimeID, providerID, profileID, workspace, "", "", onAttention)
}

// NewAttentionDetectorWithProject creates a detector with durable project identity.
func NewAttentionDetectorWithProject(runtimeID, providerID, profileID, workspace, projectID, projectName string, onAttention func(reason, context, dynamicTitle string, state registry.RuntimeState)) *AttentionDetector {
	name := strings.TrimSpace(projectName)
	if name == "" {
		name = filepath.Base(workspace)
	}
	if name == "" || name == "." || name == "/" {
		name = "Project"
	}

	return &AttentionDetector{
		runtimeID:    runtimeID,
		providerID:   providerID,
		profileID:    profileID,
		workspace:    workspace,
		projectID:    projectID,
		projectName:  name,
		controlLevel: registry.ControlLevelTerminal, // default: honest TERMINAL fallback
		lastState:    registry.StateRunning,
		lastKind:     AttentionIdle,
		lastPrompt:   PromptKindNone,
		onAttention:  onAttention,
	}
}

// SetControlPolicy configures whether PTY heuristics may emit agent attention events.
// StructuredEvents=true (or ControlLevel above TERMINAL) disables stdout scraping.
func (d *AttentionDetector) SetControlPolicy(level registry.ControlLevel, structuredEvents bool, agentID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if level != "" {
		d.controlLevel = level
	}
	d.structuredEvents = structuredEvents
	d.agentID = strings.TrimSpace(agentID)
}

// SetRegistry binds durable attention updates to the owning runtime registry.
// The constructor remains backwards-compatible for standalone detector tests;
// those callers fall back to the process registry when no owner is supplied.
func (d *AttentionDetector) SetRegistry(reg *registry.Registry) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.registry = reg
}

func (d *AttentionDetector) ownerRegistry() *registry.Registry {
	if d.registry != nil {
		return d.registry
	}
	return registry.DefaultRegistry()
}

// isShellProvider reports Project Shell / non-agent PTYs that must not emit
// interactive attention (needs_user, completed toasts, OS notify).
func isShellProvider(providerID string) bool {
	switch strings.ToLower(strings.TrimSpace(providerID)) {
	case "shell", "project-shell", "project_shell":
		return true
	default:
		return false
	}
}

// ptyHeuristicAllowed is true only for AI agents supervised in TERMINAL mode
// without a structured event adapter. Shell and EVENTS/API levels never scrape.
func ptyHeuristicAllowed(providerID string, level registry.ControlLevel, structuredEvents bool) bool {
	if isShellProvider(providerID) || structuredEvents {
		return false
	}
	switch level {
	case "", registry.ControlLevelTerminal:
		return true
	default:
		return false
	}
}

// ProcessChunk ingests terminal output chunk, analyzes state, and triggers updates when state changes.
func (d *AttentionDetector) ProcessChunk(chunk []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()

	raw := string(chunk)
	oscTitle := ExtractOSCTitle(raw)
	if oscTitle != "" && !isChromeLine(oscTitle) {
		d.applyOSCTitleLocked(oscTitle)
	}

	heuristic := ptyHeuristicAllowed(d.providerID, d.controlLevel, d.structuredEvents)
	shellQuestionnaire := isShellProvider(d.providerID) && !d.structuredEvents
	if !heuristic && !shellQuestionnaire {
		return
	}

	d.buffer.WriteString(raw)

	if d.buffer.Len() > 8192 {
		s := d.buffer.String()
		d.buffer.Reset()
		d.buffer.WriteString(s[len(s)-4096:])
	}

	cleanText := stripANSIKeepNewlines(d.buffer.String())
	recentLines := recentUsefulLines(cleanText, 2)
	lastLine := ""
	if len(recentLines) > 0 {
		lastLine = recentLines[len(recentLines)-1]
	}

	state := registry.StateRunning
	reason := ""
	kind := AttentionIdle
	promptKind := PromptKindNone
	attentionCtx := ""
	dynamicTitle := ""

	if shellQuestionnaire && !heuristic {
		if ctx, pk := classifyQuestionnaire(recentLines); ctx != "" && pk != PromptKindNone {
			state = registry.StateWaiting
			reason = "QUESTION"
			kind = AttentionNeedsUser
			promptKind = pk
			attentionCtx = truncateRunes(ctx, 200)
			dynamicTitle = oscTitle
			if dynamicTitle == "" {
				dynamicTitle = truncateRunes(attentionCtx, 72)
			}
		} else {
			return
		}
	} else {
		// Newest line wins: working/error/completed clear stale wait prompts
		// still present in the short lookback window.
		if lastLine != "" {
			for _, p := range workingPatterns {
				if p.MatchString(lastLine) {
					state = registry.StateRunning
					reason = "WORKING"
					kind = AttentionWorking
					attentionCtx = lastLine
					dynamicTitle = d.projectName + " · " + truncateRunes(attentionCtx, 48)
					break
				}
			}
		}

		if reason == "" && lastLine != "" && isLiveErrorLine(lastLine) {
			state = registry.StateFailed
			reason = "ERROR"
			kind = AttentionError
			attentionCtx = lastLine
			dynamicTitle = d.projectName + " · erro"
		}

		if reason == "" && lastLine != "" {
			for _, p := range taskCompletedPatterns {
				if p.MatchString(lastLine) {
					state = registry.StateRunning
					reason = "TASK_COMPLETED"
					kind = AttentionCompleted
					attentionCtx = lastLine
					dynamicTitle = d.projectName + " · concluído"
					break
				}
			}
		}

		if reason == "" {
			if ctx, pk := classifyNeedsUser(recentLines); ctx != "" && pk != PromptKindNone {
				state = registry.StateWaiting
				reason = "QUESTION"
				kind = AttentionNeedsUser
				promptKind = pk
				attentionCtx = truncateRunes(ctx, 200)
				dynamicTitle = d.projectName + " · " + truncateRunes(attentionCtx, 72)
			}
		}

		if dynamicTitle == "" {
			if oscTitle != "" && !isChromeLine(oscTitle) {
				dynamicTitle = oscTitle
				attentionCtx = oscTitle
				kind = AttentionWorking
			} else {
				dynamicTitle = d.projectName + " · " + strings.ToUpper(d.providerID)
				kind = AttentionIdle
			}
		}
	}

	fingerprint := ""
	if kind == AttentionNeedsUser || kind == AttentionError {
		fingerprint = attentionFingerprint(d.runtimeID, promptKind, attentionCtx)
	}

	// Same honest wait/error fingerprint = one radar/toast/OS/title mention.
	if fingerprint != "" && fingerprint == d.lastFingerprint && kind == d.lastKind {
		return
	}
	if state == d.lastState && reason == d.lastReason && kind == d.lastKind && promptKind == d.lastPrompt && dynamicTitle == d.lastTitle && time.Since(d.lastUpdateAt) < 3*time.Second {
		return
	}

	prevFingerprint := d.lastFingerprint
	d.lastState = state
	d.lastReason = reason
	d.lastKind = kind
	d.lastPrompt = promptKind
	d.lastTitle = dynamicTitle
	if fingerprint != "" {
		d.lastFingerprint = fingerprint
	} else if kind != AttentionNeedsUser && kind != AttentionError {
		d.lastFingerprint = ""
	}
	d.lastUpdateAt = time.Now()

	_ = d.ownerRegistry().UpdateAttentionMeta(d.runtimeID, registry.AttentionUpdate{
		State:        state,
		Reason:       reason,
		Context:      attentionCtx,
		PromptKind:   promptKind,
		Kind:         kind,
		Fingerprint:  fingerprint,
		ProjectID:    d.projectID,
		ProjectName:  d.projectName,
		TaskSummary:  attentionCtx,
		DynamicTitle: dynamicTitle,
	})

	if kind == AttentionNeedsUser && fingerprint != "" && fingerprint != prevFingerprint {
		// Browser/UI consume the bus + registry poll. Skip OS notify here to
		// avoid duplicate desktop + browser toasts for the same question.
		data := map[string]any{
			"project_id":       d.projectID,
			"project_name":     d.projectName,
			"attention_reason": reason,
			"attention_kind":   kind,
			"prompt_kind":      promptKind,
			"context":          attentionCtx,
			"dynamic_title":    dynamicTitle,
			"fingerprint":      fingerprint,
			"source":           "pty_heuristic",
		}
		if d.agentID != "" {
			data["agent_id"] = d.agentID
		}
		events.DefaultBus().Publish(events.NewEvent(
			d.runtimeID,
			d.providerID,
			d.profileID,
			events.EventApprovalRequired,
			"["+d.projectName+"] "+attentionCtx,
			data,
		))
	} else if kind == AttentionError && fingerprint != "" && fingerprint != prevFingerprint {
		_ = notify.Default().Notify(notify.Payload{
			Title: "Nexus · " + d.projectName,
			Body:  "Erro no terminal: " + attentionCtx + " — Abra o Nexus para revisar.",
			Tag:   fingerprint,
		})
	} else if reason == "TASK_COMPLETED" {
		data := map[string]any{
			"project_id":       d.projectID,
			"project_name":     d.projectName,
			"attention_reason": reason,
			"attention_kind":   kind,
			"context":          attentionCtx,
			"dynamic_title":    dynamicTitle,
			"source":           "pty_heuristic",
		}
		if d.agentID != "" {
			data["agent_id"] = d.agentID
		}
		events.DefaultBus().Publish(events.NewEvent(
			d.runtimeID,
			d.providerID,
			d.profileID,
			events.EventToolFinished,
			"["+d.projectName+"] "+attentionCtx,
			data,
		))
	}

	if d.onAttention != nil {
		d.onAttention(reason, attentionCtx, dynamicTitle, state)
	}
}

// applyOSCTitleLocked records the raw terminal settitle (OSC 0/2) without
// scraping agent narration or clearing an active wait/error.
func (d *AttentionDetector) applyOSCTitleLocked(title string) {
	if title == "" || title == d.lastOSCTitle {
		return
	}
	d.lastOSCTitle = title
	_ = d.ownerRegistry().UpdateTitle(d.runtimeID, title)
	if d.onAttention != nil && d.lastKind != AttentionNeedsUser && d.lastKind != AttentionError {
		d.onAttention(d.lastReason, title, title, d.lastState)
	}
}

func classifyQuestionnaire(lines []string) (context string, promptKind string) {
	if len(lines) == 0 {
		return "", PromptKindNone
	}
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || isChromeLine(line) || isGarbageAttentionText(line) {
			continue
		}
		for _, p := range questionnairePatterns {
			if p.MatchString(line) {
				return line, PromptKindChoice
			}
		}
	}
	return "", PromptKindNone
}

func classifyNeedsUser(lines []string) (context string, promptKind string) {
	if ctx, pk := classifyQuestionnaire(lines); ctx != "" {
		return ctx, pk
	}
	if len(lines) == 0 {
		return "", PromptKindNone
	}
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if isChromeLine(line) || isGarbageAttentionText(line) {
			continue
		}
		filtered = append(filtered, line)
	}
	if len(filtered) == 0 {
		return "", PromptKindNone
	}

	last := filtered[len(filtered)-1]
	prev := ""
	if len(filtered) > 1 {
		prev = filtered[len(filtered)-2]
	}

	// yn only when a real question phrase shares the line or the line above.
	if ynTokenPattern.MatchString(last) {
		ask := last
		if !hasQuestionContent(last) && prev != "" && hasQuestionContent(prev) {
			ask = prev + " " + last
		}
		if hasQuestionContent(ask) && !isGarbageAttentionText(ask) {
			return strings.TrimSpace(ask), PromptKindYN
		}
		return "", PromptKindNone
	}

	for _, p := range choicePatterns {
		if p.MatchString(last) || (prev != "" && p.MatchString(prev)) {
			ctx := last
			if prev != "" && !hasQuestionContent(last) {
				ctx = prev + "\n" + last
			}
			if isGarbageAttentionText(ctx) {
				return "", PromptKindNone
			}
			return strings.TrimSpace(ctx), PromptKindChoice
		}
	}

	// Numbered list only counts as choice when paired with an explicit ask above.
	if numberedOptionLine.MatchString(last) {
		if prev != "" && (strings.Contains(prev, "?") || looksLikeChoicePrompt(prev) || matchesQuestionPhrase(prev)) {
			ctx := strings.TrimSpace(prev + "\n" + last)
			if !isGarbageAttentionText(ctx) {
				return ctx, PromptKindChoice
			}
		}
		return "", PromptKindNone
	}

	for _, p := range questionPhrasePatterns {
		if p.MatchString(last) {
			if isGarbageAttentionText(last) {
				return "", PromptKindNone
			}
			return last, PromptKindFreeText
		}
		if prev != "" && p.MatchString(prev) {
			ctx := strings.TrimSpace(prev + "\n" + last)
			if isGarbageAttentionText(ctx) {
				return "", PromptKindNone
			}
			return ctx, PromptKindFreeText
		}
	}

	// Free-text wait only when the last useful line is a clear interrogative sentence.
	if looksLikeQuestionSentence(last) && !isGarbageAttentionText(last) {
		return last, PromptKindFreeText
	}
	return "", PromptKindNone
}

func looksLikeChoicePrompt(line string) bool {
	trimmed := strings.TrimSpace(line)
	for _, p := range choicePatterns {
		if p.MatchString(trimmed) {
			return true
		}
	}
	lower := strings.ToLower(trimmed)
	return strings.Contains(lower, "select") ||
		strings.Contains(lower, "escolha") ||
		strings.Contains(lower, "choose") ||
		strings.Contains(lower, "option")
}

func matchesQuestionPhrase(line string) bool {
	for _, p := range questionPhrasePatterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}

// isGarbageAttentionText rejects TUI noise that is not a readable human prompt
// (UTF-8 replacement diamonds, box-drawing chrome). Readable short prompts
// (e.g. "OK? [y/N]") are left for the other classifiers.
func isGarbageAttentionText(s string) bool {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return true
	}
	letters := 0
	garbage := 0
	for _, r := range trimmed {
		switch {
		case unicode.IsLetter(r):
			letters++
		case r == '\uFFFD':
			garbage++
		case r >= 0x2500 && r <= 0x257F: // box drawing
			garbage++
		case r >= 0x2580 && r <= 0x259F: // block elements
			garbage++
		case r >= 0x25A0 && r <= 0x25FF: // geometric shapes often used as TUI chrome
			garbage++
		}
	}
	if garbage == 0 {
		return false
	}
	if letters < 8 {
		return true
	}
	if garbage >= letters/2 {
		return true
	}
	return false
}

func recentUsefulLines(cleanText string, limit int) []string {
	lines := strings.Split(cleanText, "\n")
	out := make([]string, 0, limit)
	for i := len(lines) - 1; i >= 0 && len(out) < limit; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || isChromeLine(trimmed) {
			continue
		}
		out = append([]string{trimmed}, out...)
	}
	return out
}

func isChromeLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	for _, p := range chromeLinePatterns {
		if p.MatchString(trimmed) {
			return true
		}
	}
	return false
}

// isLiveErrorLine reports whether the last useful PTY line is a real process
// failure (error:/failed:/exited with code N) rather than agent narration that
// quotes those tokens inside backticks, markdown tables, or CI checklist rows.
func isLiveErrorLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	for _, p := range errorCitationNoise {
		if p.MatchString(trimmed) {
			return false
		}
	}
	for _, p := range errorPatterns {
		if p.MatchString(trimmed) {
			return true
		}
	}
	return false
}

func hasQuestionContent(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	withoutYN := ynTokenPattern.ReplaceAllString(trimmed, " ")
	withoutYN = strings.TrimSpace(withoutYN)
	if withoutYN == "" {
		return false
	}
	letters := 0
	for _, r := range withoutYN {
		if unicode.IsLetter(r) {
			letters++
		}
	}
	return letters >= 8 || strings.Contains(withoutYN, "?")
}

func looksLikeQuestionSentence(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasSuffix(trimmed, "?") {
		return false
	}
	if isChromeLine(trimmed) {
		return false
	}
	letters := 0
	for _, r := range trimmed {
		if unicode.IsLetter(r) {
			letters++
		}
	}
	return letters >= 12
}

func truncateRunes(s string, max int) string {
	runes := []rune(strings.TrimSpace(s))
	if max <= 0 || len(runes) <= max {
		return string(runes)
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func attentionFingerprint(runtimeID, promptKind, context string) string {
	norm := strings.ToLower(strings.Join(strings.Fields(context), " "))
	sum := sha256.Sum256([]byte(runtimeID + "|" + promptKind + "|" + norm))
	return hex.EncodeToString(sum[:8])
}

func formatDesktopAttentionBody(promptKind, context string) string {
	ctx := strings.Join(strings.Fields(strings.TrimSpace(context)), " ")
	if ctx == "" {
		return ""
	}
	lead := "Um agente espera sua resposta"
	switch promptKind {
	case PromptKindYN:
		lead = "Um agente pede confirmação (Sim/Não)"
	case PromptKindChoice:
		lead = "Um agente pede uma escolha"
	}
	return lead + ": " + ctx + " — Abra o Nexus e responda no terminal."
}

// stripANSIKeepNewlines removes CSI/OSC chrome but keeps line breaks so the
// classifier can inspect only the last useful lines fail-closed.
func stripANSIKeepNewlines(str string) string {
	var b strings.Builder
	inEsc := false
	for i := 0; i < len(str); i++ {
		c := str[i]
		if c == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '~' {
				inEsc = false
			}
			continue
		}
		if c == '\n' || c == '\t' || c >= 32 {
			b.WriteByte(c)
		}
		// drop other controls including \r
	}
	return b.String()
}
