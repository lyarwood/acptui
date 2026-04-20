package tui

import (
	"fmt"
	"strings"

	"github.com/lyarwood/acptui/internal/ambient"
)

func renderFiles(entries []ambient.WorkspaceEntry, cursor int, currentPath string, fileContent string, viewingFile bool, width, height int) string {
	if viewingFile {
		return renderFileContent(fileContent, currentPath, width, height)
	}

	var b strings.Builder

	header := fmt.Sprintf(" Workspace: %s", currentPath)
	b.WriteString(detailLabelStyle.Render(header))
	b.WriteString("\n\n")

	if len(entries) == 0 {
		b.WriteString(dimStyle.Render("  (empty)"))
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
		if end > len(entries) {
			end = len(entries)
		}

		for i := scrollOffset; i < end; i++ {
			e := entries[i]
			icon := "  "
			if e.IsDir {
				icon = "📁"
			}
			line := fmt.Sprintf("  %s %s", icon, e.Name)
			if !e.IsDir {
				line += dimStyle.Render(fmt.Sprintf("  (%s)", formatSize(e.Size)))
			}

			if i == cursor {
				b.WriteString(selectedStyle.Render(fmt.Sprintf("%-*s", width-6, line)))
			} else {
				b.WriteString(detailValueStyle.Render(line))
			}
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d/%d", cursor+1, len(entries))))
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

func renderFileContent(content, path string, width, height int) string {
	var b strings.Builder

	b.WriteString(detailLabelStyle.Render(fmt.Sprintf(" File: %s", path)))
	b.WriteString("\n\n")

	lines := strings.Split(content, "\n")
	maxLines := height - 8
	if maxLines > len(lines) {
		maxLines = len(lines)
	}
	for i := 0; i < maxLines; i++ {
		line := lines[i]
		if len(line) > width-8 {
			line = line[:width-11] + "..."
		}
		b.WriteString("  " + detailValueStyle.Render(line) + "\n")
	}
	if len(lines) > maxLines {
		b.WriteString(dimStyle.Render(fmt.Sprintf("\n  ... %d more lines", len(lines)-maxLines)))
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

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1fM", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1fK", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
