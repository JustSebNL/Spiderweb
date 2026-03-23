# Spiderweb Project Vision Highlights

Source basis: direct extraction from [download.mhtml](D:/Dev/codebase/Dev/Spiderweb/download.mhtml), which contains the saved `OpenClaw Redesign Strategy` conversation.

This summary strips away later style/persona discussion and keeps only the core project idea, architecture, and design vision.

## Core idea

Spiderweb exists to stop OpenClaw from wasting tokens on constant polling, heartbeat-heavy monitoring, and repeated state collection.

Instead of OpenClaw repeatedly checking Slack, GitHub, Notion, files, logs, and other systems itself, Spiderweb should become an intake fabric that watches those systems, collects deltas, filters noise, and only wakes OpenClaw when something actually matters.

The shortest version is:

`OpenClaw should move from agent-led polling to fabric-led event intake.`

## The main problem being solved

The original discussion identifies token burn as coming from a few recurring patterns:

- frequent heartbeat prompts
- repeated “anything new?” loops
- oversized routine monitoring context
- multiple connectors each doing their own polling logic
- re-reading full state instead of changed state only
- using an LLM where deterministic rules would be cheaper and more reliable

So the problem is not just prompt size. The deeper problem is the flow model.

## The intended replacement model

Old model:

`OpenClaw <-> Slack / Notion / GitHub / DB / Files / etc`

That means OpenClaw keeps checking every source itself, which turns the main model into a monitor rather than a reasoner.

New model:

`Sources -> Spiderweb -> Event bus / journal -> OpenClaw`

In this model:
- connectors poll or receive webhooks
- changes become small normalized events
- cheap rules decide what matters
- a small model may summarize or classify only when useful
- OpenClaw only wakes up for meaningful work

## The architectural vision

Spiderweb is envisioned as a layered intake mesh.

### 1. Connectors

Each external system gets a small collector.

Examples mentioned:
- Slack
- GitHub
- Notion
- filesystem
- email
- calendar
- website/scraper
- database change collector

Their role is intentionally narrow:
- poll or receive webhook
- detect change
- emit tiny structured events

They should not reason, strategize, or call the main LLM directly.

## 2. Event normalizer

All connector outputs should be converted into one internal event format so downstream logic only has to understand one schema.

The idea is that every source becomes a shared structured event with things like:
- event id
- source
- event type
- timestamp
- actor
- payload
- dedupe key
- version

This reduces complexity and makes routing cheaper.

## 3. Filter / gatekeeper layer

Most incoming events should never reach the main agent.

This layer should do:
- dedupe
- rate limiting
- batching
- noise suppression
- burst collapsing
- priority assignment
- routing by project/topic/source

Examples from the source discussion:
- many filesystem changes collapse into one event
- a GitHub push with many commits becomes one delta
- repeated healthy pings never wake an LLM
- Slack typing, reactions, and similar low-value noise can be ignored

This is one of the main token-saving layers.

## 4. Cheap cognition layer

Only after deterministic filtering should the system decide whether a given item needs:
- rules only
- a tiny model
- the main OpenClaw model

The intended order is:

`rules first -> heuristics/embeddings second -> tiny model third -> main model last`

The tiny model is meant for:
- priority classification
- one-line summaries
- entity extraction
- clustering related events
- deciding if escalation is needed

It is explicitly not meant to be the source of truth for ingestion.

## 5. OpenClaw as the specialist brain

OpenClaw should only be invoked when:
- action is needed
- ambiguity is high
- planning or reasoning is required
- user-facing output is required
- memory/project journal updates are needed

That means OpenClaw stops being a watcher and becomes a specialist decision engine.

## The central design principle

The strongest idea in the source conversation is:

`Turn heartbeats into stateful watchers.`

Instead of repeatedly asking “anything new?”, each connector should track its own state using things like:
- last cursor
- last sync timestamp
- last seen IDs
- per-source checkpoint
- retry state

Then the connector asks:

`What changed since checkpoint X?`

not:

`Give me everything again.`

This is a major part of the cost reduction vision.

## Observation vs reasoning split

A key design insight is to separate these concerns clearly:

### Observation plane

Spiderweb:
- collects changes
- stores facts
- emits events

### Cognition plane

OpenClaw:
- reasons on selected events
- plans actions
- writes outputs

### Execution plane

Workers/tools:
- send messages
- open issues
- update docs
- run workflows
- schedule tasks

This separation is intended to reduce token burn and make failures easier to isolate.

## The “single point” idea, but done safely

The source discussion agrees with the idea that information should flow to a single intake point, but warns against turning that into a fragile monolith.

The better interpretation is:
- one logical intake spine
- many modular collectors
- one journaled event contract
- multiple reducers/workers behind it

So Spiderweb should become a single entry model, not a single failure model.

## Role of Trigger.dev

Trigger.dev is considered a good fit for orchestration and workflow execution around Spiderweb.

Good uses:
- source-specific connector jobs
- scheduled fallback polling
- retries and backoff
- batching windows
- workflow state tracking
- summarization jobs
- escalation workflows

Bad use:
- turning Trigger.dev into the place where all heavy reasoning always happens

The intended role is nervous system orchestration, not brain replacement.

## Event journal / memory spine

The project vision includes a central event journal rather than one fragile central brain.

Suggested storage ideas in the source discussion include:
- PostgreSQL event tables
- Redis streams/queues
- NATS or Kafka at larger scale
- SQLite only for local development or replay
- object storage for large payload overflow

The key point is not the exact database. The key point is:
- all events are journaled before interpretation
- source truth is preserved
- decisions can be audited
- state can be replayed or rebuilt

## Anti-token rules that drive the project

These ideas come through as hard design commandments:

- No LLM for polling.
- No full-context refresh unless forced.
- Always prefer deltas, cursors, and checkpoints.
- Do not wake the main model for low-value noise.
- Journal events before interpretation.
- Use incremental summaries rather than re-summarizing everything.
- Heartbeats should be infrastructure-level, not LLM-level.
- Every connector should support dedupe and replay.

## The intended intelligence stack

The design calls for a tiered system:

### Tier 0: deterministic
- webhooks
- file watchers
- cursors
- hashes
- metadata rules
- dedupe windows
- priority rules

### Tier 1: cheap reasoning
- tiny local or cheap LLM
- classify
- summarize
- extract action items
- filter spam/noise

### Tier 2: main cognition
- OpenClaw for planning, coding decisions, security review, architecture reasoning, and multi-system orchestration

This is important because it defines Spiderweb as `mechanical first, intelligent second`.

## The role of a small model like Youtu-LLM

The source discussion is clear on one point: a small LLM should not do collection itself.

It may be useful after collection for:
- summarization
- classification
- tagging
- deciding if escalation is warranted

So the model is a preprocessor or triage clerk, not the main intake authority.

## Minimal viable direction

The source conversation proposes an incremental path instead of building everything at once.

### Phase 1
- connector runners
- normalized event schema
- durable event journal
- rule-based filtering
- OpenClaw intake queue
- little or no extra LLM logic

### Phase 2
- add a tiny model for summary, intent tagging, clustering, and priority

### Phase 3
- add cross-source correlation, routing, memory extraction, and more autonomous workflows

### Phase 4
- add self-tuning polling cadence, trust/source scoring, anomaly detection, and budget-aware escalation

## The main risk

The conversation repeatedly warns against one failure mode:

Do not accidentally build a second giant agent hidden behind the first one.

Spiderweb should remain:
- mechanical first
- cheap by default
- explainable
- replayable
- bounded in scope

If it turns into a vague autonomous brain, the same token problem returns.

## The deeper product vision

Spiderweb is not mainly imagined as a chatbot.

It is imagined as an autonomous signal mesh or nervous system for AI operations.

Its purpose is to:
- sit close to real systems
- gather and reduce signal
- preserve auditability
- wake expensive reasoning only when justified
- lower cost without losing operational awareness

That makes Spiderweb less like a standalone assistant and more like an intake-and-routing substrate for higher-level AI work.

## Best one-paragraph summary

Spiderweb is meant to redesign OpenClaw around event-fed intake rather than heartbeat-heavy polling. Small collectors watch external systems, normalize changes into a shared event schema, filter and compress noise, optionally use a small model for cheap triage, and only then escalate compact decision packets to OpenClaw. The whole vision is about separating observation from reasoning so the expensive model stops wasting tokens on monitoring and instead focuses on actual planning, action, and user-relevant work.

## Best one-sentence summary

Spiderweb is an event-driven intake mesh that turns OpenClaw from a constant poller into a selective reasoning engine.
