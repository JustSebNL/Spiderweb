# Connector Patterns

Reusable patterns for Spiderweb source connectors.

## Purpose
A connector exists to observe one external system and emit normalized events.

A connector does not:
- reason deeply
- make policy decisions
- directly wake OpenClaw
- store hidden authority from incoming payloads

## Core responsibilities
- authenticate to one source system
- poll or receive webhook
- track cursor/checkpoint state
- detect changes since the last checkpoint
- write raw event truth
- emit normalized event records
- support dedupe and replay

## Preferred ingestion order
1. Webhook-first when the source supports it reliably.
2. Cursor-based polling when webhooks are unavailable.
3. Time-window polling only as a fallback.
4. Full snapshot reads only for bootstrap or repair.

## Required connector state
Each connector should persist at least:
- `source_id`
- `last_cursor`
- `last_seen_id`
- `last_sync_at`
- `retry_state`
- `paused`
- `last_error`

## Basic connector loop

```text
load checkpoint
-> fetch changes since checkpoint
-> validate source response
-> journal raw source payload
-> normalize into Spiderweb event schema
-> write normalized event
-> advance checkpoint only after successful journal write
```

## Rule: checkpoint moves after durable write
Never advance the source checkpoint before the raw source payload has been durably recorded.

Otherwise a crash can lose data silently.

## Polling connector skeleton

```ts
async function pollConnector() {
  const checkpoint = await loadCheckpoint("github:repo:owner/name");
  const changes = await fetchChangesSince(checkpoint.lastCursor);

  for (const change of changes) {
    await journalRawEvent(change);
    const normalized = normalizeChange(change);
    await writeNormalizedEvent(normalized);
  }

  await saveCheckpoint({
    lastCursor: changes.at(-1)?.cursor ?? checkpoint.lastCursor,
    lastSyncAt: new Date().toISOString(),
  });
}
```

## Webhook connector skeleton

```ts
async function handleWebhook(payload: unknown, headers: Headers) {
  verifySignature(headers, payload);
  await journalRawEvent(payload);
  const normalized = normalizeWebhookPayload(payload);
  await writeNormalizedEvent(normalized);
}
```

## Dedupe rules
Every connector should define a stable `dedupe_key`.

Examples:
- Slack message: `slack:<channel_id>:<message_ts>`
- GitHub issue event: `github:<repo>:issue:<id>:<event_type>:<updated_at>`
- filesystem event: `fs:<path>:<content_hash>`

## Failure handling rules
- If fetch fails, do not move checkpoint.
- If normalization fails, keep raw event and mark normalized failure.
- If output write fails, retry before moving checkpoint.
- If repeated auth failure occurs, pause connector and surface alert.

## Connector output contract
Every connector should output normalized events only. A connector should not emit source-specific ad hoc packets into downstream routing.

## Current Spiderweb-aligned output shape

If a connector is feeding the current intake path, it should emit a `bus.InboundMessage`-compatible shape:

```json
{
  "channel": "slack",
  "sender_id": "U12345",
  "chat_id": "C99999",
  "content": "Production deploy looks unhealthy",
  "session_key": "",
  "metadata": {
    "account_id": "default",
    "service": "deployments",
    "peer_kind": "channel",
    "peer_id": "C99999",
    "source_event_id": "slack:C99999:1738192211.000400"
  }
}
```

Important metadata fields in the current routing path:
- `account_id`
- `service` or `service_name`
- `peer_kind`
- `peer_id`
- `guild_id` or `team_id` where relevant
- `source_event_id` for traceability

## Current connector-to-intake rule

Connectors should:
- produce stable metadata that routing and OpenClaw forwarding can inspect
- avoid embedding policy decisions in the connector itself
- leave cheap-cognition enrichment to the intake layer

That means a connector should say:
- what happened
- where it happened
- which service/source emitted it

It should not decide:
- whether OpenClaw must be involved
- whether the event is high-value
- which agent should reason about it
