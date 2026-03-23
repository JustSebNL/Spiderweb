# Routing Patterns

Reusable patterns for deciding what Spiderweb does with normalized events.

## Purpose
Routing decides whether an event should be ignored, stored, summarized, escalated, or converted into a task.

Routing happens after:
- raw journaling
- normalization
- dedupe
- basic filtering

## Three-band routing model

### Band 1: Ignore or archive
Use for:
- duplicates
- low-value telemetry
- routine healthy status noise
- irrelevant updates

Output:
- journal only
- no model call
- no OpenClaw escalation

### Band 2: Store and summarize
Use for:
- meaningful but non-urgent updates
- thread activity worth compressing
- changes useful for later context but not immediate action

Output:
- journal
- optional cheap summarization
- memory/update candidate
- no immediate OpenClaw wake-up

### Band 3: Escalate
Use for:
- action required
- ambiguity requiring reasoning
- user-facing response needed
- security or compliance concerns
- workflow interruption conditions

Output:
- compact escalation packet
- OpenClaw wake-up or equivalent high-tier route

## Routing inputs
A router should consider:
- `source`
- `event_type`
- `priority_hint`
- `dedupe_count`
- `project_tags`
- `actor`
- `requires_response`
- cheap-model classification result if available

## Routing output shape

```json
{
  "band": "escalate",
  "reason": "security-sensitive mention with direct request for action",
  "route": "openclaw",
  "summary": "Deploy secret may have been exposed in Slack thread.",
  "references": ["evt_1001", "evt_1002"]
}
```

## Rule: routing must stay explainable
A route must always be explainable in one short reason string.

If the system cannot explain why it escalated something, the route is too opaque.

## Routing order
1. Hard deny/ignore rules
2. Hard escalation rules
3. cheap summarization/classification if needed
4. final route assignment

## Examples

### Example 1: ignore
- source: health probe
- event_type: `service.healthy`
- route: ignore
- reason: repetitive low-value heartbeat

### Example 2: summarize only
- source: GitHub
- event_type: `repo.push`
- payload: 14 commits in 3 minutes
- route: store and summarize
- reason: relevant project activity, but no immediate action required

### Example 3: escalate
- source: Slack
- event_type: `message.created`
- content asks for urgent deploy rollback
- route: escalate
- reason: direct action request with operational urgency
