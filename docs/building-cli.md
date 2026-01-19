`npm install -g .` in `cline/cline` does **not** produce a `cline` command, because the repo root `package.json` is primarily for the **VS Code extension build**, not the Go CLI entrypoint.

The Go CLI entrypoint is:

- `cli/cmd/cline/main.go` ([GitHub][1])

The canonical workflow for local CLI development is **Go build/install**, not npm.

---

## Build and run the Go CLI from the repo

From the repo root:

```bash
cd /path/to/cline-repo

# Build local dev binaries (cline + cline-host)
go build -o ./cli/bin/cline ./cli/cmd/cline
go build -o ./cli/bin/cline-host ./cli/cmd/cline-host

# Run it
./cli/bin/cline version
```

Rebuild after changes to Go code under `cli/`:

```bash
go build -o ./cli/bin/cline ./cli/cmd/cline
go build -o ./cli/bin/cline-host ./cli/cmd/cline-host
```

For fastest iteration, use the built binaries with a fixed core path. A new standalone build
is only required when the core TypeScript changes (not when Go code in `cli/` changes).

```bash
# One-time (or whenever core TypeScript changes)
npm run compile-standalone
```

---

## Fast local development aliases (CLI-only changes)

```bash
export CLINE_CORE_PATH=/home/matrillo/apps/cline-modded/dist-standalone/cline-core.js

alias build-cline='
  cd /home/matrillo/apps/cline-modded && \
  go build -o /home/matrillo/apps/cline-modded/cli/bin/cline ./cli/cmd/cline && \
  go build -o /home/matrillo/apps/cline-modded/cli/bin/cline-host ./cli/cmd/cline-host && \
  chmod +x /home/matrillo/apps/cline-modded/cli/bin/cline /home/matrillo/apps/cline-modded/cli/bin/cline-host && \
  echo "Built CLI binaries" || echo "Build failed"
'

alias cline='PATH=$HOME/apps/cline-modded/cli/bin:$PATH cline'
```

- `CLINE_CORE_PATH` lets `go run` or the dev binaries find `cline-core.js` without relying on
  executable-relative paths.
- Re-run `npm run compile-standalone` only if core TypeScript changes.

## Make the dev build available as `cline` globally

### Option A: install into /usr/local/bin (common on macOS/Linux)

```bash
sudo install -m 0755 ./cli/bin/cline /usr/local/bin/cline
which cline
cline version
```

### Option B: install into a user bin dir (no sudo)

```bash
mkdir -p ~/.local/bin
install -m 0755 ./cli/bin/cline ~/.local/bin/cline

# ensure ~/.local/bin is on PATH in the shell config
export PATH="$HOME/.local/bin:$PATH"

which cline
cline version
```

---

## Why npm did not work here

`npm install -g cline` installs the published CLI package, but `npm install -g .` inside the repo root won’t necessarily create a `cline` executable unless that specific `package.json` defines a `bin` entry for it (and the build step produces it). In this repo, the Go CLI is a separate build target under `cli/` ([GitHub][1]).

Also note: the actual `cline` executable is a **Go binary**, which is why platform/GLIBC constraints can appear in some environments ([GitHub][2]).

---

[1]: https://github.com/cline/cline/issues/6768?utm_source=chatgpt.com "(WIP) I created a guide to build the CLI on Windows #6768"
[2]: https://github.com/cline/cline/issues/8455?utm_source=chatgpt.com "cline cli 1.0.9 broke support on older machines · Issue #8455"
