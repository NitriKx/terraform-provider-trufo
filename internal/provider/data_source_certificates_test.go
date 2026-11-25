package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccCertificatesDataSource_list(t *testing.T) {
	testAccBearerPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBearerProviderConfig(testAPIURL()) + `
data "trufo-ai_certificates" "test" {}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					// id is the certificate count; just check it is a number string.
					statecheck.ExpectKnownValue(
						"data.trufo-ai_certificates.test",
						tfjsonpath.New("id"),
						knownvalue.StringRegexp(regexp.MustCompile(`^\d+$`)),
					),
				},
			},
		},
	})
}
