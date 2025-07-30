package cmd

import (
	"context"
	"fmt"
	"os"
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
		klog.Info("No workspaces found")
		return nil
	}

	o.printWorkspaceTable(workspaceList.Items)
	return nil
}

func (o *StatusOptions) watchWorkspace(dynamicClient dynamic.Interface) error {
	klog.V(2).Infof("Starting watch for workspace: %s", o.WorkspaceName)
	klog.Infof("Watching workspace %s for changes (Ctrl+C to stop)...", o.WorkspaceName)
	klog.Info("")

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
			klog.Infof("=== %s at %s ===", strings.ToUpper(string(event.Type)), time.Now().Format(time.RFC3339))
			o.printWorkspaceDetails(workspace)
			klog.Info("")
		}
	}

	return nil
}

func (o *StatusOptions) printWorkspaceTable(workspaces []unstructured.Unstructured) {
	klog.V(4).Info("Printing workspace table")

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	defer w.Flush()

	if o.AllNamespaces {
		fmt.Fprintln(w, "NAMESPACE\tNAME\tINSTANCE\tRESOURCEREADY\tINFERENCEREADY\tWORKSPACEREADY\tAGE")
	} else {
		fmt.Fprintln(w, "NAME\tINSTANCE\tRESOURCEREADY\tINFERENCEREADY\tWORKSPACEREADY\tAGE")
	}

	for _, workspace := range workspaces {
		instanceType := o.getInstanceType(&workspace)
		resourceReady := o.getConditionStatus(&workspace, "ResourceReady")
		inferenceReady := o.getConditionStatus(&workspace, "InferenceReady")
		workspaceReady := o.getConditionStatus(&workspace, "WorkspaceReady")
		age := o.getAge(&workspace)

		if o.AllNamespaces {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				workspace.GetNamespace(), workspace.GetName(), instanceType,
				resourceReady, inferenceReady, workspaceReady, age)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				workspace.GetName(), instanceType,
				resourceReady, inferenceReady, workspaceReady, age)
		}
	}
}

func (o *StatusOptions) printWorkspaceDetails(workspace *unstructured.Unstructured) {
	klog.V(4).Info("Printing workspace details")

	klog.Info("Workspace Details")
	klog.Info("=================")
	klog.Infof("Name: %s", workspace.GetName())
	klog.Infof("Namespace: %s", workspace.GetNamespace())

	// Get instance type and count
	if spec, found, _ := unstructured.NestedMap(workspace.Object, "spec", "resource"); found {
		if instanceType, found, _ := unstructured.NestedString(spec, "instanceType"); found {
			klog.Infof("Instance Type: %s", instanceType)
		}
		if count, found, _ := unstructured.NestedInt64(spec, "count"); found {
			klog.Infof("Node Count: %d", count)
		}
	}

	// Check if tuning or inference
	if _, found, _ := unstructured.NestedMap(workspace.Object, "spec", "tuning"); found {
		klog.Info("Mode: Fine-tuning")
	} else {
		klog.Info("Mode: Inference")
	}

	// Status information
	if status, found, _ := unstructured.NestedMap(workspace.Object, "status"); found {
		klog.V(4).Infof("Status found: %v", status)
	}

	klog.Infof("Age: %s", o.getAge(workspace))
	klog.Info("")
}

func (o *StatusOptions) printConditions(workspace *unstructured.Unstructured) {
	klog.V(4).Info("Printing workspace conditions")

	conditions, found, err := unstructured.NestedSlice(workspace.Object, "status", "conditions")
	if err != nil {
		klog.Errorf("Error getting conditions: %v", err)
		return
	}
	if !found || len(conditions) == 0 {
		klog.Info("Conditions: None")
		return
	}

	klog.Info("Conditions:")

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "  TYPE\tSTATUS\tREASON\tMESSAGE\tLAST TRANSITION")

	for _, condition := range conditions {
		if condMap, ok := condition.(map[string]interface{}); ok {
			condType, _ := condMap["type"].(string)
			status, _ := condMap["status"].(string)
			reason, _ := condMap["reason"].(string)
			message, _ := condMap["message"].(string)
			lastTransitionTime, _ := condMap["lastTransitionTime"].(string)

			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n",
				condType, status, reason, message, lastTransitionTime)
		}
	}

	klog.Info("")
}

func (o *StatusOptions) printWorkerNodes(workspace *unstructured.Unstructured) {
	klog.V(4).Info("Printing worker node information")

	klog.Info("Worker Nodes:")

	// In a real implementation, this would fetch actual worker node information
	// from the workspace status or related resources
	klog.Info("  (Worker node information not available)")
	klog.Info("")
}

func (o *StatusOptions) getInstanceType(workspace *unstructured.Unstructured) string {
	instanceType, found, err := unstructured.NestedString(workspace.Object, "spec", "resource", "instanceType")
	if err != nil || !found {
		klog.V(6).Infof("Instance type not found for workspace %s", workspace.GetName())
		return "Unknown"
	}
	return instanceType
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

	if duration.Seconds() < 60 {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration.Minutes() < 60 {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration.Hours() < 24 {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	}
}
