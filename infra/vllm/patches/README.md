# Youtu vLLM Patches

This directory is intentionally incomplete.

The official `tencent/Youtu-LLM-2B` model card states that `vLLM` support is tied to `vllm/vllm-openai:v0.10.2` plus model-specific integration files.

Expected files:
- `youtu_llm.py`
- `configuration_youtu.py`
- `registry.py`
- `__init__.py`

These files should be copied from the official Youtu `vLLM` integration package or equivalent upstream source before building `Dockerfile.youtu-vllm`.

Until those files are added, the Docker image is scaffolded but not build-complete.
