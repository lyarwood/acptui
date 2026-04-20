package ambient

import (
	"fmt"
	"strings"
)

func FormatMarkdown(session Session, messages []Message) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n", session.Name)
	fmt.Fprintf(&b, "**Session:** `%s`\n", session.ID)
	fmt.Fprintf(&b, "**Project:** %s\n", session.ProjectID)
	fmt.Fprintf(&b, "**Model:** %s\n", session.LlmModel)
	fmt.Fprintf(&b, "**Phase:** %s\n", session.Phase)
	if session.RepoURL != "" {
		fmt.Fprintf(&b, "**Repo:** %s\n", session.RepoURL)
	}
	if session.CreatedAt != nil {
		fmt.Fprintf(&b, "**Created:** %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	b.WriteString("\n---\n\n")

	for _, m := range messages {
		switch {
		case m.IsReasoning:
			b.WriteString("<details>\n<summary>Thinking</summary>\n\n")
			b.WriteString(m.Content)
			b.WriteString("\n\n</details>\n\n")
		case m.Role == "user":
			fmt.Fprintf(&b, "## 🧑 User\n\n%s\n\n", m.Content)
		case m.Role == "assistant":
			fmt.Fprintf(&b, "## 🤖 Assistant\n\n%s\n\n", m.Content)
		default:
			fmt.Fprintf(&b, "## %s\n\n%s\n\n", m.Role, m.Content)
		}
	}

	return b.String()
}
