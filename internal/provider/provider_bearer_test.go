package provider

import (
	"os"
	"testing"
)

// testAccBearerPreCheck skips Bearer-mode acceptance tests unless TF_ACC is
// set and a token pair is present in the environment (obtained e.g. via the
// device flow with a trufo-api key).
func testAccBearerPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1 is set")
	}
	if os.Getenv("TRUFO_ACCESS_TOKEN") == "" || os.Getenv("TRUFO_REFRESH_TOKEN") == "" {
		t.Skip("TRUFO_ACCESS_TOKEN and TRUFO_REFRESH_TOKEN must be set to run Bearer-mode acceptance tests")
	}
}

func testProdAPIURL() string {
	if v := os.Getenv("TRUFO_PROD_API_URL"); v != "" {
		return v
	}
	return "https://api.trufo.ai"
}

// testAccBearerProviderConfig returns a provider block using the Bearer token
// pair from the environment, against the given API URL (default prod).
func testAccBearerProviderConfig(apiURL string) string {
	if apiURL == "" {
		apiURL = testProdAPIURL()
	}
	return `
provider "trufo-ai" {
  api_url = "` + apiURL + `"
}
`
}
