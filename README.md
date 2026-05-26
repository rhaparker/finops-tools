# finops-tools

FinOps command-line tools. The repository is a Go monorepo with a shared **core** library and a **finops** CLI built with [Cobra](https://github.com/spf13/cobra).

## Layout

| Path | Module | Role |
|------|--------|------|
| `core/` | `github.com/openshift-online/finops-tools/core` | Business logic (no CLI/HTTP dependencies) |
| `cli/` | `github.com/openshift-online/finops-tools/cli` | Cobra commands; calls into `core` |
| `go.work` | — | Ties modules together for local development |

A future REST API can live in a separate module and import the same `core` package.

## CLI commands

Every command uses **`finops <noun> <verb>`** (e.g. `finops account add`, `finops demo hello`). See `.cursor/rules/cli-commands.mdc` for conventions when adding commands.

## Requirements

- Go 1.24+

## Development

From the repository root (uses `go.work`):

```bash
go work sync
make test
make build
./bin/finops demo hello   # prints: hello
```

Or without Make:

```bash
go test ./core/... ./cli/...
go run ./cli/cmd/finops demo hello
go build -o bin/finops ./cli/cmd/finops
```

Edits under `core/` are picked up immediately by the CLI (workspace + `replace` in `cli/go.mod`).

## Configuration

FinOps stores local settings in a YAML config file:

| OS | Default path |
|----|----------------|
| Linux / macOS | `$XDG_CONFIG_HOME/finops/config.yaml` or `~/.config/finops/config.yaml` |
| Windows | `%AppData%/finops/config.yaml` |

The file is created automatically on first `finops account add`. Example:

```yaml
defaults:
  aws.auth_method: profile
  aws.linked_role: OrganizationAccountAccessRole
aws:
  account_aliases:
    rh-control: "123456789012"
    osd-staging-1: "987654321098"
    osd-tenant-1:
      account_id: "111111111111"
      payer_alias: rh-control
      role: OrganizationAccountAccessRole
gcp:
  account_aliases: {}
```

Set defaults using fully qualified names (used when `--auth-method` is omitted on `account add`):

```bash
finops config default set --name aws.auth-method --value profile
finops config default get --name aws.auth-method
finops config default set --name aws.linked_role --value OrganizationAccountAccessRole
```

Register a **payer** account by **12-digit account ID** (login + save in config):

```bash
finops account add aws 123456789012 --alias rh-control       # auth-method: flag, or config default, else saml
finops account add aws 123456789012 --auth-method profile  # overrides config default
finops account add aws 123456789012 --force
```

Register a **linked** account (authenticate to the payer first, then assume a role in the member account):

```bash
finops account add aws 111111111111 --alias osd-tenant-1 --payer rh-control
finops account add aws 111111111111 --payer rh-control --role CustomRole
```

The IAM role name defaults to `OrganizationAccountAccessRole`, or `defaults.aws.linked_role` in the finops config. The CLI builds `arn:aws:iam::<account-id>:role/<role-name>` automatically.

Without `--alias`, the account ID is used as the config key. Aliases are CLI-only; cost and AWS credential logic use 12-digit account IDs.

List registered accounts (payer vs linked):

```bash
finops account list
finops account list aws
```

**Cost Explorer (`finops cost get`) requires payer accounts only.** Linked-account credentials are for member-account APIs, not payer-level billing queries.

Static secrets (API keys, etc.) for other tools live in `~/.config/finops/.env`; AWS sessions use `~/.aws/credentials` profiles.

### AWS payer credentials

Store and verify temporary AWS credentials for a payer account (same profile layout as finops-mcp-aws):

```bash
finops account add aws 123456789012
```

**Behavior:**

1. Looks up a profile derived from the account ID in `~/.aws/credentials`, then in `~/.aws/config` (shared config / SSO).
2. If the profile exists and STS validation succeeds, reports success without logging in again.
3. With `--auth-method saml` (default), if credentials are missing or invalid, runs `rh-aws-saml-login --output env <account>` and merges credentials into `~/.aws/credentials` (other profiles are preserved).
4. With `--auth-method profile`, SAML login is skipped. `finops account add` uses an existing profile when valid (including a `~/.aws` profile named like `--alias`, e.g. `rh-control`); in an interactive terminal it prompts for access keys only when no matching profile exists. You can also configure the profile yourself (`aws configure`, `aws sso login`, etc.) and run `account add` again to confirm it works.

**SAML prerequisites** (default login, and `--force`):

- Red Hat VPN connected
- Valid Kerberos ticket (`kinit`)
- [`rh-aws-saml-login`](https://github.com/app-sre/rh-aws-saml-login) installed (e.g. `uv tool install rh-aws-saml-login`)

If SAML login prompts for a password, use your **Red Hat Kerberos** password (same as `kinit`), not your AWS console password.

#### Linked accounts (profile chaining without finops flags)

You can also configure role assumption in `~/.aws/config` and use `--auth-method profile`:

```ini
[profile rh-control]
# payer credentials (SAML output, SSO, or keys)

[profile osd-tenant-1]
role_arn = arn:aws:iam::111111111111:role/OrganizationAccountAccessRole
source_profile = rh-control
```

```bash
finops account add aws 111111111111 --alias osd-tenant-1 --auth-method profile
```

STS validation must report the **linked** account ID. This registers credentials only; for finops metadata (`payer_alias`, `role`) use `--payer` (and optional `--role`) as shown above.

### Cost (AWS)

Fetch the last 30 days of **Net Amortized Cost** from AWS Cost Explorer. Payer and linked
account aliases are supported; linked accounts query Cost Explorer through the registered payer.

```bash
finops account add aws 123456789012 --alias rh-control
finops cost get --account-alias rh-control
finops cost get --account-alias quay              # linked account (uses payer credentials)
finops cost get --account 123456789012
finops cost get --account-alias rh-control,osd-staging-1
finops cost get --account 123456789012 --format json
finops cost get --account 123456789012 --format csv
finops cost get --account 123456789012 --split-by service
finops cost get --account 123456789012 --split-by account
```

| Flag | Description |
|------|-------------|
| `--account` | One or more comma-separated **12-digit AWS account IDs** (must be registered with `account add`); at least one of `--account` or `--account-alias` is required |
| `--account-alias` | One or more comma-separated configured aliases (e.g. `rh-control`, or a linked alias such as `quay`) |
| `--auth-method` | `saml` (default) or `profile`; when omitted, uses `defaults.aws.auth_method` from config |
| `--config` | Path to finops config file (default: OS-specific config dir) |
| `--format` | `pretty-print` (default), `json`, or `csv` |
| `--split-by` | Group costs by dimension: `service` (AWS service) or `account` (linked AWS account ID); includes share % and relative cost bars in `pretty-print` |
| `--provider` | `aws` (default). `gcp` is reserved for a future release |

`pretty-print` uses colors and Unicode bars when stdout is a TTY. Set `NO_COLOR=1` to disable; `FORCE_COLOR=1` forces colors when piping to a capable viewer.

### Reports

Generate HTML reports from configured accounts. Templates use **Jinja2 syntax** (rendered with [gonja](https://github.com/nikolalohinski/gonja) in the CLI binary).

```bash
finops report list
finops report generate costs --account-alias rh-control
finops report generate costs --account-alias rh-control -o costs.html
```

The **costs** template includes:

- Total net amortized cost for the last 30 days
- Breakdown by linked AWS account
- Breakdown by AWS service
- Daily cost trend chart (embedded SVG; works when opening the HTML file locally)

| Flag | Description |
|------|-------------|
| `template` | Positional argument: report template name (run `finops report list` for options) |
| `--format` | Output format (default: `html`) |
| `--account` | Comma-separated payer AWS account IDs (at least one of `--account` or `--account-alias` is required) |
| `--account-alias` | Comma-separated configured aliases |
| `--auth-method` | `saml` (default) or `profile` |
| `--config` | Path to finops config file |
| `--credentials-file` | Path to AWS credentials file |
| `--output` / `-o` | Write HTML to a file instead of stdout |
| `--quiet` | Suppress progress messages on stderr (HTML still goes to stdout or `--output`) |

Progress lines (credential checks, each Cost Explorer query, render) are printed to **stderr** so you can redirect HTML safely, e.g. `finops report generate ... -o report.html`.

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
finops demo hello
```

## CI

- **Test** (`.github/workflows/test.yml`): runs on pull requests and pushes to `main`/`master`.
- **Release** (`.github/workflows/release.yml`): runs on version tags.
