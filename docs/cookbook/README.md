# Spiderweb Cookbook

This directory stores reusable local patterns for the project.

Use it for:
- small implementation recipes
- code snippets worth reusing
- infrastructure setup patterns
- interface examples
- integration notes that are too practical for durable architecture docs

Do not use it for:
- task tracking
- large vendor doc copies
- high-level architecture truth that belongs in the main repo docs

## Intended categories
- `trigger-patterns.md`
- `vllm-serving.md`
- `event-schema.md`
- `connector-patterns.md`
- `routing-patterns.md`
- `failure-handling.md`
- `handoff-packets.md`

## Current high-value recipes
- `connector-patterns.md`: how a source should emit Spiderweb-compatible intake messages
- `routing-patterns.md`: current forward allow/deny controls and cheap-cognition routing effects
- `failure-handling.md`: degraded-mode contracts for cheap-cognition failure and OpenClaw unavailability

## Rule of use
If a pattern is likely to be needed again, save it here once instead of rediscovering it later.
