## Local Development Intent (CLI-First)

The primary goal of local development in this repository is to run and iterate on the **Cline CLI runtime**.

- Focus: CLI behavior, CLI task execution, CLI integrations
- Non-goal (for day-to-day local workflow): VS Code extension development/debug loops

This project may still contain extension-related sources and build scripts, but local contributor workflows should optimize for CLI reliability first.

---

## Canonical Local Workflow

1. Bootstrap dependencies and generated artifacts.
2. Build CLI binaries when Go CLI code changes.
3. Build standalone core only when core TypeScript changes.
4. Run CLI and validate behavior from terminal.

Use the local shell helpers in [`cline-config`](../standards/cline-config):

- [`cline-setup`](../standards/cline-config:15)
- [`build-cline`](../standards/cline-config:21)
- [`cline-core-build`](../standards/cline-config:18)

---

## High-Risk Failure Point (Important)

### Core/CLI version or runtime-layout mismatch

The most common local failure is forcing `CLINE_CORE_PATH` to a standalone core build that does **not** match the CLI binary/runtime expectations.

Symptoms include:

- startup messages that mention generic Node incompatibility even when Node is valid
- missing runtime module errors (e.g. `Cannot find module 'vscode'`)
- missing runtime layout errors (e.g. `.../extension/package.json` not found)
- instance startup failures (`instance not found in registry`)

Why this happens:

- CLI binary and core runtime are version-coupled
- standalone core expects a specific packaged directory structure
- overriding core path can bypass bundled-compatible defaults

Mitigation:

- Default to bundled core unless intentionally testing a matched dev pair
- only opt into local core with [`cline-use-local-core`](../standards/cline-config:11)
- return to safe default with [`cline-use-bundled-core`](../standards/cline-config:12)

---

## Guardrails for Agents

- Do not assume extension-oriented scripts are appropriate for CLI-first debugging.
- Validate local CLI runtime before proposing extension-level fixes.
- When `cline` fails to start, inspect recent logs under `~/.cline/logs` before changing aliases or env vars.
- Treat `CLINE_CORE_PATH` overrides as an advanced/debug setting, not a default.
