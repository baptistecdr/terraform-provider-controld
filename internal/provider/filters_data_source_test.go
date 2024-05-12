// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFiltersDataSources(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "controld_profile" "test" {
  name = "tfacc-filters-ds-profile"
}

data "controld_native_filters" "all" {
  profile_id = controld_profile.test.id
}

data "controld_external_filters" "all" {
  profile_id = controld_profile.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.controld_native_filters.all", "filters.#"),
					resource.TestCheckResourceAttrSet("data.controld_native_filters.all", "filters.0.name"),
					resource.TestCheckResourceAttrSet("data.controld_external_filters.all", "filters.#"),
				),
			},
		},
	})
}
