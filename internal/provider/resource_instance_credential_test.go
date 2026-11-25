package provider

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// importCredentialID builds the "{gpi_id}/{gpic_id}" import ID from state.
func importCredentialID(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}
		return rs.Primary.Attributes["gpi_id"] + "/" + rs.Primary.Attributes["id"], nil
	}
}

// generatePublicKeyPEM returns a fresh PEM-encoded EC P-256 public key.
func generatePublicKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func TestAccInstanceCredentialResource_basic(t *testing.T) {
	testAccPreCheck(t)
	instanceName := "tf-acc-cred-" + acctest.RandString(8)
	publicKeyPEM := generatePublicKeyPEM(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "trufo_instance" "test" {
  gp_id = %q
  name  = %q
}

resource "trufo_instance_credential" "test" {
  gpi_id         = trufo_instance.test.id
  public_key_pem = %q
  key_algorithm  = "ES256"
  label          = "tf-acc"
}
`, testGPID(), instanceName, publicKeyPEM),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"trufo_instance_credential.test",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(regexp.MustCompile(`^gpic_`)),
					),
					statecheck.ExpectKnownValue(
						"trufo_instance_credential.test",
						tfjsonpath.New("key_algorithm"),
						knownvalue.StringExact("ES256"),
					),
					statecheck.ExpectKnownValue(
						"trufo_instance_credential.test",
						tfjsonpath.New("revoked"),
						knownvalue.Bool(false),
					),
				},
			},
			// Import as "{gpi_id}/{gpic_id}".
			{
				ResourceName:      "trufo_instance_credential.test",
				ImportState:       true,
				ImportStateIdFunc: importCredentialID("trufo_instance_credential.test"),
				ImportStateVerify: true,
				// public_key_pem is not returned by the API; ignore it on import verify.
				ImportStateVerifyIgnore: []string{"public_key_pem"},
			},
		},
	})
}
