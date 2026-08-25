// Package tui provides an interactive TUI for packaging AI models
// following the Shopware CLI pattern with panels, tabs, and real-time progress
package packagetui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
)

// Panel represents different views in the package workflow
type Panel int

const (
	PanelIntro Panel = iota
	PanelCollectInfo
	PanelConfigureAnnotations
	PanelReview
	PanelPackaging
	PanelComplete
)

// Model holds the state of the package TUI
type Model struct {
	// Current panel
	panel Panel

	// Dimensions
	width  int
	height int

	// Workflow
	workflow *workflow.PackageWorkflow
	registry string

	// Model info collected from user
	modelName    string
	modelPath    string
	artifactName string
	includeRAG   bool
	ragPath      string

	// Annotation conventions (CNCF AI Interoperability Profile)
	runtime       string
	accelerator   string
	cudaMin       string
	memoryMin     string
	mofClass      string
	mofComponents string

	// State
	loading   bool
	completed bool
	err       error
	logs      []string

	// Steps for the checklist
	steps       []Step
	currentStep int
}

// Step represents a step in the packaging workflow
type Step struct {
	Label  string
	State  StepState
	Detail string
}

// StepState represents the state of a step
type StepState int

const (
	StepPending StepState = iota
	StepRunning
	StepDone
	StepError
)

// New creates a new package TUI model
func New(registry string) *Model {
	m := &Model{
		panel:       PanelIntro,
		registry:    registry,
		width:       80,
		height:      24,
		logs:        make([]string, 0),
		includeRAG:  false,
		currentStep: 0,
		steps: []Step{
			{Label: "Collect model information", State: StepPending},
			{Label: "Configure annotation conventions", State: StepPending},
			{Label: "Review configuration", State: StepPending},
			{Label: "Package as OCI artifact", State: StepPending},
			{Label: "Inject annotations", State: StepPending},
			{Label: "Generate SBOM", State: StepPending},
			{Label: "Apply MOF classification", State: StepPending},
		},
	}
	return m
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	var err error
	m.workflow, err = workflow.NewPackageWorkflow(m.registry)
	if err != nil {
		m.err = err
		m.logs = append(m.logs, fmt.Sprintf("Error: %v", err))
	}
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

	// Body with two-column layout (left: info, right: actions)
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

	case "up", "k":
		m.handleUp()

	case "down", "j":
		m.handleDown()

	case "r":
		if m.panel == PanelPackaging {
			// Re-check/retry
		}
	}

	return m, nil
}

// handleEnter handles the enter key
func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	// This would be handled by the specific panels
	// For now, just advance through the workflow
	switch m.panel {
	case PanelIntro:
		m.panel = PanelCollectInfo
		m.steps[0].State = StepDone
		m.currentStep = 1

	case PanelCollectInfo:
		m.panel = PanelConfigureAnnotations
		m.steps[1].State = StepDone
		m.currentStep = 2

	case PanelConfigureAnnotations:
		m.panel = PanelReview
		m.steps[2].State = StepDone
		m.currentStep = 3

	case PanelReview:
		m.panel = PanelPackaging
		m.steps[3].State = StepRunning
		m.currentStep = 4
		return m, m.startPackaging()

	case PanelPackaging:
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
	case PanelCollectInfo:
		m.panel = PanelIntro
		m.steps[0].State = StepPending
		m.currentStep = 0

	case PanelConfigureAnnotations:
		m.panel = PanelCollectInfo
		m.steps[1].State = StepPending
		m.currentStep = 1

	case PanelReview:
		m.panel = PanelConfigureAnnotations
		m.steps[2].State = StepPending
		m.currentStep = 2

	case PanelPackaging:
		// Can't go back from packaging
	}

	return m, nil
}

// handleUp handles up/down navigation
func (m *Model) handleUp() {
	// Panel-specific handling
}

// handleDown handles down navigation
func (m *Model) handleDown() {
	// Panel-specific handling
}

// startPackaging starts the packaging process
func (m *Model) startPackaging() tea.Cmd {
	m.loading = true
	m.steps[3].State = StepRunning
	m.steps[3].Detail = "Starting packaging..."

	return func() tea.Msg {
		// Set up the workflow with collected info
		m.workflow.SetPackageInfo(m.modelName, m.modelPath, m.artifactName, "", m.includeRAG, m.ragPath)

		// Set annotations
		annotations := workflow.NewAnnotationSet()
		if m.runtime != "" {
			annotations.Runtime = m.runtime
		}
		if m.accelerator != "" {
			annotations.Accelerator = m.accelerator
		}
		if m.cudaMin != "" {
			annotations.CUDAMin = m.cudaMin
		}
		if m.memoryMin != "" {
			annotations.MemoryMin = m.memoryMin
		}
		if m.mofClass != "" {
			annotations.MOFClass = m.mofClass
		}
		if m.mofComponents != "" {
			annotations.MOFComponents = m.mofComponents
		}
		m.workflow.SetAnnotations(annotations)

		// Run the workflow
		err := m.workflow.Run()

		m.loading = false
		m.completed = true
		m.err = err

		if err != nil {
			m.steps[3].State = StepError
			m.steps[3].Detail = err.Error()
			m.logs = append(m.logs, fmt.Sprintf("Error: %v", err))
		} else {
			m.steps[3].State = StepDone
			m.steps[3].Detail = "Packaging completed"
			m.steps[4].State = StepDone
			m.steps[5].State = StepDone
			m.steps[6].State = StepDone
			m.logs = append(m.logs, "Model packaged successfully as OCI artifact")
			m.logs = append(m.logs, fmt.Sprintf("Artifact: %s", m.artifactName))
		}

		return nil
	}
}

// renderHeader renders the header with title and context
func (m *Model) renderHeader() string {
	title := "Step 1: Develop & Package"
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Bold(true)

	context := fmt.Sprintf("Registry: %s", m.registry)
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888FF"))

	return titleStyle.Render(title) + "  " + contextStyle.Render(context)
}

// renderStatus renders the status strip
func (m *Model) renderStatus() string {
	if m.loading {
		return renderStatusStrip("RUNNING", "Packaging model...", lipgloss.Color("#55AAFF"))
	}
	if m.err != nil {
		return renderStatusStrip("ERROR", m.err.Error(), lipgloss.Color("#FF5555"))
	}
	if m.completed {
		return renderStatusStrip("COMPLETE", "Ready for next step", lipgloss.Color("#00FF88"))
	}
	return ""
}

// renderBody renders the main body content
func (m *Model) renderBody() string {
	switch m.panel {
	case PanelIntro:
		return m.renderIntro()
	case PanelCollectInfo:
		return m.renderCollectInfo()
	case PanelConfigureAnnotations:
		return m.renderConfigureAnnotations()
	case PanelReview:
		return m.renderReview()
	case PanelPackaging:
		return m.renderPackaging()
	case PanelComplete:
		return m.renderComplete()
	}
	return ""
}

// renderFooter renders the footer with shortcut hints
func (m *Model) renderFooter() string {
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	switch m.panel {
	case PanelIntro:
		return hintStyle.Render("enter: Start  |  esc: Quit")
	case PanelCollectInfo:
		return hintStyle.Render("enter: Continue  |  esc: Back")
	case PanelConfigureAnnotations:
		return hintStyle.Render("enter: Continue  |  esc: Back")
	case PanelReview:
		return hintStyle.Render("enter: Package  |  esc: Back")
	case PanelPackaging:
		return hintStyle.Render("Wait...  |  ctrl+c: Cancel")
	case PanelComplete:
		return hintStyle.Render("enter: Continue  |  ↑/↓: Scroll")
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
	sb.WriteString("Develop & Package AI Asset\n")
	sb.WriteString(strings.Repeat("─", 40) + "\n\n")

	sb.WriteString("This step packages your AI asset locally:\n\n")
	sb.WriteString("  ✓ Model files\n")
	sb.WriteString("  ✓ RAG context (optional)\n")
	sb.WriteString("  ✓ Agentic skills (optional)\n\n")

	sb.WriteString("Automatic injection:\n\n")
	sb.WriteString("  ✓ CNCF AI Interoperability Profile annotations\n")
	sb.WriteString("  ✓ OCI artifact manifest\n")
	sb.WriteString("  ✓ agentskills.io standard format for skills\n")

	return sb.String()
}

// renderCollectInfo renders the model info collection panel
func (m *Model) renderCollectInfo() string {
	left := m.renderStepList()
	right := m.renderModelInfoForm()

	// Two-column layout
	return renderTwoColumn(m.width, left, right)
}

// renderConfigureAnnotations renders the annotations configuration panel
func (m *Model) renderConfigureAnnotations() string {
	left := m.renderStepList()
	right := m.renderAnnotationsForm()

	return renderTwoColumn(m.width, left, right)
}

// renderReview renders the review panel
func (m *Model) renderReview() string {
	left := m.renderStepList()
	right := m.renderReviewSummary()

	return renderTwoColumn(m.width, left, right)
}

// renderPackaging renders the packaging progress panel
func (m *Model) renderPackaging() string {
	left := m.renderStepList()
	right := m.renderPackagingProgress()

	return renderTwoColumn(m.width, left, right)
}

// renderComplete renders the completion panel
func (m *Model) renderComplete() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("✓ Step 1: Develop & Package Complete!\n")
	sb.WriteString(strings.Repeat("─", 40) + "\n\n")

	sb.WriteString("Packaged AI Asset:\n\n")
	sb.WriteString(fmt.Sprintf("  Model:         %s\n", m.modelName))
	sb.WriteString(fmt.Sprintf("  Artifact:      %s\n", m.artifactName))
	sb.WriteString(fmt.Sprintf("  Path:          %s\n", m.modelPath))

	if m.includeRAG {
		sb.WriteString(fmt.Sprintf("  RAG Context:   %s\n", m.ragPath))
	}

	sb.WriteString("\n")
	sb.WriteString("Annotations Injected:\n\n")
	sb.WriteString(fmt.Sprintf("  Runtime:       %s\n", m.runtime))
	sb.WriteString(fmt.Sprintf("  Accelerator:   %s\n", m.accelerator))
	sb.WriteString(fmt.Sprintf("  CUDA Min:     %s\n", m.cudaMin))
	sb.WriteString(fmt.Sprintf("  Memory Min:   %s\n", m.memoryMin))
	sb.WriteString(fmt.Sprintf("  MOF Class:    %s\n", m.mofClass))

	if len(m.logs) > 0 {
		sb.WriteString("\n")
		sb.WriteString("Logs:\n")
		for _, log := range m.logs {
			sb.WriteString(fmt.Sprintf("  %s\n", log))
		}
	}

	sb.WriteString("\n")
	sb.WriteString("Next: Step 2 - Harden & Sign\n")

	return sb.String()
}

// renderStepList renders the step checklist
func (m *Model) renderStepList() string {
	var sb strings.Builder

	sb.WriteString("Steps:\n\n")

	for i, step := range m.steps {
		cursor := "  "
		if i == m.currentStep {
			cursor = "> "
		}

		symbol := "·"
		switch step.State {
		case StepDone:
			symbol = "✓"
		case StepRunning:
			symbol = "⠋"
		case StepError:
			symbol = "✗"
		}

		labelStyle := lipgloss.NewStyle()
		if i == m.currentStep {
			labelStyle = labelStyle.Bold(true)
		}

		sb.WriteString(cursor + symbol + " " + labelStyle.Render(step.Label) + "\n")
		if step.Detail != "" {
			detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			sb.WriteString(detailStyle.Render("   "+step.Detail) + "\n")
		}
	}

	return sb.String()
}

// renderModelInfoForm renders the model info form
func (m *Model) renderModelInfoForm() string {
	var sb strings.Builder

	sb.WriteString("Model Information\n")
	sb.WriteString(strings.Repeat("─", 25) + "\n\n")

	sb.WriteString(fmt.Sprintf("Model Name:    %s\n", m.modelName))
	sb.WriteString(fmt.Sprintf("Model Path:    %s\n", m.modelPath))
	sb.WriteString(fmt.Sprintf("Artifact:      %s\n", m.artifactName))
	sb.WriteString(fmt.Sprintf("Include RAG:   %v\n", m.includeRAG))

	if m.includeRAG && m.ragPath != "" {
		sb.WriteString(fmt.Sprintf("RAG Path:      %s\n", m.ragPath))
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Press Enter to continue"))

	return sb.String()
}

// renderAnnotationsForm renders the annotations form
func (m *Model) renderAnnotationsForm() string {
	var sb strings.Builder

	sb.WriteString("Annotation Conventions\n")
	sb.WriteString(strings.Repeat("─", 28) + "\n\n")

	sb.WriteString("Runtime Requirements:\n")
	sb.WriteString(fmt.Sprintf("  Runtime:       %s\n", m.runtime))
	sb.WriteString(fmt.Sprintf("  Accelerator:   %s\n", m.accelerator))
	sb.WriteString(fmt.Sprintf("  CUDA Min:     %s\n", m.cudaMin))
	sb.WriteString(fmt.Sprintf("  Memory Min:   %s\n", m.memoryMin))

	sb.WriteString("\n")
	sb.WriteString("MOF Classification:\n")
	sb.WriteString(fmt.Sprintf("  MOF Class:    %s\n", m.mofClass))
	sb.WriteString(fmt.Sprintf("  Components:   %s\n", m.mofComponents))

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Press Enter to continue"))

	return sb.String()
}

// renderReviewSummary renders the review summary
func (m *Model) renderReviewSummary() string {
	var sb strings.Builder

	sb.WriteString("Review Configuration\n")
	sb.WriteString(strings.Repeat("─", 22) + "\n\n")

	sb.WriteString("Model:\n")
	sb.WriteString(fmt.Sprintf("  Name:    %s\n", m.modelName))
	sb.WriteString(fmt.Sprintf("  Path:    %s\n", m.modelPath))
	sb.WriteString(fmt.Sprintf("  Artifact: %s\n", m.artifactName))

	sb.WriteString("\n")
	sb.WriteString("Annotations:\n")
	sb.WriteString(fmt.Sprintf("  Runtime:    %s\n", m.runtime))
	sb.WriteString(fmt.Sprintf("  Accelerator: %s\n", m.accelerator))

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("✓ Ready to package"))

	sb.WriteString("\n\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Press Enter to start packaging"))

	return sb.String()
}

// renderPackagingProgress renders the packaging progress
func (m *Model) renderPackagingProgress() string {
	var sb strings.Builder

	sb.WriteString("Packaging...\n")
	sb.WriteString(strings.Repeat("─", 12) + "\n\n")

	if m.loading {
		sb.WriteString("→ Creating OCI artifact manifest...\n")
		sb.WriteString("→ Injecting annotations...\n")
		if m.includeRAG {
			sb.WriteString("→ Adding RAG context...\n")
		}
		sb.WriteString("→ Packaging model files...\n")
		sb.WriteString("→ Generating SBOM...\n")
		sb.WriteString("→ Applying MOF classification...\n")
	} else if m.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("✗ Error: "+m.err.Error()) + "\n")
	} else if m.completed {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF88")).Render("✓ Packaging complete\n"))
	}

	return sb.String()
}

// renderTwoColumn renders content in a two-column layout
func renderTwoColumn(width int, left, right string) string {
	// Split width roughly in half
	leftWidth := width * 40 / 100
	if leftWidth < 20 {
		leftWidth = 20
	}
	if leftWidth > width-20 {
		leftWidth = width - 20
	}

	// Pad and truncate
	left = lipgloss.NewStyle().Width(leftWidth).Render(left)
	right = lipgloss.NewStyle().Width(width - leftWidth - 2).Render(right)

	return left + "  " + right
}

// SetModelInfo sets the model information
func (m *Model) SetModelInfo(name, path, artifact string, includeRAG bool, ragPath string) {
	m.modelName = name
	m.modelPath = path
	m.artifactName = artifact
	m.includeRAG = includeRAG
	m.ragPath = ragPath
}

// SetAnnotations sets the annotation conventions
func (m *Model) SetAnnotations(runtime, accelerator, cudaMin, memoryMin, mofClass, mofComponents string) {
	m.runtime = runtime
	m.accelerator = accelerator
	m.cudaMin = cudaMin
	m.memoryMin = memoryMin
	m.mofClass = mofClass
	m.mofComponents = mofComponents
}

// AddLog adds a log message
func (m *Model) AddLog(message string) {
	m.logs = append(m.logs, message)
	if len(m.logs) > 100 {
		m.logs = m.logs[len(m.logs)-100:]
	}
}
