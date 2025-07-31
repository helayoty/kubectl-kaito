# kubectl kaito deploy

Deploy a Kaito workspace for AI model inference or fine-tuning.

## Synopsis

Deploy creates a new Kaito workspace resource for AI model deployment. This command supports both inference and fine-tuning scenarios:

- **Inference**: Deploy models for real-time inference with OpenAI-compatible APIs
- **Fine-tuning**: Fine-tune existing models with your own datasets using methods like QLoRA

The workspace will automatically provision the required GPU resources and deploy the specified model according to Kaito's preset configurations.

## Usage

```bash
kaito deploy [flags]
```

## Flags

### Required Flags

| Flag                      | Type   | Description                                |
| ------------------------- | ------ | ------------------------------------------ |
| `--workspace-name string` | string | Name of the workspace to create (required) |
| `--model string`          | string | Model name to deploy (required)            |

### Optional Flags

| Flag                       | Type   | Default | Description                                          |
| -------------------------- | ------ | ------- | ---------------------------------------------------- |
| `--instance-type string`   | string |         | GPU instance type (e.g., Standard_NC6s_v3)           |
| `--count int`              | int    | 1       | Number of GPU nodes                                  |
| `--dry-run`                | bool   | false   | Show what would be created without actually creating |
| `--enable-load-balancer`   | bool   | false   | Create LoadBalancer service for external access      |
| `--bypass-resource-checks` | bool   | false   | Skip resource availability checks                    |

### Model Access Flags

| Flag                           | Type     | Description                     |
| ------------------------------ | -------- | ------------------------------- |
| `--model-access-secret string` | string   | Secret for private model access |
| `--adapters strings`           | []string | Model adapters to load          |

### Fine-tuning Flags

| Flag                           | Type     | Default | Description                       |
| ------------------------------ | -------- | ------- | --------------------------------- |
| `--tuning`                     | bool     | false   | Enable fine-tuning mode           |
| `--tuning-method string`       | string   | qlora   | Fine-tuning method (qlora, lora)  |
| `--input-urls strings`         | []string |         | URLs to training data             |
| `--output-image string`        | string   |         | Output image for fine-tuned model |
| `--output-image-secret string` | string   |         | Secret for pushing output image   |

### Advanced Flags

| Flag                             | Type | Description          |
| -------------------------------- | ---- | -------------------- |
| `--node-selector stringToString` | map  | Node selector labels |

## Examples

### Basic Inference Deployment

```bash
# Deploy Llama-2 7B for inference
kubectl kaito deploy --workspace-name llama-workspace --model llama-2-7b
```

### Deployment with Specific Instance Type

```bash
# Deploy with specific instance type and count  
kubectl kaito deploy \
  --workspace-name phi-workspace \
  --model phi-3.5-mini-instruct \
  --instance-type Standard_NC6s_v3 \
  --count 2
```

### Fine-tuning Deployment

```bash
# Deploy for fine-tuning with QLoRA
kubectl kaito deploy \
  --workspace-name tune-phi \
  --model phi-3.5-mini-instruct \
  --tuning \
  --tuning-method qlora \
  --input-urls "https://example.com/data.parquet" \
  --output-image myregistry/phi-finetuned:latest
```

### External Access Deployment

```bash
# Deploy with load balancer for external access
kubectl kaito deploy \
  --workspace-name public-llama \
  --model llama-2-7b \
  --enable-load-balancer
```

### Dry Run

```bash
# Preview what would be created
kubectl kaito deploy \
  --workspace-name test-workspace \
  --model phi-3.5-mini-instruct \
  --dry-run
```

### Node Selector Deployment

```bash
# Deploy on specific nodes
kubectl kaito deploy \
  --workspace-name selective-workspace \
  --model llama-2-7b \
  --node-selector gpu-type=A100,zone=us-west-2a
```

## Supported Models

Use `kubectl kaito models list` to see all supported models. Common models include:

- `llama-2-7b`, `llama-2-13b`, `llama-2-70b`
- `phi-3.5-mini-instruct`, `phi-3.5-medium-instruct`
- `mistral-7b-instruct`, `mixtral-8x7b-instruct`
- `falcon-7b`, `falcon-40b`

## Instance Types

Common GPU instance types:

- `Standard_NC6s_v3` - 1x V100 (16GB)
- `Standard_NC12s_v3` - 2x V100 (32GB total)
- `Standard_NC24s_v3` - 4x V100 (64GB total)
- `Standard_NC24ads_A100_v4` - 1x A100 (80GB)

## Fine-tuning Methods

- `qlora` - QLoRA (Quantized Low-Rank Adaptation) - Memory efficient
- `lora` - LoRA (Low-Rank Adaptation) - Standard approach

## Output

The command creates a Kaito Workspace resource in the specified namespace. You can monitor the deployment status using:

```bash
kubectl kaito status --workspace-name <workspace-name>
```
