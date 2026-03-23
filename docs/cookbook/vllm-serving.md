# vLLM Serving Patterns

Reusable notes for hosting a small model as Spiderweb cheap cognition infrastructure.

## Core rule
The model server is long-lived infrastructure, not a per-task runtime.

Correct shape:
- one self-hosted Trigger.dev worker host
- one long-lived `vLLM` container
- one persistent HF cache volume
- local OpenAI-compatible endpoint for Spiderweb use

## First-start download pattern

Required behavior:
- first start downloads from Hugging Face
- weights land in a persistent mounted cache
- later restarts reuse the cache

Example run shape:

```bash
docker run -d \
  --gpus all \
  --name youtu-vllm \
  --restart unless-stopped \
  --shm-size=8g \
  -p 8000:8000 \
  -v /var/lib/spiderweb/hf-cache:/models \
  -e HF_TOKEN=$HF_TOKEN \
  spiderweb/youtu-vllm:0.10.2
```

## Why weights must not be baked into the image

- rebuilds become too heavy
- deploys become slow
- cache reuse becomes harder
- network-independent warm restart becomes impossible

## Initial serving defaults

Suggested initial `vLLM` settings:
- model: `tencent/Youtu-LLM-2B`
- host: `0.0.0.0`
- port: `8000`
- `--max-model-len 32768`
- `--gpu-memory-utilization 0.85`

Increase context length only after memory is measured on the real host.
