package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

// DeployOptions holds the options for the deploy command
type DeployOptions struct {
	configFlags *genericclioptions.ConfigFlags

	// Common fields
	WorkspaceName string
	Namespace     string
	Model         string
	InstanceType  string
	Count         int
	DryRun        bool

	// Inference specific
	ModelAccessSecret string
	Adapters          []string
	InferenceConfig   string
	PreferredNodes    []string
	LabelSelector     map[string]string

	// Special options
	BypassResourceChecks bool
	EnableLoadBalancer   bool

	// Tuning specific
	Tuning            bool
	TuningMethod      string
	InputURLs         []string
	OutputImage       string
	OutputImageSecret string
	TuningConfig      string
	InputPVC          string
	OutputPVC         string
	ModelAccessMode   string
	ModelImage        string
}

// NewDeployCmd creates the deploy command
func NewDeployCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	o := &DeployOptions{
		configFlags: configFlags,
	}

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a Kaito workspace for model inference or fine-tuning",
		Long: `Deploy creates a new Kaito workspace resource for AI model deployment.

This command supports both inference and fine-tuning scenarios:
- Inference: Deploy models for real-time inference with OpenAI-compatible APIs
- Tuning: Fine-tune existing models with your own datasets using methods like QLoRA

The workspace will automatically provision the required GPU resources and deploy
the specified model according to Kaito's preset configurations.`,
		Example: `  # Deploy Llama-2 7B for inference
  kubectl kaito deploy --workspace-name llama-workspace --model llama-2-7b

  # Deploy with specific instance type and count  
  kubectl kaito deploy --workspace-name phi-workspace --model phi-3.5-mini-instruct --instance-type Standard_NC6s_v3 --count 2

  # Deploy for fine-tuning with QLoRA
  kubectl kaito deploy --workspace-name tune-phi --model phi-3.5-mini-instruct --tuning --tuning-method qlora --input-urls "https://example.com/data.parquet" --output-image myregistry/phi-finetuned:latest

  # Deploy with load balancer for external access
  kubectl kaito deploy --workspace-name public-llama --model llama-2-7b --enable-load-balancer`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Validate(); err != nil {
				klog.Errorf("Validation failed: %v", err)
				return fmt.Errorf("validation failed: %w", err)
			}
			return o.Run()
		},
	}

	// Required flags
	cmd.Flags().StringVar(&o.WorkspaceName, "workspace-name", "", "Name of the workspace to create (required)")
	cmd.Flags().StringVar(&o.Model, "model", "", "Model name to deploy (required)")

	// Resource configuration
	cmd.Flags().StringVar(&o.InstanceType, "instance-type", "", "GPU instance type (e.g., Standard_NC6s_v3)")
	cmd.Flags().IntVar(&o.Count, "count", 1, "Number of GPU nodes")
	cmd.Flags().StringToStringVar(&o.LabelSelector, "node-selector", nil, "Node selector labels")

	// Inference specific flags
	cmd.Flags().StringVar(&o.ModelAccessSecret, "model-access-secret", "", "Secret for private model access")
	cmd.Flags().StringSliceVar(&o.Adapters, "adapters", nil, "Model adapters to load")

	// Tuning specific flags
	cmd.Flags().BoolVar(&o.Tuning, "tuning", false, "Enable fine-tuning mode")
	cmd.Flags().StringVar(&o.TuningMethod, "tuning-method", "qlora", "Fine-tuning method (qlora, lora)")
	cmd.Flags().StringSliceVar(&o.InputURLs, "input-urls", nil, "URLs to training data")
	cmd.Flags().StringVar(&o.OutputImage, "output-image", "", "Output image for fine-tuned model")
	cmd.Flags().StringVar(&o.OutputImageSecret, "output-image-secret", "", "Secret for pushing output image")

	// Special options
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", false, "Show what would be created without actually creating")
	cmd.Flags().BoolVar(&o.BypassResourceChecks, "bypass-resource-checks", false, "Skip resource availability checks")
	cmd.Flags().BoolVar(&o.EnableLoadBalancer, "enable-load-balancer", false, "Create LoadBalancer service for external access")

	// Mark required flags
	if err := cmd.MarkFlagRequired("workspace-name"); err != nil {
		klog.Errorf("Failed to mark workspace-name flag as required: %v", err)
	}
	if err := cmd.MarkFlagRequired("model"); err != nil {
		klog.Errorf("Failed to mark model flag as required: %v", err)
	}

	return cmd
}

// Validate validates the deploy options
func (o *DeployOptions) Validate() error {
	klog.V(4).Info("Validating deploy options")

	if o.WorkspaceName == "" {
		return fmt.Errorf("workspace name is required")
	}
	if o.Model == "" {
		return fmt.Errorf("model name is required")
	}

	// Validate model name against official Kaito supported models
	if err := ValidateModelName(o.Model); err != nil {
		klog.Errorf("Model validation failed: %v", err)
		return err
	}

	// Validate tuning specific requirements
	if o.Tuning {
		if len(o.InputURLs) == 0 && o.InputPVC == "" {
			return fmt.Errorf("tuning mode requires either --input-urls or --input-pvc")
		}
		if o.OutputImage == "" && o.OutputPVC == "" {
			return fmt.Errorf("tuning mode requires either --output-image or --output-pvc")
		}
	}

	klog.V(4).Info("Deploy options validation completed successfully")
	return nil
}

// Run executes the deploy command
func (o *DeployOptions) Run() error {
	klog.V(2).Infof("Starting deploy command for workspace: %s", o.WorkspaceName)

	if err := o.Validate(); err != nil {
		klog.Errorf("Validation failed: %v", err)
		return fmt.Errorf("validation failed: %w", err)
	}

	// Get namespace from config flags if not set
	if o.Namespace == "" {
		if ns, _, err := o.configFlags.ToRawKubeConfigLoader().Namespace(); err == nil && ns != "" {
			o.Namespace = ns
		} else {
			klog.V(4).Info("No namespace specified, using 'default'")
			o.Namespace = "default"
		}
	}

	if o.DryRun {
		return o.showDryRun()
	}

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

	// Create workspace
	workspace := o.buildWorkspace()

	klog.V(2).Infof("Creating workspace %s in namespace %s", o.WorkspaceName, o.Namespace)

	gvr := schema.GroupVersionResource{
		Group:    "kaito.sh",
		Version:  "v1beta1",
		Resource: "workspaces",
	}

	_, err = dynamicClient.Resource(gvr).Namespace(o.Namespace).Create(
		context.TODO(),
		workspace,
		metav1.CreateOptions{},
	)

	if err != nil {
		if errors.IsAlreadyExists(err) {
			klog.Warningf("Workspace %s already exists", o.WorkspaceName)
			klog.Info("✓ Workspace already exists")
			return nil
		}
		klog.Errorf("Failed to create workspace: %v", err)
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	klog.Infof("✓ Workspace %s created successfully", o.WorkspaceName)
	klog.Infof("ℹ️  Use 'kubectl kaito status --workspace-name %s' to check status", o.WorkspaceName)
	return nil
}

func (o *DeployOptions) buildWorkspace() *unstructured.Unstructured {
	klog.V(4).Info("Building workspace configuration")

	workspace := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kaito.sh/v1beta1",
			"kind":       "Workspace",
			"metadata": map[string]interface{}{
				"name":      o.WorkspaceName,
				"namespace": o.Namespace,
			},
			"spec": o.createWorkspaceSpec(),
		},
	}

	return workspace
}

func (o *DeployOptions) createWorkspaceSpec() map[string]interface{} {
	klog.V(4).Info("Creating workspace specification")

	spec := map[string]interface{}{
		"resource": map[string]interface{}{
			"instanceType": o.InstanceType,
		},
	}

	// Add node count if specified
	if o.Count > 0 {
		spec["resource"].(map[string]interface{})["count"] = o.Count
		klog.V(4).Infof("Set node count to %d", o.Count)
	}

	// Add label selector if specified
	if len(o.LabelSelector) > 0 {
		labelSelector := map[string]interface{}{
			"matchLabels": o.LabelSelector,
		}
		spec["resource"].(map[string]interface{})["labelSelector"] = labelSelector
		klog.V(4).Infof("Added label selector: %v", o.LabelSelector)
	}

	// Configure inference or tuning
	if o.Tuning {
		klog.V(3).Info("Configuring tuning mode")
		// Tuning configuration
		tuning := map[string]interface{}{}
		
		if o.TuningMethod != "" {
			tuning["method"] = o.TuningMethod
		}
		
		if o.Model != "" {
			tuning["preset"] = map[string]interface{}{
				"name": o.Model,
			}
		}
		
		if len(o.InputURLs) > 0 {
			tuning["input"] = map[string]interface{}{
				"urls": o.InputURLs,
			}
		}
		
		if o.OutputImage != "" {
			tuning["output"] = map[string]interface{}{
				"image": o.OutputImage,
			}
		}
		
		spec["tuning"] = tuning
	} else {
		klog.V(3).Info("Configuring inference mode")
		// Inference configuration
		inference := map[string]interface{}{}
		
		if o.Model != "" {
			inference["preset"] = map[string]interface{}{
				"name": o.Model,
			}
		}
		
		// Add model access secret if specified
		if o.ModelAccessSecret != "" {
			inference["accessMode"] = "private"
			inference["secretName"] = o.ModelAccessSecret
			klog.V(4).Info("Added private model access configuration")
		}
		
		// Add adapters if specified
		if len(o.Adapters) > 0 {
			inference["adapters"] = o.Adapters
			klog.V(4).Infof("Added adapters: %v", o.Adapters)
		}
		
		spec["inference"] = inference
	}

	return spec
}

func (o *DeployOptions) parseAdapter(adapter string) map[string]interface{} {
	// Parse adapter format: name=image,strength=value
	parts := strings.Split(adapter, ",")
	result := make(map[string]interface{})

	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			if key == "strength" {
				// Try to parse as float
				result[key] = value
			} else {
				result[key] = value
			}
		}
	}

	return result
}

func (o *DeployOptions) showDryRun() error {
	klog.V(2).Info("Running in dry-run mode")

	klog.Info("🔍 Dry-run mode: Showing what would be created")
	klog.Info("")
	klog.Info("Workspace Configuration:")
	klog.Info("========================")
	klog.Infof("Name: %s", o.WorkspaceName)
	klog.Infof("Namespace: %s", o.Namespace)
	klog.Infof("Model: %s", o.Model)
	klog.Infof("Count: %d", o.Count)

	if o.InstanceType != "" {
		klog.Infof("Instance Type: %s", o.InstanceType)
	}

	if o.Tuning {
		klog.Infof("Mode: Fine-tuning (%s)", o.TuningMethod)
		if len(o.InputURLs) > 0 {
			klog.Infof("Input URLs: %v", o.InputURLs)
		}
		if o.OutputImage != "" {
			klog.Infof("Output Image: %s", o.OutputImage)
		}
	} else {
		klog.Info("Mode: Inference")
		if len(o.Adapters) > 0 {
			klog.Infof("Adapters: %v", o.Adapters)
		}
	}

	if len(o.LabelSelector) > 0 {
		klog.Infof("Label Selector: %v", o.LabelSelector)
	}

	klog.Info("")
	klog.Info("✓ Workspace definition is valid")
	klog.Info("ℹ️  Run without --dry-run to create the workspace")

	return nil
}
