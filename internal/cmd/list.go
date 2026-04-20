package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/lyarwood/acptui/internal/ambient"
)

var (
	listJSON  bool
	listPhase string
	listModel string
	listRepo  string
	listLimit int
)

var listCmd = &cobra.Command{
	Use:   "list [project]",
	Short: "List projects, or sessions within a project",
	Long: `Without arguments, lists all projects you have access to.
With a project name argument, lists sessions in that project.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cc, err := ambient.NewClientFromConfig(Version, insecureTLS)
		if err != nil {
			return fmt.Errorf("connecting to API: %w", err)
		}

		cfg, err := ambient.LoadConfig()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// No args = list projects, with arg = list sessions for that project
		project := ""
		if len(args) > 0 {
			project = args[0]
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if project == "" {
			return listProjects(ctx, cc, cfg)
		}
		return listSessions(ctx, cc, project)
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
	listCmd.Flags().StringVar(&listPhase, "phase", "", "filter sessions by phase")
	listCmd.Flags().StringVar(&listModel, "model", "", "filter sessions by model")
	listCmd.Flags().StringVar(&listRepo, "repo", "", "filter sessions by repo URL")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "maximum number of results")
}

func listProjects(ctx context.Context, cc *ambient.ClientConfig, cfg *ambient.Config) error {
	provider, err := cc.ProviderForProject(fallbackProject(cfg))
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	projects, err := provider.ListProjects(ctx)
	if err != nil {
		return fmt.Errorf("listing projects: %w", err)
	}

	if listLimit > 0 && len(projects) > listLimit {
		projects = projects[:listLimit]
	}

	if listJSON {
		return printProjectsJSON(projects)
	}
	return printProjectsTable(projects)
}

func listSessions(ctx context.Context, cc *ambient.ClientConfig, project string) error {
	provider, err := cc.ProviderForProject(project)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}

	sessions, err := provider.ListSessions(ctx)
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}

	sessions = filterSessions(sessions)

	if listLimit > 0 && len(sessions) > listLimit {
		sessions = sessions[:listLimit]
	}

	if listJSON {
		return printSessionsJSON(sessions)
	}
	return printSessionsTable(sessions)
}

func filterSessions(sessions []ambient.Session) []ambient.Session {
	if listPhase == "" && listModel == "" && listRepo == "" {
		return sessions
	}
	var filtered []ambient.Session
	for _, s := range sessions {
		if listPhase != "" && !flexMatch(s.Phase, listPhase) {
			continue
		}
		if listModel != "" && !flexMatch(s.LlmModel, listModel) {
			continue
		}
		if listRepo != "" && !flexMatch(s.RepoURL, listRepo) {
			continue
		}
		filtered = append(filtered, s)
	}
	return filtered
}

func flexMatch(text, pattern string) bool {
	return ambient.FlexMatch(text, pattern)
}

func printProjectsJSON(projects []ambient.Project) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(projects)
}

func printProjectsTable(projects []ambient.Project) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tDISPLAY NAME\tSTATUS")

	for _, p := range projects {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.DisplayName, p.Status)
	}
	return w.Flush()
}

func printSessionsJSON(sessions []ambient.Session) error {
	type jsonSession struct {
		ID             string  `json:"id"`
		Name           string  `json:"name"`
		ProjectID      string  `json:"projectId"`
		Phase          string  `json:"phase"`
		AgentStatus    string  `json:"agentStatus,omitempty"`
		Model          string  `json:"model,omitempty"`
		RepoURL        string  `json:"repoUrl,omitempty"`
		Prompt         string  `json:"prompt,omitempty"`
		WorkflowID     string  `json:"workflowId,omitempty"`
		StartTime      *string `json:"startTime,omitempty"`
		CompletionTime *string `json:"completionTime,omitempty"`
		CreatedAt      *string `json:"createdAt,omitempty"`
		UpdatedAt      *string `json:"updatedAt,omitempty"`
	}

	out := make([]jsonSession, len(sessions))
	for i, s := range sessions {
		out[i] = jsonSession{
			ID:          s.ID,
			Name:        s.Name,
			ProjectID:   s.ProjectID,
			Phase:       s.Phase,
			AgentStatus: s.AgentStatus,
			Model:       s.LlmModel,
			RepoURL:     s.RepoURL,
			Prompt:      s.Prompt,
			WorkflowID:  s.WorkflowID,
		}
		if s.StartTime != nil {
			t := s.StartTime.Format(time.RFC3339)
			out[i].StartTime = &t
		}
		if s.CompletionTime != nil {
			t := s.CompletionTime.Format(time.RFC3339)
			out[i].CompletionTime = &t
		}
		if s.CreatedAt != nil {
			t := s.CreatedAt.Format(time.RFC3339)
			out[i].CreatedAt = &t
		}
		if s.UpdatedAt != nil {
			t := s.UpdatedAt.Format(time.RFC3339)
			out[i].UpdatedAt = &t
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func printSessionsTable(sessions []ambient.Session) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PHASE\tAGENT\tNAME\tMODEL\tREPO\tAGE")

	for _, s := range sessions {
		phase := phaseChar(s.Phase)
		agent := s.AgentStatus
		age := relativeTime(s.Age())
		repo := formatRepoShort(s.RepoURL)

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			phase, agent, s.Name, s.LlmModel, repo, age)
	}
	return w.Flush()
}

func phaseChar(phase string) string {
	switch phase {
	case "Running":
		return "*"
	case "Pending", "Creating":
		return "~"
	case "Completed":
		return "✓"
	case "Failed":
		return "!"
	case "Stopped", "Stopping":
		return "-"
	default:
		return "?"
	}
}

func formatRepoShort(url string) string {
	if len(url) > 40 {
		return url[:37] + "..."
	}
	return url
}

func relativeTime(t time.Time) string {
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
