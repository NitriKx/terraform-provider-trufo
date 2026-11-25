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

func TestAccInstanceResource_basic(t *testing.T) {
	testAccPreCheck(t)
	name := "tf-acc-" + acctest.RandString(8)
	renamed := name + "-renamed"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "trufo_instance" "test" {
  gp_id = %q
  name  = %q
}
`, testGPID(), name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"trufo_instance.test",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(regexp.MustCompile(`^gpi_`)),
					),
					statecheck.ExpectKnownValue(
						"trufo_instance.test",
						tfjsonpath.New("gp_id"),
						knownvalue.StringExact(testGPID()),
					),
					statecheck.ExpectKnownValue(
						"trufo_instance.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
				},
			},
			// Update: rename.
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "trufo_instance" "test" {
  gp_id = %q
  name  = %q
}
`, testGPID(), renamed),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"trufo_instance.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(renamed),
					),
				},
			},
			// Import by gpi_id.
			{
				ResourceName:      "trufo_instance.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
