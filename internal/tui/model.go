package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lyarwood/acptui/internal/ambient"
)

type viewState int

const (
	viewProjects viewState = iota
	viewList
	viewDetail
	viewChat
	viewCreate
	viewFiles
	viewTasks
)

type projectsLoadedMsg struct {
	projects []ambient.Project
	err      error
}

type sessionsLoadedMsg struct {
	sessions []ambient.Session
	provider *ambient.Provider
	project  string
	err      error
}

type sessionsRefreshedMsg struct {
	sessions []ambient.Session
	err      error
}

type browserOpenedMsg struct {
	err error
}

type sseConnectedMsg struct {
	ch     <-chan ambient.SSEEvent
	cancel context.CancelFunc
}

type sseEventMsg struct {
	event ambient.SSEEvent
}

type sseErrorMsg struct {
	err error
}

type messageSentMsg struct {
	err error
}

type sessionActionMsg struct {
	err error
}

type modelsLoadedMsg struct {
	models []ambient.ModelInfo
	err    error
}

type workflowsLoadedMsg struct {
	workflows []ambient.WorkflowInfo
	err       error
}

type sessionCreatedMsg struct {
	name string
	err  error
}

type filesLoadedMsg struct {
	entries []ambient.WorkspaceEntry
	err     error
}

type fileContentLoadedMsg struct {
	content string
	path    string
	err     error
}

type tasksLoadedMsg struct {
	tasks []ambient.TaskInfo
	err   error
}

type exportDoneMsg struct {
	path string
	err  error
}

type pollTickMsg struct{}

type Model struct {
	view        viewState
	projects    []ambient.Project
	projCursor  int
	projLoading bool
	sessions    []ambient.Session
	filtered    []ambient.Session
	cursor      int
	err         error
	width       int
	height      int
	filterInput textinput.Model
	filtering   bool
	filterText  string
	showHelp    bool
	cc          *ambient.ClientConfig
	provider    *ambient.Provider
	frontendURL string
	project     string

	actionInProgress string

	// Create state
	createField      createField
	createPrompt     textinput.Model
	createRepoURL    textinput.Model
	createRepoBranch textinput.Model
	createRepos      []ambient.RepoInput
	repoSuggestions  []string
	repoSugIdx       int
	createCustomURL  textinput.Model
	createCustomBr   textinput.Model
	createCustomPath textinput.Model
	models           []ambient.ModelInfo
	modelIdx         int
	workflows        []ambient.WorkflowInfo
	wfIdx            int
	wfMode           workflowMode
	creating         bool

	// Files state
	fileEntries []ambient.WorkspaceEntry
	fileCursor  int
	filePath    string
	fileContent string
	viewingFile bool

	// Tasks state
	taskList   []ambient.TaskInfo
	taskCursor int

	// Chat state
	chatSession   ambient.Session
	chatMessages  []ambient.Message
	chatInput     textinput.Model
	chatLoading   bool
	chatSending   bool
	chatScroll    int
	chatAtBottom  bool
	chatSSECh      <-chan ambient.SSEEvent
	chatSSECancel  context.CancelFunc
	chatStreaming   string // messageID of in-progress assistant message
	chatExpandedIDs map[string]bool // reasoning message IDs that are expanded
}

func newBaseModel(cc *ambient.ClientConfig) Model {
	ti := textinput.New()
	ti.Placeholder = "filter sessions..."
	ti.CharLimit = 100

	ci := textinput.New()
	ci.Placeholder = "type a message..."
	ci.CharLimit = 1000

	cp := textinput.New()
	cp.Placeholder = "describe what you want to do..."
	cp.CharLimit = 2000

	ru := textinput.New()
	ru.Placeholder = "https://github.com/org/repo"
	ru.CharLimit = 500

	rb := textinput.New()
	rb.Placeholder = "main (default)"
	rb.CharLimit = 200

	cu := textinput.New()
	cu.Placeholder = "https://github.com/org/repo.git"
	cu.CharLimit = 500

	cb := textinput.New()
	cb.Placeholder = "main"
	cb.CharLimit = 200

	cpth := textinput.New()
	cpth.Placeholder = "path/to/workflow (optional)"
	cpth.CharLimit = 500

	return Model{
		filterInput:      ti,
		chatInput:        ci,
		createPrompt:     cp,
		createRepoURL:    ru,
		createRepoBranch: rb,
		createCustomURL:  cu,
		createCustomBr:   cb,
		createCustomPath: cpth,
		frontendURL:      strings.TrimRight(cc.FrontendURL, "/"),
		cc:               cc,
	}
}

func NewModel(projects []ambient.Project, cc *ambient.ClientConfig) Model {
	m := newBaseModel(cc)
	m.view = viewProjects
	m.projects = projects
	return m
}

func NewModelWithProject(sessions []ambient.Session, provider *ambient.Provider, cc *ambient.ClientConfig, project string) Model {
	m := newBaseModel(cc)
	m.view = viewList
	m.sessions = sessions
	m.filtered = sessions
	m.provider = provider
	m.project = project
	return m
}

func (m Model) Init() tea.Cmd {
	if m.view == viewList {
		return pollTick()
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case projectsLoadedMsg:
		m.projLoading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.projects = msg.projects
			m.err = nil
		}
		return m, nil

	case sessionsLoadedMsg:
		m.projLoading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.view = viewList
			m.provider = msg.provider
			m.project = msg.project
			m.sessions = msg.sessions
			m.filtered = msg.sessions
			m.cursor = 0
			m.filterText = ""
			m.filterInput.SetValue("")
			m.err = nil
			return m, pollTick()
		}
		return m, nil

	case sessionsRefreshedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.sessions = msg.sessions
			m.applyFilter(m.filterText)
			m.err = nil
		}
		return m, nil

	case browserOpenedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m, nil

	case sseConnectedMsg:
		m.chatSSECh = msg.ch
		m.chatSSECancel = msg.cancel
		m.chatLoading = false
		return m, waitForSSE(msg.ch)

	case sseEventMsg:
		m.chatLoading = false
		switch msg.event.Type {
		case "MESSAGES_SNAPSHOT":
			for _, em := range msg.event.Messages {
				if !hasMessage(m.chatMessages, em.ID) {
					m.chatMessages = append(m.chatMessages, em)
				}
			}
		case "TEXT_MESSAGE_START":
			m.chatStreaming = msg.event.MessageID
			m.chatMessages = append(m.chatMessages, ambient.Message{
				ID:   msg.event.MessageID,
				Role: msg.event.Role,
			})
			m.chatAtBottom = true
		case "TEXT_MESSAGE_CONTENT":
			for i := range m.chatMessages {
				if m.chatMessages[i].ID == msg.event.MessageID {
					m.chatMessages[i].Content += msg.event.Delta
					break
				}
			}
		case "TEXT_MESSAGE_END":
			m.chatStreaming = ""
		case "REASONING_MESSAGE_START":
			m.chatStreaming = msg.event.MessageID
			m.chatMessages = append(m.chatMessages, ambient.Message{
				ID:          msg.event.MessageID,
				Role:        "reasoning",
				IsReasoning: true,
			})
			m.chatAtBottom = true
		case "REASONING_MESSAGE_CONTENT":
			for i := range m.chatMessages {
				if m.chatMessages[i].ID == msg.event.MessageID {
					m.chatMessages[i].Content += msg.event.Delta
					break
				}
			}
		case "REASONING_MESSAGE_END":
			m.chatStreaming = ""
		case "RUN_FINISHED":
			m.chatStreaming = ""
			m.chatSending = false
		case "RUN_ERROR":
			m.chatStreaming = ""
			m.chatSending = false
			m.err = fmt.Errorf("agent error: %s", msg.event.Error)
		}
		return m, waitForSSE(m.chatSSECh)

	case sseErrorMsg:
		m.chatLoading = false
		if msg.err != nil {
			m.err = msg.err
		}
		return m, nil

	case messageSentMsg:
		m.chatSending = false
		if msg.err != nil {
			m.err = msg.err
		}
		return m, nil

	case modelsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.models = msg.models
			for i, mo := range m.models {
				if mo.IsDefault {
					m.modelIdx = i
					break
				}
			}
		}
		return m, nil

	case workflowsLoadedMsg:
		if msg.err == nil {
			m.workflows = msg.workflows
		}
		return m, nil

	case sessionCreatedMsg:
		m.creating = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.view = viewList
			m.err = nil
			return m, m.refreshSessions()
		}
		return m, nil

	case filesLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.fileEntries = msg.entries
			m.fileCursor = 0
			m.err = nil
		}
		return m, nil

	case fileContentLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.fileContent = msg.content
			m.filePath = msg.path
			m.viewingFile = true
			m.err = nil
		}
		return m, nil

	case tasksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.taskList = msg.tasks
			m.taskCursor = 0
			m.err = nil
		}
		return m, nil

	case exportDoneMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.actionInProgress = fmt.Sprintf("Exported to %s", msg.path)
		}
		return m, nil

	case sessionActionMsg:
		m.actionInProgress = ""
		if msg.err != nil {
			m.err = msg.err
		} else {
			return m, m.refreshSessions()
		}
		return m, nil

	case pollTickMsg:
		if m.view == viewList {
			return m, tea.Batch(m.refreshSessions(), pollTick())
		}
		if m.view == viewProjects {
			return m, nil
		}
		return m, pollTick()

	case tea.KeyMsg:
		if m.filtering {
			return m.updateFiltering(msg)
		}
		switch m.view {
		case viewProjects:
			return m.updateProjects(msg)
		case viewChat:
			return m.updateChat(msg)
		case viewCreate:
			return m.updateCreate(msg)
		case viewFiles:
			return m.updateFiles(msg)
		case viewTasks:
			return m.updateTasks(msg)
		case viewDetail:
			return m.updateDetail(msg)
		default:
			return m.updateList(msg)
		}
	}

	return m, nil
}

func (m Model) updateProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case matchKey(msg, keys.Quit):
		return m, tea.Quit

	case matchKey(msg, keys.Up):
		if m.projCursor > 0 {
			m.projCursor--
		}

	case matchKey(msg, keys.Down):
		if m.projCursor < len(m.projects)-1 {
			m.projCursor++
		}

	case matchKey(msg, keys.Enter):
		if len(m.projects) > 0 && !m.projLoading {
			m.projLoading = true
			m.err = nil
			return m, m.selectProject(m.projects[m.projCursor])
		}

	case matchKey(msg, keys.Refresh):
		m.projLoading = true
		return m, m.loadProjects()

	case matchKey(msg, keys.Help):
		m.showHelp = !m.showHelp
	}

	return m, nil
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case matchKey(msg, keys.Quit):
		return m, tea.Quit

	case matchKey(msg, keys.Back):
		if len(m.projects) == 0 {
			return m, tea.Quit
		}
		m.view = viewProjects
		m.sessions = nil
		m.filtered = nil
		m.cursor = 0
		m.filterText = ""
		m.filterInput.SetValue("")
		m.err = nil

	case matchKey(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case matchKey(msg, keys.Down):
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}

	case matchKey(msg, keys.Enter):
		if len(m.filtered) > 0 {
			session := m.filtered[m.cursor]
			m.view = viewChat
			m.chatSession = session
			m.chatMessages = nil
			m.chatLoading = true
			m.chatSending = false
			m.chatScroll = -1
			m.chatAtBottom = true
			m.chatMessages = nil
			m.chatStreaming = ""
			m.chatExpandedIDs = make(map[string]bool)
			m.err = nil
			m.chatInput.SetValue("")
			m.chatInput.Focus()
			if m.chatSSECancel != nil {
				m.chatSSECancel()
			}
			return m, tea.Batch(m.connectSSE(session.ID), textinput.Blink)
		}

	case matchKey(msg, keys.WebUI):
		if len(m.filtered) > 0 {
			return m, m.openInBrowser(m.filtered[m.cursor])
		}

	case matchKey(msg, keys.Detail):
		if len(m.filtered) > 0 {
			m.view = viewDetail
			m.err = nil
		}

	case matchKey(msg, keys.Start):
		if len(m.filtered) > 0 && m.actionInProgress == "" {
			session := m.filtered[m.cursor]
			if session.IsStartable() {
				m.actionInProgress = "Starting..."
				m.err = nil
				return m, m.startSession(session.ID)
			}
		}

	case matchKey(msg, keys.Stop):
		if len(m.filtered) > 0 && m.actionInProgress == "" {
			session := m.filtered[m.cursor]
			if !session.IsStartable() {
				m.actionInProgress = "Stopping..."
				m.err = nil
				return m, m.stopSession(session.ID)
			}
		}

	case matchKey(msg, keys.New):
		m.view = viewCreate
		m.createField = fieldPrompt
		m.createPrompt.SetValue("")
		m.createRepoURL.SetValue("")
		m.createRepoBranch.SetValue("")
		m.createRepos = nil
		m.createCustomURL.SetValue("")
		m.createCustomBr.SetValue("")
		m.createCustomPath.SetValue("")
		m.wfMode = wfNone
		m.wfIdx = 0
		m.repoSuggestions = collectRepoSuggestions(m.sessions)
		m.repoSugIdx = -1
		m.creating = false
		m.err = nil
		m.createPrompt.Focus()
		var cmds []tea.Cmd
		cmds = append(cmds, textinput.Blink)
		if len(m.models) == 0 {
			cmds = append(cmds, m.loadModels())
		}
		if len(m.workflows) == 0 {
			cmds = append(cmds, m.loadWorkflows())
		}
		return m, tea.Batch(cmds...)

	case matchKey(msg, keys.Delete):
		if len(m.filtered) > 0 && m.actionInProgress == "" {
			m.actionInProgress = "Deleting..."
			m.err = nil
			return m, m.deleteSession(m.filtered[m.cursor].ID)
		}

	case matchKey(msg, keys.Filter):
		m.filtering = true
		m.filterInput.Focus()
		return m, textinput.Blink

	case matchKey(msg, keys.Refresh):
		return m, m.refreshSessions()

	case matchKey(msg, keys.Help):
		m.showHelp = !m.showHelp
	}

	return m, nil
}

func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case matchKey(msg, keys.Back), matchKey(msg, keys.Quit):
		m.view = viewList

	case matchKey(msg, keys.Enter):
		if len(m.filtered) > 0 {
			session := m.filtered[m.cursor]
			m.view = viewChat
			m.chatSession = session
			m.chatMessages = nil
			m.chatLoading = true
			m.chatSending = false
			m.chatScroll = -1
			m.chatAtBottom = true
			m.chatMessages = nil
			m.chatStreaming = ""
			m.chatExpandedIDs = make(map[string]bool)
			m.err = nil
			m.chatInput.SetValue("")
			m.chatInput.Focus()
			if m.chatSSECancel != nil {
				m.chatSSECancel()
			}
			return m, tea.Batch(m.connectSSE(session.ID), textinput.Blink)
		}

	case matchKey(msg, keys.WebUI):
		if len(m.filtered) > 0 {
			return m, m.openInBrowser(m.filtered[m.cursor])
		}
	}

	return m, nil
}

func (m Model) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.chatSSECancel != nil {
			m.chatSSECancel()
			m.chatSSECancel = nil
			m.chatSSECh = nil
		}
		m.view = viewList
		m.chatInput.Blur()
		return m, nil

	case "ctrl+c":
		return m, tea.Quit

	case "enter":
		content := strings.TrimSpace(m.chatInput.Value())
		if content != "" && !m.chatSending {
			m.chatSending = true
			m.chatAtBottom = true
			m.chatInput.SetValue("")
			m.err = nil
			return m, m.sendMessage(m.chatSession.ID, content)
		}
		return m, nil

	case "ctrl+r":
		if m.chatSSECancel != nil {
			m.chatSSECancel()
		}
		m.chatMessages = nil
		m.chatLoading = true
		return m, m.connectSSE(m.chatSession.ID)

	case "pgup":
		m.chatScroll -= 10
		if m.chatScroll < 0 {
			m.chatScroll = 0
		}
		m.chatAtBottom = false
		return m, nil

	case "pgdown":
		m.chatScroll += 10
		m.chatAtBottom = false
		return m, nil

	case "up":
		if m.chatScroll > 0 {
			m.chatScroll--
			m.chatAtBottom = false
		}
		return m, nil

	case "down":
		m.chatScroll++
		m.chatAtBottom = false
		return m, nil

	case "end":
		m.chatAtBottom = true
		return m, nil

	case "home":
		m.chatScroll = 0
		m.chatAtBottom = false
		return m, nil

	case "tab":
		m.toggleNearestReasoning()
		return m, nil

	case "ctrl+e":
		return m, m.exportSession()

	case "ctrl+f":
		m.view = viewFiles
		m.filePath = "/"
		m.viewingFile = false
		m.fileEntries = nil
		m.fileCursor = 0
		m.err = nil
		return m, m.loadFiles(m.chatSession.ID, "/")

	case "ctrl+t":
		m.view = viewTasks
		m.taskList = nil
		m.taskCursor = 0
		m.err = nil
		return m, m.loadTasks(m.chatSession.ID)
	}

	var cmd tea.Cmd
	m.chatInput, cmd = m.chatInput.Update(msg)
	return m, cmd
}

func (m Model) updateFiles(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.viewingFile {
			m.viewingFile = false
			return m, nil
		}
		m.view = viewChat
		return m, nil

	case "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if !m.viewingFile && m.fileCursor > 0 {
			m.fileCursor--
		}

	case "down", "j":
		if !m.viewingFile && m.fileCursor < len(m.fileEntries)-1 {
			m.fileCursor++
		}

	case "enter":
		if m.viewingFile || len(m.fileEntries) == 0 {
			return m, nil
		}
		entry := m.fileEntries[m.fileCursor]
		if entry.IsDir {
			m.filePath = entry.Path
			m.fileCursor = 0
			return m, m.loadFiles(m.chatSession.ID, entry.Path)
		}
		return m, m.loadFileContent(m.chatSession.ID, entry.Path)

	case "backspace":
		if m.viewingFile {
			m.viewingFile = false
			return m, nil
		}
		if m.filePath != "/" {
			parent := m.filePath
			if idx := strings.LastIndex(parent[:len(parent)-1], "/"); idx >= 0 {
				parent = parent[:idx+1]
			} else {
				parent = "/"
			}
			m.filePath = parent
			m.fileCursor = 0
			return m, m.loadFiles(m.chatSession.ID, parent)
		}
	}

	return m, nil
}

func (m Model) updateTasks(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = viewChat
		return m, nil

	case "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.taskCursor > 0 {
			m.taskCursor--
		}

	case "down", "j":
		if m.taskCursor < len(m.taskList)-1 {
			m.taskCursor++
		}

	case "r":
		return m, m.loadTasks(m.chatSession.ID)
	}

	return m, nil
}

func (m Model) updateCreate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = viewList
		m.blurAllCreateFields()
		return m, nil

	case "ctrl+c":
		return m, tea.Quit

	case "tab":
		m.createField = nextCreateField(m.createField, m.wfMode, 1)
		m.focusCreateField()
		return m, textinput.Blink

	case "shift+tab":
		m.createField = nextCreateField(m.createField, m.wfMode, -1)
		m.focusCreateField()
		return m, textinput.Blink

	case "left":
		if m.createField == fieldRepoURL && len(m.repoSuggestions) > 0 {
			m.repoSugIdx = (m.repoSugIdx + len(m.repoSuggestions)) % (len(m.repoSuggestions) + 1)
			if m.repoSugIdx == len(m.repoSuggestions) {
				m.repoSugIdx = len(m.repoSuggestions) - 1
			}
			m.createRepoURL.SetValue(m.repoSuggestions[m.repoSugIdx])
			return m, nil
		}
		if m.createField == fieldModel && len(m.models) > 0 {
			m.modelIdx = (m.modelIdx + len(m.models) - 1) % len(m.models)
			return m, nil
		}
		if m.createField == fieldWorkflow {
			m.cycleWorkflow(-1)
			return m, nil
		}

	case "right":
		if m.createField == fieldRepoURL && len(m.repoSuggestions) > 0 {
			m.repoSugIdx = (m.repoSugIdx + 1) % len(m.repoSuggestions)
			m.createRepoURL.SetValue(m.repoSuggestions[m.repoSugIdx])
			return m, nil
		}
		if m.createField == fieldModel && len(m.models) > 0 {
			m.modelIdx = (m.modelIdx + 1) % len(m.models)
			return m, nil
		}
		if m.createField == fieldWorkflow {
			m.cycleWorkflow(1)
			return m, nil
		}

	case "ctrl+a":
		if m.createField == fieldRepoURL || m.createField == fieldRepoBranch {
			url := strings.TrimSpace(m.createRepoURL.Value())
			if url != "" {
				branch := strings.TrimSpace(m.createRepoBranch.Value())
				m.createRepos = append(m.createRepos, ambient.RepoInput{URL: url, Branch: branch})
				m.createRepoURL.SetValue("")
				m.createRepoBranch.SetValue("")
			}
			return m, nil
		}

	case "ctrl+x":
		if (m.createField == fieldRepoURL || m.createField == fieldRepoBranch) && len(m.createRepos) > 0 {
			m.createRepos = m.createRepos[:len(m.createRepos)-1]
			return m, nil
		}

	case "enter":
		prompt := strings.TrimSpace(m.createPrompt.Value())
		if prompt == "" {
			m.err = fmt.Errorf("prompt is required")
			return m, nil
		}
		if m.creating {
			return m, nil
		}
		m.creating = true
		m.err = nil

		model := ""
		if len(m.models) > 0 {
			model = m.models[m.modelIdx].ID
		}

		// Include any repo URL still in the input field
		repos := m.createRepos
		if pendingURL := strings.TrimSpace(m.createRepoURL.Value()); pendingURL != "" {
			repos = append(repos, ambient.RepoInput{
				URL:    pendingURL,
				Branch: strings.TrimSpace(m.createRepoBranch.Value()),
			})
		}

		req := ambient.CreateSessionRequest{
			Prompt:      prompt,
			DisplayName: truncateForDisplay(prompt),
			Model:       model,
			Repos:       repos,
		}

		switch m.wfMode {
		case wfOOTB:
			if len(m.workflows) > 0 && m.wfIdx < len(m.workflows) {
				w := m.workflows[m.wfIdx]
				req.Workflow = &ambient.WorkflowSelection{
					GitURL: w.GitURL,
					Branch: w.Branch,
					Path:   w.Path,
				}
			}
		case wfCustom:
			gitURL := strings.TrimSpace(m.createCustomURL.Value())
			if gitURL != "" {
				branch := strings.TrimSpace(m.createCustomBr.Value())
				if branch == "" {
					branch = "main"
				}
				req.Workflow = &ambient.WorkflowSelection{
					GitURL: gitURL,
					Branch: branch,
					Path:   strings.TrimSpace(m.createCustomPath.Value()),
				}
			}
		}

		return m, m.createSession(req)
	}

	// Forward to active text input
	var cmd tea.Cmd
	switch m.createField {
	case fieldPrompt:
		m.createPrompt, cmd = m.createPrompt.Update(msg)
	case fieldRepoURL:
		m.createRepoURL, cmd = m.createRepoURL.Update(msg)
	case fieldRepoBranch:
		m.createRepoBranch, cmd = m.createRepoBranch.Update(msg)
	case fieldCustomURL:
		m.createCustomURL, cmd = m.createCustomURL.Update(msg)
	case fieldCustomBranch:
		m.createCustomBr, cmd = m.createCustomBr.Update(msg)
	case fieldCustomPath:
		m.createCustomPath, cmd = m.createCustomPath.Update(msg)
	}
	return m, cmd
}

func (m *Model) cycleWorkflow(dir int) {
	// Cycle through: None → OOTB[0] → ... → OOTB[n-1] → Custom → None
	totalOOTB := len(m.workflows)
	// total options = 1 (none) + totalOOTB + 1 (custom)
	total := totalOOTB + 2

	current := 0
	switch m.wfMode {
	case wfNone:
		current = 0
	case wfOOTB:
		current = 1 + m.wfIdx
	case wfCustom:
		current = total - 1
	}

	current = (current + dir + total) % total

	switch {
	case current == 0:
		m.wfMode = wfNone
	case current <= totalOOTB:
		m.wfMode = wfOOTB
		m.wfIdx = current - 1
	default:
		m.wfMode = wfCustom
	}
}

func (m *Model) focusCreateField() {
	m.blurAllCreateFields()
	switch m.createField {
	case fieldPrompt:
		m.createPrompt.Focus()
	case fieldRepoURL:
		m.createRepoURL.Focus()
	case fieldRepoBranch:
		m.createRepoBranch.Focus()
	case fieldCustomURL:
		m.createCustomURL.Focus()
	case fieldCustomBranch:
		m.createCustomBr.Focus()
	case fieldCustomPath:
		m.createCustomPath.Focus()
	}
}

func (m *Model) blurAllCreateFields() {
	m.createPrompt.Blur()
	m.createRepoURL.Blur()
	m.createRepoBranch.Blur()
	m.createCustomURL.Blur()
	m.createCustomBr.Blur()
	m.createCustomPath.Blur()
}

func collectRepoSuggestions(sessions []ambient.Session) []string {
	seen := make(map[string]bool)
	var suggestions []string
	for _, s := range sessions {
		if s.RepoURL != "" && !seen[s.RepoURL] {
			seen[s.RepoURL] = true
			suggestions = append(suggestions, s.RepoURL)
		}
	}
	return suggestions
}

func truncateForDisplay(s string) string {
	if len(s) > 60 {
		return s[:57] + "..."
	}
	return s
}

func (m Model) updateFiltering(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filtering = false
		m.filterInput.Blur()
		m.filterText = m.filterInput.Value()
		return m, nil

	case "esc":
		m.filtering = false
		m.filterInput.Blur()
		m.filterInput.SetValue(m.filterText)
		m.applyFilter(m.filterText)
		return m, nil

	case "tab":
		m.cycleFilterPrefix()
		m.applyFilter(m.filterInput.Value())
		return m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.applyFilter(m.filterInput.Value())
	return m, cmd
}

func (m *Model) cycleFilterPrefix() {
	val := m.filterInput.Value()
	currentPrefix := ""
	rest := val
	for _, p := range filterPrefixes[1:] {
		if strings.HasPrefix(val, p) {
			currentPrefix = p
			rest = val[len(p):]
			break
		}
	}
	nextIdx := 0
	for i, p := range filterPrefixes {
		if p == currentPrefix {
			nextIdx = (i + 1) % len(filterPrefixes)
			break
		}
	}
	m.filterInput.SetValue(filterPrefixes[nextIdx] + rest)
}

func (m *Model) applyFilter(text string) {
	m.filterText = text
	if text == "" {
		m.filtered = m.sessions
	} else {
		m.filtered = MatchSessions(m.sessions, text)
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m Model) sessionURL(session ambient.Session) string {
	return fmt.Sprintf("%s/projects/%s/sessions/%s", m.frontendURL, m.project, session.ID)
}

func (m Model) openInBrowser(session ambient.Session) tea.Cmd {
	url := m.sessionURL(session)
	return func() tea.Msg {
		err := openBrowser(url)
		return browserOpenedMsg{err: err}
	}
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
}

func (m Model) refreshSessions() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		sessions, err := m.provider.ListSessions(ctx)
		return sessionsRefreshedMsg{sessions: sessions, err: err}
	}
}

func (m Model) startSession(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := m.provider.StartSession(ctx, id)
		return sessionActionMsg{err: err}
	}
}

func (m Model) stopSession(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := m.provider.StopSession(ctx, id)
		return sessionActionMsg{err: err}
	}
}

func (m Model) deleteSession(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := m.provider.DeleteSession(ctx, id)
		return sessionActionMsg{err: err}
	}
}

func (m Model) loadFiles(sessionID, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		entries, err := m.provider.ListWorkspace(ctx, sessionID, path)
		return filesLoadedMsg{entries: entries, err: err}
	}
}

func (m Model) loadFileContent(sessionID, path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		content, err := m.provider.GetFileContent(ctx, sessionID, path)
		return fileContentLoadedMsg{content: content, path: path, err: err}
	}
}

func (m Model) loadTasks(sessionID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tasks, err := m.provider.ListTasks(ctx, sessionID)
		return tasksLoadedMsg{tasks: tasks, err: err}
	}
}

func (m *Model) toggleNearestReasoning() {
	if m.chatExpandedIDs == nil {
		m.chatExpandedIDs = make(map[string]bool)
	}
	// Find the last reasoning message (closest to the bottom/most recent)
	for i := len(m.chatMessages) - 1; i >= 0; i-- {
		if m.chatMessages[i].IsReasoning {
			id := m.chatMessages[i].ID
			m.chatExpandedIDs[id] = !m.chatExpandedIDs[id]
			return
		}
	}
}

func (m Model) selectProject(project ambient.Project) tea.Cmd {
	return func() tea.Msg {
		provider, err := m.cc.ProviderForProject(project.Name)
		if err != nil {
			return sessionsLoadedMsg{err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		sessions, err := provider.ListSessions(ctx)
		if err != nil {
			return sessionsLoadedMsg{err: err}
		}
		return sessionsLoadedMsg{sessions: sessions, provider: provider, project: project.Name}
	}
}

func (m Model) loadProjects() tea.Cmd {
	return func() tea.Msg {
		provider, err := m.cc.ProviderForProject("_bootstrap")
		if err != nil {
			return projectsLoadedMsg{err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		projects, err := provider.ListProjects(ctx)
		return projectsLoadedMsg{projects: projects, err: err}
	}
}

func (m Model) exportSession() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		messages, err := m.provider.ExportSession(ctx, m.chatSession.ID)
		if err != nil {
			return exportDoneMsg{err: err}
		}
		md := ambient.FormatMarkdown(m.chatSession, messages)
		filename := m.chatSession.ID + ".md"
		if err := os.WriteFile(filename, []byte(md), 0644); err != nil {
			return exportDoneMsg{err: fmt.Errorf("write file: %w", err)}
		}
		return exportDoneMsg{path: filename}
	}
}

func (m Model) loadModels() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		models, err := m.provider.ListModels(ctx)
		return modelsLoadedMsg{models: models, err: err}
	}
}

func (m Model) loadWorkflows() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		workflows, err := m.provider.ListWorkflows(ctx)
		return workflowsLoadedMsg{workflows: workflows, err: err}
	}
}

func (m Model) createSession(req ambient.CreateSessionRequest) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		name, err := m.provider.CreateSession(ctx, req)
		return sessionCreatedMsg{name: name, err: err}
	}
}

func (m Model) connectSSE(sessionID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := m.provider.StreamEvents(ctx, sessionID)
		if err != nil {
			cancel()
			return sseErrorMsg{err: err}
		}
		return sseConnectedMsg{ch: ch, cancel: cancel}
	}
}

func waitForSSE(ch <-chan ambient.SSEEvent) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		event, ok := <-ch
		if !ok {
			return sseErrorMsg{err: nil}
		}
		return sseEventMsg{event: event}
	}
}

func hasMessage(msgs []ambient.Message, id string) bool {
	for _, m := range msgs {
		if m.ID == id {
			return true
		}
	}
	return false
}

func (m Model) sendMessage(sessionID, content string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := m.provider.SendMessage(ctx, sessionID, content)
		return messageSentMsg{err: err}
	}
}

func pollTick() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return pollTickMsg{}
	})
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := titleStyle.Render("acptui - Ambient Code Platform TUI")

	var content string
	switch m.view {
	case viewProjects:
		content = renderProjectList(m.projects, m.projCursor, m.width, m.height-3)
		if m.projLoading {
			content += "\n" + dimStyle.Render("Loading...")
		}
	case viewList:
		content = renderSessionList(m.filtered, m.cursor, m.width, m.height-3, m.filterText)
	case viewDetail:
		if len(m.filtered) > 0 {
			content = renderDetail(m.filtered[m.cursor], m.width, m.height-3)
		}
	case viewChat:
		if m.chatLoading {
			content = dimStyle.Render("Loading messages...")
		} else {
			content = renderChat(m.chatSession, m.chatMessages, m.chatExpandedIDs, m.chatScroll, m.chatAtBottom, m.width, m.height-3, m.chatInput.View(), m.chatSending)
		}
	case viewFiles:
		content = renderFiles(m.fileEntries, m.fileCursor, m.filePath, m.fileContent, m.viewingFile, m.width, m.height-3)
	case viewTasks:
		content = renderTasks(m.taskList, m.taskCursor, m.width, m.height-3)
	case viewCreate:
		content = renderCreate(
			m.createField, m.createPrompt.View(),
			m.createRepoURL.View(), m.createRepoBranch.View(), m.createRepos,
			len(m.repoSuggestions),
			m.models, m.modelIdx,
			m.workflows, m.wfIdx, m.wfMode,
			m.createCustomURL.View(), m.createCustomBr.View(), m.createCustomPath.View(),
			m.creating, m.width, m.height-3,
		)
	}

	if m.err != nil {
		content += "\n" + errorStyle.Render("Error: "+m.err.Error())
	}

	footer := m.renderFooter()
	return title + "\n" + content + "\n" + footer
}

func (m Model) renderFooter() string {
	switch m.view {
	case viewProjects:
		if m.showHelp {
			return helpStyle.Render("enter:select project  r:refresh  ?:help  q:quit")
		}
		return helpStyle.Render("enter:select  ?:help  q:quit")
	case viewFiles:
		if m.viewingFile {
			return helpStyle.Render("esc:back to listing")
		}
		return helpStyle.Render("enter:open  backspace:parent dir  esc:back to chat")
	case viewTasks:
		return helpStyle.Render("r:refresh  esc:back to chat")
	case viewCreate:
		return helpStyle.Render("tab:next field  left/right:cycle model/workflow  ctrl+a:add repo  ctrl+x:remove repo  enter:create  esc:cancel")
	case viewChat:
		name := m.chatSession.Name
		if name == "" {
			name = m.chatSession.ID
		}
		return helpStyle.Render(fmt.Sprintf("[%s] enter:send  tab:thinking  ctrl+e:export  ctrl+f:files  ctrl+t:tasks  ctrl+r:refresh  esc:back", name))
	default:
		if m.actionInProgress != "" {
			return helpStyle.Render(fmt.Sprintf("[%s] %s", m.project, m.actionInProgress))
		}
		if m.filtering {
			return "Filter: " + m.filterInput.View() + "  " + helpStyle.Render("tab:cycle prefix  enter:apply  esc:cancel")
		}
		if m.filterText != "" {
			return helpStyle.Render(fmt.Sprintf("[%s] active filter: %s  /:edit  ?:help  q:quit", m.project, m.filterText))
		}
		if m.showHelp {
			return helpStyle.Render(fmt.Sprintf("[%s] enter:chat  n:new  w:browser  d:detail  s:start  x:stop  ctrl+d:delete  /:filter  r:refresh  ?:help  q:quit", m.project))
		}
		return helpStyle.Render(fmt.Sprintf("[%s] ?:help  q:quit", m.project))
	}
}

func matchKey(msg tea.KeyMsg, binding key.Binding) bool {
	for _, k := range binding.Keys() {
		if msg.String() == k {
			return true
		}
	}
	return false
}
