package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/lyarwood/acptui/internal/ambient"
	"github.com/lyarwood/acptui/internal/tui"
)

var Version = "dev"

var (
	themeName   string
	insecureTLS bool
)

var rootCmd = &cobra.Command{
	Use:   "acptui",
	Short: "Ambient Code TUI Viewer - browse and manage Ambient Code Platform sessions",
	Long:  "acptui connects to the Ambient Code Platform API and displays sessions in a TUI. Select a project, then browse its sessions.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if themeName != "" {
			if !tui.SetTheme(themeName) {
				return fmt.Errorf("unknown theme %q, available: %s", themeName, strings.Join(tui.ThemeNames(), ", "))
			}
		}

		cc, err := ambient.NewClientFromConfig(Version, insecureTLS)
		if err != nil {
			return fmt.Errorf("connecting to API: %w", err)
		}

		cfg, err := ambient.LoadConfig()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Use configured project to bootstrap the provider for listing projects
		bootstrap, err := cc.ProviderForProject(fallbackProject(cfg))
		if err != nil {
			return fmt.Errorf("creating client: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		projects, err := bootstrap.ListProjects(ctx)
		if err != nil {
			return fmt.Errorf("listing projects: %w", err)
		}

		model := tui.NewModel(projects, cc)
		p := tea.NewProgram(model, tea.WithAltScreen())
		_, err = p.Run()
		return err
	},
}

func fallbackProject(cfg *ambient.Config) string {
	if p := cfg.GetProject(); p != "" {
		return p
	}
	return "_bootstrap"
}

func init() {
	rootCmd.Flags().StringVar(&themeName, "theme", "", "color theme (default, catppuccin, dracula, nord, light)")
	rootCmd.Flags().BoolVar(&insecureTLS, "insecure-tls", false, "skip TLS certificate verification")
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(versionCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
