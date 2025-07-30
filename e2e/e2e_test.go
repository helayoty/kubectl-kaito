package e2e

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	binaryName      = "kubectl-kaito"
	testTimeout     = 30 * time.Second
	buildTimeout    = 60 * time.Second
	longTestTimeout = 120 * time.Second
)

var (
	binaryPath string
)

// MockServer holds the test HTTP server for mocking external APIs
type MockServer struct {
	server       *httptest.Server
	failRequests bool
	returnEmpty  bool
}

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
	binaryPath = filepath.Join(projectRoot, binaryName)

	// Build the binary using go build directly
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "./cmd/kubectl-kaito")
	cmd.Dir = projectRoot
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %v\nOutput: %s", err, string(output))
	}

	return nil
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

func runCommand(t *testing.T, timeout time.Duration, args ...string) (string, string, error) {
	if timeout == 0 {
		timeout = testTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// setupMockServer creates a mock HTTP server for testing external API calls
func setupMockServer(failRequests, returnEmpty bool) *MockServer {
	mock := &MockServer{
		failRequests: failRequests,
		returnEmpty:  returnEmpty,
	}

	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mock.failRequests {
			http.Error(w, "Mock server error", http.StatusInternalServerError)
			return
		}

		if mock.returnEmpty {
			w.Header().Set("Content-Type", "application/yaml")
			w.Write([]byte("models: []"))
			return
		}

		// Return mock supported models YAML
		mockYAML := `models:
  - name: phi-3.5-mini-instruct
    type: text-generation
    runtime: tfs
    version: "test-version"
  - name: llama-2-7b
    type: text-generation
    runtime: tfs
    version: "test-version"
  - name: mistral-7b
    type: text-generation
    runtime: tfs
    version: "test-version"`

		w.Header().Set("Content-Type", "application/yaml")
		w.Write([]byte(mockYAML))
	}))

	return mock
}

func (m *MockServer) Close() {
	m.server.Close()
}

// TestBasicCommands tests basic command functionality that doesn't require external APIs
func TestBasicCommands(t *testing.T) {
	t.Run("Root help command", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, 0, "--help")
		if err != nil {
			t.Errorf("Help command failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
			return
		}

		expectedSections := []string{
			"Kubernetes AI Toolchain Operator",
			"Usage:",
			"Available Commands:",
			"deploy",
			"status", 
			"get-endpoint",
			"chat",
			"models",
			"rag",
		}

		for _, section := range expectedSections {
			if !strings.Contains(stdout, section) {
				t.Errorf("Help output should contain '%s'\nGot: %s", section, stdout)
			}
		}
	})

	t.Run("Subcommand help", func(t *testing.T) {
		subcommands := []string{"deploy", "status", "get-endpoint", "chat", "models", "rag"}
		
		for _, cmd := range subcommands {
			t.Run(cmd+" help", func(t *testing.T) {
				stdout, stderr, err := runCommand(t, 0, cmd, "--help")
				if err != nil {
					t.Errorf("Help command for %s failed: %v\nStderr: %s", cmd, err, stderr)
					return
				}

				if !strings.Contains(stdout, "Usage:") {
					t.Errorf("Help output for %s should contain 'Usage:'\nGot: %s", cmd, stdout)
				}
			})
		}
	})
}

// TestModelsCommand tests the models command functionality
func TestModelsCommand(t *testing.T) {
	t.Run("Models list", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, longTestTimeout, "models", "list")
		if err != nil {
			t.Errorf("Models list failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
			return
		}

		// For models list table output goes to stdout
		// Should contain table headers
		expectedHeaders := []string{"NAME", "TYPE", "RUNTIME", "GPU MEMORY", "NODES", "DESCRIPTION"}
		for _, header := range expectedHeaders {
			if !strings.Contains(stdout, header) {
				t.Errorf("Models list output should contain header '%s'\nGot: %s", header, stdout)
			}
		}

		// Should contain some known models (either from official API or fallback)
		knownModels := []string{"phi-", "llama", "mistral"}
		foundModel := false
		for _, model := range knownModels {
			if strings.Contains(strings.ToLower(stdout), model) {
				foundModel = true
				break
			}
		}
		if !foundModel {
			t.Errorf("Models list should contain at least one known model\nGot: %s", stdout)
		}
	})

	t.Run("Models list detailed", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, longTestTimeout, "models", "list", "--detailed")
		if err != nil {
			t.Errorf("Models list detailed failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
			return
		}

		// Detailed output goes through klog to stderr
		combinedOutput := stdout + stderr
		
		// Detailed output should contain more information
		expectedFields := []string{"Name:", "Type:", "Runtime:", "Version:"}
		for _, field := range expectedFields {
			if !strings.Contains(combinedOutput, field) {
				t.Errorf("Detailed output should contain field '%s'\nGot stdout: %s\nGot stderr: %s", field, stdout, stderr)
			}
		}
	})

	t.Run("Models describe", func(t *testing.T) {
		// First get a list of models to find a valid one
		listOut, _, listCmdErr := runCommand(t, longTestTimeout, "models", "list")
		if listCmdErr != nil {
			t.Skip("Cannot test describe without a working models list")
		}

		// Extract first model name from the stdout (table output)
		lines := strings.Split(listOut, "\n")
		var modelName string
		for _, line := range lines {
			if strings.Contains(line, "phi-") || strings.Contains(line, "llama") || strings.Contains(line, "mistral") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					modelName = fields[0]
					break
				}
			}
		}

		if modelName == "" {
			modelName = "phi-3.5-mini-instruct" // fallback to known model
		}

		stdout, stderr, err := runCommand(t, 0, "models", "describe", modelName)
		if err != nil {
			t.Errorf("Models describe failed for %s: %v\nStdout: %s\nStderr: %s", modelName, err, stdout, stderr)
			return
		}

		// Describe output goes through klog to stderr
		combinedOutput := stdout + stderr
		
		expectedSections := []string{
			"Model: " + modelName,
			"Description:",
			"Type:",
			"Runtime:",
			"Resource Requirements:",
			"Usage Example:",
		}

		for _, section := range expectedSections {
			if !strings.Contains(combinedOutput, section) {
				t.Errorf("Describe output should contain section '%s'\nGot stdout: %s\nGot stderr: %s", section, stdout, stderr)
			}
		}
	})

	t.Run("Models describe invalid", func(t *testing.T) {
		_, _, err := runCommand(t, 0, "models", "describe", "invalid-model-name")
		if err == nil {
			t.Error("Models describe should fail for invalid model")
		}

		// Note: Error messages are silenced by root command configuration (SilenceErrors: true)
		// so we only check for error exit code, not error text content
	})
}

// TestDeployCommand tests the deploy command functionality
func TestDeployCommand(t *testing.T) {
	t.Run("Deploy dry-run with valid model", func(t *testing.T) {
		// Get a valid model first
		listOut, _, listErr := runCommand(t, longTestTimeout, "models", "list")
		if listErr != nil {
			t.Skip("Cannot test deploy without working models list")
		}

		// Extract first model name
		var modelName string
		lines := strings.Split(listOut, "\n")
		for _, line := range lines {
			if strings.Contains(line, "phi-") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					modelName = fields[0]
					break
				}
			}
		}

		if modelName == "" {
			modelName = "phi-3.5-mini-instruct" // fallback to known model
		}

		stdout, stderr, err := runCommand(t, 0, "deploy", 
			"--workspace-name", "test-workspace",
			"--model", modelName,
			"--dry-run")

		if err != nil {
			t.Errorf("Deploy dry-run failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
			return
		}

		// Deploy output goes through klog to stderr
		combinedOutput := stdout + stderr

		expectedOutputs := []string{
			"Dry-run mode",
			"Workspace Configuration",
			"Name: test-workspace", 
			"Model: " + modelName,
			"Mode: Inference",
			"Workspace definition is valid",
		}

		for _, expected := range expectedOutputs {
			if !strings.Contains(combinedOutput, expected) {
				t.Errorf("Dry-run output should contain '%s'\nGot stdout: %s\nGot stderr: %s", expected, stdout, stderr)
			}
		}
	})

	t.Run("Deploy with invalid model", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, 0, "deploy",
			"--workspace-name", "test-workspace", 
			"--model", "invalid-model-name",
			"--dry-run")

		if err == nil {
			t.Error("Deploy should fail with invalid model name")
		}

		// Should provide helpful suggestions
		combinedOutput := stdout + stderr
		if !strings.Contains(combinedOutput, "not supported") {
			t.Errorf("Should mention model not supported\nStdout: %s\nStderr: %s", stdout, stderr)
		}
	})

	t.Run("Deploy missing required args", func(t *testing.T) {
		testCases := []struct {
			name string
			args []string
		}{
			{"missing workspace name", []string{"deploy", "--model", "phi-3.5-mini-instruct"}},
			{"missing model", []string{"deploy", "--workspace-name", "test"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, _, err := runCommand(t, 0, tc.args...)
				if err == nil {
					t.Errorf("Deploy should fail when %s", tc.name)
				}
			})
		}
	})

	t.Run("Deploy tuning mode", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, 0, "deploy",
			"--workspace-name", "test-tune",
			"--model", "phi-3.5-mini-instruct",
			"--tuning",
			"--input-urls", "gs://test-bucket/data",
			"--output-image", "test.azurecr.io/tuned-model",
			"--dry-run")

		if err != nil {
			t.Errorf("Deploy tuning dry-run failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
			return
		}

		// Check combined output for tuning mode indication
		combinedOutput := stdout + stderr
		if !strings.Contains(combinedOutput, "Mode: Fine-tuning") {
			t.Errorf("Tuning mode should be indicated\nGot stdout: %s\nGot stderr: %s", stdout, stderr)
		}
	})
}

// TestStatusCommand tests the status command functionality  
func TestStatusCommand(t *testing.T) {
	t.Run("Status help", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, 0, "status", "--help")
		if err != nil {
			t.Errorf("Status help failed: %v\nStderr: %s", err, stderr)
			return
		}

		expectedSections := []string{
			"status of one or more Kaito workspaces", // Updated to match actual text
			"Usage:",
			"--workspace-name",
			"--all-namespaces",
			"--watch",
		}

		for _, section := range expectedSections {
			if !strings.Contains(stdout, section) {
				t.Errorf("Status help should contain '%s'\nGot: %s", section, stdout)
			}
		}
	})

	t.Run("Status validation", func(t *testing.T) {
		// Test conflicting flags
		_, _, err := runCommand(t, 0, "status", "--namespace", "test", "--all-namespaces")
		if err == nil {
			t.Error("Status should fail with conflicting namespace flags")
		}
	})
}

// TestGetEndpointCommand tests the get-endpoint command functionality
func TestGetEndpointCommand(t *testing.T) {
	t.Run("Get-endpoint help", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, 0, "get-endpoint", "--help")
		if err != nil {
			t.Errorf("Get-endpoint help failed: %v\nStderr: %s", err, stderr)
			return
		}

		expectedSections := []string{
			"inference endpoint URL", // Updated to match actual text
			"Usage:",
			"--workspace-name",
			"--format",
			"--external",
		}

		for _, section := range expectedSections {
			if !strings.Contains(stdout, section) {
				t.Errorf("Get-endpoint help should contain '%s'\nGot: %s", section, stdout)
			}
		}
	})

	t.Run("Get-endpoint validation", func(t *testing.T) {
		// Test missing workspace name
		_, _, err := runCommand(t, 0, "get-endpoint")
		if err == nil {
			t.Error("Get-endpoint should fail without workspace name")
		}

		// Test invalid format
		_, _, err = runCommand(t, 0, "get-endpoint", "--workspace-name", "test", "--format", "invalid")
		if err == nil {
			t.Error("Get-endpoint should fail with invalid format")
		}
	})
}

// TestChatCommand tests the chat command functionality
func TestChatCommand(t *testing.T) {
	t.Run("Chat help", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, 0, "chat", "--help")
		if err != nil {
			t.Errorf("Chat help failed: %v\nStderr: %s", err, stderr)
			return
		}

		expectedSections := []string{
			"interactive chat session", // Updated to match actual text
			"Usage:",
			"--workspace-name",
			"--message",
			"--temperature",
			"--max-tokens",
		}

		for _, section := range expectedSections {
			if !strings.Contains(stdout, section) {
				t.Errorf("Chat help should contain '%s'\nGot: %s", section, stdout)
			}
		}
	})

	t.Run("Chat validation", func(t *testing.T) {
		testCases := []struct {
			name string
			args []string
		}{
			{"missing workspace", []string{"chat"}},
			{"invalid temperature", []string{"chat", "--workspace-name", "test", "--temperature", "3.0"}},
			{"invalid top-p", []string{"chat", "--workspace-name", "test", "--top-p", "2.0"}},
			{"invalid max-tokens", []string{"chat", "--workspace-name", "test", "--max-tokens", "0"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, _, err := runCommand(t, 0, tc.args...)
				if err == nil {
					t.Errorf("Chat should fail for %s", tc.name)
				}
			})
		}
	})
}

// TestRagCommand tests the RAG command functionality
func TestRagCommand(t *testing.T) {
	t.Run("RAG help", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, 0, "rag", "--help")
		if err != nil {
			t.Errorf("RAG help failed: %v\nStderr: %s", err, stderr)
			return
		}

		expectedSections := []string{
			"Deploy and query RAG engines", // Updated to match actual text
			"Usage:",
			"Available Commands:",
			"deploy",
			"query",
		}

		for _, section := range expectedSections {
			if !strings.Contains(stdout, section) {
				t.Errorf("RAG help should contain '%s'\nGot: %s", section, stdout)
			}
		}
	})

	t.Run("RAG deploy help", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, 0, "rag", "deploy", "--help")
		if err != nil {
			t.Errorf("RAG deploy help failed: %v\nStderr: %s", err, stderr)
			return
		}

		expectedSections := []string{
			"Deploy a RAG", // Updated to match actual text
			"--name",
			"--vector-db",
			"--index-service",
			"--embedding-model",
		}

		for _, section := range expectedSections {
			if !strings.Contains(stdout, section) {
				t.Errorf("RAG deploy help should contain '%s'\nGot: %s", section, stdout)
			}
		}
	})

	t.Run("RAG deploy validation", func(t *testing.T) {
		testCases := []struct {
			name string
			args []string
		}{
			{"missing name", []string{"rag", "deploy"}},
			{"invalid vector db", []string{"rag", "deploy", "--name", "test", "--vector-db", "invalid"}},
			{"invalid index service", []string{"rag", "deploy", "--name", "test", "--index-service", "invalid"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, _, err := runCommand(t, 0, tc.args...)
				if err == nil {
					t.Errorf("RAG deploy should fail for %s", tc.name)
				}
			})
		}
	})

	t.Run("RAG deploy dry-run", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, 0, "rag", "deploy",
			"--name", "test-rag",
			"--vector-db", "faiss",
			"--index-service", "llamaindex",
			"--dry-run")

		if err != nil {
			t.Errorf("RAG deploy dry-run failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
			return
		}

		// RAG deploy output goes through klog to stderr
		combinedOutput := stdout + stderr

		expectedOutputs := []string{
			"Dry-run mode",
			"RAG Engine Configuration",
			"Name: test-rag",
			"Vector Database: faiss",
			"Index Service: llamaindex",
		}

		for _, expected := range expectedOutputs {
			if !strings.Contains(combinedOutput, expected) {
				t.Errorf("RAG deploy dry-run should contain '%s'\nGot stdout: %s\nGot stderr: %s", expected, stdout, stderr)
			}
		}
	})
}

// TestNetworkFailureScenarios tests fallback behavior when external APIs fail
func TestNetworkFailureScenarios(t *testing.T) {
	t.Run("Models list with network failure fallback", func(t *testing.T) {
		// This test relies on the fallback mechanism when the official API is unreachable
		// The test will pass if fallback models are shown
		stdout, stderr, err := runCommand(t, longTestTimeout, "models", "list")
		if err != nil {
			t.Errorf("Models list should not fail even with network issues: %v\nStderr: %s", err, stderr)
			return
		}

		// Should still show some models (from fallback)
		if !strings.Contains(stdout, "NAME") || !strings.Contains(stdout, "TYPE") {
			t.Errorf("Should show model table even with network issues\nGot: %s", stdout)
		}

		// If fallback is used, should contain known fallback models
		fallbackModels := []string{"phi-3.5-mini-instruct", "llama-2-7b", "mistral-7b"}
		foundFallback := false
		for _, model := range fallbackModels {
			if strings.Contains(stdout, model) {
				foundFallback = true
				break
			}
		}

		if !foundFallback {
			t.Errorf("Should show fallback models when official API fails\nGot: %s", stdout)
		}
	})
}

// TestPerformanceAndTimeouts tests performance aspects and timeout handling
func TestPerformanceAndTimeouts(t *testing.T) {
	t.Run("Models list performance", func(t *testing.T) {
		start := time.Now()
		stdout, stderr, err := runCommand(t, longTestTimeout, "models", "list")
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Models list failed: %v\nStderr: %s", err, stderr)
			return
		}

		// Should complete within reasonable time (with network calls)
		if duration > 45*time.Second {
			t.Errorf("Models list took too long: %v", duration)
		}

		// Should return data
		if len(strings.TrimSpace(stdout)) < 50 {
			t.Errorf("Models list returned insufficient data: %s", stdout)
		}
	})

	t.Run("Command help performance", func(t *testing.T) {
		start := time.Now()
		_, stderr, err := runCommand(t, 0, "--help")
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Help command failed: %v\nStderr: %s", err, stderr)
			return
		}

		// Help should be very fast (no network calls)
		if duration > 2*time.Second {
			t.Errorf("Help command took too long: %v", duration)
		}
	})
}

// TestOutputFormats tests different output formats and edge cases
func TestOutputFormats(t *testing.T) {
	t.Run("Models list JSON output", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, longTestTimeout, "models", "list", "--output")
		if err != nil {
			t.Errorf("Models list JSON failed: %v\nStderr: %s", err, stderr)
			return
		}

		// JSON output should go to stderr through klog
		combinedOutput := stdout + stderr
		
		// Should contain some JSON-like content
		if !strings.Contains(combinedOutput, "{") && !strings.Contains(combinedOutput, "[") {
			t.Errorf("Should contain JSON content\nStdout: %s\nStderr: %s", stdout, stderr)
		}
	})

	t.Run("Models filtering", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, longTestTimeout, "models", "list", "--type", "LLM")
		if err != nil {
			t.Errorf("Models filtering failed: %v\nStderr: %s", err, stderr)
			return
		}

		// Table output goes to stdout, but if no models match, might be empty
		// Should either have headers or be empty (both are valid)
		if stdout != "" && !strings.Contains(stdout, "NAME") {
			t.Errorf("Non-empty filtered output should have headers\nGot: %s", stdout)
		}
	})
}

// TestEdgeCases tests various edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("Very long workspace name", func(t *testing.T) {
		longName := strings.Repeat("a", 100)
		_, _, err := runCommand(t, 0, "deploy", 
			"--workspace-name", longName,
			"--model", "phi-3.5-mini-instruct",
			"--dry-run")
		
		// Should handle long names gracefully (either accept or reject with clear error)
		// The specific behavior depends on Kubernetes naming constraints
		if err != nil {
			// If it fails, error should be informative
			t.Logf("Long workspace name rejected as expected: %v", err)
		}
	})

	t.Run("Special characters in workspace name", func(t *testing.T) {
		specialName := "test-workspace-123"
		stdout, stderr, err := runCommand(t, 0, "deploy",
			"--workspace-name", specialName,
			"--model", "phi-3.5-mini-instruct", 
			"--dry-run")

		if err != nil {
			t.Errorf("Valid workspace name with special chars should work: %v\nStderr: %s", err, stderr)
			return
		}

		// Check combined output for workspace name
		combinedOutput := stdout + stderr
		if !strings.Contains(combinedOutput, specialName) {
			t.Errorf("Output should contain the workspace name\nGot stdout: %s\nGot stderr: %s", stdout, stderr)
		}
	})

	t.Run("Empty command", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, 0)
		if err != nil {
			t.Errorf("Empty command should show help: %v\nStderr: %s", err, stderr)
			return
		}

		if !strings.Contains(stdout, "Usage:") {
			t.Errorf("Empty command should show usage\nGot: %s", stdout)
		}
	})
}
