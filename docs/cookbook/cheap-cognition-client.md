# Cheap Cognition Client

This note documents the Go client contract implemented in `pkg/cognition`.

## Purpose
Spiderweb needs a native Go seam for calling the local cheap cognition model without coupling intake code directly to raw HTTP request details.

## Implemented package
- `pkg/cognition/types.go`
- `pkg/cognition/client.go`
- `pkg/cognition/client_test.go`

## What it does
The package currently provides:
- a normalized `Event` type for cheap cognition input
- a typed `ClassificationResult`
- a `Client` that talks to an OpenAI-compatible `/chat/completions` endpoint
- `ClassifyEvent(...)`
- `SummarizeText(...)`
- `NewClientFromConfig(...)`

## Current config path
The client is configured through:
- `cfg.Intake.CheapCognition.Enabled`
- `cfg.Intake.CheapCognition.BaseURL`
- `cfg.Intake.CheapCognition.APIKey`
- `cfg.Intake.CheapCognition.Model`
- `cfg.Intake.CheapCognition.TimeoutSeconds`

## Current integration
This package is now wired into a real Spiderweb path:
- the OpenClaw intake forward path classifies inbound messages before forwarding
- forwarded payload metadata now carries cheap-cognition priority/category/summary fields
- low-priority non-escalations can be skipped before they reach OpenClaw

## Current limitation
Cheap cognition is not yet wired across all intake and routing paths.

That means the next implementation step is broader integration, not redesign.
