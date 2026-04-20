# CLAUDE.md

## Project overview

acptui is a Go TUI for browsing and managing Ambient Code Platform sessions. It uses Charm libraries (Bubble Tea, Lip Gloss, Bubbles) for the TUI, Cobra for CLI argument parsing, Ginkgo/Gomega for testing, and a lightweight HTTP client for API communication with the old Kubernetes CRD-based backend.

## Build and test

```sh
make build          # build to bin/acptui
make test           # run tests via ginkgo
make lint           # run golangci-lint
make test-cover     # coverage report to coverage.html
```

## Architecture

- `internal/ambient/` — data layer: direct HTTP client for the old backend API (`/api/projects/{project}/agentic-sessions`). `Provider` exposes `ListSessions`, `ListProjects`, `StartSession`, `StopSession`, `DeleteSession`, `CreateSession`, `ListModels`, `ListWorkflows`, `ListWorkspace`, `ListTasks`, `GetFileContent`, `StreamEvents`, `SendMessage`, and `ExportSession`. `ClientConfig` stores connection parameters and creates project-scoped providers on demand via `ProviderForProject(project)`. `config.go` reads config from `~/.config/ambient/config.json`. `types.go` defines thin domain types. `export.go` formats messages as markdown.
- `internal/tui/` — Bubble Tea TUI with multiple views: project picker, session list, session detail, chat (with SSE streaming and collapsible thinking blocks), create session form, workspace file browser, and task viewer. Filtering is live with regex support and prefix syntax (`phase:`, `model:`, `repo:`). Multiple space-separated terms are ANDed. Theming is in `theme.go` — 5 built-in themes: default, catppuccin, dracula, nord, light. Selected via `--theme` flag.
- `internal/cmd/` — Cobra commands: root (launches TUI with project picker or direct project launch via `acptui <project>`), `create` (scripted session creation with repos and workflows), `export` (session conversation to markdown), `login` (OpenShift OAuth browser-based login), `version`.

## Testing conventions

- Use Ginkgo v2 (`github.com/onsi/ginkgo/v2`) and Gomega for all tests.
- Tests use external test packages (`package ambient_test`, `package tui_test`) to test only public APIs.
- Each Ginkgo test package has a `*_suite_test.go` bootstrap file.

## Key design decisions

- **Authentication**: `acptui login` implements browser-based OpenShift OAuth flow. It discovers the OAuth server by following the frontend's `/oauth/start` redirect, opens `/oauth/token/request` in the browser, and prompts the user to paste the token. The token is validated against `/api/me` and saved with the detected username. This produces a real user OAuth token (not a service account token), which is required for the runner to refresh integration credentials (GitHub, Jira, etc.).
- **Auth headers**: All requests send both `Authorization: Bearer <token>` and `X-Forwarded-Access-Token: <token>` for compatibility with the OAuth-proxied frontend. `X-Forwarded-User` and `X-Forwarded-Preferred-Username` are sent when a user is configured, enabling proper identity propagation to the backend for credential refresh and message attribution.
- **No SDK dependency**: acptui talks directly to the old Kubernetes CRD-based backend via REST, not the `ambient-api-server` SDK. The old backend returns raw `AgenticSession` custom resources at `/api/projects/{project}/agentic-sessions`. This avoids the SDK's dependency on gRPC, protobuf, and the `ambient-api-server` module.
- **Project picker**: The TUI starts on the project picker, listing all projects from `/api/projects`. Selecting a project creates a project-scoped provider and loads sessions. Esc from the session list returns to the project picker.
- **Chat streaming**: The chat view uses a persistent SSE connection to the `/agui/events` endpoint. `MESSAGES_SNAPSHOT` events populate history (accumulated across runs by message ID). `TEXT_MESSAGE_START/CONTENT/END` events stream assistant responses in real-time. `REASONING_MESSAGE_START/CONTENT/END` events stream thinking blocks. The SSE connection stays open for live updates — no polling needed.
- **Thinking blocks**: Reasoning/thinking messages are displayed as collapsible blocks. Collapsed by default showing a one-line summary; `Tab` toggles the most recent block. Live reasoning streams in real-time via `REASONING_MESSAGE_*` events.
- **Frontend URL**: Defaults to `api_url` when `frontend_url` is not set (since deployed environments typically serve the frontend and API from the same host). `Enter` on a session opens the chat view; `w` opens the web UI in the browser.
- **Workspace browsing**: Directory listing uses `GET /workspace?path=/dir`, file content uses `GET /workspace/path/to/file`. Only works when the session is Running (runner pod must be alive).
- `ClientConfig.ProviderForProject(name)` creates an HTTP client scoped to the given project.
- Session list polls every 10s for updates.
- Version is injected at build time via `-ldflags`.
- Adding a new theme: add an entry to the `themes` map in `theme.go` — all styles are automatically derived.
