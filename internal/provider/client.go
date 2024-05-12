// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	controld "github.com/baptistecdr/controld-go"
)

// configureClient extracts the *controld.API client set up during provider
// Configure() from providerData. It reports a diagnostic and returns ok=false
// if providerData is set but is not of the expected type.
func configureClient(providerData any, diags *diag.Diagnostics) (*controld.API, bool) {
	if providerData == nil {
		return nil, true
	}

	client, ok := providerData.(*controld.API)
	if !ok {
		diags.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *controld.API, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil, false
	}

	return client, true
}

// splitImportID splits a "left/right" composite import identifier used by
// resources whose state is keyed by more than just an id, such as
// profile_id/folder_id or profile_id/service.
func splitImportID(id string) (left, right string, err error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("got: %q", id)
	}
	return parts[0], parts[1], nil
}
