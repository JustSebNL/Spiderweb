# Model Hosting Notes

This file saves the external references and implementation notes that shaped the Spiderweb cheap-cognition hosting blueprint.

It is not a copy of vendor documentation. It is a local reference index with project-relevant takeaways.

## Why this file exists

The project now depends on a few external facts:
- how Trigger.dev tasks are structured
- what Trigger.dev machine sizing looks like
- how `vLLM` is expected to run natively on a host-controlled Python runtime
- what the Youtu model card says about `vLLM` compatibility

These facts should be saved locally so future work does not depend on memory.

## Saved references

### Trigger.dev docs

1. Tasks overview
   - URL: `https://trigger.dev/docs/tasks/overview`
   - Project takeaway:
     - tasks are defined with `task({ id, run, ... })`
     - tasks can have retry settings
     - tasks can specify machine requirements
     - tasks are the correct wrapper for cheap cognition jobs and lifecycle helpers

2. trigger.config.ts
   - URL: `https://trigger.dev/docs/trigger.config.ts`
   - Project takeaway:
     - Trigger.dev projects are configured through `defineConfig(...)`
     - task directories should be declared explicitly
     - this is why the repo now has a dedicated `trigger/trigger.config.ts`

3. Machines
   - URL: `https://trigger.dev/docs/machines`
   - Project takeaway:
     - public Trigger.dev task machines are documented with relatively small CPU/RAM/disk presets
     - this supports the design decision that Youtu + `vLLM` must run on a self-hosted GPU worker host rather than standard hosted task machines

4. CLI deploy
   - URL: `https://trigger.dev/docs/cli-deploy`
   - Project takeaway:
     - the Trigger workspace should deploy independently from the Go CLI workspace
     - deployment is a normal Trigger.dev project deployment, not a custom side channel

### vLLM docs

1. Serving and installation
   - URL: `https://docs.vllm.ai/`
   - Project takeaway:
     - `vLLM` provides an OpenAI-compatible API server
     - GPU runtime is expected for the main `vLLM` path
     - local Python runtime preparation matters more than container packaging for Spiderweb
     - Hugging Face cache reuse remains the standard pattern

## Youtu references

1. Youtu-LLM model card
   - URL: `https://huggingface.co/tencent/Youtu-LLM-2B`
   - Project takeaway:
     - model supports long context and is positioned as a cheap agentic worker model
     - the model card includes deployment notes for Transformers, `vLLM`, and `llama.cpp`

2. Youtu configuration file example
   - URL: `https://huggingface.co/tencent/Youtu-LLM-2B/blob/main/configuration_youtu.py`
   - Project takeaway:
     - confirms the model-specific config type and naming expected by the stack
     - supports the assumption that Youtu integration is not plain upstream generic Llama wiring

## Important design decisions derived from these references

### 1. Trigger.dev is the orchestrator, not the model host runtime

The project should not start a fresh model server per task run.

Instead:
- Trigger.dev manages lifecycle
- `vLLM` stays up as a long-lived local service
- Spiderweb uses that service through a local OpenAI-compatible endpoint

### 2. Hugging Face download must happen once and then be cached

The model should:
- download on first boot
- store weights on a persistent volume
- reuse those weights on restart

This is why the current blueprint uses a persistent local cache path such as:
- host/runtime path: `brain/model-cache`

### 3. The Youtu integration status

The current repo includes:
- Trigger workspace
- Trigger tasks
- native vLLM start helper script
- Youtu-specific `vLLM` patch files under `infra/vllm/patches/`

The patch files (`configuration_youtu.py`, `youtu_llm.py`, `registry.py`, `__init__.py`) are now present and compatible with `vLLM 0.10.2`. The native `vLLM` runtime is build-complete for the Youtu-LLM-2B model path.

## Local files this note relates to

- `dev/project.md` if you are keeping local design notes
- `dev/todo-tasks.md` if you are keeping a local task tracker
- `trigger/`
- `infra/vllm/`

## Next recommended step

The next highest-value task is operational validation: download the Youtu model on a real GPU host, verify warm-start caching, and benchmark latency to confirm the runtime meets the cheap-cognition throughput target.
