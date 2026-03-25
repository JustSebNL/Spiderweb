# Observer And Self-Care

This guide explains how to inspect Spiderweb's current self-care state, which observer endpoints exist now, and how the still-open observer follow-up items are expected to evolve.

## Current State

Today, Spiderweb has a native maintenance/self-care service.

Current behavior:
- creates a startup baseline shortly after launch
- writes runtime health snapshots
- compares later checks against the startup baseline
- runs periodic checks every 12 hours by default
- performs bounded remediation only

Current files:
- `~/.spiderweb/runtime-health.json`
- `~/.spiderweb/runtime-health.json.baseline`

Current implementation:
- `pkg/maintenance/service.go`

## How To Read The Current Self-Care Files

Baseline snapshot:
- `~/.spiderweb/runtime-health.json.baseline`

Latest runtime snapshot:
- `~/.spiderweb/runtime-health.json`

Useful things to inspect:
- overall score
- summary
- recommendations
- current-vs-baseline drift
- last remediation action

## Current Access Path

Right now, users inspect self-care primarily through:
- the generated JSON snapshot files
- gateway health endpoints
- the observer JSON endpoints
- the interactive observer dashboard
- normal runtime logs

Observer endpoints available now:
- `/observer/dashboard`
- `/observer/overview`
- `/observer/benchmarks`
- `/observer/services`
- `/observer/agents`
- `/observer/agents/summary`
- `/observer/events`
- `/observer/self-care/cycles`
- `/observer/stats/24h`
- `/observer/actions/restart`
- `/observer/actions/self-care/run`
- `/observer/actions/clear-events`
- `/observer/journal/generate`
- `/observer/journal/latest`
- `/observer/reports/generate`
- `/observer/reports/latest`

Example:

```bash
cat ~/.spiderweb/runtime-health.json
cat ~/.spiderweb/runtime-health.json.baseline
curl http://127.0.0.1:8080/observer/overview
curl http://127.0.0.1:8080/observer/dashboard
curl http://127.0.0.1:8080/observer/benchmarks
curl http://127.0.0.1:8080/observer/services
curl http://127.0.0.1:8080/observer/agents
curl http://127.0.0.1:8080/observer/agents/summary
curl http://127.0.0.1:8080/observer/events
curl http://127.0.0.1:8080/observer/self-care/cycles
curl http://127.0.0.1:8080/observer/stats/24h
curl -X POST http://127.0.0.1:8080/observer/actions/self-care/run
curl -X POST http://127.0.0.1:8080/observer/actions/restart -d '{"service":"trigger"}' -H 'Content-Type: application/json'
curl -X POST http://127.0.0.1:8080/observer/actions/clear-events -d '{"retention_days":30}' -H 'Content-Type: application/json'
curl -X POST http://127.0.0.1:8080/observer/journal/generate
curl http://127.0.0.1:8080/observer/journal/latest
curl -X POST http://127.0.0.1:8080/observer/reports/generate
curl http://127.0.0.1:8080/observer/reports/latest
curl http://127.0.0.1:8080/observer/reports/latest?format=html
```

## Planned Observer Model

Design direction:
- observer above services
- pipeline managers stay responsible for pipeline agents
- observer tracks system-wide health, drift, errors, and self-care outcomes
- observer logs self-care cycles rather than self-care owning its own narrative
- dashboard reads from observer state instead of scraping each service directly

## Benchmark Model

The intended self-care benchmark cycle is:
1. startup baseline
2. pre-care check
3. post-care result

This is important because it shows:
- how far the runtime drifted
- whether self-care improved it
- whether self-care failed to help
- whether self-care made something worse

## Dashboard And Reports

Current usage:
- users can query read-only observer JSON from the gateway
- `dashboard` gives one consolidated payload for overview, services, agents, recent events, and self-care history
- the interactive dashboard in `ui/dashboard/observer.html` consumes that same observer payload
- they can trigger an on-demand bounded self-care run
- they can request bounded restart actions for managed services
- they can clear old observer events with a bounded retention-based action
- they can generate a daily observer journal entry grounded in the recorded day’s events
- they can generate and fetch downloadable HTML observer reports
- they can inspect score, summary, recommendations, service state, agent presence, and baseline/current benchmark state
- they can inspect rolling 24-hour error and critical-event counts

Current dashboard/backend status:
- the observer dashboard JSON surface exists
- the interactive operator dashboard exists
- HTML report generation exists
- journal generation and latest-journal retrieval exist
- restart and self-care actions exist for bounded managed targets
- clear-old-events exists as a bounded operator action

Still planned:
- richer benchmark triplets for baseline, pre-care, and post-care
- more bounded operator actions beyond restart
- timed journal generation near day rollover instead of only manual generation

## Planned Storage Split

Preferred design:
- `system.db` for observer, service status, benchmarks, self-care cycles, remediations, dashboard/report data
- `agent.db` for agent and pipeline detail logs

Current implementation status:
- `system.db` now exists as the first observer persistence layer
- maintenance writes benchmark snapshots, service status, and observer events into it
- `agent.db` now exists as the first agent-presence layer
- the intake loop writes lightweight agent last-seen records into it
- the observer exposes that data through `/observer/agents`

Rule of thumb:
- observer reads system summaries
- observer reads lightweight agent presence from `agent.db`
- detailed pipeline and agent history still stays out of the observer by default

## Interactive Design Reference

Use the interactive control-plane document for the architecture view:
- [../../ui/dashboard/observer.html](../../ui/dashboard/observer.html)

## Related Docs
- [Dashboard And Operator Usage](./dashboard-and-operator-usage.md)
- [../cookbook/runtime-maintenance.md](../cookbook/runtime-maintenance.md)
- [../TECHNICAL_GUIDE.md](../TECHNICAL_GUIDE.md)
