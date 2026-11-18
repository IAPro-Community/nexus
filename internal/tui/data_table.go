package tui

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DataTableColumn and DataTableRow keep terminal table presentation generic and
// independent from domain packages.
type DataTableColumn struct {
	Title string
	Width int
}
type DataTableRow struct {
	Values     []string
	SearchText string
	Key        string
	Disabled   bool
	Detail     string
}
type DataTableOptions struct {
	Title, FilterPlaceholder string
	Columns                  []DataTableColumn
	Rows                     []DataTableRow
}

// RunDataTable renders the shared searchable table used by human CLI output.
func RunDataTable(options DataTableOptions) error {
	_, err := tea.NewProgram(newDataTableModel(options), tea.WithAltScreen()).Run()
	return err
}

// SelectDataTable returns the selected row key, or an empty key on cancel.
func SelectDataTable(options DataTableOptions) (string, error) {
	m := newDataTableModel(options)
	m.selectable = true
	final, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return "", err
	}
	return final.(dataTableModel).selected, nil
}

type dataTableModel struct {
	options    DataTableOptions
	table      table.Model
	filter     textinput.Model
	selectable bool
	selected   string
	width      int
}

func newDataTableModel(options DataTableOptions) dataTableModel {
	filter := textinput.New()
	filter.Placeholder = options.FilterPlaceholder
	filter.Prompt = "/ "
	filter.CharLimit = 120
	styles := table.DefaultStyles()
	styles.Header = styles.Header.Bold(true).Foreground(lipgloss.Color("39")).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).BorderBottom(true)
	styles.Selected = styles.Selected.Foreground(lipgloss.Color("15")).Background(lipgloss.Color("57")).Bold(true)
	t := table.New(table.WithColumns(toTableColumns(options.Columns)), table.WithRows(toTableRows(options.Rows)), table.WithFocused(true), table.WithHeight(12))
	t.SetStyles(styles)
	return dataTableModel{options: options, table: t, filter: filter, width: 80}
}
func (m dataTableModel) Init() tea.Cmd { return nil }
func (m dataTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(30, msg.Width-4)
		m.table.SetWidth(m.width)
		if m.selectable {
			specs := make([]colSpec, 0, len(m.options.Columns))
			for _, column := range m.options.Columns {
				specs = append(specs, colSpec{title: column.Title, min: min(4, column.Width), pref: column.Width})
			}
			m.table.SetColumns(fitTableColumns(m.width, specs))
			m.table.SetHeight(max(3, msg.Height-9))
		} else {
			m.table.SetHeight(max(5, msg.Height-6))
		}
	case tea.KeyMsg:
		if m.filter.Focused() {
			if msg.String() == "esc" || msg.String() == "enter" {
				m.filter.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.table.SetRows(toTableRows(filterDataTableRows(m.options.Rows, m.filter.Value())))
			m.table.SetCursor(0)
			return m, cmd
		}
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if m.selectable {
				rows := filterDataTableRows(m.options.Rows, m.filter.Value())
				cursor := m.table.Cursor()
				if cursor >= 0 && cursor < len(rows) && !rows[cursor].Disabled {
					m.selected = rows[cursor].Key
					return m, tea.Quit
				}
			}
		case "/":
			m.filter.Focus()
			return m, textinput.Blink
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}
func (m dataTableModel) View() string {
	help := "↑/↓ navegar · / filtrar · esc concluir filtro · q sair"
	if m.selectable {
		help = "↑/↓ navegar · / buscar · enter selecionar · esc cancelar"
	}
	if m.filter.Focused() {
		help = "Digite para filtrar · enter/esc concluir"
	}
	detail := ""
	if m.selectable {
		rows := filterDataTableRows(m.options.Rows, m.filter.Value())
		cursor := m.table.Cursor()
		if cursor >= 0 && cursor < len(rows) && rows[cursor].Detail != "" {
			detail = "\n" + lipgloss.NewStyle().Width(m.width).Render(safeTableText(rows[cursor].Detail))
		}
	}
	return fmt.Sprintf("%s\n%s\n\n%s%s\n%s", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Render(m.options.Title), m.filter.View(), m.table.View(), detail, lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render(help))
}

func toTableColumns(columns []DataTableColumn) []table.Column {
	out := make([]table.Column, 0, len(columns))
	for _, column := range columns {
		out = append(out, table.Column{Title: column.Title, Width: column.Width})
	}
	return out
}
func toTableRows(rows []DataTableRow) []table.Row {
	out := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		values := make(table.Row, 0, len(row.Values))
		for _, value := range row.Values {
			values = append(values, safeTableText(value))
		}
		out = append(out, values)
	}
	return out
}

func safeTableText(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, value)
}
func filterDataTableRows(rows []DataTableRow, query string) []DataTableRow {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return rows
	}
	out := make([]DataTableRow, 0, len(rows))
	for _, row := range rows {
		if strings.Contains(strings.ToLower(row.SearchText), query) {
			out = append(out, row)
		}
	}
	return out
}
