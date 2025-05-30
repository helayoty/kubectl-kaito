package cmd

import (
	"strings"
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestNewRootCmd(t *testing.T) {
	tests := []struct {
		name     string
		isPlugin bool
		expected string
	}{
		{
			name:     "as kubectl plugin",
			isPlugin: true,
			expected: "kubectl kaito",
		},
		{
			name:     "as standalone binary",
			isPlugin: false,
			expected: "kaito",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFlags := genericclioptions.NewConfigFlags(true)
			cmd := NewRootCmd(configFlags, tt.isPlugin)

			if cmd.Use != tt.expected {
				t.Errorf("NewRootCmd() Use = %v, expected %v", cmd.Use, tt.expected)
			}

			// Test that the command has the expected subcommands
			expectedSubcommands := []string{"deploy", "tune", "status", "logs", "preset", "delete"}
			actualSubcommands := []string{}
			for _, subcmd := range cmd.Commands() {
				actualSubcommands = append(actualSubcommands, subcmd.Name())
			}

			for _, expected := range expectedSubcommands {
				found := false
				for _, actual := range actualSubcommands {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected subcommand %s not found in: %v", expected, actualSubcommands)
				}
			}
		})
	}
}

func TestRootCommandStructure(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := NewRootCmd(configFlags, false)

	// Test basic command properties
	if cmd.Short == "" {
		t.Error("Root command should have a short description")
	}

	if cmd.Long == "" {
		t.Error("Root command should have a long description")
	}

	if cmd.Example == "" {
		t.Error("Root command should have examples")
	}

	// Test that examples contain expected commands
	expectedInExamples := []string{"deploy", "tune", "status", "preset", "logs"}
	for _, expected := range expectedInExamples {
		if !strings.Contains(cmd.Example, expected) {
			t.Errorf("Expected %s to be mentioned in examples", expected)
		}
	}

	// Test that SilenceUsage and SilenceErrors are set
	if !cmd.SilenceUsage {
		t.Error("Root command should have SilenceUsage set to true")
	}

	if !cmd.SilenceErrors {
		t.Error("Root command should have SilenceErrors set to true")
	}
}
