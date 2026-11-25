# Changelog

## [0.2.1](https://github.com/NitriKx/terraform-provider-trufo-ai/compare/v0.2.0...v0.2.1) (2026-11-25)


### Bug Fixes

* **release:** checkout the tag on manual dispatch so goreleaser sees it ([2f73ccb](https://github.com/NitriKx/terraform-provider-trufo-ai/commit/2f73ccb69425654f022ab7f41d65d1ba76ba8e0d))
* **release:** run goreleaser on the tag ref for manual dispatch ([a266d3c](https://github.com/NitriKx/terraform-provider-trufo-ai/commit/a266d3c041709e2aca4af7f198048e7cb8f9b873))

## [0.2.0](https://github.com/NitriKx/terraform-trufo-ai-provider/compare/v0.1.0...v0.2.0) (2026-11-25)


### Features

* initial Trufo Terraform provider with trufo-go client ([0a9e4e7](https://github.com/NitriKx/terraform-trufo-ai-provider/commit/0a9e4e761b001e1873b1ebe6aa5689f3078e7c11))

## 0.1.0 (unreleased)

- `trufo-go` client library (`github.com/NitriKx/terraform-trufo-ai-provider/trufo-go`), ported from
  [trufo-py](https://github.com/trufo-ai/trufo-py): API-key and Bearer auth with
  refresh-on-401, device (RFC 8628) and loopback (RFC 8252+PKCE) login flows,
  typed error envelope, and RA/AI-system endpoints.
- Provider `trufo` (`registry.terraform.io/nitrikx/trufo`) with:
  - `trufo_instance` resource (create/read/rename/delete, import by `gpi_id`)
  - `trufo_instance_credential` resource (key enrollment, label edit,
    delete = revoke, import as `{gpi_id}/{gpic_id}`)
  - `trufo_ai_system` resource (register, active/inactive toggle, delete = inactive)
  - `trufo_generator_product` and `trufo_certificates` data sources
