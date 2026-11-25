# Terraform Provider for Trufo

[![CI](https://github.com/mistral/terraform-trufo-ai-provider/actions/workflows/ci.yml/badge.svg)](https://github.com/mistral/terraform-trufo-ai-provider/actions/workflows/ci.yml)

Terraform provider for the [Trufo](https://trufo.ai) Provenance Platform, built on
[`github.com/mistral/trufo-go`](./trufo-go) — a Go client ported from the official
[Python SDK (trufo-py)](https://github.com/trufo-ai/trufo-py).

Provider address: `registry.terraform.io/mistral/trufo`

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

- **API key** — `api_key` attribute or `TRUFO_API_KEY` env var (`X-API-Key: {scope}:{key}`). A `gproduct-admin`
  key manages instances/credentials; `c2pa-sign-*`/`watermark-*` keys cover AI systems (production hosts).
- **Access token** — `access_token` + `refresh_token` attributes or `TRUFO_ACCESS_TOKEN` / `TRUFO_REFRESH_TOKEN`
  env vars (`Authorization: Bearer`), with automatic refresh on 401. Required for the product and certificate
  data sources. If only a `trufo-api`-scoped `api_key` is configured, the provider runs the interactive login
  (loopback browser flow, or device flow — see `auth_flow`) at configure time.

Region/host: `region = "global" | "eu" | "test"` or `api_url`, plus `TRUFO_API_URL`.

```hcl
terraform {
  required_providers {
    trufo = {
      source = "mistral/trufo"
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

Requirements: Go 1.27+, Terraform 1.16+.

```sh
go build ./...                 # build provider + client
go test ./...                  # unit tests (client uses a mock server)
TF_ACC=1 TRUFO_API_KEY=... TRUFO_TEST_GP_ID=gp_... go test ./internal/provider/ -run TestAcc -v
                               # acceptance tests against test.api.trufo.ai
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs   # regenerate docs/
```

Acceptance tests run against `https://test.api.trufo.ai` (override with `TRUFO_TEST_API_URL`) and need:

- `TRUFO_API_KEY` — a `gproduct-admin` scoped key (instances/credentials work with it on the test host)
- `TRUFO_TEST_GP_ID` — a test generator product created in the dashboard's Test view

The `trufo_ai_system` resource and the `trufo_generator_product`/`trufo_certificates` data sources need
production hosts and a `c2pa-sign-*`/`watermark-*` key or a Bearer token, so they are covered by unit tests
only.

## License

Apache-2.0 — see [LICENSE](./LICENSE).
