package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
)

type DeleteOptions struct {
	configFlags   *genericclioptions.ConfigFlags
	WorkspaceName string
	Namespace     string
	All           bool
	Force         bool
}

func NewDeleteCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	o := &DeleteOptions{
		configFlags: configFlags,
	}

	cmd := &cobra.Command{
		Use:   "delete <workspace-name>",
		Short: "Delete Kaito workspaces",
		Long: `Delete Kaito workspaces.

This command removes Kaito workspaces and their associated resources.
The GPU nodes provisioned by the workspace will also be cleaned up.`,
		Example: `  # Delete a specific workspace
  kubectl kaito delete workspace-llama-3
  
  # Delete with resource type prefix
  kubectl kaito delete workspace/workspace-llama-3
  
  # Delete all workspaces in current namespace
  kubectl kaito delete --all
  
  # Force delete without confirmation
  kubectl kaito delete workspace-llama-3 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !o.All && len(args) != 1 {
				return fmt.Errorf("workspace name is required (or use --all to delete all workspaces)")
			}

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

	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "default", "Kubernetes namespace")
	cmd.Flags().BoolVar(&o.All, "all", false, "Delete all workspaces in the namespace")
	cmd.Flags().BoolVar(&o.Force, "force", false, "Skip confirmation prompt")

	return cmd
}

func (o *DeleteOptions) Complete() error {
	// Get namespace from flags or config
	if o.Namespace == "" {
		if o.configFlags.Namespace != nil && *o.configFlags.Namespace != "" {
			o.Namespace = *o.configFlags.Namespace
		} else {
			o.Namespace = "default"
		}
	}
	return nil
}

func (o *DeleteOptions) Validate() error {
	if !o.All && o.WorkspaceName == "" {
		return fmt.Errorf("workspace name is required when not using --all")
	}
	if o.All && o.WorkspaceName != "" {
		return fmt.Errorf("cannot specify workspace name when using --all")
	}
	return nil
}

func (o *DeleteOptions) Run() error {
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

	if o.All {
		return o.deleteAllWorkspaces(dynamicClient, gvr)
	}

	return o.deleteSingleWorkspace(dynamicClient, gvr)
}

func (o *DeleteOptions) deleteSingleWorkspace(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource) error {
	// Check if workspace exists first
	_, err := dynamicClient.Resource(gvr).Namespace(o.Namespace).Get(
		context.TODO(),
		o.WorkspaceName,
		metav1.GetOptions{},
	)
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("workspace %s not found in namespace %s", o.WorkspaceName, o.Namespace)
		}
		return fmt.Errorf("failed to get workspace %s: %w", o.WorkspaceName, err)
	}

	// Ask for confirmation unless forced
	if !o.Force {
		fmt.Printf("Are you sure you want to delete workspace %s in namespace %s? (y/N): ", o.WorkspaceName, o.Namespace)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Delete operation cancelled.")
			return nil
		}
	}

	// Delete the workspace
	fmt.Printf("Deleting workspace %s...\n", o.WorkspaceName)
	err = dynamicClient.Resource(gvr).Namespace(o.Namespace).Delete(
		context.TODO(),
		o.WorkspaceName,
		metav1.DeleteOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to delete workspace %s: %w", o.WorkspaceName, err)
	}

	fmt.Printf("✓ Successfully deleted workspace %s\n", o.WorkspaceName)
	fmt.Println("Note: Associated GPU nodes will be cleaned up automatically by Kaito.")

	return nil
}

func (o *DeleteOptions) deleteAllWorkspaces(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource) error {
	// List all workspaces in the namespace
	workspaceList, err := dynamicClient.Resource(gvr).Namespace(o.Namespace).List(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	if len(workspaceList.Items) == 0 {
		fmt.Printf("No workspaces found in namespace %s.\n", o.Namespace)
		return nil
	}

	// Ask for confirmation unless forced
	if !o.Force {
		fmt.Printf("Are you sure you want to delete all %d workspace(s) in namespace %s? (y/N): ",
			len(workspaceList.Items), o.Namespace)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Delete operation cancelled.")
			return nil
		}
	}

	// Delete all workspaces
	fmt.Printf("Deleting %d workspace(s)...\n", len(workspaceList.Items))

	for _, workspace := range workspaceList.Items {
		name := workspace.GetName()
		fmt.Printf("Deleting workspace %s...\n", name)

		err = dynamicClient.Resource(gvr).Namespace(o.Namespace).Delete(
			context.TODO(),
			name,
			metav1.DeleteOptions{},
		)
		if err != nil {
			fmt.Printf("Failed to delete workspace %s: %v\n", name, err)
			continue
		}

		fmt.Printf("✓ Successfully deleted workspace %s\n", name)
	}

	fmt.Println("Note: Associated GPU nodes will be cleaned up automatically by Kaito.")

	return nil
}
