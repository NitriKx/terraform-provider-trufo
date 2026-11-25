# Changelog

## [0.2.0](https://github.com/NitriKx/terraform-provider-trufo-ai/compare/v0.1.0...v0.2.0) (2026-11-25)


### Features

* initial Trufo Terraform provider with trufo-go client ([0a9e4e7](https://github.com/NitriKx/terraform-provider-trufo-ai/commit/0a9e4e761b001e1873b1ebe6aa5689f3078e7c11))

## 0.1.0 (unreleased)

- `trufo-go` client library (`github.com/NitriKx/terraform-provider-trufo-ai/trufo-go`), ported from
  [trufo-py](https://github.com/trufo-ai/trufo-py): API-key and Bearer auth with
  refresh-on-401, device (RFC 8628) and loopback (RFC 8252+PKCE) login flows,
  typed error envelope, and RA/AI-system endpoints.
- Provider `trufo-ai` (`registry.terraform.io/nitrikx/trufo-ai`) with:
  - `trufo-ai_instance` resource (create/read/rename/delete, import by `gpi_id`)
  - `trufo-ai_instance_credential` resource (key enrollment, label edit,
    delete = revoke, import as `{gpi_id}/{gpic_id}`)
  - `trufo-ai_ai_system` resource (register, active/inactive toggle, delete = inactive)
  - `trufo-ai_generator_product` and `trufo-ai_certificates` data sources
