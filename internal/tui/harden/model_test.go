package hardentui

import "testing"

func TestInitCreatesWorkflow(t *testing.T) {
	m := New("ghcr.io/example")

	if cmd := m.Init(); cmd != nil {
		t.Fatalf("Init() returned unexpected command: %v", cmd)
	}

	if m.workflow == nil {
		t.Fatal("Init() did not create a workflow")
	}

	if m.err != nil {
		t.Fatalf("Init() set unexpected error: %v", m.err)
	}
}
