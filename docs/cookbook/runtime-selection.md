# Runtime Selection Pattern

Spiderweb should choose its cheap-cognition runtime autonomously.

## Goal
The user should run one command on a mostly bare host. The bootstrap process should inspect the machine and decide whether the local cheap-cognition layer should use:
- `vLLM` with standard Hugging Face weights, or
- `llama.cpp` with GGUF weights.

## Selection rules

### Prefer `vLLM` when
- a suitable NVIDIA GPU is present
- `nvidia-smi` works
- the host can support a native local `vLLM` process

### Fall back to `llama.cpp` when
- no supported GPU is detected
- native `vLLM` prerequisites are unavailable
- the machine is better suited to a CPU-first or lightweight local path

## Why this matters
This removes manual deployment choice from the user and keeps the project aligned with its operational goal:
- one command
- self-inspection
- automatic fit-to-host runtime selection

## Expected output of bootstrap
The bootstrap process should write the chosen values into an auto-loaded runtime env file for Spiderweb and into a generated env file for debug or downstream consumers.

Primary runtime file:
- `~/.spiderweb/runtime.env`

Additional generated reference file:
- `${SPIDERWEB_DIR}/.generated/spiderweb-runtime.env`

Expected values include:
- runtime
- model identifier
- base URL
- chosen port
