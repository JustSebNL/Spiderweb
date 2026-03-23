# Todo / Task Roadmap

## Completed
- [x] Create the internal core install chapter for DirectAdmin/LiteSpeed-compatible Spiderweb setup.
- [x] Add Go cheap-cognition client contract (`pkg/cognition`) for local Youtu/vLLM classification and summarization calls.
- [x] Add intake cheap-cognition config section with defaults for local OpenAI-compatible endpoint access.
- [x] Add single root bootstrap entrypoint `bootstrap.sh` and config file `bootstrap.conf` for bare-system setup.
- [x] Include prerequisite setup logic before Spiderweb installation.
- [x] Add automatic Go version verification and local Go installation fallback.
- [x] Implement official Spiderweb installation sequence (`make deps`, `make build`, `make install`).
- [x] Initialize project tracking files: `project.md` and `todo-tasks.md`.
- [x] Add installer config file support so install directories can be changed (not only `$HOME`).
- [x] Add internal core-install config file `install_spiderweb.conf` with customizable install paths.
- [x] Add README install usage section for the single-command bootstrap flow.
- [x] Add README documentation for bootstrap config usage and custom install paths.
- [x] Add checksum validation support for downloaded Go tarball (configurable).
- [x] Add `--dry-run` mode to validate environment/config without changing the system.
- [x] Add explicit Linux/WSL runtime guidance for Windows users.
- [x] Add Windows-shell detection with WSL distro listing/selection and automatic handoff execution.
- [x] Add explicit Linux distro/package-manager detection for adaptive prerequisite install commands.
- [x] Add Homebrew (`brew`) package manager detection as additional prerequisite-install fallback.
- [x] Add tar binary fallback support (`tar` or `gtar`) for extraction compatibility.
- [x] Add OpenClawConfig to config model with shared secret, auto-handshake, and intake toggle.
- [x] Create OpenClaw WebSocket channel (`pkg/channels/openclaw.go`) with handshake protocol.
- [x] Add WebSocket bridge endpoint to health server (`/bridge/openclaw`).
- [x] Register OpenClaw channel in channel manager.
- [x] Wire OpenClaw channel into gateway startup with WS endpoint registration.
- [x] Add OpenClaw transfer introduction auto-send on handshake.
- [x] Create `spiderweb openclaw status` CLI command.
- [x] Create `spiderweb openclaw connect` CLI command for testing WebSocket connections.
- [x] Create `spiderweb openclaw transfer` CLI command for the full transfer sequence.
- [x] Move the cheap-cognition runtime architecture to native local serving instead of Docker/container-first hosting.
- [x] Add native Trigger worker lifecycle control with owned-process shutdown behavior.
- [x] Add native Youtu runtime lifecycle control with owned-process shutdown behavior.
- [x] Add auto-loaded runtime env support from Spiderweb home for bootstrap-selected runtime values.
- [x] Wire the Go cheap-cognition client into the OpenClaw intake forward path.
- [x] Add a root `task-handoff.md` for fresh IDE session resume.
- [x] Make bootstrap install and initialize `git-lfs` for large-model support on fresh systems.
- [x] Make bootstrap Hugging Face downloads work for the public Youtu repos even when `HF_TOKEN` is unset.
- [x] Add a bootstrap pre-install feed that explains credential requirements and writes local setup notes for remaining provider configuration.
- [x] Make bootstrap persist settings and chapter progress so interrupted installs can resume from the last completed step.
- [x] Add bootstrap state inspection/reset controls so resumed installs are transparent and recoverable.
- [x] Add a documentation spine with quick start, command reference, and technical guide.

## In Progress
- [ ] Harden the native runtime maintenance service in `pkg/maintenance/service.go`.

## Next
- [ ] Add README section documenting the OpenClaw bridge setup and transfer flow.
- [ ] Add config.example.json entry for channels.openclaw configuration.
- [ ] Add integration tests for the OpenClaw channel handshake flow.
- [ ] Support multiple simultaneous OpenClaw connections (fan-out).
- [ ] Add message routing rules for filtering which messages get forwarded to OpenClaw.
- [ ] Keep Trigger.dev optional and only expand its workspace if a concrete background workflow requires it.
- [ ] Harden the native `vLLM 0.10.2` runtime bootstrap path for Youtu hosting.
- [ ] Add Youtu vLLM patch files required by the model card integration path so the native runtime can start cleanly.
- [ ] Add backoff/defer rules so maintenance yields under sustained active load instead of remediating aggressively.
- [ ] Add persistent Hugging Face cache directory configuration for first-download and warm-restart reuse.
- [ ] Expand cheap-cognition wiring beyond OpenClaw forwarding into broader intake/routing flows.
- [ ] Add failure-path behavior so Spiderweb continues journaling and routing when the cheap model service is unavailable.
- [ ] Expand `docs/cookbook/` with connector, routing, and failure-handling recipes as implementation patterns stabilize.
