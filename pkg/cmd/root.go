package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRootCmd creates the root command for kubectl-kaito
func NewRootCmd(configFlags *genericclioptions.ConfigFlags, isPlugin bool) *cobra.Command {
	var cmdName = "kaito"
	if isPlugin {
		cmdName = "kubectl kaito"
	}

	cmd := &cobra.Command{
		Use:   cmdName,
		Short: "Kubernetes AI Toolchain Operator (Kaito) CLI",
		Long: `kubectl-kaito is a command-line tool for managing AI/ML model inference 
and fine-tuning workloads using the Kubernetes AI Toolchain Operator (Kaito).

This plugin simplifies the deployment, management, and monitoring of AI models
in Kubernetes clusters through Kaito workspaces.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: fmt.Sprintf(`  # Check plugin version
  %s version

  # Deploy a model for inference
  %s deploy --name workspace-llama-3 --model llama-2-7b --gpus 1 --preset chat

  # Fine-tune a model  
  %s tune --name workspace-llama-3-tune --model llama-2-7b --dataset gs://teamA-ds --preset qlora

  # Check workspace status
  %s status workspace/workspace-llama-3

  # List available presets
  %s preset list

  # Get logs from a workspace
  %s logs workspace-llama-3`, cmdName, cmdName, cmdName, cmdName, cmdName, cmdName),
	}

	// Add global flags
	configFlags.AddFlags(cmd.PersistentFlags())

	// Add subcommands
	cmd.AddCommand(NewDeployCmd(configFlags))
	cmd.AddCommand(NewTuneCmd(configFlags))
	cmd.AddCommand(NewStatusCmd(configFlags))
	cmd.AddCommand(NewLogsCmd(configFlags))
	cmd.AddCommand(NewPresetCmd(configFlags))
	cmd.AddCommand(NewDeleteCmd(configFlags))
	cmd.AddCommand(NewVersionCmd(configFlags))

	return cmd
}
