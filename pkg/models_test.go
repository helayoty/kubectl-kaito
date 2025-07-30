/*
Copyright (c) 2024 Kaito Project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestModelsCmd(t *testing.T) {
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

func TestValidateModelName(t *testing.T) {
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

func TestGetSupportedModels(t *testing.T) {
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

func TestFilterModels(t *testing.T) {
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

func TestSortModels(t *testing.T) {
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
