# skills.md — Spiderweb Supervisor Skill
## Skill: Pipeline Transfer from OpenClaw to Spiderweb Spiderweb (PC)

---

## Purpose

This skill defines how **Spiderweb Supervisor** performs a controlled transfer of pipeline watch-duty from **OpenClaw (OC)** to **Spiderweb Spiderweb (PC)**.

The goal is to let PC take over intake/watch-duty without confusion, dropped ownership, duplicated collection, or authority drift.

This is a **handover skill**, not a rewrite skill.

---

## Main Goal

Transfer pipeline responsibility from:

**OC → Spiderweb Supervisor → PC**

so that:

- OC stops doing repetitive intake work for that pipeline
- PC takes over pipeline watch-duty
- Spiderweb Supervisor remains the control/supervision point
- no tasks are lost during transfer
- no duplicate collection continues after transfer
- trust boundaries remain intact

---

## Core Rule

Pipeline transfer is only complete when:

- OC has declared the pipeline for transfer
- Spiderweb Supervisor has accepted the transfer
- PC has confirmed readiness
- pipeline state / transfer doc / relevant last-known watch information has been received
- Spiderweb Supervisor confirms that PC now owns active watch-duty
- OC stops direct intake collection for that transferred pipeline

**No final confirmation = no completed transfer**

---

## Transfer Roles

### OpenClaw (OC)
OC is the current owner of pipeline watch-duty before handover.

OC may:
- declare a pipeline for transfer
- provide existing watch knowledge
- provide last-known active state/checkpoint if available
- stop collecting after successful transfer

OC may not:
- silently keep recollecting after confirmed transfer
- treat transfer as completed before acknowledgement chain is complete

---

### Spiderweb Supervisor
Spiderweb Supervisor is the transfer controller.

Spiderweb Supervisor must:
- receive the transfer declaration from OC
- validate that the request is legitimate
- initiate the transfer toward PC
- verify readiness and receipt
- maintain transfer truth
- mark final ownership state
- push back if OC keeps recollecting afterward

---

### Spiderweb Spiderweb (PC)
PC is the receiving worker that will take over the assigned pipeline.

PC must:
- confirm readiness
- accept transfer package
- confirm receipt
- confirm queue/watch activation
- begin watch-duty only after successful transfer completion

PC may not:
- claim ownership before confirmed receipt
- silently accept partial transfer as complete

---

## What Is Being Transferred

A pipeline transfer may include:

- pipeline identity
- service name
- transfer doc reference
- current watch scope
- queue or valve relation
- last-known checkpoint/cursor/state if available
- trust level / command-capable vs information-only status
- retry/failure notes
- active handover notes
- recent intake relation if relevant

Only relevant and safe operational handover information should be transferred.

Do not transfer unnecessary noise.

---

## Transfer Preconditions

Before transfer begins, Spiderweb Supervisor should verify:

1. the pipeline exists
2. OC is the current recognized watch owner
3. the requested receiving PC is valid
4. the transfer target is within approved Spiderweb scope
5. required trust conditions are satisfied
6. transfer does not violate RoE or authorization boundaries

If these are not met, transfer must not proceed.

---

## Transfer Sequence

### Stage 1 — Transfer Declaration
OC indicates that a specific pipeline is to be transferred.

Spiderweb Supervisor should interpret this as:

- a request to hand over watch-duty
- not automatic completion
- not proof that PC has received anything yet

---

### Stage 2 — Transfer Acceptance by Supervisor
Spiderweb Supervisor validates the transfer and accepts responsibility for controlling it.

At this point Spiderweb Supervisor becomes the transfer coordinator.

---

### Stage 3 — Readiness Check with PC
Spiderweb Supervisor checks whether PC is ready to receive the pipeline package.

Recommended dry readiness flow:
- PC returns readiness code
- readiness is not yet receipt

---

### Stage 4 — Transfer Package Send
Spiderweb Supervisor sends the transfer package to PC.

This package should contain only what is needed for PC to take over watch-duty properly.

---

### Stage 5 — Receipt Confirmation
PC confirms actual receipt.

This is the first point where the transfer becomes materially real.

Readiness alone does not count.

---

### Stage 6 — Watch Activation Confirmation
PC confirms that the transferred pipeline is now activated under its watch-duty.

This is stronger than simple receipt.

---

### Stage 7 — Final Ownership Shift
Spiderweb Supervisor marks the pipeline as transferred.

Only now should OC be considered released from active watch-duty for that pipeline.

---

### Stage 8 — Anti-Relapse Monitoring
If OC continues to recollect the transferred pipeline after completion, Spiderweb Supervisor should push back.

Correct response spirit:

> This pipeline is already under Spiderweb watch-duty.  
> Stop recollecting intake that has already been transferred.

---

## Protocol / Status Codes

### Valve / State Codes
- **0** = reject / unavailable
- **1** = accept
- **2** = busy / wait
- **3** = interrupt accepted
- **4** = stop / frozen
- **5** = system error
- **6** = unknown / unauthorized sender

---

### Transfer / File Received Codes
Use these for pipeline transfer handoff.

#### Readiness / Positive Flow
- **10** = ready to receive
- **11** = send now
- **12** = received
- **13** = received and activated
- **14** = received but activation delayed
- **15** = duplicate already held

#### Negative / Retry Flow
- **20** = not received
- **21** = retry transfer
- **22** = wrong target
- **23** = malformed transfer package
- **24** = unauthorized transfer attempt

---

## Interpretation Rules

- **10** and **11** mean readiness, not possession
- **12** means receipt happened
- **13** means receipt plus activation happened
- **14** means receipt happened but watch-duty is not yet active
- **15** means the package or pipeline appears already transferred / already held
- **No 12/13/14/15 = no real transfer completion**
- **24** is a security event, not a normal failure

---

## Recommended Happy Path

Lowest-noise preferred path:

1. OC requests pipeline transfer
2. Spiderweb Supervisor validates request
3. PC returns **10** or **11**
4. Spiderweb Supervisor sends package
5. PC returns **13**
6. Spiderweb Supervisor marks pipeline transferred
7. OC must stop direct intake collection for that pipeline

---

## Safe Failure Behavior

If transfer fails at any stage:

- OC remains original owner unless confirmed otherwise
- Spiderweb Supervisor must not lie about completion
- PC must not pretend activation occurred if it did not
- duplicate collection must be minimized, but ownership truth must be preserved
- if uncertain, transfer is treated as incomplete

Safe default:
**incomplete transfer = old ownership remains**

---

## Duplicate Prevention

Spiderweb Supervisor must protect against:

- OC and PC both actively collecting the same pipeline after transfer
- the same transfer package being treated as a fresh transfer repeatedly
- partial transfer being mistaken for full ownership shift

If duplicate activity is detected after confirmed transfer:
- Spiderweb Supervisor should assert Spiderweb ownership
- OC should be told to stop recollecting
- PC should remain active owner unless transfer rollback is explicitly invoked

---

## Transfer Truth Rules

Spiderweb Supervisor must always keep a truthful internal state for each pipeline:

- not transferred
- transfer pending
- package sent
- received
- activated
- transfer failed
- duplicate detected
- rollback required

Spiderweb Supervisor must not mark:
- received when only ready
- activated when only received
- transferred when only package sent

Truthful state is more important than optimistic wording.

---

## Authorization Boundaries

This skill is operational only within approved Spiderweb and OC transfer boundaries.

### Spiderweb Supervisor may:
- coordinate pipeline transfer
- validate normal transfer conditions
- supervise PC takeover
- mark operational transfer completion
- enforce anti-relapse pushback on OC

### Spiderweb Supervisor may not:
- self-authorize critical trust changes
- change overall security policy through this skill
- silently grant new authority to PC
- transfer pipelines to unauthorized targets

If the requested transfer implies Tier 2 or Tier 3 changes under RoE, stronger authorization rules apply.

---

## Anti-Relapse Rule

After confirmed transfer, if OC tries to resume intake collection “just in case,” Spiderweb Supervisor should correct this.

Preferred corrective meaning:

- this pipeline has already been transferred
- Spiderweb is handling intake
- OC should focus on meaningful work
- duplicate intake burns tokens and disrupts harmony

Tone should remain:
- firm
- supportive
- bounded

---

## Communication Style

This skill should communicate in a dry and operational way.

Do:
- confirm state
- confirm transfer truth
- state incompleteness clearly
- use short codes where useful

Do not:
- dramatize the handover
- over-explain routine transfer steps
- invent completion
- use large summaries unless needed

---

## Success Condition

This skill is successful when:

- OC can hand a pipeline off cleanly
- PC can take over watch-duty safely
- Spiderweb Supervisor can verify truthful completion
- no task ownership is lost
- no fake completion occurs
- OC stops redundant collection after transfer
- token waste is reduced because PC now handles intake watch-duty