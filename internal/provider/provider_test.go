package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProviderFactories wires the provider under test for acceptance tests.
var testAccProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"trufo": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck skips acceptance tests unless TF_ACC is set and the Trufo
// test credentials are present in the environment. The API key is read from
// TRUFO_API_KEY (never hardcoded); TRUFO_TEST_API_URL defaults to the test
// host and TRUFO_TEST_GP_ID to a test generator product.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1 is set")
	}
	if os.Getenv("TRUFO_API_KEY") == "" {
		t.Skip("TRUFO_API_KEY must be set (a gproduct-admin scoped key) to run acceptance tests")
	}
	if testAPIURL() == "" {
		t.Skip("TRUFO_TEST_API_URL must be set")
	}
	if testGPID() == "" {
		t.Skip("TRUFO_TEST_GP_ID must be set (a test generator product from the dashboard Test view)")
	}
}

func testAPIURL() string {
	if v := os.Getenv("TRUFO_TEST_API_URL"); v != "" {
		return v
	}
	return "https://test.api.trufo.ai"
}

func testGPID() string { return os.Getenv("TRUFO_TEST_GP_ID") }

// testAccProviderConfig returns a provider block pointing at the test host.
// Credentials come from the environment (TRUFO_API_KEY).
func testAccProviderConfig() string {
	return `
provider "trufo" {
  api_url = "` + testAPIURL() + `"
}
`
}
