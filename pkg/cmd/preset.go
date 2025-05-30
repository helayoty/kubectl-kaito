package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type PresetOptions struct {
	configFlags *genericclioptions.ConfigFlags
	Action      string
	ModelType   string
}

// Known Kaito model presets based on documentation
var knownPresets = map[string][]string{
	"llama": {
		"llama-2-7b",
		"llama-2-7b-chat",
		"llama-2-13b",
		"llama-2-13b-chat",
		"llama-2-70b",
		"llama-2-70b-chat",
		"llama-3-8b-instruct",
		"llama-3-70b-instruct",
	},
	"falcon": {
		"falcon-7b",
		"falcon-7b-instruct",
		"falcon-40b",
		"falcon-40b-instruct",
		"falcon-180b",
		"falcon-180b-chat",
	},
	"phi": {
		"phi-2",
		"phi-3-mini-4k-instruct",
		"phi-3-mini-128k-instruct",
		"phi-3-small-8k-instruct",
		"phi-3-small-128k-instruct",
		"phi-3-medium-4k-instruct",
		"phi-3-medium-128k-instruct",
		"phi-3.5-mini-instruct",
	},
	"mistral": {
		"mistral-7b",
		"mistral-7b-instruct",
	},
}

var tuningPresets = []string{
	"qlora",
	"lora",
}

func NewPresetCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preset",
		Short: "Manage Kaito model presets",
		Long: `Manage Kaito model presets.

This command helps you discover available model presets for inference
and fine-tuning operations.`,
		Example: `  # List all available model presets
  kubectl kaito preset list
  
  # List presets for a specific model family
  kubectl kaito preset list --model llama
  
  # Show details about tuning presets
  kubectl kaito preset list --model tuning`,
	}

	cmd.AddCommand(NewPresetListCmd(configFlags))

	return cmd
}

func NewPresetListCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	o := &PresetOptions{
		configFlags: configFlags,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available model presets",
		Long: `List available model presets for Kaito.

Shows the available model presets that can be used with the deploy and tune commands.`,
		Example: `  # List all available presets
  kubectl kaito preset list
  
  # List presets for llama models
  kubectl kaito preset list --model llama
  
  # List tuning presets
  kubectl kaito preset list --model tuning`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.RunList()
		},
	}

	cmd.Flags().StringVar(&o.ModelType, "model", "", "Filter by model family (llama, falcon, phi, mistral, tuning)")

	return cmd
}

func (o *PresetOptions) RunList() error {
	if o.ModelType == "" {
		// Show all presets
		o.printAllPresets()
		return nil
	}

	if o.ModelType == "tuning" {
		o.printTuningPresets()
		return nil
	}

	// Show presets for specific model family
	if presets, exists := knownPresets[strings.ToLower(o.ModelType)]; exists {
		o.printModelPresets(o.ModelType, presets)
		return nil
	}

	return fmt.Errorf("unknown model family: %s. Available families: %s, tuning",
		o.ModelType, strings.Join(o.getModelFamilies(), ", "))
}

func (o *PresetOptions) printAllPresets() {
	fmt.Println("Available Kaito Model Presets:")
	fmt.Println("==============================")
	fmt.Println()

	// Sort model families for consistent output
	families := o.getModelFamilies()
	sort.Strings(families)

	for _, family := range families {
		presets := knownPresets[family]
		o.printModelPresets(family, presets)
		fmt.Println()
	}

	fmt.Println("Tuning Presets:")
	fmt.Println("---------------")
	for _, preset := range tuningPresets {
		fmt.Printf("  %s\n", preset)
	}
	fmt.Println()

	fmt.Println("Usage Examples:")
	fmt.Println("  kubectl kaito deploy --name my-workspace --model llama-3-8b-instruct --preset instruct")
	fmt.Println("  kubectl kaito tune --name my-tuned-model --model llama-2-7b --dataset s3://my-data --preset qlora")
}

func (o *PresetOptions) printModelPresets(family string, presets []string) {
	fmt.Printf("%s Models:\n", strings.Title(family))
	fmt.Println(strings.Repeat("-", len(family)+8))

	sort.Strings(presets)
	for _, preset := range presets {
		fmt.Printf("  %s\n", preset)
	}
}

func (o *PresetOptions) printTuningPresets() {
	fmt.Println("Available Tuning Presets:")
	fmt.Println("========================")
	fmt.Println()

	for _, preset := range tuningPresets {
		fmt.Printf("  %s", preset)
		switch preset {
		case "qlora":
			fmt.Printf(" - Quantized Low-Rank Adaptation (recommended for most use cases)")
		case "lora":
			fmt.Printf(" - Low-Rank Adaptation")
		}
		fmt.Println()
	}
	fmt.Println()

	fmt.Println("Usage Example:")
	fmt.Println("  kubectl kaito tune --name my-tuned-model --model llama-2-7b --dataset s3://my-data --preset qlora")
}

func (o *PresetOptions) getModelFamilies() []string {
	families := make([]string, 0, len(knownPresets))
	for family := range knownPresets {
		families = append(families, family)
	}
	return families
}
