package cmd

import (
	"testing"

	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

// newCUDAMinTestCmd builds a throwaway command carrying only the --cuda-min
// flag, parsed from args.
func newCUDAMinTestCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	c := &cobra.Command{Use: "t", Run: func(*cobra.Command, []string) {}}
	c.Flags().String("cuda-min", "", "")
	c.SetArgs(args)
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	return c
}

// Issue #103: the CUDA minimum must only be set for NVIDIA accelerators.
func TestAskCUDAMin(t *testing.T) {
	forceNonInteractive(t)

	tests := []struct {
		name        string
		accelerator string
		args        []string
		want        string
	}{
		{"nvidia keeps the default", "nvidia-gpu", nil, "12.1"},
		{"nvidia with explicit flag", "nvidia-gpu", []string{"--cuda-min", "11.8"}, "11.8"},
		{"cpu drops the default", "cpu", nil, ""},
		{"amd-gpu drops the default", "amd-gpu", nil, ""},
		{"intel-gpu drops the default", "intel-gpu", nil, ""},
		{"none drops the default", "none", nil, ""},
		{"cpu with explicit flag is respected", "cpu", []string{"--cuda-min", "12.4"}, "12.4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCUDAMinTestCmd(t, tt.args...)
			annotations := workflow.NewAnnotationSet()
			annotations.Accelerator = tt.accelerator
			if err := askCUDAMin(c, annotations); err != nil {
				t.Fatalf("askCUDAMin() error = %v", err)
			}
			if annotations.CUDAMin != tt.want {
				t.Errorf("CUDAMin = %q, want %q", annotations.CUDAMin, tt.want)
			}
			m := annotations.ToMap()
			if got, ok := m[workflow.AnnotationCUDAVersionMin]; ok != (tt.want != "") || got != tt.want {
				t.Errorf("annotation %s = %q (present=%v), want %q", workflow.AnnotationCUDAVersionMin, got, ok, tt.want)
			}
		})
	}
}
