# Youtu Model Registry for vLLM
#
# This module registers the Youtu-LLM-2B model with vLLM's model registry.
# It enables vLLM to recognize and load the Youtu model architecture.
#
# Source: Official Youtu vLLM integration package
# Compatible with: vllm/vllm-openai:v0.10.2

from typing import Type
from vllm.model_executor.models import ModelRegistry

from .youtu_llm import YoutuLLMForCausalLM
from .configuration_youtu import YoutuConfig


def register_youtu_model():
    """
    Register the Youtu-LLM-2B model with vLLM's model registry.
    
    This function should be called during vLLM initialization to enable
    loading of Youtu-LLM-2B models through vLLM's standard interface.
    
    Example usage:
        from infra.vllm.patches.registry import register_youtu_model
        register_youtu_model()
        
        # Then vLLM can load Youtu models:
        # from vllm import LLM
        # llm = LLM(model="tencent/Youtu-LLM-2B")
    """
    try:
        # Register with vLLM's model registry
        ModelRegistry.register_model(
            model_type="youtu",
            model_cls=YoutuLLMForCausalLM,
            config_cls=YoutuConfig,
        )
        return True
    except Exception as e:
        print(f"Warning: Failed to register Youtu model with vLLM: {e}")
        return False


# Auto-register when imported
_registered = False

def ensure_registered():
    """Ensure the Youtu model is registered with vLLM."""
    global _registered
    if not _registered:
        _registered = register_youtu_model()
    return _registered


# Attempt registration on import
try:
    ensure_registered()
except Exception:
    # Silently fail if vLLM is not available
    # Registration will be attempted again when needed
    pass