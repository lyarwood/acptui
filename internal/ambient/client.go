package ambient

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Provider struct {
	httpClient *http.Client
	baseURL    string
	token      string
	user       string
	project    string
}

func NewProvider(baseURL, token, user, project string, insecureTLS bool) *Provider {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecureTLS {
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true} //nolint:gosec
	}
	return &Provider{
		httpClient: &http.Client{Timeout: 30 * time.Second, Transport: transport},
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		user:       user,
		project:    project,
	}
}

func (p *Provider) do(ctx context.Context, method, path string) ([]byte, error) {
	url := p.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	p.setAuthHeaders(req)
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateBody(body))
	}

	return body, nil
}

func (p *Provider) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("X-Forwarded-Access-Token", p.token)
	if p.user != "" {
		req.Header.Set("X-Forwarded-User", p.user)
		req.Header.Set("X-Forwarded-Preferred-Username", p.user)
	}
}

func truncateBody(b []byte) string {
	s := string(b)
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

// Old backend (Kubernetes CRD) response types

type k8sSessionList struct {
	Items []k8sSession `json:"items"`
}

type k8sSession struct {
	Metadata k8sMetadata    `json:"metadata"`
	Spec     k8sSessionSpec `json:"spec"`
	Status   k8sStatus      `json:"status"`
}

type k8sMetadata struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	CreationTimestamp string `json:"creationTimestamp"`
}

type k8sSessionSpec struct {
	DisplayName    string         `json:"displayName"`
	InitialPrompt  string         `json:"initialPrompt"`
	LLMSettings    k8sLLMSettings `json:"llmSettings"`
	Repos          []k8sRepo      `json:"repos"`
	Timeout        int            `json:"timeout"`
	UserContext    k8sUserContext `json:"userContext"`
	ActiveWorkflow *k8sWorkflow   `json:"activeWorkflow,omitempty"`
}

type k8sLLMSettings struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

type k8sRepo struct {
	URL      string `json:"url"`
	Branch   string `json:"branch"`
	AutoPush bool   `json:"autoPush"`
}

type k8sUserContext struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
}

type k8sWorkflow struct {
	GitURL string `json:"gitUrl"`
	Branch string `json:"branch"`
}

type k8sStatus struct {
	Phase              string         `json:"phase"`
	AgentStatus        string         `json:"agentStatus"`
	StartTime          string         `json:"startTime"`
	CompletionTime     string         `json:"completionTime"`
	Conditions         []k8sCondition `json:"conditions"`
	ReconciledRepos    []k8sReconRepo `json:"reconciledRepos"`
	ReconciledWorkflow *k8sReconWF    `json:"reconciledWorkflow,omitempty"`
}

type k8sCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type k8sReconRepo struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Branch string `json:"branch"`
	Status string `json:"status"`
}

type k8sReconWF struct {
	GitURL string `json:"gitUrl"`
	Branch string `json:"branch"`
	Status string `json:"status"`
}

func sessionFromK8s(s k8sSession) Session {
	session := Session{
		ID:              s.Metadata.Name,
		Name:            s.Spec.DisplayName,
		ProjectID:       s.Metadata.Namespace,
		Phase:           s.Status.Phase,
		AgentStatus:     s.Status.AgentStatus,
		LlmModel:        s.Spec.LLMSettings.Model,
		Prompt:          s.Spec.InitialPrompt,
		Timeout:         s.Spec.Timeout,
		CreatedByUserID: s.Spec.UserContext.UserID,
	}

	if session.Name == "" {
		session.Name = s.Metadata.Name
	}

	if len(s.Spec.Repos) > 0 {
		session.RepoURL = s.Spec.Repos[0].URL
	}

	if s.Spec.ActiveWorkflow != nil {
		session.WorkflowID = s.Spec.ActiveWorkflow.GitURL
	}

	if t, err := time.Parse(time.RFC3339, s.Metadata.CreationTimestamp); err == nil {
		session.CreatedAt = &t
	}
	if t, err := time.Parse(time.RFC3339, s.Status.StartTime); err == nil {
		session.StartTime = &t
	}
	if t, err := time.Parse(time.RFC3339, s.Status.CompletionTime); err == nil {
		session.CompletionTime = &t
	}

	for _, c := range s.Status.Conditions {
		session.Conditions = append(session.Conditions, Condition(c))
	}

	for _, r := range s.Status.ReconciledRepos {
		session.ReconciledRepos = append(session.ReconciledRepos, ReconciledRepo(r))
	}

	if s.Status.ReconciledWorkflow != nil {
		session.ReconciledWorkflow = &ReconciledWorkflow{
			GitURL: s.Status.ReconciledWorkflow.GitURL,
			Branch: s.Status.ReconciledWorkflow.Branch,
			Status: s.Status.ReconciledWorkflow.Status,
		}
	}

	age := session.CreatedAt
	if session.CompletionTime != nil {
		age = session.CompletionTime
	} else if session.StartTime != nil {
		age = session.StartTime
	}
	session.UpdatedAt = age

	return session
}

func (p *Provider) ListSessions(ctx context.Context) ([]Session, error) {
	body, err := p.do(ctx, http.MethodGet, "/api/projects/"+p.project+"/agentic-sessions")
	if err != nil {
		return nil, err
	}

	var list k8sSessionList
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("parse sessions: %w", err)
	}

	sessions := make([]Session, len(list.Items))
	for i, s := range list.Items {
		sessions[i] = sessionFromK8s(s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Age().After(sessions[j].Age())
	})

	return sessions, nil
}

func (p *Provider) ListProjects(ctx context.Context) ([]Project, error) {
	body, err := p.do(ctx, http.MethodGet, "/api/projects")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Items []struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			Status      string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse projects: %w", err)
	}

	projects := make([]Project, len(resp.Items))
	for i, p := range resp.Items {
		projects[i] = Project{
			Name:        p.Name,
			DisplayName: p.DisplayName,
			Status:      p.Status,
		}
	}
	return projects, nil
}

func (p *Provider) StartSession(ctx context.Context, name string) error {
	return p.doAction(ctx, "/api/projects/"+p.project+"/agentic-sessions/"+name+"/start")
}

func (p *Provider) StopSession(ctx context.Context, name string) error {
	return p.doAction(ctx, "/api/projects/"+p.project+"/agentic-sessions/"+name+"/stop")
}

func (p *Provider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	body, err := p.do(ctx, http.MethodGet, "/api/projects/"+p.project+"/models")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Models []struct {
			ID        string `json:"id"`
			Label     string `json:"label"`
			IsDefault bool   `json:"isDefault"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse models: %w", err)
	}

	models := make([]ModelInfo, len(resp.Models))
	for i, m := range resp.Models {
		models[i] = ModelInfo{ID: m.ID, Label: m.Label, IsDefault: m.IsDefault}
	}
	return models, nil
}

func (p *Provider) ListWorkflows(ctx context.Context) ([]WorkflowInfo, error) {
	body, err := p.do(ctx, http.MethodGet, "/api/workflows/ootb")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Workflows []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			GitURL      string `json:"gitUrl"`
			Branch      string `json:"branch"`
			Path        string `json:"path"`
			Enabled     bool   `json:"enabled"`
		} `json:"workflows"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse workflows: %w", err)
	}

	var workflows []WorkflowInfo
	for _, w := range resp.Workflows {
		if !w.Enabled {
			continue
		}
		workflows = append(workflows, WorkflowInfo{
			ID:          w.ID,
			Name:        w.Name,
			Description: w.Description,
			GitURL:      w.GitURL,
			Branch:      w.Branch,
			Path:        w.Path,
		})
	}
	return workflows, nil
}

func (p *Provider) CreateSession(ctx context.Context, req CreateSessionRequest) (string, error) {
	payload := map[string]any{
		"initialPrompt": req.Prompt,
		"displayName":   req.DisplayName,
		"llmSettings": map[string]any{
			"model":       req.Model,
			"temperature": 0.7,
		},
	}
	if len(req.Repos) > 0 {
		var repos []map[string]any
		for _, r := range req.Repos {
			repo := map[string]any{"url": r.URL}
			if r.Branch != "" {
				repo["branch"] = r.Branch
			}
			repos = append(repos, repo)
		}
		payload["repos"] = repos
	}
	if req.Workflow != nil {
		wf := map[string]any{"gitUrl": req.Workflow.GitURL}
		if req.Workflow.Branch != "" {
			wf["branch"] = req.Workflow.Branch
		}
		if req.Workflow.Path != "" {
			wf["path"] = req.Workflow.Path
		}
		payload["activeWorkflow"] = wf
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := p.baseURL + "/api/projects/" + p.project + "/agentic-sessions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	p.setAuthHeaders(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateBody(respBody))
	}

	var result struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	return result.Name, nil
}

func (p *Provider) DeleteSession(ctx context.Context, name string) error {
	url := p.baseURL + "/api/projects/" + p.project + "/agentic-sessions/" + name

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	p.setAuthHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (p *Provider) doAction(ctx context.Context, path string) error {
	url := p.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	p.setAuthHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (p *Provider) ListWorkspace(ctx context.Context, sessionName, path string) ([]WorkspaceEntry, error) {
	apiPath := "/api/projects/" + p.project + "/agentic-sessions/" + sessionName + "/workspace"
	if path != "" && path != "/" {
		apiPath += "?path=" + path
	}
	body, err := p.do(ctx, http.MethodGet, apiPath)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Items []struct {
			Name       string `json:"name"`
			Path       string `json:"path"`
			IsDir      bool   `json:"isDir"`
			Size       int64  `json:"size"`
			ModifiedAt string `json:"modifiedAt"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse workspace: %w", err)
	}

	entries := make([]WorkspaceEntry, len(resp.Items))
	for i, e := range resp.Items {
		entries[i] = WorkspaceEntry{
			Name: e.Name, Path: e.Path, IsDir: e.IsDir,
			Size: e.Size, ModifiedAt: e.ModifiedAt,
		}
	}
	return entries, nil
}

func (p *Provider) GetFileContent(ctx context.Context, sessionName, path string) (string, error) {
	body, err := p.do(ctx, http.MethodGet, "/api/projects/"+p.project+"/agentic-sessions/"+sessionName+"/workspace"+path)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (p *Provider) ListTasks(ctx context.Context, sessionName string) ([]TaskInfo, error) {
	body, err := p.do(ctx, http.MethodGet, "/api/projects/"+p.project+"/agentic-sessions/"+sessionName+"/agui/tasks")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Tasks []struct {
			ID           string `json:"task_id"`
			Description  string `json:"description"`
			Status       string `json:"status"`
			Summary      string `json:"summary"`
			LastToolName string `json:"last_tool_name"`
			Usage        struct {
				TotalTokens int `json:"total_tokens"`
				ToolUses    int `json:"tool_uses"`
				DurationMs  int `json:"duration_ms"`
			} `json:"usage"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse tasks: %w", err)
	}

	tasks := make([]TaskInfo, len(resp.Tasks))
	for i, t := range resp.Tasks {
		tasks[i] = TaskInfo{
			ID: t.ID, Description: t.Description, Status: t.Status,
			Summary: t.Summary, LastToolName: t.LastToolName,
			TokenUsage: t.Usage.TotalTokens, ToolUses: t.Usage.ToolUses,
			DurationMs: t.Usage.DurationMs,
		}
	}
	return tasks, nil
}

// GetMessages connects to the AG-UI events SSE stream and extracts messages.
// The server replays all persisted events (including MESSAGES_SNAPSHOT), then
// keeps the connection open for live events. We read the replay burst and
// return as soon as it ends (detected by a short read timeout between events).
// StreamEvents connects to the AG-UI SSE endpoint and returns a channel of
// parsed events. The connection stays open for live streaming. Cancel the
// context to close it.
func (p *Provider) StreamEvents(ctx context.Context, sessionName string) (<-chan SSEEvent, error) {
	url := p.baseURL + "/api/projects/" + p.project + "/agentic-sessions/" + sessionName + "/agui/events"

	sseClient := *p.httpClient
	sseClient.Timeout = 0

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	p.setAuthHeaders(req)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := sseClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close() //nolint:errcheck
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateBody(body))
	}

	events := make(chan SSEEvent, 50)

	go func() {
		defer resp.Body.Close() //nolint:errcheck
		defer close(events)

		seenIDs := make(map[string]bool)
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := line[6:]

			evt := parseSSEData(data, seenIDs)
			if evt != nil {
				select {
				case events <- *evt:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return events, nil
}

func parseSSEData(data string, seenIDs map[string]bool) *SSEEvent {
	var raw struct {
		Type      string       `json:"type"`
		MessageID string       `json:"messageId"`
		Role      string       `json:"role"`
		Content   string       `json:"content"`
		Delta     string       `json:"delta"`
		Message   string       `json:"message"`
		Messages  []sseMessage `json:"messages,omitempty"`
	}
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return nil
	}

	switch raw.Type {
	case "MESSAGES_SNAPSHOT":
		var msgs []Message
		for _, m := range raw.Messages {
			if m.ID != "" && seenIDs[m.ID] {
				continue
			}
			if m.Metadata != nil && m.Metadata.Hidden {
				if m.ID != "" {
					seenIDs[m.ID] = true
				}
				continue
			}
			if m.Role == "tool" {
				if m.ID != "" {
					seenIDs[m.ID] = true
				}
				continue
			}
			if m.Role == "assistant" && strings.TrimSpace(m.Content) == "" {
				if m.ID != "" {
					seenIDs[m.ID] = true
				}
				continue
			}
			if m.ID != "" {
				seenIDs[m.ID] = true
			}
			msgs = append(msgs, Message{
				ID:          m.ID,
				Role:        m.Role,
				Content:     m.Content,
				IsReasoning: m.Role == "reasoning",
			})
		}
		if len(msgs) > 0 {
			return &SSEEvent{Type: "MESSAGES_SNAPSHOT", Messages: msgs}
		}

	case "TEXT_MESSAGE_START":
		return &SSEEvent{Type: raw.Type, MessageID: raw.MessageID, Role: raw.Role}

	case "TEXT_MESSAGE_CONTENT":
		return &SSEEvent{Type: raw.Type, MessageID: raw.MessageID, Delta: raw.Delta}

	case "TEXT_MESSAGE_END":
		return &SSEEvent{Type: raw.Type, MessageID: raw.MessageID}

	case "REASONING_MESSAGE_START":
		return &SSEEvent{Type: raw.Type, MessageID: raw.MessageID, Role: "reasoning"}

	case "REASONING_MESSAGE_CONTENT":
		return &SSEEvent{Type: raw.Type, MessageID: raw.MessageID, Delta: raw.Delta}

	case "REASONING_MESSAGE_END":
		return &SSEEvent{Type: raw.Type, MessageID: raw.MessageID}

	case "RUN_STARTED":
		return &SSEEvent{Type: raw.Type}

	case "RUN_FINISHED":
		return &SSEEvent{Type: raw.Type}

	case "RUN_ERROR":
		return &SSEEvent{Type: raw.Type, Error: raw.Message}
	}

	return nil
}

type sseMessage struct {
	ID       string `json:"id"`
	Role     string `json:"role"`
	Content  string `json:"content"`
	Metadata *struct {
		Hidden bool `json:"hidden"`
	} `json:"metadata,omitempty"`
}

// SendMessage sends a user message to a session via the AG-UI run endpoint.
func (p *Provider) SendMessage(ctx context.Context, sessionName, content string) error {
	url := p.baseURL + "/api/projects/" + p.project + "/agentic-sessions/" + sessionName + "/agui/run"

	payload := map[string]interface{}{
		"threadId": sessionName,
		"messages": []map[string]interface{}{
			{
				"id":      fmt.Sprintf("acptui-%d", time.Now().UnixNano()),
				"role":    "user",
				"content": content,
			},
		},
		"tools": []interface{}{},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	p.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateBody(respBody))
	}

	return nil
}
