package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestSimpleDeployCmd(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := NewDeployCmd(configFlags)

	t.Run("Command structure", func(t *testing.T) {
		assert.Equal(t, "deploy", cmd.Use)
		assert.Contains(t, cmd.Short, "Deploy")
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("Required flags present", func(t *testing.T) {
		flags := cmd.Flags()

		workspaceFlag := flags.Lookup("workspace-name")
		assert.NotNil(t, workspaceFlag)

		modelFlag := flags.Lookup("model")
		assert.NotNil(t, modelFlag)
	})
}

func TestSimpleDeployOptionsValidation(t *testing.T) {
	tests := []struct {
		name        string
		options     DeployOptions
		expectError bool
	}{
		{
			name: "Valid options",
			options: DeployOptions{
				WorkspaceName: "test-workspace",
				Model:         "phi-3.5-mini-instruct",
			},
			expectError: false,
		},
		{
			name: "Missing workspace name",
			options: DeployOptions{
				Model: "phi-3.5-mini-instruct",
			},
			expectError: true,
		},
		{
			name: "Missing model",
			options: DeployOptions{
				WorkspaceName: "test-workspace",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}