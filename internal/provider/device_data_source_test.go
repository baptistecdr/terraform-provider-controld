// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDeviceDataSource(t *testing.T) {
	profileName := testAccRandomName("tfacc-device-ds-profile")
	deviceName := testAccRandomName("tfacc-device-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "controld_profile" "test" {
  name = %[1]q
}

resource "controld_device" "test" {
  name       = %[2]q
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
`, profileName, deviceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.controld_device.by_id", "name", "controld_device.test", "name"),
					resource.TestCheckResourceAttrPair("data.controld_device.by_name", "id", "controld_device.test", "id"),
					resource.TestCheckResourceAttrSet("data.controld_devices.all", "devices.#"),
				),
			},
		},
	})
}
