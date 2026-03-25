# Runtime Maintenance

Spiderweb runtime maintenance is meant to be a low-noise local janitor, not a benchmark daemon.

## Current Pattern
- create a startup baseline shortly after gateway launch
- store health snapshots in `~/.spiderweb/runtime-health.json`
- compare later checks against the startup baseline
- run periodic checks every 12 hours by default
- keep actions bounded by a small maintenance budget
- defer noncritical cleanup when the runtime was recently active
- apply restart backoff so dead-process remediation does not flap

## Preferred Signals
- owned process pid is still alive
- pid file is stale or empty
- Spiderweb-owned log file has grown beyond limit
- cheap-cognition latency has drifted above baseline
- cheap-cognition failures are accumulating

## Allowed Actions
- remove stale Spiderweb-owned pid files
- trim oversized Spiderweb-owned log files
- request restart of an owned dead process
- write health recommendations for later review

## Defer And Backoff
- recent cheap-cognition activity can defer noncritical maintenance actions
- recent restart remediation can suppress another restart request until backoff expires
- health observation still runs even when cleanup/remediation is deferred

## Avoid
- frequent HTTP liveness chatter
- large synthetic benchmark prompts
- blanket cache deletion
- restart storms

## Current Implementation Files
- `pkg/maintenance/service.go`
- `cmd/spiderweb/internal/gateway/helpers.go`
- `scripts/stop_trigger_worker.sh`
- `scripts/stop_brain_vllm.sh`
