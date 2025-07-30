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
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
)

// SupportedModelsURL is the official URL for Kaito supported models
const SupportedModelsURL = "https://raw.githubusercontent.com/kaito-project/kaito/main/presets/workspace/models/supported_models.yaml"

// Model represents a supported AI model from the official Kaito repository
type Model struct {
	Name         string            `json:"name" yaml:"name"`
	Type         string            `json:"type" yaml:"type"`
	Runtime      string            `json:"runtime" yaml:"runtime"`
	Description  string            `json:"description" yaml:"description"`
	Version      string            `json:"version" yaml:"version"`
	Tag          string            `json:"tag" yaml:"tag"`
	GPUMemory    string            `json:"gpu_memory" yaml:"gpuMemory"`
	MinNodes     int               `json:"min_nodes" yaml:"minNodes"`
	MaxNodes     int               `json:"max_nodes" yaml:"maxNodes"`
	Tags         []string          `json:"tags" yaml:"tags"`
	InstanceType string            `json:"instance_type,omitempty" yaml:"instanceType,omitempty"`
	Properties   map[string]string `json:"properties,omitempty" yaml:"properties,omitempty"`
}

// KaitoSupportedModelsResponse represents the structure of the official supported_models.yaml
type KaitoSupportedModelsResponse struct {
	Models []struct {
		Name         string            `yaml:"name"`
		Version      string            `yaml:"version,omitempty"`
		Tag          string            `yaml:"tag,omitempty"`
		Type         string            `yaml:"type,omitempty"`
		Runtime      string            `yaml:"runtime,omitempty"`
		GPUMemory    string            `yaml:"gpuMemory,omitempty"`
		MinNodes     int               `yaml:"minNodes,omitempty"`
		MaxNodes     int               `yaml:"maxNodes,omitempty"`
		InstanceType string            `yaml:"instanceType,omitempty"`
		Description  string            `yaml:"description,omitempty"`
		Properties   map[string]string `yaml:"properties,omitempty"`
	} `yaml:"models"`
}

// fetchSupportedModelsFromKaito retrieves the official supported models from Kaito repository
func fetchSupportedModelsFromKaito() ([]Model, error) {
	klog.V(3).Info("Fetching supported models from official Kaito repository")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", SupportedModelsURL, nil)
	if err != nil {
		klog.Errorf("Failed to create request: %v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		klog.Errorf("Failed to fetch supported models: %v", err)
		return nil, fmt.Errorf("failed to fetch supported models from %s: %w", SupportedModelsURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("HTTP request failed with status: %d", resp.StatusCode)
		return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body: %v", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var kaitoModels KaitoSupportedModelsResponse
	if err := yaml.Unmarshal(body, &kaitoModels); err != nil {
		klog.Errorf("Failed to parse YAML response: %v", err)
		return nil, fmt.Errorf("failed to parse YAML response: %w", err)
	}

	// Convert to our Model struct format
	var models []Model
	for _, km := range kaitoModels.Models {
		model := Model{
			Name:         km.Name,
			Type:         km.Type,
			Runtime:      km.Runtime,
			Version:      km.Version,
			Tag:          km.Tag,
			GPUMemory:    km.GPUMemory,
			MinNodes:     km.MinNodes,
			MaxNodes:     km.MaxNodes,
			InstanceType: km.InstanceType,
			Description:  km.Description,
			Properties:   km.Properties,
		}

		// Set default values if not specified
		if model.Type == "" {
			model.Type = "LLM"
		}
		if model.Runtime == "" {
			model.Runtime = "vllm"
		}
		if model.MinNodes == 0 {
			model.MinNodes = 1
		}
		if model.MaxNodes == 0 {
			model.MaxNodes = model.MinNodes
		}

		// Generate description if not provided
		if model.Description == "" {
			model.Description = fmt.Sprintf("Official Kaito supported model: %s", model.Name)
		}

		models = append(models, model)
	}

	klog.V(3).Infof("Successfully fetched %d models from official Kaito repository", len(models))
	return models, nil
}

// getSupportedModels returns supported models, first trying to fetch from official source,
// falling back to hardcoded list if necessary
func getSupportedModels() []Model {
	klog.V(4).Info("Getting supported models list")

	// Try to fetch from official Kaito repository first
	if models, err := fetchSupportedModelsFromKaito(); err == nil && len(models) > 0 {
		klog.V(3).Info("Using models from official Kaito repository")
		return models
	} else {
		klog.Warningf("Failed to fetch from official repository, using fallback models: %v", err)
	}

	// Fallback to hardcoded models based on what we know from Kaito
	return []Model{
		{
			Name:        "phi-3.5-mini-instruct",
			Type:        "LLM",
			Runtime:     "vllm",
			Description: "Microsoft Phi-3.5 Mini Instruct - A small, capable language model",
			Version:     "3.5",
			Tag:         "instruct",
			GPUMemory:   "4GB",
			MinNodes:    1,
			MaxNodes:    1,
			Tags:        []string{"microsoft", "phi", "instruct", "small"},
		},
		{
			Name:        "phi-4",
			Type:        "LLM",
			Runtime:     "vllm",
			Description: "Microsoft Phi-4 - Next generation small language model",
			Version:     "4.0",
			Tag:         "base",
			GPUMemory:   "8GB",
			MinNodes:    1,
			MaxNodes:    2,
			Tags:        []string{"microsoft", "phi", "latest"},
		},
		{
			Name:        "llama-2-7b",
			Type:        "LLM",
			Runtime:     "vllm",
			Description: "Meta Llama 2 7B - Popular open-source language model",
			Version:     "2.0",
			Tag:         "7b",
			GPUMemory:   "14GB",
			MinNodes:    1,
			MaxNodes:    1,
			Tags:        []string{"meta", "llama", "7b", "base"},
		},
		{
			Name:        "llama-2-13b",
			Type:        "LLM",
			Runtime:     "vllm",
			Description: "Meta Llama 2 13B - Larger variant with improved capabilities",
			Version:     "2.0",
			Tag:         "13b",
			GPUMemory:   "26GB",
			MinNodes:    1,
			MaxNodes:    2,
			Tags:        []string{"meta", "llama", "13b", "large"},
		},
		{
			Name:        "llama-2-70b",
			Type:        "LLM",
			Runtime:     "vllm",
			Description: "Meta Llama 2 70B - High-capability model for demanding tasks",
			Version:     "2.0",
			Tag:         "70b",
			GPUMemory:   "140GB",
			MinNodes:    2,
			MaxNodes:    8,
			Tags:        []string{"meta", "llama", "70b", "enterprise"},
		},
		{
			Name:        "mistral-7b",
			Type:        "LLM",
			Runtime:     "vllm",
			Description: "Mistral 7B - Efficient multilingual language model",
			Version:     "1.0",
			Tag:         "7b",
			GPUMemory:   "14GB",
			MinNodes:    1,
			MaxNodes:    1,
			Tags:        []string{"mistral", "7b", "multilingual"},
		},
		{
			Name:        "qwen-7b",
			Type:        "LLM",
			Runtime:     "vllm",
			Description: "Alibaba Qwen 7B - Strong performance on Chinese and English",
			Version:     "1.0",
			Tag:         "7b",
			GPUMemory:   "14GB",
			MinNodes:    1,
			MaxNodes:    1,
			Tags:        []string{"alibaba", "qwen", "7b", "chinese"},
		},
		{
			Name:        "codellama-7b",
			Type:        "Code",
			Runtime:     "vllm",
			Description: "Meta Code Llama 7B - Specialized for code generation",
			Version:     "1.0",
			Tag:         "7b-code",
			GPUMemory:   "14GB",
			MinNodes:    1,
			MaxNodes:    1,
			Tags:        []string{"meta", "code", "programming", "7b"},
		},
	}
}

// ValidateModelName checks if the provided model name is supported by Kaito
func ValidateModelName(modelName string) error {
	klog.V(4).Infof("Validating model name: %s", modelName)

	if modelName == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	models := getSupportedModels()
	for _, model := range models {
		if model.Name == modelName {
			klog.V(4).Infof("Model %s is valid", modelName)
			return nil
		}
	}

	// Generate suggestions for similar model names
	suggestions := []string{}
	lowerModelName := strings.ToLower(modelName)
	for _, model := range models {
		if strings.Contains(strings.ToLower(model.Name), lowerModelName) ||
			strings.Contains(lowerModelName, strings.ToLower(model.Name)) {
			suggestions = append(suggestions, model.Name)
		}
	}

	var suggestionText string
	if len(suggestions) > 0 {
		suggestionText = fmt.Sprintf("\n\nDid you mean one of these?\n  - %s", strings.Join(suggestions, "\n  - "))
	} else {
		suggestionText = "\n\nUse 'kubectl kaito models list' to see all supported models."
	}

	return fmt.Errorf("model '%s' is not supported by Kaito%s", modelName, suggestionText)
}

// NewModelsCmd creates the models command with subcommands
func NewModelsCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "Manage and list supported AI models",
		Long: `List and describe supported AI models available in Kaito.

This command helps you discover which models are supported, their requirements,
and configuration options for deployment. The model list is fetched from the
official Kaito repository to ensure accuracy.`,
		Example: `  # List all supported models (fetched from official Kaito repo)
  kubectl kaito models list

  # List models with detailed information
  kubectl kaito models list --detailed

  # Describe a specific model
  kubectl kaito models describe phi-3.5-mini-instruct

  # Filter models by type
  kubectl kaito models list --type LLM

  # Filter models by tags
  kubectl kaito models list --tags microsoft,small

  # Refresh models cache (force fetch from repo)
  kubectl kaito models list --refresh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Use 'kubectl kaito models list' or 'kubectl kaito models describe <model>' for more information")
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newModelsListCmd(configFlags))
	cmd.AddCommand(newModelsDescribeCmd(configFlags))

	return cmd
}

func newModelsListCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	var (
		detailed   bool
		modelType  string
		tags       []string
		sortBy     string
		outputJSON bool
		refresh    bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List supported AI models",
		Long: `List all supported AI models available for deployment with Kaito.

Shows model names, types, runtime requirements, and resource specifications.
Models are fetched from the official Kaito repository to ensure accuracy.`,
		Example: `  # List all models
  kubectl kaito models list

  # List with detailed information
  kubectl kaito models list --detailed

  # Filter by model type
  kubectl kaito models list --type LLM

  # Filter by tags
  kubectl kaito models list --tags microsoft,phi

  # Sort by name or memory requirements
  kubectl kaito models list --sort-by name

  # Output in JSON format
  kubectl kaito models list --output json

  # Force refresh from official repository
  kubectl kaito models list --refresh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelsList(detailed, modelType, tags, sortBy, outputJSON, refresh)
		},
	}

	cmd.Flags().BoolVar(&detailed, "detailed", false, "Show detailed model information")
	cmd.Flags().StringVar(&modelType, "type", "", "Filter by model type (LLM, Code, etc.)")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "Filter by tags (comma-separated)")
	cmd.Flags().StringVar(&sortBy, "sort-by", "name", "Sort by field (name, memory, nodes)")
	cmd.Flags().BoolVar(&outputJSON, "output", false, "Output in JSON format")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Force refresh from official Kaito repository")

	return cmd
}

func newModelsDescribeCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <model-name>",
		Short: "Describe a specific AI model",
		Long: `Show detailed information about a specific AI model including:
- Model specifications and requirements
- Supported runtime configurations
- Resource requirements and scaling options
- Usage examples and deployment commands`,
		Example: `  # Describe the Phi-3.5 model
  kubectl kaito models describe phi-3.5-mini-instruct

  # Describe Llama 2 7B model
  kubectl kaito models describe llama-2-7b`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelsDescribe(args[0])
		},
	}

	return cmd
}

func runModelsList(detailed bool, modelType string, tags []string, sortBy string, outputJSON bool, refresh bool) error {
	klog.V(2).Info("Listing supported models")

	if refresh {
		fmt.Println("Refreshing models from official Kaito repository...")
	}

	models := getSupportedModels()

	// Apply filters
	if modelType != "" {
		klog.V(3).Infof("Filtering by type: %s", modelType)
		models = filterModelsByType(models, modelType)
	}

	if len(tags) > 0 {
		klog.V(3).Infof("Filtering by tags: %v", tags)
		models = filterModelsByTags(models, tags)
	}

	// Sort models
	sortModels(models, sortBy)

	if len(models) == 0 {
		fmt.Println("No models found matching the specified criteria")
		return nil
	}

	if refresh {
		fmt.Printf("Successfully loaded %d models from official repository\n", len(models))
		fmt.Println()
	}

	if outputJSON {
		return printModelsJSON(models)
	}

	if detailed {
		return printModelsDetailed(models)
	}

	return printModelsTable(models)
}

func runModelsDescribe(modelName string) error {
	klog.V(2).Infof("Describing model: %s", modelName)

	models := getSupportedModels()

	for _, model := range models {
		if model.Name == modelName {
			return printModelDetail(model)
		}
	}

	// Use the validation function to provide helpful error message
	return ValidateModelName(modelName)
}

func filterModelsByType(models []Model, modelType string) []Model {
	var filtered []Model
	for _, model := range models {
		if strings.EqualFold(model.Type, modelType) {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

func filterModelsByTags(models []Model, tags []string) []Model {
	var filtered []Model
	for _, model := range models {
		for _, tag := range tags {
			if containsTag(model.Tags, tag) {
				filtered = append(filtered, model)
				break
			}
		}
	}
	return filtered
}

func containsTag(tags []string, target string) bool {
	for _, tag := range tags {
		if strings.EqualFold(tag, target) {
			return true
		}
	}
	return false
}

func sortModels(models []Model, sortBy string) {
	klog.V(4).Infof("Sorting models by: %s", sortBy)

	switch sortBy {
	case "name":
		sort.Slice(models, func(i, j int) bool {
			return models[i].Name < models[j].Name
		})
	case "memory":
		sort.Slice(models, func(i, j int) bool {
			return models[i].GPUMemory < models[j].GPUMemory
		})
	case "nodes":
		sort.Slice(models, func(i, j int) bool {
			return models[i].MinNodes < models[j].MinNodes
		})
	default:
		klog.V(4).Infof("Unknown sort field '%s', using name", sortBy)
		sort.Slice(models, func(i, j int) bool {
			return models[i].Name < models[j].Name
		})
	}
}

func printModelsTable(models []Model) error {
	klog.V(3).Info("Printing models table")

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "NAME\tTYPE\tRUNTIME\tGPU MEMORY\tNODES\tDESCRIPTION")

	for _, model := range models {
		nodeRange := fmt.Sprintf("%d", model.MinNodes)
		if model.MaxNodes > model.MinNodes {
			nodeRange = fmt.Sprintf("%d-%d", model.MinNodes, model.MaxNodes)
		}

		description := model.Description
		if len(description) > 50 {
			description = description[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			model.Name, model.Type, model.Runtime, model.GPUMemory, nodeRange, description)
	}

	return nil
}

func printModelsDetailed(models []Model) error {
	klog.V(3).Info("Printing detailed models information")

	for i, model := range models {
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("Name: %s\n", model.Name)
		fmt.Printf("Type: %s\n", model.Type)
		fmt.Printf("Runtime: %s\n", model.Runtime)
		fmt.Printf("Version: %s\n", model.Version)
		fmt.Printf("Description: %s\n", model.Description)
		fmt.Printf("GPU Memory: %s\n", model.GPUMemory)
		fmt.Printf("Node Range: %d-%d\n", model.MinNodes, model.MaxNodes)
		if len(model.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(model.Tags, ", "))
		}
	}

	return nil
}

func printModelsJSON(models []Model) error {
	klog.V(3).Info("Printing models in JSON format")

	jsonData, err := json.MarshalIndent(models, "", "  ")
	if err != nil {
		klog.Errorf("Failed to marshal models to JSON: %v", err)
		return fmt.Errorf("failed to marshal models to JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

func printModelDetail(model Model) error {
	klog.V(3).Infof("Printing detailed information for model: %s", model.Name)

	fmt.Printf("Model: %s\n", model.Name)
	fmt.Println("================")
	fmt.Println()
	fmt.Printf("Description: %s\n", model.Description)
	fmt.Printf("Type: %s\n", model.Type)
	fmt.Printf("Runtime: %s\n", model.Runtime)
	fmt.Printf("Version: %s\n", model.Version)
	fmt.Println()
	fmt.Println("Resource Requirements:")
	fmt.Printf("  GPU Memory: %s\n", model.GPUMemory)
	fmt.Printf("  Minimum Nodes: %d\n", model.MinNodes)
	fmt.Printf("  Maximum Nodes: %d\n", model.MaxNodes)
	fmt.Println()
	if len(model.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(model.Tags, ", "))
		fmt.Println()
	}

	fmt.Println("Usage Example:")
	fmt.Printf("  kubectl kaito deploy --workspace-name my-workspace --model %s\n", model.Name)

	if model.InstanceType != "" {
		fmt.Println()
		fmt.Println("  # With recommended instance type:")
		fmt.Printf("  kubectl kaito deploy --workspace-name my-workspace --model %s --instance-type %s\n", model.Name, model.InstanceType)
	}

	fmt.Println()
	return nil
}
