// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServicesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// ListProfileServices only returns services with an explicit
				// action configured on the profile, so a freshly created
				// profile starts with an empty list: configure one first.
				Config: `
resource "controld_profile" "test" {
  name = "tfacc-services-ds-profile"
}

resource "controld_service" "netflix" {
  profile_id = controld_profile.test.id
  service    = "netflix"
  do         = 1
  status     = true
}

data "controld_services" "all" {
  profile_id = controld_profile.test.id

  depends_on = [controld_service.netflix]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.controld_services.all", "services.#"),
					resource.TestCheckResourceAttrSet("data.controld_services.all", "services.0.name"),
				),
			},
		},
	})
}
