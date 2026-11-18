package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/security"
)

// UnifiedResumeSession is a read-only description of one local Codex thread.
// Source paths and account IDs remain local and are rechecked before use.
type UnifiedResumeSession struct {
	ID                string
	Title             string
	Workspace         string
	Account           string
	Profile           string
	UpdatedAt         time.Time
	UnavailableReason string
	SourcePath        string
	SourceHome        string
	AccountID         string
}

type resumeSource struct {
	home      string
	profile   string
	accountID string
	email     string
	active    bool
	host      bool
}

type resumeCandidate struct {
	UnifiedResumeSession
	imported bool
	active   bool
}

// ListUnifiedResumeSessions reads all profile and legacy stores without
// creating profiles, copying sessions, or changing authentication state.
func ListUnifiedResumeSessions(ctx context.Context) ([]UnifiedResumeSession, error) {
	data, err := config.DataDir()
	if err != nil {
		return nil, err
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	profileRoot := filepath.Join(data, "profiles", "codex")
	entries, err := os.ReadDir(profileRoot)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	var sources []resumeSource
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		root := filepath.Join(profileRoot, entry.Name())
		var p model.Profile
		if raw, readErr := os.ReadFile(filepath.Join(root, "profile.json")); readErr == nil {
			_ = json.Unmarshal(raw, &p)
		}
		active := p.Provider == "codex" && p.Name == entry.Name() && config.ValidateProfileName(p.Name) == nil && !p.Disabled && !cfg.Disabled["codex"][entry.Name()]
		home := filepath.Join(root, "home")
		sources = append(sources, makeResumeSource(home, entry.Name(), active, false))
		// Some older installations kept Codex files at the profile root.
		sources = append(sources, makeResumeSource(root, "", false, false))
	}
	if host := security.FindHostHome(); host != "" {
		sources = append(sources, makeResumeSource(filepath.Join(host, ".codex"), "", false, true))
		legacyRoot := filepath.Join(host, ".codex-accounts")
		legacy, _ := os.ReadDir(legacyRoot)
		for _, entry := range legacy {
			if entry.IsDir() {
				sources = append(sources, makeResumeSource(filepath.Join(legacyRoot, entry.Name()), "", false, false))
			}
		}
	}

	candidates := make(map[string][]resumeCandidate)
	for _, source := range sources {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		collectResumeSource(ctx, source, candidates)
	}
	result := make([]UnifiedResumeSession, 0, len(candidates))
	for _, group := range candidates {
		result = append(result, chooseResumeCandidate(group, sources, cfg.Defaults["codex"]))
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].UpdatedAt.Equal(result[j].UpdatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func makeResumeSource(home, profile string, active, host bool) resumeSource {
	source := resumeSource{home: home, profile: profile, active: active, host: host}
	for _, path := range []string{filepath.Join(home, "auth.json"), filepath.Join(home, ".codex", "auth.json")} {
		if raw, err := os.ReadFile(path); err == nil && len(raw) > 0 {
			source.accountID, source.email, _, _ = parseCodexAuthBytes(raw)
			break
		}
	}
	return source
}

func collectResumeSource(ctx context.Context, source resumeSource, out map[string][]resumeCandidate) {
	index := readResumeSQLiteIndex(ctx, source.home)
	for _, path := range []string{filepath.Join(source.home, "session_index.jsonl"), filepath.Join(source.home, ".codex", "session_index.jsonl")} {
		entries, _ := readSessionIndexEntries(path)
		for _, entry := range entries {
			if previous, ok := index[entry.ID]; !ok || indexEntryNewer(entry, previous) {
				index[entry.ID] = entry
			}
		}
	}
	markers := readCrossAccountSessionIDs(source.home)
	seen := make(map[string]bool)
	for _, root := range []string{filepath.Join(source.home, "sessions"), filepath.Join(source.home, ".codex", "sessions"), filepath.Join(source.home, "archived_sessions"), filepath.Join(source.home, ".codex", "archived_sessions")} {
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if walkErr != nil || entry == nil || entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if !strings.HasPrefix(entry.Name(), "rollout-") || !strings.HasSuffix(entry.Name(), ".jsonl") {
				return nil
			}
			id := extractSessionIDFromRolloutName(entry.Name())
			if id == "" {
				return nil
			}
			info, statErr := entry.Info()
			if statErr != nil {
				return nil
			}
			meta := synthesizeIndexEntry(path, id, info.ModTime())
			item := resumeCandidate{UnifiedResumeSession: UnifiedResumeSession{
				ID: id, Title: resumeTitle(path, id), Workspace: meta.CWD, UpdatedAt: info.ModTime(),
				SourcePath: path, SourceHome: source.home, Profile: source.profile, AccountID: source.accountID, Account: source.email,
			}, imported: markers[id], active: source.active}
			if indexed, ok := index[id]; ok {
				if strings.TrimSpace(indexed.ThreadName) != "" {
					item.Title = indexed.ThreadName
				}
				if item.Workspace == "" {
					item.Workspace = indexed.CWD
				}
				if t, parseErr := time.Parse(time.RFC3339Nano, indexed.UpdatedAt); parseErr == nil && t.After(item.UpdatedAt) {
					item.UpdatedAt = t
				}
			}
			if account := readRolloutAccountID(path); account != "" {
				if source.host && !strings.EqualFold(account, item.AccountID) {
					item.AccountID, item.Account = account, ""
				} else if item.AccountID != "" && !strings.EqualFold(account, item.AccountID) {
					item.AccountID, item.Account = "", ""
				} else {
					item.AccountID = account
				}
			} else if source.host {
				// The current host login alone cannot identify historical ownership.
				item.AccountID, item.Account = "", ""
			}
			item.active = source.active && source.accountID != "" && strings.EqualFold(source.accountID, item.AccountID)
			out[id] = append(out[id], item)
			seen[id] = true
			return nil
		})
	}
	for id, entry := range index {
		if seen[id] {
			continue
		}
		updated, _ := time.Parse(time.RFC3339Nano, entry.UpdatedAt)
		out[id] = append(out[id], resumeCandidate{UnifiedResumeSession: UnifiedResumeSession{
			ID: id, Title: entry.ThreadName, Workspace: entry.CWD, UpdatedAt: updated,
			SourceHome: source.home, Profile: source.profile, AccountID: source.accountID, Account: source.email,
		}, imported: markers[id], active: source.active})
	}
}

func resumeTitle(path, id string) string {
	f, err := os.Open(path)
	if err != nil {
		return "Codex " + shortSessionTitle(id)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for count := 0; count < 100 && scanner.Scan(); count++ {
		var event struct {
			Type    string `json:"type"`
			Payload struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"payload"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) == nil && event.Type == "event_msg" && event.Payload.Type == "user_message" {
			if text := strings.TrimSpace(event.Payload.Message); text != "" {
				runes := []rune(strings.Join(strings.Fields(text), " "))
				if len(runes) > 120 {
					runes = runes[:120]
				}
				return string(runes)
			}
		}
	}
	return "Codex " + shortSessionTitle(id)
}

func chooseResumeCandidate(group []resumeCandidate, sources []resumeSource, defaultProfile string) UnifiedResumeSession {
	var originals []resumeCandidate
	for _, item := range group {
		if !item.imported && item.SourcePath != "" {
			originals = append(originals, item)
		}
	}
	if len(originals) == 0 {
		item := group[0].UnifiedResumeSession
		item.Account, item.Profile, item.AccountID = "Desconhecida", "", ""
		item.UnavailableReason = "Origem ou arquivo original indisponível"
		return item
	}
	sort.SliceStable(originals, func(i, j int) bool {
		if originals[i].active != originals[j].active {
			return originals[i].active
		}
		return originals[i].UpdatedAt.After(originals[j].UpdatedAt)
	})
	selected := originals[0].UnifiedResumeSession
	for _, item := range originals {
		if item.AccountID == "" || selected.AccountID == "" || !strings.EqualFold(item.AccountID, selected.AccountID) {
			selected.AccountID = ""
			break
		}
	}
	if selected.AccountID == "" {
		selected.Account, selected.Profile = "Desconhecida", ""
		selected.UnavailableReason = "Origem ambígua ou não confirmada"
		return selected
	}
	if selected.Account == "" {
		for _, source := range sources {
			if source.email != "" && strings.EqualFold(source.accountID, selected.AccountID) {
				selected.Account = source.email
				break
			}
		}
		if selected.Account == "" {
			selected.Account = selected.AccountID
		}
	}
	if originals[0].active && originals[0].Profile != "" {
		return selected
	}
	selected.Profile = ""
	for _, source := range sources {
		if !source.active || !strings.EqualFold(source.accountID, selected.AccountID) {
			continue
		}
		if selected.Profile == "" || source.profile == defaultProfile {
			selected.Profile = source.profile
		}
		if source.profile == defaultProfile {
			break
		}
	}
	if selected.Profile == "" {
		selected.UnavailableReason = "Nenhum perfil autenticado disponível para esta conta"
	}
	return selected
}

// hostCodexIdentityMatchesRollout reports whether the source home is the
// host's own .codex directory and the rollout identity already matches the
// target account, in which case a differing source account ID is not a
// cross-account takeover.
func hostCodexIdentityMatchesRollout(selected UnifiedResumeSession, rolloutAccount string) bool {
	home := security.FindHostHome()
	if home == "" {
		return false
	}
	return config.FilesystemPathsEquivalent(selected.SourceHome, filepath.Join(home, ".codex")) &&
		strings.EqualFold(rolloutAccount, selected.AccountID)
}

// PrepareUnifiedResume revalidates ownership and recovers only the selected
// rollout when it exists outside the selected authenticated profile.
func PrepareUnifiedResume(ctx context.Context, selected UnifiedResumeSession) error {
	if selected.UnavailableReason != "" || selected.ID == "" || selected.Profile == "" || selected.AccountID == "" || selected.SourcePath == "" {
		return fmt.Errorf("sessão sem origem ou perfil autenticado confirmado")
	}
	sessions, err := ListUnifiedResumeSessions(ctx)
	if err != nil {
		return err
	}
	verified := false
	for _, item := range sessions {
		if item.ID == selected.ID && item.Profile == selected.Profile && item.AccountID == selected.AccountID && item.SourcePath == selected.SourcePath && item.UnavailableReason == "" {
			verified = true
			break
		}
	}
	if !verified {
		return fmt.Errorf("a origem ou a conta da sessão mudou; atualize a lista")
	}
	home, err := config.ProfileHome("codex", selected.Profile)
	if err != nil {
		return err
	}
	if config.FilesystemPathWithin(filepath.Join(home, "sessions"), selected.SourcePath) {
		return validateUnifiedProfile(selected.Profile, selected.AccountID)
	}
	lock, ok, err := TryAcquireMaintenanceLock(home)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: perfil em uso; tente novamente", ErrTUIBusy)
	}
	defer func() { _ = lock.Release() }()
	if err := validateUnifiedProfile(selected.Profile, selected.AccountID); err != nil {
		return err
	}
	if readCrossAccountSessionIDs(selected.SourceHome)[selected.ID] {
		return fmt.Errorf("a origem foi marcada como cópia de outra conta")
	}
	sourceAccount := makeResumeSource(selected.SourceHome, "", false, false).accountID
	rolloutAccount := readRolloutAccountID(selected.SourcePath)
	if rolloutAccount != "" && !strings.EqualFold(rolloutAccount, selected.AccountID) {
		return fmt.Errorf("a identidade da sessão mudou")
	}
	if sourceAccount == "" && rolloutAccount == "" {
		return fmt.Errorf("a identidade da origem não está disponível")
	}
	if sourceAccount != "" && !strings.EqualFold(sourceAccount, selected.AccountID) &&
		!hostCodexIdentityMatchesRollout(selected, rolloutAccount) {
		return fmt.Errorf("a conta da origem mudou")
	}
	destRoot := filepath.Join(home, "sessions")
	if existing := findRolloutInTree(destRoot, selected.ID); existing != "" {
		return fmt.Errorf("já existe uma cópia da sessão no perfil; origem requer revisão")
	}
	name := filepath.Base(selected.SourcePath)
	if len(name) < len("rollout-2006-01-02") || !strings.HasPrefix(name, "rollout-") {
		return fmt.Errorf("nome de sessão sem data reconhecível")
	}
	date, err := time.Parse("2006-01-02", name[len("rollout-"):len("rollout-2006-01-02")])
	if err != nil {
		return fmt.Errorf("data da sessão inválida: %w", err)
	}
	destDir := filepath.Join(destRoot, date.Format("2006"), date.Format("01"), date.Format("02"))
	if err := os.MkdirAll(destDir, 0700); err != nil {
		return err
	}
	dest := filepath.Join(destDir, name)
	pathInfo, err := os.Lstat(selected.SourcePath)
	if err != nil {
		return err
	}
	if !pathInfo.Mode().IsRegular() {
		return fmt.Errorf("o arquivo original da sessão não é regular")
	}
	in, err := os.Open(selected.SourcePath)
	if err != nil {
		return err
	}
	defer in.Close()
	before, err := in.Stat()
	if err != nil || !os.SameFile(pathInfo, before) {
		return fmt.Errorf("o arquivo original da sessão mudou")
	}
	tmp, err := os.CreateTemp(destDir, ".nexus-resume-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	copied, err := io.Copy(tmp, in)
	if err != nil {
		_ = tmp.Close()
		return err
	}
	after, err := in.Stat()
	if err != nil || copied != before.Size() || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		_ = tmp.Close()
		return fmt.Errorf("a sessão original foi alterada durante a recuperação; tente novamente")
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Link the temporary copy into place without replacing a concurrently
	// created destination. The source is never hard-linked to the profile.
	if err = os.Link(tmpPath, dest); err != nil {
		return err
	}
	entry := sessionIndexEntry{ID: selected.ID, ThreadName: selected.Title, CWD: selected.Workspace, UpdatedAt: selected.UpdatedAt.UTC().Format(time.RFC3339Nano)}
	if err = appendUnifiedResumeIndex(home, entry); err != nil {
		_ = os.Remove(dest)
		return err
	}
	return nil
}

func validateUnifiedProfile(name, accountID string) error {
	if err := config.ValidateProfileName(name); err != nil {
		return err
	}
	root, err := config.ProfileRoot("codex", name)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(root, "profile.json"))
	if err != nil {
		return fmt.Errorf("perfil autenticado indisponível")
	}
	var p model.Profile
	if json.Unmarshal(raw, &p) != nil || p.Provider != "codex" || p.Name != name || p.Disabled {
		return fmt.Errorf("perfil autenticado indisponível")
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.Disabled["codex"][name] {
		return fmt.Errorf("perfil desativado")
	}
	auth, exists := profileAuthPath(filepath.Join(root, "home"))
	if !exists || !strings.EqualFold(readCodexAuthAccountID(auth), accountID) {
		return fmt.Errorf("a conta autenticada do perfil mudou")
	}
	return nil
}

// ResumeUnified launches under the verified profile without cross-account
// seeding. The account is checked again after acquiring the TUI lease.
func (a *Adapter) ResumeUnified(ctx context.Context, selected UnifiedResumeSession) (model.Failure, error) {
	if err := PrepareUnifiedResume(ctx, selected); err != nil {
		return model.Failure{Kind: model.FailureCommand, Message: err.Error()}, err
	}
	p := model.Profile{Provider: "codex", Name: selected.Profile}
	return a.runPrepared(p, []string{"resume", selected.ID}, selected.AccountID)
}

func appendUnifiedResumeIndex(home string, entry sessionIndexEntry) error {
	path := filepath.Join(home, "session_index.jsonl")
	entries, err := readSessionIndexEntries(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, existing := range entries {
		if existing.ID == entry.ID {
			return nil
		}
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(raw, '\n'))
	return err
}
