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
	_ datasource.DataSource              = &ProxiesDataSource{}
	_ datasource.DataSourceWithConfigure = &ProxiesDataSource{}
)

func NewProxiesDataSource() datasource.DataSource {
	return &ProxiesDataSource{}
}

// ProxiesDataSource lists the proxies traffic can be redirected through,
// usable as the `via` target of a `redirect` custom rule, default rule, or
// rule folder action.
type ProxiesDataSource struct {
	client *controld.API
}

type ProxyDataSourceModel struct {
	PK          types.String  `tfsdk:"pk"`
	UID         types.String  `tfsdk:"uid"`
	City        types.String  `tfsdk:"city"`
	Country     types.String  `tfsdk:"country"`
	CountryName types.String  `tfsdk:"country_name"`
	Lat         types.Float64 `tfsdk:"lat"`
	Long        types.Float64 `tfsdk:"long"`
	Hidden      types.Bool    `tfsdk:"hidden"`
}

type ProxyCountryDataSourceModel struct {
	Country     types.String `tfsdk:"country"`
	CountryName types.String `tfsdk:"country_name"`
}

type ProxiesDataSourceModel struct {
	ID        types.String                  `tfsdk:"id"`
	Proxies   []ProxyDataSourceModel        `tfsdk:"proxies"`
	Countries []ProxyCountryDataSourceModel `tfsdk:"countries"`
}

func (d *ProxiesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_proxies"
}

func (d *ProxiesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the proxies traffic can be redirected through, usable as the `via` target of a `redirect` action on a `controld_custom_rule`, `controld_default_rule`, or `controld_rule_folder`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Placeholder identifier.",
			},
			"proxies": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The list of proxies.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"pk": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The proxy identifier, usable as `via` in a `redirect` action.",
						},
						"uid": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The proxy's unique identifier.",
						},
						"city": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The city the proxy is located in.",
						},
						"country": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The country code the proxy is located in.",
						},
						"country_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The country name the proxy is located in.",
						},
						"lat": schema.Float64Attribute{
							Computed:            true,
							MarkdownDescription: "The proxy's latitude.",
						},
						"long": schema.Float64Attribute{
							Computed:            true,
							MarkdownDescription: "The proxy's longitude.",
						},
						"hidden": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the proxy is hidden from the default list.",
						},
					},
				},
			},
			"countries": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The list of countries the proxies belong to.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"country": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The country code.",
						},
						"country_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The country name.",
						},
					},
				},
			},
		},
	}
}

func (d *ProxiesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = client
}

func (d *ProxiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProxiesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	proxies, countries, err := d.client.ListProxies(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to List Proxies", err.Error())
		return
	}

	data.ID = types.StringValue("proxies")

	data.Proxies = make([]ProxyDataSourceModel, 0, len(proxies))
	for _, p := range proxies {
		data.Proxies = append(data.Proxies, ProxyDataSourceModel{
			PK:          types.StringValue(p.PK),
			UID:         types.StringValue(p.UID),
			City:        types.StringValue(p.City),
			Country:     types.StringValue(p.Country),
			CountryName: types.StringValue(p.CountryName),
			Lat:         types.Float64Value(p.Lat),
			Long:        types.Float64Value(p.Long),
			Hidden:      types.BoolValue(p.Hidden),
		})
	}

	data.Countries = make([]ProxyCountryDataSourceModel, 0, len(countries))
	for _, c := range countries {
		data.Countries = append(data.Countries, ProxyCountryDataSourceModel{
			Country:     types.StringValue(c.Country),
			CountryName: types.StringValue(c.CountryName),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
