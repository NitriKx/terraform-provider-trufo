# API-key auth against the test host (X-API-Key). The key comes from the
# TRUFO_API_KEY environment variable — never hardcode it.
provider "trufo" {
  region = "test"
}

# Bearer auth with a pre-obtained token pair (TRUFO_ACCESS_TOKEN /
# TRUFO_REFRESH_TOKEN), required for /gproduct/list and /cert/list:
#
# provider "trufo" {
#   region        = "global"
#   access_token  = var.trufo_access_token
#   refresh_token = var.trufo_refresh_token
# }

# A trufo-api scoped API key without tokens triggers the interactive login
# (loopback browser flow, falling back to the device flow):
#
# provider "trufo" {
#   api_key  = var.trufo_api_key # trufo-api:...
#   auth_flow = "auto"           # auto | loopback | device | none
# }
