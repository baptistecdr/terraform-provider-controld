// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRuleFoldersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "controld_profile" "test" {
  name = "tfacc-folders-ds-profile"
}

resource "controld_rule_folder" "test" {
  profile_id = controld_profile.test.id
  name       = "tfacc-folders-ds"
}

data "controld_rule_folders" "all" {
  profile_id = controld_profile.test.id

  depends_on = [controld_rule_folder.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.controld_rule_folders.all", "rule_folders.#"),
				),
			},
		},
	})
}
