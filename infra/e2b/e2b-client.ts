import { E2BClient as E2BApiClient } from '@e2b/sdk';
import { EventEmitter } from 'events';

// Node.js type definitions
declare global {
  namespace NodeJS {
    interface ProcessEnv {
      E2B_API_KEY?: string;
      E2B_TEMPLATE_ID?: string;
      E2B_DEFAULT_TIMEOUT?: string;
    }
  }
}

export interface E2BTaskRequest {
  id: string;
  type: 'shell' | 'code' | 'file';
  timeout?: number;
  metadata?: Record<string, any>;
}

export interface ShellTask extends E2BTaskRequest {
  type: 'shell';
  command: string;
  cwd?: string;
  env?: Record<string, string>;
}

export interface CodeTask extends E2BTaskRequest {
  type: 'code';
  language: 'python' | 'node' | 'bash';
  code: string;
  cwd?: string;
  env?: Record<string, string>;
}

export interface FileTask extends E2BTaskRequest {
  type: 'file';
  operation: 'write' | 'read' | 'delete' | 'list';
  path: string;
  content?: string;
}

export interface E2BTaskResult {
  id: string;
  success: boolean;
  output?: string;
  error?: string;
  exitCode?: number;
  metadata?: Record<string, any>;
  duration: number;
}

export interface E2BConfig {
  apiKey: string;
  templateId: string;
  defaultTimeout: number;
  maxRetries: number;
  retryDelay: number;
}

export class E2BClient extends EventEmitter {
  private config: E2BConfig;
  private apiClient: E2BApiClient | null = null;
  private activeSandboxes: Map<string, any> = new Map();
  private isInitialized = false;

  constructor(config: Partial<E2BConfig> = {}) {
    super();
    
    this.config = {
      apiKey: config.apiKey || process.env.E2B_API_KEY || '',
      templateId: config.templateId || process.env.E2B_TEMPLATE_ID || 'spiderweb-sandbox',
      defaultTimeout: config.defaultTimeout || parseInt(process.env.E2B_DEFAULT_TIMEOUT || '300000'),
      maxRetries: config.maxRetries || 3,
      retryDelay: config.retryDelay || 1000,
      ...config
    };

    if (!this.config.apiKey) {
      throw new Error('E2B API key is required. Set E2B_API_KEY environment variable or pass it in config.');
    }
  }

  async initialize(): Promise<void> {
    if (this.isInitialized) return;

    try {
      this.apiClient = new E2BApiClient({
        apiKey: this.config.apiKey,
        templateId: this.config.templateId,
      });

      this.isInitialized = true;
      this.emit('initialized');
      
      console.log(`E2B client initialized with template: ${this.config.templateId}`);
    } catch (error) {
      this.emit('error', new Error(`Failed to initialize E2B client: ${error.message}`));
      throw error;
    }
  }

  async executeTask(task: ShellTask | CodeTask | FileTask): Promise<E2BTaskResult> {
    if (!this.isInitialized) {
      await this.initialize();
    }

    const startTime = Date.now();
    let attempt = 0;

    while (attempt <= this.config.maxRetries) {
      try {
        let result: E2BTaskResult;

        switch (task.type) {
          case 'shell':
            result = await this.executeShellTask(task as ShellTask);
            break;
          case 'code':
            result = await this.executeCodeTask(task as CodeTask);
            break;
          case 'file':
            result = await this.executeFileTask(task as FileTask);
            break;
          default:
            throw new Error(`Unknown task type: ${(task as any).type}`);
        }

        result.duration = Date.now() - startTime;
        this.emit('task_completed', result);
        
        return result;

      } catch (error) {
        attempt++;
        
        if (attempt > this.config.maxRetries) {
          const finalError = {
            id: task.id,
            success: false,
            error: error.message,
            duration: Date.now() - startTime,
            metadata: { attempt, maxRetries: this.config.maxRetries }
          };
          
          this.emit('task_failed', finalError);
          return finalError;
        }

        // Wait before retry
        await this.delay(this.config.retryDelay * attempt);
      }
    }

    throw new Error('Task execution failed after all retries');
  }

  private async executeShellTask(task: ShellTask): Promise<E2BTaskResult> {
    const sandbox = await this.createSandbox(task.timeout || this.config.defaultTimeout);
    
    try {
      const proc = await sandbox.process.start({
        cmd: task.command,
        cwd: task.cwd,
        env: task.env,
      });

      const output = await proc.output;
      
      return {
        id: task.id,
        success: proc.exitCode === 0,
        output: output.stdout,
        error: output.stderr,
        exitCode: proc.exitCode,
        metadata: task.metadata,
        duration: 0, // Will be set by parent method
      };
    } finally {
      await this.cleanupSandbox(sandbox);
    }
  }

  private async executeCodeTask(task: CodeTask): Promise<E2BTaskResult> {
    const sandbox = await this.createSandbox(task.timeout || this.config.defaultTimeout);
    
    try {
      // Write code to a temporary file
      const tempFile = `/tmp/${task.id}.${this.getFileExtension(task.language)}`;
      await sandbox.filesystem.write(tempFile, task.code);

      // Execute the code
      const command = this.getExecutionCommand(task.language, tempFile);
      const proc = await sandbox.process.start({
        cmd: command,
        cwd: task.cwd || '/tmp',
        env: task.env,
      });

      const output = await proc.output;
      
      return {
        id: task.id,
        success: proc.exitCode === 0,
        output: output.stdout,
        error: output.stderr,
        exitCode: proc.exitCode,
        metadata: task.metadata,
        duration: 0, // Will be set by parent method
      };
    } finally {
      await this.cleanupSandbox(sandbox);
    }
  }

  private async executeFileTask(task: FileTask): Promise<E2BTaskResult> {
    const sandbox = await this.createSandbox(task.timeout || this.config.defaultTimeout);
    
    try {
      let result: E2BTaskResult;

      switch (task.operation) {
        case 'write':
          if (!task.content) {
            throw new Error('File content is required for write operation');
          }
          await sandbox.filesystem.write(task.path, task.content);
          result = {
            id: task.id,
            success: true,
            output: `File written successfully: ${task.path}`,
            metadata: task.metadata,
            duration: 0,
          };
          break;

        case 'read':
          const content = await sandbox.filesystem.read(task.path);
          result = {
            id: task.id,
            success: true,
            output: content,
            metadata: task.metadata,
            duration: 0,
          };
          break;

        case 'delete':
          await sandbox.filesystem.remove(task.path);
          result = {
            id: task.id,
            success: true,
            output: `File deleted successfully: ${task.path}`,
            metadata: task.metadata,
            duration: 0,
          };
          break;

        case 'list':
          const files = await sandbox.filesystem.list(task.path);
          result = {
            id: task.id,
            success: true,
            output: JSON.stringify(files, null, 2),
            metadata: task.metadata,
            duration: 0,
          };
          break;

        default:
          throw new Error(`Unknown file operation: ${task.operation}`);
      }

      return result;
    } finally {
      await this.cleanupSandbox(sandbox);
    }
  }

  private async createSandbox(timeout: number): Promise<any> {
    if (!this.apiClient) {
      throw new Error('E2B client not initialized');
    }

    const sandbox = await this.apiClient.run({
      templateId: this.config.templateId,
      timeout: timeout,
    });

    this.activeSandboxes.set(sandbox.id, sandbox);
    this.emit('sandbox_created', { id: sandbox.id, timeout });

    return sandbox;
  }

  private async cleanupSandbox(sandbox: any): Promise<void> {
    try {
      await sandbox.close();
      this.activeSandboxes.delete(sandbox.id);
      this.emit('sandbox_closed', { id: sandbox.id });
    } catch (error) {
      this.emit('error', new Error(`Failed to cleanup sandbox ${sandbox.id}: ${error.message}`));
    }
  }

  private getFileExtension(language: string): string {
    switch (language) {
      case 'python': return 'py';
      case 'node': return 'js';
      case 'bash': return 'sh';
      default: return 'txt';
    }
  }

  private getExecutionCommand(language: string, filePath: string): string {
    switch (language) {
      case 'python': return `python3 ${filePath}`;
      case 'node': return `node ${filePath}`;
      case 'bash': return `bash ${filePath}`;
      default: return `cat ${filePath}`;
    }
  }

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  async listActiveSandboxes(): Promise<string[]> {
    return Array.from(this.activeSandboxes.keys());
  }

  async closeAllSandboxes(): Promise<void> {
    const promises = Array.from(this.activeSandboxes.values()).map(sandbox => 
      this.cleanupSandbox(sandbox)
    );
    await Promise.all(promises);
  }

  async healthCheck(): Promise<boolean> {
    try {
      if (!this.isInitialized) {
        await this.initialize();
      }

      // Create a simple test sandbox
      const sandbox = await this.createSandbox(10000);
      await sandbox.process.start({ cmd: 'echo "health check"' });
      await this.cleanupSandbox(sandbox);

      return true;
    } catch (error) {
      this.emit('error', new Error(`E2B health check failed: ${error.message}`));
      return false;
    }
  }

  destroy(): void {
    this.closeAllSandboxes();
    this.isInitialized = false;
    this.removeAllListeners();
  }
}

// Export default instance for easy use
export const e2bClient = new E2BClient();

// Auto-initialize on import
e2bClient.initialize().catch(error => {
  console.error('Failed to auto-initialize E2B client:', error.message);
});