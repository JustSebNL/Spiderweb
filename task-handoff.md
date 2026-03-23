# Task Handoff

Use this file to resume Spiderweb in a fresh IDE-backed session.

## Resume Prompt
Continue Spiderweb from this local handoff file. Start by reading `task-handoff.md`, then open `pkg/maintenance/service.go`, `cmd/spiderweb/internal/gateway/helpers.go`, `pkg/agent/loop.go`, `project.md`, and `todo-tasks.md`. Finish the native maintenance service, keep Docker out of the required path, keep Trigger.dev optional, and keep the one-command install / one-command start contract intact.

## User Contract
- install: `./bootstrap.sh`
- start: `sweb wakeup`

Everything else stays behind that boundary.

## Current Architecture Truth
- `bootstrap.sh` is the only user-facing install command.
- `scripts/` contains internal setup chapters only.
- Docker is out of the critical path.
- Cheap cognition must be native local runtime selection:
  - `vllm` on suitable NVIDIA-backed Linux/WSL hosts
  - `llama.cpp` fallback otherwise
- Trigger.dev is optional and must not become a required install or start dependency.
- WSL/Linux is the execution target.

## Already Implemented
- Bootstrap/runtime design centered on native local model serving.
- Auto-loaded runtime env from `~/.spiderweb/runtime.env`.
- Native Trigger worker lifecycle control.
- Native Youtu runtime lifecycle control.
- Cheap cognition Go client in `pkg/cognition`.
- Cheap cognition wired into the OpenClaw forward path in `pkg/agent/loop.go`.
- Gateway startup/shutdown wiring for optional Trigger, Youtu runtime, and maintenance service.

## Main Unfinished Slice
`pkg/maintenance/service.go`

This file is partially implemented and should be treated as the immediate next build target.

### Missing helpers referenced by the file
- `removeStalePID(...)`
- `trimLogIfNeeded(...)`
- `hasBudget(...)`

Likely also needed:
- `consumeBudget(...)`

## Maintenance Feature Requirements
- create a startup baseline shortly after launch
- compare later checks against that baseline
- run every 12 hours by default
- stay within a low-impact 5% maintenance budget
- prefer local pid/file/process inspection over HTTP chatter
- only tiny active probe during baseline creation
- do bounded remediation only:
  - remove stale pid files
  - trim oversized Spiderweb-owned logs
  - request restart of owned dead processes
- do not create lag spikes or restart storms

## Files To Open First
- `pkg/maintenance/service.go`
- `cmd/spiderweb/internal/gateway/helpers.go`
- `pkg/agent/loop.go`
- `pkg/servicectl/trigger.go`
- `pkg/servicectl/youtu.go`
- `project.md`
- `todo-tasks.md`

## Known Constraint
The prior session could not run shell commands because of a Windows sandbox refresh failure, so recent work was not runtime-tested.

## Immediate Next Step
Finish and harden the maintenance service, then keep `project.md` and `todo-tasks.md` aligned with the actual repo state.

## Useful Sidecar Test
If a second IDE action can run in parallel, use:

```bash
go test -v ./pkg/maintenance -run TestMaintenance 2>&1 | tee maintenance-test.log
```

Related note:
- `docs/cookbook/maintenance-test-run.md`
