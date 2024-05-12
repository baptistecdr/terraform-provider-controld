// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = &ServiceResource{}
	_ resource.ResourceWithConfigure   = &ServiceResource{}
	_ resource.ResourceWithImportState = &ServiceResource{}
)

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

// ServiceResource toggles a built-in ControlD service (e.g. a streaming or
// social media service) for a profile. Services are predefined by ControlD:
// there is no create/delete endpoint, only an update, so Create and Update
// both call UpdateProfileService, and Delete restores the service to its
// default state (bypass, disabled).
type ServiceResource struct {
	client *controld.API
}

type ServiceResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ProfileID types.String `tfsdk:"profile_id"`
	Service   types.String `tfsdk:"service"`
	Do        types.Int64  `tfsdk:"do"`
	Status    types.Bool   `tfsdk:"status"`
	Via       types.String `tfsdk:"via"`
	ViaV6     types.String `tfsdk:"via_v6"`
}

func (r *ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the action applied to a built-in ControlD service (e.g. a streaming or social media service) for a profile. Services are predefined by ControlD (see the `controld_services` data source for the full catalog); deleting this resource restores the service to its default state (bypass, disabled).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of this resource, in the form `profile_id/service`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the profile this service action belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The service identifier (PK), as returned by the `controld_services` data source.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"do": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "The action to take: `0` (block), `1` (bypass), `2` (spoof), or `3` (redirect).",
				Validators: []validator.Int64{
					int64validator.Between(0, 3),
				},
			},
			"status": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether the service action is enabled.",
			},
			"via": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The destination IP/hostname used when `do` is `spoof` or `redirect`.",
			},
			"via_v6": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The IPv6 destination used when `do` is `spoof` or `redirect`.",
			},
		},
	}
}

func (r *ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = client
}

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applyService(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.ProfileID.ValueString(), data.Service.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	services, err := r.client.ListProfileServices(ctx, controld.ListProfileServicesParams{
		ProfileID: data.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Service", err.Error())
		return
	}

	var found *controld.ProfileService
	for i := range services {
		if services[i].PK == data.Service.ValueString() {
			found = &services[i]
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	updateModelFromServiceAction(&data, found.Action)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applyService(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateProfileService(ctx, controld.UpdateProfileServiceParams{
		ProfileID: data.ProfileID.ValueString(),
		Service:   data.Service.ValueString(),
		Do:        controld.Bypass,
		Status:    controld.IntBool(false),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Reset Service", err.Error())
		return
	}
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	profileID, service, err := splitImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected Import Identifier", fmt.Sprintf("Expected import identifier with format: profile_id/service. %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("profile_id"), profileID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service"), service)...)
}

func (r *ServiceResource) applyService(ctx context.Context, data *ServiceResourceModel, diags *diag.Diagnostics) {
	params := controld.UpdateProfileServiceParams{
		ProfileID: data.ProfileID.ValueString(),
		Service:   data.Service.ValueString(),
		Do:        controld.DoType(data.Do.ValueInt64()),
		Status:    controld.IntBool(data.Status.ValueBool()),
	}
	if !data.Via.IsNull() {
		v := data.Via.ValueString()
		params.Via = &v
	}
	if !data.ViaV6.IsNull() {
		v := data.ViaV6.ValueString()
		params.ViaV6 = &v
	}

	actions, err := r.client.UpdateProfileService(ctx, params)
	if err != nil {
		diags.AddError("Unable to Update Service", err.Error())
		return
	}
	if len(actions) != 1 {
		diags.AddError("Unexpected ControlD API Response", fmt.Sprintf("Expected exactly one service action to be returned, got %d.", len(actions)))
		return
	}

	updateModelFromServiceAction(data, actions[0])
}

func updateModelFromServiceAction(data *ServiceResourceModel, action controld.Action) {
	data.Do = types.Int64Value(int64(action.Do))
	data.Status = types.BoolValue(bool(action.Status))
}
