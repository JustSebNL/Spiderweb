# vLLM Serving Patterns

Reusable notes for hosting a small model as Spiderweb cheap cognition infrastructure.

## Core rule
The model server is long-lived infrastructure, not a per-task runtime.

Correct shape:
- one self-hosted Trigger.dev worker host
- one long-lived native `vLLM` process
- one persistent HF cache volume
- local OpenAI-compatible endpoint for Spiderweb use

## First-start download pattern

Required behavior:
- first start downloads from Hugging Face
- weights land in a persistent mounted cache
- later restarts reuse the cache

Example run shape:

```bash
python3 -m venv /opt/spiderweb/brain/.venv-vllm
/opt/spiderweb/brain/.venv-vllm/bin/python -m pip install --upgrade pip setuptools wheel
/opt/spiderweb/brain/.venv-vllm/bin/python -m pip install "vllm==0.10.2" huggingface_hub

python3 -m huggingface_hub download tencent/Youtu-LLM-2B \
  --local-dir /opt/spiderweb/brain

nohup /opt/spiderweb/brain/.venv-vllm/bin/python -m vllm.entrypoints.openai.api_server \
  --model /opt/spiderweb/brain \
  --trust-remote-code \
  --host 127.0.0.1 \
  --port 8000 \
  --max-model-len 32768 \
  --gpu-memory-utilization 0.85 \
  > /opt/spiderweb/brain/youtu-vllm.log 2>&1 &
```

## Why weights must not be baked into the runtime bundle

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
