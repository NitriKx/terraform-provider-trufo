terraform {
  required_providers {
    trufo = {
      source = "nitrikx/trufo-ai"
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
provider "trufo-ai" {
  # region = "eu"
}

# Generator products are created in the Trufo dashboard (as part of a Product
# Validation application); look yours up here by ID (gp_...) or by name — set
# exactly one. Requires an active Product Validation on the product before
# instances can be created.
data "trufo-ai_generator_product" "c2pa" {
  gp_id = var.generator_product_id != "" ? var.generator_product_id : null
  name  = var.generator_product_id == "" ? var.generator_product_name : null
}

# One signing instance per signing identity: it holds its own credentials and
# is issued its own certificates.
resource "trufo-ai_instance" "signing" {
  gp_id = data.trufo-ai_generator_product.c2pa.id
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
# with the new credential added (as a second trufo-ai_instance_credential), cut
# your signer over to the new key, then remove the old resource from the
# config and apply again.
#
# Delete semantics: removing this resource from the config (or `terraform
# destroy`) REVOKES the credential on the server — revocation is permanent.
resource "trufo-ai_instance_credential" "primary" {
  gpi_id         = trufo-ai_instance.signing.id
  public_key_pem = file(var.credential_public_key_path)
  key_algorithm  = "ES256" # EC P-256; EdDSA (Ed25519) is also accepted
  label          = var.credential_label
}
