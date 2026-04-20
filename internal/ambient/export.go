package ambient

import (
	"fmt"
	"strings"
)

func FormatMarkdown(session Session, messages []Message) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# %s\n\n", session.Name))
	b.WriteString(fmt.Sprintf("**Session:** `%s`\n", session.ID))
	b.WriteString(fmt.Sprintf("**Project:** %s\n", session.ProjectID))
	b.WriteString(fmt.Sprintf("**Model:** %s\n", session.LlmModel))
	b.WriteString(fmt.Sprintf("**Phase:** %s\n", session.Phase))
	if session.RepoURL != "" {
		b.WriteString(fmt.Sprintf("**Repo:** %s\n", session.RepoURL))
	}
	if session.CreatedAt != nil {
		b.WriteString(fmt.Sprintf("**Created:** %s\n", session.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	b.WriteString("\n---\n\n")

	for _, m := range messages {
		switch {
		case m.IsReasoning:
			b.WriteString("<details>\n<summary>Thinking</summary>\n\n")
			b.WriteString(m.Content)
			b.WriteString("\n\n</details>\n\n")
		case m.Role == "user":
			b.WriteString(fmt.Sprintf("## 🧑 User\n\n%s\n\n", m.Content))
		case m.Role == "assistant":
			b.WriteString(fmt.Sprintf("## 🤖 Assistant\n\n%s\n\n", m.Content))
		default:
			b.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", m.Role, m.Content))
		}
	}

	return b.String()
}
