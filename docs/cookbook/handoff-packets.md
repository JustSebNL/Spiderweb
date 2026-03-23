# Handoff Packet Patterns

Reusable packet shapes for sending work from Spiderweb to OpenClaw.

## Purpose
A handoff packet should give OpenClaw the minimum context needed to reason or act, without making it reconstruct the entire source world from scratch.

## Rule
Spiderweb hands off compact deltas, not raw full-state snapshots, unless explicitly required for recovery or audit.

## Minimal handoff packet

```json
{
  "handoff_id": "handoff_001",
  "route": "openclaw",
  "reason": "direct request for deploy review",
  "priority": "high",
  "summary": "A Slack thread requests review of the deploy script after a failed rollout.",
  "source_refs": ["evt_1001", "evt_1002"],
  "project": "spiderweb",
  "requested_action": "assess and propose next step",
  "created_at": "2026-03-22T23:00:00Z"
}
```

## Fields that matter most
- `handoff_id`: unique transfer id
- `route`: intended destination
- `reason`: why this was escalated
- `priority`: urgency class
- `summary`: compact human-readable delta
- `source_refs`: traceable event ids
- `requested_action`: what OpenClaw is expected to do

## Optional enrichment
Add only when necessary:
- `related_thread_summary`
- `memory_refs`
- `security_flags`
- `checkpoint_ref`
- `dedupe_group`

## What should not be included by default
- full raw payload history
- repeated unchanged context
- decorative narration
- speculative conclusions presented as fact

## Handoff completion rule
A handoff is not complete when Spiderweb sends the packet.
A handoff is complete only when receipt is confirmed through the expected handshake path.
