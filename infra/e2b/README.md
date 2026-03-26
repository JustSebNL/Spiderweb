# E2B Sandbox Integration

This directory contains the E2B sandbox integration for Spiderweb, enabling secure execution of risky code, shell operations, and filesystem mutations.

## Overview

E2B provides isolated sandbox environments for executing untrusted code safely. This integration allows Spiderweb to offload risky operations to E2B while keeping the main runtime secure.

## Architecture

```
Spiderweb Runtime
    ↓ (Task Request)
Trigger.dev Orchestrator
    ↓ (Sandbox Execution)
E2B Sandbox Environment
    ↓ (Result)
Spiderweb Runtime
```

## Components

### 1. E2B Runtime Controller
- Manages E2B sandbox lifecycle
- Handles sandbox creation, execution, and cleanup
- Provides API for task submission and result retrieval

### 2. Task Contract
- Defines the interface between Spiderweb and E2B
- Specifies input/output formats for sandboxed tasks
- Handles error propagation and timeout management

### 3. Security Policies
- Defines what operations are allowed in the sandbox
- Manages resource limits and execution constraints
- Ensures proper isolation between tasks

## Usage

### Basic Task Execution

```typescript
import { E2BClient } from './e2b-client';

const client = new E2BClient();

// Execute a shell command
const result = await client.executeShell({
  command: 'git status',
  timeout: 30000,
  cwd: '/workspace'
});

console.log(result.stdout);
console.log(result.stderr);
console.log(result.exitCode);
```

### File Operations

```typescript
// Write files to sandbox
await client.writeFile({
  path: '/workspace/data.json',
  content: JSON.stringify({ key: 'value' })
});

// Read files from sandbox
const content = await client.readFile({
  path: '/workspace/output.txt'
});
```

### Code Execution

```typescript
// Execute Python code
const result = await client.executeCode({
  language: 'python',
  code: `
import requests
response = requests.get('https://api.example.com/data')
print(response.json())
`,
  timeout: 60000
});
```

## Configuration

### Environment Variables

```bash
E2B_API_KEY=your_e2b_api_key
E2B_TEMPLATE_ID=your_template_id
E2B_DEFAULT_TIMEOUT=300000  # 5 minutes
```

### Template Configuration

E2B templates define the sandbox environment:

```yaml
# e2b-template.yaml
template:
  id: spiderweb-sandbox
  description: "Spiderweb sandbox for risky operations"
  runtime:
    image: python:3.11-slim
    packages:
      - git
      - curl
      - wget
    environment:
      PYTHONPATH: /workspace
      NODE_OPTIONS: "--max-old-space-size=2048"
  resources:
    cpu: 2
    memory: 4Gi
    storage: 10Gi
  network:
    allowed_hosts:
      - "*.github.com"
      - "*.npmjs.org"
      - "*.pypi.org"
```

## Security Considerations

1. **Network Access**: Only allow necessary external connections
2. **File System**: Limit access to workspace directory only
3. **Process Limits**: Set appropriate CPU and memory limits
4. **Timeout Management**: Always set reasonable timeouts
5. **Input Validation**: Sanitize all inputs before sandbox execution

## Error Handling

The E2B integration provides comprehensive error handling:

- **Timeout Errors**: Tasks that exceed timeout limits
- **Resource Errors**: Tasks that exceed CPU/memory limits
- **Network Errors**: Failed network requests
- **Execution Errors**: Code execution failures
- **File System Errors**: File operation failures

## Monitoring and Logging

All E2B operations are logged for monitoring:

- Task submission and completion
- Resource usage metrics
- Error conditions and stack traces
- Sandbox lifecycle events

## Integration with Trigger.dev

The E2B integration works seamlessly with Trigger.dev:

1. Trigger.dev receives a task request
2. Determines if the task requires sandboxing
3. Submits the task to E2B if needed
4. Waits for completion and returns results
5. Handles retries and error conditions

## Development

### Local Testing

Use the E2B CLI for local development:

```bash
# Install E2B CLI
npm install -g @e2b/cli

# Create a sandbox
e2b sandbox create --template-id your-template-id

# Execute commands
e2b sandbox exec --sandbox-id your-sandbox-id --command "git status"

# Clean up
e2b sandbox delete --sandbox-id your-sandbox-id
```

### Testing

Run the test suite:

```bash
npm test
npm run test:e2b
```

### Debugging

Enable debug logging:

```bash
DEBUG=e2b:* npm run dev
```

## Future Enhancements

1. **Multi-language Support**: Extend beyond Python to other languages
2. **Persistent Storage**: Add support for persistent data between runs
3. **Custom Templates**: Allow users to define custom sandbox templates
4. **Resource Monitoring**: Real-time resource usage monitoring
5. **Batch Operations**: Support for batch task execution
6. **Caching**: Cache frequently used dependencies and templates