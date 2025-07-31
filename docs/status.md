# kubectl kaito status

Check the status of one or more Kaito workspaces.

## Synopsis

Check the status of Kaito workspaces, displaying the current state of workspace resources, including readiness conditions, resource allocation, and deployment status.

## Usage

```bash
kaito status [flags]
```

## Flags

| Flag                      | Type   | Default | Description                            |
| ------------------------- | ------ | ------- | -------------------------------------- |
| `--workspace-name string` | string |         | Name of the workspace to check         |
| `-A, --all-namespaces`    | bool   | false   | Check workspaces across all namespaces |
| `-n, --namespace string`  | string |         | Kubernetes namespace                   |
| `-w, --watch`             | bool   | false   | Watch for changes in real-time         |
| `--show-conditions`       | bool   | false   | Show detailed status conditions        |
| `--show-worker-nodes`     | bool   | false   | Show worker node information           |

## Examples

### Check Specific Workspace

```bash
# Check status of a specific workspace
kubectl kaito status --workspace-name my-workspace
```

### Check All Workspaces

```bash
# Check status of all workspaces in current namespace
kubectl kaito status

# Check status across all namespaces
kubectl kaito status --all-namespaces
```

### Watch for Changes

```bash
# Watch for changes in real-time
kubectl kaito status --workspace-name my-workspace --watch
```

### Detailed Status Information

```bash
# Show detailed conditions and worker node information
kubectl kaito status \
  --workspace-name my-workspace \
  --show-conditions \
  --show-worker-nodes
```

### Cross-Namespace Monitoring

```bash
# Monitor all workspaces across all namespaces with watch
kubectl kaito status --all-namespaces --watch
```

## Output Format

### Basic Output

```
NAMESPACE  NAME           NODECLAIM    RESOURCEREADY  INFERENCEREADY  WORKSPACEREADY  AGE
default    llama-workspace  nc-llama     True          True           True           5m
```

### Detailed Output (with --show-conditions)

```
Workspace: llama-workspace
Namespace: default
Age: 5m

Status Conditions:
  STATUS   MESSAGE                           LAST TRANSITION
  True     NodeClaim created successfully    2m ago
  True     Resources allocated               1m ago  
  True     Model inference ready             30s ago
  True     Workspace ready for requests      20s ago

Worker Nodes: (with --show-worker-nodes)
  NODE                    STATUS   GPU TYPE   MEMORY    READY
  aks-gpu-12345678-0     Ready    V100       16Gi      True
```

## Status Fields

### Workspace Status

- **NODECLAIM**: Name of the associated NodeClaim resource
- **RESOURCEREADY**: Whether GPU resources are allocated
- **INFERENCEREADY**: Whether the model inference is ready
- **WORKSPACEREADY**: Overall workspace readiness status
- **AGE**: Time since workspace creation

### Condition Types

When using `--show-conditions`, you'll see detailed conditions:

- **NodeClaimReady**: GPU node provisioning status
- **ResourceReady**: Resource allocation status  
- **InferenceReady**: Model loading and inference readiness
- **WorkspaceReady**: Overall workspace status

### Worker Node Information

When using `--show-worker-nodes`, additional details are shown:

- **NODE**: Kubernetes node name
- **STATUS**: Node status (Ready, NotReady)
- **GPU TYPE**: Type of GPU on the node
- **MEMORY**: Available GPU memory
- **READY**: Whether the node is ready for workloads

## Watch Mode

Use the `--watch` flag to monitor workspace status in real-time:

```bash
kubectl kaito status --workspace-name my-workspace --watch
```

This continuously updates the display as the workspace status changes, useful for monitoring deployments.

## Exit Codes

- **0**: Success
- **1**: Error (workspace not found, connection issues, etc.)

## Troubleshooting

### Common Status Issues

1. **RESOURCEREADY: False**
   - Check cluster has available GPU nodes
   - Verify instance type is available
   - Check node selectors and taints

2. **INFERENCEREADY: False**
   - Model might still be downloading
   - Check pod logs for model loading issues
   - Verify model configuration

3. **WORKSPACEREADY: False**
   - One or more conditions not met
   - Use `--show-conditions` for details

### Debugging Commands

```bash
# Get detailed conditions
kubectl kaito status --workspace-name <name> --show-conditions

# Watch deployment progress
kubectl kaito status --workspace-name <name> --watch

# Check underlying Kubernetes resources
kubectl get workspace <name> -o yaml
kubectl describe workspace <name>
```
