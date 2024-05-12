// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	_ resource.Resource                = &RuleFolderResource{}
	_ resource.ResourceWithConfigure   = &RuleFolderResource{}
	_ resource.ResourceWithImportState = &RuleFolderResource{}
)

func NewRuleFolderResource() resource.Resource {
	return &RuleFolderResource{}
}

// RuleFolderResource manages a custom rule folder (group) within a ControlD profile.
type RuleFolderResource struct {
	client *controld.API
}

type RuleFolderResourceModel struct {
	ID         types.String `tfsdk:"id"`
	ProfileID  types.String `tfsdk:"profile_id"`
	Name       types.String `tfsdk:"name"`
	Do         types.Int64  `tfsdk:"do"`
	Via        types.String `tfsdk:"via"`
	Status     types.Bool   `tfsdk:"status"`
	RulesCount types.Int64  `tfsdk:"rules_count"`
}

func (r *RuleFolderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_folder"
}

func (r *RuleFolderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom rule folder (group) within a ControlD profile. Rule folders let you group and bulk-manage custom rules under a single action.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The rule folder identifier (PK).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the profile this rule folder belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the rule folder. Changing this forces a new resource to be created, as the ControlD API does not support renaming a folder.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"do": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "The action applied to hostnames in this folder: `0` (block), `1` (bypass), `2` (spoof), or `3` (redirect). Only present in state once explicitly set; the API omits it entirely until then.",
				Validators: []validator.Int64{
					int64validator.Between(0, 3),
				},
			},
			"via": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The destination IP/hostname used when `do` is `spoof` or `redirect`. Not readable back from the API.",
			},
			"status": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the rule folder is enabled.",
			},
			"rules_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The number of rules contained in this folder.",
			},
		},
	}
}

func (r *RuleFolderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = client
}

func (r *RuleFolderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RuleFolderResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := controld.CreateProfileRuleFolderParams{
		ProfileID: data.ProfileID.ValueString(),
		Name:      data.Name.ValueString(),
	}
	if !data.Do.IsNull() && !data.Do.IsUnknown() {
		v := controld.DoType(data.Do.ValueInt64())
		params.Do = &v
	}
	if !data.Via.IsNull() && !data.Via.IsUnknown() {
		v := data.Via.ValueString()
		params.Via = &v
	}
	if !data.Status.IsNull() && !data.Status.IsUnknown() {
		v := controld.IntBool(data.Status.ValueBool())
		params.Status = &v
	}

	groups, err := r.client.CreateProfileRuleFolder(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Rule Folder", err.Error())
		return
	}

	group, err := findRuleFolder(groups, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ControlD API Response", err.Error())
		return
	}

	updateModelFromGroup(&data, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RuleFolderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RuleFolderResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groups, err := r.client.ListProfileRuleFolders(ctx, controld.ListProfileRuleFoldersParams{
		ProfileID: data.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Rule Folder", err.Error())
		return
	}

	folderID := data.ID.ValueString()
	var found *controld.Group
	for i := range groups {
		if fmt.Sprintf("%d", groups[i].PK) == folderID {
			found = &groups[i]
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	updateModelFromGroup(&data, *found)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RuleFolderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RuleFolderResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := controld.UpdateProfileRuleFolderParams{
		ProfileID: data.ProfileID.ValueString(),
		FolderID:  data.ID.ValueString(),
	}
	if !data.Do.IsNull() && !data.Do.IsUnknown() {
		v := controld.DoType(data.Do.ValueInt64())
		params.Do = &v
	}
	if !data.Via.IsNull() && !data.Via.IsUnknown() {
		v := data.Via.ValueString()
		params.Via = &v
	}
	if !data.Status.IsNull() && !data.Status.IsUnknown() {
		v := controld.IntBool(data.Status.ValueBool())
		params.Status = &v
	}

	groups, err := r.client.UpdateProfileRuleFolder(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Rule Folder", err.Error())
		return
	}

	group, err := findRuleFolder(groups, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ControlD API Response", err.Error())
		return
	}

	updateModelFromGroup(&data, group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RuleFolderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RuleFolderResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProfileRuleFolder(ctx, controld.DeleteProfileRuleFolderParams{
		ProfileID: data.ProfileID.ValueString(),
		FolderID:  data.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Delete Rule Folder", err.Error())
		return
	}
}

func (r *RuleFolderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	profileID, folderID, err := splitImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected Import Identifier", fmt.Sprintf("Expected import identifier with format: profile_id/folder_id. %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("profile_id"), profileID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), folderID)...)
}

// findRuleFolder locates the folder matching name in groups. CreateProfileRuleFolder
// and UpdateProfileRuleFolder return the full list of folders in the profile,
// so the target folder must be located by name.
func findRuleFolder(groups []controld.Group, name string) (controld.Group, error) {
	for _, g := range groups {
		if g.Group == name {
			return g, nil
		}
	}
	return controld.Group{}, fmt.Errorf("no rule folder named %q found in API response", name)
}

func updateModelFromGroup(data *RuleFolderResourceModel, group controld.Group) {
	data.ID = types.StringValue(fmt.Sprintf("%d", group.PK))
	data.Name = types.StringValue(group.Group)
	data.Status = types.BoolValue(bool(group.Action.Status))
	data.RulesCount = types.Int64Value(int64(group.Count))
	if group.Action.Do != nil {
		data.Do = types.Int64Value(int64(*group.Action.Do))
	}
}
