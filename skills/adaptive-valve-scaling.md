# skills.md — Spiderweb Employee Skill
## Skill: Adaptive Valve Scaling

---

## Purpose

This skill allows a **Spiderweb Employee** to participate in adaptive valve scaling during communication peaks and idle periods.

The goal is to preserve smooth intake flow without creating a permanent valve bloat problem.

This skill is for **Employee-side valve behavior within allowed operational scope**.  
It does **not** grant a Spiderweb Employee authority to redefine global valve policy.

---

## Main Goal

Allow a Spiderweb Employee to:

- request or activate additional valves during high communication peaks
- reduce pressure on active intake paths
- avoid one-path funnel behavior
- reduce valve count again after idle time
- downscale gradually with a drip effect
- never shrink below the protected minimum baseline

---

## Baseline Rule

The system should maintain:

- **minimum 2 active valves**
- **plus 1 spare capacity valve**

This means the protected floor is:

**2 + 1**

Downscaling must never reduce below that floor unless a higher authority explicitly changes policy.

---

## Role Boundary

A Spiderweb Employee may:

- observe communication pressure in its assigned pipeline
- request or activate additional valves within approved operational rules
- mark valves as idle candidates
- participate in controlled downscaling

A Spiderweb Employee may not:

- rewrite baseline valve policy
- reduce below the protected floor
- instantly collapse valve count aggressively
- alter system-wide scaling law
- override Spiderweb Supervisor policy

---

## When to Scale Up

A Spiderweb Employee should scale up or request additional valve capacity when communication pressure indicates that intake flow is becoming unhealthy.

Examples of scale-up signals may include:

- repeated queue growth in short time
- sustained intake bursts
- multiple pending handoffs
- active valve saturation
- rising wait/busy responses
- follow-up-heavy bursts causing clustering pressure

The exact trigger thresholds may be configurable, but the principle is:

**scale up early enough to preserve flow, not only after chaos starts**

---

## Scale-Up Behavior

During high communication peaks, the Spiderweb Employee should:

1. detect pressure rise
2. verify that current active valves are insufficient or nearing saturation
3. activate or request one additional valve at a time according to allowed scope
4. continue observing whether pressure remains elevated
5. repeat only as needed

Scale-up should be responsive, but not frantic.

Do not open valves simply because one message arrived.  
Do open valves when the intake pattern shows real pressure.

---

## When to Consider Downscaling

Downscaling should only be considered when extra valves have been idle for a defined period.

Examples of idle indicators:

- no meaningful handoff activity
- no queue pressure
- no backlog growth
- no peak conditions
- no persistent busy/wait state on active valves

The key idea is:

**do not shrink while the system is still warm**

---

## Downscaling Rule

Downscaling must use a **drip effect**.

This means:

- remove only **1 valve at a time**
- then wait through an idle timer
- then remove **1 more valve**
- repeat until the protected floor remains

Protected floor:

**2 + 1**

This prevents the system from collapsing too fast after a burst.

---

## Drip Effect Behavior

Correct downscale pattern:

1. system goes idle long enough to qualify
2. remove 1 excess valve
3. start idle timer again
4. if still idle after timer, remove 1 more excess valve
5. continue until only `2 + 1` remain

Incorrect behavior:

- dropping all extra valves at once
- shrinking while new pressure is forming
- shrinking below protected minimum
- treating a brief quiet moment as full idle state

---

## Idle Timer Principle

The idle timer must reset if meaningful activity returns.

That means:

- if traffic resumes, stop downscaling
- if pressure rises, switch back to scale-up logic
- if mild noise appears but not real pressure, continue observing carefully

The timer exists to prevent aggressive premature shrinkage.

---

## Priority of Actions

Valve logic should prefer:

1. preserve flow
2. avoid overload
3. avoid unnecessary valve sprawl
4. downscale slowly and safely
5. preserve protected minimum

If there is tension between:
- shrinking quickly
- keeping intake safe

choose:
- keeping intake safe

---

## Employee Authority Limits

A Spiderweb Employee may perform this skill only within approved operational scope.

Employee may:

- activate additional valve capacity within defined limits
- mark excess valves as idle candidates
- remove one excess valve after confirmed idle period
- repeat drip downscaling until `2 + 1` remain

Employee may not:

- change the minimum baseline
- change the +1 principle
- force global shutdown
- modify valve meaning/state codes
- override Spiderweb Supervisor decisions
- perform Tier 2 or Tier 3 changes through this skill

---

## Status Awareness

This skill should respect existing valve/status logic.

Relevant state reminders:

- `1` = accept
- `2` = busy / wait
- `3` = interrupt accepted
- `4` = stop / frozen
- `5` = system error

Busy/wait pressure should be treated as a potential scale-up signal.  
Frozen or error states must not be counted as healthy spare capacity.

---

## Safe Defaults

When unsure:

- do not over-shrink
- do not collapse multiple valves at once
- do not reduce below `2 + 1`
- prefer temporary excess over intake failure
- escalate abnormal scaling behavior upward if needed

---

## Escalation Conditions

The Spiderweb Employee should escalate to Spiderweb Supervisor when:

- valve pressure remains high despite scaling
- repeated scale-up still fails to stabilize flow
- a valve repeatedly errors
- protected floor appears insufficient
- scaling requests exceed Employee operational authority
- scaling behavior becomes unstable or contradictory

---

## Recommended Mental Model

Think of valve scaling like opening extra checkout lanes in a busy store.

When the rush starts:
- open more lanes

When the rush passes:
- do not close all lanes immediately
- close one
- wait
- close one
- wait
- stop at the protected baseline

That is the intended behavior.

---

## Success Condition

This skill is successful when:

- intake pressure during peaks is reduced
- communication bursts do not choke one intake path
- excess valves do not remain open forever without reason
- downscaling happens gradually
- the system settles back to `2 + 1`
- token-saving and flow stability are both preserved

---

## Final Rule

This skill exists to preserve harmony between:

- responsiveness during pressure
- restraint during calm

It should behave like a calm worker who says:

> Open more when needed.  
> Close slowly when safe.  
> Never choke the floor.