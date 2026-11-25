# Changelog

## 1.0.0 (2026-11-25)


### Features

* initial Trufo Terraform provider (trufo-go client, trufo provider) ([3669d87](https://github.com/NitriKx/terraform-provider-trufo/commit/3669d87b814a973ca793256fbd441ee8f0bc7e09))
* initial Trufo Terraform provider with trufo-go client ([0a9e4e7](https://github.com/NitriKx/terraform-provider-trufo/commit/0a9e4e761b001e1873b1ebe6aa5689f3078e7c11))

## 0.1.0 (unreleased)

- `trufo-go` client library (`github.com/NitriKx/terraform-provider-trufo/trufo-go`), ported from
  [trufo-py](https://github.com/trufo-ai/trufo-py): API-key and Bearer auth with refresh-on-401, device
  (RFC 8628) and loopback (RFC 8252+PKCE) login flows, typed error envelope, RA endpoints
  (gproduct/instance/credential/cert), AI systems, tasks.
- Provider `trufo` (`registry.terraform.io/nitrikx/trufo`) with:
  - `trufo_instance` resource (create/read/rename/delete, import by `gpi_id`)
  - `trufo_instance_credential` resource (key enrollment, label edit, delete = revoke, import as `{gpi_id}/{gpic_id}`)
  - `trufo_ai_system` resource (register, active/inactive toggle, delete = inactive)
  - `trufo_generator_product` and `trufo_certificates` data sources
