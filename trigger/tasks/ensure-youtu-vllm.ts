import { execFile } from "node:child_process";
import path from "node:path";
import { promisify } from "node:util";

import { task } from "@trigger.dev/sdk";

const execFileAsync = promisify(execFile);

function requiredEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

async function ensureHealthy(baseUrl: string) {
  const response = await fetch(`${baseUrl}/models`);
  if (!response.ok) {
    throw new Error(`vLLM health check failed: ${response.status} ${response.statusText}`);
  }
}

export const ensureYoutuVllm = task({
  id: "ensure-youtu-vllm",
  retry: {
    maxAttempts: 3,
  },
  run: async () => {
    const repoRoot = path.resolve(process.cwd(), "..");
    const startScript = path.join(repoRoot, "scripts", "start_youtu_vllm.sh");
    const youtuDir = process.env.BRAIN_DIR ?? process.env.YOUTU_DIR ?? path.join(repoRoot, "brain");
    const cacheDir = process.env.YOUTU_CACHE_DIR ?? path.join(youtuDir, "model-cache");
    const venvDir = process.env.YOUTU_VLLM_VENV ?? path.join(youtuDir, ".venv-vllm");
    const port = process.env.YOUTU_VLLM_PORT ?? "8000";
    const host = process.env.YOUTU_VLLM_HOST ?? "127.0.0.1";
    const baseUrl = `http://${host}:${port}/v1`;

    await ensureHealthy(baseUrl).catch(async () => {
      const env = {
        ...process.env,
        HF_TOKEN: requiredEnv("HF_TOKEN"),
        BRAIN_DIR: youtuDir,
        YOUTU_DIR: youtuDir,
        YOUTU_CACHE_DIR: cacheDir,
        YOUTU_VLLM_VENV: venvDir,
        YOUTU_VLLM_PORT: port,
        YOUTU_VLLM_HOST: host,
        YOUTU_VLLM_MODEL_PATH: process.env.YOUTU_VLLM_MODEL_PATH ?? youtuDir,
      };

      await execFileAsync("bash", [startScript], { env, cwd: repoRoot });
      await ensureHealthy(baseUrl);
    });

    return {
      ok: true,
      baseUrl,
      runtime: "native-vllm",
      script: startScript,
    };
  },
});
