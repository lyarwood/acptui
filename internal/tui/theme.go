package tui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name           string
	Accent         lipgloss.Color
	Text           lipgloss.Color
	TextBright     lipgloss.Color
	Dim            lipgloss.Color
	Border         lipgloss.Color
	Running        lipgloss.Color
	Error          lipgloss.Color
	SelectedFg     lipgloss.Color
	SelectedBg     lipgloss.Color
	PhasePending   lipgloss.Color
	PhaseRunning   lipgloss.Color
	PhaseCompleted lipgloss.Color
	PhaseFailed    lipgloss.Color
	PhaseStopped   lipgloss.Color
}

var themes = map[string]Theme{
	"default": {
		Name:           "default",
		Accent:         lipgloss.Color("205"),
		Text:           lipgloss.Color("252"),
		TextBright:     lipgloss.Color("255"),
		Dim:            lipgloss.Color("240"),
		Border:         lipgloss.Color("62"),
		Running:        lipgloss.Color("42"),
		Error:          lipgloss.Color("196"),
		SelectedFg:     lipgloss.Color("255"),
		SelectedBg:     lipgloss.Color("57"),
		PhasePending:   lipgloss.Color("220"),
		PhaseRunning:   lipgloss.Color("42"),
		PhaseCompleted: lipgloss.Color("44"),
		PhaseFailed:    lipgloss.Color("196"),
		PhaseStopped:   lipgloss.Color("240"),
	},
	"catppuccin": {
		Name:           "catppuccin",
		Accent:         lipgloss.Color("#cba6f7"),
		Text:           lipgloss.Color("#cdd6f4"),
		TextBright:     lipgloss.Color("#f5f5f5"),
		Dim:            lipgloss.Color("#6c7086"),
		Border:         lipgloss.Color("#89b4fa"),
		Running:        lipgloss.Color("#a6e3a1"),
		Error:          lipgloss.Color("#f38ba8"),
		SelectedFg:     lipgloss.Color("#1e1e2e"),
		SelectedBg:     lipgloss.Color("#cba6f7"),
		PhasePending:   lipgloss.Color("#f9e2af"),
		PhaseRunning:   lipgloss.Color("#a6e3a1"),
		PhaseCompleted: lipgloss.Color("#89dceb"),
		PhaseFailed:    lipgloss.Color("#f38ba8"),
		PhaseStopped:   lipgloss.Color("#6c7086"),
	},
	"dracula": {
		Name:           "dracula",
		Accent:         lipgloss.Color("#ff79c6"),
		Text:           lipgloss.Color("#f8f8f2"),
		TextBright:     lipgloss.Color("#ffffff"),
		Dim:            lipgloss.Color("#6272a4"),
		Border:         lipgloss.Color("#bd93f9"),
		Running:        lipgloss.Color("#50fa7b"),
		Error:          lipgloss.Color("#ff5555"),
		SelectedFg:     lipgloss.Color("#282a36"),
		SelectedBg:     lipgloss.Color("#bd93f9"),
		PhasePending:   lipgloss.Color("#f1fa8c"),
		PhaseRunning:   lipgloss.Color("#50fa7b"),
		PhaseCompleted: lipgloss.Color("#8be9fd"),
		PhaseFailed:    lipgloss.Color("#ff5555"),
		PhaseStopped:   lipgloss.Color("#6272a4"),
	},
	"nord": {
		Name:           "nord",
		Accent:         lipgloss.Color("#88c0d0"),
		Text:           lipgloss.Color("#d8dee9"),
		TextBright:     lipgloss.Color("#eceff4"),
		Dim:            lipgloss.Color("#4c566a"),
		Border:         lipgloss.Color("#5e81ac"),
		Running:        lipgloss.Color("#a3be8c"),
		Error:          lipgloss.Color("#bf616a"),
		SelectedFg:     lipgloss.Color("#2e3440"),
		SelectedBg:     lipgloss.Color("#88c0d0"),
		PhasePending:   lipgloss.Color("#ebcb8b"),
		PhaseRunning:   lipgloss.Color("#a3be8c"),
		PhaseCompleted: lipgloss.Color("#88c0d0"),
		PhaseFailed:    lipgloss.Color("#bf616a"),
		PhaseStopped:   lipgloss.Color("#4c566a"),
	},
	"light": {
		Name:           "light",
		Accent:         lipgloss.Color("90"),
		Text:           lipgloss.Color("235"),
		TextBright:     lipgloss.Color("232"),
		Dim:            lipgloss.Color("245"),
		Border:         lipgloss.Color("63"),
		Running:        lipgloss.Color("28"),
		Error:          lipgloss.Color("124"),
		SelectedFg:     lipgloss.Color("255"),
		SelectedBg:     lipgloss.Color("90"),
		PhasePending:   lipgloss.Color("136"),
		PhaseRunning:   lipgloss.Color("28"),
		PhaseCompleted: lipgloss.Color("30"),
		PhaseFailed:    lipgloss.Color("124"),
		PhaseStopped:   lipgloss.Color("245"),
	},
}

var activeTheme = themes["default"]

func SetTheme(name string) bool {
	t, ok := themes[name]
	if !ok {
		return false
	}
	activeTheme = t
	applyTheme()
	return ok
}

func ThemeNames() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	return names
}
