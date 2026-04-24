# CLAUDE.md

## Commands

Task automation uses `just` (see `justfile`). Common commands:

- `just build` — build the binary from `./cmd/mcp-domain-checker`, injecting version/commit via `-ldflags`
- `just run` — run without a separate build
- `just lint` — `golangci-lint run --config .golangci.yaml` (config uses `default: all` with a short disable list, so new code will hit many linters)
- `just test` — `go test -v -race ./...`
- `just test-cover` — same with `-coverprofile=coverage.out`
- `just deadcode` — `golang.org/x/tools/cmd/deadcode` across the module
- `just build-docker` / `just run-docker` — build/run the container (exposes `:8080`)
- `just tools-list` — invoke `@modelcontextprotocol/inspector` against a running server to list MCP tools
- `just ci` — mirror of the GitHub Actions pipeline: `lint → deadcode → test-cover → build → build-docker`. Run this before pushing.

Run a single test: `go test -v -race ./internal/pkg/namecheap -run TestName`.

The `justfile` exports Namecheap env vars (all blank by default). Fill them in locally or export them in your shell before `just run`.

## Architecture

### Transport

The server supports **two transports**, selected by the `-transport` flag (which overrides the `TRANSPORT` env var). Default is `http`.

- `http` — **streamable HTTP** (`mcp.NewStreamableHTTPHandler`) on `:8080` with a permissive CORS middleware. Long-lived service model for Docker / remote hosts. Shutdown is graceful: SIGINT/SIGTERM triggers `http.Server.Shutdown` with a `shutdownTimeout` drain window.
- `stdio` — **stdio** (`mcp.StdioTransport{}` via `Server.Run`). Client spawns the binary as a subprocess and speaks newline-delimited JSON over stdin/stdout. This is the `npx`-style local install path (`go install …@latest` + `command: "mcp-domain-checker"`).

**Stdout discipline for stdio:** the transport owns `os.Stdout`. zap's production/development configs default to stderr, so logging is safe. The `--version` flow prints to stdout but `os.Exit(0)`s before the transport starts. Any future code that writes to stdout outside the transport would corrupt stdio framing — don't.

`resolveTransport(flagVal, envVal)` in `main.go` centralises validation. Adding a new transport means extending its switch, the consts, and the dispatch in `main`.

### Tool registration is config-gated

`cmd/mcp-domain-checker/main.go:setupTools` only registers the Namecheap tool when **all four** of `NAMECHEAP_API_USER`, `NAMECHEAP_API_KEY`, `NAMECHEAP_USERNAME`, `NAMECHEAP_CLIENT_IP` are set. Missing credentials → the server starts with zero tools and logs "Namecheap tool disabled". This is intentional; don't make credentials fatal at startup.

### Generic tool wrapper pattern

`internal/pkg/tool` is a generic MCP adapter, not Namecheap-specific. Any new tool should:

1. Implement `tool.Service[In, Out]` — `Name()`, `Description()`, `Execute(in In) (Out, error)`.
2. Get wrapped via `tool.NewTool(service)` and registered in `setupTools` with `mcp.AddTool`.

`Tool.Handler` handles the MCP plumbing: calls `Execute`, marshals the result to JSON, wraps it as `mcp.TextContent` with assistant-audience annotations. New tools should not re-implement this — extend the pattern.

The Namecheap service (`internal/pkg/namecheap`) is the reference implementation and also defines its own `DomainChecker` interface for testability.

### Versioning

`version` and `commit` in `main.go` are package-level vars set via `-ldflags "-X main.version=... -X main.commit=..."`. `just build`/`just run` inject the short git hash; goreleaser injects the release tag. Don't hardcode these.

## Release flow

Releases are driven by **release-please** (`.github/workflows/release-please.yml`, `release-please-config.json`, `.release-please-manifest.json`) + **goreleaser** (`.goreleaser.yaml`). The manifest tracks the current published version; release-please reads it, scans new conventional commits on `main`, and opens/updates a release PR proposing the next version. Merging that PR tags it, which triggers goreleaser to build cross-platform archives (linux/darwin/windows × amd64/arm64) and publish to GitHub Releases. The changelog filters out `docs:`, `test:`, `ci:`, `chore:` commits.

### Semver mapping (release-type: `go`)

`release-please-config.json` sets `release-type: go`, which uses the default conventional-commits → semver mapping. Commit prefix decides the bump:

| Commit prefix                      | Bump  | In changelog | Example |
|-----------------------------------|-------|--------------|---------|
| `feat:`                            | MINOR | yes          | new feature, new flag, new transport |
| `fix:`                             | PATCH | yes          | bug fix |
| `perf:`, `refactor:`               | PATCH | yes          | internal-only improvements |
| `deps:` / `build:`                 | PATCH | yes          | dependency / build changes |
| `docs:`, `chore:`, `test:`, `ci:`, `style:` | none  | filtered out | no release triggered |
| any of the above with `!` (e.g. `feat!:`) or a `BREAKING CHANGE:` footer | MAJOR | yes, highlighted | removed flag, renamed env var, changed default behaviour |

### What counts as breaking

Anything a downstream user (Docker operator, MCP-client config, script calling the CLI) would have to change to keep working:

- removing or renaming a flag, env var, or config field (e.g. dropping `TRANSPORT`, renaming `NAMECHEAP_API_KEY`);
- changing a default that flips observable behaviour (e.g. changing the default `TRANSPORT` from `http` to `stdio`, changing the listen addr from `:8080`);
- removing a transport, or removing CORS in a way that breaks existing browser clients;
- removing a tool, or changing a tool's input/output schema in a non-additive way.

Not breaking: adding a new flag with a safe default, adding a new transport/tool, internal refactors, stricter input validation on previously-invalid inputs, graceful-shutdown additions.

When in doubt: if the change is a behavioural improvement that no reasonable caller could depend on the old behaviour of, it's not a break. If you're uncertain, prefer `feat!:` — shipping a premature MAJOR is cheaper to recover from than quietly shipping a break under `feat:`.

### Practical flow

1. Land conventional-commit PRs on `main`. Use the prefix that matches the desired bump.
2. release-please opens or updates a PR titled `chore(main): release X.Y.Z` with the aggregated changelog and bumped `.release-please-manifest.json`.
3. Review and merge the release PR when ready — this is the explicit "cut a release" action.
4. Merging tags `vX.Y.Z`; goreleaser workflow runs and publishes binaries.

Don't hand-edit `.release-please-manifest.json` or create tags manually — the flow owns both.
