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
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

// StatusOptions holds the options for the status command
type StatusOptions struct {
	configFlags *genericclioptions.ConfigFlags

	WorkspaceName    string
	Namespace        string
	AllNamespaces    bool
	ShowConditions   bool
	ShowWorkerNodes  bool
	Watch            bool
}

// NewStatusCmd creates the status command
func NewStatusCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	o := &StatusOptions{
		configFlags: configFlags,
	}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check status of Kaito workspaces",
		Long: `Check the status of one or more Kaito workspaces.

This command displays the current state of workspace resources, including
readiness conditions, resource allocation, and deployment status.`,
		Example: `  # Check status of a specific workspace
  kubectl kaito status --workspace-name my-workspace

  # Check status of all workspaces in current namespace
  kubectl kaito status

  # Check status across all namespaces
  kubectl kaito status --all-namespaces

  # Watch for changes in real-time
  kubectl kaito status --workspace-name my-workspace --watch

  # Show detailed conditions and worker node information
  kubectl kaito status --workspace-name my-workspace --show-conditions --show-worker-nodes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.validate(); err != nil {
				klog.Errorf("Validation failed: %v", err)
				return fmt.Errorf("validation failed: %w", err)
			}
			return o.run()
		},
	}

	cmd.Flags().StringVar(&o.WorkspaceName, "workspace-name", "", "Name of the workspace to check")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "Kubernetes namespace")
	cmd.Flags().BoolVarP(&o.AllNamespaces, "all-namespaces", "A", false, "Check workspaces across all namespaces")
	cmd.Flags().BoolVar(&o.ShowConditions, "show-conditions", false, "Show detailed status conditions")
	cmd.Flags().BoolVar(&o.ShowWorkerNodes, "show-worker-nodes", false, "Show worker node information")
	cmd.Flags().BoolVarP(&o.Watch, "watch", "w", false, "Watch for changes in real-time")

	return cmd
}

func (o *StatusOptions) validate() error {
	klog.V(4).Info("Validating status command options")

	if o.AllNamespaces && o.Namespace != "" {
		return fmt.Errorf("cannot specify both --namespace and --all-namespaces")
	}

	klog.V(4).Info("Status command validation completed successfully")
	return nil
}

func (o *StatusOptions) run() error {
	klog.V(2).Info("Starting status command")

	// Get REST config
	config, err := o.configFlags.ToRESTConfig()
	if err != nil {
		klog.Errorf("Failed to get REST config: %v", err)
		return fmt.Errorf("failed to get REST config: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Errorf("Failed to create dynamic client: %v", err)
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Get namespace
	if o.Namespace == "" && !o.AllNamespaces {
		if ns, _, err := o.configFlags.ToRawKubeConfigLoader().Namespace(); err == nil && ns != "" {
			o.Namespace = ns
		} else {
			klog.V(4).Info("No namespace specified, using 'default'")
			o.Namespace = "default"
		}
	}

	// Handle watch mode for specific workspace
	if o.Watch && o.WorkspaceName != "" {
		return o.watchWorkspace(dynamicClient)
	}

	// Handle specific workspace
	if o.WorkspaceName != "" {
		return o.showWorkspaceStatus(dynamicClient)
	}

	// Handle listing workspaces
	return o.listWorkspaces(dynamicClient)
}

func (o *StatusOptions) showWorkspaceStatus(dynamicClient dynamic.Interface) error {
	klog.V(3).Infof("Getting status for workspace: %s", o.WorkspaceName)

	gvr := schema.GroupVersionResource{
		Group:    "kaito.sh",
		Version:  "v1beta1",
		Resource: "workspaces",
	}

	workspace, err := dynamicClient.Resource(gvr).Namespace(o.Namespace).Get(
		context.TODO(),
		o.WorkspaceName,
		metav1.GetOptions{},
	)
	if err != nil {
		klog.Errorf("Failed to get workspace %s: %v", o.WorkspaceName, err)
		return fmt.Errorf("failed to get workspace %s: %w", o.WorkspaceName, err)
	}

	o.printWorkspaceDetails(workspace)

	if o.ShowConditions {
		o.printConditions(workspace)
	}

	if o.ShowWorkerNodes {
		o.printWorkerNodes(workspace)
	}

	return nil
}

func (o *StatusOptions) listWorkspaces(dynamicClient dynamic.Interface) error {
	klog.V(3).Info("Listing workspaces")

	gvr := schema.GroupVersionResource{
		Group:    "kaito.sh",
		Version:  "v1beta1",
		Resource: "workspaces",
	}

	var workspaceList *unstructured.UnstructuredList
	var err error

	if o.AllNamespaces {
		klog.V(4).Info("Listing workspaces across all namespaces")
		workspaceList, err = dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
	} else {
		klog.V(4).Infof("Listing workspaces in namespace: %s", o.Namespace)
		workspaceList, err = dynamicClient.Resource(gvr).Namespace(o.Namespace).List(context.TODO(), metav1.ListOptions{})
	}

	if err != nil {
		klog.Errorf("Failed to list workspaces: %v", err)
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	if len(workspaceList.Items) == 0 {
		fmt.Println("No workspaces found")
		return nil
	}

	o.printWorkspaceTable(workspaceList.Items)
	return nil
}

func (o *StatusOptions) watchWorkspace(dynamicClient dynamic.Interface) error {
	klog.V(2).Infof("Starting watch for workspace: %s", o.WorkspaceName)
	fmt.Printf("Watching workspace %s for changes (Ctrl+C to stop)...\n", o.WorkspaceName)
	fmt.Println()

	gvr := schema.GroupVersionResource{
		Group:    "kaito.sh",
		Version:  "v1beta1",
		Resource: "workspaces",
	}

	watcher, err := dynamicClient.Resource(gvr).Namespace(o.Namespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", o.WorkspaceName),
	})
	if err != nil {
		klog.Errorf("Failed to watch workspace: %v", err)
		return fmt.Errorf("failed to watch workspace: %w", err)
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		if workspace, ok := event.Object.(*unstructured.Unstructured); ok {
			fmt.Printf("=== %s at %s ===\n", strings.ToUpper(string(event.Type)), time.Now().Format(time.RFC3339))
			o.printWorkspaceDetails(workspace)
			fmt.Println()
		}
	}

	return nil
}

func (o *StatusOptions) printWorkspaceTable(workspaces []unstructured.Unstructured) {
	klog.V(4).Info("Printing workspace table")

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	defer w.Flush()

	if o.AllNamespaces {
		fmt.Fprintln(w, "NAMESPACE\tNAME\tNODECLAIM\tRESOURCEREADY\tINFERENCEREADY\tWORKSPACEREADY\tAGE")
	} else {
		fmt.Fprintln(w, "NAME\tNODECLAIM\tRESOURCEREADY\tINFERENCEREADY\tWORKSPACEREADY\tAGE")
	}

	for _, workspace := range workspaces {
		nodeClaimName := o.getNodeClaimName(&workspace)
		resourceReady := o.getConditionStatus(&workspace, "ResourceReady")
		inferenceReady := o.getConditionStatus(&workspace, "InferenceReady")
		workspaceReady := o.getConditionStatus(&workspace, "WorkspaceReady")
		age := o.getAge(&workspace)

		if o.AllNamespaces {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				workspace.GetNamespace(), workspace.GetName(), nodeClaimName,
				resourceReady, inferenceReady, workspaceReady, age)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				workspace.GetName(), nodeClaimName,
				resourceReady, inferenceReady, workspaceReady, age)
		}
	}
}

func (o *StatusOptions) printWorkspaceDetails(workspace *unstructured.Unstructured) {
	klog.V(4).Info("Printing workspace details")

	fmt.Println("Workspace Details")
	fmt.Println("=================")
	fmt.Printf("Name: %s\n", workspace.GetName())
	fmt.Printf("Namespace: %s\n", workspace.GetNamespace())

	o.printResourceDetails(workspace)
	o.printWorkspaceMode(workspace)
	o.printDeploymentStatus(workspace)

	fmt.Printf("Age: %s\n", o.getAge(workspace))
	fmt.Println()
}

func (o *StatusOptions) printResourceDetails(workspace *unstructured.Unstructured) {
	// Get instance type and count from the top-level resource section (not spec.resource)
	if resource, found := workspace.Object["resource"]; found {
		if resourceMap, ok := resource.(map[string]interface{}); ok {
			o.printInstanceDetails(resourceMap)
			o.printPreferredNodes(resourceMap)
			o.printNodeSelector(resourceMap)
		}
	}
}

func (o *StatusOptions) printInstanceDetails(resourceMap map[string]interface{}) {
	if instanceType, found := resourceMap["instanceType"]; found {
		fmt.Printf("Instance Type: %s\n", instanceType)
	}
	if count, found := resourceMap["count"]; found {
		fmt.Printf("Node Count: %v\n", count)
	}
}

func (o *StatusOptions) printPreferredNodes(resourceMap map[string]interface{}) {
	// Display preferred nodes if available
	if preferredNodes, found := resourceMap["preferredNodes"]; found {
		if nodeList, ok := preferredNodes.([]interface{}); ok && len(nodeList) > 0 {
			fmt.Print("Preferred Nodes: ")
			for i, node := range nodeList {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(node)
			}
			fmt.Println()
		}
	}
}

func (o *StatusOptions) printNodeSelector(resourceMap map[string]interface{}) {
	// Display node selector if available (alternative way to specify preferred nodes)
	if labelSelector, found := resourceMap["labelSelector"]; found {
		if labelMap, ok := labelSelector.(map[string]interface{}); ok {
			if matchLabels, found := labelMap["matchLabels"]; found {
				if labels, ok := matchLabels.(map[string]interface{}); ok && len(labels) > 0 {
					fmt.Print("Node Selector: ")
					first := true
					for key, value := range labels {
						if !first {
							fmt.Print(", ")
						}
						fmt.Printf("%s=%v", key, value)
						first = false
					}
					fmt.Println()
				}
			}
		}
	}
}

func (o *StatusOptions) printWorkspaceMode(workspace *unstructured.Unstructured) {
	// Check if tuning or inference (top-level, not spec.tuning)
	if _, found := workspace.Object["tuning"]; found {
		fmt.Println("Mode: Fine-tuning")
	} else {
		fmt.Println("Mode: Inference")
	}
}

func (o *StatusOptions) printDeploymentStatus(workspace *unstructured.Unstructured) {
	fmt.Println()
	fmt.Println("Deployment Status:")
	fmt.Println("==================")

	// Get status information
	status, found := workspace.Object["status"]
	if !found {
		fmt.Println("Status: Not Available")
		return
	}

	statusMap, ok := status.(map[string]interface{})
	if !ok {
		fmt.Println("Status: Invalid Format")
		return
	}

	// Print condition statuses
	if conditions, found := statusMap["conditions"]; found {
		if condList, ok := conditions.([]interface{}); ok {
			resourceReady := "Unknown"
			inferenceReady := "Unknown"
			workspaceReady := "Unknown"

			for _, condition := range condList {
				if condMap, ok := condition.(map[string]interface{}); ok {
					condType, _ := condMap["type"].(string)
					condStatus, _ := condMap["status"].(string)
					
					switch condType {
					case "ResourceReady":
						resourceReady = condStatus
					case "InferenceReady":
						inferenceReady = condStatus
					case "WorkspaceSucceeded":
						workspaceReady = condStatus
					}
				}
			}

			fmt.Printf("Resource Ready: %s\n", resourceReady)
			fmt.Printf("Inference Ready: %s\n", inferenceReady)
			fmt.Printf("Workspace Ready: %s\n", workspaceReady)
		}
	}

	// Print worker nodes if available
	if workerNodes, found := statusMap["workerNodes"]; found {
		if nodeList, ok := workerNodes.([]interface{}); ok && len(nodeList) > 0 {
			fmt.Println()
			fmt.Println("Worker Nodes:")
			for _, node := range nodeList {
				fmt.Printf("  %v\n", node)
			}
		}
	}

	// Print detailed conditions
	fmt.Println()
	o.printConditions(workspace)
}

func (o *StatusOptions) printConditions(workspace *unstructured.Unstructured) {
	klog.V(4).Info("Printing workspace conditions")

	conditions, found, err := unstructured.NestedSlice(workspace.Object, "status", "conditions")
	if err != nil {
		klog.Errorf("Error getting conditions: %v", err)
		return
	}
	if !found || len(conditions) == 0 {
		fmt.Println("Detailed Conditions: None")
		return
	}

	fmt.Println("Detailed Conditions:")

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "  STATUS\tMESSAGE\tLAST TRANSITION")

	for _, condition := range conditions {
		if condMap, ok := condition.(map[string]interface{}); ok {
			status, _ := condMap["status"].(string)
			message, _ := condMap["message"].(string)
			lastTransitionTime, _ := condMap["lastTransitionTime"].(string)

			fmt.Fprintf(w, "  %s\t%s\t%s\n",
				status, message, lastTransitionTime)
		}
	}

	fmt.Println()
}

func (o *StatusOptions) printWorkerNodes(workspace *unstructured.Unstructured) {
	klog.V(4).Info("Printing worker node information")

	fmt.Println("Worker Nodes:")

	// Check if worker nodes are available in the status
	if status, found := workspace.Object["status"]; found {
		if statusMap, ok := status.(map[string]interface{}); ok {
			if workerNodes, found := statusMap["workerNodes"]; found {
				if nodeList, ok := workerNodes.([]interface{}); ok && len(nodeList) > 0 {
					for _, node := range nodeList {
						fmt.Printf("  %v\n", node)
					}
				} else {
					fmt.Println("  (No worker nodes provisioned yet)")
				}
			} else {
				fmt.Println("  (Worker node information not available)")
			}
		}
	} else {
		fmt.Println("  (Workspace status not available)")
	}
	fmt.Println()
}

func (o *StatusOptions) getInstanceType(workspace *unstructured.Unstructured) string {
	instanceType, found, err := unstructured.NestedString(workspace.Object, "resource", "instanceType")
	if err != nil || !found {
		klog.V(6).Infof("Instance type not found for workspace %s", workspace.GetName())
		return "Unknown"
	}
	return instanceType
}

func (o *StatusOptions) getNodeClaimName(workspace *unstructured.Unstructured) string {
	conditions, found, err := unstructured.NestedSlice(workspace.Object, "status", "conditions")
	if err != nil || !found {
		klog.V(6).Infof("Conditions not found for workspace %s", workspace.GetName())
		return "Unknown"
	}

	// Look for NodeClaim name in condition messages
	for _, condition := range conditions {
		if condMap, ok := condition.(map[string]interface{}); ok {
			condType, _ := condMap["type"].(string)
			message, _ := condMap["message"].(string)
			reason, _ := condMap["reason"].(string)
			
			// Check NodeClaimReady condition first
			if condType == "NodeClaimReady" {
				// Try to extract NodeClaim name from message
				if nodeClaimName := extractNodeClaimFromText(message); nodeClaimName != "" {
					return nodeClaimName
				}
				// Try to extract NodeClaim name from reason
				if nodeClaimName := extractNodeClaimFromText(reason); nodeClaimName != "" {
					return nodeClaimName
				}
			}
		}
	}

	// If NodeClaim not found in NodeClaimReady condition, check other conditions
	for _, condition := range conditions {
		if condMap, ok := condition.(map[string]interface{}); ok {
			message, _ := condMap["message"].(string)
			reason, _ := condMap["reason"].(string)
			
			// Try to extract NodeClaim name from any condition message
			if nodeClaimName := extractNodeClaimFromText(message); nodeClaimName != "" {
				return nodeClaimName
			}
			// Try to extract NodeClaim name from any condition reason
			if nodeClaimName := extractNodeClaimFromText(reason); nodeClaimName != "" {
				return nodeClaimName
			}
		}
	}

	return "Unknown"
}

// extractNodeClaimFromText extracts NodeClaim name from text like "nodeClaim ws9cdafdaa5 is not ready"
func extractNodeClaimFromText(text string) string {
	if text == "" {
		return ""
	}
	
	// Common patterns:
	// "nodeClaim wsf30f0c090 is not ready"
	// "check nodeClaim status timed out. nodeClaim ws9cdafdaa5 is not ready"
	// "NodeClaim.karpenter.sh \"wsb80fa0bee\" not found"
	
	// Look for NodeClaim names that typically start with "ws" followed by alphanumeric characters
	// This is more specific than just looking for any word after "nodeClaim"
	re := regexp.MustCompile(`nodeClaim\s+(ws[a-zA-Z0-9]+)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	
	// Look for NodeClaim.karpenter.sh "name" pattern
	re = regexp.MustCompile(`NodeClaim\.karpenter\.sh\s+"([^"]+)"`)
	matches = re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	
	// Fallback: look for any alphanumeric string that looks like a NodeClaim ID after "nodeClaim"
	// but exclude common words like "status", "plugins", etc.
	re = regexp.MustCompile(`nodeClaim\s+([a-zA-Z0-9]{8,})`)
	matches = re.FindStringSubmatch(text)
	if len(matches) > 1 {
		name := matches[1]
		// Exclude common words that are not NodeClaim names
		excludeWords := map[string]bool{
			"status": true,
			"plugins": true,
			"ready": true,
			"pending": true,
			"failed": true,
		}
		if !excludeWords[strings.ToLower(name)] {
			return name
		}
	}
	
	return ""
}

func (o *StatusOptions) getConditionStatus(workspace *unstructured.Unstructured, conditionType string) string {
	conditions, found, err := unstructured.NestedSlice(workspace.Object, "status", "conditions")
	if err != nil || !found {
		klog.V(6).Infof("Conditions not found for workspace %s", workspace.GetName())
		return "Unknown"
	}

	for _, condition := range conditions {
		if condMap, ok := condition.(map[string]interface{}); ok {
			if cType, ok := condMap["type"].(string); ok && cType == conditionType {
				if status, ok := condMap["status"].(string); ok {
					return status
				}
			}
		}
	}

	return "Unknown"
}

func (o *StatusOptions) getAge(workspace *unstructured.Unstructured) string {
	creationTimestamp := workspace.GetCreationTimestamp()
	if creationTimestamp.IsZero() {
		klog.V(6).Infof("Creation timestamp not found for workspace %s", workspace.GetName())
		return "Unknown"
	}

	duration := time.Since(creationTimestamp.Time)

	switch {
	case duration.Seconds() < 60:
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	case duration.Minutes() < 60:
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	case duration.Hours() < 24:
		return fmt.Sprintf("%dh", int(duration.Hours()))
	default:
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	}
}
