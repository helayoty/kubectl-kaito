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
	"strings"
	"testing"
)

// TestAKSClusterOperations tests kubectl-kaito plugin functionality on AKS cluster
func TestAKSClusterOperations(t *testing.T) {
	// Verify we're connected to the right cluster
	if !isAKSCluster(t) {
		t.Skip("Not connected to AKS cluster, skipping AKS-specific tests")
	}

	t.Run("verify_cluster_ready", func(t *testing.T) {
		verifyClusterReady(t)
	})

	t.Run("verify_kaito_operator", func(t *testing.T) {
		verifyKaitoOperator(t)
	})

	t.Run("verify_gpu_nodes", func(t *testing.T) {
		verifyGPUNodes(t)
	})

	t.Run("deploy_validation", func(t *testing.T) {
		testAKSDeployValidation(t)
	})

	t.Run("status_no_workspaces", func(t *testing.T) {
		testStatusNoWorkspaces(t)
	})

	t.Run("get_endpoint_no_workspace", func(t *testing.T) {
		testGetEndpointNoWorkspace(t)
	})

	t.Run("models_list", func(t *testing.T) {
		testModelsList(t)
	})

	t.Run("models_describe", func(t *testing.T) {
		testModelsDescribe(t)
	})

	// TODO: Uncomment when RAG command is available
	// t.Run("rag_deploy_dry_run", func(t *testing.T) {
	// 	testRAGDeployDryRun(t)
	// })

	t.Run("chat_validation", func(t *testing.T) {
		testChatValidation(t)
	})

	t.Run("help_commands", func(t *testing.T) {
		testHelpCommands(t)
	})

	t.Run("input_validation", func(t *testing.T) {
		testInputValidation(t)
	})
}

// isAKSCluster checks if we're connected to an AKS cluster
func isAKSCluster(t *testing.T) bool {
	stdout, err := runKubectlCommand(t, testTimeout, "config", "current-context")
	if err != nil {
		t.Logf("Failed to get current context: %v", err)
		return false
	}

	// AKS clusters typically don't have "kind-" prefix and often contain azure-related names
	if len(stdout) > 5 && stdout[:5] == "kind-" {
		return false
	}

	// Additional check: AKS clusters usually have azure-related API server URLs
	stdout, err = runKubectlCommand(t, testTimeout, "cluster-info")
	if err != nil {
		t.Logf("Failed to get cluster info: %v", err)
		return false
	}

	// AKS clusters have API server URLs containing "azmk8s.io"
	return strings.Contains(stdout, "azmk8s.io")
}

// verifyGPUNodes checks if the cluster has GPU nodes
func verifyGPUNodes(t *testing.T) {
	stdout, err := runKubectlCommand(t, testTimeout, "get", "nodes", "-o", "wide")
	if err != nil {
		t.Errorf("Failed to get nodes: %v\nStdout: %s", err, stdout)
		return
	}

	// Check if any nodes have GPU instance types
	if !strings.Contains(stdout, "Standard_NC") && !strings.Contains(stdout, "Standard_ND") && !strings.Contains(stdout, "Standard_NV") {
		t.Logf("Warning: No GPU nodes detected in cluster. Output: %s", stdout)
	} else {
		t.Logf("✅ GPU nodes detected in cluster")
	}
}

// testAKSDeployValidation tests deployment validation on AKS with GPU instance types
func testAKSDeployValidation(t *testing.T) {
	stdout, stderr, err := runCommand(t, testTimeout,
		"deploy",
		"--workspace-name", "test-gpu-workspace",
		"--model", "phi-2",
		"--instance-type", "Standard_NC6s_v3",
		"--dry-run")

	if err != nil {
		t.Errorf("Deploy validation failed: %v\nStdout: %s\nStderr: %s", err, stdout, stderr)
		return
	}

	combinedOutput := stdout + stderr
	if !strings.Contains(combinedOutput, "Dry-run mode") {
		t.Errorf("Expected dry-run output not found: %s", combinedOutput)
	}

	if !strings.Contains(combinedOutput, "Standard_NC6s_v3") {
		t.Errorf("Expected GPU instance type not found in output: %s", combinedOutput)
	}

	t.Logf("✅ AKS deploy validation successful")
}

// testChatValidation tests chat command validation (without actual chat)
func testChatValidation(t *testing.T) {
	// Test missing workspace
	_, _, err := runCommand(t, testTimeout, "chat", "--message", "hello")
	if err == nil {
		t.Errorf("Expected error for missing workspace, but command succeeded")
	}
	t.Logf("✅ Chat validation correctly requires workspace")

	// Test with workspace but no endpoint (should fail gracefully)
	_, _, err = runCommand(t, testTimeout,
		"chat",
		"--workspace-name", "nonexistent-workspace",
		"--message", "hello")
	if err == nil {
		t.Errorf("Expected error for nonexistent workspace, but command succeeded")
	}
	t.Logf("✅ Chat validation correctly handles nonexistent workspace")
}
