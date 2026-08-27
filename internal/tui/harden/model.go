// Package hardentui provides an interactive TUI for hardening & compliance
// following the Shopware CLI pattern with panels, tabs, and real-time progress
package hardentui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
)

// Panel represents different views in the harden workflow
type Panel int

const (
	PanelIntro Panel = iota
	PanelSBOM
	PanelMOF
	PanelReview
	PanelHardening
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

// Step represents a step in the hardening workflow
type Step struct {
	Label  string
	State  StepState
	Detail string
}

// Model holds the state of the harden TUI
type Model struct {
	// Current panel
	panel Panel

	// Dimensions
	width  int
	height int

	// Workflow
	workflow *workflow.HardenWorkflow
	registry string

	// Model info
	modelName    string
	modelPath    string
	artifactName string

	// Hardening options
	generateSBOM bool
	includeMOF   bool
	sbomTool     string
	sbomFormat   workflow.SBOMFormat

	// Results
	sbomPath      string
	mofClass      string
	mofConfigPath string

	// State
	loading   bool
	completed bool
	err       error
	logs      []string

	// Steps
	steps       []Step
	currentStep int
}

// New creates a new harden TUI model
func New(registry string) *Model {
	return &Model{
		panel:        PanelIntro,
		registry:     registry,
		width:        80,
		height:       24,
		logs:         make([]string, 0),
		generateSBOM: true,
		includeMOF:   true,
		sbomTool:     workflow.SBOMToolOptions().Recommended(),
		sbomFormat:   workflow.SPDXJSON,
		currentStep:  0,
		steps: []Step{
			{Label: "Generate SBOM", State: StepPending},
			{Label: "Apply MOF classification", State: StepPending},
			{Label: "Apply security annotations", State: StepPending},
			{Label: "Review results", State: StepPending},
		},
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	m.workflow = workflow.NewHardenWorkflow(m.registry)
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
		m.panel = PanelSBOM
		m.steps[0].State = StepDone
		m.currentStep = 1

	case PanelSBOM:
		m.panel = PanelMOF
		m.steps[1].State = StepDone
		m.currentStep = 2

	case PanelMOF:
		m.panel = PanelReview
		m.steps[2].State = StepDone
		m.currentStep = 3

	case PanelReview:
		m.panel = PanelHardening
		m.steps[3].State = StepRunning
		m.currentStep = 4
		return m, m.startHardening()

	case PanelHardening:
		if m.completed {
			m.panel = PanelComplete
		}

	case PanelComplete:
		return m, tea.Quit
	}

	return m, nil
}

// handleEscape handles the escape key (go back)
func (m *Model) handleEscape() (tea.Model, tea.Cmd) {
	switch m.panel {
	case PanelSBOM:
		m.panel = PanelIntro
		m.steps[0].State = StepPending
		m.currentStep = 0

	case PanelMOF:
		m.panel = PanelSBOM
		m.steps[1].State = StepPending
		m.currentStep = 1

	case PanelReview:
		m.panel = PanelMOF
		m.steps[2].State = StepPending
		m.currentStep = 2

	case PanelHardening:
		// Can't go back from hardening
	}

	return m, nil
}

// startHardening starts the hardening process
func (m *Model) startHardening() tea.Cmd {
	m.loading = true
	m.steps[0].State = StepRunning
	m.steps[0].Detail = "Starting SBOM generation..."

	return func() tea.Msg {
		// Set up the workflow
		m.workflow.SetHardenInfo(m.modelName, m.modelPath, m.artifactName)
		m.workflow.SetOptions(m.generateSBOM, m.includeMOF)
		m.workflow.SetSBOMTool(m.sbomTool, m.sbomFormat)

		// Create annotations to be updated
		annotations := workflow.NewAnnotationSet()
		m.workflow.SetAnnotations(annotations)

		// Run the workflow
		err := m.workflow.Run()

		m.loading = false
		m.completed = true
		m.err = err

		if err != nil {
			m.steps[0].State = StepError
			m.steps[0].Detail = err.Error()
			m.logs = append(m.logs, fmt.Sprintf("Error: %v", err))
		} else {
			m.steps[0].State = StepDone
			m.steps[0].Detail = "SBOM generated"
			m.steps[1].State = StepDone
			m.steps[1].Detail = "MOF classification applied"
			m.steps[2].State = StepDone
			m.steps[2].Detail = "MOF metadata config generated"
			m.steps[3].State = StepDone
			m.steps[3].Detail = "Security annotations applied"

			// Get results
			m.sbomPath = m.workflow.SBOMPath()
			m.mofClass = m.workflow.MOFClass()
			m.mofConfigPath = m.workflow.MOFConfigPath()

			m.logs = append(m.logs, "SBOM generated successfully")
			m.logs = append(m.logs, fmt.Sprintf("MOF Class: %s", m.mofClass))
			if m.mofConfigPath != "" {
				m.logs = append(m.logs, fmt.Sprintf("MOF config file: %s", m.mofConfigPath))
			}
			m.logs = append(m.logs, "Security annotations applied")
		}

		return nil
	}
}

// renderHeader renders the header with title and context
func (m *Model) renderHeader() string {
	title := "Step 2: Local Hardening & Compliance"
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Bold(true)

	context := fmt.Sprintf("Model: %s | Artifact: %s", m.modelName, m.artifactName)
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888FF"))

	return titleStyle.Render(title) + "  " + contextStyle.Render(context)
}

// renderStatus renders the status strip
func (m *Model) renderStatus() string {
	if m.loading {
		return renderStatusStrip("RUNNING", "Hardening and compliance checks in progress...", lipgloss.Color("#55AAFF"))
	}
	if m.err != nil {
		return renderStatusStrip("ERROR", m.err.Error(), lipgloss.Color("#FF5555"))
	}
	if m.completed {
		return renderStatusStrip("COMPLETE", "Hardening & Compliance complete", lipgloss.Color("#00FF88"))
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
	case PanelSBOM:
		return m.renderSBOMPanel()
	case PanelMOF:
		return m.renderMOFPanel()
	case PanelReview:
		return m.renderReview()
	case PanelHardening:
		return m.renderHardening()
	case PanelComplete:
		return m.renderComplete()
	}
	return ""
}

// renderContextPanel renders the context panel with tabs
func (m *Model) renderContextPanel() string {
	// Create a context model with current state
	contextModel := NewHardenContextModel()
	contextModel.SetModelInfo(m.modelName, m.modelPath, m.artifactName)
	contextModel.SetStep(m.currentStep + 1)
	contextModel.SetOptions(m.generateSBOM, m.includeMOF)
	contextModel.SetSBOMTool(m.sbomTool, string(m.sbomFormat))

	// Update with results if available
	if m.sbomPath != "" {
		contextModel.SetSBOMPath(m.sbomPath)
	}
	if m.mofClass != "" {
		contextModel.SetMOFClass(m.mofClass)
	}

	// Add logs
	for _, log := range m.logs {
		contextModel.AddLog(log)
	}

	return contextModel.View()
}

// renderFooter renders the footer with shortcut hints
func (m *Model) renderFooter() string {
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	switch m.panel {
	case PanelIntro:
		return hintStyle.Render("enter: Start  |  esc: Quit")
	case PanelSBOM:
		return hintStyle.Render("enter: Continue  |  esc: Back")
	case PanelMOF:
		return hintStyle.Render("enter: Continue  |  esc: Back")
	case PanelReview:
		return hintStyle.Render("enter: Start Hardening  |  esc: Back")
	case PanelHardening:
		return hintStyle.Render("Wait...  |  ctrl+c: Cancel")
	case PanelComplete:
		return hintStyle.Render("enter: Continue")
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
	sb.WriteString("Local Hardening & Compliance\n")
	sb.WriteString(strings.Repeat("─", 40) + "\n\n")

	sb.WriteString("This step performs local hardening:\n\n")

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888FF"))
	sb.WriteString(infoStyle.Render("  • SBOM Generation\n"))
	sb.WriteString("    Generate Software Bill of Materials for supply chain transparency\n")
	sb.WriteString("    Attached to artifact layers for traceability\n\n")

	sb.WriteString(infoStyle.Render("  • MOF Classification\n"))
	sb.WriteString("    Apply Model Openness Framework classification (Class I, II, or III)\n")
	sb.WriteString("    Machine-readable metadata for compliance\n\n")

	sb.WriteString(infoStyle.Render("  • Security Annotations\n"))
	sb.WriteString("    Apply security framework annotations (Sigstore, SLSA)\n")

	return sb.String()
}

// renderSBOMPanel renders the SBOM configuration panel
func (m *Model) renderSBOMPanel() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("Generate SBOM\n")
	sb.WriteString(strings.Repeat("─", 15) + "\n\n")

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888FF"))
	sb.WriteString(infoStyle.Render("Software Bill of Materials\n\n"))

	sb.WriteString("What is SBOM?\n")
	sb.WriteString("  A complete inventory of all components, libraries, and dependencies\n")
	sb.WriteString("  in your AI model and its packaging.\n\n")

	sb.WriteString("Why it matters:\n")
	sb.WriteString("  ✓ Supply chain transparency\n")
	sb.WriteString("  ✓ Vulnerability tracking\n")
	sb.WriteString("  ✓ Compliance requirements\n")
	sb.WriteString("  ✓ Attached to OCI artifact layers\n\n")

	sb.WriteString("Supported SBOM tools (mutually exclusive):\n")
	sb.WriteString(workflow.SBOMToolOptions().Bullets() + "\n\n")

	checkbox := "[x]"
	if !m.generateSBOM {
		checkbox = "[ ]"
	}
	sb.WriteString(fmt.Sprintf("  %s Generate SBOM (using %s, %s)\n", checkbox, m.sbomTool, m.sbomFormat))

	return sb.String()
}

// renderMOFPanel renders the MOF classification panel
func (m *Model) renderMOFPanel() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("Apply MOF Classification\n")
	sb.WriteString(strings.Repeat("─", 22) + "\n\n")

	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888FF"))
	sb.WriteString(infoStyle.Render("Model Openness Framework\n\n"))

	sb.WriteString("What is MOF?\n")
	sb.WriteString("  A classification system for AI models based on openness and\n")
	sb.WriteString("  transparency of training data and processes.\n\n")

	sb.WriteString("MOF Classes:\n")
	sb.WriteString("  Class I:  Fully open (model, code, data, documentation)\n")
	sb.WriteString("  Class II: Partially open (some restrictions)\n")
	sb.WriteString("  Class III: Closed/Proprietary\n\n")

	checkbox := "[x]"
	if !m.includeMOF {
		checkbox = "[ ]"
	}
	sb.WriteString(fmt.Sprintf("  %s Apply MOF classification\n", checkbox))

	return sb.String()
}

// renderReview renders the review panel
func (m *Model) renderReview() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("Review Configuration\n")
	sb.WriteString(strings.Repeat("─", 22) + "\n\n")

	sb.WriteString("Hardening Options:\n\n")

	checkbox := "[x]"
	if !m.generateSBOM {
		checkbox = "[ ]"
	}
	sb.WriteString(fmt.Sprintf("  %s Generate SBOM\n", checkbox))

	checkbox = "[x]"
	if !m.includeMOF {
		checkbox = "[ ]"
	}
	sb.WriteString(fmt.Sprintf("  %s Apply MOF Classification\n", checkbox))

	sb.WriteString("\n")

	sb.WriteString("Model:\n")
	sb.WriteString(fmt.Sprintf("  Name:    %s\n", m.modelName))
	sb.WriteString(fmt.Sprintf("  Path:    %s\n", m.modelPath))
	sb.WriteString(fmt.Sprintf("  Artifact: %s\n", m.artifactName))

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("✓ Ready to harden"))

	return sb.String()
}

// renderHardening renders the hardening progress panel
func (m *Model) renderHardening() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("Hardening & Compliance Checks\n")
	sb.WriteString(strings.Repeat("─", 30) + "\n\n")

	if m.loading {
		sb.WriteString("→ Generating SBOM...\n")
		sb.WriteString("→ Applying MOF classification...\n")
		sb.WriteString("→ Applying security annotations...\n")
	} else if m.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("✗ Error: "+m.err.Error()) + "\n")
	} else if m.completed {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("✓ Hardening complete\n"))
		if m.sbomPath != "" {
			sb.WriteString(fmt.Sprintf("  SBOM: %s\n", m.sbomPath))
		}
		if m.mofClass != "" {
			sb.WriteString(fmt.Sprintf("  MOF Class: %s\n", m.mofClass))
		}
	}

	return sb.String()
}

// renderComplete renders the completion panel
func (m *Model) renderComplete() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("✓ Step 2: Local Hardening & Compliance Complete!\n")
	sb.WriteString(strings.Repeat("─", 50) + "\n\n")

	sb.WriteString("Hardening Results:\n\n")

	if m.sbomPath != "" {
		sb.WriteString(fmt.Sprintf("  ✓ SBOM generated: %s\n", m.sbomPath))
	} else {
		sb.WriteString("  ✗ SBOM generation skipped\n")
	}

	if m.mofClass != "" {
		sb.WriteString(fmt.Sprintf("  ✓ MOF Class: %s\n", m.mofClass))
	} else {
		sb.WriteString("  ✗ MOF classification skipped\n")
	}

	if m.mofConfigPath != "" {
		sb.WriteString(fmt.Sprintf("  ✓ MOF metadata config: %s\n", m.mofConfigPath))
	} else {
		sb.WriteString("  ✗ MOF metadata config skipped\n")
	}

	sb.WriteString("  ✓ Security annotations applied\n")

	if len(m.logs) > 0 {
		sb.WriteString("\n")
		sb.WriteString("Logs:\n")
		for _, log := range m.logs {
			sb.WriteString(fmt.Sprintf("  %s\n", log))
		}
	}

	sb.WriteString("\n")
	sb.WriteString("Next: Step 3 - Sign & Verify\n")

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

// SetModelInfo sets the model information
func (m *Model) SetModelInfo(name, path, artifact string) {
	m.modelName = name
	m.modelPath = path
	m.artifactName = artifact
}

// SetOptions sets the hardening options
// SetSBOMTool selects the SBOM generator (see workflow.SBOMToolOptions) and
// output format used when hardening runs.
func (m *Model) SetSBOMTool(tool string, format workflow.SBOMFormat) {
	m.sbomTool = tool
	m.sbomFormat = format
}

func (m *Model) SetOptions(generateSBOM, includeMOF bool) {
	m.generateSBOM = generateSBOM
	m.includeMOF = includeMOF
}

// AddLog adds a log message
func (m *Model) AddLog(message string) {
	m.logs = append(m.logs, message)
	if len(m.logs) > 100 {
		m.logs = m.logs[len(m.logs)-100:]
	}
}

// SBOMPath returns the generated SBOM path
func (m *Model) SBOMPath() string {
	return m.sbomPath
}

// MOFClass returns the MOF classification
func (m *Model) MOFClass() string {
	return m.mofClass
}

// MOFConfigPath returns the generated MOF config path
func (m *Model) MOFConfigPath() string {
	return m.mofConfigPath
}

// Completed returns whether the workflow is complete
func (m *Model) Completed() bool {
	return m.completed
}

// Error returns any error that occurred
func (m *Model) Error() error {
	return m.err
}
