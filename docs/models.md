# kubectl kaito models

Manage and list supported AI models available in Kaito.

## Synopsis

List and describe supported AI models available in Kaito. This command helps you discover which models are supported, their requirements, and configuration options for deployment. The model list is fetched from the official Kaito repository to ensure accuracy.

## Usage

```bash
kaito models [command]
```

## Available Commands

- [`list`](#list) - List supported AI models
- [`describe`](#describe) - Describe a specific AI model

## Global Flags

| Flag                     | Description                                          |
| ------------------------ | ---------------------------------------------------- |
| `--kubeconfig string`    | Path to the kubeconfig file to use for CLI requests  |
| `--context string`       | The name of the kubeconfig context to use            |
| `-n, --namespace string` | If present, the namespace scope for this CLI request |

---

## list

List all supported AI models available for deployment with Kaito.

### Usage

```bash
kaito models list [flags]
```

### Flags

| Flag               | Type     | Default | Description                                  |
| ------------------ | -------- | ------- | -------------------------------------------- |
| `--detailed`       | bool     | false   | Show detailed model information              |
| `--output`         | bool     | false   | Output in JSON format                        |
| `--refresh`        | bool     | false   | Force refresh from official Kaito repository |
| `--sort-by string` | string   | name    | Sort by field (name)                        |
| `--tags strings`   | []string |         | Filter by tags (comma-separated)             |
| `--type string`    | string   |         | Filter by model type (text-generation, etc.) |

### Examples

#### Basic Model List

```bash
# List all models
kubectl kaito models list
```

Output:
```
NAME                          TYPE             FAMILY    RUNTIME  TAG
deepseek-r1-distill-llama-8b  text-generation  DeepSeek  tfs      0.2.0
deepseek-r1-distill-qwen-14b  text-generation  DeepSeek  tfs      0.2.0
falcon-40b                    text-generation  Falcon    tfs      0.2.0
falcon-40b-instruct           text-generation  Falcon    tfs      0.2.0
falcon-7b                     text-generation  Falcon    tfs      0.2.0
falcon-7b-instruct            text-generation  Falcon    tfs      0.2.0
llama-3.1-8b-instruct         text-generation  Llama     tfs      0.2.0
mistral-7b-instruct           text-generation  Mistral   tfs      0.2.0
phi-3.5-mini-instruct         text-generation  Phi       tfs      0.2.0

💡 Note: For deployment guidance and instanceType requirements,
   use 'kubectl kaito models describe <model>' or refer to Kaito workspace examples.
```

#### Detailed Model Information

```bash
# List with detailed information
kubectl kaito models list --detailed
```

Output:
```
Name: falcon-7b-instruct
================
Description: Official Kaito supported model: falcon-7b-instruct
Type: text-generation
Runtime: tfs
Version: https://huggingface.co/tiiuae/falcon-7b-instruct/commit/...
Resource Requirements:
  💡 GPU requirements are not available in the official Kaito repository.
     For instanceType guidance, refer to:
     - Kaito workspace examples in the GitHub repository
     - Azure VM sizes documentation
     - Hugging Face model cards for model sizes
     - Community benchmarks
Usage Example:
  kubectl kaito deploy --workspace-name my-workspace --model falcon-7b-instruct

Name: phi-3.5-mini-instruct
================
Description: Official Kaito supported model: phi-3.5-mini-instruct
Type: text-generation
Runtime: tfs
Version: https://huggingface.co/microsoft/Phi-3.5-mini-instruct/commit/...
Resource Requirements:
  💡 GPU requirements are not available in the official Kaito repository.
     For instanceType guidance, refer to:
     - Kaito workspace examples in the GitHub repository
     - Azure VM sizes documentation
     - Hugging Face model cards for model sizes
     - Community benchmarks
Usage Example:
  kubectl kaito deploy --workspace-name my-workspace --model phi-3.5-mini-instruct
```

#### Filter by Model Type

```bash
# Filter by model type
kubectl kaito models list --type text-generation
```

#### Filter by Tags

```bash
# Filter by tags (Note: Tag filtering based on model properties)
kubectl kaito models list --tags phi
```

#### Sort by Name

```bash
# Sort by name (default sorting option)
kubectl kaito models list --sort-by name
```

#### JSON Output

```bash
# Output in JSON format
kubectl kaito models list --output json
```

Output:
```json
[
  {
    "name": "falcon-7b-instruct",
    "type": "text-generation",
    "runtime": "tfs",
    "version": "https://huggingface.co/tiiuae/falcon-7b-instruct/commit/...",
    "tag": "0.2.0",
    "description": "Official Kaito supported model: falcon-7b-instruct"
  },
  {
    "name": "phi-3.5-mini-instruct", 
    "type": "text-generation",
    "runtime": "tfs",
    "version": "https://huggingface.co/microsoft/Phi-3.5-mini-instruct/commit/...",
    "tag": "0.2.0",
    "description": "Official Kaito supported model: phi-3.5-mini-instruct"
  }
]
```

#### Force Refresh

```bash
# Force refresh from official Kaito repository
kubectl kaito models list --refresh
```

---

## describe

Describe a specific AI model in detail.

### Usage

```bash
kaito models describe [model-name]
```

### Examples

#### Describe Specific Model

```bash
# Describe a specific model
kubectl kaito models describe phi-3.5-mini-instruct
```

Output:
```
Model: phi-3.5-mini-instruct
================
Description: Official Kaito supported model: phi-3.5-mini-instruct
Type: text-generation
Runtime: tfs
Version: https://huggingface.co/microsoft/Phi-3.5-mini-instruct/commit/3145e03a9fd4cdd7cd953c34d9bbf7ad606122ca

Resource Requirements:
  💡 GPU requirements are not available in the official Kaito repository.
     For instanceType guidance, refer to:
     - Kaito workspace examples in the GitHub repository
     - Azure VM sizes documentation
     - Hugging Face model cards for model sizes
     - Community benchmarks

Usage Example:
  kubectl kaito deploy --workspace-name my-workspace --model phi-3.5-mini-instruct
```

#### Describe with JSON Output

```bash
# Get detailed model info in JSON  
kubectl kaito models describe phi-3.5-mini-instruct --output json
```

## Available Models

### Text Generation Models

All models in Kaito are currently text-generation models optimized for conversational AI:

- `falcon-7b`, `falcon-7b-instruct`, `falcon-40b`, `falcon-40b-instruct`
- `llama-3.1-8b-instruct`, `llama-3.3-70b-instruct`
- `mistral-7b`, `mistral-7b-instruct`
- `phi-2`, `phi-3-mini-4k-instruct`, `phi-3-mini-128k-instruct`, `phi-3.5-mini-instruct`, `phi-4`, `phi-4-mini-instruct`
- `qwen2.5-coder-7b-instruct`, `qwen2.5-coder-32b-instruct`
- `deepseek-r1-distill-llama-8b`, `deepseek-r1-distill-qwen-14b`

### Code-Specialized Models

Models with enhanced code generation capabilities:
- `qwen2.5-coder-7b-instruct`, `qwen2.5-coder-32b-instruct`

## Model Selection Guide

### By Use Case

| Use Case             | Recommended Models                             |
| -------------------- | ---------------------------------------------- |
| General Chat         | `phi-3.5-mini-instruct`, `falcon-7b-instruct`  |
| Code Generation      | `qwen2.5-coder-7b-instruct`, `qwen2.5-coder-32b-instruct` |
| High Performance     | `llama-3.3-70b-instruct`, `falcon-40b-instruct` |
| Resource Constrained | `phi-2`, `phi-3-mini-4k-instruct`              |

### Instance Type Guidance

💡 **Note**: Specific GPU requirements are not available in the official Kaito repository. 
Refer to Kaito workspace examples for appropriate `instanceType` values such as:
- `Standard_NC6s_v3` - for smaller models
- `Standard_NC12s_v3` - for medium models  
- `Standard_NC24ads_A100_v4` - for larger models

## Troubleshooting

### Model List Not Loading

```bash
# Force refresh from official repository
kubectl kaito models list --refresh

# Check internet connectivity to official Kaito repository
curl -I https://raw.githubusercontent.com/kaito-project/kaito/main/presets/workspace/models/supported_models.yaml
```

### Model Not Found

```bash
# List all available models
kubectl kaito models list

# Get detailed information about a specific model
kubectl kaito models describe <model-name>

# Check model type filtering
kubectl kaito models list --type text-generation
```

### Deployment Guidance

For specific deployment guidance including instanceType requirements:

1. Use the `describe` command for detailed model information
2. Refer to [Kaito workspace examples](https://github.com/kaito-project/kaito/tree/main/examples) 
3. Check Azure VM sizes documentation for GPU specifications
4. Consult Hugging Face model cards for model size information
