// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"controld": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("CONTROLD_API_TOKEN") == "" {
		t.Fatal("CONTROLD_API_TOKEN must be set for acceptance tests")
	}
}

// testAccRandomName returns a unique resource name so concurrent or
// interrupted runs against the shared ControlD account never collide on the
// name-uniqueness checks the API enforces. The API caps names at 32
// characters, so the suffix is 7 characters ("-" plus 6) and prefixes must
// leave room for it, plus 4 more for the "-new" rename suffix where used.
func testAccRandomName(prefix string) string {
	return prefix + "-" + acctest.RandString(6)
}
