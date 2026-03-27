# Quick Start

This guide is the shortest path to get Spiderweb installed and moving toward a working launch.

## User Contract
- install: `./bootstrap.sh`
- start: `sweb wakeup`

Everything else is implementation detail behind those two commands.

## 1. Install

From the repo root:

```bash
chmod +x bootstrap.sh
./bootstrap.sh
```

Useful bootstrap controls:

```bash
./bootstrap.sh --dry-run
./bootstrap.sh --show-state
./bootstrap.sh --reset-state
```

What bootstrap does:
- installs prerequisites
- installs `git-lfs`
- installs Spiderweb core
- prepares `brain/`
- chooses `vllm` or `llama.cpp`
- downloads the matching Youtu model
- writes runtime settings to `~/.spiderweb/runtime.env`
- writes setup guidance to `~/.spiderweb/setup-notes.txt`
- saves resumable install state to `~/.spiderweb/bootstrap-state.env`

## 2. Read Follow-Up Notes

After bootstrap, check:

- `~/.spiderweb/runtime.env`
- `~/.spiderweb/setup-notes.txt`

`setup-notes.txt` tells you what still needs manual configuration, especially main provider API keys.

## 3. Start Spiderweb

```bash
sweb wakeup
```

If `sweb` is not on your `PATH` yet:

```bash
$HOME/.local/bin/sweb wakeup
```

## 4. Useful Runtime Commands

```bash
sweb status
sweb gateway
sweb gateway --debug
sweb version
```

## 5. Bridge And Transfer Commands

```bash
sweb openclaw status
sweb openclaw connect
sweb openclaw transfer
```

```bash
sweb transfer path
sweb transfer list
sweb transfer init <service-name>
sweb transfer kickoff <service-name>
sweb transfer stats
```

## 6. Auth And Model Setup

If your provider requires auth:

```bash
sweb auth status
sweb auth models
sweb auth login --provider <name>
```

Common providers still require API keys or credentials after bootstrap.
That is expected: bootstrap prepares the system, but it does not invent provider access for the user.

## 7. Health And Pipeline Checks

When `sweb gateway` is running, Spiderweb exposes local health and valve endpoints:

- `/health`
- `/ready`
- `/valve/state`
- `/valve/offer`

Observer dashboard UI when gateway health server is running:

- `/observer/ui`

OpenClaw bridge endpoint when enabled:

- `/bridge/openclaw`

Runtime maintenance snapshots are written to:

- `~/.spiderweb/runtime-health.json`
- `~/.spiderweb/runtime-health.json.baseline`

## 8. If Bootstrap Was Interrupted

Rerun:

```bash
./bootstrap.sh
```

Bootstrap will resume from saved state and skip completed chapters.

If you want a full reset:

```bash
./bootstrap.sh --reset-state
```
