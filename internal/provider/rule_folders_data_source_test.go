// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRuleFoldersDataSource(t *testing.T) {
	profileName := testAccRandomName("tfacc-folders-ds-profile")
	folderName := testAccRandomName("tfacc-folders-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "controld_profile" "test" {
  name = %[1]q
}

resource "controld_rule_folder" "test" {
  profile_id = controld_profile.test.id
  name       = %[2]q
}

data "controld_rule_folders" "all" {
  profile_id = controld_profile.test.id

  depends_on = [controld_rule_folder.test]
}
`, profileName, folderName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.controld_rule_folders.all", "rule_folders.#"),
				),
			},
		},
	})
}
