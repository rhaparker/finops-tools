# finops-tools

FinOps command-line tools. The repository is a Go monorepo with a shared **core** library and a **finops** CLI built with [Cobra](https://github.com/spf13/cobra).

## Layout

| Path | Module | Role |
|------|--------|------|
| `core/` | `github.com/openshift-online/finops-tools/core` | Business logic (no CLI/HTTP dependencies) |
| `cli/` | `github.com/openshift-online/finops-tools/cli` | Cobra commands; calls into `core` |
| `go.work` | — | Ties modules together for local development |

A future REST API can live in a separate module and import the same `core` package.

## Requirements

- Go 1.24+

## Development

From the repository root (uses `go.work`):

```bash
go work sync
make test
make build
./bin/finops hello   # prints: hello
```

Or without Make:

```bash
go test ./core/... ./cli/...
go run ./cli/cmd/finops hello
go build -o bin/finops ./cli/cmd/finops
```

Edits under `core/` are picked up immediately by the CLI (workspace + `replace` in `cli/go.mod`).

## Cross-compile (local)

```bash
GOOS=linux GOARCH=amd64 go build -o bin/finops-linux-amd64 ./cli/cmd/finops
GOOS=windows GOARCH=amd64 go build -o bin/finops.exe ./cli/cmd/finops
GOOS=darwin GOARCH=arm64 go build -o bin/finops-darwin-arm64 ./cli/cmd/finops
```

## Releases

Releases are built with [GoReleaser](https://goreleaser.com/) on tag push (`v*`). Artifacts include **linux**, **darwin**, and **windows** (amd64 and arm64 where applicable).

1. Merge changes to the default branch.
2. Create and push a tag, e.g. `git tag v0.1.0 && git push origin v0.1.0`.
3. The [Release workflow](.github/workflows/release.yml) publishes binaries to GitHub Releases.

Download a release asset, extract it, and run:

```bash
finops hello
```

## CI

- **Test** (`.github/workflows/test.yml`): runs on pull requests and pushes to `main`/`master`.
- **Release** (`.github/workflows/release.yml`): runs on version tags.
