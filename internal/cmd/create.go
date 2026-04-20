package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/lyarwood/acptui/internal/ambient"
)

var (
	createPrompt      string
	createDisplayName string
	createModel       string
	createRepos       []string
	createWorkflowURL string
	createWorkflowBr  string
	createWorkflowPth string
)

var createCmd = &cobra.Command{
	Use:   "create <project>",
	Short: "Create a new session in a project",
	Long: `Create a new agentic session in the specified project.

Examples:
  # Simple session with a prompt
  acptui create kubevirt --prompt "Review the auth middleware"

  # With a repo
  acptui create kubevirt --prompt "Fix bug #42" --repo https://github.com/org/repo

  # With multiple repos
  acptui create kubevirt --prompt "Compare APIs" \
    --repo https://github.com/org/frontend \
    --repo https://github.com/org/backend

  # With a custom workflow
  acptui create kubevirt --prompt "Generate eval tasks" \
    --repo https://github.com/kubevirt/kubevirt-user-guide \
    --repo https://github.com/kubevirt/kubernetes-mcp-server \
    --workflow-url https://github.com/lyarwood/kubevirt-ai-helpers \
    --workflow-branch feat/create-eval-from-docs \
    --workflow-path workflows/create-eval-from-docs`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project := args[0]

		if createPrompt == "" {
			return fmt.Errorf("--prompt is required")
		}

		cc, err := ambient.NewClientFromConfig(Version, insecureTLS)
		if err != nil {
			return fmt.Errorf("connecting to API: %w", err)
		}

		provider, err := cc.ProviderForProject(project)
		if err != nil {
			return fmt.Errorf("creating client: %w", err)
		}

		displayName := createDisplayName
		if displayName == "" {
			displayName = createPrompt
			if len(displayName) > 60 {
				displayName = displayName[:57] + "..."
			}
		}

		req := ambient.CreateSessionRequest{
			Prompt:      createPrompt,
			DisplayName: displayName,
			Model:       createModel,
		}

		for _, r := range createRepos {
			parts := strings.SplitN(r, "@", 2)
			repo := ambient.RepoInput{URL: parts[0]}
			if len(parts) > 1 {
				repo.Branch = parts[1]
			}
			req.Repos = append(req.Repos, repo)
		}

		if createWorkflowURL != "" {
			branch := createWorkflowBr
			if branch == "" {
				branch = "main"
			}
			req.Workflow = &ambient.WorkflowSelection{
				GitURL: createWorkflowURL,
				Branch: branch,
				Path:   createWorkflowPth,
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		name, err := provider.CreateSession(ctx, req)
		if err != nil {
			return fmt.Errorf("creating session: %w", err)
		}

		result := map[string]string{
			"name":    name,
			"project": project,
			"message": "Session created successfully",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	},
}

func init() {
	createCmd.Flags().StringVar(&createPrompt, "prompt", "", "initial prompt (required)")
	createCmd.Flags().StringVar(&createDisplayName, "display-name", "", "display name (defaults to prompt)")
	createCmd.Flags().StringVar(&createModel, "model", "claude-sonnet-4-6", "LLM model")
	createCmd.Flags().StringArrayVar(&createRepos, "repo", nil, "repository URL (can be repeated; use url@branch for a specific branch)")
	createCmd.Flags().StringVar(&createWorkflowURL, "workflow-url", "", "custom workflow git URL")
	createCmd.Flags().StringVar(&createWorkflowBr, "workflow-branch", "main", "custom workflow branch")
	createCmd.Flags().StringVar(&createWorkflowPth, "workflow-path", "", "custom workflow path within repo")
}
