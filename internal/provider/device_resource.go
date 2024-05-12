// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	controld "github.com/baptistecdr/controld-go"
)

var (
	_ resource.Resource                = &DeviceResource{}
	_ resource.ResourceWithConfigure   = &DeviceResource{}
	_ resource.ResourceWithImportState = &DeviceResource{}
)

// validDeviceIcons lists the icon identifiers accepted by the ControlD API.
var validDeviceIcons = []string{
	string(controld.DesktopWindows), string(controld.DesktopMac), string(controld.DesktopLinux),
	string(controld.MobileIOS), string(controld.MobileAndroid),
	string(controld.BrowserChrome), string(controld.BrowserFirefox), string(controld.BrowserEdge), string(controld.BrowserBrave), string(controld.BrowserOther),
	string(controld.TVApple), string(controld.TVAndroid), string(controld.TVFireTV), string(controld.TVSamsung), string(controld.TVOther),
	string(controld.RouterAsus), string(controld.RouterDDWRT), string(controld.RouterFirewalla), string(controld.RouterFreshTomato), string(controld.RouterGLiNET),
	string(controld.RouterOpenWRT), string(controld.RouterOPNsense), string(controld.RouterPfSense), string(controld.RouterSynology), string(controld.RouterUbiquiti),
	string(controld.RouterWindows), string(controld.RouterLinux), string(controld.RouterOther),
}

func NewDeviceResource() resource.Resource {
	return &DeviceResource{}
}

// DeviceResource manages a ControlD device.
type DeviceResource struct {
	client *controld.API
}

// DeviceResourceModel describes the resource data model.
type DeviceResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ProfileID         types.String `tfsdk:"profile_id"`
	ProfileID2        types.String `tfsdk:"profile_id2"`
	Icon              types.String `tfsdk:"icon"`
	Desc              types.String `tfsdk:"desc"`
	Stats             types.Int64  `tfsdk:"stats"`
	Status            types.Int64  `tfsdk:"status"`
	LegacyIPv4Status  types.Bool   `tfsdk:"legacy_ipv4_status"`
	LearnIP           types.Bool   `tfsdk:"learn_ip"`
	Restricted        types.Bool   `tfsdk:"restricted"`
	DDNSStatus        types.Bool   `tfsdk:"ddns_status"`
	DDNSSubdomain     types.String `tfsdk:"ddns_subdomain"`
	DDNSExtStatus     types.Bool   `tfsdk:"ddns_ext_status"`
	DDNSExtHost       types.String `tfsdk:"ddns_ext_host"`
	CtrldCustomConfig types.String `tfsdk:"ctrld_custom_config"`
	DeviceID          types.String `tfsdk:"device_id"`
}

func (r *DeviceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (r *DeviceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a ControlD device. A device represents an endpoint (computer, phone, router, ...) that resolves DNS through a ControlD profile.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The device identifier (PK).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the device.",
			},
			"profile_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the profile applied to this device.",
			},
			"profile_id2": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The identifier of a secondary profile applied to this device.",
			},
			"icon": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The device icon. One of: `" + fmt.Sprintf("%v", validDeviceIcons) + "`. Changing this forces a new resource to be created, as the ControlD API does not support updating it.",
				Validators: []validator.String{
					stringvalidator.OneOf(validDeviceIcons...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"desc": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A description for the device.",
			},
			"stats": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "The analytics level for the device: `0` (off), `1` (basic), or `2` (full). Only present in state once explicitly set; the API omits it entirely until then.",
				Validators: []validator.Int64{
					int64validator.Between(0, 2),
				},
			},
			"status": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The device status: `0` (pending), `1` (active), `2` (soft disabled), or `3` (hard disabled).",
				Validators: []validator.Int64{
					int64validator.Between(0, 3),
				},
			},
			"legacy_ipv4_status": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether legacy IPv4 resolving is enabled for this device.",
			},
			"learn_ip": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the device should learn and remember the IPs it connects from.",
			},
			"restricted": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether the device is restricted to learned IPs only. Only present in state once explicitly set; the API omits it entirely until then.",
			},
			"ddns_status": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether ControlD-managed Dynamic DNS is enabled for this device. Only present in state once explicitly set; the API omits it entirely until then.",
			},
			"ddns_subdomain": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The subdomain used for ControlD-managed Dynamic DNS. Only present in state once explicitly set; the API omits it entirely until then.",
			},
			"ddns_ext_status": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether external Dynamic DNS is enabled for this device. Not readable back from the API.",
			},
			"ddns_ext_host": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The externally managed hostname used for Dynamic DNS. Not readable back from the API.",
			},
			"ctrld_custom_config": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Custom configuration overrides for the ctrld client running on this device. Not readable back from the API.",
			},
			"device_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique device identifier used to build its DNS resolver endpoints.",
			},
		},
	}
}

func (r *DeviceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = client
}

func (r *DeviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeviceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := controld.CreateDeviceParams{
		Name:      data.Name.ValueString(),
		ProfileID: data.ProfileID.ValueString(),
		Icon:      controld.IconName(data.Icon.ValueString()),
	}
	if !data.ProfileID2.IsNull() {
		v := data.ProfileID2.ValueString()
		params.ProfileID2 = &v
	}
	if !data.Desc.IsNull() && !data.Desc.IsUnknown() {
		v := data.Desc.ValueString()
		params.Desc = &v
	}
	if !data.Stats.IsNull() && !data.Stats.IsUnknown() {
		v := controld.AnalyticsLevel(data.Stats.ValueInt64())
		params.Stats = &v
	}
	if !data.LegacyIPv4Status.IsNull() && !data.LegacyIPv4Status.IsUnknown() {
		v := controld.IntBool(data.LegacyIPv4Status.ValueBool())
		params.LegacyIPv4Status = &v
	}
	if !data.LearnIP.IsNull() && !data.LearnIP.IsUnknown() {
		v := controld.IntBool(data.LearnIP.ValueBool())
		params.LearnIP = &v
	}
	if !data.Restricted.IsNull() && !data.Restricted.IsUnknown() {
		v := controld.IntBool(data.Restricted.ValueBool())
		params.Restricted = &v
	}
	if !data.DDNSStatus.IsNull() && !data.DDNSStatus.IsUnknown() {
		v := controld.IntBool(data.DDNSStatus.ValueBool())
		params.DDNSStatus = &v
	}
	if !data.DDNSSubdomain.IsNull() && !data.DDNSSubdomain.IsUnknown() {
		v := data.DDNSSubdomain.ValueString()
		params.DDNSSubdomain = &v
	}
	if !data.DDNSExtStatus.IsNull() {
		v := controld.IntBool(data.DDNSExtStatus.ValueBool())
		params.DDNSExtStatus = &v
	}
	if !data.DDNSExtHost.IsNull() {
		v := data.DDNSExtHost.ValueString()
		params.DDNSExtHost = &v
	}

	device, err := r.client.CreateDevice(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Device", err.Error())
		return
	}

	r.updateModelFromDevice(&data, device)

	if !data.CtrldCustomConfig.IsNull() {
		if err := r.applyCustomConfig(ctx, data); err != nil {
			resp.Diagnostics.AddError("Unable to Apply Device ctrld Custom Config", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeviceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	devices, err := r.client.ListDevices(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Device", err.Error())
		return
	}

	var found *controld.Device
	for i := range devices {
		if devices[i].PK == data.ID.ValueString() {
			found = &devices[i]
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.updateModelFromDevice(&data, *found)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DeviceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	profileID := data.ProfileID.ValueString()
	params := controld.UpdateDeviceParams{
		DeviceID:  data.ID.ValueString(),
		Name:      &name,
		ProfileID: &profileID,
	}
	if !data.ProfileID2.IsNull() {
		v := data.ProfileID2.ValueString()
		params.ProfileID2 = &v
	}
	if !data.Desc.IsNull() && !data.Desc.IsUnknown() {
		v := data.Desc.ValueString()
		params.Desc = &v
	}
	if !data.Stats.IsNull() && !data.Stats.IsUnknown() {
		v := controld.AnalyticsLevel(data.Stats.ValueInt64())
		params.Stats = &v
	}
	if !data.Status.IsNull() && !data.Status.IsUnknown() {
		v := controld.DeviceStatus(data.Status.ValueInt64())
		params.Status = &v
	}
	if !data.LegacyIPv4Status.IsNull() && !data.LegacyIPv4Status.IsUnknown() {
		v := controld.IntBool(data.LegacyIPv4Status.ValueBool())
		params.LegacyIPv4Status = &v
	}
	if !data.LearnIP.IsNull() && !data.LearnIP.IsUnknown() {
		v := controld.IntBool(data.LearnIP.ValueBool())
		params.LearnIP = &v
	}
	if !data.Restricted.IsNull() && !data.Restricted.IsUnknown() {
		v := controld.IntBool(data.Restricted.ValueBool())
		params.Restricted = &v
	}
	if !data.DDNSStatus.IsNull() && !data.DDNSStatus.IsUnknown() {
		v := controld.IntBool(data.DDNSStatus.ValueBool())
		params.DDNSStatus = &v
	}
	if !data.DDNSSubdomain.IsNull() && !data.DDNSSubdomain.IsUnknown() {
		v := data.DDNSSubdomain.ValueString()
		params.DDNSSubdomain = &v
	}
	if !data.DDNSExtStatus.IsNull() {
		v := controld.IntBool(data.DDNSExtStatus.ValueBool())
		params.DDNSExtStatus = &v
	}
	if !data.DDNSExtHost.IsNull() {
		v := data.DDNSExtHost.ValueString()
		params.DDNSExtHost = &v
	}
	if !data.CtrldCustomConfig.IsNull() {
		v := data.CtrldCustomConfig.ValueString()
		params.CtrldCustomConfig = &v
	}

	device, err := r.client.UpdateDevice(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Device", err.Error())
		return
	}

	r.updateModelFromDevice(&data, device)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DeviceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteDevice(ctx, controld.DeleteDeviceParams{
		DeviceID: data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Delete Device", err.Error())
		return
	}
}

func (r *DeviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyCustomConfig sets ctrld_custom_config, which CreateDevice does not accept.
func (r *DeviceResource) applyCustomConfig(ctx context.Context, data DeviceResourceModel) error {
	v := data.CtrldCustomConfig.ValueString()
	_, err := r.client.UpdateDevice(ctx, controld.UpdateDeviceParams{
		DeviceID:          data.ID.ValueString(),
		CtrldCustomConfig: &v,
	})
	return err
}

// updateModelFromDevice copies fields readable from the ControlD API into data,
// leaving write-only attributes (ddns_ext_status, ddns_ext_host,
// ctrld_custom_config) untouched.
func (r *DeviceResource) updateModelFromDevice(data *DeviceResourceModel, device controld.Device) {
	data.ID = types.StringValue(device.PK)
	data.Name = types.StringValue(device.Name)
	data.ProfileID = types.StringValue(device.Profile.PK)
	if device.Icon != nil {
		data.Icon = types.StringValue(string(*device.Icon))
	}
	data.Desc = types.StringValue(device.Desc)
	data.Status = types.Int64Value(int64(device.Status))
	data.LegacyIPv4Status = types.BoolValue(bool(device.LegacyIPv4.Status))
	data.LearnIP = types.BoolValue(bool(device.LearnIP))
	data.DeviceID = types.StringValue(device.DeviceID)

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
}
