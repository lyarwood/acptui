package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lyarwood/acptui/internal/ambient"
)

func renderProjectList(projects []ambient.Project, cursor int, width, height int) string {
	if len(projects) == 0 {
		msg := "No projects found.\nRun `acptui login` to authenticate."
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center,
			dimStyle.Render(msg))
	}

	colName := 30
	colDisplayName := 40
	colStatus := 12
	remaining := width - colName - colDisplayName - colStatus - 6
	if remaining > 0 {
		colName += remaining / 2
		colDisplayName += remaining - remaining/2
	}

	header := fmt.Sprintf(" %-*s %-*s %-*s",
		colName, "NAME",
		colDisplayName, "DISPLAY NAME",
		colStatus, "STATUS")
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
	if end > len(projects) {
		end = len(projects)
	}

	for i := scrollOffset; i < end; i++ {
		p := projects[i]
		name := truncateStr(p.Name, colName)
		displayName := truncateStr(p.DisplayName, colDisplayName)
		status := truncateStr(p.Status, colStatus)

		row := fmt.Sprintf(" %-*s %-*s %-*s",
			colName, name,
			colDisplayName, displayName,
			colStatus, status)

		if i == cursor {
			row = selectedStyle.Width(width).Render(row)
		} else {
			row = lipgloss.NewStyle().Width(width).Render(row)
		}
		rows = append(rows, row)
	}

	content := strings.Join(rows, "\n")

	position := fmt.Sprintf(" %d/%d", cursor+1, len(projects))
	statusBar := statusBarStyle.Width(width).Render(position)

	return lipgloss.JoinVertical(lipgloss.Left, header, content, statusBar)
}
