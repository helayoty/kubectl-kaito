# End-to-End Tests for kubectl-kaito

This directory contains end-to-end tests for the kubectl-kaito plugin using real Kubernetes clusters.

## Overview

The e2e tests create and use real clusters to validate the kubectl-kaito plugin functionality:

- **Kind Cluster**: Tests basic functionality with CPU nodes and nginx deployment
- **AKS Cluster**: Tests GPU-specific functionality and chat capabilities

## Prerequisites

### Required Tools

1. **Go** (1.19+) - for building the binary
2. **kubectl** - for interacting with clusters
3. **kind** - for local Kubernetes testing
4. **Azure CLI** - for AKS cluster operations
5. **Docker** - required by kind

### Installation Commands

```bash
# Install kind
GO111MODULE="on" go install sigs.k8s.io/kind@latest

# Install Azure CLI (Ubuntu/Debian)
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Install Azure CLI (macOS)
brew install azure-cli

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
```

### Azure Setup

1. **Login to Azure**:
   ```bash
   az login
   ```

2. **Set default subscription** (if you have multiple):
   ```bash
   az account set --subscription "your-subscription-id"
   ```

3. **Register required providers**:
   ```bash
   az provider register --namespace Microsoft.ContainerService
   az provider register --namespace Microsoft.Compute
   ```

## Test Categories

### 1. Basic Tests (No Cluster Required)
- Help command validation
- Models list/describe functionality
- Input validation

### 2. Kind Cluster Tests
- Creates a local Kind cluster with CPU nodes
- Deploys nginx for basic functionality testing
- Tests dry-run operations
- Tests status command with no workspaces

### 3. AKS Cluster Tests
- Creates an AKS cluster with GPU nodes (Standard_NC6s_v3)
- Tests validation for GPU-specific operations
- Tests chat and RAG functionality validation
- **Note**: These tests create billable Azure resources

## Running Tests

### All Tests
```bash
cd e2e
go test -v -timeout 30m
```

### Basic Tests Only
```bash
cd e2e
go test -v -run TestBasicHelp -timeout 5m
go test -v -run TestModelsCommand -timeout 5m
go test -v -run TestInputValidation -timeout 5m
```

### Kind Tests Only
```bash
cd e2e
go test -v -run TestKindClusterOperations -timeout 15m
```

### AKS Tests Only
```bash
cd e2e
go test -v -run TestAKSClusterOperations -timeout 30m
```

## Test Environment Variables

You can customize the test behavior with environment variables:

```bash
# Skip Kind tests
export SKIP_KIND_TESTS=true

# Skip AKS tests
export SKIP_AKS_TESTS=true

# Custom AKS resource group name
export AKS_RESOURCE_GROUP=my-kaito-e2e-rg

# Custom AKS location
export AKS_LOCATION=eastus

# Custom cluster names
export KIND_CLUSTER_NAME=my-kaito-kind
export AKS_CLUSTER_NAME=my-kaito-aks
```

## Cost Considerations

### AKS Cluster Costs
The AKS tests create the following billable resources:
- **AKS Control Plane**: ~$0.10/hour
- **Standard_NC6s_v3 VM**: ~$0.90/hour (GPU node)
- **Disk Storage**: ~$0.05/month per GB
- **Load Balancer**: ~$0.025/hour

**Estimated cost for 30-minute test run**: ~$0.50-1.00

### Cost Optimization
- Tests automatically clean up resources after completion
- Use `SKIP_AKS_TESTS=true` to avoid costs during development
- Consider running AKS tests only in CI/CD pipelines

## Cluster Specifications

### Kind Cluster
```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
  extraMounts:
  - hostPath: /var/run/docker.sock
    containerPath: /var/run/docker.sock
```

### AKS Cluster
- **Node VM Size**: Standard_NC6s_v3 (1 GPU, 6 cores, 112 GB RAM)
- **Node Count**: 1
- **Kubernetes Version**: 1.28.0
- **Add-ons**: monitoring
- **Features**: GPU support for AI/ML workloads

## Test Output

### Successful Run Example
```
=== RUN   TestBasicHelp
=== RUN   TestBasicHelp/root_help
=== RUN   TestBasicHelp/models_help
--- PASS: TestBasicHelp (0.50s)

=== RUN   TestKindClusterOperations
✓ kind is available
Creating Kind cluster: kaito-e2e-kind
Kind cluster created successfully
Waiting for cluster to be ready: kind-kaito-e2e-kind
Cluster is ready with 2 nodes
Deploying nginx to Kind cluster
Deployment default/nginx-test is ready
--- PASS: TestKindClusterOperations (45.2s)

=== RUN   TestAKSClusterOperations
✓ Azure CLI is available
✓ Azure CLI is authenticated
Creating AKS cluster: kaito-e2e-aks
AKS cluster created successfully
--- PASS: TestAKSClusterOperations (420.8s)
```

## Troubleshooting

### Common Issues

1. **Kind cluster creation fails**:
   ```bash
   # Check Docker is running
   docker version
   
   # Clean up any existing clusters
   kind delete cluster --name kaito-e2e-kind
   ```

2. **AKS authentication fails**:
   ```bash
   # Re-login to Azure
   az login
   
   # Check current account
   az account show
   ```

3. **GPU quota issues**:
   ```bash
   # Check quota in region
   az vm list-usage --location westus2 --query "[?contains(name.value, 'NC')]"
   
   # Request quota increase if needed
   ```

4. **kubectl context issues**:
   ```bash
   # List available contexts
   kubectl config get-contexts
   
   # Manually switch context
   kubectl config use-context kind-kaito-e2e-kind
   ```

### Debug Mode

For verbose output and debugging:

```bash
cd e2e
go test -v -run TestKindClusterOperations -timeout 15m -args -test.v
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  e2e-basic:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    - name: Run basic tests
      run: |
        cd e2e
        go test -v -run TestBasicHelp -timeout 5m
        go test -v -run TestModelsCommand -timeout 5m

  e2e-kind:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    - name: Setup Kind
      uses: helm/kind-action@v1.4.0
      with:
        install_only: true
    - name: Run Kind tests
      run: |
        cd e2e
        go test -v -run TestKindClusterOperations -timeout 15m

  e2e-aks:
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    - name: Azure Login
      uses: azure/login@v1
      with:
        creds: ${{ secrets.AZURE_CREDENTIALS }}
    - name: Run AKS tests
      run: |
        cd e2e
        go test -v -run TestAKSClusterOperations -timeout 30m
```

## Cleanup

If tests fail and don't clean up properly:

```bash
# Clean up Kind clusters
kind delete cluster --name kaito-e2e-kind

# Clean up AKS resources
az group delete --name kaito-e2e-rg --yes --no-wait

# Reset kubectl context
kubectl config use-context docker-desktop  # or your default context
```