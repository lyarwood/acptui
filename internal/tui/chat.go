package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lyarwood/acptui/internal/ambient"
)

func renderChat(session ambient.Session, messages []ambient.Message, expandedIDs map[string]bool, scrollOffset int, atBottom bool, awaitingInput bool, width, height int, inputView string, sending bool) string {
	// Session header bar
	name := session.Name
	if name == "" {
		name = session.ID
	}
	phaseText := phaseStyle(session.Phase)
	headerLine := fmt.Sprintf(" %s  %s", name, phaseText)
	if session.AgentStatus != "" {
		headerLine += "  " + renderAgentStatus(session.AgentStatus)
	}
	if session.Phase == "Stopped" || session.Phase == "Completed" || session.Phase == "Failed" {
		headerLine += "  " + dimStyle.Render("(send a message to resume)")
	}
	sessionHeader := headerStyle.Width(width).Render(headerLine)

	// Message area (minus header, separator, input)
	messageArea := height - 5

	if len(messages) == 0 && !sending {
		empty := lipgloss.Place(width, messageArea, lipgloss.Center, lipgloss.Center,
			dimStyle.Render("No messages yet."))
		sep := dimStyle.Render(strings.Repeat("─", width))
		inp := "│ " + inputView
		return sessionHeader + "\n" + empty + "\n" + sep + "\n" + inp
	}

	var lines []string
	for _, m := range messages {
		if m.IsReasoning {
			expanded := expandedIDs != nil && expandedIDs[m.ID]
			lines = append(lines, renderThinkingBlock(m, expanded, width-4)...)
			lines = append(lines, "")
			continue
		}
		prefix := renderRole(m.Role)
		content := wordWrap(m.Content, width-4)
		lines = append(lines, prefix)
		for _, l := range strings.Split(content, "\n") {
			lines = append(lines, "  "+l)
		}
		lines = append(lines, "")
	}

	if sending {
		lines = append(lines, dimStyle.Render("  Sending..."))
		lines = append(lines, "")
	}

	maxScroll := len(lines) - messageArea
	if maxScroll < 0 {
		maxScroll = 0
	}

	if atBottom || scrollOffset > maxScroll {
		scrollOffset = maxScroll
	}
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	end := scrollOffset + messageArea
	if end > len(lines) {
		end = len(lines)
	}

	visible := lines[scrollOffset:end]
	content := strings.Join(visible, "\n")

	visibleCount := len(visible)
	if visibleCount < messageArea {
		content += strings.Repeat("\n", messageArea-visibleCount)
	}

	var separator, input string
	if awaitingInput {
		separator = phasePendingStyle.Render(strings.Repeat("─", width))
		input = phasePendingStyle.Render("▶ ") + inputView
	} else {
		separator = dimStyle.Render(strings.Repeat("─", width))
		input = "│ " + inputView
	}

	return sessionHeader + "\n" + content + "\n" + separator + "\n" + input
}

func renderAgentStatus(status string) string {
	switch status {
	case "idle":
		return dimStyle.Render("[idle]")
	case "busy", "working", "running":
		return phaseRunningStyle.Render("[" + status + "]")
	default:
		return dimStyle.Render("[" + status + "]")
	}
}

func renderThinkingBlock(m ambient.Message, expanded bool, maxWidth int) []string {
	summary := m.Content
	if len(summary) > 60 {
		summary = summary[:57] + "..."
	}
	// Replace newlines in summary
	summary = strings.ReplaceAll(summary, "\n", " ")

	if !expanded {
		return []string{dimStyle.Render(fmt.Sprintf("▸ Thinking: %s  [tab to expand]", summary))}
	}

	var lines []string
	lines = append(lines, dimStyle.Render("▾ Thinking:"))
	content := wordWrap(m.Content, maxWidth)
	for _, l := range strings.Split(content, "\n") {
		lines = append(lines, dimStyle.Render("  "+l))
	}
	return lines
}

func renderRole(role string) string {
	switch role {
	case "user":
		return phasePendingStyle.Render("▶ You")
	case "assistant":
		return phaseRunningStyle.Render("▶ Assistant")
	default:
		return dimStyle.Render(fmt.Sprintf("▶ %s", role))
	}
}

func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	var result strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if len(line) <= width {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}
		for len(line) > width {
			breakAt := strings.LastIndex(line[:width], " ")
			if breakAt <= 0 {
				breakAt = width
			}
			result.WriteString(line[:breakAt])
			result.WriteString("\n")
			line = strings.TrimLeft(line[breakAt:], " ")
		}
		if len(line) > 0 {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}
	return strings.TrimRight(result.String(), "\n")
}
