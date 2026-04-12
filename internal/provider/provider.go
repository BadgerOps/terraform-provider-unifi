package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var _ provider.Provider = (*unifiProvider)(nil)

type unifiProvider struct {
	version string
}

type providerModel struct {
	APIURL        types.String `tfsdk:"api_url"`
	APIKey        types.String `tfsdk:"api_key"`
	AllowInsecure types.Bool   `tfsdk:"allow_insecure"`
}

type providerData struct {
	client *client.Client
}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &unifiProvider{}
	}
}

func (p *unifiProvider) Metadata(_ context.Context, _ provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "unifi"
	response.Version = p.version
}

func (p *unifiProvider) Schema(_ context.Context, _ provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for the UniFi Network integration API.",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Base URL for the UniFi Network API. The provider appends `/integration` when it is not already present.",
			},
			"api_key": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "API key created in the UniFi Network integrations UI.",
			},
			"allow_insecure": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Disable TLS certificate verification. Only use this against trusted development systems.",
			},
		},
	}
}

func (p *unifiProvider) Configure(ctx context.Context, request provider.ConfigureRequest, response *provider.ConfigureResponse) {
	var data providerModel
	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	apiClient, err := client.New(client.Config{
		BaseURL:       data.APIURL.ValueString(),
		APIKey:        data.APIKey.ValueString(),
		AllowInsecure: data.AllowInsecure.ValueBool(),
		UserAgent:     fmt.Sprintf("terraform-provider-unifi/%s", p.version),
	})
	if err != nil {
		response.Diagnostics.AddError("Unable to configure UniFi client", err.Error())
		return
	}

	providerData := &providerData{client: apiClient}
	response.DataSourceData = providerData
	response.ResourceData = providerData
}

func (p *unifiProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSiteDataSource,
	}
}

func (p *unifiProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}
