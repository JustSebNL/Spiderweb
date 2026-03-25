# Youtu Model Configuration for vLLM
#
# This module defines the configuration class for the Tencent Youtu-LLM-2B model.
# It extends the standard vLLM configuration with Youtu-specific parameters.
#
# Source: Official Youtu vLLM integration package
# Compatible with: vllm/vllm-openai:v0.10.2

from typing import Optional, Dict, Any
from transformers import PretrainedConfig


class YoutuConfig(PretrainedConfig):
    """
    Configuration class for Youtu-LLM-2B model.
    
    This configuration extends the base transformer config with
    Youtu-specific architectural parameters.
    
    Attributes:
        vocab_size: Vocabulary size (default: 32000 for Youtu-LLM-2B)
        hidden_size: Hidden dimension size (default: 2048)
        intermediate_size: FFN intermediate size (default: 5504)
        num_hidden_layers: Number of transformer layers (default: 22)
        num_attention_heads: Number of attention heads (default: 16)
        num_key_value_heads: Number of key-value heads for GQA (default: 16)
        hidden_act: Activation function (default: "silu")
        max_position_embeddings: Maximum sequence length (default: 4096)
        initializer_range: Weight initialization std (default: 0.02)
        rms_norm_eps: RMSNorm epsilon (default: 1e-6)
        use_cache: Whether to use KV cache (default: True)
        tie_word_embeddings: Tie input/output embeddings (default: False)
        rope_theta: RoPE base frequency (default: 10000.0)
        rope_scaling: RoPE scaling configuration (optional)
        attention_dropout: Attention dropout rate (default: 0.0)
    """
    
    model_type = "youtu"
    
    def __init__(
        self,
        vocab_size: int = 32000,
        hidden_size: int = 2048,
        intermediate_size: int = 5504,
        num_hidden_layers: int = 22,
        num_attention_heads: int = 16,
        num_key_value_heads: int = 16,
        hidden_act: str = "silu",
        max_position_embeddings: int = 4096,
        initializer_range: float = 0.02,
        rms_norm_eps: float = 1e-6,
        use_cache: bool = True,
        tie_word_embeddings: bool = False,
        rope_theta: float = 10000.0,
        rope_scaling: Optional[Dict[str, Any]] = None,
        attention_dropout: float = 0.0,
        **kwargs,
    ):
        self.vocab_size = vocab_size
        self.hidden_size = hidden_size
        self.intermediate_size = intermediate_size
        self.num_hidden_layers = num_hidden_layers
        self.num_attention_heads = num_attention_heads
        self.num_key_value_heads = num_key_value_heads
        self.hidden_act = hidden_act
        self.max_position_embeddings = max_position_embeddings
        self.initializer_range = initializer_range
        self.rms_norm_eps = rms_norm_eps
        self.use_cache = use_cache
        self.tie_word_embeddings = tie_word_embeddings
        self.rope_theta = rope_theta
        self.rope_scaling = rope_scaling
        self.attention_dropout = attention_dropout
        
        super().__init__(
            tie_word_embeddings=tie_word_embeddings,
            **kwargs,
        )


# Auto-registration for transformers
try:
    from transformers import CONFIG_MAPPING
    CONFIG_MAPPING.register("youtu", YoutuConfig)
except ImportError:
    pass