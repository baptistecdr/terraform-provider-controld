// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRuleFolderResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRuleFolderResourceConfig("tf-acc-test-folder", 0, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_rule_folder.test", "name", "tf-acc-test-folder"),
					resource.TestCheckResourceAttr("controld_rule_folder.test", "do", "0"),
					resource.TestCheckResourceAttr("controld_rule_folder.test", "status", "true"),
					resource.TestCheckResourceAttrSet("controld_rule_folder.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "controld_rule_folder.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccRuleFolderImportStateIDFunc("controld_rule_folder.test"),
			},
			// Update and Read testing
			{
				Config: testAccRuleFolderResourceConfig("tf-acc-test-folder-renamed", 1, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_rule_folder.test", "name", "tf-acc-test-folder-renamed"),
					resource.TestCheckResourceAttr("controld_rule_folder.test", "do", "1"),
					resource.TestCheckResourceAttr("controld_rule_folder.test", "status", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// testAccRuleFolderImportStateIDFunc builds the "profile_id/folder_id"
// composite identifier controld_rule_folder expects on import.
func testAccRuleFolderImportStateIDFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["profile_id"], rs.Primary.ID), nil
	}
}

func testAccRuleFolderResourceConfig(name string, do int, status bool) string {
	return fmt.Sprintf(`
resource "controld_profile" "test" {
  name = "tf-acc-test-rule-folder-profile"
}

resource "controld_rule_folder" "test" {
  profile_id = controld_profile.test.id
  name       = %[1]q
  do         = %[2]d
  status     = %[3]t
}
`, name, do, status)
}
