# TUI Guide

Model CLI uses [Huh](https://github.com/charmbracelet/huh) for prompts,
[Bubble Tea](https://github.com/charmbracelet/bubbletea) for the interactive
context panel, and [Lip Gloss](https://github.com/charmbracelet/lipgloss) for
terminal styling. The interactive Quick Test Drive is in the repository README;
this page describes the terminal experience itself.

## Wizard

Start the guided flow with:

```bash
model-cli wizard
```

The wizard follows these seven phases:

1. Develop & Package
2. Local Hardening & Compliance
3. Supply Chain Check
4. Manifest-Level Validation
5. GitOps Admission & Policy Enforcement
6. Infrastructure & Resource Orchestration
7. Runtime Execution & Optimization

At context-panel pauses, the line above the panel states exactly what Enter
will do next. For example, after model details, Enter starts local packaging;
after packaging, Enter starts SBOM generation and MOF classification.

The wizard executes local packaging, hardening, local compliance, optional OCI
publication, and manifest retrieval. Signing and verification are demonstrated
in the wizard; use standalone `sign` and `verify` commands to execute them.
When no Kubernetes cluster is connected, GitOps, infrastructure, and runtime
phases are guided simulations.

## Context Panel

The panel appears at phase transitions and has six tabs:

| Tab | Contents |
|-----|----------|
| Progress | Current phase, progress bar, and completed, current, pending, or skipped phases |
| Config | Selected registry, GitOps, signer, and runtime values |
| Model | Model name, source path, artifact reference, and completed operation status |
| Logs | Recent workflow events |
| Env | Model CLI version, terminal detection, and Go version information |
| Help | Keyboard controls |

Use `Tab` and `Shift+Tab` to change tabs, or `1` through `6` to select a tab.
Press Enter to continue the displayed workflow action. Press Esc to cancel the
wizard.

The progress bar reflects the seven-phase journey. A check mark means the phase
completed, an arrow marks the current phase, and a dot is pending. The panel
also supports a dash for phases explicitly marked as skipped.

## Prompt Behavior

Values supplied through command-line flags take precedence. Otherwise, the CLI
uses a saved value from `~/.model-cli.yaml` when one exists, or prompts for a
choice. An explicit empty flag value is preserved rather than prompting again.

Prompts are disabled in non-interactive shells, with `--non-interactive`, or
when `MODEL_CLI_NO_INTERACTIVE=1`. Required missing values then produce an
error naming the flag to provide. The `wizard` command requires an interactive
terminal.

Tool menus show supported choices and mark the current recommendation. Some
choices are mutually exclusive providers, such as ORAS versus ModelPack; the
serving-topology demonstration additionally shows direct vLLM, KServe, and
KServe-managed vLLM.

## Styling And Accessibility

The wizard uses white bold headers, blue step labels, green success messages,
light-blue information, and orange warnings. Colors supplement text labels and
symbols rather than carrying meaning by themselves.

Wizard styles are defined in `cmd/wizard.go`. Context-panel rendering and
controls are implemented in `internal/tui/context_tui.go`.

Use a terminal with at least 100 columns and 24-bit color support for the most
readable layout. VS Code, iTerm2, and Alacritty meet this requirement.