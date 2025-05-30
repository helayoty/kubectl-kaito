package cmd

import (
	"strings"
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestNewPresetCmd(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := NewPresetCmd(configFlags)

	// Test basic command properties
	if cmd.Use != "preset" {
		t.Errorf("Expected Use to be 'preset', got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Preset command should have a short description")
	}

	if cmd.Long == "" {
		t.Error("Preset command should have a long description")
	}

	// Test that it has a list subcommand
	found := false
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "list" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Preset command should have a 'list' subcommand")
	}
}

func TestNewPresetListCmd(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := NewPresetListCmd(configFlags)

	// Test basic command properties
	if cmd.Use != "list" {
		t.Errorf("Expected Use to be 'list', got %s", cmd.Use)
	}

	// Test that model flag exists
	flag := cmd.Flags().Lookup("model")
	if flag == nil {
		t.Error("Expected --model flag to exist")
	}
}

func TestPresetOptionsRunList(t *testing.T) {
	tests := []struct {
		name        string
		modelType   string
		expectError bool
	}{
		{
			name:        "all presets",
			modelType:   "",
			expectError: false,
		},
		{
			name:        "llama models",
			modelType:   "llama",
			expectError: false,
		},
		{
			name:        "falcon models",
			modelType:   "falcon",
			expectError: false,
		},
		{
			name:        "phi models",
			modelType:   "phi",
			expectError: false,
		},
		{
			name:        "mistral models",
			modelType:   "mistral",
			expectError: false,
		},
		{
			name:        "tuning presets",
			modelType:   "tuning",
			expectError: false,
		},
		{
			name:        "invalid model family",
			modelType:   "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFlags := genericclioptions.NewConfigFlags(true)
			o := &PresetOptions{
				configFlags: configFlags,
				ModelType:   tt.modelType,
			}

			err := o.RunList()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestKnownPresets(t *testing.T) {
	// Test that known presets are properly defined
	expectedFamilies := []string{"llama", "falcon", "phi", "mistral"}

	for _, family := range expectedFamilies {
		presets, exists := knownPresets[family]
		if !exists {
			t.Errorf("Expected family %s to exist in knownPresets", family)
			continue
		}

		if len(presets) == 0 {
			t.Errorf("Family %s should have at least one preset", family)
		}

		// Test that presets have the family name in them
		for _, preset := range presets {
			if !strings.Contains(preset, family) {
				// Allow some exceptions like phi-2
				if !(family == "phi" && preset == "phi-2") {
					t.Errorf("Preset %s should contain family name %s", preset, family)
				}
			}
		}
	}
}

func TestTuningPresets(t *testing.T) {
	expectedTuningPresets := []string{"qlora", "lora"}

	if len(tuningPresets) != len(expectedTuningPresets) {
		t.Errorf("Expected %d tuning presets, got %d", len(expectedTuningPresets), len(tuningPresets))
	}

	for _, expected := range expectedTuningPresets {
		found := false
		for _, actual := range tuningPresets {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tuning preset %s not found", expected)
		}
	}
}

func TestGetModelFamilies(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	o := &PresetOptions{configFlags: configFlags}

	families := o.getModelFamilies()

	expectedFamilies := []string{"llama", "falcon", "phi", "mistral"}
	if len(families) != len(expectedFamilies) {
		t.Errorf("Expected %d families, got %d", len(expectedFamilies), len(families))
	}

	for _, expected := range expectedFamilies {
		found := false
		for _, actual := range families {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected family %s not found in result", expected)
		}
	}
}
