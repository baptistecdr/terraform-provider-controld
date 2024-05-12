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

type FilterDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Status      types.Bool   `tfsdk:"status"`
}

type FiltersDataSourceModel struct {
	ID        types.String            `tfsdk:"id"`
	ProfileID types.String            `tfsdk:"profile_id"`
	Filters   []FilterDataSourceModel `tfsdk:"filters"`
}

func filtersDataSourceSchema(markdownDescription string) schema.Schema {
	return schema.Schema{
		MarkdownDescription: markdownDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of this data source, equal to `profile_id`.",
			},
			"profile_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the profile to list filters for.",
			},
			"filters": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The list of filters.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The filter identifier (PK), usable as `filter` in a `controld_filter` resource.",
						},
						"description": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The description of the filter.",
						},
						"status": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the filter is currently enabled.",
						},
					},
				},
			},
		},
	}
}

func filtersToDataSourceModel(filters []controld.Filter) []FilterDataSourceModel {
	out := make([]FilterDataSourceModel, 0, len(filters))
	for _, f := range filters {
		out = append(out, FilterDataSourceModel{
			Name:        types.StringValue(f.PK),
			Description: types.StringValue(f.Description),
			Status:      types.BoolValue(bool(f.Status)),
		})
	}
	return out
}

// NativeFiltersDataSource lists ControlD's native filter lists (e.g. malware,
// ads) and their current status for a profile.
var (
	_ datasource.DataSource              = &NativeFiltersDataSource{}
	_ datasource.DataSourceWithConfigure = &NativeFiltersDataSource{}
)

func NewNativeFiltersDataSource() datasource.DataSource {
	return &NativeFiltersDataSource{}
}

type NativeFiltersDataSource struct {
	client *controld.API
}

func (d *NativeFiltersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_native_filters"
}

func (d *NativeFiltersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = filtersDataSourceSchema("Lists ControlD's native filter lists (e.g. malware, ads) and their current status for a profile.")
}

func (d *NativeFiltersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = client
}

func (d *NativeFiltersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FiltersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filters, err := d.client.ListProfileNativeFilters(ctx, controld.ListProfileFiltersParams{
		ProfileID: data.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to List Native Filters", err.Error())
		return
	}

	data.ID = data.ProfileID
	data.Filters = filtersToDataSourceModel(filters)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ExternalFiltersDataSource lists third-party filter lists that can be
// enabled for a profile.
var (
	_ datasource.DataSource              = &ExternalFiltersDataSource{}
	_ datasource.DataSourceWithConfigure = &ExternalFiltersDataSource{}
)

func NewExternalFiltersDataSource() datasource.DataSource {
	return &ExternalFiltersDataSource{}
}

type ExternalFiltersDataSource struct {
	client *controld.API
}

func (d *ExternalFiltersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_external_filters"
}

func (d *ExternalFiltersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = filtersDataSourceSchema("Lists third-party filter lists that can be enabled for a profile.")
}

func (d *ExternalFiltersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = client
}

func (d *ExternalFiltersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FiltersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filters, err := d.client.ListProfileExternalFilters(ctx, controld.ListProfileFiltersParams{
		ProfileID: data.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to List External Filters", err.Error())
		return
	}

	data.ID = data.ProfileID
	data.Filters = filtersToDataSourceModel(filters)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
