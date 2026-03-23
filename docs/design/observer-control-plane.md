# Observer Control Plane Design

This document defines the next runtime architecture for Spiderweb's observer, self-care, logging, and dashboard layers.

It is the written implementation companion to:
- [observer-control-plane.html](./observer-control-plane.html)
- [../TECHNICAL_GUIDE.md](../TECHNICAL_GUIDE.md)
- [../knowledge-base/observer-and-self-care.md](../knowledge-base/observer-and-self-care.md)

## 1. Why This Redesign Exists

The current runtime already has a native maintenance service, but it is still too narrow for the control-plane role Spiderweb needs.

Current state:
- maintenance creates a startup baseline
- maintenance writes health snapshots
- maintenance runs a periodic self-check
- maintenance performs bounded remediation

What is still missing:
- a system-wide observer above services
- a dashboard-friendly query surface
- a clear split between system-wide logging and agent/pipeline detail
- a benchmark cycle that preserves baseline, pre-care, and post-care state
- an architecture that can grow without turning maintenance into an unbounded side brain

## 2. Design Goals

The redesign should:
- keep core runtime work separate from self-care and logging
- keep observer logic above services rather than inside pipeline management
- preserve bounded self-care behavior
- avoid mixing system logs with detailed agent transcripts
- create a dashboard/report surface that reads from a single aggregation layer
- keep the current runtime inspectable and cheap

## 3. Runtime Budget Model

Spiderweb should treat background behavior as a split policy:
- `90%` core runtime
- `5%` self-care
- `5%` logging and observability

Important constraint:
- this is a scheduling and behavior budget
- it is not permission for unbounded RAM, unbounded logs, or unbounded background tasks

Hard caps are still required:
- max database size
- max snapshot retention
- max log volume
- max concurrent background work
- max actions per self-care cycle

## 4. System Shape

```text
Core Runtime
  - gateway
  - cheap cognition runtime
  - trigger worker(s)
  - pipeline manager(s)
  - pipeline agents

Observer / Control Plane
  - service registry
  - health aggregation
  - drift detection
  - self-care cycle logging
  - dashboard query surface
  - report export

Self-Care Executor
  - stale pid cleanup
  - log trimming
  - bounded restart requests

Storage
  - system.db
  - agent.db
```

## 5. Ownership Boundaries

### 5.1 Core runtime

Core runtime owns:
- message handling
- routing
- pipeline scheduling
- pipeline execution
- normal service operation

Core runtime does not own:
- global health aggregation
- system-wide drift analysis
- long-term control-plane reporting

### 5.2 Pipeline manager

Pipeline manager owns:
- coordination of X pipeline agents
- queue depth
- pipeline-local failures and retries
- pipeline throughput management

Pipeline manager does not own:
- whole-system observer responsibilities
- cross-service health reporting beyond summarized status

### 5.3 Observer

Observer owns:
- system-wide service state
- health scoring
- benchmark comparison
- self-care cycle logging
- dashboard/report data surface
- default/debug view state

Observer does not own:
- pipeline business logic
- raw transcript logging by default
- service orchestration beyond bounded self-care requests

### 5.4 Self-care executor

Self-care executor owns:
- bounded checks
- bounded remediation actions
- returning structured results to the observer

Self-care executor does not own:
- historical logging
- dashboard rendering
- pipeline logic

## 6. Storage Split

The control plane should use two SQLite databases.

### 6.1 `system.db`

`system.db` is for system-wide control-plane state.

It should store:
- service registry and current service status
- observer events
- benchmark snapshots
- self-care cycles
- remediation actions
- dashboard query material
- HTML report history and metadata

This is the main source for:
- dashboard pages
- service up/down inspection
- self-care history
- benchmark comparisons

### 6.2 `agent.db`

`agent.db` is for detailed agent and pipeline activity.

It should store:
- agent records
- pipeline event logs
- queue state details
- agent sessions
- task-level error records
- detailed pipeline execution traces

This should not be the default observer data source.

Rule:
- observer reads summaries and references
- detailed pipeline and agent history remains in `agent.db`

### 6.3 Shared constraints

Both databases should use:
- WAL mode
- short write transactions
- explicit retention policies
- correlation IDs for cross-referencing related events

## 7. Data Contracts

### 7.1 `ServiceStatus`

```json
{
  "service_id": "gateway",
  "service_type": "gateway",
  "state": "up",
  "last_seen_at": "2026-03-23T15:00:00Z",
  "latency_ms": 24,
  "error_rate": 0,
  "queue_depth": 0,
  "last_error": ""
}
```

### 7.2 `ObserverEvent`

```json
{
  "event_type": "service.degraded",
  "service_id": "cheap-runtime",
  "severity": "warn",
  "message": "Latency drift is materially above baseline",
  "payload": {
    "baseline_ms": 180,
    "current_ms": 420
  },
  "correlation_id": "corr_01",
  "created_at": "2026-03-23T15:00:00Z"
}
```

### 7.3 `BenchmarkSnapshot`

```json
{
  "snapshot_id": "snap_01",
  "kind": "pre_care",
  "score": 71,
  "summary": "Cheap cognition runtime degraded",
  "created_at": "2026-03-23T15:00:00Z"
}
```

Kinds:
- `baseline`
- `periodic`
- `pre_care`
- `post_care`

### 7.4 `SelfCareCycle`

```json
{
  "cycle_id": "cycle_01",
  "baseline_snapshot_id": "snap_baseline",
  "pre_snapshot_id": "snap_pre",
  "post_snapshot_id": "snap_post",
  "status": "improved",
  "improvement_delta": 13,
  "started_at": "2026-03-23T15:00:00Z",
  "completed_at": "2026-03-23T15:00:08Z"
}
```

### 7.5 `RemediationAction`

```json
{
  "action_type": "restart_request",
  "target_service": "cheap-runtime",
  "result": "success",
  "message": "Requested restart after dead pid detection"
}
```

## 8. Benchmark Model

Every self-care cycle should preserve three important views:
1. startup baseline
2. pre-care check
3. post-care result

This is required so Spiderweb can answer:
- how far the runtime drifted from startup
- whether self-care improved the situation
- whether self-care had no effect
- whether self-care caused a regression

The observer should log all three.
The self-care executor should not overwrite the pre-care state with the post-care state.

## 9. API Surface

The observer should expose a dashboard-friendly API through the existing health server or a closely related service layer.

Recommended endpoints:
- `GET /observer/overview`
- `GET /observer/services`
- `GET /observer/events`
- `GET /observer/benchmarks`
- `GET /observer/self-care/cycles`
- `GET /observer/reports/:id`
- `POST /observer/mode`
- `POST /observer/self-care/run`

Later additions:
- `GET /observer/stream` for SSE or websocket updates
- `POST /observer/actions/restart`
- `POST /observer/actions/ack`

## 10. Dashboard Modes

### 10.1 Default mode

Default mode should be low-noise.

It should show:
- service up/down/degraded state
- recent errors and warnings
- benchmark status
- recent self-care outcomes
- latest recommendations

### 10.2 Debug mode

Debug mode should add detail, not chaos.

It may add:
- more event payload detail
- transition history
- remediation reasoning
- richer timing data

It still must stay bounded by retention and storage limits.

## 11. Dashboard UX Requirements

The dashboard should feel like a professional operator console, not a plain CRUD admin page.

Visual direction:
- dark base with glass-like layered panels
- warm accent system using ember, flame, and charred-brown tones
- clear semantic status colors for healthy, warning, degraded, and offline states
- motion used for clarity, not novelty

Interaction expectations:
- hover lift and reveal for cards
- expandable menus and control drawers
- readable notification banners for info, success, warning, and error
- clear state transitions when services or agents go online, offline, or degraded

The dashboard should support:
- overview metrics
- service registry view
- pipeline manager view
- pipeline agent status view
- self-care and benchmark history view
- event and notification view
- rolling 24-hour operational stats for errors and critical events

### 11.1 Pipeline agent visibility

The dashboard must expose pipeline agent presence as a first-class surface.

For each pipeline agent, show:
- agent id or display name
- owning pipeline manager
- pipeline name
- current state
- last seen time
- current task summary or last task summary
- recent error or degradation flag

Recommended state taxonomy:
- `online`
- `idle`
- `busy`
- `degraded`
- `offline`
- `restarting`

### 11.2 Notification model

The dashboard should support a compact notification stack with consistent severity handling.

Recommended severities:
- `info`
- `success`
- `warning`
- `error`

Recommended notification sources:
- observer events
- self-care actions
- service status changes
- pipeline manager summaries

Notification behavior should be:
- bottom-right toast stack
- oldest item at the top of the stack
- newer items appended below
- timed fade-out after a bounded display window
- dismissible without disturbing the layout

### 11.3 Recent operational stats

The dashboard should include a short-horizon operational window, starting with the last 24 hours.

Minimum tracked metrics:
- total errors in the last 24 hours
- total critical events in the last 24 hours
- notable critical-event timeline entries

Examples of critical events:
- service hard-down transitions
- offline pipeline agents
- failed restart attempts
- repeated self-care failure on the same target
- gateway unreachable states

### 11.4 Menus and controls

The dashboard should provide:
- a mode switch between default and debug
- a panel or menu for dashboard actions
- report export controls
- filtering by service, pipeline, or severity

Control surfaces should open smoothly and stay visually aligned with the glass-panel system.

## 12. Report Export

The observer should support downloadable HTML reports.

A report should include:
- generated time
- current service summary
- startup baseline
- latest pre-care check
- latest post-care result
- cycle history
- remediation summary
- recent warnings/errors

This should be:
- a single HTML file
- largely self-contained
- suitable for offline sharing

## 13. Current Implementation Mapping

Current code that already exists:
- `pkg/maintenance/service.go`
- `pkg/health/server.go`
- `cmd/spiderweb/internal/gateway/helpers.go`

What these currently provide:
- health endpoints
- maintenance snapshots
- startup baseline
- bounded remediation

What still needs to be added:
- observer service
- `system.db`
- `agent.db`
- observer endpoints
- dashboard pages
- report exporter

## 14. Proposed Repo Layout

Recommended additions:
- `pkg/observer/`
- `pkg/systemdb/`
- `pkg/agentdb/`

Likely files:
- `pkg/observer/service.go`
- `pkg/observer/events.go`
- `pkg/observer/benchmarks.go`
- `pkg/observer/reports.go`
- `pkg/systemdb/store.go`
- `pkg/systemdb/schema.go`
- `pkg/agentdb/store.go`
- `pkg/agentdb/schema.go`

## 15. Phased Implementation Plan

### Phase 1: Foundation

Build the minimum observer/control-plane backend:
- create `system.db`
- add observer event and benchmark storage
- keep the current maintenance service as the executor
- write self-care cycle records to `system.db`

### Phase 2: API Surface

Add observer endpoints:
- overview
- services
- benchmarks
- self-care cycles
- mode state

### Phase 3: Dashboard

Add a local dashboard view:
- overview page
- services page
- logs/events page
- benchmark and self-care page

### Phase 4: Reports

Add downloadable HTML export:
- current system report
- historical self-care cycle summary

### Phase 5: Agent split

Add `agent.db` and move detailed pipeline/agent logging out of the control plane.

## 16. Non-Goals

This redesign does not mean:
- observer becomes the main runtime brain
- observer replaces pipeline managers
- self-care gets permission for expensive reasoning loops
- debug mode becomes a raw transcript firehose
- the project must expose a production multi-user control plane immediately

## 17. Recommended Immediate Next Steps

The next implementation steps should be:
1. add a minimal `pkg/observer/` package
2. introduce `system.db` and benchmark/self-care cycle tables
3. make maintenance return structured cycle results to the observer
4. add read-only observer endpoints to `pkg/health/server.go`
5. add a first local dashboard page after the backend shape is stable
