package tui

import (
	"strings"

	"github.com/lyarwood/acptui/internal/ambient"
)

var filterPrefixes = []string{"", "project:", "phase:", "model:", "repo:"}

func MatchSessions(sessions []ambient.Session, text string) []ambient.Session {
	var filtered []ambient.Session
	for _, s := range sessions {
		if MatchSession(s, text) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func MatchSession(s ambient.Session, text string) bool {
	for _, term := range strings.Fields(text) {
		if !matchTerm(s, term) {
			return false
		}
	}
	return true
}

func matchTerm(s ambient.Session, term string) bool {
	prefix, query, hasPrefix := strings.Cut(term, ":")
	if hasPrefix {
		switch strings.ToLower(prefix) {
		case "project":
			return flexMatch(s.ProjectID, query)
		case "phase":
			return flexMatch(s.Phase, query)
		case "model":
			return flexMatch(s.LlmModel, query)
		case "repo":
			return flexMatch(s.RepoURL, query)
		}
	}

	return flexMatch(s.Name, term) ||
		flexMatch(s.Prompt, term) ||
		flexMatch(s.ProjectID, term) ||
		flexMatch(s.LlmModel, term) ||
		flexMatch(s.RepoURL, term)
}

func flexMatch(text, pattern string) bool {
	return ambient.FlexMatch(text, pattern)
}
