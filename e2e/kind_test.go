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
	"testing"
)

// TestKindClusterOperations tests kubectl-kaito plugin functionality on Kind cluster
func TestKindClusterOperations(t *testing.T) {
	// Verify we're connected to the right cluster
	if !isKindCluster(t) {
		t.Skip("Not connected to Kind cluster, skipping Kind-specific tests")
	}

	t.Run("verify_cluster_ready", func(t *testing.T) {
		verifyClusterReady(t)
	})

	t.Run("verify_kaito_operator", func(t *testing.T) {
		verifyKaitoOperator(t)
	})

	t.Run("deploy_dry_run", func(t *testing.T) {
		testDeployDryRun(t)
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

	t.Run("rag_deploy_dry_run", func(t *testing.T) {
		testRAGDeployDryRun(t)
	})

	t.Run("help_commands", func(t *testing.T) {
		testHelpCommands(t)
	})

	t.Run("input_validation", func(t *testing.T) {
		testInputValidation(t)
	})
}

// isKindCluster checks if we're connected to a Kind cluster
func isKindCluster(t *testing.T) bool {
	stdout, err := runKubectlCommand(t, testTimeout, "config", "current-context")
	if err != nil {
		t.Logf("Failed to get current context: %v", err)
		return false
	}

	// Kind clusters have context names like "kind-clustername"
	return len(stdout) > 5 && stdout[:5] == "kind-"
}
