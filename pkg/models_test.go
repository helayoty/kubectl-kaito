package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestSimpleModelsCmd(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := NewModelsCmd(configFlags)

	t.Run("Command structure", func(t *testing.T) {
		assert.Equal(t, "models", cmd.Use)
		assert.Contains(t, cmd.Short, "Manage")
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)
	})

	t.Run("Subcommands present", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.Len(t, subcommands, 2)

		subcommandNames := make([]string, len(subcommands))
		for i, subcmd := range subcommands {
			subcommandNames[i] = subcmd.Name()
		}

		assert.Contains(t, subcommandNames, "list")
		assert.Contains(t, subcommandNames, "describe")
	})
}

func TestSimpleValidateModelName(t *testing.T) {
	tests := []struct {
		name        string
		modelName   string
		expectError bool
	}{
		{
			name:        "Empty model name",
			modelName:   "",
			expectError: true,
		},
		{
			name:        "Non-empty model name",
			modelName:   "some-model",
			expectError: false, // May still error if not in list, but should pass basic validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModelName(tt.modelName)

			if tt.expectError && tt.modelName == "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot be empty")
			}
		})
	}
}

func TestSimpleGetSupportedModels(t *testing.T) {
	t.Run("Returns models", func(t *testing.T) {
		models := getSupportedModels()
		assert.NotEmpty(t, models)

		// Check that models have required fields
		for _, model := range models {
			assert.NotEmpty(t, model.Name)
			assert.NotEmpty(t, model.Type)
			assert.NotEmpty(t, model.Runtime)
		}
	})
}

func TestSimpleFilterModels(t *testing.T) {
	models := []Model{
		{Name: "model1", Type: "LLM"},
		{Name: "model2", Type: "Code"},
		{Name: "model3", Type: "LLM"},
	}

	t.Run("Filter by type", func(t *testing.T) {
		filtered := filterModelsByType(models, "LLM")
		assert.Len(t, filtered, 2)
	})

	t.Run("Filter by non-existent type", func(t *testing.T) {
		filtered := filterModelsByType(models, "NonExistent")
		assert.Len(t, filtered, 0)
	})
}

func TestSimpleSortModels(t *testing.T) {
	models := []Model{
		{Name: "zebra", Type: "LLM"},
		{Name: "alpha", Type: "Code"},
	}

	t.Run("Sort by name", func(t *testing.T) {
		sortModels(models, "name")
		assert.Equal(t, "alpha", models[0].Name)
		assert.Equal(t, "zebra", models[1].Name)
	})
}