// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

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
	_ resource.Resource                = &DefaultRuleResource{}
	_ resource.ResourceWithConfigure   = &DefaultRuleResource{}
	_ resource.ResourceWithImportState = &DefaultRuleResource{}
)

func NewDefaultRuleResource() resource.Resource {
	return &DefaultRuleResource{}
}

// DefaultRuleResource manages the default (fallback) rule of a ControlD
// profile. Every profile has exactly one default rule, so this resource is a
// singleton keyed by profile_id: Create and Update both call UpdateProfileDefaultRule,
// and Delete resets the rule to the ControlD default (bypass, enabled) rather
// than removing anything.
type DefaultRuleResource struct {
	client *controld.API
}

type DefaultRuleResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ProfileID types.String `tfsdk:"profile_id"`
	Do        types.Int64  `tfsdk:"do"`
	Status    types.Bool   `tfsdk:"status"`
	Via       types.String `tfsdk:"via"`
}

func (r *DefaultRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_default_rule"
}

func (r *DefaultRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the default (fallback) rule of a ControlD profile: the action applied to any query that isn't matched by a more specific filter, service, or custom rule. Every profile has exactly one default rule, so this resource is a singleton per `profile_id`; deleting it resets the rule to the ControlD default (bypass, enabled).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of this resource, equal to `profile_id`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the profile this default rule belongs to.",
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
				MarkdownDescription: "Whether the default rule is enabled.",
			},
			"via": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The destination IP/hostname used when `do` is `spoof` or `redirect`.",
			},
		},
	}
}

func (r *DefaultRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = client
}

func (r *DefaultRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DefaultRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applyRule(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = data.ProfileID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DefaultRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DefaultRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.ListProfileDefaultRule(ctx, controld.ListProfileDefaultRuleParams{
		ProfileID: data.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Default Rule", err.Error())
		return
	}

	data.Do = types.Int64Value(int64(rule.Do))
	data.Status = types.BoolValue(bool(rule.Status))
	if rule.Via != nil {
		data.Via = types.StringValue(*rule.Via)
	} else {
		data.Via = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DefaultRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DefaultRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applyRule(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DefaultRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DefaultRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateProfileDefaultRule(ctx, controld.UpdateProfileDefaultRuleParams{
		ProfileID: data.ProfileID.ValueString(),
		Do:        controld.Bypass,
		Status:    controld.IntBool(true),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Reset Default Rule", err.Error())
		return
	}
}

func (r *DefaultRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("profile_id"), req, resp)
}

func (r *DefaultRuleResource) applyRule(ctx context.Context, data *DefaultRuleResourceModel, diags *diag.Diagnostics) {
	params := controld.UpdateProfileDefaultRuleParams{
		ProfileID: data.ProfileID.ValueString(),
		Do:        controld.DoType(data.Do.ValueInt64()),
		Status:    controld.IntBool(data.Status.ValueBool()),
	}
	if !data.Via.IsNull() && !data.Via.IsUnknown() {
		v := data.Via.ValueString()
		params.Via = &v
	}

	rule, err := r.client.UpdateProfileDefaultRule(ctx, params)
	if err != nil {
		diags.AddError("Unable to Update Default Rule", err.Error())
		return
	}

	data.Do = types.Int64Value(int64(rule.Do))
	data.Status = types.BoolValue(bool(rule.Status))
	if rule.Via != nil {
		data.Via = types.StringValue(*rule.Via)
	} else {
		data.Via = types.StringNull()
	}
}
