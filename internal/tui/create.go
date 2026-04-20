package tui

import (
	"fmt"
	"strings"

	"github.com/lyarwood/acptui/internal/ambient"
)

type createField int

const (
	fieldPrompt createField = iota
	fieldRepoURL
	fieldRepoBranch
	fieldModel
	fieldWorkflow
	fieldCustomURL
	fieldCustomBranch
	fieldCustomPath
)

func nextCreateField(current createField, wfMode workflowMode, dir int) createField {
	order := []createField{fieldPrompt, fieldRepoURL, fieldRepoBranch, fieldModel, fieldWorkflow}
	if wfMode == wfCustom {
		order = append(order, fieldCustomURL, fieldCustomBranch, fieldCustomPath)
	}
	for i, f := range order {
		if f == current {
			next := (i + dir + len(order)) % len(order)
			return order[next]
		}
	}
	return fieldPrompt
}

type workflowMode int

const (
	wfNone workflowMode = iota
	wfOOTB
	wfCustom
)

func renderCreate(
	field createField,
	promptView string,
	repoURLView, repoBranchView string,
	repos []ambient.RepoInput,
	repoSuggestions int,
	models []ambient.ModelInfo, modelIdx int,
	workflows []ambient.WorkflowInfo, wfIdx int, wfMode workflowMode,
	customURLView, customBranchView, customPathView string,
	creating bool,
	width, height int,
) string {
	var b strings.Builder

	title := detailLabelStyle.Render("New Session")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Prompt
	writeFieldLabel(&b, "Prompt", field == fieldPrompt)
	b.WriteString(promptView)
	b.WriteString("\n\n")

	// Repos section
	writeFieldLabel(&b, "Repo URL", field == fieldRepoURL)
	b.WriteString(repoURLView)
	if field == fieldRepoURL && repoSuggestions > 0 {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("left/right:suggestions"))
	}
	b.WriteString("\n")
	writeFieldLabel(&b, "Repo Branch", field == fieldRepoBranch)
	b.WriteString(repoBranchView)
	if field == fieldRepoURL || field == fieldRepoBranch {
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("ctrl+a:add"))
	}
	b.WriteString("\n")

	if len(repos) > 0 {
		for i, r := range repos {
			entry := fmt.Sprintf("  %d. %s", i+1, r.URL)
			if r.Branch != "" {
				entry += " @ " + r.Branch
			}
			b.WriteString(detailValueStyle.Render(entry))
			b.WriteString("\n")
		}
		if field == fieldRepoURL || field == fieldRepoBranch {
			b.WriteString(dimStyle.Render("              ctrl+x:remove last"))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	// Model
	writeFieldLabel(&b, "Model", field == fieldModel)
	if len(models) > 0 {
		m := models[modelIdx]
		text := m.Label
		if m.IsDefault {
			text += " (default)"
		}
		if field == fieldModel {
			b.WriteString(fmt.Sprintf("< %s >", text))
		} else {
			b.WriteString(detailValueStyle.Render(text))
		}
	} else {
		b.WriteString(dimStyle.Render("loading..."))
	}
	b.WriteString("\n\n")

	// Workflow
	writeFieldLabel(&b, "Workflow", field == fieldWorkflow)
	switch wfMode {
	case wfNone:
		text := "None"
		if field == fieldWorkflow {
			b.WriteString(fmt.Sprintf("< %s >", text))
		} else {
			b.WriteString(dimStyle.Render(text))
		}
	case wfCustom:
		text := "Custom"
		if field == fieldWorkflow {
			b.WriteString(fmt.Sprintf("< %s >", text))
		} else {
			b.WriteString(detailValueStyle.Render(text))
		}
	case wfOOTB:
		if len(workflows) > 0 && wfIdx < len(workflows) {
			w := workflows[wfIdx]
			text := w.Name
			if field == fieldWorkflow {
				b.WriteString(fmt.Sprintf("< %s >", text))
			} else {
				b.WriteString(detailValueStyle.Render(text))
			}
		} else {
			b.WriteString(dimStyle.Render("loading..."))
		}
	}
	b.WriteString("\n")

	if wfMode == wfOOTB && len(workflows) > 0 && wfIdx < len(workflows) {
		desc := workflows[wfIdx].Description
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		b.WriteString("              ")
		b.WriteString(dimStyle.Render(desc))
		b.WriteString("\n")
	}

	if wfMode == wfCustom {
		b.WriteString("\n")
		writeFieldLabel(&b, "  Git URL", field == fieldCustomURL)
		b.WriteString(customURLView)
		b.WriteString("\n\n")

		writeFieldLabel(&b, "  Branch", field == fieldCustomBranch)
		b.WriteString(customBranchView)
		b.WriteString("\n\n")

		writeFieldLabel(&b, "  Path", field == fieldCustomPath)
		b.WriteString(customPathView)
		b.WriteString("\n")
	}

	if creating {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  Creating session..."))
	}

	content := b.String()

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
		Render(content)
}

func writeFieldLabel(b *strings.Builder, name string, active bool) {
	if active {
		b.WriteString(detailLabelStyle.Render("> " + name))
	} else {
		b.WriteString(dimStyle.Render("  " + name))
	}
	pad := 14 - len(name)
	if pad > 0 {
		b.WriteString(strings.Repeat(" ", pad))
	}
}
