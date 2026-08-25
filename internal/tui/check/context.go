// Package checktui provides the context panel for compliance checking
package checktui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tab represents a tab in the context panel
type Tab int

const (
	TabProgress Tab = iota
	TabConfig
	TabModel
	TabChecks
	TabResults
	TabLogs
	TabHelp
	TabEnv
)

// ContextModel is the interactive context panel model for compliance checking
type ContextModel struct {
	// Configuration
	Registry string
	GitOps   string
	Signer   string
	Runtime  string

	// Model information
	ModelName    string
	ModelPath    string
	ArtifactName string
	ArtifactPath string

	// Results
	Passed           bool
	SBOMCheck        bool
	MOFCheck         bool
	Missing          []string
	AnnotationsCheck []string

	// Step progress
	CurrentStep int
	TotalSteps  int

	// Logs
	Logs []string

	// Active tab
	ActiveTab Tab

	// Dimensions
	width  int
	height int
}

// NewCheckContextModel creates a new interactive context model for compliance checking
func NewCheckContextModel() *ContextModel {
	return &ContextModel{
		CurrentStep: 1,
		TotalSteps:  4,
		ActiveTab:   TabProgress,
		Logs:        make([]string, 0),
		width:       40,
		height:      15,
		Passed:      false,
		SBOMCheck:   false,
		MOFCheck:    false,
		Missing:     make([]string, 0),
	}
}

// Init initializes the context model
func (m *ContextModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *ContextModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		m.HandleKey(msg.String())
	}
	return m, nil
}

// View renders the context panel
func (m *ContextModel) View() string {
	// Box style
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#555555")).
		Padding(1, 2)

	// Header
	header := m.renderHeader()

	// Tab bar
	tabBar := m.renderTabBar()

	// Tab content
	content := m.renderTabContent()

	// Combine
	var sb strings.Builder
	sb.WriteString(boxStyle.Render(header + "\n" + tabBar + "\n" + content))

	return sb.String()
}

// renderHeader renders the header
func (m *ContextModel) renderHeader() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#55AAFF")).
		Bold(true)
	return headerStyle.Render("Context Panel")
}

// renderTabBar renders the tab navigation bar
func (m *ContextModel) renderTabBar() string {
	tabs := []string{"Progress", "Config", "Model", "Checks", "Results", "Logs", "Help", "Env"}

	var sb strings.Builder

	for i, tab := range tabs {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Padding(0, 1)

		if i == int(m.ActiveTab) {
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Bold(true).
				Underline(true)
		}

		sb.WriteString(style.Render(tab))

		if i < len(tabs)-1 {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render("|"))
		}
	}

	return sb.String()
}

// renderTabContent renders the content of the active tab
func (m *ContextModel) renderTabContent() string {
	switch m.ActiveTab {
	case TabProgress:
		return m.renderProgressTab()
	case TabConfig:
		return m.renderConfigTab()
	case TabModel:
		return m.renderModelTab()
	case TabChecks:
		return m.renderChecksTab()
	case TabResults:
		return m.renderResultsTab()
	case TabLogs:
		return m.renderLogsTab()
	case TabHelp:
		return m.renderHelpTab()
	case TabEnv:
		return m.renderEnvTab()
	}
	return ""
}

// renderProgressTab renders the progress tab
func (m *ContextModel) renderProgressTab() string {
	var sb strings.Builder

	progressPercent := (m.CurrentStep * 100) / m.TotalSteps
	barWidth := 12
	filled := int(progressPercent * barWidth / 100)
	empty := barWidth - filled

	progressBar := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Render(strings.Repeat("░", empty))

	sb.WriteString(fmt.Sprintf("Step %d of %d\n", m.CurrentStep, m.TotalSteps))
	sb.WriteString(fmt.Sprintf("%s %d%%\n\n", progressBar, progressPercent))

	// Status of each step
	sb.WriteString("Check Steps:\n")
	steps := []string{"Intro", "Checking", "Results", "Complete"}
	for i, step := range steps {
		symbol := "·"
		if i < m.CurrentStep {
			symbol = "✓"
		}
		if i == m.CurrentStep {
			symbol = "→"
		}
		if i > m.CurrentStep {
			symbol = "·"
		}

		// Color based on results
		color := "#555555"
		if i == m.CurrentStep {
			color = "#55AAFF"
		}
		if i < m.CurrentStep {
			color = "#00FF88"
		}

		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		sb.WriteString(fmt.Sprintf("  %s %s\n", symbol, style.Render(step)))
	}

	return sb.String()
}

// renderConfigTab renders the configuration tab
func (m *ContextModel) renderConfigTab() string {
	var sb strings.Builder

	sb.WriteString("Configuration\n")
	sb.WriteString(strings.Repeat("─", 12) + "\n\n")

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888FF"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))

	if m.Registry != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", keyStyle.Render("Registry"), valueStyle.Render(m.Registry)))
	}
	if m.GitOps != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", keyStyle.Render("GitOps"), valueStyle.Render(m.GitOps)))
	}
	if m.Signer != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", keyStyle.Render("Signer"), valueStyle.Render(m.Signer)))
	}
	if m.Runtime != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", keyStyle.Render("Runtime"), valueStyle.Render(m.Runtime)))
	}

	return sb.String()
}

// renderModelTab renders the model tab
func (m *ContextModel) renderModelTab() string {
	var sb strings.Builder

	sb.WriteString("Model Information\n")
	sb.WriteString(strings.Repeat("─", 17) + "\n\n")

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888FF"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))

	if m.ModelName != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", keyStyle.Render("Name"), valueStyle.Render(m.ModelName)))
	}
	if m.ModelPath != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", keyStyle.Render("Path"), valueStyle.Render(m.ModelPath)))
	}
	if m.ArtifactName != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", keyStyle.Render("Artifact"), valueStyle.Render(m.ArtifactName)))
	}
	if m.ArtifactPath != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", keyStyle.Render("Artifact Path"), valueStyle.Render(m.ArtifactPath)))
	}

	return sb.String()
}

// renderChecksTab renders the checks tab
func (m *ContextModel) renderChecksTab() string {
	var sb strings.Builder

	sb.WriteString("Compliance Checks\n")
	sb.WriteString(strings.Repeat("─", 17) + "\n\n")

	sb.WriteString("Required checks:\n")
	sb.WriteString("  • Required annotations\n")
	sb.WriteString("  • SBOM presence\n")
	sb.WriteString("  • MOF classification\n")

	return sb.String()
}

// renderResultsTab renders the results tab
func (m *ContextModel) renderResultsTab() string {
	var sb strings.Builder

	sb.WriteString("Check Results\n")
	sb.WriteString(strings.Repeat("─", 13) + "\n\n")

	if !m.Passed && len(m.Missing) == 0 && !m.SBOMCheck && !m.MOFCheck {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Checks not yet run"))
		return sb.String()
	}

	// Overall status
	if m.Passed {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("✓ ALL PASSED\n\n"))
	} else {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("✗ FAILED\n\n"))
	}

	// Individual checks
	status := "✓"
	if !m.SBOMCheck {
		status = "✗"
	}
	sb.WriteString(fmt.Sprintf("%s SBOM\n", status))

	status = "✓"
	if !m.MOFCheck {
		status = "✗"
	}
	sb.WriteString(fmt.Sprintf("%s MOF Classification\n", status))

	if len(m.Missing) > 0 {
		sb.WriteString("\n")
		sb.WriteString("Missing:\n")
		for _, item := range m.Missing {
			sb.WriteString(fmt.Sprintf("  ✗ %s\n", item))
		}
	}

	return sb.String()
}

// renderLogsTab renders the logs tab
func (m *ContextModel) renderLogsTab() string {
	var sb strings.Builder

	sb.WriteString("Recent Logs\n")
	sb.WriteString(strings.Repeat("─", 11) + "\n\n")

	if len(m.Logs) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("No logs yet"))
		return sb.String()
	}

	// Show last 10 logs
	start := 0
	if len(m.Logs) > 10 {
		start = len(m.Logs) - 10
	}

	for i := start; i < len(m.Logs); i++ {
		log := m.Logs[i]
		// Truncate long logs
		if len(log) > 40 {
			log = log[:40] + "..."
		}
		sb.WriteString(fmt.Sprintf("  - %s\n", log))
	}

	return sb.String()
}

// renderHelpTab renders the help tab
func (m *ContextModel) renderHelpTab() string {
	var sb strings.Builder

	sb.WriteString("Keyboard Shortcuts\n")
	sb.WriteString(strings.Repeat("─", 19) + "\n\n")

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	sb.WriteString(fmt.Sprintf("%s\n", helpStyle.Render("Tab / Shift+Tab: Switch tabs")))
	sb.WriteString(fmt.Sprintf("%s\n", helpStyle.Render("1-8: Select specific tab")))
	sb.WriteString(fmt.Sprintf("%s\n", helpStyle.Render("Enter: Continue / Select")))
	sb.WriteString(fmt.Sprintf("%s\n", helpStyle.Render("Esc: Back / Cancel")))
	sb.WriteString(fmt.Sprintf("%s\n", helpStyle.Render("Ctrl+C: Quit")))

	return sb.String()
}

// renderEnvTab renders the environment tab
func (m *ContextModel) renderEnvTab() string {
	var sb strings.Builder

	sb.WriteString("Environment\n")
	sb.WriteString(strings.Repeat("─", 12) + "\n\n")

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	sb.WriteString(fmt.Sprintf("%s\n", infoStyle.Render("Model CLI Version: 0.1.0")))
	sb.WriteString(fmt.Sprintf("%s\n", infoStyle.Render("Terminal: Detected")))
	sb.WriteString(fmt.Sprintf("%s\n", infoStyle.Render("Go Version: 1.20+")))

	return sb.String()
}

// SetStep sets the current step
func (m *ContextModel) SetStep(step int) {
	m.CurrentStep = step
}

// SetConfig sets the configuration
func (m *ContextModel) SetConfig(registry, gitOps, signer, runtime string) {
	m.Registry = registry
	m.GitOps = gitOps
	m.Signer = signer
	m.Runtime = runtime
}

// SetModelInfo sets the model information
func (m *ContextModel) SetModelInfo(name, path, artifact, artifactPath string) {
	m.ModelName = name
	m.ModelPath = path
	m.ArtifactName = artifact
	m.ArtifactPath = artifactPath
}

// SetResults sets the check results
func (m *ContextModel) SetResults(passed, sbomCheck, mofCheck bool, missing []string) {
	m.Passed = passed
	m.SBOMCheck = sbomCheck
	m.MOFCheck = mofCheck
	m.Missing = missing
}

// AddLog adds a log message
func (m *ContextModel) AddLog(message string) {
	m.Logs = append(m.Logs, message)
	if len(m.Logs) > 50 {
		m.Logs = m.Logs[len(m.Logs)-50:]
	}
}

// HandleKey handles keyboard input for the context model
func (m *ContextModel) HandleKey(key string) {
	switch key {
	case "tab":
		m.ActiveTab = (m.ActiveTab + 1) % 8

	case "shift+tab":
		m.ActiveTab = (m.ActiveTab - 1 + 8) % 8

	case "1":
		m.ActiveTab = TabProgress

	case "2":
		m.ActiveTab = TabConfig

	case "3":
		m.ActiveTab = TabModel

	case "4":
		m.ActiveTab = TabChecks

	case "5":
		m.ActiveTab = TabResults

	case "6":
		m.ActiveTab = TabLogs

	case "7":
		m.ActiveTab = TabHelp

	case "8":
		m.ActiveTab = TabEnv
	}
}
