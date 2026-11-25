# Security Policy

## Reporting a vulnerability

Please report security vulnerabilities **privately** via
[GitHub Security Advisories](https://github.com/NitriKx/terraform-provider-trufo/security/advisories/new).
Do not open public issues for vulnerabilities.

We aim to acknowledge reports within 72 hours.

## Scope notes

This provider handles Trufo API keys (`X-API-Key`) and OAuth access/refresh tokens (`Bearer`):

- Credentials are only read from provider attributes, the `TRUFO_API_KEY` / `TRUFO_ACCESS_TOKEN` /
  `TRUFO_REFRESH_TOKEN` environment variables, or the `~/.trufo/session` file (written with `0600`,
  mirroring the trufo-py CLI). They are never logged, never written to Terraform state (marked
  sensitive), and never sent anywhere except the configured Trufo API host.
- The loopback login flow binds `127.0.0.1` only, uses PKCE (S256) and a constant-time state check.
- TLS 1.3+ is enforced on the default HTTP transport; plaintext HTTP base URLs are rejected
  (except loopback test servers).

If you believe you found a way to leak credentials, bypass the single-credential rule, or reach a
non-Trufo host, that is a vulnerability — please report it privately as above.

## Supported versions

Security fixes are applied to the latest release only.
