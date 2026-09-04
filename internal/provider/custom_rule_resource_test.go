// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCustomRuleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCustomRuleResourceConfig(0, true, "blocked for testing"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_custom_rule.test", "hostname", "tf-acc-test.example.com"),
					resource.TestCheckResourceAttr("controld_custom_rule.test", "do", "0"),
					resource.TestCheckResourceAttr("controld_custom_rule.test", "status", "true"),
					resource.TestCheckResourceAttr("controld_custom_rule.test", "group", "0"),
					resource.TestCheckResourceAttr("controld_custom_rule.test", "comment", "blocked for testing"),
					resource.TestCheckResourceAttrSet("controld_custom_rule.test", "order"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "controld_custom_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccCustomRuleImportStateIDFunc("controld_custom_rule.test"),
			},
			// Update and Read testing
			{
				Config: testAccCustomRuleResourceConfig(1, false, "updated comment"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_custom_rule.test", "do", "1"),
					resource.TestCheckResourceAttr("controld_custom_rule.test", "status", "false"),
					resource.TestCheckResourceAttr("controld_custom_rule.test", "comment", "updated comment"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// testAccCustomRuleImportStateIDFunc builds the "profile_id/hostname"
// composite identifier controld_custom_rule expects on import.
func testAccCustomRuleImportStateIDFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["profile_id"], rs.Primary.Attributes["hostname"]), nil
	}
}

func testAccCustomRuleResourceConfig(do int, status bool, comment string) string {
	return fmt.Sprintf(`
resource "controld_profile" "test" {
  name = "tf-acc-test-custom-rule-profile"
}

resource "controld_custom_rule" "test" {
  profile_id = controld_profile.test.id
  hostname   = "tf-acc-test.example.com"
  do         = %[1]d
  status     = %[2]t
  comment    = %[3]q
}
`, do, status, comment)
}
