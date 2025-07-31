# kubectl kaito chat

Start an interactive chat session with a deployed Kaito workspace model.

## Synopsis

Start an interactive chat session with a deployed Kaito workspace model. This command provides a chat interface to interact with deployed models using OpenAI-compatible APIs in interactive mode.

## Usage

```bash
kaito chat [flags]
```

## Flags

| Flag                      | Type   | Default | Description                                   |
| ------------------------- | ------ | ------- | --------------------------------------------- |
| `--workspace-name string` | string |         | Name of the workspace (required)              |
| `-n, --namespace string`  | string |         | Kubernetes namespace                          |
| `--system-prompt string`  | string |         | System prompt for the conversation            |
| `--temperature float`     | float  | 0.7     | Temperature for response generation (0.0-2.0) |
| `--max-tokens int`        | int    | 1024    | Maximum tokens in response                    |
| `--top-p float`           | float  | 0.9     | Top-p (nucleus sampling) parameter (0.0-1.0)  |

## Examples

### Basic Interactive Chat

```bash
# Start interactive chat session
kubectl kaito chat --workspace-name my-llama
```

This opens an interactive session:
```
🤖 Connected to workspace: my-llama (llama-2-7b)
💬 Type your message (or 'quit' to exit):

> Hello! How are you?
Assistant: Hello! I'm doing well, thank you for asking. I'm here and ready to help you with any questions or tasks you might have. How can I assist you today?

> What is machine learning?
Assistant: Machine learning is a subset of artificial intelligence (AI) that focuses on creating systems that can learn and improve from experience without being explicitly programmed for every task...

> quit
👋 Goodbye!
```

### Configure Inference Parameters

```bash
# Configure inference parameters
kubectl kaito chat \
  --workspace-name my-llama \
  --temperature 0.5 \
  --max-tokens 512
```

### Use System Prompt

```bash
# Use system prompt for context
kubectl kaito chat \
  --workspace-name my-llama \
  --system-prompt "You are a helpful coding assistant"
```

Example with system prompt:
```
🤖 Connected to workspace: my-llama (llama-2-7b)
🎯 System prompt: You are a helpful coding assistant
💬 Type your message (or 'quit' to exit):

> How do I create a Python function?
Assistant: To create a Python function, you use the `def` keyword followed by the function name and parameters. Here's the basic syntax:

```python
def function_name(parameters):
    """Optional docstring"""
    # Function body
    return value  # Optional
```

For example:
```python
def greet(name):
    """Greets a person with their name"""
    return f"Hello, {name}!"

# Call the function
message = greet("Alice")
print(message)  # Output: Hello, Alice!
```
```

### Non-Interactive Usage (Pipe Input)

```bash
# Pipe input for non-interactive usage
echo "What is AI?" | kubectl kaito chat --workspace-name my-llama
```

Output:
```
Assistant: Artificial Intelligence (AI) refers to the simulation of human intelligence in machines that are programmed to think and learn like humans...
```

### Advanced Configuration

```bash
# Full configuration with all parameters
kubectl kaito chat \
  --workspace-name my-workspace \
  --namespace production \
  --system-prompt "You are an expert in Kubernetes and cloud computing" \
  --temperature 0.3 \
  --max-tokens 2048 \
  --top-p 0.95
```

## Interactive Commands

When in interactive mode, you can use these commands:

| Command          | Description                    |
| ---------------- | ------------------------------ |
| `quit` or `exit` | Exit the chat session          |
| `clear`          | Clear the conversation history |
| `help`           | Show available commands        |
| `status`         | Show current configuration     |

### Example Interactive Session

```
🤖 Connected to workspace: my-llama (llama-2-7b)
💬 Type your message (or 'quit' to exit):

> help
Available commands:
  quit/exit - Exit chat session
  clear     - Clear conversation history  
  help      - Show this help
  status    - Show current configuration

> status
Configuration:
  Workspace: my-llama
  Model: llama-2-7b
  Temperature: 0.7
  Max tokens: 1024
  Top-p: 0.9
  System prompt: (none)

> clear
🗑️ Conversation history cleared

> Hello again!
Assistant: Hello! How can I help you today?

> quit
👋 Goodbye!
```

## Parameters

### Temperature (0.0 - 2.0)

Controls randomness in responses:
- **0.0**: Deterministic, always picks most likely response
- **0.7**: Balanced creativity and coherence (default)
- **1.0**: More creative and varied responses
- **2.0**: Highly random, may be incoherent

### Max Tokens

Maximum number of tokens in the response:
- **Default**: 1024 tokens
- **Range**: 1 - model's maximum context length
- **Note**: Includes both input and output tokens

### Top-p (0.0 - 1.0)

Nucleus sampling parameter:
- **0.1**: Very focused, uses only top 10% probable tokens
- **0.9**: Balanced selection (default)
- **1.0**: Consider all possible tokens

### System Prompt

Sets the AI's behavior and context:
```bash
--system-prompt "You are a helpful assistant that responds in a professional tone"
--system-prompt "You are a coding expert. Provide code examples and explanations"
--system-prompt "You are a creative writer. Help with storytelling and narrative"
```

## Output Format

### Interactive Mode

```
🤖 Connected to workspace: <workspace-name> (<model-name>)
[🎯 System prompt: <system-prompt>]  # If system prompt is set
💬 Type your message (or 'quit' to exit):

> <user input>
Assistant: <model response>

> <user input>
Assistant: <model response>
```

### Non-Interactive Mode (Piped Input)

```
Assistant: <model response>
```

## Error Handling

### Workspace Not Ready

```
❌ Error: Workspace 'my-workspace' is not ready for inference
   Use 'kubectl kaito status --workspace-name my-workspace' to check status
```

### Connection Issues

```
❌ Error: Unable to connect to workspace endpoint
   Endpoint: http://my-workspace.default.svc.cluster.local/chat/completions
   Please check if the workspace is running and accessible
```

### Model Errors

```
❌ Model Error: Input too long (1500 tokens > 1024 max)
   Try reducing your message length or increasing --max-tokens
```

## Troubleshooting

### Check Workspace Status

```bash
# Ensure workspace is ready
kubectl kaito status --workspace-name my-workspace
```

### Test Endpoint

```bash
# Test if endpoint is accessible
kubectl kaito get-endpoint --workspace-name my-workspace
```

### Network Connectivity

```bash
# Port forward for testing
kubectl port-forward service/my-workspace 8080:80

# Test with curl
curl -X POST "http://localhost:8080/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "test"}],
    "max_tokens": 10
  }'
```
