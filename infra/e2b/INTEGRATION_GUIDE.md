# E2B and Trigger.dev Integration Guide

This guide explains how to integrate E2B sandbox execution with Trigger.dev for secure, isolated task execution in Spiderweb.

## Overview

The integration provides a complete sandbox solution for executing risky operations safely:

- **E2B Client**: Manages sandbox lifecycle and task execution
- **Trigger.dev Tasks**: Provides orchestration and scheduling
- **Security**: Isolated execution environments with resource limits
- **Monitoring**: Comprehensive logging and health checks

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

## Setup

### 1. Install Dependencies

```bash
cd infra/e2b
npm install
```

### 2. Configure Environment

Create a `.env` file in the `infra/e2b` directory:

```bash
E2B_API_KEY=your_e2b_api_key_here
E2B_TEMPLATE_ID=spiderweb-sandbox
E2B_DEFAULT_TIMEOUT=300000
```

### 3. Build the Integration

```bash
npm run build
```

## Usage

### Basic Task Execution

#### Shell Commands

```typescript
import { E2BTaskFactory, E2BTaskManager } from './infra/e2b/trigger-e2b-contract';

// Create a shell task
const shellTask = E2BTaskFactory.createShellTask(
  'task-001',
  'git status',
  {
    cwd: '/workspace',
    env: { GIT_CONFIG: 'safe' },
    timeout: 30000
  }
);

// Execute the task
const result = await E2BTaskManager.executeTask(shellTask);
console.log('Shell result:', result);
```

#### Code Execution

```typescript
// Execute Python code
const pythonTask = E2BTaskFactory.createCodeTask(
  'task-002',
  'python',
  `
import requests
response = requests.get('https://api.example.com/data')
print(response.json())
`,
  {
    cwd: '/workspace',
    env: { PYTHONPATH: '/workspace' },
    timeout: 60000
  }
);

const result = await E2BTaskManager.executeTask(pythonTask);
console.log('Python result:', result);
```

#### File Operations

```typescript
// Write a file
const writeTask = E2BTaskFactory.createFileTask(
  'task-003',
  'write',
  '/workspace/data.json',
  {
    content: JSON.stringify({ key: 'value' }),
    timeout: 10000
  }
);

const result = await E2BTaskManager.executeTask(writeTask);
console.log('Write result:', result);

// Read a file
const readTask = E2BTaskFactory.createFileTask(
  'task-004',
  'read',
  '/workspace/data.json'
);

const readResult = await E2BTaskManager.executeTask(readTask);
console.log('Read result:', readResult.result.output);
```

### Batch Execution

```typescript
import { E2BTaskManager } from './infra/e2b/trigger-e2b-contract';

const tasks = [
  E2BTaskFactory.createShellTask('batch-001', 'git status'),
  E2BTaskFactory.createCodeTask('batch-002', 'python', 'print("Hello from sandbox")'),
  E2BTaskFactory.createFileTask('batch-003', 'list', '/workspace')
];

// Execute in parallel
const results = await E2BTaskManager.executeBatch(tasks);
console.log('Batch results:', results);

// Execute sequentially
const sequentialResults = await E2BTaskManager.executeSequential(tasks);
console.log('Sequential results:', sequentialResults);
```

### Health Monitoring

```typescript
import { E2BTaskManager } from './infra/e2b/trigger-e2b-contract';

// Check health
const health = await E2BTaskManager.healthCheck();
console.log('E2B Health:', health);

// Cleanup
await E2BTaskManager.cleanup();
```

## Trigger.dev Integration

### Register Tasks

In your Trigger.dev configuration:

```typescript
import { E2BTasks } from './infra/e2b/trigger-e2b-contract';
import { TriggerClient } from '@trigger.dev/sdk';

const client = new TriggerClient({
  id: 'spiderweb-e2b',
  apiKey: process.env.TRIGGER_API_KEY
});

// Register all E2B tasks
client.register(E2BTasks.shell);
client.register(E2BTasks.code);
client.register(E2BTasks.file);
client.register(E2BTasks.orchestrator);
client.register(E2BTasks.healthCheck);
```

### Trigger Tasks

```typescript
import { E2BTaskFactory } from './infra/e2b/trigger-e2b-contract';

// Trigger a shell task
const shellTask = E2BTaskFactory.createShellTask(
  'trigger-001',
  'npm install',
  { cwd: '/workspace/project' }
);

const result = await client.trigger('e2b-shell', shellTask);
console.log('Triggered shell task:', result);
```

## Security Considerations

### Network Access

E2B templates should restrict network access:

```yaml
# e2b-template.yaml
network:
  allowed_hosts:
    - "*.github.com"
    - "*.npmjs.org"
    - "*.pypi.org"
    - "api.example.com"
```

### File System

Limit file system access to workspace directory:

```yaml
# e2b-template.yaml
filesystem:
  allowed_paths:
    - "/workspace"
    - "/tmp"
```

### Resource Limits

Set appropriate resource limits:

```yaml
# e2b-template.yaml
resources:
  cpu: 2
  memory: 4Gi
  storage: 10Gi
```

## Error Handling

The integration provides comprehensive error handling:

```typescript
try {
  const result = await E2BTaskManager.executeTask(task);
  if (result.success) {
    console.log('Task completed successfully');
    console.log('Output:', result.result.output);
  } else {
    console.error('Task failed:', result.result.error);
  }
} catch (error) {
  console.error('Execution error:', error.message);
}
```

## Monitoring and Logging

### Task Events

The E2B client emits events for monitoring:

```typescript
import { e2bClient } from './infra/e2b/e2b-client';

e2bClient.on('task_completed', (result) => {
  console.log('Task completed:', result.id, result.success);
});

e2bClient.on('task_failed', (error) => {
  console.error('Task failed:', error.id, error.error);
});

e2bClient.on('sandbox_created', (info) => {
  console.log('Sandbox created:', info.id);
});

e2bClient.on('sandbox_closed', (info) => {
  console.log('Sandbox closed:', info.id);
});
```

### Health Checks

Regular health checks ensure system reliability:

```typescript
setInterval(async () => {
  const health = await E2BTaskManager.healthCheck();
  if (!health.healthy) {
    console.error('E2B integration unhealthy:', health.details);
    // Alert or take corrective action
  }
}, 60000); // Check every minute
```

## Best Practices

### 1. Task Design

- Keep tasks focused and atomic
- Set appropriate timeouts
- Use metadata for tracking and debugging
- Handle errors gracefully

### 2. Resource Management

- Monitor sandbox usage
- Clean up resources regularly
- Set reasonable resource limits
- Use batch execution for related tasks

### 3. Security

- Validate all inputs before sandbox execution
- Use minimal required permissions
- Monitor for suspicious activity
- Regularly update sandbox templates

### 4. Monitoring

- Log all task executions
- Monitor resource usage
- Set up alerts for failures
- Track performance metrics

## Troubleshooting

### Common Issues

1. **API Key Errors**: Ensure `E2B_API_KEY` is set correctly
2. **Template Not Found**: Verify `E2B_TEMPLATE_ID` exists
3. **Timeout Errors**: Increase timeout for long-running tasks
4. **Network Errors**: Check template network configuration

### Debug Mode

Enable debug logging:

```bash
DEBUG=e2b:* npm run dev
```

### Health Checks

Run health checks to diagnose issues:

```typescript
const health = await E2BTaskManager.healthCheck();
console.log('Health status:', health);
```

## Examples

### Complete Example

```typescript
import { E2BTaskFactory, E2BTaskManager } from './infra/e2b/trigger-e2b-contract';

async function runPipeline() {
  try {
    // 1. Check health
    const health = await E2BTaskManager.healthCheck();
    if (!health.healthy) {
      throw new Error('E2B integration is not healthy');
    }

    // 2. Create tasks
    const tasks = [
      E2BTaskFactory.createShellTask('pipeline-001', 'git clone https://github.com/example/repo.git'),
      E2BTaskFactory.createCodeTask('pipeline-002', 'python', `
import os
os.chdir('/workspace/repo')
os.system('python setup.py install')
`),
      E2BTaskFactory.createShellTask('pipeline-003', 'npm install', { cwd: '/workspace/repo/frontend' })
    ];

    // 3. Execute tasks sequentially
    const results = await E2BTaskManager.executeSequential(tasks);

    // 4. Check results
    const successCount = results.filter(r => r.success).length;
    console.log(`Pipeline completed: ${successCount}/${results.length} tasks succeeded`);

    return results;
  } catch (error) {
    console.error('Pipeline failed:', error.message);
    throw error;
  }
}

// Run the pipeline
runPipeline().catch(console.error);
```

## Future Enhancements

1. **Custom Templates**: Allow users to define custom sandbox templates
2. **Persistent Storage**: Add support for persistent data between runs
3. **Resource Monitoring**: Real-time resource usage monitoring
4. **Batch Operations**: Enhanced batch execution with dependency management
5. **Caching**: Cache frequently used dependencies and templates

## Support

For issues and questions:

1. Check the troubleshooting section
2. Review the health check output
3. Enable debug logging
4. Check E2B and Trigger.dev documentation
5. Report issues in the project repository