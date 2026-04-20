package tui

import (
	"fmt"
	"strings"

	"github.com/lyarwood/acptui/internal/ambient"
)

func renderTasks(tasks []ambient.TaskInfo, cursor int, width, height int) string {
	var b strings.Builder

	b.WriteString(detailLabelStyle.Render(" Tasks"))
	b.WriteString("\n\n")

	if len(tasks) == 0 {
		b.WriteString(dimStyle.Render("  No tasks."))
	} else {
		availableHeight := height - 8
		if availableHeight < 1 {
			availableHeight = 1
		}

		scrollOffset := 0
		if cursor >= availableHeight {
			scrollOffset = cursor - availableHeight + 1
		}

		end := scrollOffset + availableHeight
		if end > len(tasks) {
			end = len(tasks)
		}

		for i := scrollOffset; i < end; i++ {
			t := tasks[i]
			status := renderTaskStatus(t.Status)
			summary := t.Summary
			if summary == "" {
				summary = t.Description
			}
			if len(summary) > width-20 {
				summary = summary[:width-23] + "..."
			}

			line := fmt.Sprintf("  %s  %s", status, summary)
			detail := dimStyle.Render(fmt.Sprintf("      %s  tokens:%d  tools:%d  %dms",
				t.ID[:min(12, len(t.ID))], t.TokenUsage, t.ToolUses, t.DurationMs))

			if i == cursor {
				b.WriteString(selectedStyle.Render(fmt.Sprintf("%-*s", width-6, line)))
				b.WriteString("\n")
				b.WriteString(detail)
			} else {
				b.WriteString(detailValueStyle.Render(line))
				b.WriteString("\n")
				b.WriteString(detail)
			}
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d/%d", cursor+1, len(tasks))))
	}

	innerWidth := width - 6
	if innerWidth < 40 {
		innerWidth = 40
	}
	innerHeight := height - 4
	if innerHeight < 10 {
		innerHeight = 10
	}

	return detailBorderStyle.
		Width(innerWidth).
		Height(innerHeight).
		Render(b.String())
}

func renderTaskStatus(status string) string {
	switch status {
	case "completed":
		return phaseCompletedStyle.Render("✓")
	case "running":
		return phaseRunningStyle.Render("*")
	case "failed", "error":
		return phaseFailedStyle.Render("!")
	default:
		return dimStyle.Render("?")
	}
}
