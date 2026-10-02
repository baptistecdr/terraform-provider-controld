// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDefaultRuleResource(t *testing.T) {
	profileName := testAccRandomName("tfacc-default-rule")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDefaultRuleResourceConfig(profileName, 1, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_default_rule.test", "do", "1"),
					resource.TestCheckResourceAttr("controld_default_rule.test", "status", "true"),
					resource.TestCheckResourceAttrPair("controld_default_rule.test", "id", "controld_profile.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "controld_default_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccDefaultRuleResourceConfig(profileName, 0, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_default_rule.test", "do", "0"),
					resource.TestCheckResourceAttr("controld_default_rule.test", "status", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccDefaultRuleResourceConfig(profileName string, do int, status bool) string {
	return fmt.Sprintf(`
resource "controld_profile" "test" {
  name = %[1]q
}

resource "controld_default_rule" "test" {
  profile_id = controld_profile.test.id
  do         = %[2]d
  status     = %[3]t
}
`, profileName, do, status)
}
