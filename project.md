# Project Blueprint - Spiderweb

## Overview
Spiderweb is a Go-based CLI project. This workspace contains source, docs, and a Makefile-driven build/install process.

## Document Role
`project.md` is the blueprint document.

It should describe:
- what the system is supposed to do
- why each subsystem exists
- how the parts are expected to interact
- implementation constraints
- deployment shape
- interface contracts
- operational rules
- MVP boundaries

It should not be treated as a scratchpad or a completion tracker.

`docs/cookbook/` is the reusable pattern library.

It should describe:
- implementation recipes
- useful code snippets
- integration examples
- operational patterns worth reusing

`todo-tasks.md` is the execution tracker.

It should describe:
- what has already been completed
- what is currently in progress
- what still needs to be built

## OpenClaw Intake Patch Context
Spiderweb can act as a patch-in intake layer for OpenClaw, focused on reducing token burn and repetitive polling while keeping OpenClaw on high-value reasoning.

- Support colleague that handles watch-duty and intake.
- Feeds OpenClaw only what matters; no repeated collection.
- Integrates with minimal intrusion into OpenClaw.
- Flow: **Services / Pipelines → Spiderweb → Queue / Valve Layer → OpenClaw**
- Core spirit: “Dude….. no worries! I’ve got this 😏”

## Current Objective
Provide a production-usable one-command bootstrap path for Linux hosts that can take a machine from mostly bare to Spiderweb-ready without making the user step through setup decisions.

The intended user experience is:
- run one root command,
- let the system inspect the machine,
- let the bootstrap flow run every required internal chapter,
- leave the host ready to move toward `sweb wakeup`.

## Bootstrap Design
### User-facing rule
There is one install entrypoint:
- `./bootstrap.sh`

That is the only command the user should need to run for full host setup.

### Internal chapter rule
The `scripts/` directory may contain helper scripts, but those are internal chapters of the bootstrap book.

They are not separate install products.
They are not competing entrypoints.
They exist only so `bootstrap.sh` can compose the full setup flow in smaller steps.

If a setup chapter exists under `scripts/`, it should be callable by `bootstrap.sh` and documented as internal implementation detail.

### Goals
1. Be safe for non-root hosting users where possible.
2. Work across common Linux distros and package managers.
3. Fall back to user-local Go install when system Go is missing or outdated.
4. Keep install targets user-local by default while supporting configurable custom paths.
5. Make runtime and model-serving decisions autonomously based on host capability.
6. Prepare the full Spiderweb + cheap-cognition stack, not just the Go binary.
7. Keep Trigger.dev optional rather than making it part of the required install/start path.

### Required bootstrap behaviors
- Run a pre-install intake step that can explain credential requirements before package installation starts.
- Check and install required tools such as `git`, `make`, `tar`, `curl`, `python3`, Node/npm, and other runtime prerequisites when missing.
- Install `git-lfs` on fresh systems and initialize it for the current user so large-model pulls remain supported.
- Detect Linux distro and adapt prerequisite installation commands by package manager.
- Support package managers: `apt-get`, `dnf`, `yum`, `zypper`, `pacman`, `apk`, `brew`.
- Support archive extraction via either `tar` or `gtar` when needed.
- Verify Go version and enforce the configured minimum version.
- Install Go locally when needed and export paths in user profiles where appropriate.
- Install Spiderweb core and place the `sweb` binary on a usable path.
- Prepare `youtu-llm/` directories.
- Prepare `trigger/` only when that optional workspace is explicitly enabled.
- Install Hugging Face download tooling.
- Pull the selected Youtu model into the local `youtu-llm/` area during bootstrap rather than leaving the repo empty.
- Inspect hardware and runtime capability automatically.
- Choose the cheap-cognition runtime automatically.
- Download the matching model format from Hugging Face.
- Start the matching local model service.
- Write generated runtime configuration for Spiderweb consumption.
- Write an auto-loaded runtime env file into Spiderweb home so `sweb wakeup` can pick up the chosen cheap-cognition runtime without manual shell sourcing.
- Write a local setup-notes file that tells the user which remaining provider keys still need to be configured after bootstrap.
- Persist bootstrap session state locally so a restarted install can continue from the last completed chapter instead of starting over.
- Finish in a state that is ready to progress toward `sweb wakeup`.

### Runtime selection behavior
`bootstrap.sh` must decide at install time:
- `vllm` when the host has a suitable NVIDIA-backed GPU for native local serving
- `llama.cpp` when it does not

This decision should happen without asking the user in the normal path.

### Config files
- Root bootstrap config: `bootstrap.conf`
- Core Spiderweb install config: `install_spiderweb.conf`

`bootstrap.conf` controls the end-to-end stack setup.
`install_spiderweb.conf` exists as internal configuration for the core install chapter.
It is not intended to be a second user-facing entrypoint.

### Operational notes
- The bootstrap path is designed for Linux shell execution via `bash ./bootstrap.sh`.
- On Windows hosts, the bootstrap path should hand off into WSL rather than pretending native Windows shell support exists.
- If package installation requires elevated privileges, the script may use `sudo` when available.
- If no supported package manager exists, bootstrap should fail clearly with manual prerequisite guidance rather than continuing in a misleading partial state.
- If bootstrap needs credentials that are missing, it should explain them clearly and prompt only when the run is interactive.
- If the run is non-interactive, bootstrap should emit exact acquisition guidance into a local setup-notes file rather than failing silently.
- If bootstrap is interrupted, the next run should auto-load the previous bootstrap state and skip completed chapters.
- Bootstrap should provide a way to inspect or reset saved bootstrap state so users are never trapped by stale installer state.

### Current status
- Root bootstrap entrypoint exists at repository root: `bootstrap.sh`.
- Root bootstrap config exists at repository root: `bootstrap.conf`.
- Internal setup chapters live under `scripts/`.
- The bootstrap flow is designed as the only user-facing install command.
- The runtime-selection path is designed to choose `vllm` or `llama.cpp` automatically.
- Bootstrap now writes runtime state into Spiderweb home as `~/.spiderweb/runtime.env`, and the config loader reads that file automatically during startup.
- Bootstrap now persists install progress and resolved settings into `~/.spiderweb/bootstrap-state.env` so interrupted installs can resume.
- The current remaining blocker for full Youtu `vLLM` readiness is the missing Youtu-specific patch files under `infra/vllm/patches/`.

## OpenClaw Bridge (Chat Interface + Transfer Sequence)

### Objective
Enable real-time 1-on-1 communication between Spiderweb and OpenClaw via a WebSocket bridge, positioning Spiderweb as the new message intake colleague.

### Architecture
```
Services / Pipelines → Spiderweb (intake) → Queue / Valve → OpenClaw (reasoning)
```

### Components
1. **OpenClaw WebSocket Channel** (`pkg/channels/openclaw.go`)
   - Accepts WebSocket connections from OpenClaw peers
   - Handshake protocol with shared secret auth
   - Auto-sends transfer introduction on connect
   - Bidirectional message routing through the message bus
   - Ping/pong keepalive

2. **Bridge Endpoint** (health server `/bridge/openclaw`)
   - WebSocket upgrade handler registered at configurable path
   - Registered dynamically when OpenClaw channel is enabled

3. **Transfer Sequence CLI** (`spiderweb openclaw transfer`)
   - Drives the handoff: Spiderweb introduces itself as intake colleague
   - Opens a transfer chat for collaboration
   - Sends structured introduction message
   - Validates valve endpoint readiness

4. **Config** (`channels.openclaw`)
   - `enabled`: toggle the bridge
   - `shared_secret`: optional auth token for handshake
   - `auto_handshake`: auto-send transfer intro on connect
   - `intake_enabled`: advertise intake capability
   - `webhook_path`: WebSocket endpoint path (default: `/bridge/openclaw`)

### CLI Commands
- `spiderweb openclaw status` — check bridge connection state
- `spiderweb openclaw connect` — test WebSocket connection as an OpenClaw peer
- `spiderweb openclaw transfer` — run the full transfer sequence

### Design Spirit
"Dude….. no worries! I've got this 😏"

## Youtu-LLM Hosting Blueprint

### Current direction
Trigger.dev is no longer a required part of the install or runtime path.

The core path should be:
- `bootstrap.sh`
- local cheap-cognition runtime
- direct Spiderweb client calls
- `sweb wakeup`

Trigger.dev may still be used later for optional background workflows, but Spiderweb must not depend on it to install, start, or reach a working local cognition stack.


### Purpose
Spiderweb needs a low-cost model for the cheap cognition layer.

This model is not the main reasoning brain. It is the intake-side worker used for:
- short summarization
- priority classification
- entity extraction
- escalation gating
- compact event triage before waking OpenClaw

The selected target model is `tencent/Youtu-LLM-2B`.
The selected serving runtime is `vLLM`.
The selected orchestration layer for the core path is Spiderweb itself.
Trigger.dev is optional and out of the critical path.

### Required deployment shape
The model must be:
1. pulled from Hugging Face,
2. cached on persistent local storage,
3. served by a long-lived `vLLM` process,
4. made available to Spiderweb agents through an OpenAI-compatible local endpoint.

This does **not** mean starting a fresh model server for every task.

The correct design is:
- Spiderweb bootstrap prepares the runtime once.
- the selected cheap-cognition runtime stays up as a long-lived native local service.
- Spiderweb agents call that local service when cheap cognition is needed.
- Trigger.dev, if enabled, reuses the same native helper path instead of introducing a second runtime-management strategy.

### Hard constraints
This must run on a Linux or WSL host that Spiderweb controls directly.

Reasons:
- `vLLM` is only useful here with GPU-backed inference.
- the cheap-cognition runtime must be a local dependency Spiderweb can start and inspect directly.
- the install path must not depend on Docker CLI control.
- Trigger.dev public task machines are not the baseline deployment target for Spiderweb bootstrap.
- the Youtu model card documents `vLLM` support around `0.10.2` with model-specific integration patches.

### High-level architecture

```text
Spiderweb agents
  -> Spiderweb cheap-cognition call
  -> Trigger task or direct internal client
  -> local vLLM OpenAI-compatible endpoint
  -> Youtu-LLM result
  -> Spiderweb routing decision
  -> optional escalation to OpenClaw
```

Operationally, the serving path is:

```text
GPU host or GPU-enabled WSL host
  ├─ Spiderweb bootstrap and runtime config
  ├─ long-lived youtu-vllm process
  └─ persistent HF cache directory
```

### Component responsibilities

#### Trigger.dev
Trigger.dev is optional.

If enabled later, it is responsible for:
- reusing the native runtime helpers Spiderweb already owns
- wrapping background workflows that benefit from Trigger semantics
- ensuring the local model service exists before optional tasks use it

Trigger.dev is not responsible for:
- hosting inference inside the task process itself
- downloading model weights repeatedly
- acting as the permanent runtime for the model process
- becoming a second required install or start path

#### vLLM service
The `vLLM` service is responsible for:
- loading the Youtu model
- keeping weights resident in memory
- exposing an OpenAI-compatible API
- serving cheap cognition requests with low local latency

#### Spiderweb agents
Spiderweb agents are responsible for:
- calling the cheap cognition endpoint only after deterministic filters run first
- sending compact event payloads rather than raw full-context dumps
- escalating to OpenClaw only when the cheap layer or rules decide escalation is needed

### Hugging Face download and cache behavior
The model weights must not be baked into an image layer or any ephemeral bootstrap artifact.

Required behavior:
- On first startup, `vLLM` pulls `tencent/Youtu-LLM-2B` from Hugging Face.
- The model is cached to a mounted persistent directory.
- On later restarts, `vLLM` reuses the local cache instead of downloading again.
- Public model pulls should work even when `HF_TOKEN` is unset; authenticated access should only be required when Hugging Face demands it.

Recommended cache path:
- host path: `~/.spiderweb/hf-cache` or another configured Spiderweb-owned directory

This is required because:
- the model is multiple GB in size,
- cold restarts should not depend on a repeated full download,
- network failure should not destroy a working deployment once the model is cached.

### Required filesystem layout
Recommended project layout for this slice:

```text
infra/
  vllm/
    patches/
      youtu_llm.py
      configuration_youtu.py
      registry.py
      __init__.py
scripts/
  start_youtu_vllm.sh
trigger/
  tasks/
    ensure-youtu-vllm.ts
    classify-event.ts
    summarize-event.ts
```

### Native vLLM runtime
A plain upstream `vLLM` install should not be assumed to work for Youtu-LLM without the Youtu-specific integration files.

The model card indicates that Youtu support is tied to:
- `vLLM 0.10.2`
- model-specific patch files

That means Spiderweb should prepare a dedicated Python runtime for Youtu hosting.

Implementation notes:
- create a local virtualenv under `youtu-llm/.venv-vllm`
- install `vllm==0.10.2`
- apply the Youtu patch files into that runtime once sourced
- start the API server as a native long-lived process
- start with `32768` context instead of the full `128k` until GPU memory behavior is measured
- raise context length only after stability and memory tests

### Native process contract
The model server process must be started with:
- GPU access enabled when `vllm` is selected
- persistent model cache configured
- `HF_TOKEN` available if Hugging Face access requires authentication
- a stable pid file and log file for health checks and restart logic
- a local OpenAI-compatible endpoint bound to the configured host and port

### Optional Trigger helper task
If Trigger.dev is enabled later, its helper task should only ensure the native model service exists and is healthy.

Responsibilities:
- check whether the local `vLLM` endpoint is healthy
- call the internal start helper if it is not
- wait for `/v1/models` or equivalent health response
- fail clearly if the service does not become healthy

Trigger.dev should not introduce a second runtime-management strategy.
It should use the same native start helper as the bootstrap path.

### Cheap cognition task contract
A separate Trigger task should call the local `vLLM` endpoint for classification or summarization.

Requirements:
- use the OpenAI-compatible API exposed by `vLLM`
- send compact structured payloads only
- use deterministic pre-filtering before the call happens
- return compact structured output suitable for Spiderweb routing

Example task shape:

```ts
import OpenAI from "openai";
import { task } from "@trigger.dev/sdk";

export const classifyEvent = task({
  id: "classify-event",
  run: async ({ event }) => {
    const client = new OpenAI({
      apiKey: process.env.YOUTU_VLLM_API_KEY ?? "dummy",
      baseURL: process.env.YOUTU_VLLM_BASE_URL ?? "http://127.0.0.1:8000/v1",
    });

    const response = await client.chat.completions.create({
      model: "tencent/Youtu-LLM-2B",
      temperature: 0.2,
      messages: [
        {
          role: "system",
          content: "You are Spiderweb's cheap intake classifier. Return compact structured triage output.",
        },
        {
          role: "user",
          content: JSON.stringify(event),
        },
      ],
    });

    return response.choices[0]?.message?.content ?? "";
  },
});
```

### Network and API contract
The local service endpoint should be treated as an internal dependency.

Recommended endpoint:
- `http://127.0.0.1:8000/v1`

Required properties:
- not exposed publicly unless explicitly required
- health-checkable locally
- stable model identifier
- low-latency path between Trigger worker and `vLLM`

### Environment variables
Required runtime variables on the host:
- `HF_TOKEN`
- `YOUTU_VLLM_BASE_URL`
- `YOUTU_VLLM_API_KEY`
- `YOUTU_VLLM_VENV`
- `YOUTU_VLLM_PORT`

### Deployment sequence
1. Prepare a GPU host or GPU-enabled WSL environment.
2. Run `bootstrap.sh` to install Spiderweb and prepare the cheap-cognition runtime.
3. Create persistent model cache storage under the local `youtu-llm/` area.
4. Install the Youtu-compatible `vLLM 0.10.2` runtime.
5. Add the Youtu patch files once sourced.
6. Start and warm the native model service.
7. Point Spiderweb cheap cognition calls to the local `vLLM` endpoint.
8. Validate warm-start behavior so the second start uses cache instead of a fresh Hugging Face download.

### Operational rules
- Keep one long-lived `vLLM` server per worker host in v1.
- Do not cold-start the model for each job.
- Do not send raw noisy events directly to the model.
- Always journal the original event before cheap cognition is applied.
- Journal the cheap-cognition result separately from the raw event.
- Escalate to OpenClaw only after rules or Youtu classification indicate meaningful action is needed.
- Treat the model service as restartable infrastructure, not as irreplaceable state.

### Failure handling
If the model service is unavailable:
- Spiderweb must continue journaling events
- deterministic routing rules must keep working
- cheap cognition may be skipped or deferred
- OpenClaw escalation should still remain possible for high-priority events

This prevents the small-model layer from becoming a single point of operational failure.

## Runtime Maintenance Blueprint

### Purpose
Spiderweb should service its own local runtime health without creating noisy background traffic or user-visible lag.

This is not meant to be an always-on benchmark engine.
It is meant to be a low-priority janitor that establishes a startup baseline, checks for drift on a slow cadence, and performs bounded cleanup or restart actions when the local runtime is measurably worse.

### Required behavior
- establish a startup baseline shortly after gateway launch
- run a self-check on a 12-hour cadence by default
- stay within a low-impact maintenance lane capped to roughly 5% of available maintenance budget
- prefer local file, pid, and process inspection over HTTP chatter
- compare later checks against the startup baseline
- remediate only Spiderweb-owned files, logs, pid files, and processes
- avoid restart storms by using bounded remediation and backoff-friendly rules

### Startup baseline
The first maintenance pass after startup should record:
- cheap-cognition runtime availability
- cheap-cognition latency from a small local probe
- cheap-cognition classification counters
- optional Trigger worker process state
- initial runtime score and summary

This baseline becomes the local point of comparison for later checks.

### 12-hour self-check behavior
Periodic checks should be lightweight and mostly local.

Preferred signals:
- owned process is still alive
- pid file is valid or stale
- log growth is within bound
- cheap-cognition failure count is drifting upward
- cheap-cognition latency is materially worse than baseline
- cheap-cognition skip behavior is becoming excessive

### Allowed remediation
The maintenance lane may:
- remove stale pid files for dead owned processes
- trim oversized Spiderweb-owned log files
- request restart of an owned dead process when auto-remediation is enabled
- write updated health snapshots and recommendations

It should not:
- spam health endpoints
- run large synthetic benchmark prompts
- clear useful caches blindly
- compete with foreground traffic

### Ownership rules
Maintenance actions must only touch Spiderweb-owned runtime artifacts.

Examples:
- `~/.spiderweb/runtime-health.json`
- Spiderweb-owned pid files
- Spiderweb-owned runtime logs
- Spiderweb-owned local model runtime state

### Current implementation direction
The repo now contains a native maintenance service intended to run inside gateway startup and shutdown flow.

Current responsibilities:
- write a startup baseline
- write recurring health snapshots
- observe cheap cognition and optional Trigger state
- perform bounded cleanup and restart actions

Current truth:
- the maintenance service is present in code but still needs hardening before it should be considered complete
- shell/runtime verification was not possible in the current session
- the missing Youtu `vLLM` patch files remain a blocker for full native `vLLM` readiness

### MVP boundary
For v1, this slice should remain narrow:
- one model: `tencent/Youtu-LLM-2B`
- one runtime: `vLLM 0.10.2`
- one self-hosted Trigger.dev worker host with GPU
- one bootstrap task to ensure the model service is running
- one or two cheap cognition tasks for compact classification and summarization
- one persistent Hugging Face cache volume

Out of scope for v1:
- autoscaling
- multiple cheap models
- public multi-tenant inference service
- advanced tool-calling through the small model
- dynamic model routing
- distributed multi-host `vLLM` serving

### Design summary
The model must be downloaded from Hugging Face once, cached locally, served continuously by the selected native runtime, and then used by Spiderweb agents as a local cheap-cognition dependency. One concrete integration now exists in the OpenClaw forward path: Spiderweb classifies forwarded intake, annotates the forwarded payload with triage metadata, and can skip low-priority non-escalations before they reach OpenClaw.

## Bare-System Bootstrap Requirement

Spiderweb requires an end-to-end bootstrap path for new Linux hosts.

The project must provide a shell-based installation flow that can take a mostly bare host and prepare it for Spiderweb operation with minimal manual intervention.

### The bootstrap script must be responsible for
- installing required system packages and tools
- installing Spiderweb core and the `sweb` binary
- preparing the Trigger.dev workspace
- preparing the `youtu-llm/` local model area
- inspecting the host hardware/runtime automatically
- choosing the cheap-cognition runtime without asking the user
- pulling the matching Youtu model format from Hugging Face
- starting the matching local model service
- writing the chosen runtime and endpoint into generated runtime config
- leaving the machine ready for `sweb wakeup`

### Runtime selection rule
The system should choose autonomously:
- `vllm` when a suitable GPU/NVIDIA-backed host is available
- `llama.cpp` when it is not

The user should not have to manually decide between the two in the normal path.

### Current implementation direction
The dedicated stack bootstrap entrypoint is:
- `bootstrap.sh`

Its config file is:
- `bootstrap.conf`

### Current truth
The stack bootstrap flow is scaffolded, but full model-serving readiness still depends on the missing Youtu-specific `vLLM` patch files under `infra/vllm/patches/`.

That means the bootstrap flow currently establishes the correct installation path and system shape, but it is not yet fully build-complete for Youtu serving until those patch files are added.
