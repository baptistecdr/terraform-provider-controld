// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDeviceResource(t *testing.T) {
	profileName := testAccRandomName("tfacc-device-profile")
	deviceName := testAccRandomName("tfacc-device")
	renamed := deviceName + "-new"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDeviceResourceConfig(profileName, deviceName, "desktop-mac"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_device.test", "name", deviceName),
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
				Config: testAccDeviceResourceConfig(profileName, renamed, "desktop-mac"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_device.test", "name", renamed),
				),
			},
			// Updating a non-name attribute must not resend the unchanged name,
			// which the API rejects as a duplicate of the device's own name.
			{
				Config: testAccDeviceResourceConfigWithStats(profileName, renamed, "desktop-mac", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_device.test", "name", renamed),
					resource.TestCheckResourceAttr("controld_device.test", "stats", "1"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccDeviceResourceConfig(profileName, name, icon string) string {
	return fmt.Sprintf(`
resource "controld_profile" "test" {
  name = %[1]q
}

resource "controld_device" "test" {
  name       = %[2]q
  profile_id = controld_profile.test.id
  icon       = %[3]q
  learn_ip   = true
}
`, profileName, name, icon)
}

func testAccDeviceResourceConfigWithStats(profileName, name, icon string, stats int) string {
	return fmt.Sprintf(`
resource "controld_profile" "test" {
  name = %[1]q
}

resource "controld_device" "test" {
  name       = %[2]q
  profile_id = controld_profile.test.id
  icon       = %[3]q
  learn_ip   = true
  stats      = %[4]d
}
`, profileName, name, icon, stats)
}

// ControlD creates every device as pending (0) regardless of the requested
// status, so status = 1 must not fail with an inconsistent result on create.
func TestAccDeviceResourceStatusOnCreate(t *testing.T) {
	profileName := testAccRandomName("tfacc-device-status-pf")
	deviceName := testAccRandomName("tfacc-device-status")

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
  status     = 1
}
`, profileName, deviceName),
			},
		},
	})
}
