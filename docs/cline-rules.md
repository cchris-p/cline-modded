Here’s how it works for the **CLI specifically** (your `go run ./cli/cmd/cline` flow), independent of the VS Code extension.

## 1. What rules files does the CLI actually use?

The CLI uses the **same Node core** as the extensions, so all rule loading happens there. That means when you run `cline` in a directory, the core will look for these files relative to the workspace root (for the CLI, effectively the directory you run `cline` from):

**Workspace rules (project-scoped):**
- `./.clinerules` (single file)
- `./.clinerules/` (directory of Markdown files)
- `./.windsurfrules` (single file)
- `./.cursorrules` (single file)
- `./AGENTS.md` (single file)

From `src/core/storage/disk.ts` / `dist-standalone/cline-core.js` you can see the core’s constants:
- `clineRules: ".clinerules"`
- `windsurfRules: ".windsurfrules"`
- `cursorRulesFile: ".cursorrules"`
- `agentsRulesFile: "AGENTS.md"`

All of these are treated as **rule sources**. `.clinerules` (file or directory) is the “native” format; the others are compatibility shims that are read and injected into the system prompt in the same way.

There are also **global rules** (outside the repo, under your Cline data dir) but for your question about a given chat session in a workspace, the important piece is the workspace root rules above.

## 2. Is `.windsurfrules` supported in the CLI?

Yes.

The same core that powers the CLI has explicit support for `.windsurfrules`:
- `windsurfRules: ".windsurfrules"` in storage.
- `windsurfRulesLocalFileInstructions` in `src/core/prompts/responses.ts` / `dist-standalone/cline-core.js`, which wraps the file contents as:
  ```text
  # .windsurfrules
  The following is provided by a root-level .windsurfrules file where the user has specified instructions for this working directory (...)
  ```

Because the CLI just talks to that core over gRPC, **any `.windsurfrules` in your workspace root is loaded and applied to CLI tasks exactly the same way as in VS Code / Windsurf**.

So if you run:
```bash
cd /path/to/project
cline
```
and `/path/to/project/.windsurfrules` exists, its contents will be injected into the AI’s system prompt for that session.

## 3. How to configure rules for the CLI (practical steps)

For a typical project where you want rules active whenever you run `cline` there:

### Option A: Native Cline rules (`.clinerules`)

1. **Go to your project root** (this is what the CLI will treat as the workspace root):
   ```bash
   cd /path/to/project
   ```
2. **Create a single rules file**:
   ```bash
   echo "Your project rules here" > .clinerules
   ```
   or create a **directory of rules**:
   ```bash
   mkdir -p .clinerules
   echo "TS guidelines" > .clinerules/01-typescript-style.md
   echo "Testing rules" > .clinerules/02-testing.md
   ```
   All markdown files in `.clinerules/` (excluding `workflows`, `hooks`, `skills` subdirs) are combined and used as rules.
3. **Start the CLI**:
   ```bash
   cline
   ```
   Every task you start from that directory will have these rules loaded.

### Option B: Windsurf-style rules (`.windsurfrules`)

If you already maintain `.windsurfrules` for Windsurf, or prefer that format:

1. At your project root, create/edit:
   ```bash
   cd /path/to/project
   $EDITOR .windsurfrules
   ```
2. Put your instructions inside. For example:
   ```markdown
   # Windsurf / Cline Rules
   - Always use Go for examples in this repo.
   - Prefer table-driven tests.
   ```
3. Run `cline` from that directory:
   ```bash
   cline
   ```
   The core will detect `.windsurfrules` and inject it into the model’s system prompt for all tasks in that workspace.

You can even have **both** `.clinerules` and `.windsurfrules` present; core will merge rule sources according to its internal order and rule toggles.

## 4. Important nuance for the CLI

- **Workspace root = where you run `cline`** in the CLI. The host bridge for CLI reports the current working directory as the workspace root, so rules need to live there (or you run `cline` from your actual project root).
- There is **no special CLI-only “rules config” command**; it’s all file-based detection. `cline config` is for settings, not for defining rules.
- Hooks & workflows are also driven off the same `.clinerules` tree:
  - Hooks: `.clinerules/hooks/*`
  - Workflows: `.clinerules/workflows/*.md`
  - Skills: `.clinerules/skills/` or `.cline/skills/`
  These work from the CLI as well because they’re implemented in the shared core + hostbridge.

---

**TL;DR for your use case**

If you want rules to be detected and active when you run the CLI:
- Put your rules in **either** `.clinerules` / `.clinerules/*.md` **or** `.windsurfrules` at the project root.
- Run `cline` **from that directory**.
- The CLI will automatically pick them up via the shared core; you don’t need any VS Code extension involved.

If you tell me which convention you’d like to standardize on (`.clinerules` vs `.windsurfrules`), I can sketch a concrete example tailored to your repos (e.g., Go-style rules for this CLI repo).