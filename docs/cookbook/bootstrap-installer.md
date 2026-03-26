# Bootstrap Installer Pattern

This note describes the intended bare-system bootstrap flow.

## Goal
One root shell command should be able to prepare a mostly bare Linux host so Spiderweb can move toward a working intake stack.

## Required flow
1. Run a pre-install feed for required and optional credentials.
2. Install required system packages.
3. Make sure `git-lfs` is installed and initialized for the current user.
4. Install Spiderweb core using the internal core-install chapter.
5. Prepare `brain/` and model cache directories.
6. Install Hugging Face download tooling.
7. Inspect hardware and choose runtime autonomously.
8. If GPU path is suitable: pull standard Youtu weights into the local `brain/` area and prepare `vLLM`.
9. If GPU path is not suitable: pull GGUF weights into the local `brain/` area and prepare `llama.cpp`.
10. Generate runtime env values for the chosen local endpoint.
11. Write those values to Spiderweb home as `~/.spiderweb/runtime.env` so startup can consume them automatically.
12. Write setup guidance to `~/.spiderweb/setup-notes.txt` for any provider keys still needed after bootstrap.
13. Persist bootstrap progress and resolved settings to `~/.spiderweb/bootstrap-state.env`.
14. Leave the machine ready for `sweb wakeup`.

## Entry point and chapters
- User-facing entrypoint: `bootstrap.sh`
- User-facing config: `bootstrap.conf`
- Internal helper chapters: `scripts/`

The user should run only `bootstrap.sh`.
The helper scripts under `scripts/` are implementation chapters invoked by the root bootstrap flow.

If bootstrap is interrupted, the next run should resume from the persisted state file instead of replaying already-completed chapters.

Useful controls:
- `./bootstrap.sh --show-state`
- `./bootstrap.sh --reset-state`

## Important note
The Youtu-specific `vLLM` patch files are present under `infra/vllm/patches/`. The native `vLLM` runtime is build-complete for Youtu-LLM-2B. Remaining work is operational validation on a real GPU host.

## Important rule
This bootstrap path should aim for repeatable host preparation, but it must still remain truthful about incomplete steps. If it skips or cannot complete model serving, it should state that clearly rather than silently claiming a ready system.

## Hugging Face note
The public Youtu repositories should be downloadable even when `HF_TOKEN` is unset.

`HF_TOKEN` should remain optional unless Hugging Face access rules change or the chosen model path becomes gated.

## Credential note
Bootstrap can only gather credentials that matter to bootstrap itself.

Main LLM provider keys are still a Spiderweb runtime concern, so the bootstrap flow should leave exact guidance in `~/.spiderweb/setup-notes.txt` when those still need to be configured.
