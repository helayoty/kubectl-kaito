package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	binaryName   = "kubectl-kaito"
	testTimeout  = 30 * time.Second
	buildTimeout = 60 * time.Second
)

var (
	binaryPath string
)

func TestMain(m *testing.M) {
	// Build the binary before running tests
	if err := buildBinary(); err != nil {
		panic("Failed to build binary: " + err.Error())
	}

	// Run tests
	code := m.Run()

	// Cleanup
	cleanup()

	os.Exit(code)
}

func buildBinary() error {
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	// Get the project root directory
	projectRoot, err := getProjectRoot()
	if err != nil {
		return err
	}

	// Set binary path
	binaryPath = filepath.Join(projectRoot, "bin", binaryName)

	// Build the binary
	cmd := exec.CommandContext(ctx, "make", "build")
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func getProjectRoot() (string, error) {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Go up one level from e2e to project root
	return filepath.Dir(wd), nil
}

func cleanup() {
	// Remove binary if it exists
	if binaryPath != "" {
		os.Remove(binaryPath)
	}
}

func runCommand(t *testing.T, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// TestBasicFunctionality tests basic commands that don't require a cluster
func TestBasicFunctionality(t *testing.T) {
	t.Run("version command", testVersionCommand)
	t.Run("help commands", testHelpCommands)
	t.Run("preset list", testPresetList)
	t.Run("dry run commands", testDryRunCommands)
}

func testVersionCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"version full", []string{"version"}},
		{"version short", []string{"version", "--short"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCommand(t, tt.args...)
			if err != nil {
				t.Errorf("Version command failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
				return
			}

			if stdout == "" {
				t.Error("Version command produced no output")
			}

			// For short version, output should be a single line
			if len(tt.args) > 1 && tt.args[1] == "--short" {
				lines := strings.Split(strings.TrimSpace(stdout), "\n")
				if len(lines) != 1 {
					t.Errorf("Short version should output single line, got %d lines", len(lines))
				}
			}
		})
	}
}

func testHelpCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"main help", []string{"--help"}},
		{"deploy help", []string{"deploy", "--help"}},
		{"tune help", []string{"tune", "--help"}},
		{"preset help", []string{"preset", "--help"}},
		{"status help", []string{"status", "--help"}},
		{"logs help", []string{"logs", "--help"}},
		{"delete help", []string{"delete", "--help"}},
		{"version help", []string{"version", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCommand(t, tt.args...)
			if err != nil {
				t.Errorf("Help command failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
				return
			}

			if !strings.Contains(stdout, "Usage:") {
				t.Error("Help output should contain 'Usage:'")
			}
		})
	}
}

func testPresetList(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			"list all presets",
			[]string{"preset", "list"},
			[]string{"Llama", "Falcon", "Phi", "Mistral", "llama-2-7b", "falcon-7b", "phi-2", "mistral-7b"},
		},
		{
			"list llama presets",
			[]string{"preset", "list", "--model", "llama"},
			[]string{"llama-2-7b", "llama-2-7b-chat", "llama-3-8b-instruct"},
		},
		{
			"list falcon presets",
			[]string{"preset", "list", "--model", "falcon"},
			[]string{"falcon-7b", "falcon-7b-instruct"},
		},
		{
			"list tuning presets",
			[]string{"preset", "list", "--model", "tuning"},
			[]string{"qlora", "lora"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCommand(t, tt.args...)
			if err != nil {
				t.Errorf("Preset list failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
				return
			}

			for _, expected := range tt.expected {
				if !strings.Contains(stdout, expected) {
					t.Errorf("Output should contain '%s'\nGot: %s", expected, stdout)
				}
			}
		})
	}
}

func testDryRunCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			"deploy dry run",
			[]string{"deploy", "--name", "test-workspace", "--model", "llama-2-7b", "--dry-run"},
		},
		{
			"tune dry run",
			[]string{"tune", "--name", "test-tune", "--model", "phi-2", "--dataset", "gs://test-data", "--preset", "lora", "--dry-run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCommand(t, tt.args...)
			if err != nil {
				t.Errorf("Dry run command failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
				return
			}

			// Verify dry-run output
			if !strings.Contains(stdout, "Dry-run mode") {
				t.Error("Dry-run output should contain 'Dry-run mode'")
			}

			if !strings.Contains(stdout, "would be created") {
				t.Error("Dry-run output should indicate what would be created")
			}
		})
	}
}

// TestCommandErrorHandling tests error conditions
func TestCommandErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			"deploy missing name",
			[]string{"deploy", "--model", "llama-2-7b"},
			true,
		},
		{
			"deploy missing model",
			[]string{"deploy", "--name", "test"},
			true,
		},
		{
			"tune missing dataset",
			[]string{"tune", "--name", "test", "--model", "llama-2-7b"},
			true,
		},
		{
			"preset invalid model",
			[]string{"preset", "list", "--model", "invalid"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runCommand(t, tt.args...)
			if tt.expectError && err == nil {
				t.Error("Expected command to fail but it succeeded")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected command to succeed but it failed: %v", err)
			}
		})
	}
}
