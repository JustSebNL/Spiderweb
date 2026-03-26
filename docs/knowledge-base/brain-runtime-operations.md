# Brain Runtime Operations

This guide explains the current `brain` cheap-cognition runtime: where it lives, which files matter, and how to diagnose common failures.

## What `brain` Is

`brain` is the local cheap-cognition runtime area used by Spiderweb for smaller local model work before escalating to heavier reasoning paths.

Current runtime policy:
- canonical namespace is `brain`
- old `YOUTU_*` env names may still be honored as compatibility fallbacks
- runtime selection is currently `vllm` or `llama.cpp`

## Main Paths

Common defaults:
- `BRAIN_DIR=$SPIDERWEB_DIR/brain`
- `BRAIN_MODEL_CACHE_DIR=$BRAIN_DIR/model-cache`
- `HF_HOME=$HOME/.spiderweb/hf`
- `HF_HUB_CACHE=$HF_HOME/hub`

Useful operator files:
- `~/.spiderweb/runtime.env`
- `brain/brain-vllm.pid`
- `brain/brain-vllm.log`
- `brain/llama-server.pid`
- `brain/llama-server.log`

## Important Runtime Environment Values

The generated runtime env can contain:
- `BRAIN_DIR`
- `BRAIN_MODEL_CACHE_DIR`
- `BRAIN_VLLM_VENV`
- `BRAIN_VLLM_HOST`
- `BRAIN_VLLM_PORT`
- `BRAIN_VLLM_PID_FILE`
- `BRAIN_VLLM_LOG_FILE`
- `BRAIN_VLLM_MAX_MODEL_LEN`
- `BRAIN_VLLM_GPU_MEMORY_UTILIZATION`
- `BRAIN_LLAMA_CPP_PORT`
- `HF_HOME`
- `HF_HUB_CACHE`

Inspect them with:

```bash
cat ~/.spiderweb/runtime.env
```

## Expected Processes

### `vllm`

Expected when:
- runtime selection resolves to `vllm`
- required patch files exist under `infra/vllm/patches/`
- the Python venv under `BRAIN_VLLM_VENV` is present

Main process:
- `python -m vllm.entrypoints.openai.api_server`

Expected files:
- `brain/brain-vllm.pid`
- `brain/brain-vllm.log`

### `llama.cpp`

Expected when:
- runtime selection resolves to `llama_cpp`
- or `vllm` is in `auto` mode but the host lacks GPU support

Main process:
- `llama-server`

Expected files:
- `brain/llama-server.pid`
- `brain/llama-server.log`

## When `vllm` vs `llama.cpp` Is Expected

`vllm`:
- preferred when the host has GPU support
- Youtu integration files are present under `infra/vllm/patches/`
- requires the vLLM virtualenv and Python entrypoint

`llama.cpp`:
- used as the practical fallback when the host lacks GPU support
- still serves the local OpenAI-compatible endpoint path for cheap cognition

Important current behavior:
- if `CHEAP_COGNITION_RUNTIME=vllm` is explicitly requested and the host lacks GPU, bootstrap fails clearly
- if runtime is `auto` and the host lacks GPU, bootstrap falls back to `llama.cpp`

## Useful Runtime Checks

Check the selected values:

```bash
cat ~/.spiderweb/runtime.env
```

Check whether `vllm` appears alive:

```bash
cat brain/brain-vllm.pid
tail -n 80 brain/brain-vllm.log
```

Check whether `llama.cpp` appears alive:

```bash
cat brain/llama-server.pid
tail -n 80 brain/llama-server.log
```

Check local endpoint reachability:

```bash
curl http://127.0.0.1:8000/v1/models
curl http://127.0.0.1:8081/v1/models
```

Use the port that matches the selected runtime in `runtime.env`.

## Common Startup Failures

### No GPU available for `vllm`

Symptom:
- startup fails early for `vllm` when explicitly configured
- or `auto` falls back to `llama.cpp`

Cause:
- host does not have a suitable NVIDIA-backed GPU
- CUDA drivers are not installed or not detected

### Missing vLLM virtualenv

Symptom:
- startup fails because the Python binary under `BRAIN_VLLM_VENV/bin/python` is missing

Cause:
- runtime environment was not prepared fully
- or the venv path is wrong

### Stale PID file

Symptom:
- runtime looks already running when it is not
- or stop/start behavior is inconsistent

What to inspect:
- PID file contents
- whether the process still exists
- runtime log freshness

### Missing Hugging Face cache or download state

Symptom:
- long startup
- repeated downloads
- missing local model files

What to inspect:
- `HF_HOME`
- `HF_HUB_CACHE`
- `BRAIN_MODEL_CACHE_DIR`

## Operator Rule Of Thumb

Use `brain` status to answer:
- which runtime is selected
- whether the local cheap-cognition model is alive
- where its logs and pid files live
- whether failure is coming from model download, runtime startup, or endpoint availability

## Related Docs
- [Startup And Daily Use](./startup-and-daily-use.md)
- [Troubleshooting](./troubleshooting.md)
- [Observer And Self-Care](./observer-and-self-care.md)
