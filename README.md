# acptui - Ambient Code Platform TUI

A terminal UI for browsing and managing [Ambient Code Platform](../platform/) sessions.

## Features

- **Project picker** — lists all projects you have access to; select one to browse its sessions
- **Session list** — phase, agent status, name, model, repo, age with live filtering
- **Chat view** — SSE-streamed conversation with real-time assistant responses and collapsible thinking blocks
- **Session management** — create, start, stop, delete sessions from the TUI
- **Create form** — prompt, repos (with suggestions from existing sessions), model selector, workflow picker (OOTB + custom)
- **Detail view** — full session info with reconciliation conditions, repos, and workflow status
- **File browser** — browse the session workspace (directories and file contents)
- **Task viewer** — background task list with status, token usage, and tool metrics
- **Web UI integration** — open any session in the browser with `w`
- **CLI mode** — non-interactive `list` subcommand with `--json` output
- **5 color themes** — `default`, `catppuccin`, `dracula`, `nord`, `light`

## Installation

```sh
make install
```

Or build locally:

```sh
make build
./bin/acptui
```

## Authentication

### Log in via OpenShift OAuth (recommended)

```sh
acptui login https://ambient-code.apps.example.com
```

This opens your browser to the OpenShift login page. After authenticating, you'll be shown a token — paste it back into the terminal. acptui validates the token, detects your username, and saves everything to `~/.config/ambient/config.json`.

The OAuth token is a real user token, which allows the platform to refresh integration credentials (GitHub, GitLab, Jira, Google) on your behalf during agent runs.

### Alternative: log in with acpctl

You can also authenticate using [`acpctl`](../platform/components/ambient-cli/):

```sh
acpctl login --token <your-token> --url https://api.example.com --project my-project
```

Note: if you use a service account token with acpctl, integration credential refresh (GitHub, Jira, etc.) may not work during agent runs. Use `acptui login` for full OAuth support.

### Configuration file

Credentials are stored in `~/.config/ambient/config.json`:

```json
{
  "api_url": "https://ambient-code.apps.example.com",
  "access_token": "<your-token>",
  "user": "your-username"
}
```

### Environment variables

All config values can be overridden with environment variables. These take precedence over the config file.

| Variable | Description | Default |
|---|---|---|
| `AMBIENT_TOKEN` | Bearer token for API authentication | (from config file) |
| `AMBIENT_USER` | Username for identity propagation | (from config file) |
| `AMBIENT_API_URL` | API server URL | `http://localhost:8000` |
| `AMBIENT_FRONTEND_URL` | Frontend web UI URL (for browser open) | (falls back to `api_url`) |
| `AMBIENT_CONFIG` | Path to config file | `~/.config/ambient/config.json` |

## Usage

### TUI (default)

```sh
acptui                        # launches project picker
acptui --theme catppuccin     # use a different color theme
acptui --insecure-tls         # skip TLS certificate verification
```

#### Key bindings

**Project picker:**

| Key | Action |
|---|---|
| `enter` | Select project and load its sessions |
| `r` | Refresh project list |
| `?` | Toggle help |
| `q` / `ctrl+c` | Quit |

**Session list:**

| Key | Action |
|---|---|
| `enter` | Open chat view for selected session |
| `n` | Create a new session |
| `w` | Open session in web browser |
| `d` / `space` | View session details (conditions, repos, workflow) |
| `s` | Start/resume a stopped session |
| `x` | Stop a running session |
| `ctrl+d` | Delete session |
| `/` | Open filter input |
| `tab` | Cycle filter prefix (`phase:`, `model:`, `repo:`) |
| `r` | Refresh session list |
| `?` | Toggle help |
| `esc` | Back to project picker |
| `q` / `ctrl+c` | Quit |

**Chat view:**

| Key | Action |
|---|---|
| `enter` | Send message |
| `tab` | Toggle thinking block (expand/collapse) |
| `up` / `down` | Scroll one line |
| `pgup` / `pgdn` | Scroll 10 lines |
| `home` / `end` | Jump to top / bottom |
| `ctrl+f` | Browse workspace files |
| `ctrl+t` | View background tasks |
| `ctrl+r` | Reconnect SSE stream (refresh) |
| `esc` | Back to session list |

**Create session form:**

| Key | Action |
|---|---|
| `tab` / `shift+tab` | Navigate between fields |
| `left` / `right` | Cycle model, workflow, or repo suggestions |
| `ctrl+a` | Add repo from URL/branch inputs |
| `ctrl+x` | Remove last added repo |
| `enter` | Create session |
| `esc` | Cancel |

**File browser (from chat `ctrl+f`):**

| Key | Action |
|---|---|
| `enter` | Open directory or view file |
| `backspace` | Go to parent directory |
| `up` / `down` / `j` / `k` | Navigate |
| `esc` | Back to chat |

**Task viewer (from chat `ctrl+t`):**

| Key | Action |
|---|---|
| `up` / `down` / `j` / `k` | Navigate |
| `r` | Refresh task list |
| `esc` | Back to chat |

#### Filtering

Type bare text to search across all fields, or use a prefix to target a specific field. All filter values support regex:

```
phase:Running                # match session phase
model:sonnet                 # match model name
repo:github.com/org/repo     # match repository URL
phase:^(Running|Creating)$   # regex: running or creating sessions
```

Multiple terms are ANDed together:

```
phase:Running model:sonnet
```

Invalid regex patterns fall back to substring matching.

#### Themes

```sh
acptui --theme catppuccin    # pastel purple/green
acptui --theme dracula       # pink/purple from Dracula
acptui --theme nord          # cool blue/green from Nord
acptui --theme light         # high contrast for light terminals
acptui --theme default       # the default theme
```

### CLI

```sh
acptui list                              # list all projects
acptui list kubevirt                     # list sessions in a project
acptui list kubevirt --json              # JSON output
acptui list kubevirt --phase Running     # filter by phase
acptui list kubevirt --model sonnet      # filter by model
acptui list kubevirt --repo github.com   # filter by repo
acptui list --limit 10                   # limit results
acptui list --json                       # projects as JSON
acptui login https://ambient-code.apps.example.com   # authenticate
acptui version                           # print version
```

## Session phases

Sessions progress through a lifecycle represented by phase indicators in the list view:

| Phase | Description |
|---|---|
| Running (green) | Session is actively executing |
| Pending / Creating (yellow) | Session is waiting or being provisioned |
| Completed (cyan) | Session finished successfully |
| Failed (red) | Session encountered an error |
| Stopped (dim) | Session was stopped by a user |

## Development

### Prerequisites

- Go 1.24+
- [Ginkgo](https://onsi.github.io/ginkgo/) test framework
- [golangci-lint](https://golangci-lint.run/)

### Makefile targets

| Target | Description |
|---|---|
| `make build` | Build binary to `bin/acptui` |
| `make test` | Run all tests via Ginkgo |
| `make test-verbose` | Run tests with verbose output |
| `make test-cover` | Run tests with coverage report (`coverage.html`) |
| `make lint` | Run golangci-lint |
| `make fmt` | Format code |
| `make vet` | Run go vet |
| `make clean` | Remove build artifacts |
| `make install` | Build and install to `$GOPATH/bin` |
| `make run` | Build and run |

### Project structure

```
cmd/acptui/main.go                  # Entrypoint
internal/
  ambient/                        # Data layer (HTTP client)
    types.go                      # Domain types (Session, Message, SSEEvent, etc.)
    client.go                     # Provider: API calls, SSE streaming
    config.go                     # Config loading, ClientConfig, SaveConfig
  cmd/                            # Cobra CLI commands
    root.go                       # Root command (launches TUI with project picker)
    list.go                       # Non-interactive listing (projects or sessions)
    login.go                      # OpenShift OAuth login flow
    version.go                    # Version output
  tui/                            # Bubble Tea TUI
    model.go                      # Main model (7 view states, SSE streaming)
    keys.go                       # Key bindings
    theme.go                      # 5 built-in color themes
    styles.go                     # Lip Gloss styles (derived from active theme)
    projects.go                   # Project picker view
    list.go                       # Session list view
    detail.go                     # Session detail view (conditions, repos, workflow)
    chat.go                       # Chat view (streaming responses, thinking blocks)
    create.go                     # Session creation form (prompt, repos, model, workflow)
    files.go                      # Workspace file browser
    tasks.go                      # Background task viewer
    filter.go                     # Session filter logic (prefix matching, regex)
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components (text input)
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Ginkgo](https://github.com/onsi/ginkgo) / [Gomega](https://github.com/onsi/gomega) - Testing
