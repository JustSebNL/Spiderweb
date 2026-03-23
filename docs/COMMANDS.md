# Command Reference

This is the operator-facing command map for Spiderweb.

Root command:

```bash
sweb
```

Alias:

```bash
spiderweb
```

## Core Commands

### `sweb wakeup`
Run the Spiderweb launch sequence.

Alias:
- `sweb wake`

Use when:
- you want to trigger the launch flow after install

### `sweb status`
Show Spiderweb status.

Alias:
- `sweb s`

### `sweb version`
Show version information.

Alias:
- `sweb v`

### `sweb gateway`
Start the gateway runtime.

Alias:
- `sweb g`

Useful form:

```bash
sweb gateway --debug
```

### `sweb onboard`
Run onboarding flow.

Alias:
- `sweb o`

### `sweb agent`
Run agent mode.

### `sweb migrate`
Run migration-related commands.

## Auth Commands

```bash
sweb auth login
sweb auth logout
sweb auth status
sweb auth models
```

Use auth commands when:
- a provider requires stored credentials
- a provider error tells you to run `sweb auth login --provider <name>`

## Cron Commands

```bash
sweb cron list
sweb cron add
sweb cron remove
sweb cron enable
sweb cron disable
```

Alias:
- `sweb c`

Use cron commands to manage scheduled local tasks stored in the workspace cron state.

## Skills Commands

```bash
sweb skills list
sweb skills search
sweb skills show <name>
sweb skills install <name-or-source>
sweb skills install-builtin
sweb skills list-builtin
sweb skills remove <name>
```

Aliases:
- `remove`: `rm`, `uninstall`

Use skills commands to inspect, install, or remove reusable skill packages.

## Transfer Commands

```bash
sweb transfer path
sweb transfer list
sweb transfer init <service-name>
sweb transfer kickoff <service-name>
sweb transfer stats
```

Chat subcommands:

```bash
sweb transfer chat start <service-name>
sweb transfer chat send <chat-id>
sweb transfer chat tail <chat-id>
```

Use transfer commands to manage service handoff documents and transfer coordination.

## OpenClaw Commands

```bash
sweb openclaw status
sweb openclaw connect --name openclaw
sweb openclaw transfer
sweb openclaw transfer --service inbox
sweb openclaw transfer --dry-run
```

Use OpenClaw commands when Spiderweb is acting as the intake/bridge layer.

Practical bridge path:
- enable `channels.openclaw` in config
- start `sweb gateway`
- run `sweb openclaw status`
- use `sweb openclaw connect` for a direct bridge test
- use `sweb openclaw transfer` for the full handoff sequence

## Practical Sequences

### Install then launch

```bash
./bootstrap.sh
sweb wakeup
```

### Start gateway and inspect health

```bash
sweb gateway
```

Then check local endpoints:
- `http://127.0.0.1:13370/health`
- `http://127.0.0.1:13370/ready`
- `http://127.0.0.1:13370/valve/state`

### Recover bootstrap session

```bash
./bootstrap.sh --show-state
./bootstrap.sh
./bootstrap.sh --reset-state
```

## Notes

- This document catalogs the command surface, not every individual flag.
- For live flag details on a specific command, use:

```bash
sweb <command> --help
```
