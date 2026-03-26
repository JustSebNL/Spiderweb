import { Task } from '@trigger.dev/sdk';
import { e2bClient, E2BTaskRequest, ShellTask, CodeTask, FileTask, E2BTaskResult } from './e2b-client';

/**
 * Trigger.dev task contract for E2B sandbox execution
 * 
 * This contract defines how Trigger.dev tasks can request E2B sandbox execution
 * for risky operations while maintaining proper error handling and result propagation.
 */

export interface TriggerE2BTaskRequest {
  id: string;
  type: 'e2b_shell' | 'e2b_code' | 'e2b_file';
  payload: ShellTask | CodeTask | FileTask;
  metadata?: {
    triggerTaskId?: string;
    spiderwebTaskId?: string;
    priority?: 'high' | 'medium' | 'low';
    tags?: string[];
  };
}

export interface TriggerE2BTaskResult {
  id: string;
  success: boolean;
  result: E2BTaskResult;
  metadata: {
    triggerTaskId?: string;
    spiderwebTaskId?: string;
    executionTime: number;
    sandboxId?: string;
  };
}

/**
 * E2B Shell Task for Trigger.dev
 * 
 * Executes shell commands in an isolated E2B sandbox environment.
 */
export const e2bShellTask = new Task({
  id: "e2b-shell",
  run: async (request: TriggerE2BTaskRequest): Promise<TriggerE2BTaskResult> => {
    const startTime = Date.now();
    
    if (request.type !== 'e2b_shell') {
      throw new Error(`Invalid task type for e2b-shell: ${request.type}`);
    }

    const shellRequest = request.payload as ShellTask;
    
    try {
      const result = await e2bClient.executeTask(shellRequest);
      
      return {
        id: request.id,
        success: result.success,
        result,
        metadata: {
          triggerTaskId: request.metadata?.triggerTaskId,
          spiderwebTaskId: request.metadata?.spiderwebTaskId,
          executionTime: Date.now() - startTime,
          sandboxId: result.metadata?.sandboxId,
        }
      };
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      return {
        id: request.id,
        success: false,
        result: {
          id: request.id,
          success: false,
          error: errorMessage,
          duration: Date.now() - startTime,
          metadata: request.metadata
        },
        metadata: {
          triggerTaskId: request.metadata?.triggerTaskId,
          spiderwebTaskId: request.metadata?.spiderwebTaskId,
          executionTime: Date.now() - startTime,
        }
      };
    }
  }
});

/**
 * E2B Code Execution Task for Trigger.dev
 * 
 * Executes code in various languages within an isolated E2B sandbox environment.
 */
export const e2bCodeTask = new Task({
  id: "e2b-code",
  run: async (request: TriggerE2BTaskRequest): Promise<TriggerE2BTaskResult> => {
    const startTime = Date.now();
    
    if (request.type !== 'e2b_code') {
      throw new Error(`Invalid task type for e2b-code: ${request.type}`);
    }

    const codeRequest = request.payload as CodeTask;
    
    try {
      const result = await e2bClient.executeTask(codeRequest);
      
      return {
        id: request.id,
        success: result.success,
        result,
        metadata: {
          triggerTaskId: request.metadata?.triggerTaskId,
          spiderwebTaskId: request.metadata?.spiderwebTaskId,
          executionTime: Date.now() - startTime,
          sandboxId: result.metadata?.sandboxId,
        }
      };
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      return {
        id: request.id,
        success: false,
        result: {
          id: request.id,
          success: false,
          error: errorMessage,
          duration: Date.now() - startTime,
          metadata: request.metadata
        },
        metadata: {
          triggerTaskId: request.metadata?.triggerTaskId,
          spiderwebTaskId: request.metadata?.spiderwebTaskId,
          executionTime: Date.now() - startTime,
        }
      };
    }
  }
});

/**
 * E2B File Operations Task for Trigger.dev
 * 
 * Performs file system operations within an isolated E2B sandbox environment.
 */
export const e2bFileTask = new Task({
  id: "e2b-file",
  run: async (request: TriggerE2BTaskRequest): Promise<TriggerE2BTaskResult> => {
    const startTime = Date.now();
    
    if (request.type !== 'e2b_file') {
      throw new Error(`Invalid task type for e2b-file: ${request.type}`);
    }

    const fileRequest = request.payload as FileTask;
    
    try {
      const result = await e2bClient.executeTask(fileRequest);
      
      return {
        id: request.id,
        success: result.success,
        result,
        metadata: {
          triggerTaskId: request.metadata?.triggerTaskId,
          spiderwebTaskId: request.metadata?.spiderwebTaskId,
          executionTime: Date.now() - startTime,
          sandboxId: result.metadata?.sandboxId,
        }
      };
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      return {
        id: request.id,
        success: false,
        result: {
          id: request.id,
          success: false,
          error: errorMessage,
          duration: Date.now() - startTime,
          metadata: request.metadata
        },
        metadata: {
          triggerTaskId: request.metadata?.triggerTaskId,
          spiderwebTaskId: request.metadata?.spiderwebTaskId,
          executionTime: Date.now() - startTime,
        }
      };
    }
  }
});

/**
 * E2B Orchestrator Task for Trigger.dev
 * 
 * Orchestrates multiple E2B tasks and provides batch execution capabilities.
 */
export const e2bOrchestratorTask = new Task({
  id: "e2b-orchestrator",
  run: async (requests: TriggerE2BTaskRequest[]): Promise<TriggerE2BTaskResult[]> => {
    const startTime = Date.now();
    const results: TriggerE2BTaskResult[] = [];
    
    for (const request of requests) {
      try {
        let taskResult: TriggerE2BTaskResult;
        
        switch (request.type) {
          case 'e2b_shell':
            taskResult = await e2bShellTask.trigger(request);
            break;
          case 'e2b_code':
            taskResult = await e2bCodeTask.trigger(request);
            break;
          case 'e2b_file':
            taskResult = await e2bFileTask.trigger(request);
            break;
          default:
            throw new Error(`Unknown E2B task type: ${request.type}`);
        }
        
        results.push(taskResult);
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        results.push({
          id: request.id,
          success: false,
          result: {
            id: request.id,
            success: false,
            error: errorMessage,
            duration: Date.now() - startTime,
            metadata: request.metadata
          },
          metadata: {
            triggerTaskId: request.metadata?.triggerTaskId,
            spiderwebTaskId: request.metadata?.spiderwebTaskId,
            executionTime: Date.now() - startTime,
          }
        });
      }
    }
    
    return results;
  }
});

/**
 * E2B Health Check Task for Trigger.dev
 * 
 * Performs health checks on the E2B integration.
 */
export const e2bHealthCheckTask = new Task({
  id: "e2b-health-check",
  run: async (): Promise<{ healthy: boolean; details: any }> => {
    try {
      const healthy = await e2bClient.healthCheck();
      const activeSandboxes = await e2bClient.listActiveSandboxes();
      
      return {
        healthy,
        details: {
          activeSandboxes,
          timestamp: new Date().toISOString(),
          clientInitialized: true
        }
      };
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      return {
        healthy: false,
        details: {
          error: errorMessage,
          timestamp: new Date().toISOString(),
          clientInitialized: false
        }
      };
    }
  }
});

/**
 * Utility functions for creating E2B task requests
 */
export const E2BTaskFactory = {
  createShellTask: (
    id: string,
    command: string,
    options: {
      cwd?: string;
      env?: Record<string, string>;
      timeout?: number;
      metadata?: Record<string, any>;
    } = {}
  ): TriggerE2BTaskRequest => ({
    id,
    type: 'e2b_shell',
    payload: {
      id,
      type: 'shell',
      command,
      cwd: options.cwd,
      env: options.env,
      timeout: options.timeout,
      metadata: options.metadata
    }
  }),

  createCodeTask: (
    id: string,
    language: 'python' | 'node' | 'bash',
    code: string,
    options: {
      cwd?: string;
      env?: Record<string, string>;
      timeout?: number;
      metadata?: Record<string, any>;
    } = {}
  ): TriggerE2BTaskRequest => ({
    id,
    type: 'e2b_code',
    payload: {
      id,
      type: 'code',
      language,
      code,
      cwd: options.cwd,
      env: options.env,
      timeout: options.timeout,
      metadata: options.metadata
    }
  }),

  createFileTask: (
    id: string,
    operation: 'write' | 'read' | 'delete' | 'list',
    path: string,
    options: {
      content?: string;
      timeout?: number;
      metadata?: Record<string, any>;
    } = {}
  ): TriggerE2BTaskRequest => ({
    id,
    type: 'e2b_file',
    payload: {
      id,
      type: 'file',
      operation,
      path,
      content: options.content,
      timeout: options.timeout,
      metadata: options.metadata
    }
  })
};

/**
 * E2B Task Manager for Trigger.dev
 * 
 * Provides higher-level operations for managing E2B tasks.
 */
export class E2BTaskManager {
  /**
   * Execute a single E2B task
   */
  static async executeTask(request: TriggerE2BTaskRequest): Promise<TriggerE2BTaskResult> {
    switch (request.type) {
      case 'e2b_shell':
        return await e2bShellTask.trigger(request);
      case 'e2b_code':
        return await e2bCodeTask.trigger(request);
      case 'e2b_file':
        return await e2bFileTask.trigger(request);
      default:
        throw new Error(`Unknown E2B task type: ${request.type}`);
    }
  }

  /**
   * Execute multiple E2B tasks in parallel
   */
  static async executeBatch(requests: TriggerE2BTaskRequest[]): Promise<TriggerE2BTaskResult[]> {
    return await e2bOrchestratorTask.trigger(requests);
  }

  /**
   * Execute multiple E2B tasks sequentially
   */
  static async executeSequential(requests: TriggerE2BTaskRequest[]): Promise<TriggerE2BTaskResult[]> {
    const results: TriggerE2BTaskResult[] = [];
    
    for (const request of requests) {
      const result = await this.executeTask(request);
      results.push(result);
    }
    
    return results;
  }

  /**
   * Check E2B health status
   */
  static async healthCheck(): Promise<{ healthy: boolean; details: any }> {
    return await e2bHealthCheckTask.trigger();
  }

  /**
   * Close all active E2B sandboxes
   */
  static async cleanup(): Promise<void> {
    await e2bClient.closeAllSandboxes();
  }
}

// Export all tasks for easy registration
export const E2BTasks = {
  shell: e2bShellTask,
  code: e2bCodeTask,
  file: e2bFileTask,
  orchestrator: e2bOrchestratorTask,
  healthCheck: e2bHealthCheckTask
};

export default E2BTasks;