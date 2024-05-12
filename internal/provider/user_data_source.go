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
	_ datasource.DataSource              = &UserDataSource{}
	_ datasource.DataSourceWithConfigure = &UserDataSource{}
)

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

// UserDataSource fetches details about the ControlD account tied to the
// configured API token.
type UserDataSource struct {
	client *controld.API
}

type UserDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Email       types.String `tfsdk:"email"`
	Status      types.Bool   `tfsdk:"status"`
	RuleProfile types.String `tfsdk:"rule_profile"`
	ResolverUID types.String `tfsdk:"resolver_uid"`
	ProxyAccess types.Bool   `tfsdk:"proxy_access"`
	TwoFA       types.Bool   `tfsdk:"twofa"`
}

func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches details about the ControlD account tied to the configured API token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The account identifier (PK).",
			},
			"email": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The account email address.",
			},
			"status": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the account is active.",
			},
			"rule_profile": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The identifier of the account's default profile.",
			},
			"resolver_uid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The account's default resolver identifier.",
			},
			"proxy_access": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the account has proxy access.",
			},
			"twofa": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether two-factor authentication is enabled on the account.",
			},
		},
	}
}

func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, ok := configureClient(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}
	d.client = client
}

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := d.client.ListUser(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read User", err.Error())
		return
	}

	data.ID = types.StringValue(user.PK)
	data.Email = types.StringValue(user.Email)
	data.Status = types.BoolValue(bool(user.Status))
	data.RuleProfile = types.StringValue(user.RuleProfile)
	data.ResolverUID = types.StringValue(user.ResolverUid)
	data.ProxyAccess = types.BoolValue(bool(user.ProxyAccess))
	data.TwoFA = types.BoolValue(bool(user.Twofa))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
