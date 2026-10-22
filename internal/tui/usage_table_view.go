package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/buildinfo"
	"github.com/IAPro-Community/nexus/internal/profile"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// View-only rendering for the existing usageTableModel.
// State transitions, terminal events, sorting and quota semantics remain unchanged.
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
	if m.modalMode == ModalAddProfile {
		return m.renderAddProfileModal(availWidth)
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
		subStyle.Render("v"+buildinfo.Version),
		safePill, yoloPill, planPill, contPill)
	headerRight := subStyle.Render("~/" + wsBasename)
	gap := availWidth - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	header := headerLeft + strings.Repeat(" ", max(1, gap)) + headerRight
	header = clipToWidth(header, availWidth)

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
	tabsLine := clipToWidth(fmt.Sprintf("%s  %s  %s", tab1, tab2, filterHint), availWidth)

	tableRows := tableBodyHeight(m.height)
	var tableContent string
	if m.activeTab == TabAccounts {
		tableContent = m.renderAccountTable(availWidth, tableRows)
	} else {
		tableContent = m.sessionTable.View()
	}

	var previewLine string
	if m.activeTab == TabAccounts {
		idx := m.accountTable.Cursor()
		if idx >= 0 && idx < len(m.filteredAccounts) {
			row := m.filteredAccounts[idx]
			if row.IsUnconfigured {
				previewLine = fmt.Sprintf("▶ ⏎/a adicionar perfil %s (nexus add %s <nome>)", row.Provider, row.Provider)
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

	helpText := "↑↓  ⏎ lançar  a add  r conta  R tudo  s detalhe  d padrão  l login  / filtro  q sair"
	if m.filter.Focused() {
		helpText = "digite para filtrar · Enter/Esc conclui"
	} else if m.activeTab == TabAccounts && m.accountTable.Cursor() >= 0 && m.accountTable.Cursor() < len(m.filteredAccounts) && m.filteredAccounts[m.accountTable.Cursor()].IsUnconfigured {
		helpText = "↑↓  ⏎/a adicionar perfil  r conta  R tudo  s detalhe  / filtro  q sair"
	} else if availWidth < 90 {
		helpText = "↑↓  ⏎ lançar  a add  s detalhe  / filtro  q sair"
	}
	help := subStyle.Render(truncateRunes(helpText, availWidth))
	previewLine = clipToWidth(previewLine, availWidth)

	// Never clip preview/help: if the frame overruns, shrink the table body.
	body := lipgloss.JoinVertical(lipgloss.Left, header, tabsLine, tableContent, previewLine, help)
	if m.height > 0 && lipgloss.Height(body) > m.height {
		shrink := lipgloss.Height(body) - m.height
		tableRows = max(1, tableRows-shrink)
		if m.activeTab == TabAccounts {
			tableContent = m.renderAccountTable(availWidth, tableRows)
		}
		body = lipgloss.JoinVertical(lipgloss.Left, header, tabsLine, tableContent, previewLine, help)
	}
	return fitViewToTerminal(body, availWidth, m.height)
}

func clipToWidth(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(width).Inline(true).Render(s)
}

func fitViewToTerminal(content string, width, height int) string {
	lines := strings.Split(content, "\n")
	if width > 0 {
		for i, line := range lines {
			if lipgloss.Width(line) > width {
				lines[i] = clipToWidth(line, width)
			}
		}
	}
	if height > 0 && len(lines) > height {
		// Prefer keeping the last two chrome lines (preview + help) when possible.
		if height >= 2 && len(lines) >= 2 {
			kept := make([]string, 0, height)
			bodyBudget := height - 2
			if bodyBudget > len(lines)-2 {
				bodyBudget = len(lines) - 2
			}
			kept = append(kept, lines[:bodyBudget]...)
			kept = append(kept, lines[len(lines)-2], lines[len(lines)-1])
			lines = kept
		} else {
			lines = lines[:height]
		}
	}
	content = strings.Join(lines, "\n")
	if height > 0 {
		if extra := height - len(lines); extra > 0 {
			content += strings.Repeat("\n", extra)
		}
	}
	return content
}

func (m usageTableModel) renderAccountTable(width, bodyRows int) string {
	cols := usageColumns(width)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).
		BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("57")).Bold(true)

	headerCells := make([]string, 0, len(cols))
	for _, c := range cols {
		headerCells = append(headerCells, renderTableCell(c.Title, c.Width, headerStyle))
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerCells...)

	cursor := m.accountTable.Cursor()
	if cursor < 0 && len(m.filteredAccounts) > 0 {
		cursor = 0
	}
	start, end := accountViewport(len(m.filteredAccounts), cursor, bodyRows)
	lines := make([]string, 0, 1+(end-start))
	lines = append(lines, header)
	for i := start; i < end; i++ {
		row := m.filteredAccounts[i]
		cells := renderAccountCells(row, cols, i+1)
		line := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
		if i == cursor {
			line = selectedStyle.Width(width).MaxWidth(width).Render(line)
		}
		lines = append(lines, clipToWidth(line, width))
	}
	// Pad empty viewport rows so preview/help stay pinned to the bottom.
	for len(lines)-1 < bodyRows {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func accountViewport(total, cursor, bodyRows int) (start, end int) {
	if total <= 0 || bodyRows <= 0 {
		return 0, 0
	}
	if bodyRows >= total {
		return 0, total
	}
	start = cursor - bodyRows/2
	if start < 0 {
		start = 0
	}
	end = start + bodyRows
	if end > total {
		end = total
		start = end - bodyRows
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

func renderAccountCells(row UsageTableRow, cols []table.Column, index int) []string {
	accountWidth := 24
	fiveWidth, weeklyWidth := 0, 0
	for _, c := range cols {
		switch c.Title {
		case "CONTA":
			accountWidth = c.Width
		case "5H":
			fiveWidth = c.Width
		case "SEMANA":
			weeklyWidth = c.Width
		}
	}
	star := ""
	if row.IsDefault {
		star = "★"
	}
	cells := make([]string, 0, len(cols))
	for _, c := range cols {
		switch c.Title {
		case "#":
			cells = append(cells, renderTableCell(strconv.Itoa(index), c.Width, lipgloss.NewStyle().Foreground(lipgloss.Color("242"))))
		case "PROV":
			cells = append(cells, renderTableCell(formatProviderBadge(row.Provider), c.Width, lipgloss.NewStyle()))
		case "CONTA":
			cells = append(cells, renderTableCell(formatAccountIdentity(row, accountWidth, star), c.Width, lipgloss.NewStyle()))
		case "5H":
			cells = append(cells, renderTableCell(formatQuotaCell(row.FiveHour, fiveWidth, row.Status), c.Width, usageQuotaStyle(row.Status, row.FiveHour)))
		case "SEMANA":
			cells = append(cells, renderTableCell(formatQuotaCell(row.Weekly, weeklyWidth, row.Status), c.Width, usageQuotaStyle(row.Status, row.Weekly)))
		case "STATUS":
			statusDisplay := row.Status
			if !row.FetchedAt.IsZero() && time.Since(row.FetchedAt) > staleQuotaAge {
				statusDisplay = statusDisplay + " · " + compactAge(row.FetchedAt)
			}
			cells = append(cells, renderTableCell(statusDisplay, c.Width, usageStatusStyle(row.Status)))
		default:
			cells = append(cells, renderTableCell("", c.Width, lipgloss.NewStyle()))
		}
	}
	return cells
}

func renderTableCell(value string, width int, style lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	plain := truncateMiddle(value, width)
	inner := lipgloss.NewStyle().Width(width).MaxWidth(width).Inline(true).Render(plain)
	return style.Padding(0, 1).Render(inner)
}

func formatQuotaCell(raw string, width int, status string) string {
	if raw == "" || raw == "—" {
		return "—"
	}
	if width <= 0 || lipgloss.Width(raw) <= width {
		return raw
	}
	// For rate-limited accounts, the reset time is the most actionable signal.
	// Prefer keeping percent + reset and drop the bar when the column is tight.
	isRateLimited := status == "RATE LTD" || status == "RATE LIMITED"
	if idx := strings.Index(raw, " · "); idx > 0 {
		if isRateLimited {
			textReset := strings.TrimSpace(raw[idx+3:])
			percent := strings.TrimSpace(raw[:idx])
			if idx2 := strings.Index(percent, "] "); idx2 > 0 {
				percent = strings.TrimSpace(percent[idx2+2:])
			}
			keep := percent + " · " + textReset
			if lipgloss.Width(keep) <= width {
				return keep
			}
		}
		compact := strings.TrimSpace(raw[:idx])
		if lipgloss.Width(compact) <= width {
			return compact
		}
	}
	return truncateMiddle(raw, width)
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

func usageQuotaStyle(status, value string) lipgloss.Style {
	if quotaLooksExhausted(value) {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	}
	switch status {
	case "ESGOTADA", "RATE LTD", "SEM QUOTA", "RATE LIMITED":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	case "OK", "COM QUOTA":
		pct, ok := parseQuotaPercent(value)
		if ok && pct <= 30 {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	case "ESTIMADA":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	}
}

func quotaLooksExhausted(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.Contains(value, "  0%") || strings.Contains(value, " 0%") || strings.HasPrefix(trimmed, "0%")
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
	// Width is content before padding/border. Keep the painted box on-screen.
	boxWidth := width - 6
	if boxWidth < 24 {
		boxWidth = max(10, width-2)
	}
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Width(boxWidth).
		MaxWidth(width).
		Padding(1, 1)

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
	sb.WriteString(titleStyle.Render(fmt.Sprintf("Limites & Quota — %s:%s", row.Provider, row.Profile)) + "\n\n")
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

	return fitViewToTerminal(modalStyle.Render(sb.String()), width, m.height)
}

func (m usageTableModel) renderAddProfileModal(width int) string {
	boxWidth := width - 6
	if boxWidth < 36 {
		boxWidth = max(20, width-2)
	}
	if boxWidth > 64 {
		boxWidth = 64
	}
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Width(boxWidth).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	accentStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Configurar Provedor · "+accentStyle.Render(m.addProfileProvider)) + "\n\n")
	fmt.Fprintf(&sb, "Informe o nome do novo perfil para %s:\n", accentStyle.Render(m.addProfileProvider))
	sb.WriteString(subStyle.Render("(Enter vazio usa 'default')") + "\n\n")
	sb.WriteString(m.addProfileInput.View() + "\n\n")
	sb.WriteString(subStyle.Render("⏎ confirmar e autenticar  ·  Esc cancelar"))

	rendered := modalStyle.Render(sb.String())
	return fitViewToTerminal(rendered, width, m.height)
}
