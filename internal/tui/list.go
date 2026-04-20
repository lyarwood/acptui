package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/lyarwood/acptui/internal/ambient"
)

func renderSessionList(sessions []ambient.Session, cursor int, width, height int, filter string) string {
	if len(sessions) == 0 {
		msg := "No sessions found.\nConnect to the Ambient Code Platform with `acpctl login` first."
		if filter != "" {
			msg = fmt.Sprintf("No sessions matching %q", filter)
		}
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center,
			dimStyle.Render(msg))
	}

	colPhase := 10
	colAgent := 8
	colModel := 20
	colRepo := 25
	colAge := 12
	colName := width - colPhase - colAgent - colModel - colRepo - colAge - 7
	if colName < 20 {
		colName = 20
	}

	header := fmt.Sprintf(" %-*s %-*s %-*s %-*s %-*s %-*s",
		colPhase, "PHASE",
		colAgent, "AGENT",
		colName, "NAME",
		colModel, "MODEL",
		colRepo, "REPO",
		colAge, "AGE")
	header = headerStyle.Width(width).Render(header)

	availableHeight := height - 4
	if availableHeight < 1 {
		availableHeight = 1
	}

	scrollOffset := 0
	if cursor >= availableHeight {
		scrollOffset = cursor - availableHeight + 1
	}

	var rows []string
	end := scrollOffset + availableHeight
	if end > len(sessions) {
		end = len(sessions)
	}

	for i := scrollOffset; i < end; i++ {
		s := sessions[i]
		row := formatRow(s, colPhase, colAgent, colName, colModel, colRepo, colAge)
		if i == cursor {
			row = selectedStyle.Width(width).Render(row)
		} else {
			row = lipgloss.NewStyle().Width(width).Render(row)
		}
		rows = append(rows, row)
	}

	content := strings.Join(rows, "\n")

	position := fmt.Sprintf(" %d/%d", cursor+1, len(sessions))
	statusBar := statusBarStyle.Width(width).Render(position)

	return lipgloss.JoinVertical(lipgloss.Left, header, content, statusBar)
}

func formatRow(s ambient.Session, colPhase, colAgent, colName, colModel, colRepo, colAge int) string {
	phase := padAndStyle(s.Phase, colPhase, phaseStyle)
	agent := padAndStyle(s.AgentStatus, colAgent, agentStyle)
	name := truncateStr(s.Name, colName)
	model := truncateStr(s.LlmModel, colModel)
	repo := truncateStr(formatRepoURL(s.RepoURL), colRepo)
	age := formatRelativeTime(s.Age())

	return fmt.Sprintf(" %s %s %-*s %-*s %-*s %-*s",
		phase,
		agent,
		colName, name,
		colModel, model,
		colRepo, repo,
		colAge, age)
}

func padAndStyle(text string, width int, styleFn func(string) string) string {
	text = truncateStr(text, width)
	padded := fmt.Sprintf("%-*s", width, text)
	return styleFn(padded)
}

func phaseStyle(text string) string {
	trimmed := strings.TrimSpace(text)
	switch trimmed {
	case "Running":
		return phaseRunningStyle.Render(text)
	case "Pending", "Creating":
		return phasePendingStyle.Render(text)
	case "Completed":
		return phaseCompletedStyle.Render(text)
	case "Failed":
		return phaseFailedStyle.Render(text)
	case "Stopped", "Stopping":
		return phaseStoppedStyle.Render(text)
	default:
		return dimStyle.Render(text)
	}
}

func agentStyle(text string) string {
	trimmed := strings.TrimSpace(text)
	switch trimmed {
	case "busy", "working", "running":
		return phaseRunningStyle.Render(text)
	default:
		return dimStyle.Render(text)
	}
}

func formatRepoURL(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, ".git")
	return url
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}
