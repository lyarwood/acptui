package tui

import (
	"fmt"
	"strings"

	"github.com/lyarwood/acptui/internal/ambient"
)

func renderDetail(session ambient.Session, width, height int) string {
	var b strings.Builder

	row := func(label, value string) {
		b.WriteString(detailLabelStyle.Render(label))
		b.WriteString(detailValueStyle.Render(value))
		b.WriteString("\n")
	}

	row("Session ID", session.ID)
	row("Name", session.Name)
	b.WriteString("\n")

	row("Project", session.ProjectID)
	row("Phase", renderPhase(session.Phase))
	row("Model", session.LlmModel)
	b.WriteString("\n")

	if session.RepoURL != "" {
		row("Repo URL", session.RepoURL)
	}
	if session.WorkflowID != "" {
		row("Workflow", session.WorkflowID)
	}
	b.WriteString("\n")

	if session.Prompt != "" {
		prompt := session.Prompt
		if len(prompt) > 200 {
			prompt = prompt[:197] + "..."
		}
		row("Prompt", prompt)
		b.WriteString("\n")
	}

	if session.StartTime != nil {
		row("Start Time", session.StartTime.Format("2006-01-02 15:04:05"))
	}
	if session.CompletionTime != nil {
		row("Completion", session.CompletionTime.Format("2006-01-02 15:04:05"))
	}
	if session.CreatedAt != nil {
		row("Created", session.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	if session.UpdatedAt != nil {
		row("Updated", session.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
	if session.Timeout > 0 {
		row("Timeout", fmt.Sprintf("%ds", session.Timeout))
	}
	b.WriteString("\n")

	if session.CreatedByUserID != "" {
		row("Created By", session.CreatedByUserID)
	}

	// Reconciled Repos
	if len(session.ReconciledRepos) > 0 {
		b.WriteString("\n")
		b.WriteString(detailLabelStyle.Render("Reconciled Repos"))
		b.WriteString("\n")
		for _, r := range session.ReconciledRepos {
			status := renderConditionStatus(r.Status)
			line := fmt.Sprintf("  %s  %s", status, r.URL)
			if r.Branch != "" {
				line += " @ " + r.Branch
			}
			if r.Name != "" {
				line += dimStyle.Render(fmt.Sprintf(" (%s)", r.Name))
			}
			b.WriteString(detailValueStyle.Render(line))
			b.WriteString("\n")
		}
	}

	// Reconciled Workflow
	if session.ReconciledWorkflow != nil {
		b.WriteString("\n")
		status := renderConditionStatus(session.ReconciledWorkflow.Status)
		row("Reconciled WF", fmt.Sprintf("%s  %s @ %s",
			status,
			session.ReconciledWorkflow.GitURL,
			session.ReconciledWorkflow.Branch))
	}

	// Conditions
	if len(session.Conditions) > 0 {
		b.WriteString("\n")
		b.WriteString(detailLabelStyle.Render("Conditions"))
		b.WriteString("\n")
		for _, c := range session.Conditions {
			status := renderConditionStatus(c.Status)
			line := fmt.Sprintf("  %s  %-25s %s", status, c.Type, dimStyle.Render(c.Message))
			b.WriteString(detailValueStyle.Render(line))
			b.WriteString("\n")
		}
	}

	innerWidth := width - 6
	if innerWidth < 40 {
		innerWidth = 40
	}
	innerHeight := height - 4
	if innerHeight < 10 {
		innerHeight = 10
	}

	return detailBorderStyle.
		Width(innerWidth).
		Height(innerHeight).
		Render(b.String())
}

func renderPhase(phase string) string {
	switch phase {
	case "Running":
		return phaseRunningStyle.Render(phase)
	case "Pending", "Creating":
		return phasePendingStyle.Render(phase)
	case "Completed":
		return phaseCompletedStyle.Render(phase)
	case "Failed":
		return phaseFailedStyle.Render(phase)
	case "Stopped", "Stopping":
		return phaseStoppedStyle.Render(phase)
	default:
		return phase
	}
}

func renderConditionStatus(status string) string {
	switch status {
	case "True", "Ready", "Active":
		return phaseRunningStyle.Render("✓")
	case "False":
		return phaseFailedStyle.Render("✗")
	default:
		return dimStyle.Render("?")
	}
}
