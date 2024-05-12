// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	controld "github.com/baptistecdr/controld-go"
)

var (
	_ datasource.DataSource              = &ProfilesDataSource{}
	_ datasource.DataSourceWithConfigure = &ProfilesDataSource{}
)

func NewProfilesDataSource() datasource.DataSource {
	return &ProfilesDataSource{}
}

// ProfilesDataSource lists every ControlD profile in the account.
type ProfilesDataSource struct {
	client *controld.API
}

type ProfilesDataSourceModel struct {
	ID       types.String             `tfsdk:"id"`
	Profiles []ProfileDataSourceModel `tfsdk:"profiles"`
}

func (d *ProfilesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_profiles"
}

func (d *ProfilesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all ControlD profiles in the account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Placeholder identifier.",
			},
			"profiles": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The list of profiles.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The profile identifier (PK).",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the profile.",
						},
						"updated": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The Unix timestamp of the last update to the profile.",
						},
					},
				},
			},
		},
	}
}

func (d *ProfilesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = client
}

func (d *ProfilesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProfilesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profiles, err := d.client.ListProfiles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to List Profiles", err.Error())
		return
	}

	data.ID = types.StringValue("profiles")
	data.Profiles = make([]ProfileDataSourceModel, 0, len(profiles))
	for _, p := range profiles {
		data.Profiles = append(data.Profiles, ProfileDataSourceModel{
			ID:      types.StringValue(p.PK),
			Name:    types.StringValue(p.Name),
			Updated: types.Int64Value(p.Updated.Unix()),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
