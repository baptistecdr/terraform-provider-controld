// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	controld "github.com/baptistecdr/controld-go"
)

var (
	_ datasource.DataSource              = &ProfileDataSource{}
	_ datasource.DataSourceWithConfigure = &ProfileDataSource{}
)

func NewProfileDataSource() datasource.DataSource {
	return &ProfileDataSource{}
}

// ProfileDataSource reads a single ControlD profile, looked up by id or name.
type ProfileDataSource struct {
	client *controld.API
}

type ProfileDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Updated types.Int64  `tfsdk:"updated"`
}

func (d *ProfileDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_profile"
}

func (d *ProfileDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a single ControlD profile, identified by `id` or `name`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The profile identifier (PK). Exactly one of `id` or `name` must be set.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the profile. Exactly one of `id` or `name` must be set.",
			},
			"updated": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The Unix timestamp of the last update to the profile.",
			},
		},
	}
}

func (d *ProfileDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = client
}

func (d *ProfileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProfileDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() && data.Name.IsNull() {
		resp.Diagnostics.AddError("Missing Profile Identifier", "Either `id` or `name` must be set.")
		return
	}

	profiles, err := d.client.ListProfiles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Profile", err.Error())
		return
	}

	var found *controld.Profile
	for i := range profiles {
		if !data.ID.IsNull() && profiles[i].PK == data.ID.ValueString() {
			found = &profiles[i]
			break
		}
		if !data.Name.IsNull() && profiles[i].Name == data.Name.ValueString() {
			found = &profiles[i]
			break
		}
	}
	if found == nil {
		resp.Diagnostics.AddError("Profile Not Found", fmt.Sprintf("No profile found matching id %q / name %q.", data.ID.ValueString(), data.Name.ValueString()))
		return
	}

	data.ID = types.StringValue(found.PK)
	data.Name = types.StringValue(found.Name)
	data.Updated = types.Int64Value(found.Updated.Unix())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
