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

type TuneOptions struct {
	configFlags   *genericclioptions.ConfigFlags
	Name          string
	BaseModel     string
	Dataset       string
	Preset        string
	InstanceType  string
	Namespace     string
	LabelSelector map[string]string
	DryRun        bool
}

func NewTuneCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	o := &TuneOptions{
		configFlags: configFlags,
	}

	cmd := &cobra.Command{
		Use:   "tune",
		Short: "Fine-tune an AI model using Kaito",
		Long: `Fine-tune an AI model using Kaito workspaces.

This command creates a Kaito workspace for fine-tuning an existing model
with your custom dataset.`,
		Example: `  # Fine-tune llama-2 model with custom dataset
  kubectl kaito tune --name workspace-llama-2-tune --model llama-2-7b --dataset gs://teamA-ds --preset qlora
  
  # Fine-tune with specific instance type
  kubectl kaito tune --name my-tuned-model --model falcon-7b --dataset s3://my-bucket/data --instance-type Standard_NC24ads_A100_v4
  
  # Preview fine-tuning configuration
  kubectl kaito tune --name test-tune --model phi-2 --dataset gs://test-data --preset lora --dry-run`,
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
	cmd.Flags().StringVar(&o.BaseModel, "model", "", "Base model name (required)")
	cmd.Flags().StringVar(&o.Dataset, "dataset", "", "Dataset location (required)")
	cmd.Flags().StringVar(&o.Preset, "preset", "qlora", "Fine-tuning preset (default: qlora)")
	cmd.Flags().StringVar(&o.InstanceType, "instance-type", "", "Azure VM instance type for GPU nodes")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "default", "Kubernetes namespace")
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "Preview fine-tuning configuration without creating resources")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("model")
	_ = cmd.MarkFlagRequired("dataset")

	return cmd
}

func (o *TuneOptions) Complete() error {
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

	// Initialize label selector
	o.LabelSelector = map[string]string{
		"apps": fmt.Sprintf("%s-tune", o.BaseModel),
	}

	return nil
}

func (o *TuneOptions) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("name is required")
	}
	if o.BaseModel == "" {
		return fmt.Errorf("model is required")
	}
	if o.Dataset == "" {
		return fmt.Errorf("dataset is required")
	}
	return nil
}

func (o *TuneOptions) Run() error {
	// Create workspace resource for fine-tuning
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
					"count":        1,
					"instanceType": o.InstanceType,
					"labelSelector": map[string]interface{}{
						"matchLabels": o.LabelSelector,
					},
				},
				"tuning": map[string]interface{}{
					"preset": map[string]interface{}{
						"name": o.Preset,
					},
					"method": "qlora",
					"input": map[string]interface{}{
						"urls": []string{o.Dataset},
					},
					"output": map[string]interface{}{
						"adapters": map[string]interface{}{
							"enabled": true,
						},
					},
				},
			},
		},
	}

	// Handle dry-run mode
	if o.DryRun {
		fmt.Printf("🔍 Dry-run mode: Showing what would be created for fine-tuning\n\n")
		fmt.Printf("Fine-tuning Workspace Configuration:\n")
		fmt.Printf("====================================\n")
		fmt.Printf("Name: %s\n", o.Name)
		fmt.Printf("Namespace: %s\n", o.Namespace)
		fmt.Printf("Base Model: %s\n", o.BaseModel)
		fmt.Printf("Dataset: %s\n", o.Dataset)
		fmt.Printf("Preset: %s\n", o.Preset)
		fmt.Printf("Instance Type: %s\n", o.InstanceType)
		fmt.Printf("Label Selector: %v\n", o.LabelSelector)
		fmt.Println()
		fmt.Printf("✓ Fine-tuning workspace definition is valid\n")
		fmt.Printf("ℹ️  Run without --dry-run to start fine-tuning\n")
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
	fmt.Printf("Creating fine-tuning workspace %s in namespace %s...\n", o.Name, o.Namespace)

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

	fmt.Printf("✓ Successfully created fine-tuning workspace %s\n", o.Name)
	fmt.Printf("Base Model: %s\n", o.BaseModel)
	fmt.Printf("Dataset: %s\n", o.Dataset)
	fmt.Printf("Preset: %s\n", o.Preset)
	fmt.Printf("Instance Type: %s\n", o.InstanceType)
	fmt.Printf("Namespace: %s\n", o.Namespace)
	fmt.Println()
	fmt.Printf("Monitor the fine-tuning with: kubectl kaito status %s\n", o.Name)

	return nil
}
