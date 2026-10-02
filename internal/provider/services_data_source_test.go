// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServicesDataSource(t *testing.T) {
	profileName := testAccRandomName("tfacc-services-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// ListProfileServices only returns services with an explicit
				// action configured on the profile, so a freshly created
				// profile starts with an empty list: configure one first.
				Config: fmt.Sprintf(`
resource "controld_profile" "test" {
  name = %[1]q
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
`, profileName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.controld_services.all", "services.#"),
					resource.TestCheckResourceAttrSet("data.controld_services.all", "services.0.name"),
				),
			},
		},
	})
}
