# Terraform Provider for Trufo

[![CI](https://github.com/NitriKx/terraform-provider-trufo/actions/workflows/ci.yml/badge.svg)](https://github.com/NitriKx/terraform-provider-trufo/actions/workflows/ci.yml)
[![Registry](https://img.shields.io/badge/registry-nitrikx%2Ftrufo-lightgrey?logo=terraform)](https://registry.terraform.io/providers/nitrikx/trufo/latest/docs)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

Terraform provider for the [Trufo](https://trufo.ai) Provenance Platform, built on
[`github.com/NitriKx/terraform-provider-trufo/trufo-go`](https://github.com/NitriKx/terraform-provider-trufo/tree/main/trufo-go)
— a Go client ported from the official [Python SDK (trufo-py)](https://github.com/trufo-ai/trufo-py).

Provider address: `registry.terraform.io/nitrikx/trufo`

## Documentation

- Official reference docs on the [Terraform Registry](https://registry.terraform.io/providers/nitrikx/trufo/latest/docs) (generated from [`docs/`](./docs) + [`templates/`](./templates))
- Runnable examples in [`examples/`](./examples), including a [full signing-instance setup](./examples/full-signing-instance/) and a [key rotation guide](./docs/guides/key-rotation.md)
- [Contributing](./CONTRIBUTING.md) · [Support](./.github/SUPPORT.md) · [Security](./SECURITY.md) · [Issues](https://github.com/NitriKx/terraform-provider-trufo/issues)

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.16
- [Go](https://go.dev/doc/install) 1.27+ (only to build from source)

## Compatibility

| Trufo Provider | Terraform Plugin Protocol | Terraform CLI |
| :---: | :---: | :---: |
| `>= 0.1.x` | `6` | `>= 1.16` |

## Resources & data sources

| Type | API | Notes |
| --- | --- | --- |
| `trufo_instance` | `POST /gproduct/instance/*` | Full CRUD; import by `gpi_…` |
| `trufo_instance_credential` | `POST /gproduct/instance/credential/*` | Enroll a public key; delete = revoke; import as `{gpi_id}/{gpic_id}` |
| `trufo_ai_system` | `POST /ai-system/*` | Register + active/inactive toggle; delete = inactive; production hosts only |
| `trufo_generator_product` (data source) | `POST /gproduct/list`, `/gproduct/info` | Products are created in the dashboard; Bearer auth |
| `trufo_certificates` (data source) | `POST /cert/list` | Bearer auth |

## Authentication

Every Trufo API request carries exactly one credential, and so does the provider:

- **Access token (Bearer)** — `access_token` + `refresh_token` attributes, `TRUFO_ACCESS_TOKEN` / `TRUFO_REFRESH_TOKEN`
  env vars, or a persisted `~/.trufo/session` file (written automatically after every refresh, like the trufo-py CLI).
  Sent as `Authorization: Bearer`, refreshed automatically on 401. **Required** for the generator product and
  certificate data sources and for instance/credential management on production hosts (those endpoints require an
  MFA-verified token). If only a `trufo-api`-scoped `api_key` is configured, the provider runs the interactive login
  (loopback browser flow, or device flow — see `auth_flow`) at configure time and persists the session.
- **API key** — `api_key` attribute or `TRUFO_API_KEY` env var (`X-API-Key: {scope}:{key}`). Covers
  `trufo_ai_system` with a `c2pa-sign-*`/`watermark-*` scoped key (production hosts), and instance/credential
  management with a `gproduct-admin` scoped key on the test host.

## Regions and hosts

The provider defaults to the **global production API** (`https://api.trufo.ai`). Other hosts via `region` or a
full `api_url` (or the `TRUFO_API_URL` env var):

| Host | `region` | Use |
| --- | --- | --- |
| `https://api.trufo.ai` | `global` (default) | Production, global |
| `https://eu.api.trufo.ai` | `eu` | Production, EU — media submitted there is processed and stored in EU infrastructure |
| `https://test.api.trufo.ai` | `test` | Test API — certificate test flow and test-scoped keys; nothing carries over to production |

```hcl
provider "trufo" {} # production, global (default)

provider "trufo" {
  alias  = "eu"
  region = "eu"
}

provider "trufo" {
  alias  = "test"
  region = "test"
}
```

## Example usage

```hcl
terraform {
  required_providers {
    trufo = {
      source  = "nitrikx/trufo"
      version = "~> 0.1"
    }
  }
}

provider "trufo" {
  region = "test" # TRUFO_API_KEY env var provides the API key
}

resource "trufo_instance" "signing_service" {
  gp_id = "gp_..."
  name  = "signing-service-eu-1"
}

resource "trufo_instance_credential" "primary" {
  gpi_id         = trufo_instance.signing_service.id
  public_key_pem = file("instance-public-key.pem")
  key_algorithm  = "ES256"
  label          = "primary-2026"
}
```

More examples in [`examples/`](./examples); generated reference docs in [`docs/`](./docs).

## Development

```sh
make build       # or: go build ./...
make test        # unit tests (client uses a mock server)
make testacc     # acceptance tests against test.api.trufo.ai (needs TF_ACC=1 + env, see below)
make generate    # regenerate docs/ + terraform fmt examples/
```

### Using a development build

Build the binary and point Terraform at it with a CLI config override (`~/.terraformrc`):

```hcl
provider_installation {
  dev_overrides {
    "nitrikx/trufo" = "/path/to/bin"
  }
  direct {}
}
```

Acceptance tests run against `https://test.api.trufo.ai` (override with `TRUFO_TEST_API_URL`; AI system tests
use `https://api.trufo.ai`, override with `TRUFO_PROD_API_URL`) and need:

- `TRUFO_API_KEY` — a `gproduct-admin` scoped key (instances/credentials work with it on the test host)
- `TRUFO_ACCESS_TOKEN` + `TRUFO_REFRESH_TOKEN` — a Bearer token pair, e.g. minted via the device flow with a
  `trufo-api` scoped key (`trufo login --device` with the Python SDK); covers the product/certificate data
  sources and the AI system endpoints
- `TRUFO_TEST_GP_ID` — a test generator product created in the dashboard's Test view

## Repository & provider address

- Source: <https://github.com/NitriKx/terraform-provider-trufo>
- Provider address: `registry.terraform.io/nitrikx/trufo` (Terraform source: `nitrikx/trufo`)

## Release process

1. Push to `main` (branch protection requires green CI: `test` + `docs`).
2. [release-please](.github/workflows/release-please.yml) opens a "release PR" that bumps the version (from
   conventional commits) and updates `CHANGELOG.md`. Merging it creates the `vX.Y.Z` tag and GitHub release.
   The workflow uses the automatic `GITHUB_TOKEN` (`contents: write` + `pull-requests: write`).
   Note: tag pushes made with `GITHUB_TOKEN` do not trigger new workflow runs, so the `Release`
   (goreleaser) workflow is also runnable manually (`gh workflow run Release -f tag=vX.Y.Z`), or
   automatically if you give release-please a PAT instead: create a classic PAT with `repo` scope,
   store it as repo secret `RELEASE_PLEASE_TOKEN`, and pass `token: ${{ secrets.RELEASE_PLEASE_TOKEN }}`
   to the release-please action — then the tag push triggers the Release workflow by itself.
3. On the tag, the `Release` workflow runs [goreleaser](.goreleaser.yaml): builds the provider for
   linux/darwin/windows (amd64/arm64), writes `checksums.txt`, and attaches everything to the release.
4. Publish the version on the Terraform Registry: sign in at <https://registry.terraform.io>
   with the GitHub account, add provider `nitrikx/trufo`, and publish the release (the UI links
   the GitHub release and verifies checksums; `terraform-registry-manifest.json` is read from the repo).
   Only if you want to automate this step do you need a registry API token (profile → tokens).
5. CI (`CI` workflow) runs on every push/PR: build, `go vet`, `gofmt`, unit tests for both modules,
   `terraform fmt` check, and a docs freshness check. Acceptance tests run manually via `workflow_dispatch`
   with the Trufo secrets configured on the repo (`TRUFO_TEST_API_KEY`, `TRUFO_ACCESS_TOKEN`,
   `TRUFO_REFRESH_TOKEN`, `TRUFO_TEST_GP_ID`).

## Dependency updates (Renovate, auto-merge on green CI)

[Renovate](renovate.json) (the Mend Renovate GitHub App — install it at <https://github.com/apps/renovate>
for this repo) opens PRs for Go modules (both modules) and GitHub Actions. Minor/patch updates are
**auto-merged automatically once CI is green** (`automerge` + `platformAutomerge` in `renovate.json`);
major updates stay as PRs for review.

## License

MIT — see [LICENSE](./LICENSE).
