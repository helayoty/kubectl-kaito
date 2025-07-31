# kubectl kaito get-endpoint

Get the inference endpoint URL for a deployed Kaito workspace.

## Synopsis

Get the inference endpoint URL for a deployed Kaito workspace. This command retrieves the service endpoint that can be used to send inference requests to the deployed model. The endpoint supports OpenAI-compatible APIs.

## Usage

```bash
kaito get-endpoint [flags]
```

## Flags

| Flag                      | Type   | Default | Description                                  |
| ------------------------- | ------ | ------- | -------------------------------------------- |
| `--workspace-name string` | string |         | Name of the workspace (required)             |
| `-n, --namespace string`  | string |         | Kubernetes namespace                         |
| `--format string`         | string | url     | Output format: url or json                   |
| `--external`              | bool   | false   | Get external endpoint (LoadBalancer/Ingress) |

## Examples

### Basic Endpoint Retrieval

```bash
# Get endpoint URL for a workspace
kubectl kaito get-endpoint --workspace-name my-workspace
```

Output:
```
http://my-workspace.default.svc.cluster.local/chat/completions
```

### JSON Format Output

```bash
# Get endpoint in JSON format with metadata
kubectl kaito get-endpoint --workspace-name my-workspace --format json
```

Output:
```json
{
  "workspace": "my-workspace",
  "namespace": "default",
  "endpoint": "http://my-workspace.default.svc.cluster.local/chat/completions",
  "type": "ClusterIP",
  "port": 80,
  "ready": true,
  "model": "llama-2-7b"
}
```

### External Endpoint

```bash
# Get external endpoint if available (LoadBalancer/Ingress)
kubectl kaito get-endpoint --workspace-name my-workspace --external
```

Output (when LoadBalancer is available):
```
http://20.123.45.67/chat/completions
```

### Cross-Namespace Access

```bash
# Get endpoint for workspace in different namespace
kubectl kaito get-endpoint \
  --workspace-name my-workspace \
  --namespace production
```

## Output Formats

### URL Format (default)

Returns just the endpoint URL, suitable for scripting:

```bash
ENDPOINT=$(kubectl kaito get-endpoint --workspace-name my-workspace)
curl -X POST "$ENDPOINT" -H "Content-Type: application/json" -d '{
  "messages": [{"role": "user", "content": "Hello!"}]
}'
```

### JSON Format

Returns detailed information about the endpoint:

```json
{
  "workspace": "my-workspace",
  "namespace": "default", 
  "endpoint": "http://my-workspace.default.svc.cluster.local/chat/completions",
  "type": "ClusterIP",
  "port": 80,
  "ready": true,
  "model": "llama-2-7b",
  "external_endpoint": "http://20.123.45.67/chat/completions"
}
```

## Endpoint Types

### Internal Endpoints (default)

- **ClusterIP**: `http://<workspace>.<namespace>.svc.cluster.local/chat/completions`
- Accessible only from within the Kubernetes cluster
- Default service type for security

### External Endpoints (--external flag)

- **LoadBalancer**: `http://<external-ip>/chat/completions`
- **NodePort**: `http://<node-ip>:<port>/chat/completions`
- **Ingress**: `http://<ingress-host>/chat/completions`
- Accessible from outside the cluster

## API Compatibility

The endpoints are OpenAI-compatible and support:

### Chat Completions API

```bash
POST /chat/completions
Content-Type: application/json

{
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello!"}
  ],
  "temperature": 0.7,
  "max_tokens": 150
}
```

### Completions API (some models)

```bash
POST /completions
Content-Type: application/json

{
  "prompt": "Once upon a time",
  "max_tokens": 150,
  "temperature": 0.7
}
```

## Usage Examples

### Using with curl

```bash
# Get the endpoint
ENDPOINT=$(kubectl kaito get-endpoint --workspace-name my-llama)

# Send a chat request
curl -X POST "$ENDPOINT" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "What is artificial intelligence?"}
    ],
    "max_tokens": 100
  }'
```

### Using with Python

```python
import requests
import subprocess

# Get endpoint from kubectl
result = subprocess.run([
    "kubectl", "kaito", "get-endpoint", 
    "--workspace-name", "my-llama"
], capture_output=True, text=True)

endpoint = result.stdout.strip()

# Make request
response = requests.post(endpoint, json={
    "messages": [
        {"role": "user", "content": "Hello!"}
    ]
})

print(response.json())
```

### Port Forwarding for Local Access

```bash
# Port forward the service for local access
kubectl port-forward service/my-workspace 8080:80

# Use local endpoint
curl -X POST "http://localhost:8080/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{"messages": [{"role": "user", "content": "Hello!"}]}'
```

## Troubleshooting

### Endpoint Not Ready

```bash
# Check workspace status first
kubectl kaito status --workspace-name my-workspace

# Ensure INFERENCEREADY is True before getting endpoint
```

### External Endpoint Not Available

```bash
# Check if LoadBalancer service exists
kubectl get svc -l workspace=my-workspace

# Deploy with LoadBalancer if needed
kubectl kaito deploy --workspace-name my-workspace --model llama-2-7b --enable-load-balancer
```

### Connection Issues

```bash
# Test internal connectivity from within cluster
kubectl run test-pod --rm -i --tty --image=curlimages/curl -- sh
# Then inside the pod:
curl -X POST "http://my-workspace.default.svc.cluster.local/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{"messages": [{"role": "user", "content": "test"}]}'
```
