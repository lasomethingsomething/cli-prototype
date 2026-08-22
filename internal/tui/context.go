package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ContextPanel displays contextual information in a formatted panel at the bottom
type ContextPanel struct {
	// Current step information
	CurrentStep int
	TotalSteps  int

	// Configuration
	Registry string
	GitOps   string
	Signer   string
	Runtime  string

	// Model information
	ModelName    string
	ModelPath    string
	ArtifactName string

	// Success flags
	PackageSucceeded bool
	SignSucceeded    bool
	VerifySucceeded  bool
	DeploySucceeded  bool

	// Logs
	Logs []string
}

// NewContextPanel creates a new context panel
func NewContextPanel() *ContextPanel {
	return &ContextPanel{
		CurrentStep: 1,
		TotalSteps:  7,
		Logs:        make([]string, 0),
	}
}

// SetStep sets the current step
func (c *ContextPanel) SetStep(step int) {
	c.CurrentStep = step
}

// SetConfig sets the configuration
func (c *ContextPanel) SetConfig(registry, gitOps, signer, runtime string) {
	c.Registry = registry
	c.GitOps = gitOps
	c.Signer = signer
	c.Runtime = runtime
}

// SetModelInfo sets the model information
func (c *ContextPanel) SetModelInfo(name, path, artifact string) {
	c.ModelName = name
	c.ModelPath = path
	c.ArtifactName = artifact
}

// SetResults sets the success flags
func (c *ContextPanel) SetResults(packageSucceeded, signSucceeded, verifySucceeded, deploySucceeded bool) {
	c.PackageSucceeded = packageSucceeded
	c.SignSucceeded = signSucceeded
	c.VerifySucceeded = verifySucceeded
	c.DeploySucceeded = deploySucceeded
}

// AddLog adds a log message
func (c *ContextPanel) AddLog(message string) {
	c.Logs = append(c.Logs, message)
	if len(c.Logs) > 10 {
		c.Logs = c.Logs[1:]
	}
}

// Render renders the context panel as a side panel (Shopware CLI style)
func (c *ContextPanel) Render() string {
	var sb strings.Builder

	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#555555")).
		Padding(1, 2)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#55AAFF")).
		Bold(true)

	sb.WriteString(boxStyle.Render(
		headerStyle.Render("Context Panel") + "\n\n" +
			c.renderPanelContent(),
	))

	return sb.String()
}

// RenderSidePanel renders the panel for side-by-side display
func (c *ContextPanel) RenderSidePanel() string {
	return c.Render()
}

// renderPanelContent renders the content of the panel
func (c *ContextPanel) renderPanelContent() string {
	var sb strings.Builder

	progressPercent := (c.CurrentStep * 100) / c.TotalSteps
	barWidth := 12
	filled := int(progressPercent * barWidth / 100)
	empty := barWidth - filled

	progressBar := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render(strings.Repeat("░", empty))

	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888FF")).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88"))
	logStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Italic(true)

	sb.WriteString(sectionStyle.Render("Progress") + "\n")
	sb.WriteString(fmt.Sprintf("  Step %d of %d\n", c.CurrentStep, c.TotalSteps))
	sb.WriteString(fmt.Sprintf("  %s %d%%\n\n", progressBar, progressPercent))

	if c.Registry != "" || c.GitOps != "" || c.Signer != "" || c.Runtime != "" {
		sb.WriteString(sectionStyle.Render("Configuration") + "\n")
		if c.Registry != "" {
			sb.WriteString(fmt.Sprintf("  Registry:  %s\n", valueStyle.Render(c.Registry)))
		}
		if c.GitOps != "" {
			sb.WriteString(fmt.Sprintf("  GitOps:    %s\n", valueStyle.Render(c.GitOps)))
		}
		if c.Signer != "" {
			sb.WriteString(fmt.Sprintf("  Signer:    %s\n", valueStyle.Render(c.Signer)))
		}
		if c.Runtime != "" {
			sb.WriteString(fmt.Sprintf("  Runtime:   %s\n", valueStyle.Render(c.Runtime)))
		}
		sb.WriteString("\n")
	}

	if c.ModelName != "" || c.ModelPath != "" || c.ArtifactName != "" {
		sb.WriteString(sectionStyle.Render("Model") + "\n")
		if c.ModelName != "" {
			sb.WriteString(fmt.Sprintf("  Name:      %s\n", valueStyle.Render(c.ModelName)))
		}
		if c.ModelPath != "" {
			sb.WriteString(fmt.Sprintf("  Path:      %s\n", valueStyle.Render(c.ModelPath)))
		}
		if c.ArtifactName != "" {
			sb.WriteString(fmt.Sprintf("  Artifact:  %s\n", valueStyle.Render(c.ArtifactName)))
		}
		sb.WriteString("\n")
	}

	if c.PackageSucceeded || c.SignSucceeded || c.VerifySucceeded || c.DeploySucceeded {
		sb.WriteString(sectionStyle.Render("Status") + "\n")
		if c.PackageSucceeded {
			sb.WriteString("  " + successStyle.Render("✓ Package") + "\n")
		}
		if c.SignSucceeded {
			sb.WriteString("  " + successStyle.Render("✓ Sign") + "\n")
		}
		if c.VerifySucceeded {
			sb.WriteString("  " + successStyle.Render("✓ Verify") + "\n")
		}
		if c.DeploySucceeded {
			sb.WriteString("  " + successStyle.Render("✓ Deploy") + "\n")
		}
		sb.WriteString("\n")
	}

	if len(c.Logs) > 0 {
		sb.WriteString(sectionStyle.Render("Recent Logs") + "\n")
		for i := len(c.Logs) - 1; i >= 0 && i >= len(c.Logs)-3; i-- {
			if i >= 0 {
				log := c.Logs[i]
				if len(log) > 45 {
					log = log[:45] + "..."
				}
				sb.WriteString("  - " + logStyle.Render(log) + "\n")
			}
		}
	}

	return sb.String()
}
