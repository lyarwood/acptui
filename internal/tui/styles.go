package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle          lipgloss.Style
	statusBarStyle      lipgloss.Style
	selectedStyle       lipgloss.Style
	dimStyle            lipgloss.Style
	detailBorderStyle   lipgloss.Style
	detailLabelStyle    lipgloss.Style
	detailValueStyle    lipgloss.Style
	headerStyle         lipgloss.Style
	errorStyle          lipgloss.Style
	helpStyle           lipgloss.Style
	phasePendingStyle   lipgloss.Style
	phaseRunningStyle   lipgloss.Style
	phaseCompletedStyle lipgloss.Style
	phaseFailedStyle    lipgloss.Style
	phaseStoppedStyle   lipgloss.Style
)

func init() {
	applyTheme()
}

func applyTheme() {
	t := activeTheme

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Accent).
		Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
		Foreground(t.Dim).
		Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.SelectedFg).
		Background(t.SelectedBg)

	dimStyle = lipgloss.NewStyle().
		Foreground(t.Dim)

	detailBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1, 2)

	detailLabelStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Accent).
		Width(18)

	detailValueStyle = lipgloss.NewStyle().
		Foreground(t.Text)

	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Text).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(t.Dim)

	errorStyle = lipgloss.NewStyle().
		Foreground(t.Error).
		Bold(true)

	helpStyle = lipgloss.NewStyle().
		Foreground(t.Dim)

	phasePendingStyle = lipgloss.NewStyle().
		Foreground(t.PhasePending).
		Bold(true)

	phaseRunningStyle = lipgloss.NewStyle().
		Foreground(t.PhaseRunning).
		Bold(true)

	phaseCompletedStyle = lipgloss.NewStyle().
		Foreground(t.PhaseCompleted).
		Bold(true)

	phaseFailedStyle = lipgloss.NewStyle().
		Foreground(t.PhaseFailed).
		Bold(true)

	phaseStoppedStyle = lipgloss.NewStyle().
		Foreground(t.PhaseStopped)
}
