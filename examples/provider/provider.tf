terraform {
  required_providers {
    trufo = {
      source = "nitrikx/trufo-ai"
    }
  }
}

# Default: the global PRODUCTION API, https://api.trufo.ai.
# Credentials come from the environment — never hardcode them:
#   TRUFO_API_KEY                    API key (X-API-Key: {scope}:{key})
#   TRUFO_ACCESS_TOKEN + TRUFO_REFRESH_TOKEN   Bearer token pair
#   TRUFO_API_URL                   overrides the host
provider "trufo-ai" {}

# EU production API (https://eu.api.trufo.ai): media submitted there is
# processed and stored in EU infrastructure.
provider "trufo-ai" {
  alias  = "eu"
  region = "eu"
}

# Test API (https://test.api.trufo.ai): certificate test flow and test-scoped
# keys (e.g. c2pa-sign-test, gproduct-admin on the test host).
provider "trufo-ai" {
  alias  = "test"
  region = "test"
}

# A dedicated host, if your organization was provisioned one:
#
# provider "trufo-ai" {
#   alias   = "dedicated"
#   api_url = "https://your-host.api.trufo.ai"
# }

# A trufo-api scoped API key without tokens triggers the interactive login
# (loopback browser flow, falling back to the device flow):
#
# provider "trufo-ai" {
#   api_key   = var.trufo_api_key # trufo-api:...
#   auth_flow = "auto"            # auto | loopback | device | none
# }
