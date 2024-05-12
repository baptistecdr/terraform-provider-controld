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
	_ datasource.DataSource              = &ServicesDataSource{}
	_ datasource.DataSourceWithConfigure = &ServicesDataSource{}
)

func NewServicesDataSource() datasource.DataSource {
	return &ServicesDataSource{}
}

// ServicesDataSource lists the built-in services and their current action for
// a ControlD profile.
type ServicesDataSource struct {
	client *controld.API
}

type ServiceDataSourceModel struct {
	Name           types.String   `tfsdk:"name"`
	Category       types.String   `tfsdk:"category"`
	UnlockLocation types.String   `tfsdk:"unlock_location"`
	Locations      []types.String `tfsdk:"locations"`
	Do             types.Int64    `tfsdk:"do"`
	Status         types.Bool     `tfsdk:"status"`
}

type ServicesDataSourceModel struct {
	ID        types.String             `tfsdk:"id"`
	ProfileID types.String             `tfsdk:"profile_id"`
	Services  []ServiceDataSourceModel `tfsdk:"services"`
}

func (d *ServicesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_services"
}

func (d *ServicesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the built-in ControlD services and their current action for a profile.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of this data source, equal to `profile_id`.",
			},
			"profile_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the profile to list services for.",
			},
			"services": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The list of services.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The service identifier (PK), usable as `service` in a `controld_service` resource.",
						},
						"category": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The service category.",
						},
						"unlock_location": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The unlock location required to bypass geo-restrictions for this service.",
						},
						"locations": schema.ListAttribute{
							Computed:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "The locations this service can be unlocked in.",
						},
						"do": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The current action for this service: `0` (block), `1` (bypass), `2` (spoof), or `3` (redirect).",
						},
						"status": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the service action is currently enabled.",
						},
					},
				},
			},
		},
	}
}

func (d *ServicesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = client
}

func (d *ServicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServicesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	services, err := d.client.ListProfileServices(ctx, controld.ListProfileServicesParams{
		ProfileID: data.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to List Services", err.Error())
		return
	}

	data.ID = data.ProfileID
	data.Services = make([]ServiceDataSourceModel, 0, len(services))
	for _, s := range services {
		locations := make([]types.String, 0, len(s.Locations))
		for _, l := range s.Locations {
			locations = append(locations, types.StringValue(l))
		}
		data.Services = append(data.Services, ServiceDataSourceModel{
			Name:           types.StringValue(s.PK),
			Category:       types.StringValue(s.Category),
			UnlockLocation: types.StringValue(s.UnlockLocation),
			Locations:      locations,
			Do:             types.Int64Value(int64(s.Action.Do)),
			Status:         types.BoolValue(bool(s.Action.Status)),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
