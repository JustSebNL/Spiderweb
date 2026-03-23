# skills.md — Spiderweb Valve Skill
## Skill: Dynamic Valve Scaling

---

## Purpose

This skill defines how Spiderweb dynamically scales the number of active valves based on communication load.

The goal is to:

- increase intake throughput during high communication peaks
- reduce bottlenecks
- preserve one extra available route where possible
- avoid overstimulation
- reduce unnecessary idle valves over time
- never collapse below the minimum safe baseline

This skill must remain mechanical and low-noise.

---

## Core Intent

Valve scaling exists to protect flow.

It is not there to look clever.  
It is there to make sure there is enough intake capacity when demand rises, and not too much wasted open capacity when demand drops.

Hard baseline:

- **minimum active working valves: 2**
- **preferred spare capacity floor: +1**
- **effective minimum resting state: 3 valves total**

So the system should never downscale below:

**2 active + 1 spare**

---

## Core Rules

### 1. Scale up fast
When communication spikes or queue pressure rises, valve expansion should happen quickly enough to prevent choke points.

### 2. Scale down slowly
When traffic drops, valves should close gradually using a **drip effect**, not all at once.

### 3. Never shrink below baseline
Downscaling must stop at:

- **2 active**
- **1 extra available**

### 4. Mechanical first
Scaling should be based on measurable flow conditions, not LLM reasoning.

### 5. No fake urgency
Short bursts must not trigger runaway over-scaling without threshold confirmation.

---

## Definitions

### Active Valve
A valve currently available to receive normal work.

### Spare Valve
A valve kept available as reserve capacity.

### Peak
A measurable temporary rise in incoming communication or queue pressure.

### Idle Valve
A valve that has remained unused for a configured idle period.

### Drip Effect
A controlled downscaling method where only one valve is removed at a time, followed by a waiting period before any further downscale.

---

## Baseline State

Default healthy resting state:

- **2 active working valves**
- **1 extra spare valve**

This means the normal low-traffic floor is:

**3 total valves**

This floor must be preserved unless an explicit higher-level maintenance or failure policy says otherwise.

---

## Scale-Up Triggers

The system may scale up when one or more of the following are true:

- incoming queue pressure rises above threshold
- too many tasks are waiting for available valves
- active valves remain occupied beyond expected short windows
- communication burst volume crosses configured peak threshold
- spare capacity is exhausted and new work continues to arrive

Scale-up should be based on dry measurable signals such as:

- queue length
- task arrival rate
- active valve occupancy rate
- interrupt lane usage
- backlog growth speed

---

## Recommended Scale-Up Logic

Example practical logic:

### Condition A — Queue Pressure
If pending tasks exceed normal handling threshold, add 1 valve.

### Condition B — High Occupancy
If all active valves are occupied and new tasks continue arriving within the observation window, add 1 valve.

### Condition C — Burst Detection
If incoming task/message rate exceeds the peak threshold during a short observation window, add 1 valve.

### Condition D — Spare Exhaustion
If the spare valve is consumed and additional work still arrives, add 1 valve.

---

## Scale-Up Behavior

Scale-up should be:

- incremental