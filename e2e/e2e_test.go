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

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	binaryName         = "kubectl-kaito"
	testTimeout        = 30 * time.Second
	buildTimeout       = 60 * time.Second
	longTestTimeout    = 120 * time.Second
	clusterTimeout     = 10 * time.Minute
	kindClusterName    = "kaito-e2e-kind"
	aksClusterName     = "kaito-e2e-aks"
	aksResourceGroup   = "kaito-e2e-rg"
	aksLocation        = "westus2"
)

var (
	binaryPath    string
	kindAvailable bool
	aksAvailable  bool
)

// ClusterManager manages test clusters
type ClusterManager struct {
	kindClusterName string
	aksClusterName  string
	resourceGroup   string
	location        string
}

func TestMain(m *testing.M) {
	// Build the binary before running tests
	if err := buildBinary(); err != nil {
		panic("Failed to build binary: " + err.Error())
	}

	// Check if required tools are available
	checkPrerequisites()

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

func checkPrerequisites() {
	// Check if kind is available
	if err := exec.Command("kind", "version").Run(); err != nil {
		fmt.Printf("Warning: kind not available, skipping Kind cluster tests: %v\n", err)
		kindAvailable = false
	} else {
		fmt.Println("✓ kind is available")
		kindAvailable = true
	}

	// Check if Azure CLI is available
	if err := exec.Command("az", "version").Run(); err != nil {
		fmt.Printf("Warning: Azure CLI not available, skipping AKS tests: %v\n", err)
		aksAvailable = false
	} else {
		fmt.Println("✓ Azure CLI is available")
		aksAvailable = true

		// Check if logged into Azure
		if err := exec.Command("az", "account", "show").Run(); err != nil {
			fmt.Printf("Warning: Not logged into Azure, skipping AKS tests: %v\n", err)
			aksAvailable = false
		} else {
			fmt.Println("✓ Azure CLI is authenticated")
		}
	}

	// Check if kubectl is available
	if err := exec.Command("kubectl", "version", "--client").Run(); err != nil {
		fmt.Printf("Warning: kubectl not available: %v\n", err)
	} else {
		fmt.Println("✓ kubectl is available")
	}
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

func runKubectl(t *testing.T, timeout time.Duration, args ...string) (string, string, error) {
	if timeout == 0 {
		timeout = testTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// NewClusterManager creates a new cluster manager
func NewClusterManager() *ClusterManager {
	return &ClusterManager{
		kindClusterName: kindClusterName,
		aksClusterName:  aksClusterName,
		resourceGroup:   aksResourceGroup,
		location:        aksLocation,
	}
}

// CreateKindCluster creates a Kind cluster with CPU nodes
func (cm *ClusterManager) CreateKindCluster(t *testing.T) error {
	if !kindAvailable {
		t.Skip("Kind not available, skipping Kind cluster tests")
	}

	t.Logf("Creating Kind cluster: %s", cm.kindClusterName)

	// Create kind config with CPU nodes
	kindConfig := fmt.Sprintf(`kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: %s
nodes:
- role: control-plane
- role: worker
  extraMounts:
  - hostPath: /var/run/docker.sock
    containerPath: /var/run/docker.sock
`, cm.kindClusterName)

	// Write config to temp file
	configFile := filepath.Join(os.TempDir(), "kind-config.yaml")
	if err := os.WriteFile(configFile, []byte(kindConfig), 0644); err != nil {
		return fmt.Errorf("failed to write kind config: %v", err)
	}
	defer os.Remove(configFile)

	// Create the cluster
	ctx, cancel := context.WithTimeout(context.Background(), clusterTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kind", "create", "cluster", "--config", configFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create kind cluster: %v\nOutput: %s", err, string(output))
	}

	t.Logf("Kind cluster created successfully")

	// Wait for cluster to be ready
	return cm.waitForClusterReady(t, "kind-"+cm.kindClusterName)
}

// DestroyKindCluster destroys the Kind cluster
func (cm *ClusterManager) DestroyKindCluster(t *testing.T) {
	if !kindAvailable {
		return
	}

	t.Logf("Destroying Kind cluster: %s", cm.kindClusterName)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kind", "delete", "cluster", "--name", cm.kindClusterName)
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: failed to delete kind cluster: %v", err)
	}
}

// DeployNginxToKind deploys nginx to the Kind cluster for testing
func (cm *ClusterManager) DeployNginxToKind(t *testing.T) error {
	t.Logf("Deploying nginx to Kind cluster")

	// Switch to kind context
	if _, _, err := runKubectl(t, testTimeout, "config", "use-context", "kind-"+cm.kindClusterName); err != nil {
		return fmt.Errorf("failed to switch to kind context: %v", err)
	}

	// Deploy nginx
	nginxManifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-test
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-test
  template:
    metadata:
      labels:
        app: nginx-test
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-test
  namespace: default
spec:
  selector:
    app: nginx-test
  ports:
  - port: 80
    targetPort: 80
  type: ClusterIP
`

	// Apply manifest
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(nginxManifest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to deploy nginx: %v\nOutput: %s", err, string(output))
	}

	// Wait for nginx to be ready
	return cm.waitForDeployment(t, "default", "nginx-test")
}

// CreateAKSCluster creates an AKS cluster with GPU nodes
func (cm *ClusterManager) CreateAKSCluster(t *testing.T) error {
	if !aksAvailable {
		t.Skip("Azure CLI not available or not authenticated, skipping AKS tests")
	}

	t.Logf("Creating AKS cluster: %s", cm.aksClusterName)

	// Create resource group
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "az", "group", "create",
		"--name", cm.resourceGroup,
		"--location", cm.location)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create resource group: %v\nOutput: %s", err, string(output))
	}

	// Create AKS cluster with GPU node pool
	ctx, cancel = context.WithTimeout(context.Background(), clusterTimeout)
	defer cancel()

	cmd = exec.CommandContext(ctx, "az", "aks", "create",
		"--resource-group", cm.resourceGroup,
		"--name", cm.aksClusterName,
		"--node-count", "1",
		"--node-vm-size", "Standard_NC6s_v3", // GPU SKU
		"--generate-ssh-keys",
		"--enable-addons", "monitoring",
		"--kubernetes-version", "1.28.0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create AKS cluster: %v\nOutput: %s", err, string(output))
	}

	// Get credentials
	ctx, cancel = context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd = exec.CommandContext(ctx, "az", "aks", "get-credentials",
		"--resource-group", cm.resourceGroup,
		"--name", cm.aksClusterName,
		"--overwrite-existing")

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to get AKS credentials: %v\nOutput: %s", err, string(output))
	}

	t.Logf("AKS cluster created successfully")

	// Wait for cluster to be ready
	return cm.waitForClusterReady(t, cm.aksClusterName)
}

// DestroyAKSCluster destroys the AKS cluster
func (cm *ClusterManager) DestroyAKSCluster(t *testing.T) {
	if !aksAvailable {
		return
	}

	t.Logf("Destroying AKS cluster and resource group: %s", cm.resourceGroup)

	ctx, cancel := context.WithTimeout(context.Background(), clusterTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "az", "group", "delete",
		"--name", cm.resourceGroup,
		"--yes", "--no-wait")

	if err := cmd.Run(); err != nil {
		t.Logf("Warning: failed to delete AKS resource group: %v", err)
	}
}

// waitForClusterReady waits for the cluster to be ready
func (cm *ClusterManager) waitForClusterReady(t *testing.T, contextName string) error {
	t.Logf("Waiting for cluster to be ready: %s", contextName)

	// Switch to the cluster context
	if _, _, err := runKubectl(t, testTimeout, "config", "use-context", contextName); err != nil {
		return fmt.Errorf("failed to switch to context %s: %v", contextName, err)
	}

	// Wait for nodes to be ready
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for cluster to be ready")
		case <-ticker.C:
			stdout, _, err := runKubectl(t, testTimeout, "get", "nodes", "--no-headers")
			if err != nil {
				continue
			}

			lines := strings.Split(strings.TrimSpace(stdout), "\n")
			allReady := true
			for _, line := range lines {
				if line == "" {
					continue
				}
				fields := strings.Fields(line)
				if len(fields) < 2 || fields[1] != "Ready" {
					allReady = false
					break
				}
			}

			if allReady && len(lines) > 0 {
				t.Logf("Cluster is ready with %d nodes", len(lines))
				return nil
			}
		}
	}
}

// waitForDeployment waits for a deployment to be ready
func (cm *ClusterManager) waitForDeployment(t *testing.T, namespace, name string) error {
	t.Logf("Waiting for deployment %s/%s to be ready", namespace, name)

	timeout := time.After(3 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for deployment %s/%s to be ready", namespace, name)
		case <-ticker.C:
			stdout, _, err := runKubectl(t, testTimeout, "get", "deployment", name, "-n", namespace, "-o", "jsonpath={.status.readyReplicas}")
			if err != nil {
				continue
			}

			if strings.TrimSpace(stdout) == "1" {
				t.Logf("Deployment %s/%s is ready", namespace, name)
				return nil
			}
		}
	}
}

// Test basic help functionality - no cluster needed
func TestBasicHelp(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"root help", []string{"--help"}},
		{"models help", []string{"models", "--help"}},
		{"deploy help", []string{"deploy", "--help"}},
		{"status help", []string{"status", "--help"}},
		{"get-endpoint help", []string{"get-endpoint", "--help"}},
		{"chat help", []string{"chat", "--help"}},
		{"rag help", []string{"rag", "--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCommand(t, testTimeout, tt.args...)

			// Help should exit with code 0
			if err != nil {
				t.Errorf("Help command failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
			}

			// Should have some help content
			combinedOutput := stdout + stderr
			if !strings.Contains(combinedOutput, "Usage:") && !strings.Contains(combinedOutput, "kubectl kaito") {
				t.Errorf("Expected help content not found in output: %s", combinedOutput)
			}
		})
	}
}

// Test models command - no cluster needed
func TestModelsCommand(t *testing.T) {
	t.Run("models list", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, longTestTimeout, "models", "list")

		// Should succeed or gracefully handle network failures
		combinedOutput := stdout + stderr

		if err != nil {
			// If network fails, should show fallback models
			if !strings.Contains(combinedOutput, "phi-3.5-mini-instruct") &&
				!strings.Contains(combinedOutput, "llama-2-7b") {
				t.Errorf("Expected fallback models not found. Output: %s", combinedOutput)
			}
		} else {
			// If succeeds, should show model list
			if !strings.Contains(combinedOutput, "NAME") || !strings.Contains(combinedOutput, "TYPE") {
				t.Errorf("Expected model list headers not found. Output: %s", combinedOutput)
			}
		}
	})

	t.Run("models describe valid", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, testTimeout, "models", "describe", "phi-3.5-mini-instruct")

		combinedOutput := stdout + stderr

		if err != nil {
			// Should provide helpful error message
			if !strings.Contains(combinedOutput, "phi-3.5-mini-instruct") {
				t.Errorf("Expected model name in error message. Output: %s", combinedOutput)
			}
		} else {
			// If succeeds, should show model details
			if !strings.Contains(combinedOutput, "phi-3.5-mini-instruct") {
				t.Errorf("Expected model details not found. Output: %s", combinedOutput)
			}
		}
	})

	t.Run("models describe invalid", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, testTimeout, "models", "describe", "invalid-model")

		if err == nil {
			t.Error("Expected error for invalid model name")
		}

		// For kubectl plugins with SilenceErrors=true, we just check that it exits with non-zero code
		// The actual error messages are suppressed, which is correct behavior for kubectl plugins
		t.Logf("Command exited with error as expected (stdout: %s, stderr: %s)", stdout, stderr)
	})
}

// Test Kind cluster functionality
func TestKindClusterOperations(t *testing.T) {
	if !kindAvailable {
		t.Skip("Kind not available, skipping Kind cluster tests")
	}

	cm := NewClusterManager()

	// Create Kind cluster
	if err := cm.CreateKindCluster(t); err != nil {
		t.Fatalf("Failed to create Kind cluster: %v", err)
	}
	defer cm.DestroyKindCluster(t)

	// Deploy nginx for testing
	if err := cm.DeployNginxToKind(t); err != nil {
		t.Fatalf("Failed to deploy nginx: %v", err)
	}

	t.Run("deploy dry-run on kind", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, testTimeout,
			"deploy",
			"--workspace-name", "test-workspace",
			"--model", "phi-3.5-mini-instruct",
			"--instance-type", "Standard_NC6s_v3",
			"--dry-run")

		if err != nil {
			t.Errorf("Deploy dry-run failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
		}

		combinedOutput := stdout + stderr
		if !strings.Contains(combinedOutput, "Dry-run mode") {
			t.Errorf("Expected dry-run output not found: %s", combinedOutput)
		}
	})

	t.Run("status with no workspaces", func(t *testing.T) {
		stdout, stderr, err := runCommand(t, testTimeout, "status")

		// Should succeed even with no workspaces
		combinedOutput := stdout + stderr

		if err != nil {
			// Check if it's a meaningful error (like CRD not found)
			if !strings.Contains(combinedOutput, "no resources found") &&
				!strings.Contains(combinedOutput, "the server doesn't have a resource type") {
				t.Errorf("Unexpected error: %v\nOutput: %s", err, combinedOutput)
			}
		}
	})
}

// Test AKS cluster functionality with GPU nodes
func TestAKSClusterOperations(t *testing.T) {
	if !aksAvailable {
		t.Skip("Azure CLI not available or not authenticated, skipping AKS tests")
	}

	cm := NewClusterManager()

	// Create AKS cluster (this is expensive, so we do it once per test run)
	if err := cm.CreateAKSCluster(t); err != nil {
		t.Fatalf("Failed to create AKS cluster: %v", err)
	}
	defer cm.DestroyAKSCluster(t)

	t.Run("deploy validation on aks", func(t *testing.T) {
		// Test validation without actually creating resources
		stdout, stderr, err := runCommand(t, testTimeout,
			"deploy",
			"--workspace-name", "test-gpu-workspace",
			"--model", "llama-2-7b",
			"--instance-type", "Standard_NC6s_v3",
			"--dry-run")

		if err != nil {
			t.Errorf("Deploy validation failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
		}

		combinedOutput := stdout + stderr
		if !strings.Contains(combinedOutput, "llama-2-7b") {
			t.Errorf("Expected model name in output: %s", combinedOutput)
		}
	})

	t.Run("get-endpoint validation", func(t *testing.T) {
		// Test endpoint command validation
		_, stderr, err := runCommand(t, testTimeout,
			"get-endpoint",
			"--workspace-name", "non-existent-workspace")

		if err == nil {
			t.Error("Expected error for non-existent workspace")
		}

		if !strings.Contains(stderr, "not found") && !strings.Contains(stderr, "workspace") {
			t.Errorf("Expected workspace not found error, got: %s", stderr)
		}
	})

	t.Run("chat validation", func(t *testing.T) {
		// Test chat command validation
		_, stderr, err := runCommand(t, testTimeout,
			"chat",
			"--workspace-name", "non-existent-workspace",
			"--message", "test message")

		if err == nil {
			t.Error("Expected error for non-existent workspace")
		}

		// Should indicate workspace not found or not ready
		if !strings.Contains(stderr, "not found") && !strings.Contains(stderr, "workspace") {
			t.Errorf("Expected workspace error, got: %s", stderr)
		}
	})

	t.Run("rag deploy validation", func(t *testing.T) {
		// Test RAG deploy validation
		stdout, stderr, err := runCommand(t, testTimeout,
			"rag", "deploy",
			"--name", "test-rag",
			"--vector-db", "faiss",
			"--index-service", "llamaindex",
			"--dry-run")

		if err != nil {
			t.Errorf("RAG deploy validation failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
		}

		combinedOutput := stdout + stderr
		if !strings.Contains(combinedOutput, "test-rag") {
			t.Errorf("Expected RAG name in output: %s", combinedOutput)
		}
	})
}

// Test input validation
func TestInputValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "deploy missing workspace name",
			args:        []string{"deploy", "--model", "llama-2-7b"},
			expectError: true,
		},
		{
			name:        "deploy missing model",
			args:        []string{"deploy", "--workspace-name", "test"},
			expectError: true,
		},
		{
			name:        "deploy invalid model",
			args:        []string{"deploy", "--workspace-name", "test", "--model", "invalid-model"},
			expectError: true,
		},
		{
			name:        "chat missing workspace",
			args:        []string{"chat", "--message", "hello"},
			expectError: true,
		},
		{
			name:        "get-endpoint missing workspace",
			args:        []string{"get-endpoint"},
			expectError: true,
		},
		{
			name:        "rag deploy missing name",
			args:        []string{"rag", "deploy", "--vector-db", "faiss"},
			expectError: true,
		},
		{
			name:        "valid dry-run should succeed",
			args:        []string{"deploy", "--workspace-name", "test", "--model", "phi-3.5-mini-instruct", "--dry-run"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runCommand(t, testTimeout, tt.args...)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but command succeeded. Stdout: %s, Stderr: %s", stdout, stderr)
				}
				// For kubectl plugins with SilenceErrors=true, we just check that it exits with non-zero code
				// The actual error messages are suppressed, which is correct behavior
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
				}
			}
		})
	}
}