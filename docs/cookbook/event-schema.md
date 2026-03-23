# Event Schema Patterns

Spiderweb should normalize source events before routing or model usage.

## Minimal normalized event shape

```json
{
  "event_id": "evt_01",
  "source": "slack",
  "source_object": "channel:C123",
  "event_type": "message.created",
  "occurred_at": "2026-03-15T10:21:00Z",
  "actor": "user_abc",
  "importance_hint": "unknown",
  "payload": {
    "text": "Can someone review the deploy script?"
  },
  "dedupe_key": "slack:C123:1742.22",
  "version": 1
}
```

## Fields that matter most
- `event_id`: unique internal id
- `source`: originating system
- `event_type`: stable internal taxonomy
- `occurred_at`: source timestamp
- `payload`: compact relevant fields only
- `dedupe_key`: collapse repeated observations safely
- `version`: schema version for migrations later

## Rule
Journal raw source truth first, then derive normalized events from it.
