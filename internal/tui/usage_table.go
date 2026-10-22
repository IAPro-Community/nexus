package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/conversation"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/profile"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// ExecutionMode defines the operating mode for launched runtimes.
type ExecutionMode int

const (
	ModeSafe ExecutionMode = iota
	ModeYOLO
	ModePlan
)

func (m ExecutionMode) String() string {
	switch m {
	case ModeYOLO:
		return "YOLO"
	case ModePlan:
		return "Plan"
	default:
		return "Safe"
	}
}

func modeFlags(mode ExecutionMode, continueSession bool) []string {
	var flags []string
	switch mode {
	case ModeYOLO:
		flags = append(flags, "--yolo")
	case ModePlan:
		flags = append(flags, "--plan")
	}
	if continueSession {
		flags = append(flags, "--continue")
	}
	return flags
}

// ActiveTab defines which view is displayed in the main table area.
type ActiveTab int

const (
	TabAccounts ActiveTab = iota
	TabSessions
)

// ModalMode defines the overlay modal currently open.
type ModalMode int

const (
	ModalNone ModalMode = iota
	ModalQuotaDetails
	ModalResumeChoice
	ModalAddProfile
)

// InstalledProviderInfo describes an AI CLI detected on the system.
type InstalledProviderInfo struct {
	ID        string
	Name      string
	Version   string
	Installed bool
	Profiles  int
}

// UsageTableRow is the presentation representation of an account quota window or provider row.
type UsageTableRow struct {
	Provider    string
	Profile     string
	Account     string
	Plan        string
	Group       string
	FiveHour    string
	Weekly      string
	Status      string
	ModelName   string
	LastUpdated string
	// Source is the UsageSource that produced this row (OFFICIAL_API, OBSERVATION, …).
	Source string
	// SnapshotStatus is the raw UsageStatus behind the display label in Status.
	SnapshotStatus string
	// FetchedAt is when the underlying observation was taken.
	FetchedAt      time.Time
	IsUnconfigured bool
	IsDefault      bool
}

// UnifiedUsageOptions holds all context needed to run the unified TUI.
type UnifiedUsageOptions struct {
	Rows             []UsageTableRow
	UnconfiguredCLIs []InstalledProviderInfo
	Sessions         []conversation.Conversation
	Accounts         map[string]model.AccountInfo
	Defaults         map[string]string // provider -> default profile
	Workspace        string
	InitialMode      ExecutionMode
	InitialContinue  bool
	// ProfilesToLoad triggers async quota fetch when non-empty.
	ProfilesToLoad []model.Profile
	// LoadSessionsAsync defers conversation.ListRecent until after the TUI opens.
	LoadSessionsAsync bool
}

// ActionType defines the user's intent upon exiting the TUI.
type ActionType int

const (
	ActionNone ActionType = iota
	ActionRunProfile
	ActionResumeConversation
	ActionSetDefault
	ActionLogin
	ActionAdd
	ActionQuit
)

// SelectionResult is returned when the user chooses an action.
type SelectionResult struct {
	Action         ActionType
	Provider       string
	ProfileName    string
	ConversationID string
	Args           []string
}

type usageTableModel struct {
	options UnifiedUsageOptions

	// State
	activeTab       ActiveTab
	execMode        ExecutionMode
	continueSession bool
	modalMode       ModalMode
	statusMsg       string
	loadingQuotas   bool
	loadingSessions bool
	spinFrame       int
	cachedQuotaView map[string]quotaViewCache
	// refreshing tracks provider:profile keys with an in-flight on-demand refresh.
	refreshing map[string]bool
	// Scheduled Codex probes continue after the initial cached-first load,
	// including when an interactive TUI releases its profile lock later.
	lastCodexProbe time.Time

	// Accounts table
	allAccounts      []UsageTableRow
	filteredAccounts []UsageTableRow
	accountTable     table.Model

	// Sessions table
	allSessions      []conversation.Conversation
	filteredSessions []conversation.Conversation
	sessionTable     table.Model

	// Filter input
	filter textinput.Model

	// Add profile modal
	addProfileProvider string
	addProfileInput    textinput.Model

	// Selection & lifecycle
	chosenResult *SelectionResult
	quitting     bool
	width        int
	height       int
}

type quotaViewCache struct {
	view quota.QuotaView
	acc  model.AccountInfo
}

type quotasLoadedMsg struct {
	rows     []UsageTableRow
	accounts map[string]model.AccountInfo
	silent   bool
}

type sessionsLoadedMsg struct {
	sessions []conversation.Conversation
}

type spinnerTickMsg struct{}

type quotaPollTickMsg struct{}

// accountRefreshedMsg carries the result of an on-demand refresh of one account.
type accountRefreshedMsg struct {
	provider string
	profile  string
	rows     []UsageTableRow
	acc      model.AccountInfo
}

// accountsRefreshedMsg carries the result of an on-demand refresh of all accounts.
type accountsRefreshedMsg struct {
	results []accountRefreshedMsg
}

const quotaPollInterval = 60 * time.Second

// staleQuotaAge is the threshold after which a cached quota reading is
// considered stale and an age suffix is appended to the STATUS column.
const staleQuotaAge = 45 * time.Minute

// Recover a current quota off the input loop. Avoid hammering the Codex
// app-server when a CLI session is active or the provider is unavailable.
const codexAutoRefreshInterval = 5 * time.Minute

var codexAutoRefreshState = struct {
	sync.Mutex
	attempted map[string]time.Time
}{attempted: make(map[string]time.Time)}

func codexNeedsAutoRefresh(snap model.UsageSnapshot, now time.Time) bool {
	if snap.FetchedAt.IsZero() {
		return true
	}
	age := now.Sub(snap.FetchedAt)
	if age < -time.Minute || age > quota.DefaultTTL {
		return true
	}
	switch snap.Status {
	case model.UsageLive, model.UsageCached, model.UsageRateLimited:
		return false
	default:
		return true
	}
}

func claimCodexAutoRefresh(provider, name string, now time.Time) bool {
	if provider != "codex" {
		return false
	}
	key := provider + ":" + name
	codexAutoRefreshState.Lock()
	defer codexAutoRefreshState.Unlock()
	if last := codexAutoRefreshState.attempted[key]; !last.IsZero() && now.Sub(last) < codexAutoRefreshInterval {
		return false
	}
	codexAutoRefreshState.attempted[key] = now
	return true
}

func quotaPollTick() tea.Cmd {
	return tea.Tick(quotaPollInterval, func(time.Time) tea.Msg {
		return quotaPollTickMsg{}
	})
}

func loadQuotasCmd(profiles []model.Profile, defaults map[string]string) tea.Cmd {
	return loadQuotasCmdOpts(profiles, defaults, false)
}

func loadQuotasCmdOpts(profiles []model.Profile, defaults map[string]string, silent bool) tea.Cmd {
	return func() tea.Msg {
		type result struct {
			rows []UsageTableRow
			accs map[string]model.AccountInfo
		}
		out := result{accs: make(map[string]model.AccountInfo)}
		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, 4)
		for _, p := range profiles {
			wg.Add(1)
			go func(p model.Profile) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				// This worker is asynchronous. Quota probing never executes on
				// the Bubble Tea input loop or in the rendering path.
				if p.Provider == "codex" {
					snap := profile.GetUsageSnapshot(p.Provider, p.Name)
					now := time.Now()
					if codexNeedsAutoRefresh(snap, now) && claimCodexAutoRefresh(p.Provider, p.Name, now) {
						_ = profile.RefreshUsageSnapshot(p.Provider, p.Name)
					}
				}
				acc := profile.GetAccountInfo(p.Provider, p.Name)
				qv := profile.GetQuotaView(p.Provider, p.Name, acc.Plan, acc.Email)
				rows := buildUsageRows(p.Provider, p.Name, qv, acc, defaults)
				mu.Lock()
				out.accs[p.Provider+":"+p.Name] = acc
				out.rows = append(out.rows, rows...)
				mu.Unlock()
			}(p)
		}
		wg.Wait()
		return quotasLoadedMsg{rows: out.rows, accounts: out.accs, silent: silent}
	}
}

// refreshAccountCmd re-reads one account's quota off the Bubble Tea loop so the
// UI stays responsive while the provider is queried.
func refreshAccountCmd(provider, profileName string, defaults map[string]string) tea.Cmd {
	return func() tea.Msg {
		profile.RefreshUsageSnapshot(provider, profileName)
		acc := profile.GetAccountInfo(provider, profileName)
		qv := profile.GetQuotaView(provider, profileName, acc.Plan, acc.Email)
		return accountRefreshedMsg{
			provider: provider,
			profile:  profileName,
			rows:     buildUsageRows(provider, profileName, qv, acc, defaults),
			acc:      acc,
		}
	}
}

// refreshAllCmd forces a live refresh of every configured profile concurrently.
func refreshAllCmd(profiles []model.Profile, defaults map[string]string) tea.Cmd {
	return func() tea.Msg {
		var (
			mu      sync.Mutex
			wg      sync.WaitGroup
			results []accountRefreshedMsg
			sem     = make(chan struct{}, 4)
		)
		for _, p := range profiles {
			wg.Add(1)
			go func(p model.Profile) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				profile.RefreshUsageSnapshot(p.Provider, p.Name)
				acc := profile.GetAccountInfo(p.Provider, p.Name)
				qv := profile.GetQuotaView(p.Provider, p.Name, acc.Plan, acc.Email)
				mu.Lock()
				results = append(results, accountRefreshedMsg{
					provider: p.Provider,
					profile:  p.Name,
					rows:     buildUsageRows(p.Provider, p.Name, qv, acc, defaults),
					acc:      acc,
				})
				mu.Unlock()
			}(p)
		}
		wg.Wait()
		return accountsRefreshedMsg{results: results}
	}
}

// buildUsageRows renders one presentation row per model group of a quota view.
// BuildUsageRowsForTest exposes buildUsageRows for cross-surface parity tests.
func BuildUsageRowsForTest(provider, profileName string, qv quota.QuotaView, acc model.AccountInfo, defaults map[string]string) []UsageTableRow {
	return buildUsageRows(provider, profileName, qv, acc, defaults)
}

func buildUsageRows(provider, profileName string, qv quota.QuotaView, acc model.AccountInfo, defaults map[string]string) []UsageTableRow {
	isDefault := defaults[provider] == profileName
	if len(qv.ModelGroups) == 0 {
		return []UsageTableRow{{
			Provider: provider, Profile: profileName, Account: acc.Email, Plan: acc.Plan,
			FiveHour: "—", Weekly: "—", Status: "SEM DADOS", ModelName: "—",
			Source: qv.Source, SnapshotStatus: qv.Status, FetchedAt: qv.FetchedAt,
			IsDefault: isDefault,
		}}
	}
	rows := make([]UsageTableRow, 0, len(qv.ModelGroups))
	for _, group := range qv.ModelGroups {
		fiveHour := formatQuotaWindow(group.Windows, "5h", qv.Status)
		weekly := formatQuotaWindow(group.Windows, "weekly", qv.Status)
		if fiveHour == "-" && weekly == "-" {
			label := unknownQuotaLabel(qv.Status)
			fiveHour, weekly = label, label
		}
		modelName := qv.ModelGroups[0].Name
		if len(qv.ModelGroups) > 1 {
			modelName = group.Name
		}
		lastUpdated := ""
		if !qv.FetchedAt.IsZero() {
			lastUpdated = quota.FormatFreshness(qv.FetchedAt)
		}
		rows = append(rows, UsageTableRow{
			Provider: provider, Profile: profileName, Account: acc.Email, Plan: acc.Plan,
			Group: group.Name, FiveHour: fiveHour, Weekly: weekly,
			Status: formatGroupStatus(group, qv.Status), ModelName: modelName,
			LastUpdated: lastUpdated, Source: qv.Source, SnapshotStatus: qv.Status,
			FetchedAt: qv.FetchedAt, IsDefault: isDefault,
		})
	}
	return rows
}

func loadSessionsCmd(workspace string) tea.Cmd {
	return func() tea.Msg {
		return sessionsLoadedMsg{sessions: conversation.ListRecent(30, workspace)}
	}
}

func spinnerTick() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return spinnerTickMsg{} })
}

func formatQuotaWindow(windows []quota.Window, kind, snapshotStatus string) string {
	for _, window := range windows {
		matches := kind == "5h" && (window.Kind == "5h" || window.Kind == "daily" || window.Kind == "claude_5h" || window.Kind == "claude_five_hour")
		if kind == "weekly" {
			matches = window.Kind == "weekly" || window.Kind == "claude_weekly"
		}
		if !matches {
			continue
		}
		reset := compactResetDesc(window.ResetDesc)
		switch snapshotStatus {
		case string(model.UsageEstimated):
			return fmt.Sprintf("~%2.0f%% · %s", window.Remaining, reset)
		// RATE_LIMITED from the official read carries exact numbers; showing them
		// is more useful than hiding the window behind a label.
		case string(model.UsageLive), string(model.UsageCached), string(model.UsageRateLimited):
			bar := miniQuotaBar(window.Remaining, 10)
			return fmt.Sprintf("%s %2.0f%% · %s", bar, window.Remaining, reset)
		default:
			return unknownQuotaLabel(snapshotStatus)
		}
	}
	return "—"
}

// sourceLabel names the origin of a quota reading for the operator.
func sourceLabel(source string) string {
	switch model.UsageSource(source) {
	case model.SourceOfficialAPI:
		return "API oficial"
	case model.SourceObservation:
		return "rollout local"
	case model.SourceCLIOutput:
		return "saída do CLI"
	case model.SourceLocalFiles:
		return "arquivos locais"
	case model.SourceResponseHeader:
		return "cabeçalhos HTTP"
	default:
		return "sem fonte"
	}
}

// statusLabelLong is the operator-facing name of a snapshot status.
func statusLabelLong(status string) string {
	switch model.UsageStatus(status) {
	case model.UsageLive:
		return "AO VIVO"
	case model.UsageCached:
		return "EM CACHE"
	case model.UsageEstimated:
		return "ESTIMADA"
	case model.UsageRateLimited:
		return "BLOQUEADA"
	case model.UsageError:
		return "ERRO"
	case model.UsageUnsupported:
		return "NÃO SUPORTADA"
	default:
		return "SEM DADOS"
	}
}

// compactAge renders the age of an observation in Portuguese short form.
func compactAge(at time.Time) string {
	if at.IsZero() {
		return "nunca"
	}
	d := time.Since(at)
	if d < 0 {
		d = 0
	}
	switch {
	case d < 15*time.Second:
		return "agora"
	case d < time.Minute:
		return fmt.Sprintf("há %ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("há %dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("há %dh", int(d.Hours()))
	default:
		return fmt.Sprintf("há %dd", int(d.Hours()/24))
	}
}

// freshnessLine summarizes status, origin and age in a single operator-readable line.
func freshnessLine(status, source string, at time.Time) string {
	return fmt.Sprintf("%s · %s · %s", statusLabelLong(status), sourceLabel(source), compactAge(at))
}

func compactResetDesc(reset string) string {
	reset = strings.TrimSpace(reset)
	if reset == "" {
		return "?"
	}
	if strings.EqualFold(reset, "Quota available") {
		return "disponível"
	}
	// AGY emits a complete English sentence; keep only the duration so the
	// Portuguese "Reset em" prefix in the detail modal does not double-speak.
	lower := strings.ToLower(reset)
	if strings.HasPrefix(lower, "refreshes in ") {
		return strings.TrimSpace(reset[len("Refreshes in "):])
	}
	reset = strings.TrimPrefix(reset, "resets ")
	reset = strings.TrimPrefix(reset, "Resets ")
	reset = strings.ReplaceAll(reset, " on ", " ")
	return reset
}

// formatResetLine renders a provider reset description for the detail modal.
func formatResetLine(reset string) string {
	compact := compactResetDesc(reset)
	if compact == "?" {
		return ""
	}
	if compact == "disponível" {
		return "Quota disponível"
	}
	return "Reset em " + compact
}

func miniQuotaBar(remaining float64, width int) string {
	if width < 4 {
		width = 4
	}
	filled := int((remaining/100.0)*float64(width) + 0.5)
	if remaining > 0 && filled == 0 {
		filled = 1
	}
	if filled > width {
		filled = width
	}
	if remaining <= 0 {
		filled = 0
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

func formatGroupStatus(group quota.ModelGroup, snapshotStatus string) string {
	switch snapshotStatus {
	case string(model.UsageUnknown), string(model.UsageError), string(model.UsageUnsupported), "":
		return "SEM DADOS"
	case string(model.UsageEstimated):
		return "ESTIMADA"
	case string(model.UsageRateLimited):
		return "RATE LTD"
	}
	knownWindows := 0
	allExhausted := true
	for _, window := range group.Windows {
		if window.Kind == "unknown" {
			continue
		}
		if window.ResetTime != nil && window.ResetTime.Before(time.Now()) {
			continue
		}
		knownWindows++
		if window.Remaining > 0 {
			allExhausted = false
		}
	}
	if knownWindows == 0 {
		return "SEM DADOS"
	}
	if allExhausted {
		return "ESGOTADA"
	}
	return "OK"
}

func unknownQuotaLabel(status string) string {
	switch status {
	case string(model.UsageUnknown):
		return "SEM DADOS"
	case string(model.UsageError):
		return "erro"
	case string(model.UsageRateLimited):
		return "rate ltd"
	default:
		return "—"
	}
}

// RunUnifiedUsage launches the modern, unified Quotas & Runtimes TUI.
func RunUnifiedUsage(opts UnifiedUsageOptions) (*SelectionResult, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		if len(opts.ProfilesToLoad) > 0 {
			msg := loadQuotasCmd(opts.ProfilesToLoad, opts.Defaults)()
			if loaded, ok := msg.(quotasLoadedMsg); ok {
				opts.Rows = loaded.rows
				for k, v := range loaded.accounts {
					if opts.Accounts == nil {
						opts.Accounts = map[string]model.AccountInfo{}
					}
					opts.Accounts[k] = v
				}
			}
		}
		printUsageTable(opts.Rows)
		return nil, nil
	}
	p := tea.NewProgram(newUnifiedUsageModel(opts), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		return nil, err
	}
	if res, ok := m.(usageTableModel); ok {
		return res.chosenResult, nil
	}
	return nil, nil
}

// RunUsageTable is a backward-compatible adapter that wraps RunUnifiedUsage.
func RunUsageTable(rows []UsageTableRow) (*UsageTableRow, error) {
	opts := UnifiedUsageOptions{
		Rows:        rows,
		InitialMode: ModeSafe,
	}
	res, err := RunUnifiedUsage(opts)
	if err != nil {
		return nil, err
	}
	if res != nil && res.Action == ActionRunProfile && res.ProfileName != "" {
		for _, r := range rows {
			if r.Provider == res.Provider && r.Profile == res.ProfileName {
				return &r, nil
			}
		}
		return &UsageTableRow{Provider: res.Provider, Profile: res.ProfileName}, nil
	}
	return nil, nil
}

func printUsageTable(rows []UsageTableRow) {
	// Quota cells carry a bar, a percentage and a reset time, so their columns are
	// wider than the identity ones; narrower widths would misalign every row.
	const layout = "%-10s %-20s %-20s %-14s %-30s %-30s %-10s\n"
	fmt.Println("Nexus · Uso por grupo de modelo")
	fmt.Printf(layout, "PROVEDOR", "PERFIL", "CONTA", "PLANO", "5H", "SEMANA", "STATUS")
	for _, row := range rows {
		fmt.Printf(layout,
			row.Provider,
			truncateMiddle(row.Profile, 20),
			truncateMiddle(row.Account, 20),
			truncateMiddle(row.Plan, 14),
			truncateMiddle(row.FiveHour, 30),
			truncateMiddle(row.Weekly, 30),
			row.Status,
		)
	}
}

func newUnifiedUsageModel(opts UnifiedUsageOptions) usageTableModel {
	// Combine configured account rows with installed but unconfigured CLIs
	allRows := make([]UsageTableRow, 0, len(opts.Rows)+len(opts.UnconfiguredCLIs))
	allRows = append(allRows, opts.Rows...)

	// Mark default accounts
	for i := range allRows {
		if opts.Defaults[allRows[i].Provider] == allRows[i].Profile {
			allRows[i].IsDefault = true
		}
	}

	// Add unconfigured installed CLIs as informative rows
	for _, cli := range opts.UnconfiguredCLIs {
		if cli.Installed && cli.Profiles == 0 {
			vStr := cli.Version
			if vStr == "" {
				vStr = "instalado"
			}
			allRows = append(allRows, UsageTableRow{
				Provider:       cli.ID,
				Profile:        "(sem perfil)",
				Account:        cli.Name + " · " + vStr,
				Plan:           "—",
				Group:          "—",
				FiveHour:       "—",
				Weekly:         "—",
				Status:         "OFF",
				ModelName:      "—",
				IsUnconfigured: true,
			})
		}
	}

	filter := textinput.New()
	filter.Placeholder = "filtrar por perfil, conta, provedor ou modelo (/)"
	filter.Prompt = "/ "
	filter.CharLimit = 100

	addInput := textinput.New()
	addInput.Placeholder = "ex: default, work, pessoal (Enter usa 'default')"
	addInput.Prompt = "> "
	addInput.CharLimit = 50

	styles := table.DefaultStyles()
	styles.Header = styles.Header.Bold(true).Foreground(lipgloss.Color("39")).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true)
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("15")).Background(lipgloss.Color("57")).Bold(true)

	initW, initH := 120, 30
	if tw, th, err := term.GetSize(int(os.Stdout.Fd())); err == nil && tw > 0 && th > 0 {
		initW, initH = tw, th
	}

	// Accounts table setup (bubbles kept for cursor/keys; View is custom-rendered).
	sortUsageRows(allRows)
	filteredAccounts := make([]UsageTableRow, len(allRows))
	copy(filteredAccounts, allRows)
	accountRows := tableBodyHeight(initH)
	accTable := table.New(table.WithColumns(usageColumns(initW)), table.WithRows(toUsageTableRowsForWidth(filteredAccounts, initW)), table.WithFocused(true), table.WithHeight(accountRows))
	accTable.SetStyles(styles)

	// Sessions table setup
	allSess := opts.Sessions
	filteredSess := make([]conversation.Conversation, len(allSess))
	copy(filteredSess, allSess)
	sessTable := table.New(table.WithColumns(sessionColumns(initW)), table.WithRows(toSessionTableRowsForWidth(filteredSess, initW)), table.WithFocused(false), table.WithHeight(accountRows+tableHeaderLines))
	sessTable.SetStyles(styles)

	return usageTableModel{
		options:          opts,
		activeTab:        TabAccounts,
		execMode:         opts.InitialMode,
		continueSession:  opts.InitialContinue,
		modalMode:        ModalNone,
		loadingQuotas:    len(opts.ProfilesToLoad) > 0,
		loadingSessions:  opts.LoadSessionsAsync,
		cachedQuotaView:  map[string]quotaViewCache{},
		refreshing:       map[string]bool{},
		allAccounts:      allRows,
		filteredAccounts: filteredAccounts,
		accountTable:     accTable,
		allSessions:      allSess,
		filteredSessions: filteredSess,
		sessionTable:     sessTable,
		filter:           filter,
		addProfileInput:  addInput,
		width:            initW,
		height:           initH,
	}
}

func (m usageTableModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.loadingQuotas {
		cmds = append(cmds, loadQuotasCmd(m.options.ProfilesToLoad, m.options.Defaults), spinnerTick())
	}
	if m.loadingSessions {
		cmds = append(cmds, loadSessionsCmd(m.options.Workspace))
	}
	// Keep the TUI in sync with QuotaMonitor writes without blocking keystrokes.
	cmds = append(cmds, quotaPollTick())
	return tea.Batch(cmds...)
}

func (m usageTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case quotasLoadedMsg:
		m.loadingQuotas = false
		configured := make([]UsageTableRow, 0, len(msg.rows))
		configured = append(configured, msg.rows...)
		for i := range configured {
			if m.options.Defaults[configured[i].Provider] == configured[i].Profile {
				configured[i].IsDefault = true
			}
		}
		// Keep unconfigured CLI rows.
		for _, row := range m.allAccounts {
			if row.IsUnconfigured {
				configured = append(configured, row)
			}
		}
		m.allAccounts = configured
		m.refreshFilteredAccounts()
		if msg.accounts != nil {
			if m.options.Accounts == nil {
				m.options.Accounts = map[string]model.AccountInfo{}
			}
			for k, v := range msg.accounts {
				m.options.Accounts[k] = v
			}
		}
		if !msg.silent {
			// Render cached rows immediately, then probe live sources off the
			// Bubble Tea event loop. Merely polling local quota.json for another
			// minute can repeat a 25-day-old Codex observation indefinitely.
			m.statusMsg = "Lendo quotas atuais em segundo plano…"
			if len(m.options.ProfilesToLoad) > 0 {
				m.lastCodexProbe = time.Now()
				return m, refreshAllCmd(m.options.ProfilesToLoad, m.options.Defaults)
			}
		}
		return m, nil

	case quotaPollTickMsg:
		next := quotaPollTick()
		if m.loadingQuotas || m.modalMode != ModalNone || len(m.options.ProfilesToLoad) == 0 {
			return m, next
		}
		cacheUpdate := loadQuotasCmdOpts(m.options.ProfilesToLoad, m.options.Defaults, true)
		if time.Since(m.lastCodexProbe) >= 5*time.Minute {
			var codexProfiles []model.Profile
			for _, p := range m.options.ProfilesToLoad {
				if p.Provider == "codex" && !p.Disabled {
					codexProfiles = append(codexProfiles, p)
				}
			}
			if len(codexProfiles) > 0 {
				m.lastCodexProbe = time.Now()
				return m, tea.Batch(cacheUpdate, refreshAllCmd(codexProfiles, m.options.Defaults), next)
			}
		}
		return m, tea.Batch(cacheUpdate, next)

	case sessionsLoadedMsg:
		m.loadingSessions = false
		m.allSessions = msg.sessions
		m.filteredSessions = filterSessionRows(m.allSessions, m.filter.Value())
		m.sessionTable.SetRows(toSessionTableRowsForWidth(m.filteredSessions, m.width))
		return m, nil

	case accountRefreshedMsg:
		key := msg.provider + ":" + msg.profile
		delete(m.refreshing, key)
		// Swap this account's rows in place; other accounts keep their state.
		kept := make([]UsageTableRow, 0, len(m.allAccounts)+len(msg.rows))
		inserted := false
		for _, row := range m.allAccounts {
			if row.Provider == msg.provider && row.Profile == msg.profile && !row.IsUnconfigured {
				if !inserted {
					kept = append(kept, msg.rows...)
					inserted = true
				}
				continue
			}
			kept = append(kept, row)
		}
		if !inserted {
			kept = append(kept, msg.rows...)
		}
		for i := range kept {
			if m.options.Defaults[kept[i].Provider] == kept[i].Profile {
				kept[i].IsDefault = true
			}
		}
		m.allAccounts = kept
		m.refreshFilteredAccounts()
		if m.options.Accounts == nil {
			m.options.Accounts = map[string]model.AccountInfo{}
		}
		m.options.Accounts[key] = msg.acc
		// The modal must not serve the pre-refresh view.
		delete(m.cachedQuotaView, key)
		if len(msg.rows) > 0 {
			m.statusMsg = fmt.Sprintf("%s:%s → %s", msg.provider, msg.profile,
				freshnessLine(msg.rows[0].SnapshotStatus, msg.rows[0].Source, msg.rows[0].FetchedAt))
		} else {
			m.statusMsg = fmt.Sprintf("%s:%s sem dados de cota", msg.provider, msg.profile)
		}
		return m, nil

	case accountsRefreshedMsg:
		// Merge all refreshed accounts into allAccounts.
		for _, sub := range msg.results {
			key := sub.provider + ":" + sub.profile
			kept := make([]UsageTableRow, 0, len(m.allAccounts)+len(sub.rows))
			inserted := false
			for _, row := range m.allAccounts {
				if row.Provider == sub.provider && row.Profile == sub.profile && !row.IsUnconfigured {
					if !inserted {
						kept = append(kept, sub.rows...)
						inserted = true
					}
					continue
				}
				kept = append(kept, row)
			}
			if !inserted {
				kept = append(kept, sub.rows...)
			}
			for i := range kept {
				if m.options.Defaults[kept[i].Provider] == kept[i].Profile {
					kept[i].IsDefault = true
				}
			}
			m.allAccounts = kept
			if m.options.Accounts == nil {
				m.options.Accounts = map[string]model.AccountInfo{}
			}
			m.options.Accounts[key] = sub.acc
			delete(m.cachedQuotaView, key)
			delete(m.refreshing, key)
		}
		m.refreshFilteredAccounts()
		// Count results for status message.
		ok, stale := 0, 0
		for _, sub := range msg.results {
			if len(sub.rows) == 0 {
				stale++
				continue
			}
			switch sub.rows[0].SnapshotStatus {
			case string(model.UsageUnknown), string(model.UsageUnsupported), string(model.UsageError), "":
				stale++
			default:
				ok++
			}
		}
		total := len(msg.results)
		if stale > 0 {
			m.statusMsg = fmt.Sprintf("%d atualizada(s) · %d sem dados", ok, stale)
		} else {
			m.statusMsg = fmt.Sprintf("%d conta(s) atualizada(s)", total)
		}
		return m, nil

	case spinnerTickMsg:
		if m.loadingQuotas || m.loadingSessions || len(m.refreshing) > 0 {
			m.spinFrame = (m.spinFrame + 1) % 4
			return m, spinnerTick()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeTables()
		return m, nil

	case tea.KeyMsg:
		k := msg.String()

		// Global quit: ctrl+c always exits immediately
		if k == "ctrl+c" {
			m.quitting = true
			m.chosenResult = &SelectionResult{Action: ActionQuit}
			return m, tea.Quit
		}

		// Handle active modal
		if m.modalMode == ModalAddProfile {
			switch k {
			case "esc":
				m.modalMode = ModalNone
				m.addProfileInput.Blur()
				m.resizeTables()
				return m, nil
			case "enter":
				profileName := strings.TrimSpace(m.addProfileInput.Value())
				if profileName == "" {
					profileName = "default"
				}
				m.modalMode = ModalNone
				m.addProfileInput.Blur()
				m.quitting = true
				m.chosenResult = &SelectionResult{
					Action:      ActionAdd,
					Provider:    m.addProfileProvider,
					ProfileName: profileName,
				}
				return m, tea.Quit
			default:
				var cmd tea.Cmd
				m.addProfileInput, cmd = m.addProfileInput.Update(msg)
				return m, cmd
			}
		}
		if m.modalMode != ModalNone {
			if k == "esc" || k == "q" || k == "Q" || k == "enter" || k == " " {
				m.modalMode = ModalNone
				m.resizeTables()
				return m, nil
			}
			return m, nil
		}

		// Handle search/filter focus
		if m.filter.Focused() {
			if k == "esc" {
				m.filter.Blur()
				return m, nil
			}
			if k == "enter" {
				m.filter.Blur()
			} else {
				var cmd tea.Cmd
				m.filter, cmd = m.filter.Update(msg)
				q := m.filter.Value()
				if m.activeTab == TabAccounts {
					m.refreshFilteredAccounts()
					if len(m.filteredAccounts) > 0 && m.accountTable.Cursor() < 0 {
						m.accountTable.SetCursor(0)
					}
				} else {
					m.filteredSessions = filterSessionRows(m.allSessions, q)
					m.sessionTable.SetRows(toSessionTableRowsForWidth(m.filteredSessions, m.width))
					if len(m.filteredSessions) > 0 && m.sessionTable.Cursor() < 0 {
						m.sessionTable.SetCursor(0)
					}
				}
				return m, cmd
			}
		}

		// Main screen keys: esc or q exits immediately!
		if k == "esc" || k == "q" || k == "Q" {
			m.quitting = true
			m.chosenResult = &SelectionResult{Action: ActionQuit}
			return m, tea.Quit
		}

		// Tab switching between Accounts and Sessions
		if k == "tab" || k == "shift+tab" {
			if m.activeTab == TabAccounts {
				m.activeTab = TabSessions
				m.accountTable.Blur()
				m.sessionTable.Focus()
			} else {
				m.activeTab = TabAccounts
				m.sessionTable.Blur()
				m.accountTable.Focus()
			}
			return m, nil
		}

		// Mode switching keys
		switch k {
		case "1":
			m.execMode = ModeSafe
			m.statusMsg = "Modo alterado para Safe (padrão interativo supervisionado)"
			return m, nil
		case "2", "y", "Y":
			m.execMode = ModeYOLO
			m.statusMsg = "Modo alterado para ⚡ YOLO (auto-aprovação e bypass de sandbox)"
			return m, nil
		case "3", "p", "P":
			m.execMode = ModePlan
			m.statusMsg = "Modo alterado para 📋 Plan (modo planejamento)"
			return m, nil
		case "m", "M":
			m.execMode = (m.execMode + 1) % 3
			m.statusMsg = fmt.Sprintf("Modo alterado para %s", m.execMode.String())
			return m, nil
		case "c", "C":
			m.continueSession = !m.continueSession
			if m.continueSession {
				m.statusMsg = "Opção Continuar Sessão: LIGADA (--continue)"
			} else {
				m.statusMsg = "Opção Continuar Sessão: DESLIGADA"
			}
			return m, nil
		case "/":
			m.filter.Focus()
			return m, textinput.Blink
		case "s", "S":
			// Open Quota Details modal for highlighted account
			if m.activeTab == TabAccounts && len(m.filteredAccounts) > 0 {
				m.modalMode = ModalQuotaDetails
				m.ensureQuotaModalCached()
			}
			return m, nil
		case "d", "D":
			// Set as default profile
			if m.activeTab == TabAccounts && len(m.filteredAccounts) > 0 {
				idx := m.accountTable.Cursor()
				if idx >= 0 && idx < len(m.filteredAccounts) {
					row := m.filteredAccounts[idx]
					if !row.IsUnconfigured {
						_ = config.SetDefaultProfile(row.Provider, row.Profile)
						for i := range m.allAccounts {
							if m.allAccounts[i].Provider == row.Provider {
								m.allAccounts[i].IsDefault = (m.allAccounts[i].Profile == row.Profile)
							}
						}
						for i := range m.filteredAccounts {
							if m.filteredAccounts[i].Provider == row.Provider {
								m.filteredAccounts[i].IsDefault = (m.filteredAccounts[i].Profile == row.Profile)
							}
						}
						m.setAccountRows()
						m.statusMsg = fmt.Sprintf("✓ Definido %s:%s como perfil padrão.", row.Provider, row.Profile)
					}
				}
			}
			return m, nil
		case "r":
			// Re-read quota for the highlighted account without blocking the UI.
			if m.activeTab == TabAccounts && len(m.filteredAccounts) > 0 {
				idx := m.accountTable.Cursor()
				if idx >= 0 && idx < len(m.filteredAccounts) {
					row := m.filteredAccounts[idx]
					if row.IsUnconfigured {
						m.statusMsg = fmt.Sprintf("⚠ %s não está configurado.", row.Provider)
						return m, nil
					}
					key := row.Provider + ":" + row.Profile
					if m.refreshing[key] {
						return m, nil
					}
					if m.refreshing == nil {
						m.refreshing = map[string]bool{}
					}
					m.refreshing[key] = true
					m.statusMsg = fmt.Sprintf("atualizando %s…", key)
					return m, tea.Batch(
						refreshAccountCmd(row.Provider, row.Profile, m.options.Defaults),
						spinnerTick(),
					)
				}
			}
			return m, nil
		case "R":
			// Force a live refresh of all configured accounts.
			if m.activeTab == TabAccounts && len(m.options.ProfilesToLoad) > 0 {
				if m.refreshing == nil {
					m.refreshing = map[string]bool{}
				}
				for _, p := range m.options.ProfilesToLoad {
					m.refreshing[p.Provider+":"+p.Name] = true
				}
				m.statusMsg = "atualizando todas as contas…"
				return m, tea.Batch(
					refreshAllCmd(m.options.ProfilesToLoad, m.options.Defaults),
					spinnerTick(),
				)
			}
			return m, nil
		case "l", "L":
			// Trigger login
			switch m.activeTab {
			case TabAccounts:
				if len(m.filteredAccounts) > 0 {
					idx := m.accountTable.Cursor()
					if idx >= 0 && idx < len(m.filteredAccounts) {
						row := m.filteredAccounts[idx]
						if !row.IsUnconfigured {
							m.quitting = true
							m.chosenResult = &SelectionResult{
								Action:      ActionLogin,
								Provider:    row.Provider,
								ProfileName: row.Profile,
							}
							return m, tea.Quit
						}
					}
				}
			}
			return m, nil
		case "a", "A":
			// Open Add Profile modal for highlighted provider
			switch m.activeTab {
			case TabAccounts:
				if len(m.filteredAccounts) > 0 {
					idx := m.accountTable.Cursor()
					if idx >= 0 && idx < len(m.filteredAccounts) {
						row := m.filteredAccounts[idx]
						m.modalMode = ModalAddProfile
						m.addProfileProvider = row.Provider
						m.addProfileInput.Reset()
						m.addProfileInput.Focus()
						return m, textinput.Blink
					}
				}
			}
			return m, nil
		case "enter":
			switch m.activeTab {
			case TabAccounts:
				idx := m.accountTable.Cursor()
				if idx >= 0 && idx < len(m.filteredAccounts) {
					row := m.filteredAccounts[idx]
					if row.IsUnconfigured {
						m.modalMode = ModalAddProfile
						m.addProfileProvider = row.Provider
						m.addProfileInput.Reset()
						m.addProfileInput.Focus()
						return m, textinput.Blink
					}
					flags := modeFlags(m.execMode, m.continueSession)
					m.quitting = true
					m.chosenResult = &SelectionResult{
						Action:      ActionRunProfile,
						Provider:    row.Provider,
						ProfileName: row.Profile,
						Args:        flags,
					}
					return m, tea.Quit
				}
			case TabSessions:
				idx := m.sessionTable.Cursor()
				if idx >= 0 && idx < len(m.filteredSessions) {
					sess := m.filteredSessions[idx]
					flags := modeFlags(m.execMode, false)
					m.quitting = true
					m.chosenResult = &SelectionResult{
						Action:         ActionResumeConversation,
						Provider:       sess.Provider,
						ProfileName:    "",
						ConversationID: sess.ID,
						Args:           flags,
					}
					return m, tea.Quit
				}
			}
			return m, nil
		}
	}

	// Update table models
	var cmd tea.Cmd
	if m.activeTab == TabAccounts {
		m.accountTable, cmd = m.accountTable.Update(msg)
	} else {
		m.sessionTable, cmd = m.sessionTable.Update(msg)
	}
	return m, cmd
}

func (m *usageTableModel) resizeTables() {
	w := m.width
	if w < 40 {
		w = 40
	}
	// Custom account render uses data-row count; bubbles SetHeight includes its header.
	accountRows := tableBodyHeight(m.height)
	sessionTotal := accountRows + tableHeaderLines
	replaceTableContents(&m.accountTable, usageColumns(m.width), toUsageTableRowsForWidth(m.filteredAccounts, m.width), w, accountRows)
	replaceTableContents(&m.sessionTable, sessionColumns(m.width), toSessionTableRowsForWidth(m.filteredSessions, m.width), w, sessionTotal)
}

func replaceTableContents(t *table.Model, cols []table.Column, rows []table.Row, w, h int) {
	// bubbles table re-renders immediately on SetColumns/SetRows. Mixing a new
	// column count with stale cells panics (index out of range).
	cursor := t.Cursor()
	t.SetRows(nil)
	t.SetColumns(cols)
	t.SetWidth(w)
	t.SetHeight(h)
	t.SetRows(rows)
	if len(rows) == 0 {
		return
	}
	if cursor < 0 {
		cursor = 0
	}
	t.SetCursor(cursor)
}

func (m *usageTableModel) refreshFilteredAccounts() {
	selected := m.selectedAccountKey()
	sortUsageRows(m.allAccounts)
	m.filteredAccounts = filterUsageRows(m.allAccounts, m.filter.Value())
	m.setAccountRows()
	m.restoreAccountCursor(selected)
}

func (m *usageTableModel) selectedAccountKey() string {
	idx := m.accountTable.Cursor()
	if idx < 0 || idx >= len(m.filteredAccounts) {
		return ""
	}
	row := m.filteredAccounts[idx]
	return row.Provider + ":" + row.Profile
}

func (m *usageTableModel) restoreAccountCursor(key string) {
	if key == "" || len(m.filteredAccounts) == 0 {
		return
	}
	for i, row := range m.filteredAccounts {
		if row.Provider+":"+row.Profile == key {
			m.accountTable.SetCursor(i)
			return
		}
	}
	if m.accountTable.Cursor() >= len(m.filteredAccounts) {
		m.accountTable.SetCursor(len(m.filteredAccounts) - 1)
	}
}

func (m *usageTableModel) setAccountRows() {
	m.accountTable.SetRows(toUsageTableRowsForWidth(m.filteredAccounts, m.width))
}

// usageChromeLines is the chrome around the accounts/sessions table:
// header, tabs, preview, help.
const usageChromeLines = 4

// tableHeaderLines is title + bottom border for the accounts/sessions grid.
const tableHeaderLines = 2

// tableColPad is bubbles table DefaultStyles Padding(0, 1) on header and cell.
const tableColPad = 2

func tableBodyHeight(termHeight int) int {
	h := termHeight - usageChromeLines - tableHeaderLines
	if h < 6 {
		return 6
	}
	return h
}

type colSpec struct {
	title     string
	min       int
	pref      int
	flexible  bool
	growQuota bool
	maxFlex   int
}

func usageColumns(width int) []table.Column {
	if width < 40 {
		width = 40
	}
	switch {
	case width < 84:
		return fitTableColumns(width, []colSpec{
			{title: "#", min: 3, pref: 3},
			{title: "PROV", min: 6, pref: 7},
			{title: "CONTA", min: 12, pref: 24, flexible: true},
			{title: "STATUS", min: 10, pref: 10},
		})
	case width < 108:
		return fitTableColumns(width, []colSpec{
			{title: "#", min: 3, pref: 3},
			{title: "PROV", min: 6, pref: 7},
			{title: "CONTA", min: 14, pref: 28, flexible: true},
			{title: "5H", min: 16, pref: 24, growQuota: true},
			{title: "STATUS", min: 10, pref: 10},
		})
	default:
		return fitTableColumns(width, []colSpec{
			{title: "#", min: 3, pref: 3},
			{title: "PROV", min: 6, pref: 7},
			{title: "CONTA", min: 18, pref: 32, flexible: true},
			{title: "5H", min: 18, pref: 24, growQuota: true},
			{title: "SEMANA", min: 18, pref: 26, growQuota: true},
			{title: "STATUS", min: 10, pref: 10},
		})
	}
}

func sessionColumns(width int) []table.Column {
	if width < 40 {
		width = 40
	}
	switch {
	case width < 70:
		return fitTableColumns(width, []colSpec{
			{title: "PROV", min: 6, pref: 7},
			{title: "SESSÃO", min: 6, pref: 8},
			{title: "TÍTULO", min: 10, pref: 16, flexible: true, maxFlex: 48},
		})
	case width < 110:
		return fitTableColumns(width, []colSpec{
			{title: "PROVEDOR", min: 8, pref: 10},
			{title: "SESSÃO", min: 8, pref: 10},
			{title: "MODIFICADO", min: 10, pref: 12},
			{title: "TÍTULO", min: 12, pref: 18, flexible: true, maxFlex: 40},
			{title: "WORKSPACE", min: 10, pref: 14},
		})
	default:
		return fitTableColumns(width, []colSpec{
			{title: "PROVEDOR", min: 8, pref: 11},
			{title: "SESSÃO", min: 8, pref: 14},
			{title: "MODIFICADO", min: 10, pref: 16},
			{title: "TÍTULO", min: 14, pref: 24, flexible: true, maxFlex: 48},
			{title: "WORKSPACE", min: 10, pref: 22},
		})
	}
}

func fitTableColumns(termWidth int, specs []colSpec) []table.Column {
	n := len(specs)
	budget := termWidth - n*tableColPad
	if budget < n {
		budget = n
	}
	widths := make([]int, n)
	used := 0
	flexIdx := -1
	quotaIdxs := make([]int, 0, 2)
	for i, spec := range specs {
		widths[i] = spec.pref
		used += spec.pref
		if spec.flexible {
			flexIdx = i
		}
		if spec.growQuota {
			quotaIdxs = append(quotaIdxs, i)
		}
	}
	if used > budget {
		overflow := used - budget
		for i := 0; i < n && overflow > 0; i++ {
			if specs[i].title == "STATUS" || specs[i].title == "PROV" || specs[i].title == "PROVEDOR" || specs[i].title == "#" {
				continue
			}
			reducible := widths[i] - specs[i].min
			if reducible <= 0 {
				continue
			}
			take := min(reducible, overflow)
			widths[i] -= take
			overflow -= take
		}
		for i := n - 1; i >= 0 && overflow > 0; i-- {
			reducible := widths[i] - specs[i].min
			if reducible <= 0 {
				continue
			}
			take := min(reducible, overflow)
			widths[i] -= take
			overflow -= take
		}
	} else if leftover := budget - used; leftover > 0 {
		// Fill the terminal: CONTA first, then quota columns, never leave a void.
		if flexIdx >= 0 {
			add := leftover
			if specs[flexIdx].maxFlex > 0 {
				capAdd := specs[flexIdx].maxFlex - widths[flexIdx]
				if capAdd < 0 {
					capAdd = 0
				}
				if add > capAdd {
					add = capAdd
				}
			} else if len(quotaIdxs) > 0 && widths[flexIdx]+leftover > 72 {
				// Keep CONTA readable but hand surplus to bars on very wide terminals.
				capAdd := 72 - widths[flexIdx]
				if capAdd < 0 {
					capAdd = 0
				}
				if add > capAdd {
					add = capAdd
				}
			}
			widths[flexIdx] += add
			leftover -= add
		}
		if leftover > 0 && len(quotaIdxs) > 0 {
			share := leftover / len(quotaIdxs)
			for i, idx := range quotaIdxs {
				extra := share
				if i == len(quotaIdxs)-1 {
					extra = leftover - share*(len(quotaIdxs)-1)
				}
				widths[idx] += extra
			}
		} else if leftover > 0 && flexIdx >= 0 {
			widths[flexIdx] += leftover
		}
	}
	cols := make([]table.Column, n)
	for i, spec := range specs {
		cols[i] = table.Column{Title: spec.title, Width: widths[i]}
	}
	return cols
}

func renderedTableWidth(cols []table.Column) int {
	sum := 0
	for _, c := range cols {
		sum += c.Width + tableColPad
	}
	return sum
}

func toUsageTableRowsForWidth(rows []UsageTableRow, width int) []table.Row {
	cols := usageColumns(width)
	showFive := false
	showWeekly := false
	accountWidth := 24
	for _, c := range cols {
		switch c.Title {
		case "5H":
			showFive = true
		case "SEMANA":
			showWeekly = true
		case "CONTA":
			accountWidth = c.Width
		}
	}

	result := make([]table.Row, 0, len(rows))
	for i, row := range rows {
		provBadge := formatProviderBadge(row.Provider)
		star := ""
		if row.IsDefault {
			star = "★"
		}

		identity := formatAccountIdentity(row, accountWidth, star)

		cells := []string{strconv.Itoa(i + 1), provBadge, identity}
		if showFive {
			five := row.FiveHour
			if !showWeekly && five != "—" && five != "SEM DADOS" {
				// Mid width: keep compact percent only when weekly hidden.
				five = compactFiveHour(five)
			}
			cells = append(cells, five)
		}
		if showWeekly {
			cells = append(cells, row.Weekly)
		}
		cells = append(cells, row.Status)
		result = append(result, table.Row(cells))
	}
	return result
}

func formatAccountIdentity(row UsageTableRow, width int, star string) string {
	if row.IsUnconfigured {
		return truncateMiddle(row.Account, width)
	}
	profile := row.Profile + star
	group := shortGroupLabel(row.Group)
	account := row.Account
	base := profile
	if group != "" {
		base = profile + " · " + group
	}
	if account != "" {
		combined := base + " · " + account
		if lipgloss.Width(combined) <= width {
			return combined
		}
		// Prefer keeping profile; truncate email.
		remain := width - lipgloss.Width(base) - 3
		if remain > 6 {
			return base + " · " + truncateMiddle(account, remain)
		}
		return truncateMiddle(base, width)
	}
	return truncateMiddle(base, width)
}

func shortGroupLabel(group string) string {
	g := strings.TrimSpace(group)
	if g == "" || g == "—" {
		return ""
	}
	lower := strings.ToLower(g)
	switch {
	case strings.Contains(lower, "gemini"):
		return "Gemini"
	case strings.Contains(lower, "claude"):
		return "Claude"
	case strings.Contains(lower, "gpt"):
		return "GPT"
	default:
		if len(g) > 10 {
			return g[:10]
		}
		return g
	}
}

func compactFiveHour(five string) string {
	// "[████░░░░] 82% · 20:55" → "82% · 20:55" when space is tight for weekly column.
	if idx := strings.Index(five, "] "); idx >= 0 && idx+2 < len(five) {
		return strings.TrimSpace(five[idx+2:])
	}
	return five
}

func truncateMiddle(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:min(len(s), width)]
	}
	keep := width - 1
	left := keep / 2
	right := keep - left
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if left+right >= len(runes) {
		return string(runes[:width])
	}
	return string(runes[:left]) + "…" + string(runes[len(runes)-right:])
}

func toSessionTableRowsForWidth(sess []conversation.Conversation, width int) []table.Row {
	cols := sessionColumns(width)
	showModified := false
	showWorkspace := false
	titleW, sessW := 24, 10
	for _, c := range cols {
		switch c.Title {
		case "MODIFICADO":
			showModified = true
		case "WORKSPACE":
			showWorkspace = true
		case "TÍTULO":
			titleW = c.Width
		case "SESSÃO":
			sessW = c.Width
		}
	}
	result := make([]table.Row, 0, len(sess))
	for _, s := range sess {
		provBadge := formatProviderBadge(s.Provider)
		sID := s.ID
		if len(sID) > sessW {
			if sessW <= 2 {
				sID = sID[:sessW]
			} else {
				sID = sID[:sessW-2] + ".."
			}
		}
		title := truncateRunes(s.Title, titleW)
		wsShort := filepath.Base(s.Workspace)
		if wsShort == "" || wsShort == "." {
			wsShort = "~"
		}
		wsShort = "~/" + wsShort
		row := table.Row{provBadge, sID}
		if showModified {
			row = append(row, formatTimeAgo(s.LastModified))
		}
		row = append(row, title)
		if showWorkspace {
			row = append(row, truncateRunes(wsShort, 22))
		}
		result = append(result, row)
	}
	return result
}

func truncateRunes(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	if maxWidth <= 2 {
		return string(runes[:maxWidth])
	}
	return string(runes[:maxWidth-2]) + ".."
}
