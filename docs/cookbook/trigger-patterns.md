# Trigger Patterns

Reusable Trigger.dev patterns for Spiderweb.

## 1. Bootstrap a local sidecar service

Use this when a Trigger task must ensure a local dependency is running before work starts.

Example use:
- start `youtu-vllm`
- verify health
- return local base URL

```ts
import { execFile } from "node:child_process";
import path from "node:path";
import { promisify } from "node:util";
import { task } from "@trigger.dev/sdk";

const execFileAsync = promisify(execFile);

export const ensureService = task({
  id: "ensure-service",
  run: async () => {
    const repoRoot = path.resolve(process.cwd(), "..");
    const startScript = path.join(repoRoot, "scripts", "start_youtu_vllm.sh");
    const brainDir = process.env.BRAIN_DIR ?? path.join(repoRoot, "brain");

    await execFileAsync("bash", [startScript], {
      cwd: repoRoot,
      env: {
        ...process.env,
        BRAIN_DIR: brainDir,
        YOUTU_DIR: process.env.YOUTU_DIR ?? brainDir,
      },
    });

    const response = await fetch("http://127.0.0.1:8000/health");
    if (!response.ok) {
      throw new Error("Service health check failed");
    }

    return { ok: true };
  },
});
```

## 2. Call a local OpenAI-compatible endpoint

Use this when the model is hosted by `vLLM` locally.

```ts
async function chatCompletion(messages: Array<{ role: string; content: string }>) {
  const response = await fetch(`${process.env.YOUTU_VLLM_BASE_URL}/chat/completions`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${process.env.YOUTU_VLLM_API_KEY ?? "dummy"}`,
    },
    body: JSON.stringify({
      model: "tencent/Youtu-LLM-2B",
      temperature: 0.2,
      messages,
    }),
  });

  if (!response.ok) {
    throw new Error(`Model request failed: ${response.status}`);
  }

  return response.json();
}
```

## 3. Keep deterministic filtering before model calls

Rule:
- do not send raw noise directly to the cheap model
- dedupe first
- collapse bursts first
- strip irrelevant payload fields first

This is a cost and latency rule, not just a code-style preference.
