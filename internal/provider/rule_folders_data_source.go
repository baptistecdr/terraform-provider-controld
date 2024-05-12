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
	_ datasource.DataSource              = &RuleFoldersDataSource{}
	_ datasource.DataSourceWithConfigure = &RuleFoldersDataSource{}
)

func NewRuleFoldersDataSource() datasource.DataSource {
	return &RuleFoldersDataSource{}
}

// RuleFoldersDataSource lists the custom rule folders (groups) of a ControlD profile.
type RuleFoldersDataSource struct {
	client *controld.API
}

type RuleFolderDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Do         types.Int64  `tfsdk:"do"`
	Status     types.Bool   `tfsdk:"status"`
	RulesCount types.Int64  `tfsdk:"rules_count"`
}

type RuleFoldersDataSourceModel struct {
	ID          types.String                `tfsdk:"id"`
	ProfileID   types.String                `tfsdk:"profile_id"`
	RuleFolders []RuleFolderDataSourceModel `tfsdk:"rule_folders"`
}

func (d *RuleFoldersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_folders"
}

func (d *RuleFoldersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the custom rule folders (groups) of a ControlD profile.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of this data source, equal to `profile_id`.",
			},
			"profile_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the profile to list rule folders for.",
			},
			"rule_folders": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The list of rule folders.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The rule folder identifier (PK), usable as `group` in a `controld_custom_rule` resource.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the rule folder.",
						},
						"do": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The action applied to hostnames in this folder.",
						},
						"status": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether the rule folder is enabled.",
						},
						"rules_count": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The number of rules contained in this folder.",
						},
					},
				},
			},
		},
	}
}

func (d *RuleFoldersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = client
}

func (d *RuleFoldersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RuleFoldersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groups, err := d.client.ListProfileRuleFolders(ctx, controld.ListProfileRuleFoldersParams{
		ProfileID: data.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to List Rule Folders", err.Error())
		return
	}

	data.ID = data.ProfileID
	data.RuleFolders = make([]RuleFolderDataSourceModel, 0, len(groups))
	for _, g := range groups {
		item := RuleFolderDataSourceModel{
			ID:         types.StringValue(fmt.Sprintf("%d", g.PK)),
			Name:       types.StringValue(g.Group),
			Status:     types.BoolValue(bool(g.Action.Status)),
			RulesCount: types.Int64Value(int64(g.Count)),
		}
		if g.Action.Do != nil {
			item.Do = types.Int64Value(int64(*g.Action.Do))
		}
		data.RuleFolders = append(data.RuleFolders, item)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
