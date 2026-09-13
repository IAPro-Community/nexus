package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kivervinicius/ai-cli/internal/conversation"
	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
	"github.com/kivervinicius/ai-cli/internal/localization"
	"github.com/kivervinicius/ai-cli/internal/profile"
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

const quotaPollInterval = 60 * time.Second

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

// buildUsageRows renders one presentation row per model group of a quota view.
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
			bar := miniQuotaBar(window.Remaining, 8)
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

	styles := table.DefaultStyles()
	styles.Header = styles.Header.Bold(true).Foreground(lipgloss.Color("39")).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true)
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("15")).Background(lipgloss.Color("57")).Bold(true)

	initW, initH := 120, 30
	if tw, th, err := term.GetSize(int(os.Stdout.Fd())); err == nil && tw > 0 && th > 0 {
		initW, initH = tw, th
	}

	// Accounts table setup
	filteredAccounts := make([]UsageTableRow, len(allRows))
	copy(filteredAccounts, allRows)
	accTable := table.New(table.WithColumns(usageColumns(initW)), table.WithRows(toUsageTableRowsForWidth(filteredAccounts, initW)), table.WithFocused(true), table.WithHeight(tableBodyHeight(initH)))
	accTable.SetStyles(styles)

	// Sessions table setup
	allSess := opts.Sessions
	filteredSess := make([]conversation.Conversation, len(allSess))
	copy(filteredSess, allSess)
	sessTable := table.New(table.WithColumns(sessionColumns(initW)), table.WithRows(toSessionTableRowsForWidth(filteredSess, initW)), table.WithFocused(false), table.WithHeight(tableBodyHeight(initH)))
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
		q := m.filter.Value()
		m.filteredAccounts = filterUsageRows(m.allAccounts, q)
		m.setAccountRows()
		if msg.accounts != nil {
			if m.options.Accounts == nil {
				m.options.Accounts = map[string]model.AccountInfo{}
			}
			for k, v := range msg.accounts {
				m.options.Accounts[k] = v
			}
		}
		if !msg.silent {
			m.statusMsg = "Quotas atualizadas"
		}
		return m, nil

	case quotaPollTickMsg:
		next := quotaPollTick()
		if m.loadingQuotas || m.modalMode != ModalNone || len(m.options.ProfilesToLoad) == 0 {
			return m, next
		}
		return m, tea.Batch(loadQuotasCmdOpts(m.options.ProfilesToLoad, m.options.Defaults, true), next)

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
		m.filteredAccounts = filterUsageRows(m.allAccounts, m.filter.Value())
		m.setAccountRows()
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

	case tea.KeyMsg:
		k := msg.String()

		// Global quit: ctrl+c always exits immediately
		if k == "ctrl+c" {
			m.quitting = true
			m.chosenResult = &SelectionResult{Action: ActionQuit}
			return m, tea.Quit
		}

		// Handle active modal
		if m.modalMode != ModalNone {
			if k == "esc" || k == "q" || k == "Q" || k == "enter" || k == " " {
				m.modalMode = ModalNone
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
					m.filteredAccounts = filterUsageRows(m.allAccounts, q)
					m.setAccountRows()
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
		case "r", "R":
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
			// Hint on adding profile
			switch m.activeTab {
			case TabAccounts:
				if len(m.filteredAccounts) > 0 {
					idx := m.accountTable.Cursor()
					if idx >= 0 && idx < len(m.filteredAccounts) {
						row := m.filteredAccounts[idx]
						m.statusMsg = fmt.Sprintf("Para configurar %s: execute 'nexus add %s <nome>'", row.Provider, row.Provider)
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
						m.statusMsg = fmt.Sprintf("⚠ %s não está configurado. Execute: nexus add %s <nome>", row.Provider, row.Provider)
						return m, nil
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
	h := tableBodyHeight(m.height)
	m.accountTable.SetColumns(usageColumns(m.width))
	m.accountTable.SetWidth(w)
	m.accountTable.SetHeight(h)
	m.accountTable.SetRows(toUsageTableRowsForWidth(m.filteredAccounts, m.width))
	m.sessionTable.SetColumns(sessionColumns(m.width))
	m.sessionTable.SetWidth(w)
	m.sessionTable.SetHeight(h)
	m.sessionTable.SetRows(toSessionTableRowsForWidth(m.filteredSessions, m.width))
}

func (m *usageTableModel) setAccountRows() {
	m.accountTable.SetRows(toUsageTableRowsForWidth(m.filteredAccounts, m.width))
}

// usageChromeLines is the fixed chrome around the accounts/sessions table:
// header, tabs, blank after tabs, blank after table, preview, help.
const usageChromeLines = 6

func tableBodyHeight(termHeight int) int {
	h := termHeight - usageChromeLines
	if h < 6 {
		return 6
	}
	return h
}

func (m usageTableModel) View() string {
	if m.quitting {
		return ""
	}

	availWidth := m.width
	if availWidth < 40 {
		availWidth = 40
	}

	if m.modalMode == ModalQuotaDetails {
		return m.renderQuotaModal(availWidth)
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	accentStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	planStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("213"))

	cwd := m.options.Workspace
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	wsBasename := filepath.Base(cwd)

	safePill := subStyle.Render("1 Safe")
	yoloPill := subStyle.Render("2 YOLO")
	planPill := subStyle.Render("3 Plan")
	switch m.execMode {
	case ModeSafe:
		safePill = accentStyle.Render("●1 Safe")
	case ModeYOLO:
		yoloPill = warnStyle.Render("●2 YOLO")
	case ModePlan:
		planPill = planStyle.Render("●3 Plan")
	}
	contPill := subStyle.Render("c Continuar:off")
	if m.continueSession {
		contPill = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).Render("c Continuar:on")
	}

	headerLeft := fmt.Sprintf("%s %s  %s %s %s  %s",
		titleStyle.Render("Nexus"),
		subStyle.Render("v0.4.0"),
		safePill, yoloPill, planPill, contPill)
	headerRight := subStyle.Render("~/" + wsBasename)
	gap := availWidth - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	if gap < 1 {
		gap = 1
	}
	header := headerLeft + strings.Repeat(" ", gap) + headerRight

	tab1Label := fmt.Sprintf("Contas %d", len(m.filteredAccounts))
	tab2Label := fmt.Sprintf("Sessões %d", len(m.filteredSessions))
	tab1 := subStyle.Render(tab1Label)
	tab2 := subStyle.Render(tab2Label)
	if m.activeTab == TabAccounts {
		tab1 = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("57")).Render(" " + tab1Label + " ")
	} else {
		tab2 = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("57")).Render(" " + tab2Label + " ")
	}
	filterHint := subStyle.Render("/ filtro")
	if m.filter.Focused() {
		filterHint = m.filter.View()
	}
	tabsLine := fmt.Sprintf("%s  %s  %s", tab1, tab2, filterHint)

	var tableContent string
	if m.activeTab == TabAccounts {
		tableContent = m.accountTable.View()
	} else {
		tableContent = m.sessionTable.View()
	}

	var previewLine string
	if m.activeTab == TabAccounts {
		idx := m.accountTable.Cursor()
		if idx >= 0 && idx < len(m.filteredAccounts) {
			row := m.filteredAccounts[idx]
			if row.IsUnconfigured {
				previewLine = fmt.Sprintf("▶ %s sem perfil — nexus add %s <nome>", row.Provider, row.Provider)
			} else {
				flagNote := m.execMode.String()
				if m.execMode == ModeYOLO && m.continueSession {
					flagNote = "YOLO+continue"
				} else if m.execMode == ModePlan && m.continueSession {
					flagNote = "Plan+continue"
				} else if m.continueSession {
					flagNote = "Safe+continue"
				}
				extra := ""
				if row.FiveHour != "" && row.FiveHour != "—" && (availWidth < 80 || strings.Contains(row.FiveHour, "SEM")) {
					extra = " · 5h " + row.FiveHour
				}
				previewLine = fmt.Sprintf("▶ ⏎ %s:%s (%s)%s", row.Provider, row.Profile, flagNote, extra)
				if fresh := freshnessLine(row.SnapshotStatus, row.Source, row.FetchedAt); fresh != "" {
					candidate := previewLine + subStyle.Render(" · "+fresh)
					if lipgloss.Width(candidate) <= availWidth {
						previewLine = candidate
					}
				}
			}
		}
	} else {
		idx := m.sessionTable.Cursor()
		if idx >= 0 && idx < len(m.filteredSessions) {
			sess := m.filteredSessions[idx]
			id := sess.ID
			if len(id) > 8 {
				id = id[:8]
			}
			previewLine = fmt.Sprintf("▶ ⏎ retomar %s (%s) · %s", id, sess.Provider, m.execMode.String())
		}
	}

	status := ""
	if m.loadingQuotas || m.loadingSessions || len(m.refreshing) > 0 {
		frames := []string{"⠋", "⠙", "⠹", "⠸"}
		spin := frames[m.spinFrame%len(frames)]
		parts := []string{}
		if m.loadingQuotas {
			parts = append(parts, "quotas")
		}
		if m.loadingSessions {
			parts = append(parts, "sessões")
		}
		if n := len(m.refreshing); n > 0 {
			parts = append(parts, fmt.Sprintf("atualizando %d", n))
		}
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true).Render(
			fmt.Sprintf("%s %s…", spin, strings.Join(parts, "+")),
		)
	} else if m.statusMsg != "" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true).Render(m.statusMsg)
	}
	if status != "" {
		previewLine = previewLine + "  " + status
	}

	helpText := "↑↓  ⏎ lançar  r atualizar  s detalhe  d padrão  l login  / filtro  q sair"
	if m.filter.Focused() {
		helpText = "digite para filtrar · Enter/Esc conclui"
	}
	help := subStyle.Render(truncateRunes(helpText, availWidth))

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabsLine,
		tableContent,
		previewLine,
		help,
	)
}

func (m *usageTableModel) ensureQuotaModalCached() {
	idx := m.accountTable.Cursor()
	if idx < 0 || idx >= len(m.filteredAccounts) {
		return
	}
	row := m.filteredAccounts[idx]
	key := row.Provider + ":" + row.Profile
	if _, ok := m.cachedQuotaView[key]; ok {
		return
	}
	acc := m.options.Accounts[key]
	qv := profile.GetQuotaView(row.Provider, row.Profile, acc.Plan, acc.Email)
	if m.cachedQuotaView == nil {
		m.cachedQuotaView = map[string]quotaViewCache{}
	}
	m.cachedQuotaView[key] = quotaViewCache{view: qv, acc: acc}
}

func (m usageTableModel) renderQuotaModal(width int) string {
	boxWidth := width - 4
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Width(boxWidth).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	accentStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	magentaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("213"))
	groupStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))

	idx := m.accountTable.Cursor()
	if idx < 0 || idx >= len(m.filteredAccounts) {
		return modalStyle.Render("Nenhuma conta selecionada.\n\n[Pressione Esc ou q para voltar]")
	}
	row := m.filteredAccounts[idx]
	if row.IsUnconfigured {
		return modalStyle.Render(fmt.Sprintf("Provedor %s ainda não possui perfis.\nExecute: nexus add %s <nome>\n\n[Pressione Esc ou q para voltar]", row.Provider, row.Provider))
	}

	key := row.Provider + ":" + row.Profile
	cached, ok := m.cachedQuotaView[key]
	if !ok {
		acc := m.options.Accounts[key]
		cached = quotaViewCache{
			view: profile.GetQuotaView(row.Provider, row.Profile, acc.Plan, acc.Email),
			acc:  acc,
		}
	}
	acc := cached.acc
	qv := cached.view

	var sb strings.Builder
	sb.WriteString(titleStyle.Render(fmt.Sprintf("📊 Limites & Quota — %s:%s", row.Provider, row.Profile)) + "\n\n")
	fmt.Fprintf(&sb, " Conta:   %s (%s)\n", accentStyle.Render(acc.Email), magentaStyle.Render(acc.Plan))
	fmt.Fprintf(&sb, " Status:  %s\n", acc.Status)

	availLabel := qv.AvailabilityLabel()
	availColor := "42"
	if availLabel == "SEM DADOS" {
		availColor = "214"
	} else if !qv.IsAvailable() || availLabel == "QUOTA ESGOTADA" {
		availColor = "196"
	}
	availStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(availColor))
	fmt.Fprintf(&sb, " Quota:   %s\n", availStyle.Render(availLabel))
	fmt.Fprintf(&sb, " Leitura: %s\n", subStyle.Render(freshnessLine(qv.Status, qv.Source, qv.FetchedAt)))
	sb.WriteString("\n")

	sb.WriteString(accentStyle.Render("MODELOS & CAPACIDADE:") + "\n")
	hasWindows := false
	for _, group := range qv.ModelGroups {
		if qv.HasMultipleGroups() && group.Name != "" {
			fmt.Fprintf(&sb, "\n  %s\n", groupStyle.Render(group.Name+":"))
		}
		for _, w := range group.Windows {
			hasWindows = true
			fmt.Fprintf(&sb, "  %s: %s\n", w.Label, w.Bar)
			if line := formatResetLine(w.ResetDesc); line != "" {
				sb.WriteString(subStyle.Render(fmt.Sprintf("                  %s\n", line)))
			}
		}
	}
	if !hasWindows {
		sb.WriteString(subStyle.Render("  SEM DADOS — aguardando evidência de cota desta conta\n"))
	}
	sb.WriteString("\n")
	sb.WriteString(subStyle.Render("Pressione qualquer tecla ou Esc/q para voltar"))

	return modalStyle.Render(sb.String())
}

func usageColumns(width int) []table.Column {
	if width < 80 {
		accountW := max(18, width-8-10-3)
		return []table.Column{
			{Title: "PROV", Width: 8},
			{Title: "CONTA", Width: accountW},
			{Title: "STATUS", Width: 10},
		}
	}
	if width < 132 {
		// Drop weekly detail; keep 5h + status. Account takes remainder.
		fixed := 8 + 18 + 10 + 3
		accountW := max(20, width-fixed)
		return []table.Column{
			{Title: "PROV", Width: 8},
			{Title: "CONTA", Width: accountW},
			{Title: "5H", Width: 18},
			{Title: "STATUS", Width: 10},
		}
	}
	fixed := 8 + 22 + 24 + 10 + 4
	accountW := max(24, width-fixed)
	return []table.Column{
		{Title: "PROV", Width: 8},
		{Title: "CONTA", Width: accountW},
		{Title: "5H", Width: 22},
		{Title: "SEMANA", Width: 24},
		{Title: "STATUS", Width: 10},
	}
}

func sessionColumns(width int) []table.Column {
	if width < 70 {
		titleW := max(12, width-8-8-2)
		return []table.Column{
			{Title: "PROV", Width: 8},
			{Title: "SESSÃO", Width: 8},
			{Title: "TÍTULO", Width: titleW},
		}
	}
	if width < 110 {
		fixed := 10 + 10 + 12 + 14 + 4
		titleW := max(16, width-fixed)
		return []table.Column{
			{Title: "PROVEDOR", Width: 10},
			{Title: "SESSÃO", Width: 10},
			{Title: "MODIFICADO", Width: 12},
			{Title: "TÍTULO", Width: titleW},
			{Title: "WORKSPACE", Width: 14},
		}
	}
	fixed := 11 + 14 + 16 + 22 + 4
	titleW := max(24, width-fixed)
	return []table.Column{
		{Title: "PROVEDOR", Width: 11},
		{Title: "SESSÃO", Width: 14},
		{Title: "MODIFICADO", Width: 16},
		{Title: "TÍTULO", Width: titleW},
		{Title: "WORKSPACE", Width: 22},
	}
}

func toUsageTableRows(rows []UsageTableRow) []table.Row {
	return toUsageTableRowsForWidth(rows, 132)
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
	for _, row := range rows {
		provBadge := formatProviderBadge(row.Provider)
		star := ""
		if row.IsDefault {
			star = "★"
		}

		identity := formatAccountIdentity(row, accountWidth, star)
		statusStyle := usageStatusStyle(row.Status)
		quotaStyle := usageQuotaStyle(row.Status)

		cells := []string{provBadge, identity}
		if showFive {
			five := row.FiveHour
			if !showWeekly && five != "—" && five != "SEM DADOS" {
				// Mid width: keep compact percent only when weekly hidden.
				five = compactFiveHour(five)
			}
			cells = append(cells, quotaStyle.Render(five))
		}
		if showWeekly {
			cells = append(cells, quotaStyle.Render(row.Weekly))
		}
		cells = append(cells, statusStyle.Render(row.Status))
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

func usageStatusStyle(status string) lipgloss.Style {
	switch status {
	case "ESGOTADA", "RATE LTD", "SEM QUOTA", "RATE LIMITED":
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	case "OK", "COM QUOTA":
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	case "ESTIMADA":
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	case "SEM DADOS", "OFF":
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	default:
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	}
}

func usageQuotaStyle(status string) lipgloss.Style {
	switch status {
	case "ESGOTADA", "RATE LTD", "SEM QUOTA", "RATE LIMITED":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	case "OK", "COM QUOTA":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	case "ESTIMADA":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	}
}

func toSessionTableRows(sess []conversation.Conversation) []table.Row {
	return toSessionTableRowsForWidth(sess, 132)
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

func filterUsageRows(rows []UsageTableRow, query string) []UsageTableRow {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return rows
	}
	filtered := make([]UsageTableRow, 0, len(rows))
	for _, row := range rows {
		search := strings.ToLower(strings.Join([]string{
			row.Provider, row.Profile, row.Account, row.Plan,
			row.Group, row.FiveHour, row.Weekly, row.Status, row.ModelName,
		}, " "))
		if strings.Contains(search, query) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func filterSessionRows(sess []conversation.Conversation, query string) []conversation.Conversation {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return sess
	}
	var out []conversation.Conversation
	for _, s := range sess {
		search := strings.ToLower(strings.Join([]string{
			s.Provider, s.ID, s.Title, s.Workspace,
		}, " "))
		if strings.Contains(search, query) {
			out = append(out, s)
		}
	}
	return out
}

func formatProviderBadge(p string) string {
	switch strings.ToLower(p) {
	case "codex":
		return "CODEX"
	case "agy":
		return "AGY"
	case "claude":
		return "CLAUDE"
	case "opencode":
		return "OPENCD"
	case "gemini":
		return "GEMINI"
	case "cursor":
		return "CURSOR"
	default:
		s := strings.ToUpper(p)
		if len(s) > 6 {
			s = s[:6]
		}
		return s
	}
}

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return localization.T("tui.unknown")
	}
	diff := time.Since(t)
	if diff < time.Minute {
		return localization.T("tui.now")
	}
	if diff < time.Hour {
		return localization.T("tui.minutes_ago", map[string]any{"Count": int(diff.Minutes())})
	}
	if diff < 24*time.Hour {
		return localization.T("tui.hours_ago", map[string]any{"Count": int(diff.Hours())})
	}
	if diff < 48*time.Hour {
		return localization.T("tui.yesterday")
	}
	return localization.T("tui.days_ago", map[string]any{"Count": int(diff.Hours() / 24)})
}
