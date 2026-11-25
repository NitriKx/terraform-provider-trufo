# Full example: C2PA signing instance with an enrolled key

End-to-end setup of a Trufo C2PA signing instance:

1. Look up a generator product by name (`data "trufo_generator_product"`) — products are
   created in the Trufo dashboard as part of a Product Validation application.
2. Create a signing instance under it (`trufo_instance`).
3. Enroll the **public** key of an ES256 (EC P-256) key pair on the instance
   (`trufo_instance_credential`).

## Authentication

Export exactly one credential before running Terraform (never hardcode it):

```sh
# API key (X-API-Key: {scope}:{key}) — a gproduct-admin key manages instances/credentials
export TRUFO_API_KEY='gproduct-admin:...'

# OR a Bearer token pair — required for the trufo_generator_product data source
export TRUFO_ACCESS_TOKEN='...'
export TRUFO_REFRESH_TOKEN='...'
```

The provider defaults to the global production API (`https://api.trufo.ai`). Use
`region = "eu"` / `region = "test"`, `api_url`, or `TRUFO_API_URL` for another host.

## Generate the signing key pair

The private key **never** goes into Terraform — not in variables, tfvars, state, or VCS
(`*.pem` is gitignored). It stays on your signing system and is only used later to sign
the CSR-JWT client assertion that requests certificate enrollment (`POST /ra/c2pa/csr-jwt`):

```sh
openssl ecparam -name prime256v1 -genkey -noout -out instance-key.pem        # private key — keep secret
openssl ec -in instance-key.pem -pubout -out instance-public-key.pem         # public key — enroll this
```

Only `instance-public-key.pem` is referenced by the Terraform config (via
`var.credential_public_key_path`, default `instance-public-key.pem`).

## Run

```sh
cd examples/full-signing-instance

terraform init

terraform apply \
  -var 'generator_product_name=my-generator-product' \
  -var 'instance_name=c2pa-signing-1' \
  -var 'credential_public_key_path=instance-public-key.pem' \
  -var 'credential_label=primary-2026-11'
```

Or put the values in a `terraform.tfvars` (gitignored here) and run plain `terraform apply`.

## Variables

| Variable | Description | Default |
| --- | --- | --- |
| `generator_product_name` | Generator product name looked up via `trufo_generator_product` (required) | — |
| `instance_name` | Name of the `trufo_instance` signing instance | `c2pa-signing-1` |
| `credential_public_key_path` | Path to the PEM public key to enroll (ES256 / EC P-256) | `instance-public-key.pem` |
| `credential_label` | Label of the `trufo_instance_credential` | `primary-2026-11` |

## Outputs

`generator_product_id`, `instance_id`, `instance_name`, `credential_id`,
`credential_label`, `credential_revoked`.

## Key rotation and revocation

- An instance may hold at most **two active credentials**. To rotate: add a second
  `trufo_instance_credential` (new key pair, new label), apply, cut your signer over to
  the new key, then remove the old credential resource and apply again.
- **Deleting a `trufo_instance_credential` resource revokes the credential** on the server
  (`terraform destroy` included) — revocation is permanent.
- Import an existing credential with `terraform import trufo_instance_credential.primary '{gpi_id}/{gpic_id}'`.
