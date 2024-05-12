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
	_ datasource.DataSource              = &DeviceDataSource{}
	_ datasource.DataSourceWithConfigure = &DeviceDataSource{}
)

func NewDeviceDataSource() datasource.DataSource {
	return &DeviceDataSource{}
}

// DeviceDataSource reads a single ControlD device, looked up by id or name.
type DeviceDataSource struct {
	client *controld.API
}

type DeviceDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	ProfileID        types.String `tfsdk:"profile_id"`
	Icon             types.String `tfsdk:"icon"`
	Desc             types.String `tfsdk:"desc"`
	Stats            types.Int64  `tfsdk:"stats"`
	Status           types.Int64  `tfsdk:"status"`
	LegacyIPv4Status types.Bool   `tfsdk:"legacy_ipv4_status"`
	LearnIP          types.Bool   `tfsdk:"learn_ip"`
	Restricted       types.Bool   `tfsdk:"restricted"`
	DDNSStatus       types.Bool   `tfsdk:"ddns_status"`
	DDNSSubdomain    types.String `tfsdk:"ddns_subdomain"`
	DeviceID         types.String `tfsdk:"device_id"`
}

func (d *DeviceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (d *DeviceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a single ControlD device, identified by `id` or `name`.",
		Attributes:          deviceDataSourceAttributes(true),
	}
}

func (d *DeviceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = client
}

func (d *DeviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DeviceDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.IsNull() && data.Name.IsNull() {
		resp.Diagnostics.AddError("Missing Device Identifier", "Either `id` or `name` must be set.")
		return
	}

	devices, err := d.client.ListDevices(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Device", err.Error())
		return
	}

	var found *controld.Device
	for i := range devices {
		if !data.ID.IsNull() && devices[i].PK == data.ID.ValueString() {
			found = &devices[i]
			break
		}
		if !data.Name.IsNull() && devices[i].Name == data.Name.ValueString() {
			found = &devices[i]
			break
		}
	}
	if found == nil {
		resp.Diagnostics.AddError("Device Not Found", fmt.Sprintf("No device found matching id %q / name %q.", data.ID.ValueString(), data.Name.ValueString()))
		return
	}

	data = deviceToDataSourceModel(*found)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// deviceDataSourceAttributes returns the schema attributes shared by the
// singular and plural device data sources. When singular is true, id and name
// are optional lookup keys; otherwise they are computed only.
func deviceDataSourceAttributes(singular bool) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Optional:            singular,
			Computed:            true,
			MarkdownDescription: "The device identifier (PK).",
		},
		"name": schema.StringAttribute{
			Optional:            singular,
			Computed:            true,
			MarkdownDescription: "The name of the device.",
		},
		"profile_id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The identifier of the profile applied to this device.",
		},
		"icon": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The device icon.",
		},
		"desc": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The description of the device.",
		},
		"stats": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "The analytics level for the device.",
		},
		"status": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "The device status.",
		},
		"legacy_ipv4_status": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether legacy IPv4 resolving is enabled for this device.",
		},
		"learn_ip": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether the device should learn and remember the IPs it connects from.",
		},
		"restricted": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether the device is restricted to learned IPs only.",
		},
		"ddns_status": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether ControlD-managed Dynamic DNS is enabled for this device.",
		},
		"ddns_subdomain": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The subdomain used for ControlD-managed Dynamic DNS.",
		},
		"device_id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The unique device identifier used to build its DNS resolver endpoints.",
		},
	}
}

func deviceToDataSourceModel(device controld.Device) DeviceDataSourceModel {
	data := DeviceDataSourceModel{
		ID:               types.StringValue(device.PK),
		Name:             types.StringValue(device.Name),
		ProfileID:        types.StringValue(device.Profile.PK),
		Desc:             types.StringValue(device.Desc),
		Status:           types.Int64Value(int64(device.Status)),
		LegacyIPv4Status: types.BoolValue(bool(device.LegacyIPv4.Status)),
		LearnIP:          types.BoolValue(bool(device.LearnIP)),
		DeviceID:         types.StringValue(device.DeviceID),
	}
	if device.Icon != nil {
		data.Icon = types.StringValue(string(*device.Icon))
	}
	if device.Stats != nil {
		data.Stats = types.Int64Value(int64(*device.Stats))
	}
	if device.Restricted != nil {
		data.Restricted = types.BoolValue(bool(*device.Restricted))
	}
	if device.DDNS != nil {
		data.DDNSStatus = types.BoolValue(device.DDNS.Status == 1)
		data.DDNSSubdomain = types.StringValue(device.DDNS.Subdomain)
	}
	return data
}
