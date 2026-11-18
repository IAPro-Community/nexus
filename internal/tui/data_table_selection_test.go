package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectableDataTableFiltersAndSelectsOriginalRow(t *testing.T) {
	m := newDataTableModel(DataTableOptions{
		Columns: []DataTableColumn{{Title: "Title", Width: 20}},
		Rows: []DataTableRow{
			{Key: "first", Values: []string{"Alpha"}, SearchText: "Alpha account one"},
			{Key: "second", Values: []string{"Beta"}, SearchText: "Beta account two"},
		},
	})
	m.selectable = true
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(dataTableModel)
	for _, r := range "account two" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(dataTableModel)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(dataTableModel)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(dataTableModel)
	if cmd == nil || m.selected != "second" {
		t.Fatalf("selection = %q, expected matching original row", m.selected)
	}
}

func TestSelectableDataTableShowsUnavailableReasonAtNarrowWidth(t *testing.T) {
	m := newDataTableModel(DataTableOptions{
		Columns: []DataTableColumn{{Title: "Title", Width: 20}},
		Rows:    []DataTableRow{{Key: "blocked", Values: []string{"Thread"}, Disabled: true, Detail: "Origem ambígua ou não confirmada"}},
	})
	m.selectable = true
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 15})
	if view := updated.(dataTableModel).View(); !strings.Contains(view, "Origem ambígua ou não confirmada") {
		t.Fatalf("reason missing from selector: %s", view)
	}
}

func TestSelectableDataTableDoesNotRenderTerminalControlCharacters(t *testing.T) {
	m := newDataTableModel(DataTableOptions{
		Columns: []DataTableColumn{{Title: "Title", Width: 30}},
		Rows:    []DataTableRow{{Key: "entry", Values: []string{"hello\x1b]0;untrusted\a"}, Detail: "project\x1b[2J"}},
	})
	m.selectable = true
	view := m.View()
	if strings.Contains(view, "\x1b]0;untrusted") || strings.Contains(view, "\x1b[2J") {
		t.Fatalf("untrusted control sequence reached terminal: %q", view)
	}
}

func TestSelectableDataTableDoesNotSelectUnavailableRow(t *testing.T) {
	m := newDataTableModel(DataTableOptions{Columns: []DataTableColumn{{Title: "Title", Width: 20}}, Rows: []DataTableRow{{Key: "blocked", Values: []string{"Blocked"}, Disabled: true}}})
	m.selectable = true
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(dataTableModel)
	if cmd != nil || m.selected != "" {
		t.Fatalf("unavailable row selected: %q", m.selected)
	}
}
