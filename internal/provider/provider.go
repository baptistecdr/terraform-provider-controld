// Copyright (c) baptistecdr
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	controld "github.com/baptistecdr/controld-go"
)

// Ensure ControldProvider satisfies various provider interfaces.
var _ provider.Provider = &ControldProvider{}

// ControldProvider defines the provider implementation.
type ControldProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ControldProviderModel describes the provider data model.
type ControldProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
}

func (p *ControldProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "controld"
	resp.Version = p.version
}

func (p *ControldProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The ControlD provider allows you to manage your [ControlD](https://controld.com/) organization: profiles, devices, custom rules, filters, and more, through the [ControlD API](https://docs.controld.com/reference).",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				MarkdownDescription: "The API token used to authenticate against the ControlD API. Can also be set using the `CONTROLD_API_TOKEN` environment variable. You can generate a token from the [ControlD dashboard](https://controld.com/dashboard/settings/api).",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *ControldProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ControldProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiToken := os.Getenv("CONTROLD_API_TOKEN")
	if !data.APIToken.IsNull() && data.APIToken.ValueString() != "" {
		apiToken = data.APIToken.ValueString()
	}

	if apiToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing ControlD API Token",
			"The provider cannot create the ControlD API client as there is a missing or empty value for the ControlD API token. "+
				"Set the api_token value in the configuration or use the CONTROLD_API_TOKEN environment variable.",
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := controld.New(apiToken, controld.UserAgent("terraform-provider-controld/"+p.version))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create ControlD API Client",
			"An unexpected error occurred when creating the ControlD API client: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ControldProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProfileResource,
		NewDeviceResource,
		NewDefaultRuleResource,
		NewRuleFolderResource,
		NewCustomRuleResource,
		NewServiceResource,
		NewFilterResource,
	}
}

func (p *ControldProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProfileDataSource,
		NewProfilesDataSource,
		NewDeviceDataSource,
		NewDevicesDataSource,
		NewRuleFoldersDataSource,
		NewServicesDataSource,
		NewNativeFiltersDataSource,
		NewExternalFiltersDataSource,
		NewUserDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ControldProvider{
			version: version,
		}
	}
}
