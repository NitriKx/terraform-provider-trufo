
---
page_title: "Trufo Provider"
description: |-
  The Trufo provider manages Trufo Provenance Platform resources (generator product instances,
  instance credentials, AI systems, certificates) through the Trufo API.
---

# Trufo Provider

The Trufo provider manages resources of the [Trufo](https://trufo.ai) Provenance Platform through the Trufo API:

- **Generator product instances** (`trufo_instance`) — one signing identity per independent key holder
- **Instance credentials** (`trufo_instance_credential`) — public-key enrollment on an instance (delete = revoke)
- **AI systems** (`trufo_ai_system`) — registered AI systems referenced when signing and watermarking
- **Read-only lookups** — `trufo_generator_product` and `trufo_certificates` data sources

The API client used by the provider is [`github.com/NitriKx/terraform-provider-trufo/trufo-go`](./trufo-go), a Go port of the official
[Python SDK](https://github.com/trufo-ai/trufo-py).

## Regions and hosts

The provider defaults to the **global production API** (`https://api.trufo.ai`). Pick another host with `region`
or a full `api_url` (or the `TRUFO_API_URL` environment variable):

| Host | `region` | Use |
| --- | --- | --- |
| `https://api.trufo.ai` | `global` (default) | Production, global |
| `https://eu.api.trufo.ai` | `eu` | Production, EU — media submitted there is processed and stored in EU infrastructure |
| `https://test.api.trufo.ai` | `test` | Test API — certificate test flow and test-scoped keys; nothing carries over to production |

```terraform
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

## Authentication

Every API request carries exactly one credential. The provider supports both, like the Python SDK:

- **API key** — `api_key` attribute or `TRUFO_API_KEY` env var, sent as `X-API-Key: {scope}:{key}`.
  A `gproduct-admin` scoped key manages instances and credentials; `c2pa-sign-*`/`watermark-*` keys cover the
  AI system endpoints (production hosts only).
- **Access token** — `access_token` + `refresh_token` (env `TRUFO_ACCESS_TOKEN` / `TRUFO_REFRESH_TOKEN`),
  sent as `Authorization: Bearer`, refreshed automatically on 401. Required for `trufo_generator_product` and
  `trufo_certificates`. If only a `trufo-api` scoped `api_key` is set, the provider runs the interactive login
  at configure time (`auth_flow`: `auto`/`loopback`/`device`/`none`).

## Example usage

```terraform
terraform {
  required_providers {
    trufo = {
      source = "nitrikx/trufo"
    }
  }
}

# Default: the global PRODUCTION API, https://api.trufo.ai.
# Credentials come from the environment — never hardcode them:
#   TRUFO_API_KEY                    API key (X-API-Key: {scope}:{key})
#   TRUFO_ACCESS_TOKEN + TRUFO_REFRESH_TOKEN   Bearer token pair
#   TRUFO_API_URL                   overrides the host
provider "trufo" {}

# EU production API (https://eu.api.trufo.ai): media submitted there is
# processed and stored in EU infrastructure.
provider "trufo" {
  alias  = "eu"
  region = "eu"
}

# Test API (https://test.api.trufo.ai): certificate test flow and test-scoped
# keys (e.g. c2pa-sign-test, gproduct-admin on the test host).
provider "trufo" {
  alias  = "test"
  region = "test"
}

# A dedicated host, if your organization was provisioned one:
#
# provider "trufo" {
#   alias   = "dedicated"
#   api_url = "https://your-host.api.trufo.ai"
# }

# A trufo-api scoped API key without tokens triggers the interactive login
# (loopback browser flow, falling back to the device flow):
#
# provider "trufo" {
#   api_key   = var.trufo_api_key # trufo-api:...
#   auth_flow = "auto"            # auto | loopback | device | none
# }
```

## Full example

A complete C2PA signing instance with an enrolled key — generator product lookup by name, instance
creation, and ES256 (EC P-256) public-key enrollment (see
[`examples/full-signing-instance/`](https://github.com/NitriKx/terraform-provider-trufo/tree/main/examples/full-signing-instance)):

```terraform
terraform {
  required_providers {
    trufo = {
      source = "nitrikx/trufo"
    }
  }
}

# Default: the global PRODUCTION API, https://api.trufo.ai.
# Pick another host with `region` or a full `api_url` (or the TRUFO_API_URL env var):
#   region  = "eu"    # https://eu.api.trufo.ai — EU production, media processed/stored in EU infrastructure
#   region  = "test"  # https://test.api.trufo.ai — test API, test-scoped keys, nothing carries over to production
#   api_url = "https://your-host.api.trufo.ai" # a dedicated host, if your org was provisioned one
#
# Credentials come from the environment — never hardcode them in config or state:
#   TRUFO_API_KEY                             API key, sent as X-API-Key: {scope}:{key}
#                                             (a gproduct-admin key manages instances/credentials)
#   TRUFO_ACCESS_TOKEN + TRUFO_REFRESH_TOKEN  Bearer token pair (required for the generator product
#                                             data source below; refreshed automatically on 401)
provider "trufo" {
  # region = "eu"
}

# Generator products are created in the Trufo dashboard (as part of a Product
# Validation application); look yours up here by ID (gp_...) or by name — set
# exactly one. Requires an active Product Validation on the product before
# instances can be created.
data "trufo_generator_product" "c2pa" {
  gp_id = var.generator_product_id != "" ? var.generator_product_id : null
  name  = var.generator_product_id == "" ? var.generator_product_name : null
}

# One signing instance per signing identity: it holds its own credentials and
# is issued its own certificates.
resource "trufo_instance" "signing" {
  gp_id = data.trufo_generator_product.c2pa.id
  name  = var.instance_name
}

# Enroll the PUBLIC key of an ES256 (EC P-256) key pair on the instance.
#
# Generate the key pair once, locally — keep the private key out of Terraform
# entirely (never in variables, tfvars, state, or VCS; *.pem is gitignored):
#
#   openssl ecparam -name prime256v1 -genkey -noout -out instance-key.pem
#   openssl ec -in instance-key.pem -pubout -out instance-public-key.pem
#
# Only the public key (instance-public-key.pem) is enrolled here. The private
# key (instance-key.pem) never goes into Terraform or its state — it stays on
# your signing systems and is only used later to sign the CSR-JWT client
# assertion that requests certificate enrollment from the Trufo RA
# (POST /ra/c2pa/csr-jwt).
#
# An instance may hold at most TWO active credentials. To rotate a key: apply
# with the new credential added (as a second trufo_instance_credential), cut
# your signer over to the new key, then remove the old resource from the
# config and apply again.
#
# Delete semantics: removing this resource from the config (or `terraform
# destroy`) REVOKES the credential on the server — revocation is permanent.
resource "trufo_instance_credential" "primary" {
  gpi_id         = trufo_instance.signing.id
  public_key_pem = file(var.credential_public_key_path)
  key_algorithm  = "ES256" # EC P-256; EdDSA (Ed25519) is also accepted
  label          = var.credential_label
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `access_token` (String, Sensitive) Access token (Bearer auth), e.g. obtained via the device flow. Default: the `TRUFO_ACCESS_TOKEN` environment variable. Requires `refresh_token`.
- `api_key` (String, Sensitive) Trufo API key in `{scope}:{key}` form, sent as `X-API-Key`. Default: the `TRUFO_API_KEY` environment variable. A key with the `trufo-api` scope is used to run the interactive login flow (see `auth_flow`) when no tokens are configured.
- `api_url` (String) Trufo API base URL, e.g. `https://api.trufo.ai` (production), `https://eu.api.trufo.ai` (EU production) or `https://test.api.trufo.ai` (test). Overrides `region`; use it for dedicated hosts. Default: `https://api.trufo.ai` (production), or the `TRUFO_API_URL` environment variable.
- `auth_flow` (String) How to obtain access/refresh tokens when only a `trufo-api`-scoped `api_key` is set: `auto` (loopback browser flow, falling back to the device flow), `loopback` (browser on this machine), `device` (prints a verification URL and code) or `none` (use the API key directly). Default: `auto`.
- `refresh_token` (String, Sensitive) Refresh token, rotated on every use; the provider refreshes the access token automatically on 401. Default: the `TRUFO_REFRESH_TOKEN` environment variable. Requires `access_token`.
- `region` (String) Content region: `global` (`https://api.trufo.ai`, **default**), `eu` (`https://eu.api.trufo.ai`, EU production — media submitted there is processed and stored in EU infrastructure) or `test` (`https://test.api.trufo.ai`, test API with test-scoped keys; nothing carries over to production). Ignored when `api_url` is set.


