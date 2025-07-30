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
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// GetEndpointOptions holds the options for the get-endpoint command
type GetEndpointOptions struct {
	configFlags *genericclioptions.ConfigFlags

	WorkspaceName string
	Namespace     string
	External      bool
	Format        string
}

// NewGetEndpointCmd creates the get-endpoint command
func NewGetEndpointCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	o := &GetEndpointOptions{
		configFlags: configFlags,
	}

	cmd := &cobra.Command{
		Use:   "get-endpoint",
		Short: "Get inference endpoints for a Kaito workspace",
		Long: `Get the inference endpoint URL for a deployed Kaito workspace.

This command retrieves the service endpoint that can be used to send inference
requests to the deployed model. The endpoint supports OpenAI-compatible APIs.`,
		Example: `  # Get endpoint URL for a workspace
  kubectl kaito get-endpoint --workspace-name my-workspace

  # Get endpoint in JSON format with metadata
  kubectl kaito get-endpoint --workspace-name my-workspace --format json

  # Get external endpoint if available (LoadBalancer/Ingress)
  kubectl kaito get-endpoint --workspace-name my-workspace --external`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.validate(); err != nil {
				return err
			}
			return o.run()
		},
	}

	cmd.Flags().StringVar(&o.WorkspaceName, "workspace-name", "", "Name of the workspace (required)")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "Kubernetes namespace")
	cmd.Flags().BoolVar(&o.External, "external", false, "Get external endpoint (LoadBalancer/Ingress)")
	cmd.Flags().StringVar(&o.Format, "format", "url", "Output format: url or json")

	if err := cmd.MarkFlagRequired("workspace-name"); err != nil {
		klog.Errorf("Failed to mark workspace-name flag as required: %v", err)
	}

	return cmd
}

func (o *GetEndpointOptions) validate() error {
	klog.V(4).Info("Validating get-endpoint options")

	if o.WorkspaceName == "" {
		return fmt.Errorf("workspace name is required")
	}
	if o.Format != "url" && o.Format != "json" {
		return fmt.Errorf("format must be 'url' or 'json'")
	}

	klog.V(4).Info("Get-endpoint validation completed successfully")
	return nil
}

func (o *GetEndpointOptions) run() error {
	klog.V(2).Infof("Getting endpoint for workspace: %s", o.WorkspaceName)

	// Get namespace
	if o.Namespace == "" {
		if ns, _, err := o.configFlags.ToRawKubeConfigLoader().Namespace(); err == nil && ns != "" {
			o.Namespace = ns
		} else {
			klog.V(4).Info("No namespace specified, using 'default'")
			o.Namespace = "default"
		}
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

	// Create kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Errorf("Failed to create kubernetes client: %v", err)
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Check workspace status first
	if err := o.checkWorkspaceReady(dynamicClient); err != nil {
		return err
	}

	// Get the endpoint
	endpoint, err := o.getServiceEndpoint(context.TODO(), clientset)
	if err != nil {
		klog.Errorf("Failed to get service endpoint: %v", err)
		return fmt.Errorf("failed to get service endpoint: %w", err)
	}

	// Output the result
	if o.Format == "json" {
		output := map[string]interface{}{
			"workspace": o.WorkspaceName,
			"namespace": o.Namespace,
			"endpoint":  endpoint,
			"type":      "inference",
		}
		if o.External {
			output["access"] = "external"
		} else {
			output["access"] = "cluster"
		}

		jsonOutput, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			klog.Errorf("Failed to marshal JSON: %v", err)
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println(endpoint)
	}

	return nil
}

func (o *GetEndpointOptions) checkWorkspaceReady(dynamicClient dynamic.Interface) error {
	klog.V(3).Info("Checking workspace readiness")

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

	// Check if workspace has status
	status, found := workspace.Object["status"]
	if !found {
		return fmt.Errorf("workspace %s has no status information", o.WorkspaceName)
	}

	// Check workspace ready condition
	if !o.isWorkspaceReady(status) {
		return fmt.Errorf("workspace %s is not ready yet. Use 'kubectl kaito status --workspace-name %s' to check status", o.WorkspaceName, o.WorkspaceName)
	}

	klog.V(3).Info("Workspace is ready")
	return nil
}

func (o *GetEndpointOptions) isWorkspaceReady(status interface{}) bool {
	statusMap, ok := status.(map[string]interface{})
	if !ok {
		klog.V(6).Info("Status is not a map")
		return false
	}

	conditions, found := statusMap["conditions"]
	if !found {
		klog.V(6).Info("No conditions found in status")
		return false
	}

	conditionsList, ok := conditions.([]interface{})
	if !ok {
		klog.V(6).Info("Conditions is not a slice")
		return false
	}

	for _, condition := range conditionsList {
		condMap, ok := condition.(map[string]interface{})
		if !ok {
			continue
		}

		condType, ok := condMap["type"].(string)
		if !ok || condType != "WorkspaceReady" {
			continue
		}

		condStatus, ok := condMap["status"].(string)
		if ok && condStatus == "True" {
			return true
		}
	}

	return false
}

func (o *GetEndpointOptions) getServiceEndpoint(ctx context.Context, clientset kubernetes.Interface) (string, error) {
	klog.V(3).Infof("Getting service endpoint for workspace: %s", o.WorkspaceName)

	// Get the service for the workspace (service name equals workspace name)
	svc, err := clientset.CoreV1().Services(o.Namespace).Get(ctx, o.WorkspaceName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get service for workspace %s: %v", o.WorkspaceName, err)
		return "", fmt.Errorf("failed to get service for workspace %s: %v", o.WorkspaceName, err)
	}

	if o.External {
		// Check for LoadBalancer endpoint
		if svc.Spec.Type == "LoadBalancer" {
			for _, ingress := range svc.Status.LoadBalancer.Ingress {
				var endpoint string
				if ingress.IP != "" {
					endpoint = fmt.Sprintf("http://%s:80", ingress.IP)
				} else if ingress.Hostname != "" {
					endpoint = fmt.Sprintf("http://%s:80", ingress.Hostname)
				}
				if endpoint != "" {
					klog.V(3).Infof("Found external LoadBalancer endpoint: %s", endpoint)
					return endpoint, nil
				}
			}
		}
		// Could also check for Ingress resources here
	}

	// Return cluster-internal service endpoint
	if svc.Spec.ClusterIP != "" && svc.Spec.ClusterIP != "None" {
		endpoint := fmt.Sprintf("http://%s.%s.svc.cluster.local:80", o.WorkspaceName, o.Namespace)
		klog.V(3).Infof("Using cluster-internal endpoint: %s", endpoint)
		return endpoint, nil
	}

	return "", fmt.Errorf("service %s has no cluster IP", o.WorkspaceName)
}
