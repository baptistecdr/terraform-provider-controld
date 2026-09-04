// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProxiesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "controld_proxies" "all" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.controld_proxies.all", "proxies.#"),
					resource.TestCheckResourceAttrSet("data.controld_proxies.all", "proxies.0.pk"),
					resource.TestCheckResourceAttrSet("data.controld_proxies.all", "countries.#"),
				),
			},
		},
	})
}
