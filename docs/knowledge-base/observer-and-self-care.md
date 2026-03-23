# Observer And Self-Care

This guide explains how to inspect Spiderweb's current self-care state, which observer endpoints exist now, and how the fuller observer model is intended to evolve.

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
- the first observer JSON endpoints
- normal runtime logs

Observer endpoints available now:
- `/observer/overview`
- `/observer/benchmarks`
- `/observer/services`

Example:

```bash
cat ~/.spiderweb/runtime-health.json
cat ~/.spiderweb/runtime-health.json.baseline
curl http://127.0.0.1:8080/observer/overview
curl http://127.0.0.1:8080/observer/benchmarks
curl http://127.0.0.1:8080/observer/services
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
- they can inspect score, summary, recommendations, service state, and baseline/current benchmark state

Planned usage:
- users connect to a dashboard backed by the observer/control-plane layer
- they inspect service up/down state, recent events, self-care outcomes, drift history, and pipeline-agent presence
- they download HTML reports that compare baseline, check, and post-care results

That dashboard/report surface is design-approved, and the first read-only observer backend endpoints now exist, but the full dashboard and report exporter are not implemented yet.

## Planned Storage Split

Preferred design:
- `system.db` for observer, service status, benchmarks, self-care cycles, remediations, dashboard/report data
- `agent.db` for agent and pipeline detail logs

Rule of thumb:
- observer reads system summaries
- pipeline and agent detail stays out of the observer by default

## Interactive Design Reference

Use the interactive control-plane document for the architecture view:
- [../design/observer-control-plane.html](../design/observer-control-plane.html)

## Related Docs
- [../cookbook/runtime-maintenance.md](../cookbook/runtime-maintenance.md)
- [../TECHNICAL_GUIDE.md](../TECHNICAL_GUIDE.md)
