# Youtu vLLM Integration Package
# 
# This package provides vLLM integration for the Tencent Youtu-LLM-2B model.
# These files are required for the native Spiderweb vLLM runtime path.
#
# Source: Official Youtu vLLM integration package
# Compatible with: vllm/vllm-openai:v0.10.2

from .youtu_llm import YoutuLLMForCausalLM
from .configuration_youtu import YoutuConfig
from .registry import register_youtu_model

__all__ = [
    "YoutuLLMForCausalLM",
    "YoutuConfig", 
    "register_youtu_model",
]