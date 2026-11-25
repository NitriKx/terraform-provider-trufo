# Contributing

Thanks for contributing to the Trufo Terraform provider!

## Reporting vulnerabilities

Please **do not** open public issues for security vulnerabilities. Report them privately via
[GitHub Security Advisories](https://github.com/NitriKx/terraform-provider-trufo/security/advisories/new)
(see [SECURITY.md](./SECURITY.md)). This provider handles Trufo API keys and Bearer tokens — treat
anything touching credentials, the login flows (device/loopback), or state handling as
security-sensitive.

## Workflow

1. Fork the repo and create a branch from `main`.
2. Write conventional commits (`feat:`, `fix:`, `docs:`, `chore:`, …) — release-please uses them
   to version releases.
3. Add/adjust tests: unit tests for `trufo-go` use an `httptest` mock server; provider acceptance
   tests (`TF_ACC=1`, see README) run against the Trufo **test** host only.
4. Run `make generate` if you changed schemas, descriptions, templates or examples — CI checks that
   `docs/` is up to date and that examples are `terraform fmt`-clean.
5. Open a PR; CI (build, vet, gofmt, unit tests, docs freshness) must be green. Renovate auto-merges
   dependency PRs once CI is green.

## Code conventions

- `trufo-go` mirrors the Python SDK (`trufo-py`) semantics: one credential per request, refresh
  tokens are single-use and rotated, errors are `*trufo.APIError` with the server's `detail` code.
  When in doubt, match `trufo-py` behavior and reference it in a comment.
- Never log or persist credentials; the session file `~/.trufo/session` is written 0600 only.
