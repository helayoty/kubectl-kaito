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
| `--sort-by string` | string   | name    | Sort by field (name, memory, nodes)          |
| `--tags strings`   | []string |         | Filter by tags (comma-separated)             |
| `--type string`    | string   |         | Filter by model type (LLM, Code, etc.)       |

### Examples

#### Basic Model List

```bash
# List all models
kubectl kaito models list
```

Output:
```
NAME                     TYPE    RUNTIME    GPU MEMORY    NODES    DESCRIPTION
llama-2-7b              LLM     vLLM       13Gi          1        Llama 2 7B model for general conversation
llama-2-13b             LLM     vLLM       26Gi          1        Llama 2 13B model with enhanced capabilities
llama-2-70b             LLM     vLLM       140Gi         4        Llama 2 70B large model for complex tasks
phi-3.5-mini-instruct   LLM     vLLM       8Gi           1        Microsoft Phi-3.5 Mini instruction-tuned
mistral-7b-instruct     LLM     vLLM       13Gi          1        Mistral 7B instruction-tuned model
falcon-7b               LLM     vLLM       15Gi          1        Falcon 7B model for general use
```

#### Detailed Model Information

```bash
# List with detailed information
kubectl kaito models list --detailed
```

Output:
```
NAME: llama-2-7b
TYPE: LLM
RUNTIME: vLLM
GPU MEMORY: 13Gi
NODES: 1
TAGS: meta, llama, chat, 7b
INSTANCE TYPES: Standard_NC6s_v3, Standard_NC12s_v3
DESCRIPTION: Llama 2 7B model optimized for general conversation and instruction following

NAME: phi-3.5-mini-instruct
TYPE: LLM  
RUNTIME: vLLM
GPU MEMORY: 8Gi
NODES: 1
TAGS: microsoft, phi, instruct, small
INSTANCE TYPES: Standard_NC6s_v3
DESCRIPTION: Microsoft's Phi-3.5 Mini model fine-tuned for instruction following
```

#### Filter by Model Type

```bash
# Filter by model type
kubectl kaito models list --type LLM
```

#### Filter by Tags

```bash
# Filter by tags
kubectl kaito models list --tags microsoft,phi
```

#### Sort by Memory Requirements

```bash
# Sort by memory requirements
kubectl kaito models list --sort-by memory
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
    "name": "llama-2-7b",
    "type": "LLM",
    "runtime": "vLLM",
    "gpu_memory": "13Gi", 
    "nodes": 1,
    "tags": ["meta", "llama", "chat", "7b"],
    "instance_types": ["Standard_NC6s_v3", "Standard_NC12s_v3"],
    "description": "Llama 2 7B model optimized for general conversation"
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
Type: LLM
Runtime: vLLM

Description:
Microsoft's Phi-3.5 Mini is a lightweight, state-of-the-art open model 
built upon datasets used for Phi-3. The model is instruction-tuned and 
optimized for chat, code, and reasoning scenarios.

Specifications:
  Parameters: 3.8B
  Context Length: 128K tokens
  GPU Memory Required: 8Gi
  Recommended Nodes: 1
  
Supported Instance Types:
  - Standard_NC6s_v3 (1x V100 16GB)
  - Standard_NC12s_v3 (2x V100 32GB) 
  - Standard_NC24ads_A100_v4 (1x A100 80GB)

Tags: microsoft, phi, instruct, small, efficient

Fine-tuning Support: Yes
  Supported Methods: qlora, lora
  
Usage Examples:
  # Deploy for inference
  kubectl kaito deploy --workspace-name phi-workspace --model phi-3.5-mini-instruct
  
  # Deploy for fine-tuning
  kubectl kaito deploy --workspace-name tune-phi --model phi-3.5-mini-instruct --tuning

Documentation: https://huggingface.co/microsoft/Phi-3.5-mini-instruct
```

#### Describe with JSON Output

```bash
# Get detailed model info in JSON
kubectl kaito models describe llama-2-7b --output json
```

## Model Categories

### Large Language Models (LLM)

General-purpose conversational models:
- `llama-2-7b`, `llama-2-13b`, `llama-2-70b`
- `phi-3.5-mini-instruct`, `phi-3.5-medium-instruct`
- `mistral-7b-instruct`, `mixtral-8x7b-instruct`

### Code Models

Specialized for code generation:
- `codellama-7b`, `codellama-13b`
- `phi-3.5-mini-instruct` (also supports code)

### Multimodal Models

Support text and image inputs:
- `llava-1.6-mistral-7b`
- `phi-3.5-vision-instruct`

## Model Selection Guide

### By Use Case

| Use Case             | Recommended Models                             |
| -------------------- | ---------------------------------------------- |
| General Chat         | `phi-3.5-mini-instruct`, `llama-2-7b`          |
| Code Generation      | `codellama-7b`, `phi-3.5-mini-instruct`        |
| Complex Reasoning    | `llama-2-13b`, `mixtral-8x7b-instruct`         |
| High Performance     | `llama-2-70b`, `mixtral-8x22b-instruct`        |
| Resource Constrained | `phi-3.5-mini-instruct`, `mistral-7b-instruct` |

### By GPU Memory

| GPU Memory | Suitable Models                                |
| ---------- | ---------------------------------------------- |
| 16GB       | `phi-3.5-mini-instruct`, `mistral-7b-instruct` |
| 32GB       | `llama-2-7b`, `codellama-7b`                   |
| 64GB+      | `llama-2-13b`, `mixtral-8x7b-instruct`         |
| 256GB+     | `llama-2-70b`, `mixtral-8x22b-instruct`        |

## Instance Type Recommendations

### Azure GPU Instance Types

| Instance Type            | GPU     | Memory | Suitable Models           |
| ------------------------ | ------- | ------ | ------------------------- |
| Standard_NC6s_v3         | 1x V100 | 16GB   | phi-3.5-mini, mistral-7b  |
| Standard_NC12s_v3        | 2x V100 | 32GB   | llama-2-7b, codellama-7b  |
| Standard_NC24s_v3        | 4x V100 | 64GB   | llama-2-13b, mixtral-8x7b |
| Standard_NC24ads_A100_v4 | 1x A100 | 80GB   | llama-2-13b, mixtral-8x7b |

## Troubleshooting

### Model List Not Loading

```bash
# Force refresh from official repository
kubectl kaito models list --refresh

# Check internet connectivity
curl -I https://raw.githubusercontent.com/kaito-project/kaito/main/presets/models/supported_models.yaml
```

### Model Not Found

```bash
# List all available models
kubectl kaito models list

# Search for similar models
kubectl kaito models list --tags <tag>
```
