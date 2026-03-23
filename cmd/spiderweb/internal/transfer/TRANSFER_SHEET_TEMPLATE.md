# Spiderweb Transfer Sheet (Template)

Fill rule: if any field is unknown, write `UNKNOWN` (do not leave blank).

## 0) Transfer Safety Checklist (Must Be Completed)

- [ ] Service/pipeline is named and uniquely identifiable.
- [ ] Intake mechanism is described precisely (push vs poll, endpoints, auth).
- [ ] Trust level is explicit (information-only by default).
- [ ] Routing target is explicit (where work lands).
- [ ] Checkpoint/cursor is explicit (or explicitly `none`).
- [ ] Failure + retry behavior is explicit (including freeze/stop conditions).
- [ ] Usage stats are filled (even rough estimates).
- [ ] Owner handoff rule is explicit (who owns watch-duty after transfer).

## 1) Identity

- **Service / Pipeline Name:** `<name>`
- **Category:** `<chat | email | repo | files | alerts | other>`
- **Owner (Current Watch):** `<OpenClaw | Spiderweb | Human>`
- **Target Receiver:** `<Spiderweb agent id/name>`
- **Transfer Doc Version:** `<v0.x>`
- **Last Updated:** `<YYYY-MM-DD>`

### Required Identifiers (to prevent ambiguity)

- **Service unique key:** `<stable id string>`
- **Conversation key:** `<thread id | channel id | repo id | mailbox id | folder id | path>`
- **Message/event unique id field:** `<field name + example>`

## 2) Purpose (Why This Exists)

- **What is this service for?** `<1-3 sentences>`
- **What counts as “work that matters”?** `<bullet list>`
- **What is noise and should be ignored?** `<bullet list>`

## 3) Current Usage (How It Is Used Today)

- **How OpenClaw currently interacts with it:** `<polling? manual checks? push notifications?>`
- **Current pain / token burn sources:** `<bullet list>`
- **What Spiderweb should take over:** `<bullet list>`

## 4) Authority & Trust

- **Default classification:** `information-only`
- **Command-capable traffic allowed?** `<no | yes (with constraints)>`
- **If yes, what validates command capability?**
  - **Sender identity:** `<who is allowed>`
  - **Verification proof:** `<signature | OAuth | token | other>`
  - **Allowed command types:** `<strict list>`
  - **Disallowed command types:** `<strict list>`

### Data Sensitivity

- **Contains PII?** `<yes | no | unknown>`
- **Contains secrets?** `<yes | no | unknown>`
- **Retention expectations:** `<days | weeks | none | unknown>`
- **Redaction rules before queueing:** `<what must be removed or hashed>`

## 5) Intake Mechanics (How Data Arrives)

- **Inbound mode:** `<webhook | polling | socket | email fetch | file watcher | other>`
- **Endpoints / URLs / Channels:** `<list>`
- **Auth required:** `<none | token | OAuth | key>`
- **Rate limits / quotas:** `<numbers>`
- **Message schema (fields we rely on):**
  - `id`: `<…>`
  - `timestamp`: `<…>`
  - `sender`: `<…>`
  - `thread / conversation id`: `<…>`
  - `text`: `<…>`
  - `attachments`: `<…>`

### Dedup Key (Must Be Defined)

- **Dedup key formula:** `<e.g. hash(text + attachments + event_id)>`
- **Is the upstream id stable?** `<yes | no | unknown>`
- **If upstream id is missing, fallback id strategy:** `<…>`

## 6) Valve / Queue Routing

- **Primary valve:** `<normal | interrupt>`
- **When interrupt is allowed:** `<never | only urgent | allow_interrupt flag | other>`
- **Target queue / destination:** `<which agent/session/channel>`
- **Required handshake:** `yes`

### Numeric Valve State Codes (Standard)

- `0` reject / unavailable
- `1` accept
- `2` busy / wait
- `3` interrupt accepted
- `4` stop / frozen
- `5` system error
- `6` unknown / unauthorized

## 7) Follow-Up Behavior (Cheap Relation Hints)

- **Follow-up enabled:** `<true | false>`
- **Working window size (max items):** `<default 5>`
- **Working window time (seconds):** `<default 180>`

### Numeric Follow-Up Suspicion Codes

- `0` none
- `1` low
- `2` medium
- `3` high

## 8) Checkpoints / State (Where We Resume)

- **Cursor / checkpoint type:** `<timestamp | id | offset | etag | none>`
- **Where stored:** `<file/db/memory>`
- **How to recover after restart:** `<steps>`
- **Known duplicate risks:** `<cases>`

### Backfill / Replay Policy

- **If Spiderweb was down, do we backfill?** `<yes | no>`
- **Backfill max lookback:** `<e.g. 24h>`
- **Backfill start marker:** `<cursor value or rule>`

## 9) Deduplication & Coalescing

- **Dedup window (seconds):** `<e.g. 60>`
- **Coalesce window (ms):** `<e.g. 750>`
- **Max batch messages:** `<e.g. 5>`
- **Max batch chars:** `<e.g. 4000>`

## 10) Failure Modes & Retry

- **Common failures:** `<401, timeouts, rate limits, payload too big, etc>`
- **Retry policy:** `<backoff rules>`
- **When to stop / freeze (state=4):** `<conditions>`
- **Operator alerts:** `<who/where>`

## 11) Usage Statistics (Fill In)

- **Messages per day:** `<x>`
- **Peak hours (local time):** `<HH:MM–HH:MM>`
- **Peak rate:** `<x messages/min>`
- **Typical burst size:** `<x messages>`
- **Average message length:** `<x chars>`
- **Attachments per day:** `<x>`
- **Noisy-to-useful ratio:** `<e.g. 90% noise>`
- **Typical latency from event → intake:** `<seconds>`

## 12) Setup / Secrets / Dependencies

- **Environment variables required:** `<list>`
- **Secrets required (do not paste values):** `<names only>`
- **External tooling required:** `<git-lfs, docker, cli tools, etc>`
- **Where configuration lives:** `<path>`

## 13) Collaboration Log (Must Be Used)

Everything discussed must be recorded. No drops.

- **Chat log directory:** `<workspace>/transfer-logs`
- **Chat id:** `<id>`
- **Chat token storage:** `<where token is stored>`
- **Log file path:** `<workspace>/transfer-logs/chat_<id>.md`

## 14) Transfer Protocol (Dry Handshake)

The sender must not consider transfer complete until confirm is successful.

- **Offer:** `POST /valve/handshake/offer`
- **Transfer:** `POST /valve/handshake/transfer`
- **Confirm:** `GET /valve/handshake/confirm?receipt_id=...`

Record the intended usage:

- **Handshake queue name:** `<normal | interrupt>`
- **Offer TTL expectation:** `<seconds>`
- **Max payload bytes:** `<limit>`

## 15) Minimal Transfer Package (What Must Be Sent During Handoff)

- `service_name`
- `intake_mode`
- `routing_target`
- `trust_level`
- `checkpoint/cursor`
- `rate/limits`
- `known_failure_modes`
- `pause/resume rules`

## 16) Go/No-Go Acceptance Criteria

- **Go conditions (must all be true):**
  - `[ ]` Routing target exists and is reachable.
  - `[ ]` Intake authentication is configured.
  - `[ ]` Checkpoint/cursor can be stored and restored.
  - `[ ]` Failure policy is defined (retry/freeze/alert).
  - `[ ]` Handshake path is confirmed usable.
- **No-Go conditions (any triggers block transfer):**
  - `[ ]` Trust level unknown.
  - `[ ]` Sender identity unknown for command-capable traffic.
  - `[ ]` No dedup key defined.
  - `[ ]` No checkpoint defined when required.

## 17) Notes

- `<anything else that helps Spiderweb say: “I know this station and can take over.”>`
