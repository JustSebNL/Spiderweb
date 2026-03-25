#   /$$$$$$            /$$       /$$                                             /$$      
#  /$$__  $$          |__/      | $$                                            | $$      
# | $$  \__/  /$$$$$$  /$$  /$$$$$$$  /$$$$$$   /$$$$$$  /$$  /$$  /$$  /$$$$$$ | $$$$$$$ 
# |  $$$$$$  /$$__  $$| $$ /$$__  $$ /$$__  $$ /$$__  $$| $$ | $$ | $$ /$$__  $$| $$__  $$
#  \____  $$| $$  \ $$| $$| $$  | $$| $$$$$$$$| $$  \__/| $$ | $$ | $$| $$$$$$$$| $$  \ $$
#  /$$  \ $$| $$  | $$| $$| $$  | $$| $$_____/| $$      | $$ | $$ | $$| $$_____/| $$  | $$
# |  $$$$$$/| $$$$$$$/| $$|  $$$$$$$|  $$$$$$$| $$      |  $$$$$/$$$$/|  $$$$$$$| $$$$$$$/
#  \______/ | $$____/ |__/ \_______/ \_______/|__/       \_____/\___/  \_______/|_______/ 
#           | $$
#           | $$
#           |__/





# Youtu vLLM Patches

This directory contains the vLLM integration files for the Tencent Youtu-LLM-2B model.

## Files

- `__init__.py` - Package initialization and exports
- `configuration_youtu.py` - YoutuConfig class defining model architecture parameters
- `youtu_llm.py` - YoutuLLMForCausalLM model implementation
- `registry.py` - vLLM model registry integration

## Requirements

These files are compatible with:
- vLLM version: `vllm/vllm-openai:v0.10.2`
- Python dependencies: `torch`, `transformers`

## Usage

### Automatic Registration

The model is automatically registered with vLLM when the package is imported:

```python
from infra.vllm.patches import YoutuLLMForCausalLM
```

### Manual Registration

For explicit control over registration timing:

```python
from infra.vllm.patches.registry import register_youtu_model
register_youtu_model()
```

### Loading with vLLM

Once registered, load Youtu models through vLLM's standard interface:

```python
from vllm import LLM

llm = LLM(model="tencent/Youtu-LLM-2B")
```

## Model Configuration

The Youtu-LLM-2B model has the following default configuration:

| Parameter | Value |
|-----------|-------|
| vocab_size | 32000 |
| hidden_size | 2048 |
| intermediate_size | 5504 |
| num_hidden_layers | 22 |
| num_attention_heads | 16 |
| num_key_value_heads | 16 |
| max_position_embeddings | 4096 |
| rope_theta | 10000.0 |

## Architecture

The model uses:
- **Grouped Query Attention (GQA)** for efficient attention computation
- **SwiGLU** activation function in the MLP layers
- **RMSNorm** for layer normalization
- **Rotary Position Embeddings (RoPE)** for position encoding

## Notes

These files implement the standard Youtu-LLM-2B architecture as documented in the official model card. For production deployments, ensure you have obtained the model weights from the official Tencent/HuggingFace repository.