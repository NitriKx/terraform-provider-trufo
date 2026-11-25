# Terraform Provider for Trufo

[![CI](https://github.com/NitriKx/terraform-provider-trufo-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/NitriKx/terraform-provider-trufo-ai/actions/workflows/ci.yml)

Terraform provider for the [Trufo](https://trufo.ai) Provenance Platform, built on
[`github.com/NitriKx/terraform-provider-trufo-ai/trufo-go`](./trufo-go) — a Go client ported from the official
[Python SDK (trufo-py)](https://github.com/trufo-ai/trufo-py).

Provider address: `registry.terraform.io/nitrikx/trufo-ai`

## Resources & data sources

| Type | API | Notes |
| --- | --- | --- |
| `trufo-ai_instance` | `POST /gproduct/instance/*` | Full CRUD; import by `gpi_…` |
| `trufo-ai_instance_credential` | `POST /gproduct/instance/credential/*` | Enroll a public key; delete = revoke; import as `{gpi_id}/{gpic_id}` |
| `trufo-ai_ai_system` | `POST /ai-system/*` | Register + active/inactive toggle; delete = inactive; production hosts only |
| `trufo-ai_generator_product` (data source) | `POST /gproduct/list`, `/gproduct/info` | Products are created in the dashboard; Bearer auth |
| `trufo-ai_certificates` (data source) | `POST /cert/list` | Bearer auth |

## Authentication

Every Trufo API request carries exactly one credential, and so does the provider:

- **API key** — `api_key` attribute or `TRUFO_API_KEY` env var (`X-API-Key: {scope}:{key}`). A `gproduct-admin`
  key manages instances/credentials; `c2pa-sign-*`/`watermark-*` keys cover AI systems (production hosts).
- **Access token** — `access_token` + `refresh_token` attributes or `TRUFO_ACCESS_TOKEN` / `TRUFO_REFRESH_TOKEN`
  env vars (`Authorization: Bearer`), with automatic refresh on 401. Required for the product and certificate
  data sources. If only a `trufo-api`-scoped `api_key` is configured, the provider runs the interactive login
  (loopback browser flow, or device flow — see `auth_flow`) at configure time.

## Regions and hosts

The provider defaults to the **global production API** (`https://api.trufo.ai`). Other hosts via `region` or a
full `api_url` (or the `TRUFO_API_URL` env var):

| Host | `region` | Use |
| --- | --- | --- |
| `https://api.trufo.ai` | `global` (default) | Production, global |
| `https://eu.api.trufo.ai` | `eu` | Production, EU — media submitted there is processed and stored in EU infrastructure |
| `https://test.api.trufo.ai` | `test` | Test API — certificate test flow and test-scoped keys; nothing carries over to production |

```hcl
provider "trufo-ai" {} # production, global (default)

provider "trufo-ai" {
  alias  = "eu"
  region = "eu"
}

provider "trufo-ai" {
  alias  = "test"
  region = "test"
}
```

```hcl
terraform {
  required_providers {
    trufo = {
      source = "nitrikx/trufo-ai"
    }
  }
}

provider "trufo-ai" {
  region = "test" # TRUFO_API_KEY env var provides the API key
}

resource "trufo-ai_instance" "signing_service" {
  gp_id = "gp_..."
  name  = "signing-service-eu-1"
}

resource "trufo-ai_instance_credential" "primary" {
  gpi_id         = trufo-ai_instance.signing_service.id
  public_key_pem = file("instance-public-key.pem")
  key_algorithm  = "ES256"
  label          = "primary-2026"
}
```

More examples in [`examples/`](./examples); generated reference docs in [`docs/`](./docs).

## Development

Requirements: Go 1.27+, Terraform 1.16+.

```sh
go build ./...                 # build provider + client
go test ./...                  # unit tests (client uses a mock server)
TF_ACC=1 TRUFO_API_KEY=... TRUFO_TEST_GP_ID=gp_... go test ./internal/provider/ -run TestAcc -v
                               # acceptance tests against test.api.trufo.ai
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name trufo-ai   # regenerate docs/
```

Acceptance tests run against `https://test.api.trufo.ai` (override with `TRUFO_TEST_API_URL`; AI system tests
use `https://api.trufo.ai`, override with `TRUFO_PROD_API_URL`) and need:

- `TRUFO_API_KEY` — a `gproduct-admin` scoped key (instances/credentials work with it on the test host)
- `TRUFO_ACCESS_TOKEN` + `TRUFO_REFRESH_TOKEN` — a Bearer token pair, e.g. minted via the device flow with a
  `trufo-api` scoped key (`trufo login --device` with the Python SDK); covers the product/certificate data
  sources and the AI system endpoints
- `TRUFO_TEST_GP_ID` — a test generator product created in the dashboard's Test view

## Repository & provider address

- Source: <https://github.com/NitriKx/terraform-provider-trufo-ai>
- Provider address: `registry.terraform.io/nitrikx/trufo-ai` (Terraform source: `nitrikx/trufo-ai`)

## Release process

1. Create the GitHub repo `NitriKx/terraform-provider-trufo-ai` and push. You need a one-off
   credential for this (a PAT with `repo` scope, or SSH) — the API being down only affects
   `gh repo create`; creating the repo in the web UI and pushing works the same:
   `git remote add origin https://github.com/NitriKx/terraform-provider-trufo-ai.git && git push -u origin main`
2. Releases are automated with [release-please](.github/workflows/release-please.yml): on every
   push to `main` it opens a "release PR" that bumps the version (from conventional commits) and
   updates `CHANGELOG.md`. Merging that PR creates the `vX.Y.Z` tag and GitHub release.
   The workflow uses the automatic `GITHUB_TOKEN` (`contents: write` + `pull-requests: write`).
   Note: tag pushes made with `GITHUB_TOKEN` do not trigger new workflow runs, so the `Release`
   (goreleaser) workflow is also runnable manually (`gh workflow run Release -f tag=vX.Y.Z`), or
   automatically if you give release-please a PAT instead: create a classic PAT with `repo` scope,
   store it as repo secret `RELEASE_PLEASE_TOKEN`, and pass `token: ${{ secrets.RELEASE_PLEASE_TOKEN }}`
   to the release-please action — then the tag push triggers the Release workflow by itself.
3. On the tag, the `Release` workflow runs [goreleaser](.goreleaser.yaml): builds the provider for
   linux/darwin/windows (amd64/arm64), writes `checksums.txt`, and attaches everything to the
   release.
4. Publish the version on the Terraform Registry: sign in at <https://registry.terraform.io>
   with the GitHub account, add provider `nitrikx/trufo-ai`, and publish the release (the UI links
   the GitHub release and verifies checksums). Only if you want to automate this step do you
   need a registry API token (profile → tokens at registry.terraform.io).
5. CI (`CI` workflow) runs on every push/PR: build, `go vet`, `gofmt`, unit tests for both modules,
   and a docs freshness check. Acceptance tests run manually via `workflow_dispatch` with the
   Trufo secrets configured on the repo (`TRUFO_TEST_API_KEY`, `TRUFO_ACCESS_TOKEN`,
   `TRUFO_REFRESH_TOKEN`, `TRUFO_TEST_GP_ID`).

## Dependency updates (Renovate, auto-merge on green CI)

[Renovate](renovate.json) (the Mend Renovate GitHub App — install it at <https://github.com/apps/renovate>
for this repo) opens PRs for Go modules (both modules) and GitHub Actions. Minor/patch updates are
**auto-merged automatically once CI is green** (`automerge` + `platformAutomerge` in `renovate.json`);
major updates stay as PRs for review. For platform auto-merge to work, the repo needs:

- Settings → General → "Allow auto-merge" enabled (set via API: `gh api -X PATCH repos/NitriKx/terraform-provider-trufo-ai -F allow_auto_merge=true`)
- Branch protection on `main` requiring the CI status checks (`test`, `docs`) and disallowing force-pushes

## License

MIT — see [LICENSE](./LICENSE).
