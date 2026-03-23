# Startup And Daily Use

This guide covers the main commands a user is expected to run after installation.

## Standard Start Path

```bash
sweb wakeup
```

If `sweb` is not on your `PATH` yet:

```bash
$HOME/.local/bin/sweb wakeup
```

## Useful Runtime Commands

```bash
sweb status
sweb gateway
sweb gateway --debug
sweb version
```

## Auth Commands

If your provider requires credentials or OAuth:

```bash
sweb auth status
sweb auth models
sweb auth login --provider <name>
```

## Transfer Commands

```bash
sweb transfer path
sweb transfer list
sweb transfer init <service-name>
sweb transfer kickoff <service-name>
sweb transfer stats
```

## OpenClaw Commands

```bash
sweb openclaw status
sweb openclaw connect
sweb openclaw transfer
```

## Health And Runtime Endpoints

When `sweb gateway` is running, Spiderweb exposes:
- `/health`
- `/ready`
- `/valve/state`
- `/valve/offer`

OpenClaw bridge endpoint when enabled:
- `/bridge/openclaw`

Example local path:

```text
http://127.0.0.1:13370/health
```

## Current Runtime Files

Useful runtime files for daily inspection:
- `~/.spiderweb/runtime.env`
- `~/.spiderweb/runtime-health.json`
- `~/.spiderweb/runtime-health.json.baseline`

## Debug Use

Use:

```bash
sweb gateway --debug
```

This is the current gateway-side debug path.

Current observer reads are available through:

```bash
curl http://127.0.0.1:8080/observer/overview
curl http://127.0.0.1:8080/observer/services
```

Longer-term observer debug mode is part of the control-plane design, but the full dashboard/debug surface is not fully implemented yet.

## Related Docs
- [Observer And Self-Care](./observer-and-self-care.md)
- [../COMMANDS.md](../COMMANDS.md)
- [../QUICKSTART.md](../QUICKSTART.md)
