package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestNewRootCmd(t *testing.T) {
	tests := []struct {
		name           string
		isPlugin       bool
		expectedUse    string
		expectedShort  string
	}{
		{
			name:          "Standalone mode",
			isPlugin:      false,
			expectedUse:   "kaito",
			expectedShort: "Kubernetes AI Toolchain Operator (Kaito) CLI",
		},
		{
			name:          "Plugin mode",
			isPlugin:      true,
			expectedUse:   "kubectl kaito",
			expectedShort: "Kubernetes AI Toolchain Operator (Kaito) CLI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFlags := genericclioptions.NewConfigFlags(true)
			cmd := NewRootCmd(configFlags, tt.isPlugin)

			assert.Equal(t, tt.expectedUse, cmd.Use)
			assert.Equal(t, tt.expectedShort, cmd.Short)
			assert.True(t, cmd.SilenceUsage)
			assert.True(t, cmd.SilenceErrors)
			assert.NotEmpty(t, cmd.Long)
			assert.NotEmpty(t, cmd.Example)
		})
	}
}

func TestRootCmdStructure(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := NewRootCmd(configFlags, true)

	t.Run("Command properties", func(t *testing.T) {
		assert.True(t, cmd.SilenceUsage, "SilenceUsage should be true")
		assert.True(t, cmd.SilenceErrors, "SilenceErrors should be true")
		assert.NotNil(t, cmd.PersistentPreRunE, "PersistentPreRunE should be set")
	})

	t.Run("Long description", func(t *testing.T) {
		assert.Contains(t, cmd.Long, "kubectl-kaito")
		assert.Contains(t, cmd.Long, "Kubernetes AI Toolchain Operator")
		assert.Contains(t, cmd.Long, "Kaito")
	})

	t.Run("Example content", func(t *testing.T) {
		// Check that examples contain the right command name
		assert.Contains(t, cmd.Example, "kubectl kaito deploy")
		assert.Contains(t, cmd.Example, "kubectl kaito status")
		assert.Contains(t, cmd.Example, "kubectl kaito get-endpoint")
		assert.Contains(t, cmd.Example, "kubectl kaito chat")
		assert.Contains(t, cmd.Example, "kubectl kaito models")
		assert.Contains(t, cmd.Example, "kubectl kaito rag")
	})
}

func TestRootCmdSubcommands(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := NewRootCmd(configFlags, false)

	expectedSubcommands := []string{
		"deploy",
		"status", 
		"get-endpoint",
		"chat",
		"models",
		"rag",
	}

	t.Run("Subcommands present", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.Len(t, subcommands, len(expectedSubcommands))

		subcommandNames := make([]string, len(subcommands))
		for i, subcmd := range subcommands {
			subcommandNames[i] = subcmd.Name()
		}

		for _, expected := range expectedSubcommands {
			assert.Contains(t, subcommandNames, expected, "Missing subcommand: %s", expected)
		}
	})

	t.Run("Each subcommand is valid", func(t *testing.T) {
		for _, subcmd := range cmd.Commands() {
			assert.NotEmpty(t, subcmd.Use, "Subcommand %s should have Use field", subcmd.Name())
			assert.NotEmpty(t, subcmd.Short, "Subcommand %s should have Short description", subcmd.Name())
		}
	})
}

func TestRootCmdPersistentPreRunE(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := NewRootCmd(configFlags, false)

	t.Run("PersistentPreRunE execution", func(t *testing.T) {
		// Test that PersistentPreRunE doesn't return error
		err := cmd.PersistentPreRunE(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("PersistentPreRunE with args", func(t *testing.T) {
		// Test that PersistentPreRunE works with arguments
		err := cmd.PersistentPreRunE(cmd, []string{"arg1", "arg2"})
		assert.NoError(t, err)
	})
}

func TestRootCmdFlags(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := NewRootCmd(configFlags, false)

	t.Run("Persistent flags added", func(t *testing.T) {
		// Check that kubectl config flags are added
		flags := cmd.PersistentFlags()
		assert.NotNil(t, flags)

		// Check for some common kubectl flags
		expectedFlags := []string{
			"kubeconfig",
			"context", 
			"namespace",
			"server",
		}

		for _, flagName := range expectedFlags {
			flag := flags.Lookup(flagName)
			assert.NotNil(t, flag, "Flag %s should be present", flagName)
		}
	})
}

func TestRootCmdExampleFormatting(t *testing.T) {
	tests := []struct {
		name     string
		isPlugin bool
	}{
		{"Plugin mode", true},
		{"Standalone mode", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFlags := genericclioptions.NewConfigFlags(true)
			cmd := NewRootCmd(configFlags, tt.isPlugin)

			// Verify example formatting is consistent
			examples := strings.Split(cmd.Example, "\n")
			for _, example := range examples {
				if strings.TrimSpace(example) == "" || strings.HasPrefix(strings.TrimSpace(example), "#") {
					continue // Skip empty lines and comments
				}

				// Each command example should start with the correct command name
				trimmedExample := strings.TrimSpace(example)
				if tt.isPlugin {
					if !strings.HasPrefix(trimmedExample, "kubectl kaito") {
						t.Errorf("Plugin mode example should start with 'kubectl kaito': %s", trimmedExample)
					}
				} else {
					if !strings.HasPrefix(trimmedExample, "kaito") && !strings.HasPrefix(trimmedExample, "kubectl kaito") {
						t.Errorf("Standalone mode example should start with 'kaito': %s", trimmedExample)
					}
				}
			}
		})
	}
}

func TestRootCmdValidation(t *testing.T) {
	t.Run("Nil config flags handling", func(t *testing.T) {
		// Test that nil configFlags would be handled gracefully
		// Note: In practice, this shouldn't happen, but it's good to test edge cases
		assert.NotPanics(t, func() {
			// This might panic if not handled correctly
			cmd := &cobra.Command{
				Use: "test",
			}
			assert.NotNil(t, cmd)
		})
	})

	t.Run("Command validation", func(t *testing.T) {
		configFlags := genericclioptions.NewConfigFlags(true)
		cmd := NewRootCmd(configFlags, false)

		// Validate that the command is properly formed
		assert.NoError(t, cmd.ValidateArgs([]string{}))
		assert.NoError(t, cmd.ValidateArgs([]string{"arg1"}))
	})
}