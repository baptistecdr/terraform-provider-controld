// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProfileDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "controld_profile" "test" {
  name = "tf-acc-test-profile-data-source"
}

data "controld_profile" "by_id" {
  id = controld_profile.test.id
}

data "controld_profile" "by_name" {
  name = controld_profile.test.name
}

data "controld_profiles" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.controld_profile.by_id", "name", "controld_profile.test", "name"),
					resource.TestCheckResourceAttrPair("data.controld_profile.by_name", "id", "controld_profile.test", "id"),
					resource.TestCheckResourceAttrSet("data.controld_profiles.all", "profiles.#"),
				),
			},
		},
	})
}
