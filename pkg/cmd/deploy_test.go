package cmd

import (
	"testing"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestNewDeployCmd(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := NewDeployCmd(configFlags)

	// Test basic command properties
	if cmd.Use != "deploy" {
		t.Errorf("Expected Use to be 'deploy', got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Deploy command should have a short description")
	}

	if cmd.Long == "" {
		t.Error("Deploy command should have a long description")
	}

	// Test required flags
	requiredFlags := []string{"name", "model"}
	for _, flagName := range requiredFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected --%s flag to exist", flagName)
		}
	}

	// Test optional flags
	optionalFlags := []string{"gpus", "preset", "instance-type", "namespace"}
	for _, flagName := range optionalFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected --%s flag to exist", flagName)
		}
	}
}

func TestDeployOptionsComplete(t *testing.T) {
	tests := []struct {
		name              string
		namespace         string
		instanceType      string
		preset            string
		expectedNamespace string
		expectedInstance  string
		expectedPreset    string
	}{
		{
			name:              "default values",
			namespace:         "",
			instanceType:      "",
			preset:            "",
			expectedNamespace: "default",
			expectedInstance:  "Standard_NC24ads_A100_v4",
			expectedPreset:    "base",
		},
		{
			name:              "custom values",
			namespace:         "custom-ns",
			instanceType:      "Standard_NC12s_v3",
			preset:            "instruct",
			expectedNamespace: "custom-ns",
			expectedInstance:  "Standard_NC12s_v3",
			expectedPreset:    "instruct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFlags := genericclioptions.NewConfigFlags(true)
			o := &DeployOptions{
				configFlags:  configFlags,
				Namespace:    tt.namespace,
				InstanceType: tt.instanceType,
				Preset:       tt.preset,
				Model:        "test-model",
			}

			err := o.Complete()
			if err != nil {
				t.Errorf("Complete() returned error: %v", err)
			}

			if o.Namespace != tt.expectedNamespace {
				t.Errorf("Expected namespace %s, got %s", tt.expectedNamespace, o.Namespace)
			}

			if o.InstanceType != tt.expectedInstance {
				t.Errorf("Expected instance type %s, got %s", tt.expectedInstance, o.InstanceType)
			}

			if o.Preset != tt.expectedPreset {
				t.Errorf("Expected preset %s, got %s", tt.expectedPreset, o.Preset)
			}

			// Test that label selector is set
			if o.LabelSelector == nil {
				t.Error("LabelSelector should be initialized")
			}

			expectedApp := o.Model
			if actualApp, exists := o.LabelSelector["apps"]; !exists || actualApp != expectedApp {
				t.Errorf("Expected LabelSelector[apps] to be %s, got %s", expectedApp, actualApp)
			}
		})
	}
}

func TestDeployOptionsValidate(t *testing.T) {
	tests := []struct {
		name        string
		options     DeployOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid options",
			options: DeployOptions{
				Name:  "test-workspace",
				Model: "llama-3-8b-instruct",
				GPUs:  1,
			},
			expectError: false,
		},
		{
			name: "missing name",
			options: DeployOptions{
				Model: "llama-3-8b-instruct",
				GPUs:  1,
			},
			expectError: true,
			errorMsg:    "name is required",
		},
		{
			name: "missing model",
			options: DeployOptions{
				Name: "test-workspace",
				GPUs: 1,
			},
			expectError: true,
			errorMsg:    "model is required",
		},
		{
			name: "invalid GPU count",
			options: DeployOptions{
				Name:  "test-workspace",
				Model: "llama-3-8b-instruct",
				GPUs:  0,
			},
			expectError: true,
			errorMsg:    "gpus must be at least 1",
		},
		{
			name: "negative GPU count",
			options: DeployOptions{
				Name:  "test-workspace",
				Model: "llama-3-8b-instruct",
				GPUs:  -1,
			},
			expectError: true,
			errorMsg:    "gpus must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestDeployCommandFlags(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := NewDeployCmd(configFlags)

	// Test default values
	tests := []struct {
		flagName     string
		expectedType string
		hasDefault   bool
		defaultValue interface{}
	}{
		{"name", "string", false, ""},
		{"model", "string", false, ""},
		{"gpus", "int", true, "1"},
		{"preset", "string", false, ""},
		{"instance-type", "string", false, ""},
		{"namespace", "string", true, "default"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag --%s should exist", tt.flagName)
				return
			}

			if tt.hasDefault {
				if flag.DefValue != tt.defaultValue {
					t.Errorf("Flag --%s should have default value %v, got %v",
						tt.flagName, tt.defaultValue, flag.DefValue)
				}
			}
		})
	}
}
