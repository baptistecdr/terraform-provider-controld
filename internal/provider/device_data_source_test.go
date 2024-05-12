// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDeviceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "controld_profile" "test" {
  name = "tfacc-device-ds-profile"
}

resource "controld_device" "test" {
  name       = "tf-acc-test-device-data-source"
  profile_id = controld_profile.test.id
  icon       = "desktop-mac"
}

data "controld_device" "by_id" {
  id = controld_device.test.id
}

data "controld_device" "by_name" {
  name = controld_device.test.name
}

data "controld_devices" "all" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.controld_device.by_id", "name", "controld_device.test", "name"),
					resource.TestCheckResourceAttrPair("data.controld_device.by_name", "id", "controld_device.test", "id"),
					resource.TestCheckResourceAttrSet("data.controld_devices.all", "devices.#"),
				),
			},
		},
	})
}
