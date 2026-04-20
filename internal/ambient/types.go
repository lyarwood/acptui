package ambient

import (
	"regexp"
	"strings"
	"time"
)

type Session struct {
	ID                 string
	Name               string
	ProjectID          string
	Phase              string
	LlmModel           string
	RepoURL            string
	Repos              string
	Prompt             string
	WorkflowID         string
	StartTime          *time.Time
	CompletionTime     *time.Time
	CreatedAt          *time.Time
	UpdatedAt          *time.Time
	Timeout            int
	CreatedByUserID    string
	AssignedUserID     string
	AgentStatus        string
	SdkSessionID       string
	SdkRestartCount    int
	ParentSessionID    string
	Conditions         []Condition
	ReconciledRepos    []ReconciledRepo
	ReconciledWorkflow *ReconciledWorkflow
}

func (s Session) IsStartable() bool {
	switch s.Phase {
	case "Pending", "Stopped", "Failed", "Completed", "":
		return true
	default:
		return false
	}
}

func (s Session) Age() time.Time {
	if s.UpdatedAt != nil {
		return *s.UpdatedAt
	}
	if s.CreatedAt != nil {
		return *s.CreatedAt
	}
	return time.Time{}
}

type Project struct {
	Name        string
	DisplayName string
	Status      string
}

type Condition struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

type ReconciledRepo struct {
	Name   string
	URL    string
	Branch string
	Status string
}

type ReconciledWorkflow struct {
	GitURL string
	Branch string
	Status string
}

type ModelInfo struct {
	ID        string
	Label     string
	IsDefault bool
}

type WorkflowInfo struct {
	ID          string
	Name        string
	Description string
	GitURL      string
	Branch      string
	Path        string
}

type WorkflowSelection struct {
	GitURL string
	Branch string
	Path   string
}

type RepoInput struct {
	URL    string
	Branch string
}

type CreateSessionRequest struct {
	Prompt      string
	DisplayName string
	Model       string
	Repos       []RepoInput
	Workflow    *WorkflowSelection
}

type WorkspaceEntry struct {
	Name       string
	Path       string
	IsDir      bool
	Size       int64
	ModifiedAt string
}

type TaskInfo struct {
	ID           string
	Description  string
	Status       string
	Summary      string
	LastToolName string
	TokenUsage   int
	ToolUses     int
	DurationMs   int
}

type Message struct {
	ID          string
	Role        string
	Content     string
	IsReasoning bool
}

type SSEEvent struct {
	Type      string
	MessageID string
	Role      string
	Content   string
	Delta     string
	Messages  []Message
	Error     string
}

func FlexMatch(text, pattern string) bool {
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return strings.Contains(strings.ToLower(text), strings.ToLower(pattern))
	}
	return re.MatchString(text)
}
