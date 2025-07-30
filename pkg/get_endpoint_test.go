package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestSimpleGetEndpointCmd(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := NewGetEndpointCmd(configFlags)

	t.Run("Command structure", func(t *testing.T) {
		assert.Equal(t, "get-endpoint", cmd.Use)
		assert.Contains(t, cmd.Short, "Get")
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("Required flags present", func(t *testing.T) {
		flags := cmd.Flags()

		workspaceFlag := flags.Lookup("workspace-name")
		assert.NotNil(t, workspaceFlag)

		formatFlag := flags.Lookup("format")
		assert.NotNil(t, formatFlag)

		externalFlag := flags.Lookup("external")
		assert.NotNil(t, externalFlag)
	})
}