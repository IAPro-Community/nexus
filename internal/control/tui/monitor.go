package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MonitorModel is a read-only sidecar. It never sends Attach, Input, Resize,
// lease, or stop commands; every refresh uses short-lived status/events RPCs.
type MonitorModel struct {
	runtimeID string
	status    protocol.StatusData
	usage     protocol.UsageData
	events    []protocol.EventData
	errorText string
	updatedAt time.Time
	quitting  bool
}

func NewMonitorModel(runtimeID string) MonitorModel {
	return MonitorModel{runtimeID: strings.TrimSpace(runtimeID)}
}

type monitorTick time.Time
type monitorData struct {
	status protocol.StatusData
	usage  protocol.UsageData
	events []protocol.EventData
	err    error
}

func (m MonitorModel) Init() tea.Cmd {
	return func() tea.Msg { return loadMonitorData(m.runtimeID) }
}

func loadMonitorData(runtimeID string) tea.Msg {
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		return monitorData{err: err}
	}
	defer client.Close()
	status, err := client.Status()
	if err != nil {
		return monitorData{err: err}
	}
	eventsList, eventsErr := client.Events(20)
	if eventsErr != nil {
		return monitorData{status: status, err: eventsErr}
	}
	usage, usageErr := client.Usage()
	if usageErr != nil {
		return monitorData{status: status, events: eventsList, err: usageErr}
	}
	return monitorData{status: status, usage: usage, events: eventsList}
}

func (m MonitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case monitorData:
		if msg.err != nil {
			m.errorText = msg.err.Error()
		} else {
			m.status = msg.status
			m.usage = msg.usage
			m.events = msg.events
			m.errorText = ""
		}
		m.updatedAt = time.Now()
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return monitorTick(t) })
	case monitorTick:
		return m, func() tea.Msg { return loadMonitorData(m.runtimeID) }
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "esc" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m MonitorModel) View() string {
	if m.quitting {
		return ""
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#8B5CF6")).Render("NEXUS CONTROL MONITOR")
	if m.runtimeID == "" {
		return title + "\n\nNo runtime selected.\n[q] Quit\n"
	}
	status := m.status
	quotaText := firstNonEmpty(m.usage.Status, status.QuotaStatus, "unknown")
	if quotaText != "unknown" && quotaText != "UNKNOWN" && m.usage.FetchedAtUnix > 0 {
		quotaText = fmt.Sprintf("%s (%.0f%% left)", quotaText, m.usage.PercentLeft)
	}
	lines := []string{
		title,
		fmt.Sprintf("Runtime: %s", m.runtimeID),
		fmt.Sprintf("Provider/Profile: %s:%s", status.ProviderID, status.ProfileID),
		fmt.Sprintf("Startup: %s", status.StartupStage),
		fmt.Sprintf("State: %s    PID: %d", status.State, status.PID),
		fmt.Sprintf("Attention: %s", firstNonEmpty(status.AttentionKind, status.AttentionReason, "none")),
		fmt.Sprintf("Attention context: %s", firstNonEmpty(status.AttentionContext, "none")),
		fmt.Sprintf("Continuity: %s", firstNonEmpty(status.Continuity, "unknown")),
		"Quota: " + quotaText,
	}
	if status.LastFault != "" {
		lines = append(lines, "Last error: "+status.LastFault)
	}
	if m.errorText != "" {
		lines = append(lines, "Transport: "+m.errorText)
	}
	lines = append(lines, "", "Recent events:")
	for _, event := range m.events {
		lines = append(lines, fmt.Sprintf("  [%s] %-18s %s", event.Timestamp.Format("15:04:05"), event.Type, event.Summary))
	}
	if len(m.events) == 0 {
		lines = append(lines, "  (none)")
	}
	lines = append(lines, "", "Read-only sidecar · no PTY attach or writer lease · [q] Quit")
	return strings.Join(lines, "\n") + "\n"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// RunControlMonitor launches the read-only sidecar in a separate terminal.
func RunControlMonitor(ctx context.Context, runtimeID string) error {
	if strings.TrimSpace(runtimeID) == "" {
		for _, session := range registry.DefaultRegistry().ListActive() {
			runtimeID = session.RuntimeID
			break
		}
	}
	model := NewMonitorModel(runtimeID)
	program := tea.NewProgram(model, tea.WithAltScreen())
	done := make(chan struct{})
	if ctx != nil {
		go func() {
			select {
			case <-ctx.Done():
				program.Quit()
			case <-done:
			}
		}()
	}
	_, err := program.Run()
	close(done)
	return err
}
