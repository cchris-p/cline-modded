Not directly / not automatically.

`npm install -g cline` installs **a packaged copy** of the `cline` CLI into the **global npm install location** on the machine (`/usr/lib/node_modules/...` or similar). Editing the upstream Go project files in some random folder will **not** change the already-installed global CLI.

## What happens after a global install

* The executable that runs when typing `cline ...` is a **built artifact** placed in the global npm prefix.
* Changing source code elsewhere **does nothing** unless the global installation is rebuilt/reinstalled from that modified source.

## If the goal is “edit code → immediately run modified CLI”

Use one of these workflows:

### Option A) Link a local checkout (best for development)

1. Clone the repo locally
2. Build it (Go build steps)
3. Install / link the resulting binary onto `PATH`

For a pure Node package workflow (if the project supports it):

```bash
git clone <repo>
cd <repo>
npm install
npm link
```

Then `cline` resolves to the locally linked version instead of the global registry copy.

### Option B) Reinstall globally from the local folder

If the package supports it:

```bash
cd /path/to/local/checkout
npm install -g .
```

That overwrites the global `cline` with whatever exists in that folder.

### Option C) Just reinstall from npm after changes (not great)

This only works if changes were published to npm (not local edits):

```bash
npm install -g cline
```

## Quick test: where is `cline` coming from?

```bash
which cline
cline --version
npm root -g
```

That will show whether the command is coming from the global node_modules install or a linked/local build.

If the repo path being edited and the `which cline` path do not match, edits will not affect the CLI.
