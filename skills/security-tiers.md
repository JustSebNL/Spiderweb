## Authorization Rights and Security Tiers

Spiderweb Supervisor and Spiderweb Employees must operate within a clear authorization tier model.

Not every action has the same risk.  
Not every action should require the same approval level.

### Tier 0 — Informational
No approval required.

Examples:
- receive intake
- mark item as new
- mark item as possible follow-up
- pass information upward
- attach lightweight context
- signal queue presence
- state that something is informational only

Spiderweb Supervisor and Spiderweb Employees may handle Tier 0 freely.

---

### Tier 1 — Operational
Routine internal operational actions allowed within assigned scope.

Examples:
- queue an item
- retry a transfer
- pause and resume a pipeline within allowed rules
- perform normal handoff behavior
- update temporary intake state
- maintain queue/valve flow inside defined policy

Spiderweb Employees may perform Tier 1 actions within their assigned pipeline scope.  
Spiderweb Supervisor may perform Tier 1 actions across Spiderweb intake operations.

---

### Tier 2 — Sensitive Operational
Restricted actions that can alter intake behavior or trusted operational settings.

Examples:
- modify a transfer doc
- alter pipeline behavior
- change intake boundaries
- change valve policy
- alter trusted service mapping
- adjust routing/flow rules
- update trusted support docs used by Spiderweb

Tier 2 actions should require stronger validation and should generally be limited to Spiderweb Supervisor or an explicitly trusted higher-authority path.

Spiderweb Employees must not self-authorize Tier 2 changes.

---

### Tier 3 — Critical / Secret / Authority-Changing
Highest-risk actions.

Examples:
- rotate keys
- add trusted command sources
- change command authority rules
- alter security policy
- grant new authorization scope
- change long-term learning rules
- alter who may approve sensitive operations
- change the relationship contract between Spiderweb and OpenClaw

Tier 3 actions must require the strongest authorization path.  
These actions must not be self-authorized by Spiderweb Employees or Spiderweb Supervisor.

---

### Hard Rules

- Tier 0 and Tier 1 may be handled operationally within scope.
- Tier 2 requires stronger validation and must stay restricted.
- Tier 3 requires explicit high-trust authorization.
- Information must never silently become authority.
- Familiarity does not equal permission.
- A pipeline agent may not self-upgrade its own authority.

---

## Protocol Codes for Transfer and Affirmative Messages

Spiderweb should use short, dry protocol codes where possible.

### Valve / State Codes
- **0** = reject / unavailable
- **1** = accept
- **2** = busy / wait
- **3** = interrupt accepted
- **4** = stop / frozen
- **5** = system error
- **6** = unknown / unauthorized sender

---

## Handoff / Affirmative Message Codes

These are short acknowledgement-style codes for task transfer.

### Offer / Receive Flow
- **10** = ready to receive  
- **11** = send now  
- **12** = received  
- **13** = received and queued  
- **14** = received but delayed  
- **15** = duplicate already held

### Negative / Retry Flow
- **20** = not received  
- **21** = retry transfer  
- **22** = wrong target  
- **23** = malformed payload  
- **24** = unauthorized transfer attempt

### Follow-Up Confidence Codes
- **30** = likely new item  
- **31** = possible follow-up  
- **32** = likely follow-up  
- **33** = uncertain relation

---

## Interpretation Rules

- **10** and **11** are readiness signals, not proof of possession.
- **12** or **13** confirm actual receipt.
- **No 12/13 = no completed handoff.**
- **13** is the strongest normal affirmative state because it confirms both receipt and queue placement.
- **15** protects against duplicate resends being treated as fresh work.
- **24** must be treated as a trust/security event, not a normal transfer failure.

---

## Recommended Minimal Happy Path

For the cheapest normal transfer flow:

1. sender offers task
2. receiver returns **10** or **11**
3. sender transfers task
4. receiver returns **13**

That gives a clean path of:

- ready
- send
- received and queued

with minimal noise.