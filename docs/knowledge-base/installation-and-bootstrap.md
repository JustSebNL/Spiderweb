# Installation And Bootstrap

This guide explains the supported install path and the main files Spiderweb writes during setup.

## Supported Install Path

User contract:
- install: `./bootstrap.sh`
- start: `sweb wakeup`

The root bootstrap script is the only user-facing install command.

## Basic Install

From the repo root:

```bash
chmod +x bootstrap.sh
./bootstrap.sh
```

If you are on Windows, run the bootstrap path inside WSL.

## Useful Bootstrap Controls

```bash
./bootstrap.sh --dry-run
./bootstrap.sh --show-state
./bootstrap.sh --reset-state
```

## What Bootstrap Does

Bootstrap prepares the machine from front to back:
- installs prerequisites
- installs Spiderweb core
- prepares `brain/`
- prepares a persistent Hugging Face cache under `~/.spiderweb/hf`
- selects the cheap-cognition runtime
- downloads the matching model format
- writes runtime settings to `~/.spiderweb/runtime.env`
- writes setup notes to `~/.spiderweb/setup-notes.txt`
- saves resumable install state to `~/.spiderweb/bootstrap-state.env`

## Important Files

Local runtime/setup files:
- `~/.spiderweb/runtime.env`
- `~/.spiderweb/setup-notes.txt`
- `~/.spiderweb/bootstrap-state.env`
- `~/.spiderweb/hf/`

Repo-side install files:
- `bootstrap.sh`
- `bootstrap.conf`
- `install_spiderweb.conf`

## Important Settings

Bootstrap settings live in `bootstrap.conf`.

Common settings:
- `SPIDERWEB_DIR`
- `INSTALL_PREFIX`
- `BRAIN_DIR`
- `HF_HOME_DIR`
- `HF_HUB_CACHE_DIR`
- `CHEAP_COGNITION_RUNTIME`
- `AUTO_START_MODEL_SERVICE`

Example:

```bash
SPIDERWEB_DIR="/opt/spiderweb"
INSTALL_PREFIX="/opt/spiderweb/.local"
BRAIN_DIR="/opt/spiderweb/brain"
HF_HOME_DIR="$HOME/.spiderweb/hf"
HF_HUB_CACHE_DIR="$HF_HOME_DIR/hub"
CHEAP_COGNITION_RUNTIME="auto"
AUTO_START_MODEL_SERVICE="1"
```

## After Install

Check the generated notes and env:

```bash
cat ~/.spiderweb/setup-notes.txt
cat ~/.spiderweb/runtime.env
```

Then start Spiderweb:

```bash
sweb wakeup
```

The generated runtime env now also carries:
- `HF_HOME`
- `HF_HUB_CACHE`

That keeps Hugging Face downloads and warm restarts pointed at a stable cache directory.

## Related Docs
- [Startup And Daily Use](./startup-and-daily-use.md)
- [../QUICKSTART.md](../QUICKSTART.md)
- [../cookbook/bootstrap-installer.md](../cookbook/bootstrap-installer.md)
