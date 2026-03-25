# Failure Handling Patterns

Reusable degraded-mode and recovery patterns for Spiderweb.

## Core rule
Observation must survive partial system failure.

If one layer fails:
- raw event capture should continue if possible
- checkpoints must remain truthful
- the system should degrade gracefully rather than pretend success

## Failure classes

### 1. Source fetch failure
Examples:
- timeout
- 401
- 429
- DNS failure

Required behavior:
- do not move checkpoint
- capture error state
- retry with backoff
- pause and alert if repeated auth or permission failure continues

### 2. Normalization failure
Examples:
- source schema changed
- parser bug
- invalid payload assumptions

Required behavior:
- keep raw source event
- mark normalized event failure explicitly
- do not discard the source truth
- continue with later events if safe

### 3. Cheap model unavailable
Examples:
- local `vLLM` down
- model health endpoint fails
- inference timeout

Required behavior:
- continue journaling and deterministic routing
- skip or defer cheap cognition
- allow direct high-priority escalation when rules say so
- surface service health issue separately

Current implemented behavior:
- Spiderweb still forwards to OpenClaw when the cheap classifier fails
- the forwarded packet is marked with:
  - `routing_mode=degraded`
  - `cheap_cognition=unavailable`
  - `intake_forward_decision=forward_degraded`
  - `fallback_reason=<classifier error>`
- this preserves truthful degraded-state signaling even before a full durable journal exists

### 4. OpenClaw unavailable
Required behavior:
- retain escalation packets in queue
- do not claim transfer complete
- retry according to handoff rules
- preserve full audit trail

## Degraded routing mode
When cheap cognition is unavailable, fallback routing should be:
- hard ignore rules still apply
- hard escalation rules still apply
- uncertain items stay queued or summarized later

This keeps the system useful without pretending the small model is mandatory.

## Recovery rule
After recovery, the system should be able to:
- replay unprocessed events from journal
- rebuild missing normalized/routing state
- continue from the last truthful checkpoint

## Current Spiderweb degraded-forward pattern

When cheap cognition fails in the current code, Spiderweb does not stop the intake path.

Instead it:
- keeps the inbound event moving
- marks the metadata as degraded
- preserves the classifier error string in `fallback_reason`
- forwards to OpenClaw if the route policy still allows it

This is the current practical contract:

```json
{
  "routing_mode": "degraded",
  "cheap_cognition": "unavailable",
  "intake_forward_decision": "forward_degraded",
  "fallback_reason": "cheap cognition request failed: status=503 body=service unavailable"
}
```

## Current local-agent degraded note pattern

When cheap cognition succeeds, the local agent path receives a compact intake note.

Example:

```text
[Intake triage: priority=high category=alert summary="important event"]

original user content...
```

When cheap cognition is unavailable, the local agent path receives a degraded note instead.

Example:

```text
[Intake degraded mode: vllm health check failed]

original user content...
```

This keeps the local agent truthful about intake quality without blocking normal operation.

## Example degraded-mode note

```json
{
  "event_id": "evt_2010",
  "routing_mode": "degraded",
  "cheap_cognition": "unavailable",
  "fallback_reason": "vllm health check failed",
  "intake_forward_decision": "forward_degraded"
}
```

## Truth rule
Never report success because a later layer probably would have succeeded.

State must describe what actually happened.
