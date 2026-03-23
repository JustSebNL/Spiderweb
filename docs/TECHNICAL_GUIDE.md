# Technical Guide

This guide explains how Spiderweb is intended to work beyond the quick-start path.

## 1. System Shape

Spiderweb is a Go CLI that can operate as:
- a lightweight local assistant
- a gateway runtime
- an intake layer in front of OpenClaw
- a low-cost event triage system using a smaller local model

Current user contract:
- install: `./bootstrap.sh`
- start: `sweb wakeup`

## 2. Runtime Layers

### Bootstrap layer
The bootstrap layer prepares the machine from front to back:
- package install
- Go/tooling setup
- Spiderweb install
- cheap-cognition runtime selection
- model download
- runtime env generation
- resumable install state

Important files:
- `bootstrap.sh`
- `bootstrap.conf`
- `install_spiderweb.conf`
- `~/.spiderweb/bootstrap-state.env`
- `~/.spiderweb/runtime.env`
- `~/.spiderweb/setup-notes.txt`

### Core runtime
Core runtime is launched through the CLI and workspace config.

Important entrypoints:
- `cmd/spiderweb/main.go`
- `cmd/spiderweb/internal/gateway/helpers.go`
- `cmd/spiderweb/internal/spiderweb/wake.go`

### Cheap cognition layer
Spiderweb can call a smaller local model before escalating to heavier reasoning.

Current model direction:
- `tencent/Youtu-LLM-2B`

Runtime selection:
- `vllm` when the host can support it
- `llama.cpp` fallback otherwise

Important files:
- `pkg/cognition/client.go`
- `pkg/agent/loop.go`
- `scripts/start_youtu_vllm.sh`
- `scripts/stop_youtu_vllm.sh`

## 3. Gateway And Endpoints

When the gateway is running, Spiderweb exposes local service endpoints.

Health and readiness:
- `/health`
- `/ready`

Valve/intake state:
- `/valve/state`
- `/valve/offer`

OpenClaw bridge when enabled:
- `/bridge/openclaw`

Operational intent:
- Spiderweb filters and annotates intake
- cheap cognition can classify low-value vs escalation-worthy input
- OpenClaw receives higher-value work instead of raw constant noise

## 4. OpenClaw Intake Patch Model

Design direction:

```text
Services / Pipelines -> Spiderweb -> Queue / Valve -> OpenClaw
```

Spiderweb should:
- watch and normalize input
- reduce repetitive polling burden
- classify/summarize before escalation
- annotate forwarded payloads
- skip low-value non-escalations when appropriate

Current integration point already present:
- `pkg/agent/loop.go` classifies forwarded intake before sending to OpenClaw

## 5. Maintenance And Self-Service

Spiderweb includes a native maintenance service intended to stay low-noise.

Current design goals:
- startup baseline
- 12-hour self-check cadence
- 5% maintenance budget
- local process/file inspection first
- bounded remediation only
- no constant health chatter

Current maintenance behaviors:
- write health snapshot to `~/.spiderweb/runtime-health.json`
- write startup baseline snapshot
- detect stale pid files
- trim oversized Spiderweb-owned logs
- request restart of dead owned processes
- defer noncritical actions when the runtime was recently active
- apply restart backoff so remediation does not flap

Important files:
- `pkg/maintenance/service.go`
- `docs/cookbook/runtime-maintenance.md`

## 6. Pipeline Checks

For this project, “pipeline checks” should be understood as runtime flow checks rather than CI-only checks.

### Bootstrap pipeline checks
- did bootstrap complete
- what runtime was selected
- where was state written
- is there a saved bootstrap state

Useful files:
- `~/.spiderweb/bootstrap-state.env`
- `~/.spiderweb/setup-notes.txt`
- `~/.spiderweb/runtime.env`

### Gateway pipeline checks
- health endpoint responds
- ready endpoint responds
- valve state endpoint responds
- OpenClaw bridge is registered when enabled

### Cheap cognition pipeline checks
- model runtime pid file exists and belongs to a live process
- runtime logs are not growing out of control
- health snapshot shows acceptable score
- baseline vs current latency has not drifted badly

### Transfer pipeline checks
- transfer documents exist in workspace
- chat/log state exists
- service kickoff artifacts are present

## 7. Command Surface

Top-level command groups:
- `onboard`
- `agent`
- `auth`
- `gateway`
- `status`
- `cron`
- `migrate`
- `skills`
- `transfer`
- `openclaw`
- `version`
- `wakeup`

Use [COMMANDS.md](D:/Dev/codebase/Dev/Spiderweb/docs/COMMANDS.md) for the operator-facing view.

## 8. State And Persistence

Spiderweb deliberately persists state to avoid repeated setup and repeated work.

Important persistent artifacts:
- bootstrap state
- runtime env
- setup notes
- maintenance health snapshot
- maintenance baseline snapshot
- transfer documents
- launch sequence state

This is aligned with the project direction:
- do not make the user start from zero
- do not repeat finished setup chapters
- keep state inspectable and resettable

## 9. Optional Trigger.dev

Trigger.dev is optional.

If used, it should:
- reuse native Spiderweb runtime helpers
- stay out of the required install/start path
- not replace the native model/runtime architecture

It must not become a second required orchestration plane for core local use.

## 10. Current Known Gaps

These are important current truths:
- Youtu-specific `vLLM` patch files are still missing under `infra/vllm/patches/`
- native `vLLM` readiness is therefore not fully complete
- shell/runtime verification from the current Windows-side session is limited
- some docs and operational flows still need final validation from WSL

## 11. Recommended Reading Order

For operators:
1. [QUICKSTART.md](D:/Dev/codebase/Dev/Spiderweb/docs/QUICKSTART.md)
2. [COMMANDS.md](D:/Dev/codebase/Dev/Spiderweb/docs/COMMANDS.md)

For builders:
1. [project.md](D:/Dev/codebase/Dev/Spiderweb/project.md)
2. [todo-tasks.md](D:/Dev/codebase/Dev/Spiderweb/todo-tasks.md)
3. [runtime-maintenance.md](D:/Dev/codebase/Dev/Spiderweb/docs/cookbook/runtime-maintenance.md)
4. [bootstrap-installer.md](D:/Dev/codebase/Dev/Spiderweb/docs/cookbook/bootstrap-installer.md)
