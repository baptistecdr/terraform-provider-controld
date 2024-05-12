// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccServiceResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccServiceResourceConfig(1, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_service.test", "service", "netflix"),
					resource.TestCheckResourceAttr("controld_service.test", "do", "1"),
					resource.TestCheckResourceAttr("controld_service.test", "status", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "controld_service.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccServiceImportStateIDFunc("controld_service.test"),
			},
			// Update and Read testing
			{
				Config: testAccServiceResourceConfig(0, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_service.test", "do", "0"),
					resource.TestCheckResourceAttr("controld_service.test", "status", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// testAccServiceImportStateIDFunc builds the "profile_id/service" composite
// identifier controld_service expects on import.
func testAccServiceImportStateIDFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["profile_id"], rs.Primary.Attributes["service"]), nil
	}
}

// testAccServiceResourceConfig targets "netflix", a real ControlD service
// catalog identifier confirmed against the live API. The catalog itself
// isn't part of this provider's schema, so it can't be looked up dynamically.
func testAccServiceResourceConfig(do int, status bool) string {
	return fmt.Sprintf(`
resource "controld_profile" "test" {
  name = "tfacc-service-profile"
}

resource "controld_service" "test" {
  profile_id = controld_profile.test.id
  service    = "netflix"
  do         = %[1]d
  status     = %[2]t
}
`, do, status)
}
