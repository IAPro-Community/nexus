package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kivervinicius/ai-cli/internal/conversation"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
)

func TestFilterUsageRowsFindsAcrossIdentityAndGroup(t *testing.T) {
	rows := []UsageTableRow{{Provider: "agy", Profile: "work", Account: "work@example.com", Group: "Gemini Models"}, {Provider: "codex", Profile: "personal", Group: "GPT Models"}}
	if got := filterUsageRows(rows, "gemini"); len(got) != 1 || got[0].Profile != "work" {
		t.Fatalf("group filter = %#v", got)
	}
	if got := filterUsageRows(rows, "personal"); len(got) != 1 || got[0].Provider != "codex" {
		t.Fatalf("profile filter = %#v", got)
	}
}

func TestFilterUsageRowsIncludesUnknownProviders(t *testing.T) {
	rows := []UsageTableRow{
		{Provider: "agy", Profile: "work", FiveHour: "90%", Weekly: "80%", Status: "DISPONIVEL"},
		{Provider: "codex", Profile: "personal", FiveHour: "— desconhecido", Weekly: "— desconhecido", Status: "UNKNOWN"},
	}
	if got := filterUsageRows(rows, "codex"); len(got) != 1 || got[0].Status != "UNKNOWN" {
		t.Fatalf("expected codex UNKNOWN row, got %#v", got)
	}
	// Empty filter returns all
	if got := filterUsageRows(rows, ""); len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}
}

func TestUnifiedUsageEscAndQQuit(t *testing.T) {
	opts := UnifiedUsageOptions{
		Rows: []UsageTableRow{{Provider: "codex", Profile: "work"}},
	}
	m := newUnifiedUsageModel(opts)

	// Test 'q' key
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit cmd on 'q'")
	}
	fm := res.(usageTableModel)
	if !fm.quitting || fm.chosenResult == nil || fm.chosenResult.Action != ActionQuit {
		t.Fatalf("expected ActionQuit on 'q', got %#v", fm.chosenResult)
	}

	// Test 'esc' key
	m2 := newUnifiedUsageModel(opts)
	res2, cmd2 := m2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd2 == nil {
		t.Fatal("expected quit cmd on 'esc'")
	}
	fm2 := res2.(usageTableModel)
	if !fm2.quitting || fm2.chosenResult == nil || fm2.chosenResult.Action != ActionQuit {
		t.Fatalf("expected ActionQuit on 'esc', got %#v", fm2.chosenResult)
	}
}

func TestUnifiedUsageUnconfiguredCLIsRendered(t *testing.T) {
	opts := UnifiedUsageOptions{
		Rows: []UsageTableRow{
			{Provider: "codex", Profile: "work", Status: "DISPONIVEL"},
		},
		UnconfiguredCLIs: []InstalledProviderInfo{
			{ID: "claude", Name: "Claude Code", Version: "2.1.250", Installed: true, Profiles: 0},
			{ID: "cursor", Name: "Cursor CLI", Version: "2026.08", Installed: true, Profiles: 0},
		},
	}
	m := newUnifiedUsageModel(opts)
	if len(m.allAccounts) != 3 {
		t.Fatalf("expected 3 rows (1 configured + 2 unconfigured), got %d", len(m.allAccounts))
	}

	foundClaude := false
	for _, r := range m.allAccounts {
		if r.Provider == "claude" {
			foundClaude = true
			if !r.IsUnconfigured || r.Status != "OFF" {
				t.Fatalf("expected Claude to be unconfigured OFF, got %#v", r)
			}
		}
	}
	if !foundClaude {
		t.Fatal("expected unconfigured Claude row in table")
	}
}

func TestUnifiedUsageModeTogglingAndFlags(t *testing.T) {
	opts := UnifiedUsageOptions{
		Rows:        []UsageTableRow{{Provider: "codex", Profile: "work"}},
		InitialMode: ModeSafe,
	}
	m := newUnifiedUsageModel(opts)

	// Press '2' or 'y' for YOLO
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = res.(usageTableModel)
	if m.execMode != ModeYOLO {
		t.Fatalf("expected YOLO mode, got %v", m.execMode)
	}

	// Press 'c' to toggle continue
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = res.(usageTableModel)
	if !m.continueSession {
		t.Fatal("expected continueSession to be true")
	}

	// Press Enter to select
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected quit cmd on enter")
	}
	fm := res.(usageTableModel)
	if fm.chosenResult == nil {
		t.Fatal("expected non-nil chosenResult")
	}
	if fm.chosenResult.Action != ActionRunProfile {
		t.Fatalf("expected ActionRunProfile, got %v", fm.chosenResult.Action)
	}
	// Flags should include --yolo and --continue
	expectedFlags := []string{"--yolo", "--continue"}
	if len(fm.chosenResult.Args) != len(expectedFlags) {
		t.Fatalf("expected args %v, got %v", expectedFlags, fm.chosenResult.Args)
	}
	if fm.chosenResult.Args[0] != "--yolo" || fm.chosenResult.Args[1] != "--continue" {
		t.Fatalf("unexpected args: %v", fm.chosenResult.Args)
	}
}

func TestUnifiedUsageRestoresSelectionAfterEmptyFilter(t *testing.T) {
	opts := UnifiedUsageOptions{
		Rows: []UsageTableRow{
			{Provider: "codex", Profile: "work"},
			{Provider: "agy", Profile: "personal"},
		},
	}
	m := newUnifiedUsageModel(opts)

	// Enter filter mode and produce no matching rows. bubbles/table sets its
	// cursor to -1 for an empty result set.
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = res.(usageTableModel)
	for _, r := range "does-not-exist" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(usageTableModel)
	}

	// Replace the query with an empty value through the same update path used by
	// textinput after the user deletes its contents.
	m.filter.SetValue("")
	m.filter.Focus()
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = res.(usageTableModel)

	if m.accountTable.Cursor() < 0 {
		t.Fatalf("expected a valid cursor after filtered rows return, got %d", m.accountTable.Cursor())
	}
	// Enter both confirms the filter and chooses the highlighted row.
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(usageTableModel)
	if cmd == nil || m.chosenResult == nil {
		t.Fatalf("expected Enter to select the restored row, result=%#v cmd=%v", m.chosenResult, cmd)
	}
	if m.chosenResult.ProfileName != "work" {
		t.Fatalf("expected first restored row to be selected, got %#v", m.chosenResult)
	}
}

func TestUsageColumnsAdaptiveWidths(t *testing.T) {
	for _, width := range []int{60, 80, 132, 160} {
		cols := usageColumns(width)
		sum := 0
		statusW := 0
		hasModel := false
		for _, c := range cols {
			sum += c.Width
			if c.Title == "STATUS" {
				statusW = c.Width
			}
			if c.Title == "MODELO" || c.Title == "MODEL" {
				hasModel = true
			}
		}
		if hasModel {
			t.Fatalf("width %d should not include MODELO column", width)
		}
		if statusW < 10 {
			t.Fatalf("width %d STATUS=%d want >=10 so SEM DADOS fits", width, statusW)
		}
		if sum > width {
			t.Fatalf("width %d column sum %d exceeds terminal width", width, sum)
		}
	}
}

func TestSessionColumnsAdaptiveWidths(t *testing.T) {
	for _, width := range []int{60, 80, 100, 132} {
		cols := sessionColumns(width)
		sum := 0
		for _, c := range cols {
			sum += c.Width
		}
		if sum > width {
			t.Fatalf("width %d session column sum %d exceeds terminal", width, sum)
		}
		rows := toSessionTableRowsForWidth([]conversation.Conversation{{
			Provider: "agy", ID: "sess-abcdefgh", Title: "a very long conversation title for truncation",
			Workspace: "/tmp/workspace-long-name", LastModified: time.Now(),
		}}, width)
		if len(rows) != 1 {
			t.Fatalf("width %d rows=%d", width, len(rows))
		}
		if len(rows[0]) != len(cols) {
			t.Fatalf("width %d row cells=%d cols=%d", width, len(rows[0]), len(cols))
		}
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("abcdef", 4); got != "ab.." {
		t.Fatalf("got %q", got)
	}
	if got := truncateRunes("ab", 4); got != "ab" {
		t.Fatalf("got %q", got)
	}
}

func TestUsageStatusLabelsFitStatusColumn(t *testing.T) {
	labels := []string{"OK", "ESGOTADA", "SEM DADOS", "OFF", "ESTIMADA"}
	for _, width := range []int{80, 132, 160} {
		cols := usageColumns(width)
		statusW := 0
		for _, c := range cols {
			if c.Title == "STATUS" {
				statusW = c.Width
			}
		}
		for _, label := range labels {
			if len([]rune(label)) > statusW {
				t.Fatalf("label %q len=%d exceeds STATUS width %d at cols=%d", label, len([]rune(label)), statusW, width)
			}
			if strings.Contains(label, "…") || strings.Contains(label, "..") {
				t.Fatalf("status label must not be truncated form: %q", label)
			}
		}
	}
}

func TestTableBodyHeightFillsTerminal(t *testing.T) {
	// Screenshot-like terminal with 10 account rows must leave table tall enough.
	h := tableBodyHeight(30)
	if h < 10 {
		t.Fatalf("tableBodyHeight(30)=%d want >=10 to fill chrome-compact layout", h)
	}
	if h != 30-usageChromeLines {
		t.Fatalf("tableBodyHeight(30)=%d want %d", h, 30-usageChromeLines)
	}
}

func TestFormatGroupStatusHonestLabels(t *testing.T) {
	if got := formatGroupStatus(quota.ModelGroup{}, string(model.UsageUnknown)); got != "SEM DADOS" {
		t.Fatalf("unknown=%q", got)
	}
	zero := 0.0
	exhausted := quota.ModelGroup{Windows: []quota.Window{{Kind: "5h", Remaining: zero}}}
	if got := formatGroupStatus(exhausted, string(model.UsageLive)); got != "ESGOTADA" {
		t.Fatalf("exhausted=%q", got)
	}
}

func TestFreshnessLineNamesStatusSourceAndAge(t *testing.T) {
	got := freshnessLine(string(model.UsageLive), string(model.SourceOfficialAPI), time.Now())
	if got != "AO VIVO · API oficial · agora" {
		t.Fatalf("live=%q", got)
	}
	got = freshnessLine(string(model.UsageEstimated), string(model.SourceObservation), time.Now().Add(-7*time.Minute))
	if got != "ESTIMADA · rollout local · há 7m" {
		t.Fatalf("estimated=%q", got)
	}
	got = freshnessLine(string(model.UsageUnknown), string(model.SourceNone), time.Time{})
	if got != "SEM DADOS · sem fonte · nunca" {
		t.Fatalf("unknown=%q", got)
	}
}

func TestFormatResetLineStripsEnglishPrefixes(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Refreshes in 46h 13m", "Reset em 46h 13m"},
		{"refreshes in 46h 53m", "Reset em 46h 53m"},
		{"resets 08:37 on 15 Sep", "Reset em 08:37 15 Sep"},
		{"Quota available", "Quota disponível"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := formatResetLine(tc.in); got != tc.want {
			t.Fatalf("formatResetLine(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatQuotaWindowShowsNumbersWhenRateLimited(t *testing.T) {
	windows := []quota.Window{{Kind: "5h", Remaining: 0, ResetDesc: "resets 21:55"}}
	got := formatQuotaWindow(windows, "5h", string(model.UsageRateLimited))
	if !strings.Contains(got, "0%") || !strings.Contains(got, "21:55") {
		t.Fatalf("rate limited window must keep its measured values, got %q", got)
	}
	if strings.Contains(got, "rate ltd") {
		t.Fatalf("rate limited window must not hide numbers behind a label, got %q", got)
	}
}

func TestBuildUsageRowsCarriesSourceAndRawStatus(t *testing.T) {
	remaining := 56.0
	snap := model.UsageSnapshot{
		ProviderID: "codex",
		ProfileID:  "kivergmail",
		Status:     model.UsageLive,
		Source:     model.SourceOfficialAPI,
		FetchedAt:  time.Now(),
		Windows: []model.UsageWindow{
			{Kind: "weekly", Group: "claude_gpt", RemainingPercent: &remaining, ResetDescription: "resets 08:37 on 15 Sep"},
		},
	}
	qv := quota.BuildQuotaView(snap, "kiver.omegaedu@gmail.com", "ChatGPT Plus")
	rows := buildUsageRows("codex", "kivergmail", qv, model.AccountInfo{Email: "kiver.omegaedu@gmail.com"}, nil)
	if len(rows) != 1 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[0].Source != string(model.SourceOfficialAPI) {
		t.Fatalf("source=%q", rows[0].Source)
	}
	if rows[0].SnapshotStatus != string(model.UsageLive) {
		t.Fatalf("snapshot status=%q", rows[0].SnapshotStatus)
	}
	if rows[0].FetchedAt.IsZero() {
		t.Fatal("FetchedAt must be carried to the row")
	}
}

func TestBuildUsageRowsWithoutGroupsIsSemDados(t *testing.T) {
	qv := quota.QuotaView{Provider: "codex", Profile: "novo", Status: string(model.UsageUnknown), Source: string(model.SourceNone)}
	rows := buildUsageRows("codex", "novo", qv, model.AccountInfo{}, nil)
	if len(rows) != 1 || rows[0].Status != "SEM DADOS" {
		t.Fatalf("rows=%#v", rows)
	}
}

func TestRefreshKeyMarksAccountInFlight(t *testing.T) {
	opts := UnifiedUsageOptions{
		Rows: []UsageTableRow{{Provider: "codex", Profile: "work", Status: "OK"}},
	}
	m := newUnifiedUsageModel(opts)
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = res.(usageTableModel)
	if cmd == nil {
		t.Fatal("expected an async refresh command")
	}
	if !m.refreshing["codex:work"] {
		t.Fatalf("expected codex:work marked in flight, got %#v", m.refreshing)
	}
	if m.quitting {
		t.Fatal("refresh must not quit the TUI")
	}
	// A second press while in flight must not enqueue another probe.
	res2, cmd2 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd2 != nil {
		t.Fatal("expected no second command while a refresh is in flight")
	}
	if len(res2.(usageTableModel).refreshing) != 1 {
		t.Fatal("expected a single in-flight refresh")
	}
}

func TestAccountRefreshedReplacesOnlyThatAccount(t *testing.T) {
	opts := UnifiedUsageOptions{
		Rows: []UsageTableRow{
			{Provider: "codex", Profile: "work", Status: "SEM DADOS"},
			{Provider: "agy", Profile: "personal", Status: "OK"},
		},
		UnconfiguredCLIs: []InstalledProviderInfo{{ID: "claude", Name: "Claude Code", Installed: true, Profiles: 0}},
	}
	m := newUnifiedUsageModel(opts)
	m.refreshing = map[string]bool{"codex:work": true}

	res, _ := m.Update(accountRefreshedMsg{
		provider: "codex",
		profile:  "work",
		rows: []UsageTableRow{{
			Provider: "codex", Profile: "work", Status: "OK",
			SnapshotStatus: string(model.UsageLive), Source: string(model.SourceOfficialAPI),
			FetchedAt: time.Now(),
		}},
		acc: model.AccountInfo{Email: "work@example.com"},
	})
	m = res.(usageTableModel)

	if m.refreshing["codex:work"] {
		t.Fatal("in-flight marker must be cleared")
	}
	var codexStatus, agyStatus string
	claudeKept := false
	for _, row := range m.allAccounts {
		switch {
		case row.Provider == "codex" && row.Profile == "work":
			codexStatus = row.Status
		case row.Provider == "agy":
			agyStatus = row.Status
		case row.Provider == "claude":
			claudeKept = true
		}
	}
	if codexStatus != "OK" {
		t.Fatalf("codex status=%q want refreshed OK", codexStatus)
	}
	if agyStatus != "OK" {
		t.Fatalf("agy row must be untouched, got %q", agyStatus)
	}
	if !claudeKept {
		t.Fatal("unconfigured CLI rows must survive a single-account refresh")
	}
	if !strings.Contains(m.statusMsg, "AO VIVO") {
		t.Fatalf("status message should report the new freshness, got %q", m.statusMsg)
	}
}
