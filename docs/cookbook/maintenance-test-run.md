# Maintenance Test Run

Use this when you want an IDE-side runner to execute the maintenance tests while another session keeps building.

## Focused Test Command

```bash
go test -v ./pkg/maintenance -run TestMaintenance
```

## Verbose Log Capture

```bash
go test -v ./pkg/maintenance -run TestMaintenance 2>&1 | tee maintenance-test.log
```

## What These Tests Cover
- busy-window defer logic
- restart backoff logic
- stale pid cleanup
- oversized log trimming
- snapshot behavior when maintenance should yield instead of restarting

## What To Share Back
Useful things to paste back into the main session:
- the first compile error, if any
- any failing test names
- the `t.Logf(...)` lines from the failing test
- the final `PASS` or `FAIL` summary
