// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEndpointScheduleResource(t *testing.T) {
	profileName := testAccRandomName("tfacc-sched-profile")
	altProfileName := testAccRandomName("tfacc-sched-alt")
	deviceName := testAccRandomName("tfacc-sched-device")
	scheduleName := testAccRandomName("tfacc-schedule")
	renamed := scheduleName + "-new"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccEndpointScheduleResourceConfig(profileName, altProfileName, deviceName, scheduleName, "Europe/Paris", "21:00", "23:00"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_endpoint_schedule.test", "name", scheduleName),
					resource.TestCheckResourceAttr("controld_endpoint_schedule.test", "time_zone", "Europe/Paris"),
					resource.TestCheckResourceAttr("controld_endpoint_schedule.test", "status", "true"),
					resource.TestCheckResourceAttr("controld_endpoint_schedule.test", "windows.#", "2"),
					resource.TestCheckResourceAttr("controld_endpoint_schedule.test", "windows.0.weekday", "mon"),
					resource.TestCheckResourceAttr("controld_endpoint_schedule.test", "windows.0.time_start", "21:00"),
					resource.TestCheckResourceAttrSet("controld_endpoint_schedule.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "controld_endpoint_schedule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing: rename, change time zone and windows
			{
				Config: testAccEndpointScheduleResourceConfig(profileName, altProfileName, deviceName, renamed, "Europe/London", "20:00", "22:30"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_endpoint_schedule.test", "name", renamed),
					resource.TestCheckResourceAttr("controld_endpoint_schedule.test", "time_zone", "Europe/London"),
					resource.TestCheckResourceAttr("controld_endpoint_schedule.test", "windows.0.time_start", "20:00"),
					resource.TestCheckResourceAttr("controld_endpoint_schedule.test", "windows.0.time_end", "22:30"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccEndpointScheduleResourceConfig(profileName, altProfileName, deviceName, name, timeZone, start, end string) string {
	return fmt.Sprintf(`
resource "controld_profile" "test" {
  name = %[1]q
}

resource "controld_profile" "alt" {
  name = %[2]q
}

resource "controld_device" "test" {
  name       = %[3]q
  profile_id = controld_profile.test.id
  icon       = "desktop-mac"
}

resource "controld_endpoint_schedule" "test" {
  name      = %[4]q
  device_id = controld_device.test.id
  time_zone = %[5]q

  windows = [
    {
      weekday    = "mon"
      time_start = %[6]q
      time_end   = %[7]q
      profile_id = controld_profile.alt.id
    },
    {
      weekday    = "tue"
      time_start = %[6]q
      time_end   = %[7]q
      profile_id = controld_profile.alt.id
    },
  ]
}
`, profileName, altProfileName, deviceName, name, timeZone, start, end)
}
