# Troubleshooting

This guide is the first-stop checklist when Spiderweb is installed but not behaving correctly.

## 1. Bootstrap Problems

If install did not finish cleanly:

```bash
./bootstrap.sh --show-state
./bootstrap.sh
```

If you need to fully reset the bootstrap progress:

```bash
./bootstrap.sh --reset-state
```

Inspect:
- `~/.spiderweb/bootstrap-state.env`
- `~/.spiderweb/setup-notes.txt`

## 2. Binary Or PATH Problems

If `sweb` is not found:

```bash
$HOME/.local/bin/sweb version
```

If that works, the binary exists but your shell `PATH` is not updated yet.

## 3. Gateway Not Reachable

Start the gateway:

```bash
sweb gateway
```

Then check:

```text
http://127.0.0.1:13370/health
http://127.0.0.1:13370/ready
```

If those are unreachable:
- confirm the gateway process is actually running
- confirm the configured host and port
- check the terminal output from `sweb gateway`

## 4. Provider Auth Problems

Inspect provider auth state:

```bash
sweb auth status
sweb auth models
```

If needed:

```bash
sweb auth login --provider <name>
```

Also review:
- `~/.spiderweb/runtime.env`
- `~/.spiderweb/setup-notes.txt`

## 5. Self-Care Or Runtime Health Problems

Inspect:

```bash
cat ~/.spiderweb/runtime-health.json
cat ~/.spiderweb/runtime-health.json.baseline
```

Look for:
- low score
- repeated degraded summaries
- repeated remediation attempts
- stale pid and log-growth recommendations

## 6. Cheap-Cognition Runtime Problems

Check whether the selected runtime and brain path make sense in:

```bash
cat ~/.spiderweb/runtime.env
```

Things to verify:
- `BRAIN_DIR`
- `HF_HOME`
- `HF_HUB_CACHE`
- chosen runtime
- model/runtime path assumptions

If model downloads keep repeating or warm restarts do not reuse cached assets:
- confirm `HF_HOME` and `HF_HUB_CACHE` are present in `~/.spiderweb/runtime.env`
- confirm the cache directories still exist and are writable

## 7. Observer Questions

Current truth:
- native maintenance/self-care exists now
- read-only observer JSON endpoints exist now
- the interactive observer dashboard exists now
- report generation and journal generation exist now
- some panels still fall back gracefully when live observer data is unavailable

So if you are looking for observer state today, the primary inspection surfaces are:
- health endpoints
- observer endpoints
- the interactive observer dashboard
- runtime logs
- self-care snapshot files

Useful calls:

```bash
curl http://127.0.0.1:8080/observer/overview
curl http://127.0.0.1:8080/observer/dashboard
curl http://127.0.0.1:8080/observer/benchmarks
curl http://127.0.0.1:8080/observer/services
curl http://127.0.0.1:8080/observer/agents
curl http://127.0.0.1:8080/observer/events
curl http://127.0.0.1:8080/observer/self-care/cycles
curl http://127.0.0.1:8080/observer/stats/24h
curl http://127.0.0.1:8080/observer/reports/latest
curl http://127.0.0.1:8080/observer/journal/latest
```

## Related Docs
- [Installation And Bootstrap](./installation-and-bootstrap.md)
- [Startup And Daily Use](./startup-and-daily-use.md)
- [Dashboard And Operator Usage](./dashboard-and-operator-usage.md)
- [Brain Runtime Operations](./brain-runtime-operations.md)
- [Observer And Self-Care](./observer-and-self-care.md)
- [../cookbook/failure-handling.md](../cookbook/failure-handling.md)
