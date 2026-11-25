# Changelog

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
