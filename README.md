# kubectl-kaito

[![Go Version](https://img.shields.io/github/go-mod/go-version/kaito-project/kubectl-kaito)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/kaito-project/kubectl-kaito)](https://github.com/kaito-project/kubectl-kaito/releases)
[![CodeQL](https://github.com/kaito-project/kubectl-kaito/actions/workflows/codeql.yml/badge.svg)](https://github.com/kaito-project/kubectl-kaito/actions/workflows/codeql.yml)

A powerful kubectl plugin for deploying and managing AI/ML models in Kubernetes using the [Kaito AI Toolchain Operator](https://github.com/kaito-project/kaito).

## 🚀 What is kubectl-kaito?

kubectl-kaito simplifies AI model deployment on Kubernetes by providing:

- **🤖 One-command model deployment** - Deploy popular LLMs like Llama, Falcon, and Phi with a single command
- **⚡ Automatic GPU provisioning** - Intelligent GPU node scaling and resource management
- **🔧 Model fine-tuning** - Easy fine-tuning with QLora/LoRA using your custom datasets
- **📊 Real-time monitoring** - Workspace status tracking and log streaming
- **🎯 Zero configuration** - Works with existing kubectl setup and cluster contexts

## 📋 Prerequisites

| Requirement            | Version | Installation                                                             |
| ---------------------- | ------- | ------------------------------------------------------------------------ |
| **kubectl**            | v1.20+  | [Install Guide](https://kubernetes.io/docs/tasks/tools/install-kubectl/) |
| **Kaito Operator**     | Latest  | [Install Guide](https://github.com/kaito-project/kaito#installation)     |
| **Kubernetes Cluster** | v1.24+  | With GPU support (Azure/AWS/GCP)                                         |

**Quick verification:**
```bash
kubectl version --client
kubectl get nodes  # Should show GPU-capable nodes
```

## 📦 Installation

### Via Krew (Recommended)

```bash
# Install the plugin
kubectl krew install kaito

# Verify installation
kubectl kaito version
```

### Manual Installation

**Linux/macOS:**
```bash
# Download latest release
KAITO_VERSION=$(curl -s https://api.github.com/repos/kaito-project/kubectl-kaito/releases/latest | grep '"tag_name"' | cut -d'"' -f4)
curl -LO "https://github.com/kaito-project/kubectl-kaito/releases/download/${KAITO_VERSION}/kubectl-kaito-${KAITO_VERSION}-linux-amd64.tar.gz"

# Install
tar -xzf kubectl-kaito-${KAITO_VERSION}-linux-amd64.tar.gz
sudo mv kubectl-kaito /usr/local/bin/
chmod +x /usr/local/bin/kubectl-kaito
```

**Build from source:**
```bash
git clone https://github.com/kaito-project/kubectl-kaito.git
cd kubectl-kaito
make build
sudo cp bin/kubectl-kaito /usr/local/bin/
```

## 🚀 Quick Start

### Install the Plugin

```bash
# Via krew (kubectl plugin manager)
kubectl krew install kaito

# Verify installation
kubectl kaito version
```

### Deploy Your First Model

```bash
# Check available models
kubectl kaito preset list

# Deploy Llama-2 model for inference
kubectl kaito deploy --name my-llama --model llama-2-7b

# Preview deployment (dry-run)
kubectl kaito deploy --name test-workspace --model llama-2-7b --dry-run

# Deploy with specific configuration
kubectl kaito deploy --name llama-workspace --model llama-2-7b-chat --gpus 2 --preset chat

# Deploy larger model with specific instance type
kubectl kaito deploy --name gpu-model --model llama-2-70b-chat --instance-type Standard_NC24ads_A100_v4
```

## 🛠️ Core Commands

### Deploy Models for Inference

```bash
# Deploy popular models
kubectl kaito deploy --name llama-workspace --model llama-3-8b-instruct
kubectl kaito deploy --name falcon-workspace --model falcon-7b-instruct
kubectl kaito deploy --name phi-workspace --model phi-3.5-mini-instruct

# Deploy with specific GPU requirements
kubectl kaito deploy --name gpu-model --model llama-3-70b-instruct --instance-type Standard_NC24ads_A100_v4
```

### Fine-tune Models

```bash
# Fine-tune with your dataset
kubectl kaito tune --name custom-llama --model llama-2-7b --dataset gs://my-bucket/training-data --preset qlora

# Fine-tune with specific configuration
kubectl kaito tune --name tuned-falcon --model falcon-7b --dataset s3://data/custom --instance-type Standard_NC12s_v3

# Preview fine-tuning configuration
kubectl kaito tune --name test-tune --model phi-2 --dataset gs://test-data --preset lora --dry-run
```

### Monitor and Debug

```bash
# Check workspace status
kubectl kaito status                    # All workspaces
kubectl kaito status my-workspace       # Specific workspace
kubectl kaito status --all-namespaces   # Cross-namespace

# Stream logs
kubectl kaito logs my-workspace --follow --tail 100

# View workspace details
kubectl kaito describe my-workspace
```

### Manage Workspaces

```bash
# List all workspaces
kubectl kaito list

# Delete workspace
kubectl kaito delete my-workspace

# Force delete all workspaces in namespace
kubectl kaito delete --all --force
```

## 🤖 Supported AI Models

### Inference Models

| **Model Family** | **Available Models**                                                                                                                                                                     | **Recommended GPU** |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------- |
| **Llama**        | llama-2-7b, llama-2-7b-chat, llama-2-13b, llama-2-13b-chat, llama-2-70b, llama-2-70b-chat, llama-3-8b-instruct, llama-3-70b-instruct                                                     | A100                |
| **Falcon**       | falcon-7b, falcon-7b-instruct, falcon-40b, falcon-40b-instruct, falcon-180b, falcon-180b-chat                                                                                            | A100                |
| **Phi**          | phi-2, phi-3-mini-4k-instruct, phi-3-mini-128k-instruct, phi-3-small-8k-instruct, phi-3-small-128k-instruct, phi-3-medium-4k-instruct, phi-3-medium-128k-instruct, phi-3.5-mini-instruct | V100/A100           |
| **Mistral**      | mistral-7b, mistral-7b-instruct                                                                                                                                                          | V100/A100           |

### Fine-tuning Methods

- **QLora** - Quantized Low-Rank Adaptation (memory efficient)
- **LoRA** - Low-Rank Adaptation (standard approach)

### GPU Instance Types

| **Instance Type**          | **GPU**    | **Memory** | **Best For**            |
| -------------------------- | ---------- | ---------- | ----------------------- |
| `Standard_NC12s_v3`        | Tesla V100 | 112 GB     | Small models (7B)       |
| `Standard_NC24ads_A100_v4` | A100 40GB  | 440 GB     | Medium models (13B-40B) |
| `Standard_ND96asr_v4`      | A100 80GB  | 900 GB     | Large models (70B+)     |

## 🔧 Advanced Usage

### Multi-namespace Deployment

```bash
# Deploy to specific namespace
kubectl kaito deploy --name prod-model --model llama-3-8b-instruct --namespace production

# Monitor across namespaces
kubectl kaito status --all-namespaces

# Namespace-specific operations
kubectl kaito delete --all --namespace development
```

### Configuration and Context

```bash
# Use specific kubeconfig
kubectl kaito --kubeconfig=/path/to/config deploy --name test --model phi-2

# Use different cluster context
kubectl kaito --context=prod-cluster status

# Set default namespace
kubectl config set-context --current --namespace=ai-workloads
```

### Batch Operations

```bash
# Deploy multiple models
kubectl kaito deploy --name llama-7b --model llama-2-7b &
kubectl kaito deploy --name llama-13b --model llama-2-13b &
kubectl kaito deploy --name falcon-7b --model falcon-7b-instruct &
wait

# Monitor all deployments
kubectl kaito status --watch
```

## 🔍 Troubleshooting

### Common Issues

<details>
<summary><b>🚨 Workspace stuck in "Pending" state</b></summary>

**Causes:**
- Insufficient GPU quota
- Node provisioning issues
- Resource constraints

**Solutions:**
```bash
# Check node status
kubectl get nodes

# View workspace events
kubectl describe workspace <workspace-name>

# Check machine provisioning
kubectl get machine

# Verify GPU quota (Azure)
az vm list-usage --location eastus --query "[?contains(name.value, 'Standard_NC')]"
```
</details>

<details>
<summary><b>🚨 Pod failures or crashes</b></summary>

**Diagnosis:**
```bash
# Check pod logs
kubectl kaito logs <workspace-name>

# View pod details
kubectl get pods -l app.kubernetes.io/name=<workspace-name>

# Check resource usage
kubectl top pods
```
</details>

<details>
<summary><b>🚨 Plugin not found after installation</b></summary>

**Solutions:**
```bash
# Verify Krew installation
kubectl krew version

# Check plugin list
kubectl krew list | grep kaito

# Reinstall if needed
kubectl krew uninstall kaito
kubectl krew install kaito
```
</details>

### Debug Commands

```bash
# Kaito operator status
kubectl get pods -n kube-system | grep kaito

# Workspace diagnostics
kubectl get workspace <name> -o yaml
kubectl get events --field-selector involvedObject.name=<workspace-name>

# Resource monitoring
kubectl top nodes
kubectl top pods
```

## 📖 Examples

### Complete ML Workflow

```bash
#!/bin/bash
# Complete AI model deployment workflow

# 0. Check plugin version
echo "📋 Checking kubectl-kaito version..."
kubectl kaito version

# 1. Explore available models
echo "🔍 Discovering available models..."
kubectl kaito preset list --model llama

# 2. Preview deployment
echo "🔍 Previewing deployment..."
kubectl kaito deploy --name llama-inference --model llama-2-7b-chat --dry-run

# 3. Deploy base model for inference
echo "🚀 Deploying Llama-2 for inference..."
kubectl kaito deploy --name llama-inference --model llama-2-7b-chat

# 4. Wait for deployment
echo "⏳ Waiting for deployment..."
kubectl kaito status llama-inference --watch

# 5. Fine-tune with custom data
echo "🔧 Starting fine-tuning..."
kubectl kaito tune --name llama-custom --model llama-2-7b --dataset gs://my-data --preset qlora

# 6. Monitor both workspaces
echo "📊 Monitoring workspaces..."
kubectl kaito status --all-namespaces

# 7. Clean up
read -p "Press enter to clean up resources..."
kubectl kaito delete llama-inference
kubectl kaito delete llama-custom
```

### Production Deployment

```bash
# Production-ready deployment with monitoring
kubectl kaito deploy \
  --name production-llama \
  --model llama-2-70b-chat \
  --instance-type Standard_ND96asr_v4 \
  --namespace production

# Set up monitoring
kubectl kaito status production-llama --watch &
kubectl kaito logs production-llama --follow > llama-prod.log &
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md).

**Quick contributing steps:**
1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes and add tests
4. Run tests: `make test`
5. Submit a pull request

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🔗 Related Projects

- **[Kaito](https://github.com/kaito-project/kaito)** - Kubernetes AI Toolchain Operator
- **[GPU Provisioner](https://github.com/Azure/gpu-provisioner)** - Automatic GPU node provisioning
- **[Karpenter](https://github.com/aws/karpenter)** - Kubernetes node lifecycle management

## 📞 Support

- 📚 **Documentation**: [Kaito Documentation](https://github.com/kaito-project/kaito/docs)
- 🐛 **Issues**: [GitHub Issues](https://github.com/kaito-project/kubectl-kaito/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/kaito-project/kubectl-kaito/discussions)
- 📧 **Security**: [Security Policy](SECURITY.md)

---

<div align="center">

**[⭐ Star this project](https://github.com/kaito-project/kubectl-kaito)** if you find it useful!

Made with ❤️ by the [Kaito Project](https://github.com/kaito-project) team

</div> 