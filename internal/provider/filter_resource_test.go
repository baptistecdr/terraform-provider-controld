// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccFilterResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccFilterResourceConfig(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_filter.test", "filter", "malware"),
					resource.TestCheckResourceAttr("controld_filter.test", "status", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "controld_filter.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccFilterImportStateIDFunc("controld_filter.test"),
			},
			// Update and Read testing
			{
				Config: testAccFilterResourceConfig(false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_filter.test", "status", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// testAccFilterImportStateIDFunc builds the "profile_id/filter" composite
// identifier controld_filter expects on import.
func testAccFilterImportStateIDFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["profile_id"], rs.Primary.Attributes["filter"]), nil
	}
}

// testAccFilterResourceConfig targets "malware", a real native filter
// identifier confirmed against the live API.
func testAccFilterResourceConfig(status bool) string {
	return fmt.Sprintf(`
resource "controld_profile" "test" {
  name = "tfacc-filter-profile"
}

resource "controld_filter" "test" {
  profile_id = controld_profile.test.id
  filter     = "malware"
  status     = %[1]t
}
`, status)
}
