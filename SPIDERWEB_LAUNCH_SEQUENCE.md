# Spiderweb Launch Sequence (Storyboard)

This is the “wake the interface up” ritual: you start it once, you are warned you can’t go back, and if the sequence is interrupted it restarts on the next launch.

## Chapter 0 — Point of No Return

**Warning:** once the launch sequence begins, you do not “step backward”.  
If the process stops mid-way (crash/kill/close), the next launch restarts from Chapter 1.

Goal: make setup and transfer coordination deterministic and auditable.

## Chapter 1 — Wake the Interface

Say (out loud or in chat):

- “Spiderweb, wake up.”
- “Dude….. no worries! I’ve got this 😏”

Conversation starters:

- “What pipeline are we transferring today?”
- “What counts as signal vs noise in this pipeline?”
- “What is the minimum safe checkpoint/cursor?”

## Chapter 2 — Declare the Transfer Target

Write down:

- service/pipeline name (must be unique)
- who owns watch duty before and after transfer
- which queue the intake should go to by default (`normal` vs `interrupt`)

Conversation starters:

- “What breaks if we drop one message?”
- “What breaks if we duplicate one message?”
- “What’s the ‘stop/freeze’ condition?”

## Chapter 3 — Start the Collaboration Room (Nothing Dropped)

Create an invite-only transfer chat room.

Rules:

- everything discussed is written to `transfer-logs`
- no messages are “summarized away” during transfer coordination
- the transcript is the record

Conversation starters:

- “List required auth/secrets by name (no values).”
- “List failure modes: 401, 429, timeouts, payload limits.”

## Chapter 4 — Fill the Transfer Sheet (Impossible to Omit)

Create/complete a transfer sheet for the pipeline.

Non-negotiables:

- identity keys (service key, conversation key, message id)
- intake mechanism (push/pull), endpoints, auth
- dedup key formula
- checkpoint/cursor strategy
- retry/freeze/alert rules
- usage stats (even rough): msgs/day, peak hours, bursts

Conversation starters:

- “What is the dedup key formula?”
- “What is the resume cursor and where is it stored?”

## Chapter 5 — Verify the Valve Handshake (Dry Numeric)

Establish: offer → transfer → confirm.

The sender does not consider transfer complete until confirm is successful.

Conversation starters:

- “Which queue is allowed to interrupt, and when?”
- “What is the maximum payload size we accept?”

## Chapter 6 — Follow-Up Recognition (Cheap Relation Hint)

We do not try to be clever. We only mark suspicion in a small window.

Conversation starters:

- “Do we treat ‘Also/BTW/One more thing’ as follow-up?”
- “What is the window size/time?”

## Chapter 7 — Live Run & Audit

Confirm:

- the chat transcript is growing under `transfer-logs`
- intake stats match expectations (rate/peak)
- no unexpected command authority leaks in

Conversation starters:

- “Is anything arriving that should be filtered as noise?”
- “Are we seeing bursts at the expected hours?”

## Chapter 8 — Transfer Complete (Teardown)

When transfer is complete:

- declare completion in the chat transcript
- stop/nuke the temporary collaboration engine (E2B container)
- keep `transfer-logs` as the permanent audit record

