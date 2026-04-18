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

The server currently uses **streamable HTTP** (`mcp.NewStreamableHTTPHandler`) on `:8080` with a permissive CORS middleware — not stdio. That means it's designed to run as a long-lived service (Docker, remote host) that MCP clients connect to over HTTP, not as a subprocess spawned by the client. Keep this in mind when changing transport or wiring up new clients.

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

Releases are driven by **release-please** (`.github/workflows/release-please.yml`) + **goreleaser** (`.goreleaser.yaml`). Conventional commits on `main` produce release PRs; merging them tags a version, which triggers goreleaser to build cross-platform archives (linux/darwin/windows × amd64/arm64) and publish to GitHub Releases. The changelog filters out `docs:`, `test:`, `ci:`, `chore:` commits.
