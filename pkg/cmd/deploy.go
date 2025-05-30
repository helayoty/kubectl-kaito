package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
)

type DeployOptions struct {
	configFlags   *genericclioptions.ConfigFlags
	Name          string
	Model         string
	GPUs          int
	Preset        string
	InstanceType  string
	Namespace     string
	LabelSelector map[string]string
	DryRun        bool
}

func NewDeployCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	o := &DeployOptions{
		configFlags: configFlags,
	}

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy an AI model for inference using Kaito",
		Long: `Deploy an AI model for inference using Kaito workspaces.

This command creates a Kaito workspace that automatically provisions GPU nodes
and sets up the inference server for the specified model.`,
		Example: `  # Deploy llama-3 model with 1 GPU
  kubectl kaito deploy --name workspace-llama-3 --model llama-3-8b-instruct --gpus 1 --preset instruct
  
  # Deploy falcon-7b model with specific instance type
  kubectl kaito deploy --name workspace-falcon-7b --model falcon-7b-instruct --instance-type Standard_NC24ads_A100_v4
  
  # Preview deployment without creating resources
  kubectl kaito deploy --name workspace-test --model llama-2-7b --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVar(&o.Name, "name", "", "Name of the workspace (required)")
	cmd.Flags().StringVar(&o.Model, "model", "", "Model name (required)")
	cmd.Flags().IntVar(&o.GPUs, "gpus", 1, "Number of GPUs")
	cmd.Flags().StringVar(&o.Preset, "preset", "", "Model preset (e.g., instruct, base)")
	cmd.Flags().StringVar(&o.InstanceType, "instance-type", "", "Azure VM instance type for GPU nodes")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "default", "Kubernetes namespace")
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "Preview deployment without creating resources")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("model")

	return cmd
}

func (o *DeployOptions) Complete() error {
	// Get namespace from flags or config
	if o.Namespace == "" {
		if o.configFlags.Namespace != nil && *o.configFlags.Namespace != "" {
			o.Namespace = *o.configFlags.Namespace
		} else {
			o.Namespace = "default"
		}
	}

	// Set default instance type if not provided
	if o.InstanceType == "" {
		o.InstanceType = "Standard_NC24ads_A100_v4"
	}

	// Set default preset if not provided
	if o.Preset == "" {
		o.Preset = "base"
	}

	// Initialize label selector
	o.LabelSelector = map[string]string{
		"apps": o.Model,
	}

	return nil
}

func (o *DeployOptions) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("name is required")
	}
	if o.Model == "" {
		return fmt.Errorf("model is required")
	}
	if o.GPUs < 1 {
		return fmt.Errorf("gpus must be at least 1")
	}
	return nil
}

func (o *DeployOptions) Run() error {
	// Create workspace resource
	workspace := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kaito.sh/v1beta1",
			"kind":       "Workspace",
			"metadata": map[string]interface{}{
				"name":      o.Name,
				"namespace": o.Namespace,
			},
			"spec": map[string]interface{}{
				"resource": map[string]interface{}{
					"count":        o.GPUs,
					"instanceType": o.InstanceType,
					"labelSelector": map[string]interface{}{
						"matchLabels": o.LabelSelector,
					},
				},
				"inference": map[string]interface{}{
					"preset": map[string]interface{}{
						"name": fmt.Sprintf("%s-%s", o.Model, o.Preset),
					},
				},
			},
		},
	}

	// Handle dry-run mode
	if o.DryRun {
		fmt.Printf("🔍 Dry-run mode: Showing what would be created\n\n")
		fmt.Printf("Workspace Configuration:\n")
		fmt.Printf("========================\n")
		fmt.Printf("Name: %s\n", o.Name)
		fmt.Printf("Namespace: %s\n", o.Namespace)
		fmt.Printf("Model: %s-%s\n", o.Model, o.Preset)
		fmt.Printf("GPUs: %d\n", o.GPUs)
		fmt.Printf("Instance Type: %s\n", o.InstanceType)
		fmt.Printf("Label Selector: %v\n", o.LabelSelector)
		fmt.Println()
		fmt.Printf("✓ Workspace definition is valid\n")
		fmt.Printf("ℹ️  Run without --dry-run to create the workspace\n")
		return nil
	}

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

	// Create the workspace
	fmt.Printf("Creating workspace %s in namespace %s...\n", o.Name, o.Namespace)

	_, err = dynamicClient.Resource(gvr).Namespace(o.Namespace).Create(
		context.TODO(),
		workspace,
		metav1.CreateOptions{},
	)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return fmt.Errorf("workspace %s already exists in namespace %s", o.Name, o.Namespace)
		}
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	fmt.Printf("✓ Successfully created workspace %s\n", o.Name)
	fmt.Printf("Model: %s-%s\n", o.Model, o.Preset)
	fmt.Printf("GPUs: %d\n", o.GPUs)
	fmt.Printf("Instance Type: %s\n", o.InstanceType)
	fmt.Printf("Namespace: %s\n", o.Namespace)
	fmt.Println()
	fmt.Printf("Monitor the deployment with: kubectl kaito status %s\n", o.Name)

	return nil
}
