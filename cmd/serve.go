package cmd

import (
	"github.com/lasomethingsomething/cli-prototype/config"
	"github.com/lasomethingsomething/cli-prototype/internal/workflow"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve a model using vLLM or KServe",
	Long: `Serve your ML model for inference using vLLM or KServe runtime.

Examples:
  model-cli serve
  model-cli serve --runtime vllm --model-path ./models/phi-4-mini --host 0.0.0.0 --port 8080
  model-cli serve --runtime kserve --model-path my-model --host 0.0.0.0 --port 8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		// Get flags

		// Interactive prompts
		if err := askSelectIfEmpty(cmd, "runtime", &cfg.Runtime, "Select runtime:", "", []string{"vllm", "kserve"}); err != nil {
			return err
		}
		runtime := cfg.Runtime
		warnIfSaveFails(config.Save(cfg))

		var modelPath string
		if err := askString(cmd, "model-path", &modelPath, "Model path:", ""); err != nil {
			return err
		}

		var host string
		if err := askString(cmd, "host", &host, "Host:", ""); err != nil {
			return err
		}

		var port string
		if err := askString(cmd, "port", &port, "Port:", ""); err != nil {
			return err
		}

		// Phase 5: Large Binary Asset Optimization
		var modelSize string
		if err := askString(cmd, "model-size", &modelSize, "Model size (optional):", "Approximate model size for optimization (e.g., 14GB, 70GB)"); err != nil {
			return err
		}

		// Phase 5: Reference Skill DLC
		var loadSkills bool
		if err := askConfirm(cmd, "load-skills", &loadSkills, "Load agentic skills?", "Enable Reference Skill DLC for dynamic skill loading"); err != nil {
			return err
		}

		var skillRefs []string
		if loadSkills {
			var skillRef string
			if err := askString(cmd, "skill-ref", &skillRef, "Skill reference:", "Skill reference (e.g., my-skill:v1, agentskills.io/skill-name)"); err != nil {
				return err
			}
			skillRefs = append(skillRefs, skillRef)
		}

		// Create serve workflow
		wf, err := workflow.NewServeWorkflow(runtime)
		if err != nil {
			return err
		}
		wf.SetServeInfo(modelPath, host, port)
		wf.SetModelInfo(modelSize, skillRefs)

		return wf.Run()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().String("runtime", "", "Runtime: vllm or kserve")
	serveCmd.Flags().String("model-path", "", "Path to model files")
	serveCmd.Flags().String("host", "", "Host to bind to")
	serveCmd.Flags().String("port", "", "Port to listen on")
	serveCmd.Flags().String("model-size", "", "Model size for optimization (e.g., 14GB)")
	serveCmd.Flags().Bool("load-skills", false, "Load agentic skills")
	serveCmd.Flags().String("skill-ref", "", "Skill reference (e.g., my-skill:v1)")
}
