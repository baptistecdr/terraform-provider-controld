// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDeviceResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDeviceResourceConfig("tf-acc-test-device", "desktop-mac"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_device.test", "name", "tf-acc-test-device"),
					resource.TestCheckResourceAttr("controld_device.test", "icon", "desktop-mac"),
					resource.TestCheckResourceAttrPair("controld_device.test", "profile_id", "controld_profile.test", "id"),
					resource.TestCheckResourceAttrSet("controld_device.test", "id"),
					resource.TestCheckResourceAttrSet("controld_device.test", "device_id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "controld_device.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Not readable back from the API.
				ImportStateVerifyIgnore: []string{"ddns_ext_status", "ddns_ext_host", "ctrld_custom_config"},
			},
			// Update and Read testing
			{
				Config: testAccDeviceResourceConfig("tf-acc-test-device-renamed", "desktop-mac"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_device.test", "name", "tf-acc-test-device-renamed"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccDeviceResourceConfig(name, icon string) string {
	return fmt.Sprintf(`
resource "controld_profile" "test" {
  name = "tf-acc-test-device-profile"
}

resource "controld_device" "test" {
  name       = %[1]q
  profile_id = controld_profile.test.id
  icon       = %[2]q
  learn_ip   = true
}
`, name, icon)
}
