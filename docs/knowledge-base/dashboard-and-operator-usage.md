# Dashboard And Operator Usage

This guide explains how to use the observer dashboard as an operator surface instead of only reading raw JSON.

## Current Entry Points

Interactive UI:
- [../../ui/dashboard/observer.html](../../ui/dashboard/observer.html)
- `http://127.0.0.1:8080/observer/ui`

Observer JSON:
- `/observer/dashboard`
- `/observer/overview`
- `/observer/services`
- `/observer/agents`
- `/observer/agents/summary`
- `/observer/events`
- `/observer/self-care/cycles`
- `/observer/stats/24h`
- `/observer/reports/latest`
- `/observer/journal/latest`

## What The Dashboard Shows

Top shell:
- `Core` status light: tracked service uptime summary
- `Observer` status light: current observer health score and summary state
- `Attention` status light: how many tracked services or agents currently need review

Overview area:
- live health gauges
- benchmark comparison
- current observer summary and recommendations
- refresh state and next refresh countdown

Service area:
- tracked managed services
- current state
- lightweight service action/message
- restart availability for managed services

Agent area:
- summarized agent presence from `agent.db`
- current state
- pipeline id
- manager id
- last seen
- last task summary when available

Artifacts area:
- latest report metadata
- latest journal metadata
- journal snippet

Task lane:
- observer-shaped operational queue
- running work stays visually muted
- attention states stay visible

## Common Operator Actions

From the dashboard UI:
- refresh observer data
- run bounded self-care
- generate a fresh HTML report
- generate a journal entry
- clear old observer events
- restart a managed service when restart is available

From HTTP directly:

```bash
curl http://127.0.0.1:8080/observer/dashboard
curl http://127.0.0.1:8080/observer/reports/latest
curl http://127.0.0.1:8080/observer/journal/latest
curl -X POST http://127.0.0.1:8080/observer/actions/self-care/run
curl -X POST http://127.0.0.1:8080/observer/reports/generate
curl -X POST http://127.0.0.1:8080/observer/journal/generate
curl -X POST http://127.0.0.1:8080/observer/actions/clear-events \
  -H 'Content-Type: application/json' \
  -d '{"retention_days":30}'
```

Restarting a managed service:

```bash
curl -X POST http://127.0.0.1:8080/observer/actions/restart \
  -H 'Content-Type: application/json' \
  -d '{"service":"trigger"}'
```

## How To Read Services

Use the service panel to answer:
- which managed services are up
- which ones are degraded or down
- which ones Spiderweb can restart itself

State guidance:
- `up` / `online` / `healthy`: running normally
- `watch`: still up, but drift or instability is present
- `degraded`: needs attention and may require restart or self-care
- `restarting`: a bounded restart request is in flight
- `down` / `offline` / `error`: hard failure state

Use `Read more...` on a service card for the current operator context.

## How To Read Agents

The agent panel is summary-only by design.

It should tell you:
- which agents are present
- which pipeline they belong to
- who their manager is
- whether they are online, idle, busy, degraded, restarting, or offline
- when they were last seen
- what they were last doing in compact form

It should not act like a transcript browser by default.

## How To Read 24-Hour Stats

The `Last 24 Hours` panel is short-horizon operational context.

Use it for:
- error count
- critical event count
- recent timeline events

This section helps answer:
- did something just start failing today
- are failures isolated or repeating
- is there a visible cluster of attention-worthy events

## How To Read Self-Care Cycles

The self-care panel shows recent bounded maintenance cycles.

Read each cycle as:
- when it ran
- what the post-care summary says
- what the score delta was

Use `Read more...` for:
- baseline score
- pre-check score
- post-care score
- last remediation action

Rule:
- never collapse pre-check and post-care into one indistinct result
- improvement and regression both need to remain visible

## Reports And Journal

Reports:
- HTML snapshot of current observer state
- useful for handoff and offline review

Journal:
- human-readable observer narrative grounded in the day’s recorded events
- should stay bounded and fact-based even when the tone has personality

Current status:
- manual generation exists now
- timed scheduled generation is still an open follow-up

## Debug Mode

Current dashboard debug mode does more than only change refresh cadence.

It also:
- pulls a deeper event slice
- shows richer observer context
- makes the task lane more event-heavy

Debug mode should be used when:
- an operator is actively investigating drift, failures, or repeated attention states

Default mode should be used when:
- the operator wants the concise system picture

## Related Docs
- [Observer And Self-Care](./observer-and-self-care.md)
- [Startup And Daily Use](./startup-and-daily-use.md)
- [Troubleshooting](./troubleshooting.md)
