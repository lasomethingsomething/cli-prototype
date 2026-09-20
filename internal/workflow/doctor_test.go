package workflow

import (
	"os/exec"
	"testing"
)

// TestToolInterface verifies that all tool implementations satisfy the Tool interface
func TestToolInterface(t *testing.T) {
	tools := AllTools()
	
	for _, tool := range tools {
		// These calls should not panic if the interface is properly implemented
		_ = tool.Name()
		_ = tool.Category()
		_ = tool.IsInstalled()
		_ = tool.InstallInstructions()
		_ = tool.Description()
	}
}

// TestAllToolsReturnsNonEmptyList verifies that AllTools returns a non-empty list
func TestAllToolsReturnsNonEmptyList(t *testing.T) {
	tools := AllTools()
	if len(tools) == 0 {
		t.Error("AllTools() returned an empty list")
	}
}

// TestBrewInstallableTools verifies that BrewInstallableTools returns only brew tools
func TestBrewInstallableTools(t *testing.T) {
	brewTools := BrewInstallableTools()
	
	if len(brewTools) == 0 {
		t.Error("BrewInstallableTools() returned an empty list")
	}
	
	for _, tool := range brewTools {
		if tool.Category() != CategoryBrew {
			t.Errorf("BrewInstallableTools() returned a non-brew tool: %s (category: %s)", tool.Name(), tool.Category())
		}
	}
}

// TestCheckAll verifies that CheckAll returns a valid report
func TestCheckAll(t *testing.T) {
	report := CheckAll()
	
	if report == nil {
		t.Error("CheckAll() returned nil")
		return
	}
	
	if len(report.Results) == 0 {
		t.Error("CheckAll() returned a report with no results")
	}
	
	// Verify that all tools are checked
	checkedTools := make(map[string]bool)
	for _, result := range report.Results {
		checkedTools[result.Tool.Name()] = true
	}
	
	for _, tool := range AllTools() {
		if !checkedTools[tool.Name()] {
			t.Errorf("Tool %s was not checked", tool.Name())
		}
	}
}

// TestCheckTool verifies that CheckTool works for a known tool
func TestCheckTool(t *testing.T) {
	// Test with a known tool
	toolName := "oras"
	result, err := CheckTool(toolName)
	
	if err != nil {
		t.Errorf("CheckTool(%s) returned an error: %v", toolName, err)
		return
	}
	
	if result == nil {
		t.Errorf("CheckTool(%s) returned nil", toolName)
		return
	}
	
	if result.Tool.Name() != toolName {
		t.Errorf("CheckTool(%s) returned wrong tool: %s", toolName, result.Tool.Name())
	}
	
	// Verify IsInstalled() was called
	// We can't verify the exact value, but we can verify it didn't panic
	_ = result.Installed
}

// TestCheckToolUnknown verifies that CheckTool returns an error for unknown tools
func TestCheckToolUnknown(t *testing.T) {
	_, err := CheckTool("nonexistent-tool")
	
	if err == nil {
		t.Error("CheckTool(\"nonexistent-tool\") should have returned an error")
	}
}

// TestGetTool verifies that GetTool works for known tools
func TestGetTool(t *testing.T) {
	// Test with known tools
	knownTools := []string{"oras", "syft", "cosign", "flux", "xcode-clt", "kserve"}
	
	for _, toolName := range knownTools {
		tool, err := GetTool(toolName)
		
		if err != nil {
			t.Errorf("GetTool(%s) returned an error: %v", toolName, err)
			continue
		}
		
		if tool == nil {
			t.Errorf("GetTool(%s) returned nil", toolName)
			continue
		}
		
		if tool.Name() != toolName {
			t.Errorf("GetTool(%s) returned wrong tool: %s", toolName, tool.Name())
		}
	}
}

// TestGetToolUnknown verifies that GetTool returns an error for unknown tools
func TestGetToolUnknown(t *testing.T) {
	_, err := GetTool("nonexistent-tool")
	
	if err == nil {
		t.Error("GetTool(\"nonexistent-tool\") should have returned an error")
	}
}

// TestDoctorReportAllInstalled verifies the AllInstalled method
func TestDoctorReportAllInstalled(t *testing.T) {
	// Create a report with all tools installed
	allInstalled := &DoctorReport{
		Results: []ToolResult{
			{Tool: &orasTool{}, Installed: true, Error: nil},
			{Tool: &syftTool{}, Installed: true, Error: nil},
		},
	}
	
	if !allInstalled.AllInstalled() {
		t.Error("AllInstalled() returned false for a report with all tools installed")
	}
	
	// Create a report with one tool missing
	oneMissing := &DoctorReport{
		Results: []ToolResult{
			{Tool: &orasTool{}, Installed: true, Error: nil},
			{Tool: &syftTool{}, Installed: false, Error: nil},
		},
	}
	
	if oneMissing.AllInstalled() {
		t.Error("AllInstalled() returned true for a report with a missing tool")
	}
}

// TestDoctorReportMissingTools verifies the MissingTools method
func TestDoctorReportMissingTools(t *testing.T) {
	report := &DoctorReport{
		Results: []ToolResult{
			{Tool: &orasTool{}, Installed: true, Error: nil},
			{Tool: &syftTool{}, Installed: false, Error: nil},
			{Tool: &cosignTool{}, Installed: false, Error: nil},
		},
	}
	
	missing := report.MissingTools()
	
	if len(missing) != 2 {
		t.Errorf("MissingTools() returned %d tools, expected 2", len(missing))
	}
	
	// Check that the missing tools are syft and cosign
	missingNames := make(map[string]bool)
	for _, tool := range missing {
		missingNames[tool.Name()] = true
	}
	
	if !missingNames["syft"] || !missingNames["cosign"] {
		t.Error("MissingTools() did not return the expected missing tools")
	}
}

// TestDoctorReportMissingBrewTools verifies the MissingBrewTools method
func TestDoctorReportMissingBrewTools(t *testing.T) {
	report := &DoctorReport{
		Results: []ToolResult{
			{Tool: &orasTool{}, Installed: true, Error: nil},  // brew, installed
			{Tool: &syftTool{}, Installed: false, Error: nil}, // brew, missing
			{Tool: &xcodeCLTTool{}, Installed: false, Error: nil}, // environment, missing
			{Tool: &kserveTool{}, Installed: false, Error: nil}, // cluster, missing
		},
	}
	
	missing := report.MissingBrewTools()
	
	if len(missing) != 1 {
		t.Errorf("MissingBrewTools() returned %d tools, expected 1", len(missing))
	}
	
	if len(missing) > 0 && missing[0].Name() != "syft" {
		t.Errorf("MissingBrewTools() returned %s, expected syft", missing[0].Name())
	}
}

// TestDoctorSummary verifies the DoctorSummary method
func TestDoctorSummary(t *testing.T) {
	report := &DoctorReport{
		Results: []ToolResult{
			{Tool: &orasTool{}, Installed: true, Error: nil},  // brew, installed
			{Tool: &syftTool{}, Installed: false, Error: nil}, // brew, missing
			{Tool: &cosignTool{}, Installed: false, Error: nil}, // brew, missing
			{Tool: &xcodeCLTTool{}, Installed: false, Error: nil}, // environment, missing
			{Tool: &kserveTool{}, Installed: false, Error: nil}, // cluster, missing
		},
	}
	
	brewMissing, brewTotal, otherMissing := report.DoctorSummary()
	
	if brewMissing != 2 {
		t.Errorf("DoctorSummary() returned brewMissing=%d, expected 2", brewMissing)
	}
	
	if brewTotal != 3 {
		t.Errorf("DoctorSummary() returned brewTotal=%d, expected 3", brewTotal)
	}
	
	if otherMissing != 2 {
		t.Errorf("DoctorSummary() returned otherMissing=%d, expected 2", otherMissing)
	}
}

// TestToolCategories verifies that all tools have the correct category
func TestToolCategories(t *testing.T) {
	// Define expected categories
	expectedCategories := map[string]ToolCategory{
		"oras":         CategoryBrew,
		"syft":         CategoryBrew,
		"cosign":       CategoryBrew,
		"flux":         CategoryBrew,
		"podman":       CategoryBrew,
		"kubectl":      CategoryBrew,
		"minikube":     CategoryBrew,
		"notation":     CategoryBrew,
		"xcode-clt":    CategoryEnvironment,
		"ssh-key":      CategoryEnvironment,
		"kserve":       CategoryCluster,
		"cert-manager":  CategoryCluster,
		"metrics-server": CategoryCluster,
	}
	
	for _, tool := range AllTools() {
		expected, ok := expectedCategories[tool.Name()]
		if !ok {
			t.Errorf("Tool %s not in expected categories", tool.Name())
			continue
		}
		
		if tool.Category() != expected {
			t.Errorf("Tool %s has category %s, expected %s", tool.Name(), tool.Category(), expected)
		}
	}
}

// TestOrasToolIsInstalled verifies the oras tool's IsInstalled method
func TestOrasToolIsInstalled(t *testing.T) {
	tool := &orasTool{}
	
	// Run oras version to see if it's installed
	installed := tool.IsInstalled()
	
	// Verify that IsInstalled returns the same result as exec.Command
	cmd := exec.Command("oras", "version")
	expected := cmd.Run() == nil
	
	if installed != expected {
		t.Errorf("orasTool.IsInstalled() returned %v, expected %v", installed, expected)
	}
}

// TestToolInstallInstructions verifies that all tools have non-empty install instructions
func TestToolInstallInstructions(t *testing.T) {
	for _, tool := range AllTools() {
		instructions := tool.InstallInstructions()
		if instructions == "" {
			t.Errorf("Tool %s has empty install instructions", tool.Name())
		}
	}
}

// TestToolDescription verifies that all tools have non-empty descriptions
func TestToolDescription(t *testing.T) {
	for _, tool := range AllTools() {
		description := tool.Description()
		if description == "" {
			t.Errorf("Tool %s has empty description", tool.Name())
		}
	}
}

// TestFluxToolIsInstalled verifies the flux tool's IsInstalled method
func TestFluxToolIsInstalled(t *testing.T) {
	tool := &fluxTool{}

	// Verify that IsInstalled returns the same result as exec.LookPath
	_, err := exec.LookPath("flux")
	expected := err == nil
	installed := tool.IsInstalled()

	if installed != expected {
		t.Errorf("fluxTool.IsInstalled() returned %v, expected %v", installed, expected)
	}
}

// TestPodmanToolIsInstalled verifies the podman tool's IsInstalled method
func TestPodmanToolIsInstalled(t *testing.T) {
	tool := &podmanTool{}

	// Verify that IsInstalled returns the same result as exec.LookPath
	_, err := exec.LookPath("podman")
	expected := err == nil
	installed := tool.IsInstalled()

	if installed != expected {
		t.Errorf("podmanTool.IsInstalled() returned %v, expected %v", installed, expected)
	}
}
