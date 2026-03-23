# OpenClaw Bridge

This guide explains how to use Spiderweb as the intake and bridge layer in front of OpenClaw.

## Purpose

Spiderweb can sit in front of OpenClaw as the lower-cost intake and watch-duty layer.

Flow:

```text
Services / Pipelines -> Spiderweb -> Queue / Valve -> OpenClaw
```

Spiderweb handles:
- intake
- filtering
- deduplication
- lightweight cheap-cognition triage
- transfer coordination

OpenClaw stays focused on higher-value reasoning.

## Config

Enable the OpenClaw bridge in your config:

```json
{
  "channels": {
    "openclaw": {
      "enabled": true,
      "shared_secret": "",
      "allow_from": [],
      "auto_handshake": true,
      "intake_enabled": true,
      "webhook_path": "/bridge/openclaw"
    }
  }
}
```

Fields:
- `enabled`: turns the bridge on
- `shared_secret`: optional handshake secret
- `allow_from`: optional sender allowlist
- `auto_handshake`: send transfer introduction automatically on connect
- `intake_enabled`: advertise Spiderweb as intake layer
- `webhook_path`: bridge path, defaults to `/bridge/openclaw`

## Start The Gateway

The bridge is served by the gateway process.

```bash
sweb gateway
```

Relevant local endpoints:
- `/health`
- `/ready`
- `/valve/state`
- `/bridge/openclaw`

## Check Bridge Status

```bash
sweb openclaw status
```

This reports:
- whether the bridge is enabled
- gateway host and port
- WebSocket path
- auto-handshake setting
- intake-enabled setting
- current gateway health status

## Test A Bridge Connection

```bash
sweb openclaw connect --name openclaw
```

This command:
- connects to Spiderweb as a WebSocket peer
- sends the handshake payload
- prints the handshake acknowledgement
- lets you type test messages interactively

Use it when you want to validate the bridge before doing a full intake handoff.

## Run The Transfer Sequence

```bash
sweb openclaw transfer
```

Or target a specific service scope:

```bash
sweb openclaw transfer --service inbox
```

Dry-run mode:

```bash
sweb openclaw transfer --dry-run
```

The transfer command:
1. checks gateway health
2. checks valve state
3. opens a transfer chat
4. sends Spiderweb's intake introduction
5. prints the handoff summary

## Expected Handoff Shape

After the transfer sequence, the intended operating model is:
- Spiderweb receives service or pipeline traffic first
- Spiderweb filters and triages intake
- OpenClaw receives the higher-value work

This is not meant to replace OpenClaw.
It is meant to reduce repetitive noise and token burn before work reaches OpenClaw.

## Troubleshooting

If `sweb openclaw status` says the bridge is disabled:
- enable `channels.openclaw.enabled` in config

If `sweb openclaw status` cannot reach the gateway:
- start `sweb gateway`
- verify host and port in config

If `sweb openclaw transfer` fails on the health check:
- make sure the gateway is running before starting the transfer sequence

If the WebSocket test fails:
- verify the configured `webhook_path`
- verify any `shared_secret`
- verify that the OpenClaw bridge is enabled

## Related Docs
- [Startup And Daily Use](./startup-and-daily-use.md)
- [../COMMANDS.md](../COMMANDS.md)
- [../TECHNICAL_GUIDE.md](../TECHNICAL_GUIDE.md)
