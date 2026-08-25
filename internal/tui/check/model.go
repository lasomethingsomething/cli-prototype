// Package checktui provides an interactive TUI for compliance checking
// following the Shopware CLI pattern with panels, tabs, and real-time progress
package checktui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
)

// Panel represents different views in the check workflow
type Panel int

const (
	PanelIntro Panel = iota
	PanelChecking
	PanelResults
	PanelComplete
)

// StepState represents the state of a step
type StepState int

const (
	StepPending StepState = iota
	StepRunning
	StepDone
	StepError
)

// Step represents a step in the check workflow
type Step struct {
	Label  string
	State  StepState
	Detail string
}

// Model holds the state of the check TUI
type Model struct {
	// Current panel
	panel Panel

	// Dimensions
	width  int
	height int

	// Workflow
	workflow *workflow.CheckWorkflow

	// Model info
	modelName    string
	modelPath    string
	artifactName string
	artifactPath string

	// Results
	passed           bool
	missing          []string
	sbomCheck        bool
	mofCheck         bool
	annotationsCheck []workflow.AnnotationCheckResult

	// State
	loading   bool
	completed bool
	err       error
	logs      []string

	// Steps
	steps       []Step
	currentStep int
}

// New creates a new check TUI model
func New() *Model {
	return &Model{
		panel:       PanelIntro,
		width:       80,
		height:      24,
		logs:        make([]string, 0),
		currentStep: 0,
		steps: []Step{
			{Label: "Check required annotations", State: StepPending},
			{Label: "Check SBOM presence", State: StepPending},
			{Label: "Check MOF classification", State: StepPending},
		},
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	m.workflow = workflow.NewCheckWorkflow()
	return nil
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// View renders the complete UI
func (m *Model) View() string {
	var sb strings.Builder

	// Header
	header := m.renderHeader()
	sb.WriteString(header)
	sb.WriteString("\n")

	// Status strip
	status := m.renderStatus()
	if status != "" {
		sb.WriteString(status)
		sb.WriteString("\n")
	}

	// Body with context panel
	body := m.renderBody()
	sb.WriteString(body)

	// Footer with hints
	footer := m.renderFooter()
	sb.WriteString("\n")
	sb.WriteString(footer)

	return sb.String()
}

// handleKey handles keyboard input
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "enter":
		return m.handleEnter()

	case "esc":
		return m.handleEscape()
	}

	return m, nil
}

// handleEnter handles the enter key
func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.panel {
	case PanelIntro:
		m.panel = PanelChecking
		m.steps[0].State = StepRunning
		m.currentStep = 1
		return m, m.startChecking()

	case PanelChecking:
		if m.completed {
			m.panel = PanelResults
		}

	case PanelResults:
		m.panel = PanelComplete

	case PanelComplete:
		return m, tea.Quit
	}

	return m, nil
}

// handleEscape handles the escape key (go back)
func (m *Model) handleEscape() (tea.Model, tea.Cmd) {
	switch m.panel {
	case PanelChecking:
		// Can't go back from checking
	case PanelResults:
		m.panel = PanelChecking
	case PanelComplete:
		m.panel = PanelResults
	}

	return m, nil
}

// startChecking starts the compliance check process
func (m *Model) startChecking() tea.Cmd {
	m.loading = true
	m.steps[0].State = StepRunning
	m.steps[0].Detail = "Checking annotations..."

	return func() tea.Msg {
		// Set up the workflow
		m.workflow.SetCheckInfo(m.modelPath, m.artifactPath)

		// Run the workflow
		err := m.workflow.Run()

		m.loading = false
		m.completed = true
		m.err = err

		if err != nil {
			m.logs = append(m.logs, fmt.Sprintf("Error: %v", err))
		} else {
			m.logs = append(m.logs, "Compliance check completed")
		}

		// Get results
		m.passed = m.workflow.Passed()
		m.missing = m.workflow.Missing()
		m.sbomCheck = m.workflow.SBOMCheck()
		m.mofCheck = m.workflow.MOFCheck()
		m.annotationsCheck = m.workflow.AnnotationsCheck()

		// Update steps
		m.steps[0].State = StepDone
		m.steps[0].Detail = "Annotations checked"
		m.steps[1].State = StepDone
		m.steps[1].Detail = "SBOM checked"
		m.steps[2].State = StepDone
		m.steps[2].Detail = "MOF checked"

		return nil
	}
}

// renderHeader renders the header with title and context
func (m *Model) renderHeader() string {
	title := "Local Compliance Check"
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Bold(true)

	context := fmt.Sprintf("Model: %s | Artifact: %s", m.modelName, m.artifactName)
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888FF"))

	return titleStyle.Render(title) + "  " + contextStyle.Render(context)
}

// renderStatus renders the status strip
func (m *Model) renderStatus() string {
	if m.loading {
		return renderStatusStrip("RUNNING", "Checking compliance...", lipgloss.Color("#55AAFF"))
	}
	if m.err != nil {
		return renderStatusStrip("ERROR", m.err.Error(), lipgloss.Color("#FF5555"))
	}
	if m.completed && m.passed {
		return renderStatusStrip("PASSED", "All compliance checks passed", lipgloss.Color("#00FF88"))
	}
	if m.completed && !m.passed {
		return renderStatusStrip("FAILED", "Compliance checks failed", lipgloss.Color("#FF5555"))
	}
	return ""
}

// renderBody renders the main body content with context panel
func (m *Model) renderBody() string {
	// Main content based on panel
	mainContent := m.renderMainContent()

	// Context panel on the right
	contextPanel := m.renderContextPanel()

	// Two-column layout
	return renderTwoColumn(m.width, mainContent, contextPanel)
}

// renderMainContent renders the main workflow content
func (m *Model) renderMainContent() string {
	switch m.panel {
	case PanelIntro:
		return m.renderIntro()
	case PanelChecking:
		return m.renderChecking()
	case PanelResults:
		return m.renderResults()
	case PanelComplete:
		return m.renderComplete()
	}
	return ""
}

// renderContextPanel renders the context panel with tabs
func (m *Model) renderContextPanel() string {
	// Create a context model with current state
	contextModel := NewCheckContextModel()
	contextModel.SetModelInfo(m.modelName, m.modelPath, m.artifactName, m.artifactPath)
	contextModel.SetStep(m.currentStep + 1)

	// Update with results if available
	if m.completed {
		contextModel.SetResults(m.passed, m.sbomCheck, m.mofCheck, m.missing)
		for _, log := range m.logs {
			contextModel.AddLog(log)
		}
	}

	return contextModel.View()
}

// renderFooter renders the footer with shortcut hints
func (m *Model) renderFooter() string {
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	switch m.panel {
	case PanelIntro:
		return hintStyle.Render("enter: Start  |  esc: Quit")
	case PanelChecking:
		return hintStyle.Render("Wait...  |  ctrl+c: Cancel")
	case PanelResults:
		if m.passed {
			return hintStyle.Render("enter: Continue")
		}
		return hintStyle.Render("enter: View Details  |  esc: Back")
	case PanelComplete:
		return hintStyle.Render("enter: Finish")
	}
	return ""
}

// renderStatusStrip renders a status strip
func renderStatusStrip(label, message string, color lipgloss.Color) string {
	labelStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
	messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA"))
	return labelStyle.Render(label) + "  " + messageStyle.Render(message)
}

// renderIntro renders the introduction panel
func (m *Model) renderIntro() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("Local Compliance Check\n")
	sb.WriteString(strings.Repeat("─", 25) + "\n\n")

	sb.WriteString("Validates a locally-built artifact before push.\n\n")

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888FF"))
	sb.WriteString(infoStyle.Render("Checks performed:\n\n"))

	sb.WriteString("  ✓ Required annotations present\n")
	sb.WriteString("    - org.cncf.ai.artifact.type\n")
	sb.WriteString("    - org.cncf.ai.artifact.runtime\n")
	sb.WriteString("    - org.cncf.ai.artifact.accelerator\n\n")

	sb.WriteString("  ✓ SBOM (Software Bill of Materials) present\n")
	sb.WriteString("    - Attached to artifact layers\n")
	sb.WriteString("    - Required for supply chain transparency\n\n")

	sb.WriteString("  ✓ MOF (Model Openness Framework) classification\n")
	sb.WriteString("    - Class I, II, or III\n")
	sb.WriteString("    - Machine-readable metadata\n")

	return sb.String()
}

// renderChecking renders the checking progress panel
func (m *Model) renderChecking() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("Running Compliance Checks\n")
	sb.WriteString(strings.Repeat("─", 25) + "\n\n")

	if m.loading {
		sb.WriteString("→ Checking required annotations...\n")
		sb.WriteString("→ Checking SBOM presence...\n")
		sb.WriteString("→ Checking MOF classification...\n")
	} else if m.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("✗ Error: "+m.err.Error()) + "\n")
	} else if m.completed {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("✓ Compliance checks completed\n"))
	}

	return sb.String()
}

// renderResults renders the results panel
func (m *Model) renderResults() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("Compliance Check Results\n")
	sb.WriteString(strings.Repeat("─", 26) + "\n\n")

	// Overall status
	if m.passed {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("✓ ALL CHECKS PASSED\n\n"))
	} else {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("✗ COMPLIANCE CHECK FAILED\n\n"))
	}

	// Individual checks
	sb.WriteString("Individual Checks:\n\n")

	status := "✓ PASS"
	if !m.sbomCheck {
		status = "✗ FAIL"
	}
	sb.WriteString(fmt.Sprintf("  %s - SBOM present\n", status))

	status = "✓ PASS"
	if !m.mofCheck {
		status = "✗ FAIL"
	}
	sb.WriteString(fmt.Sprintf("  %s - MOF classification present\n", status))

	// Annotations check
	if len(m.annotationsCheck) > 0 {
		sb.WriteString("\n")
		sb.WriteString("Annotation Checks:\n")
		for _, check := range m.annotationsCheck {
			status := "✓"
			if !check.Passed {
				status = "✗"
			}
			sb.WriteString(fmt.Sprintf("  %s %s: %s\n", status, check.Name, check.Actual))
		}
	}

	// Missing items
	if len(m.missing) > 0 {
		sb.WriteString("\n")
		sb.WriteString("Missing Required Items:\n")
		for _, item := range m.missing {
			sb.WriteString(fmt.Sprintf("  ✗ %s\n", item))
		}
	}

	sb.WriteString("\n")
	if m.passed {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("Artifact is compliant and ready for push"))
	} else {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("Artifact cannot be pushed - compliance issues must be resolved"))
	}

	return sb.String()
}

// renderComplete renders the completion panel
func (m *Model) renderComplete() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("✓ Local Compliance Check Complete!\n")
	sb.WriteString(strings.Repeat("─", 32) + "\n\n")

	sb.WriteString("Summary:\n\n")

	if m.passed {
		sb.WriteString("  ✓ All compliance checks passed\n")
	} else {
		sb.WriteString("  ✗ Compliance checks failed\n")
	}

	sb.WriteString(fmt.Sprintf("  SBOM:        %s\n", checkResult(m.sbomCheck)))
	sb.WriteString(fmt.Sprintf("  MOF:         %s\n", checkResult(m.mofCheck)))

	if len(m.missing) > 0 {
		sb.WriteString("\n")
		sb.WriteString("Missing:\n")
		for _, item := range m.missing {
			sb.WriteString(fmt.Sprintf("  ✗ %s\n", item))
		}
	}

	sb.WriteString("\n")
	if m.passed {
		sb.WriteString("Next: Push artifact to registry\n")
	} else {
		sb.WriteString("Action required: Fix compliance issues before pushing\n")
	}

	return sb.String()
}

// renderTwoColumn renders content in a two-column layout
func renderTwoColumn(width int, left, right string) string {
	leftWidth := width * 60 / 100
	if leftWidth < 40 {
		leftWidth = 40
	}
	if leftWidth > width-40 {
		leftWidth = width - 40
	}

	left = lipgloss.NewStyle().Width(leftWidth).Render(left)
	right = lipgloss.NewStyle().Width(width - leftWidth - 2).Render(right)

	return left + "  " + right
}

func checkResult(passed bool) string {
	if passed {
		return "✓ PASS"
	}
	return "✗ FAIL"
}

// SetModelInfo sets the model information
func (m *Model) SetModelInfo(name, path, artifact, artifactPath string) {
	m.modelName = name
	m.modelPath = path
	m.artifactName = artifact
	m.artifactPath = artifactPath
}

// Passed returns whether all checks passed
func (m *Model) Passed() bool {
	return m.passed
}

// Missing returns the list of missing required items
func (m *Model) Missing() []string {
	return m.missing
}

// SBOMCheck returns whether SBOM check passed
func (m *Model) SBOMCheck() bool {
	return m.sbomCheck
}

// MOFCheck returns whether MOF check passed
func (m *Model) MOFCheck() bool {
	return m.mofCheck
}

// Completed returns whether the workflow is complete
func (m *Model) Completed() bool {
	return m.completed
}

// Error returns any error that occurred
func (m *Model) Error() error {
	return m.err
}
