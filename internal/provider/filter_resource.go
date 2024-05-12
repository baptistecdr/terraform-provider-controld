// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	controld "github.com/baptistecdr/controld-go"
)

var (
	_ resource.Resource                = &FilterResource{}
	_ resource.ResourceWithConfigure   = &FilterResource{}
	_ resource.ResourceWithImportState = &FilterResource{}
)

func NewFilterResource() resource.Resource {
	return &FilterResource{}
}

// FilterResource toggles a native ControlD filter (e.g. a malware or ads
// filter list) for a profile. Filters are predefined by ControlD: there is no
// create/delete endpoint, only an update, so Create and Update both call
// UpdateProfileFilter, and Delete disables the filter.
type FilterResource struct {
	client *controld.API
}

type FilterResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ProfileID types.String `tfsdk:"profile_id"`
	Filter    types.String `tfsdk:"filter"`
	Status    types.Bool   `tfsdk:"status"`
}

func (r *FilterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_filter"
}

func (r *FilterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Enables or disables a native ControlD filter list (e.g. malware, ads) for a profile. Filters are predefined by ControlD (see the `controld_native_filters` data source for the full catalog); deleting this resource disables the filter.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of this resource, in the form `profile_id/filter`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the profile this filter belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"filter": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The filter identifier (PK), as returned by the `controld_native_filters` data source.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether the filter is enabled.",
			},
		},
	}
}

func (r *FilterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = client
}

func (r *FilterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FilterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyFilter(ctx, data); err != nil {
		resp.Diagnostics.AddError("Unable to Update Filter", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.ProfileID.ValueString(), data.Filter.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FilterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FilterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filters, err := r.client.ListProfileNativeFilters(ctx, controld.ListProfileFiltersParams{
		ProfileID: data.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Filter", err.Error())
		return
	}

	var found *controld.Filter
	for i := range filters {
		if filters[i].PK == data.Filter.ValueString() {
			found = &filters[i]
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Status = types.BoolValue(bool(found.Status))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FilterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FilterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyFilter(ctx, data); err != nil {
		resp.Diagnostics.AddError("Unable to Update Filter", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FilterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FilterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateProfileFilter(ctx, controld.UpdateProfileFilterParams{
		ProfileID: data.ProfileID.ValueString(),
		Filter:    data.Filter.ValueString(),
		Status:    controld.IntBool(false),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Disable Filter", err.Error())
		return
	}
}

func (r *FilterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	profileID, filter, err := splitImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected Import Identifier", fmt.Sprintf("Expected import identifier with format: profile_id/filter. %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("profile_id"), profileID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("filter"), filter)...)
}

func (r *FilterResource) applyFilter(ctx context.Context, data FilterResourceModel) error {
	_, err := r.client.UpdateProfileFilter(ctx, controld.UpdateProfileFilterParams{
		ProfileID: data.ProfileID.ValueString(),
		Filter:    data.Filter.ValueString(),
		Status:    controld.IntBool(data.Status.ValueBool()),
	})
	return err
}
