# Delete Task Feature: Current Status and Recommended Path Forward

## Summary

- **Recommendation:** Do **not** merge `feature/delete-task` as-is. Re-implement “delete task” directly in the **current TypeScript CLI** (Option A).
- **Why:** The `feature/delete-task` branch implements a **different CLI architecture** (Go/Cobra) that no longer matches the repo’s current CLI (`cli/src/index.ts`, TypeScript + Ink + Commander). It is effectively a parallel product and is currently incompatible.

## What exists today (on `main`)

### TypeScript CLI (authoritative)

- **Entry point:** `cli/src/index.ts`
- **Command framework:** `commander`
- **UI layer:** `ink` (interactive TUI-style UI)
- **Existing commands:**
  - `cline task` (run/resume tasks)
  - `cline history` (interactive history browser)
  - `cline config`, `cline auth`, `cline mcp`, etc.

### Core delete implementation exists

- **Core handler:** `src/core/controller/task/deleteTasksWithIds.ts`
- **Behavior:**
  - Prompts the user to confirm deletion.
  - Clears the active task if it matches.
  - Deletes task files under the task directory.
  - Removes empty task dir; if no tasks remain, removes `tasks/` and `checkpoints/` directories.
  - Posts updated state.

### Key compatibility issue: confirmation prompt in CLI mode

The core delete flow uses:

- `HostProvider.window.showMessage({ modal: true, items: ["Delete"] })`

In **CLI mode**, `CliWindowServiceClient.showMessage` currently:

- Prints the message
- Returns an **empty** `SelectedResponse` (no `selectedOption`)

That means `deleteTasksWithIds` will treat the response as “not confirmed” and **will not delete anything**.

So Option A requires **either**:

- Implementing a real selection/confirmation in the CLI `showMessage` stub **when** `modal=true` and `items` are provided, **or**
- Adding a `--force` path that bypasses the core confirmation.

## What exists in `feature/delete-task`

### Go CLI implementation (stale relative to current repo)

Based on `changes.diff`, the branch adds:

- A Go/Cobra subcommand: `cline task delete <task-id>` with flags:
  - `--all`
  - `--force`
- Disk-level deletion logic that:
  - edits `taskHistory.json`
  - removes task directories
  - best-effort cleanup of folder locks in `locks.db`

### Why merging is a bad idea

- **Different CLI stack:** the branch is centered around a Go CLI, while current development is TypeScript (`cli/src/*`). Merging would mix two CLIs with overlapping responsibilities.
- **Build breakage:** the Go CLI in that branch depended on repo layout assumptions that are no longer true (e.g., `replace` paths that expect generated Go modules to exist). In practice, it does not build cleanly in the current repo state.
- **Product direction:** the current repo clearly positions `cli/` as a TypeScript CLI that reuses the core Controller code.

## Option A (recommended): Re-implement delete task in the TypeScript CLI

### UX decision points (choose one)

- **A1. Command-only (fastest):**
  - Add `cline history delete <task-id>`
  - Add optional `--force`
  - (Optional) add `--all` later

- **A2. Command + interactive history UI shortcut (best UX):**
  - Same as A1
  - Add a keybinding in `HistoryView` (e.g. `d`) to delete the currently selected task

### Where it fits in the current CLI

- **Command wiring location:** `cli/src/index.ts`
- `history` command already exists and already accepts `--config`. A `delete` subcommand under `history` is the least disruptive.

Examples:

- `cline history delete <taskId>`
- `cline history delete <taskId1> <taskId2> ...` (multi-delete)
- `cline history delete --all`

### Implementation checklist

1. **Add a new CLI subcommand**
   - Location: `cli/src/index.ts`
   - Add `history delete` (and optionally aliases like `del`, `rm`).

2. **Initialize CLI context the same way other commands do**
   - Reuse `initializeCli({ config })`.

3. **Resolve which tasks to delete**
   - For explicit IDs: use args.
   - For `--all`: read from `StateManager.get().getGlobalStateKey("taskHistory")`.

4. **Fix confirmation behavior (required)**

   Pick one:

   - **Approach 1 (preferred): implement confirmation in CLI `showMessage`**
     - Update `CliWindowServiceClient.showMessage` in `cli/src/controllers/index.ts`:
       - If `request.options?.modal` and `request.options?.items?.length > 0`, prompt in the terminal for a selection.
       - Return `SelectedResponse` with `selectedOption` set to the chosen item.
     - This makes `deleteTasksWithIds` work without changing core logic.

   - **Approach 2: bypass confirmation via a `--force` flag**
     - If `--force`, call a deletion routine that does not prompt.
     - Note: `deleteTaskWithId` is currently not exported, so this approach likely requires changes in core (new exported helper or an option in the request).

5. **Perform the deletion**
   - Call core handler: `deleteTasksWithIds(ctx.controller, StringArrayRequest.create({ value: ids }))`.

6. **Exit codes and messaging**
   - For missing IDs or unknown task ID:
     - Decide whether to hard-fail (`exit(1)`) or warn and continue.

7. **Tests**
   - Add/extend unit tests in `cli/src/index.test.ts` or component tests.
   - At minimum:
     - `history delete` deletes one task
     - confirmation flow works
     - respects `--config` (custom CLINE_DIR)

### Notes / pitfalls

- **`CLINE_DIR` vs `--config`:**
  - CLI context supports both (`initializeCliContext` uses `config.clineDir || process.env.CLINE_DIR || ~/.cline`).
  - Prefer `--config` for deterministic tests.

- **Folder locks (`locks.db`):**
  - The old Go implementation tried to remove folder locks explicitly.
  - The current core delete handler does **not** touch SQLite locks.
  - This is probably OK if locks are correctly released during normal operation, but if you observe stale folder locks, consider adding best-effort cleanup later.

## What to do with `feature/delete-task`

- **Preferred:** keep it around as a historical reference only, then delete it once Option A is implemented and merged.
- **If you want to preserve some work:** copy the UX semantics (flags, output behavior) but do not try to reuse the Go code directly.

## Suggested next steps (practical)

1. Implement “confirmation selection” behavior in `CliWindowServiceClient.showMessage` for modal/item prompts.
2. Add `cline history delete <task-id> [--force]` and wire it to `deleteTasksWithIds`.
3. Add a minimal test that ensures deletion proceeds when user selects `Delete`.
4. (Optional) Add `d` shortcut in `HistoryView` to delete selected task and refresh the list.
