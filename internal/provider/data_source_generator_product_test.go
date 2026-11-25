package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccGeneratorProductDataSource_byID(t *testing.T) {
	testAccBearerPreCheck(t)
	if testGPID() == "" {
		t.Skip("TRUFO_TEST_GP_ID must be set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBearerProviderConfig(testAPIURL()) + fmt.Sprintf(`
data "trufo-ai_generator_product" "test" {
  gp_id = %q
}
`, testGPID()),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.trufo-ai_generator_product.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(testGPID()),
					),
					statecheck.ExpectKnownValue(
						"data.trufo-ai_generator_product.test",
						tfjsonpath.New("gp_id"),
						knownvalue.StringRegexp(regexp.MustCompile(`^gp_`)),
					),
					statecheck.ExpectKnownValue(
						"data.trufo-ai_generator_product.test",
						tfjsonpath.New("oid"),
						knownvalue.StringRegexp(regexp.MustCompile(`^org_`)),
					),
				},
			},
		},
	})
}
