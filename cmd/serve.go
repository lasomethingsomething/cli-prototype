package cmd

import (
	"github.com/charmbracelet/huh"
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
		runtimeFlag, _ := cmd.Flags().GetString("runtime")
		modelPathFlag, _ := cmd.Flags().GetString("model-path")
		hostFlag, _ := cmd.Flags().GetString("host")
		portFlag, _ := cmd.Flags().GetString("port")
		modelSizeFlag, _ := cmd.Flags().GetString("model-size")
		loadSkillsFlag, _ := cmd.Flags().GetBool("load-skills")
		skillRefFlag, _ := cmd.Flags().GetString("skill-ref")

		// Interactive prompts
		var runtime string
		if runtimeFlag != "" {
			runtime = runtimeFlag
		} else if cfg.Runtime == "" {
			if err := huh.NewSelect[string]().
				Title("Select runtime:").
				Options(huh.NewOptions("vllm", "kserve")...).
				Value(&runtime).
				Run(); err != nil {
				return err
			}
			cfg.Runtime = runtime
			warnIfSaveFails(config.Save(cfg))
		} else {
			runtime = cfg.Runtime
		}

		var modelPath string
		if modelPathFlag != "" {
			modelPath = modelPathFlag
		} else {
			if err := huh.NewInput().
				Title("Model path:").
				Value(&modelPath).
				Run(); err != nil {
				return err
			}
		}

		var host string
		if hostFlag != "" {
			host = hostFlag
		} else {
			if err := huh.NewInput().
				Title("Host:").
				Value(&host).
				Run(); err != nil {
				return err
			}
		}

		var port string
		if portFlag != "" {
			port = portFlag
		} else {
			if err := huh.NewInput().
				Title("Port:").
				Value(&port).
				Run(); err != nil {
				return err
			}
		}

		// Phase 5: Large Binary Asset Optimization
		var modelSize string
		if modelSizeFlag != "" {
			modelSize = modelSizeFlag
		} else {
			if err := huh.NewInput().
				Title("Model size (optional):").
				Description("Approximate model size for optimization (e.g., 14GB, 70GB)").
				Value(&modelSize).
				Run(); err != nil {
				return err
			}
		}

		// Phase 5: Reference Skill DLC
		var loadSkills bool
		if loadSkillsFlag {
			loadSkills = true
		} else {
			if err := huh.NewConfirm().
				Title("Load agentic skills?").
				Description("Enable Reference Skill DLC for dynamic skill loading").
				Value(&loadSkills).
				Run(); err != nil {
				return err
			}
		}

		var skillRefs []string
		if loadSkills {
			var skillRef string
			if skillRefFlag != "" {
				skillRef = skillRefFlag
			} else {
				if err := huh.NewInput().
					Title("Skill reference:").
					Description("Skill reference (e.g., my-skill:v1, agentskills.io/skill-name)").
					Value(&skillRef).
					Run(); err != nil {
					return err
				}
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
