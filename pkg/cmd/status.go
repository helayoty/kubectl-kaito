package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
)

type StatusOptions struct {
	configFlags   *genericclioptions.ConfigFlags
	WorkspaceName string
	Namespace     string
	AllNamespaces bool
	Watch         bool
}

func NewStatusCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	o := &StatusOptions{
		configFlags: configFlags,
	}

	cmd := &cobra.Command{
		Use:   "status [workspace-name]",
		Short: "Check the status of Kaito workspaces",
		Long: `Check the status of Kaito workspaces.

This command shows the current status of workspace deployments, including
resource readiness, inference readiness, and other important information.`,
		Example: `  # Check status of a specific workspace
  kubectl kaito status workspace-llama-3
  
  # Check status with resource type prefix
  kubectl kaito status workspace/workspace-llama-3
  
  # List all workspaces in current namespace
  kubectl kaito status
  
  # List workspaces in all namespaces
  kubectl kaito status --all-namespaces
  
  # Watch workspace status updates
  kubectl kaito status workspace-llama-3 --watch`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				// Parse workspace name, handle "workspace/name" format
				workspaceRef := args[0]
				if strings.Contains(workspaceRef, "/") {
					parts := strings.Split(workspaceRef, "/")
					if len(parts) == 2 && parts[0] == "workspace" {
						o.WorkspaceName = parts[1]
					} else {
						return fmt.Errorf("invalid workspace reference format: %s", workspaceRef)
					}
				} else {
					o.WorkspaceName = workspaceRef
				}
			}

			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "Kubernetes namespace")
	cmd.Flags().BoolVarP(&o.AllNamespaces, "all-namespaces", "A", false, "Show workspaces in all namespaces")
	cmd.Flags().BoolVarP(&o.Watch, "watch", "w", false, "Watch for changes")

	return cmd
}

func (o *StatusOptions) Complete() error {
	// Get namespace from flags or config
	if !o.AllNamespaces && o.Namespace == "" {
		if o.configFlags.Namespace != nil && *o.configFlags.Namespace != "" {
			o.Namespace = *o.configFlags.Namespace
		} else {
			o.Namespace = "default"
		}
	}
	return nil
}

func (o *StatusOptions) Validate() error {
	if o.AllNamespaces && o.Namespace != "" {
		return fmt.Errorf("cannot specify both --namespace and --all-namespaces")
	}
	return nil
}

func (o *StatusOptions) Run() error {
	// Get REST config
	config, err := o.configFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get REST config: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Define GVR for Kaito workspace
	gvr := schema.GroupVersionResource{
		Group:    "kaito.sh",
		Version:  "v1beta1",
		Resource: "workspaces",
	}

	if o.Watch {
		return o.watchWorkspace(dynamicClient, gvr)
	}

	if o.WorkspaceName != "" {
		return o.showWorkspaceStatus(dynamicClient, gvr)
	}

	return o.listWorkspaces(dynamicClient, gvr)
}

func (o *StatusOptions) showWorkspaceStatus(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource) error {
	workspace, err := dynamicClient.Resource(gvr).Namespace(o.Namespace).Get(
		context.TODO(),
		o.WorkspaceName,
		metav1.GetOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to get workspace %s: %w", o.WorkspaceName, err)
	}

	o.printWorkspaceDetail(workspace)
	return nil
}

func (o *StatusOptions) listWorkspaces(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource) error {
	var workspaceList *unstructured.UnstructuredList
	var err error

	if o.AllNamespaces {
		workspaceList, err = dynamicClient.Resource(gvr).List(
			context.TODO(),
			metav1.ListOptions{},
		)
	} else {
		workspaceList, err = dynamicClient.Resource(gvr).Namespace(o.Namespace).List(
			context.TODO(),
			metav1.ListOptions{},
		)
	}

	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	if len(workspaceList.Items) == 0 {
		if o.AllNamespaces {
			fmt.Println("No workspaces found in any namespace.")
		} else {
			fmt.Printf("No workspaces found in namespace %s.\n", o.Namespace)
		}
		return nil
	}

	o.printWorkspacesTable(workspaceList.Items)
	return nil
}

func (o *StatusOptions) watchWorkspace(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource) error {
	if o.WorkspaceName == "" {
		return fmt.Errorf("workspace name is required for watch mode")
	}

	fmt.Printf("Watching workspace %s in namespace %s...\n", o.WorkspaceName, o.Namespace)
	fmt.Println("Press Ctrl+C to stop watching")
	fmt.Println()

	for {
		workspace, err := dynamicClient.Resource(gvr).Namespace(o.Namespace).Get(
			context.TODO(),
			o.WorkspaceName,
			metav1.GetOptions{},
		)
		if err != nil {
			fmt.Printf("Error getting workspace: %v\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		fmt.Printf("\033[2J\033[H") // Clear screen
		fmt.Printf("Last updated: %s\n\n", time.Now().Format("15:04:05"))
		o.printWorkspaceDetail(workspace)

		time.Sleep(5 * time.Second)
	}
}

func (o *StatusOptions) printWorkspacesTable(workspaces []unstructured.Unstructured) {
	// Print header
	if o.AllNamespaces {
		fmt.Printf("%-30s %-15s %-25s %-15s %-15s %-10s %-15s %-10s\n",
			"NAME", "NAMESPACE", "INSTANCE", "RESOURCEREADY", "INFERENCEREADY", "JOBSTARTED", "WORKSPACEREADY", "AGE")
	} else {
		fmt.Printf("%-30s %-25s %-15s %-15s %-10s %-15s %-10s\n",
			"NAME", "INSTANCE", "RESOURCEREADY", "INFERENCEREADY", "JOBSTARTED", "WORKSPACEREADY", "AGE")
	}

	// Print workspaces
	for _, workspace := range workspaces {
		name := workspace.GetName()
		namespace := workspace.GetNamespace()

		// Extract status information
		status, _, _ := unstructured.NestedMap(workspace.Object, "status")
		instanceType := extractStringValue(workspace.Object, "spec", "resource", "instanceType")

		resourceReady := extractConditionStatus(status, "ResourceReady")
		inferenceReady := extractConditionStatus(status, "InferenceReady")
		jobStarted := extractConditionStatus(status, "JobStarted")
		workspaceReady := extractConditionStatus(status, "WorkspaceReady")

		// Calculate age
		creationTime := workspace.GetCreationTimestamp()
		age := time.Since(creationTime.Time).Truncate(time.Second)

		if o.AllNamespaces {
			fmt.Printf("%-30s %-15s %-25s %-15s %-15s %-10s %-15s %-10s\n",
				name, namespace, instanceType, resourceReady, inferenceReady, jobStarted, workspaceReady, age)
		} else {
			fmt.Printf("%-30s %-25s %-15s %-15s %-10s %-15s %-10s\n",
				name, instanceType, resourceReady, inferenceReady, jobStarted, workspaceReady, age)
		}
	}
}

func (o *StatusOptions) printWorkspaceDetail(workspace *unstructured.Unstructured) {
	name := workspace.GetName()
	namespace := workspace.GetNamespace()

	fmt.Printf("Name:      %s\n", name)
	fmt.Printf("Namespace: %s\n", namespace)

	// Extract spec information
	instanceType := extractStringValue(workspace.Object, "spec", "resource", "instanceType")
	if instanceType != "" {
		fmt.Printf("Instance:  %s\n", instanceType)
	}

	// Extract status information
	status, found, _ := unstructured.NestedMap(workspace.Object, "status")
	if !found {
		fmt.Println("Status:    No status available")
		return
	}

	fmt.Println("\nConditions:")
	conditions, found, _ := unstructured.NestedSlice(status, "conditions")
	if found {
		for _, conditionInterface := range conditions {
			condition, ok := conditionInterface.(map[string]interface{})
			if !ok {
				continue
			}

			condType, _, _ := unstructured.NestedString(condition, "type")
			condStatus, _, _ := unstructured.NestedString(condition, "status")
			message, _, _ := unstructured.NestedString(condition, "message")

			fmt.Printf("  %s: %s", condType, condStatus)
			if message != "" {
				fmt.Printf(" (%s)", message)
			}
			fmt.Println()
		}
	}

	// Show events or additional info if available
	fmt.Println()
}

func extractStringValue(obj map[string]interface{}, keys ...string) string {
	value, found, _ := unstructured.NestedString(obj, keys...)
	if !found {
		return ""
	}
	return value
}

func extractConditionStatus(status map[string]interface{}, conditionType string) string {
	conditions, found, _ := unstructured.NestedSlice(status, "conditions")
	if !found {
		return "Unknown"
	}

	for _, conditionInterface := range conditions {
		condition, ok := conditionInterface.(map[string]interface{})
		if !ok {
			continue
		}

		cType, _, _ := unstructured.NestedString(condition, "type")
		if cType == conditionType {
			cStatus, _, _ := unstructured.NestedString(condition, "status")
			return cStatus
		}
	}

	return "Unknown"
}
