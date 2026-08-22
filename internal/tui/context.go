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
	VerifySucceeded   bool
	DeploySucceeded   bool
	
	// Logs
	Logs []string
	
	// Active tab
	ActiveTab int
	Tabs      []string
}

// NewContextPanel creates a new context panel
func NewContextPanel() *ContextPanel {
	return &ContextPanel{
		CurrentStep: 1,
		TotalSteps:  7,
		ActiveTab:   0,
		Tabs:       []string{"Progress", "Config", "Model", "Logs", "Help", "Env"},
		Logs:       make([]string, 0),
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

// SetActiveTab sets the active tab
func (c *ContextPanel) SetActiveTab(tab int) {
	if tab >= 0 && tab < len(c.Tabs) {
		c.ActiveTab = tab
	}
}

// Render renders the context panel as a side panel (Shopware CLI style)
func (c *ContextPanel) Render() string {
	var sb strings.Builder
	
	// Panel box
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#555555")).
		Padding(1, 2)
	
	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#55AAFF")).
		Bold(true)
	
	sb.WriteString(boxStyle.Render(
		headerStyle.Render(" Context Panel ") + "\n\n" +
		c.renderPanelContent(),
	))
	
	return sb.String()
}

// RenderSidePanel renders the panel for side-by-side display
func (c *ContextPanel) RenderSidePanel() string {
	return c.Render()
}

// renderPanelContent renders the content of the panel (all info, not tabbed)
func (c *ContextPanel) renderPanelContent() string {
	var sb strings.Builder
	
	// Progress section
	progressPercent := (c.CurrentStep * 100) / c.TotalSteps
	barWidth := 15
	filled := int(progressPercent * barWidth / 100)
	empty := barWidth - filled
	
	progressBar := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render(strings.Repeat("░", empty))
	
	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888FF"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
	
	sb.WriteString(sectionStyle.Render("Progress\n"))
	sb.WriteString(fmt.Sprintf("  Step %d of %d\n", c.CurrentStep, c.TotalSteps))
	sb.WriteString(fmt.Sprintf("  %s %d%%\n\n", progressBar, progressPercent))
	
	// Config section
	sb.WriteString(sectionStyle.Render("Configuration\n"))
	if c.Registry != "" {
		sb.WriteString(fmt.Sprintf("  Registry:    %s\n", valueStyle.Render(c.Registry)))
	}
	if c.GitOps != "" {
		sb.WriteString(fmt.Sprintf("  GitOps:      %s\n", valueStyle.Render(c.GitOps)))
	}
	if c.Signer != "" {
		sb.WriteString(fmt.Sprintf("  Signer:      %s\n", valueStyle.Render(c.Signer)))
	}
	if c.Runtime != "" {
		sb.WriteString(fmt.Sprintf("  Runtime:     %s\n", valueStyle.Render(c.Runtime)))
	}
	sb.WriteString("\n")
	
	// Model section
	sb.WriteString(sectionStyle.Render("Model\n"))
	if c.ModelName != "" {
		sb.WriteString(fmt.Sprintf("  Name:        %s\n", valueStyle.Render(c.ModelName)))
	}
	if c.ModelPath != "" {
		sb.WriteString(fmt.Sprintf("  Path:        %s\n", valueStyle.Render(c.ModelPath)))
	}
	if c.ArtifactName != "" {
		sb.WriteString(fmt.Sprintf("  Artifact:    %s\n", valueStyle.Render(c.ArtifactName)))
	}
	sb.WriteString("\n")
	
	// Status section
	if c.PackageSucceeded || c.SignSucceeded || c.VerifySucceeded || c.DeploySucceeded {
		sb.WriteString(sectionStyle.Render("Status\n"))
		if c.PackageSucceeded {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("  ✓ Package\n"))
		}
		if c.SignSucceeded {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("  ✓ Sign\n"))
		}
		if c.VerifySucceeded {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("  ✓ Verify\n"))
		}
		if c.DeploySucceeded {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("  ✓ Deploy\n"))
		}
	}
	
	// Logs section
	if len(c.Logs) > 0 {
		sb.WriteString("\n" + sectionStyle.Render("Recent Logs\n"))
		for i := len(c.Logs) - 1; i >= 0 && i >= len(c.Logs)-3; i-- {
			if i >= 0 {
				logStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Italic(true)
				log := c.Logs[i]
				if len(log) > 40 {
					log = log[:40] + "..."
				}
				sb.WriteString(fmt.Sprintf("  - %s\n", logStyle.Render(log)))
			}
		}
	}
	
	return sb.String()
}

// renderTabBar renders the tab navigation bar
func (c *ContextPanel) renderTabBar() string {
	var sb strings.Builder
	
	for i, tab := range c.Tabs {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Padding(0, 1)
		
		if i == c.ActiveTab {
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Bold(true).
				Underline(true)
		}
		
		sb.WriteString(style.Render(tab))
		
		if i < len(c.Tabs)-1 {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render("|"))
		}
	}
	
	return sb.String()
}

// renderTabContent renders the content of all tabs (since we can't switch dynamically in static render)
// For a static display, we show the most important info from each category
func (c *ContextPanel) renderTabContent() string {
	var sb strings.Builder
	
	// Progress info
	progressPercent := (c.CurrentStep * 100) / c.TotalSteps
	sb.WriteString(fmt.Sprintf("Step %d/%d (%d%%)", c.CurrentStep, c.TotalSteps, progressPercent))
	sb.WriteString(" | ")
	
	// Config info
	sb.WriteString(fmt.Sprintf("Registry: %s, GitOps: %s, Signer: %s", 
		c.Registry, c.GitOps, c.Signer))
	sb.WriteString(" | ")
	
	// Model info
	sb.WriteString(fmt.Sprintf("Model: %s (%s)", c.ModelName, c.ArtifactName))
	
	// Add success indicators
	if c.PackageSucceeded {
		sb.WriteString(" | ")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("✓ Package"))
	}
	if c.SignSucceeded {
		sb.WriteString(" ")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("✓ Sign"))
	}
	if c.VerifySucceeded {
		sb.WriteString(" ")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("✓ Verify"))
	}
	if c.DeploySucceeded {
		sb.WriteString(" ")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("✓ Deploy"))
	}
	
	// Logs
	if len(c.Logs) > 0 {
		sb.WriteString("\n")
		logsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		lastLog := c.Logs[len(c.Logs)-1]
		if len(lastLog) > 60 {
			lastLog = lastLog[:60] + "..."
		}
		sb.WriteString(logsStyle.Render("Last: " + lastLog))
	}
	
	sb.WriteString("\n")
	return sb.String()
}

// RenderFull renders all tab contents in a multi-line format
func (c *ContextPanel) RenderFull() string {
	var sb strings.Builder
	
	// Progress tab
	progressPercent := (c.CurrentStep * 100) / c.TotalSteps
	sb.WriteString(fmt.Sprintf("Progress: Step %d/%d (%d%%)\n", c.CurrentStep, c.TotalSteps, progressPercent))
	
	// Config tab
	sb.WriteString(fmt.Sprintf("Config: Registry=%s, GitOps=%s, Signer=%s, Runtime=%s\n", 
		c.Registry, c.GitOps, c.Signer, c.Runtime))
	
	// Model tab
	sb.WriteString(fmt.Sprintf("Model: Name=%s, Path=%s, Artifact=%s\n", 
		c.ModelName, c.ModelPath, c.ArtifactName))
	
	// Logs tab
	if len(c.Logs) > 0 {
		sb.WriteString("Logs:\n")
		for _, log := range c.Logs {
			sb.WriteString(fmt.Sprintf("  - %s\n", log))
		}
	} else {
		sb.WriteString("Logs: No logs yet\n")
	}
	
	// Help tab
	sb.WriteString("Help: Use Tab/Shift+Tab to switch context views\n")
	
	// Environment tab
	sb.WriteString("Environment: Model CLI Wizard\n")
	
	return sb.String()
}
