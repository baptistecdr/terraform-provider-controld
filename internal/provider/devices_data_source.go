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
	_ datasource.DataSource              = &DevicesDataSource{}
	_ datasource.DataSourceWithConfigure = &DevicesDataSource{}
)

func NewDevicesDataSource() datasource.DataSource {
	return &DevicesDataSource{}
}

// DevicesDataSource lists every ControlD device in the account.
type DevicesDataSource struct {
	client *controld.API
}

type DevicesDataSourceModel struct {
	ID      types.String            `tfsdk:"id"`
	Devices []DeviceDataSourceModel `tfsdk:"devices"`
}

func (d *DevicesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_devices"
}

func (d *DevicesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all ControlD devices in the account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Placeholder identifier.",
			},
			"devices": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The list of devices.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: deviceDataSourceAttributes(false),
				},
			},
		},
	}
}

func (d *DevicesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = client
}

func (d *DevicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DevicesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	devices, err := d.client.ListDevices(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to List Devices", err.Error())
		return
	}

	data.ID = types.StringValue("devices")
	data.Devices = make([]DeviceDataSourceModel, 0, len(devices))
	for _, device := range devices {
		data.Devices = append(data.Devices, deviceToDataSourceModel(device))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
