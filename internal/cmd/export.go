package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/lyarwood/acptui/internal/ambient"
)

var exportOutput string

var exportCmd = &cobra.Command{
	Use:   "export <project> <session-id>",
	Short: "Export a session conversation to a markdown file",
	Long: `Export the full conversation history of a session to a local markdown file.

Examples:
  acptui export kubevirt session-abc123
  acptui export kubevirt session-abc123 -o review.md`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		project := args[0]
		sessionID := args[1]

		cc, err := ambient.NewClientFromConfig(Version, insecureTLS)
		if err != nil {
			return fmt.Errorf("connecting to API: %w", err)
		}

		provider, err := cc.ProviderForProject(project)
		if err != nil {
			return fmt.Errorf("creating client: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Get session metadata
		sessions, err := provider.ListSessions(ctx)
		if err != nil {
			return fmt.Errorf("listing sessions: %w", err)
		}
		var session ambient.Session
		for _, s := range sessions {
			if s.ID == sessionID {
				session = s
				break
			}
		}
		if session.ID == "" {
			session = ambient.Session{ID: sessionID, Name: sessionID, ProjectID: project}
		}

		// Export messages
		messages, err := provider.ExportSession(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("exporting session: %w", err)
		}

		md := ambient.FormatMarkdown(session, messages)

		filename := exportOutput
		if filename == "" {
			filename = sessionID + ".md"
		}

		if err := os.WriteFile(filename, []byte(md), 0644); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}

		fmt.Printf("Exported %d messages to %s\n", len(messages), filename)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file path (default: <session-id>.md)")
}
