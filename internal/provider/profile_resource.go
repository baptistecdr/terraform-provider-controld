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
	_ resource.Resource                = &ProfileResource{}
	_ resource.ResourceWithConfigure   = &ProfileResource{}
	_ resource.ResourceWithImportState = &ProfileResource{}
)

func NewProfileResource() resource.Resource {
	return &ProfileResource{}
}

// ProfileResource manages a ControlD DNS profile.
type ProfileResource struct {
	client *controld.API
}

// ProfileResourceModel describes the resource data model.
//
// disable_ttl, lock_status, lock_message, and password are write-only from the
// API's perspective: the ControlD API never returns them back, so Read()
// leaves whatever is already in state untouched for those attributes.
type ProfileResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	CloneProfileID types.String `tfsdk:"clone_profile_id"`
	DisableTTL     types.Int64  `tfsdk:"disable_ttl"`
	LockStatus     types.Bool   `tfsdk:"lock_status"`
	LockMessage    types.String `tfsdk:"lock_message"`
	Password       types.String `tfsdk:"password"`
	Updated        types.Int64  `tfsdk:"updated"`
}

func (r *ProfileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_profile"
}

func (r *ProfileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a ControlD DNS profile. A profile groups filters, custom rules, and services that can be applied to one or more devices.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The profile identifier (PK).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the profile.",
			},
			"clone_profile_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The identifier of an existing profile to clone when creating this profile. Changing this forces a new resource to be created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"disable_ttl": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "The number of seconds after which the profile lock automatically expires. The ControlD API echoes this back as an update, but the controld-go client library this provider is built on doesn't expose it, so it can't be read back into state.",
			},
			"lock_status": schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: "Whether the profile is locked, preventing further changes (including deletion). The ControlD API echoes this back on update, but the controld-go client library this provider is built on doesn't expose it, so it can't be read back into state.\n\n" +
					"**Warning:** setting this to `true` can leave the profile stuck. The `password` set alongside it does *not* reliably unlock the profile again through the API (confirmed against the live API: the same password is rejected with \"Please provide a valid password\"); unlocking may require your ControlD account login password via the web dashboard instead. `terraform destroy` will fail with the API's locked-profile error until the profile is unlocked by some other means. Avoid setting this attribute unless you have a tested way to unlock the profile again.",
			},
			"lock_message": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The message shown when the profile is locked. The ControlD API echoes this back on update, but the controld-go client library this provider is built on doesn't expose it, so it can't be read back into state.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The password associated with the profile. Never returned by the API. See the warning on `lock_status`: this value has not been confirmed to actually unlock a locked profile.",
			},
			"updated": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The Unix timestamp of the last update to the profile.",
			},
		},
	}
}

func (r *ProfileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = client
}

func (r *ProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProfileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := controld.CreateProfileParams{
		Name: data.Name.ValueString(),
	}
	if !data.CloneProfileID.IsNull() {
		cloneID := data.CloneProfileID.ValueString()
		params.CloneProfileID = &cloneID
	}

	profiles, err := r.client.CreateProfile(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Profile", err.Error())
		return
	}
	if len(profiles) != 1 {
		resp.Diagnostics.AddError("Unexpected ControlD API Response", fmt.Sprintf("Expected exactly one profile to be created, got %d.", len(profiles)))
		return
	}

	data.ID = types.StringValue(profiles[0].PK)
	data.Updated = types.Int64Value(profiles[0].Updated.Unix())

	if err := r.applyWriteOnlyOptions(ctx, data); err != nil {
		resp.Diagnostics.AddError("Unable to Apply Profile Options", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProfileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profiles, err := r.client.ListProfiles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Profile", err.Error())
		return
	}

	var found *controld.Profile
	for i := range profiles {
		if profiles[i].PK == data.ID.ValueString() {
			found = &profiles[i]
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(found.Name)
	data.Updated = types.Int64Value(found.Updated.Unix())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProfileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	params := controld.UpdateProfileParams{
		ProfileID: data.ID.ValueString(),
		Name:      &name,
	}

	profiles, err := r.client.UpdateProfile(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Profile", err.Error())
		return
	}
	if len(profiles) != 1 {
		resp.Diagnostics.AddError("Unexpected ControlD API Response", fmt.Sprintf("Expected exactly one profile to be updated, got %d.", len(profiles)))
		return
	}

	data.Updated = types.Int64Value(profiles[0].Updated.Unix())

	if err := r.applyWriteOnlyOptions(ctx, data); err != nil {
		resp.Diagnostics.AddError("Unable to Apply Profile Options", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProfileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProfile(ctx, controld.DeleteProfileParams{
		ProfileID: data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Delete Profile", err.Error())
		return
	}
}

func (r *ProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyWriteOnlyOptions pushes the disable_ttl/lock_status/lock_message/password
// attributes via UpdateProfile whenever at least one of them is set, since the
// ControlD API only accepts them alongside a profile update.
func (r *ProfileResource) applyWriteOnlyOptions(ctx context.Context, data ProfileResourceModel) error {
	if data.DisableTTL.IsNull() && data.LockStatus.IsNull() && data.LockMessage.IsNull() && data.Password.IsNull() {
		return nil
	}

	params := controld.UpdateProfileParams{
		ProfileID: data.ID.ValueString(),
	}
	if !data.DisableTTL.IsNull() {
		v := int(data.DisableTTL.ValueInt64())
		params.DisableTTL = &v
	}
	if !data.LockStatus.IsNull() {
		v := controld.IntBool(data.LockStatus.ValueBool())
		params.LockStatus = &v
	}
	if !data.LockMessage.IsNull() {
		v := data.LockMessage.ValueString()
		params.LockMessage = &v
	}
	if !data.Password.IsNull() {
		v := data.Password.ValueString()
		params.Password = &v
	}

	_, err := r.client.UpdateProfile(ctx, params)
	return err
}
