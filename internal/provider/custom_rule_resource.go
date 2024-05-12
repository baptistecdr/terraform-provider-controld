// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	controld "github.com/baptistecdr/controld-go"
)

var (
	_ resource.Resource                = &CustomRuleResource{}
	_ resource.ResourceWithConfigure   = &CustomRuleResource{}
	_ resource.ResourceWithImportState = &CustomRuleResource{}
)

func NewCustomRuleResource() resource.Resource {
	return &CustomRuleResource{}
}

// CustomRuleResource manages a single custom rule (hostname-based DNS action)
// within a ControlD profile.
type CustomRuleResource struct {
	client *controld.API
}

type CustomRuleResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ProfileID types.String `tfsdk:"profile_id"`
	Hostname  types.String `tfsdk:"hostname"`
	Do        types.Int64  `tfsdk:"do"`
	Status    types.Bool   `tfsdk:"status"`
	Via       types.String `tfsdk:"via"`
	ViaV6     types.String `tfsdk:"via_v6"`
	Group     types.Int64  `tfsdk:"group"`
	Order     types.Int64  `tfsdk:"order"`
}

func (r *CustomRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_rule"
}

func (r *CustomRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single custom rule within a ControlD profile: a hostname-based DNS action (block, bypass, spoof, or redirect).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of this resource, equal to `hostname`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"profile_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the profile this rule belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hostname": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The hostname this rule applies to. Changing this forces a new resource to be created.",
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
				MarkdownDescription: "Whether the rule is enabled.",
			},
			"via": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The destination IP/hostname used when `do` is `spoof` or `redirect`.",
			},
			"via_v6": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The IPv6 destination used when `do` is `spoof` or `redirect`.",
			},
			"group": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "The identifier of the rule folder (`controld_rule_folder`) this rule belongs to. `0` means no folder.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"order": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The position of the rule within its folder.",
			},
		},
	}
}

func (r *CustomRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	r.client = client
}

func (r *CustomRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CustomRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := controld.CreateProfileCustomRuleParams{
		ProfileID: data.ProfileID.ValueString(),
		Do:        controld.DoType(data.Do.ValueInt64()),
		Status:    controld.IntBool(data.Status.ValueBool()),
		Hostnames: []string{data.Hostname.ValueString()},
	}
	if !data.Via.IsNull() {
		v := data.Via.ValueString()
		params.Via = &v
	}
	if !data.ViaV6.IsNull() {
		v := data.ViaV6.ValueString()
		params.ViaV6 = &v
	}
	if !data.Group.IsNull() && data.Group.ValueInt64() != 0 {
		v := int(data.Group.ValueInt64())
		params.Group = &v
	}

	rules, err := r.client.CreateProfileCustomRule(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Custom Rule", err.Error())
		return
	}
	if len(rules) != 1 {
		resp.Diagnostics.AddError("Unexpected ControlD API Response", fmt.Sprintf("Expected exactly one custom rule to be created, got %d.", len(rules)))
		return
	}

	data.ID = data.Hostname
	data.Do = types.Int64Value(int64(rules[0].Do))
	data.Status = types.BoolValue(bool(rules[0].Status))
	if rules[0].Order != nil {
		data.Order = types.Int64Value(int64(*rules[0].Order))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CustomRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := findCustomRule(ctx, r.client, data.ProfileID.ValueString(), int(data.Group.ValueInt64()), data.Hostname.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Custom Rule", err.Error())
		return
	}
	if rule == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Do = types.Int64Value(int64(rule.Action.Do))
	data.Status = types.BoolValue(bool(rule.Action.Status))
	data.Group = types.Int64Value(int64(rule.Group))
	data.Order = types.Int64Value(int64(rule.Order))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CustomRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := controld.UpdateProfileCustomRuleParams{
		ProfileID: data.ProfileID.ValueString(),
		Do:        controld.DoType(data.Do.ValueInt64()),
		Status:    controld.IntBool(data.Status.ValueBool()),
		Hostnames: []string{data.Hostname.ValueString()},
	}
	if !data.Via.IsNull() {
		v := data.Via.ValueString()
		params.Via = &v
	}
	if !data.ViaV6.IsNull() {
		v := data.ViaV6.ValueString()
		params.ViaV6 = &v
	}
	if !data.Group.IsNull() && data.Group.ValueInt64() != 0 {
		v := int(data.Group.ValueInt64())
		params.Group = &v
	}

	rules, err := r.client.UpdateProfileCustomRule(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Custom Rule", err.Error())
		return
	}
	if len(rules) != 1 {
		resp.Diagnostics.AddError("Unexpected ControlD API Response", fmt.Sprintf("Expected exactly one custom rule to be updated, got %d.", len(rules)))
		return
	}

	data.Do = types.Int64Value(int64(rules[0].Do))
	data.Status = types.BoolValue(bool(rules[0].Status))
	if rules[0].Order != nil {
		data.Order = types.Int64Value(int64(*rules[0].Order))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CustomRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteProfileCustomRule(ctx, controld.DeleteProfileCustomRuleParams{
		ProfileID: data.ProfileID.ValueString(),
		Hostname:  data.Hostname.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Delete Custom Rule", err.Error())
		return
	}
}

// ImportState accepts a "profile_id/hostname" identifier. Since
// ListProfileCustomRules is scoped to a single rule folder, the folder the
// rule lives in is located up front (searching every folder in the profile if
// necessary) so the subsequent automatic Read call can find it directly.
func (r *CustomRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	profileID, hostname, err := splitImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected Import Identifier", fmt.Sprintf("Expected import identifier with format: profile_id/hostname. %s", err))
		return
	}

	rule, err := findCustomRuleAnyFolder(ctx, r.client, profileID, hostname)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Import Custom Rule", err.Error())
		return
	}
	if rule == nil {
		resp.Diagnostics.AddError("Custom Rule Not Found", fmt.Sprintf("No custom rule for hostname %q found in profile %q.", hostname, profileID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), hostname)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("profile_id"), profileID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("hostname"), hostname)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("do"), int64(rule.Action.Do))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status"), bool(rule.Action.Status))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group"), int64(rule.Group))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("order"), int64(rule.Order))...)
}

// findCustomRule searches for hostname within a single rule folder identified
// by folderID (0 for the default/ungrouped folder), returning nil if not found.
func findCustomRule(ctx context.Context, client *controld.API, profileID string, folderID int, hostname string) (*controld.Rule, error) {
	rules, err := listCustomRulesInFolder(ctx, client, profileID, folderID)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		if rules[i].PK == hostname {
			return &rules[i], nil
		}
	}
	return nil, nil
}

// listCustomRulesInFolder lists the custom rules in a single folder. The
// ControlD API rejects "0" as a folder id for the default/ungrouped folder
// (it must be listed via an empty folder path segment instead), but
// controld-go's ListProfileCustomRules refuses an empty FolderID, so folder 0
// is fetched with a raw request instead.
func listCustomRulesInFolder(ctx context.Context, client *controld.API, profileID string, folderID int) ([]controld.Rule, error) {
	if folderID != 0 {
		return client.ListProfileCustomRules(ctx, controld.ListProfileCustomRulesParams{
			ProfileID: profileID,
			FolderID:  strconv.Itoa(folderID),
		})
	}

	raw, err := client.Raw(ctx, http.MethodGet, fmt.Sprintf("/profiles/%s/rules/", profileID), nil, nil)
	if err != nil {
		return nil, err
	}
	var body struct {
		Rules []controld.Rule `json:"rules"`
	}
	if err := json.Unmarshal(raw.Body, &body); err != nil {
		return nil, fmt.Errorf("unable to unmarshal custom rules: %w", err)
	}
	return body.Rules, nil
}

// findCustomRuleAnyFolder searches for hostname across every rule folder in
// the profile (root folder "0" first, as the common case), used when the
// folder is not already known (i.e. during import).
func findCustomRuleAnyFolder(ctx context.Context, client *controld.API, profileID, hostname string) (*controld.Rule, error) {
	if rule, err := findCustomRule(ctx, client, profileID, 0, hostname); err != nil {
		return nil, err
	} else if rule != nil {
		return rule, nil
	}

	groups, err := client.ListProfileRuleFolders(ctx, controld.ListProfileRuleFoldersParams{ProfileID: profileID})
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		rule, err := findCustomRule(ctx, client, profileID, g.PK, hostname)
		if err != nil {
			return nil, err
		}
		if rule != nil {
			return rule, nil
		}
	}
	return nil, nil
}
