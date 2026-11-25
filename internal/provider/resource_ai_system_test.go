package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccAISystemResource_basic exercises the AI system endpoints on the
// production hosts with a Bearer token (c2pa_sign permission). The resource
// is destroyed (set inactive) at the end of the test.
func TestAccAISystemResource_basic(t *testing.T) {
	testAccBearerPreCheck(t)
	name := "tf-acc-aisys-" + acctest.RandString(8)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBearerProviderConfig("") + fmt.Sprintf(`
resource "trufo_ai_system" "test" {
  name                 = %q
  version              = "0.0.1"
  ai_involvement_modes = ["ai_generated"]
  nickname             = "tf-acc"
  status               = "active"
}
`, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"trufo_ai_system.test",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(regexp.MustCompile(`^aisys_`)),
					),
					statecheck.ExpectKnownValue(
						"trufo_ai_system.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("active"),
					),
				},
			},
			// Update: toggle status to inactive and back.
			{
				Config: testAccBearerProviderConfig("") + fmt.Sprintf(`
resource "trufo_ai_system" "test" {
  name                 = %q
  version              = "0.0.1"
  ai_involvement_modes = ["ai_generated"]
  nickname             = "tf-acc"
  status               = "inactive"
}
`, name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"trufo_ai_system.test",
						tfjsonpath.New("status"),
						knownvalue.StringExact("inactive"),
					),
				},
			},
			{
				ResourceName:      "trufo_ai_system.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
