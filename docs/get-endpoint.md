# kubectl kaito get-endpoint

Get the inference endpoint URL for a deployed Kaito workspace.

## Synopsis

Get the inference endpoint URL for a deployed Kaito workspace. This command retrieves all available service endpoints that can be used to send inference requests to the deployed model. The endpoints support OpenAI-compatible APIs.

The command automatically discovers all accessible endpoints:

- **LoadBalancer**: Direct public access (if configured)
- **API Proxy**: Kubernetes API proxy (works anywhere kubectl works)  
- **Cluster-internal**: Direct cluster access (for pods only)

The URL format returns the best available endpoint (prefers external if available), while JSON format shows all discovered endpoints with detailed information.

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

## Examples

### Basic Endpoint Retrieval

```bash
# Get endpoint URL for a workspace
kubectl kaito get-endpoint --workspace-name my-workspace
```

Output (from outside cluster):

```
https://your-api-server.com/api/v1/namespaces/default/services/my-workspace:80/proxy
```

### JSON Format Output - All Endpoints

```bash
# Get all available endpoints in JSON format
kubectl kaito get-endpoint --workspace-name my-workspace --format json
```

Output (showing all available endpoints):

```json
{
  "workspace": "my-workspace",
  "namespace": "default",
  "endpoints": [
    {
      "url": "https://your-api-server.com/api/v1/namespaces/default/services/my-workspace:80/proxy",
      "type": "APIProxy",
      "access": "cluster",
      "description": "Kubernetes API proxy (works anywhere kubectl works)"
    }
  ]
}
```

With LoadBalancer (if configured):

```json
{
  "workspace": "my-workspace", 
  "namespace": "default",
  "endpoints": [
    {
      "url": "http://203.0.113.42:80",
      "type": "LoadBalancer",
      "access": "external",
      "description": "Direct public access via LoadBalancer"
    },
    {
      "url": "https://your-api-server.com/api/v1/namespaces/default/services/my-workspace:80/proxy",
      "type": "APIProxy", 
      "access": "cluster",
      "description": "Kubernetes API proxy (works anywhere kubectl works)"
    }
  ]
}
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

Returns the best available endpoint URL, suitable for scripting. Prefers external endpoints if available:

```bash
ENDPOINT=$(kubectl kaito get-endpoint --workspace-name my-workspace)
curl -X POST "$ENDPOINT" -H "Content-Type: application/json" -d '{
  "messages": [{"role": "user", "content": "Hello!"}]
}'
```

### JSON Format

Returns all available endpoints with detailed information:

```json
{
  "workspace": "my-workspace",
  "namespace": "default",
  "endpoints": [
    {
      "url": "https://your-api-server.com/api/v1/namespaces/default/services/my-workspace:80/proxy",
      "type": "APIProxy",
      "access": "cluster", 
      "description": "Kubernetes API proxy (works anywhere kubectl works)"
    }
  ]
}
```

## Endpoint Types

The command automatically discovers and returns all available endpoint types:

### API Proxy (cluster access)

- **Format**: `https://<api-server>/api/v1/namespaces/<namespace>/services/<workspace>:80/proxy`
- **Authentication**: Uses your kubectl credentials
- **Access**: Works anywhere kubectl works (local machine, CI/CD, etc.)
- **Security**: Authenticated via Kubernetes RBAC

### LoadBalancer (external access)

- **Format**: `http://<external-ip>:80`
- **Authentication**: None (direct access)
- **Access**: Public internet access
- **Security**: Unprotected (configure firewall rules as needed)
- **Availability**: Only if service type is LoadBalancer

### Cluster-Internal (pod access)

- **Format**: `http://<workspace>.<namespace>.svc.cluster.local:80`
- **Authentication**: None (direct access)
- **Access**: Only from within the Kubernetes cluster
- **Security**: Protected by cluster network policies

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
