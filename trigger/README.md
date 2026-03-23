# Spiderweb Trigger Workspace

This workspace hosts the optional Trigger.dev tasks used by Spiderweb for cheap cognition and model lifecycle helpers.

## Scope
- optionally ensure the local Youtu `vLLM` service is running
- call the local OpenAI-compatible endpoint for cheap classification/summarization
- reuse the same native runtime helpers as Spiderweb itself

## Setup
1. Copy `.env.example` to `.env`.
2. Fill in Trigger.dev credentials and local runtime paths.
3. Add the required Youtu `vLLM` patch files under `../infra/vllm/patches/`.
4. Install dependencies.
5. Run `trigger.dev dev` or deploy from this directory.

## Important
This workspace is optional.
Spiderweb must not require Trigger.dev in the install/start critical path.
If enabled, Trigger should run as a native process managed by Spiderweb lifecycle hooks rather than as an orphaned background service.
