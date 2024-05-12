// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProfileResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProfileResourceConfig("tf-acc-test-profile"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_profile.test", "name", "tf-acc-test-profile"),
					resource.TestCheckResourceAttrSet("controld_profile.test", "id"),
					resource.TestCheckResourceAttrSet("controld_profile.test", "updated"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "controld_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The API bumps updated by a second or two on every read of a
				// freshly created profile, so it won't match exactly across
				// the create and import calls.
				ImportStateVerifyIgnore: []string{"updated"},
			},
			// Update and Read testing
			{
				Config: testAccProfileResourceConfig("tf-acc-test-profile-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("controld_profile.test", "name", "tf-acc-test-profile-renamed"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProfileResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "controld_profile" "test" {
  name = %[1]q
}
`, name)
}
