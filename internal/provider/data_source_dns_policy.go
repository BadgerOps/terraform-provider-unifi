package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ datasource.DataSource              = (*dnsPolicyDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*dnsPolicyDataSource)(nil)
)

type dnsPolicyDataSource struct {
	clientProvider *providerData
}

type dnsPolicyDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	SiteID           types.String `tfsdk:"site_id"`
	Type             types.String `tfsdk:"type"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	Domain           types.String `tfsdk:"domain"`
	IPv4Address      types.String `tfsdk:"ipv4_address"`
	IPv6Address      types.String `tfsdk:"ipv6_address"`
	TargetDomain     types.String `tfsdk:"target_domain"`
	MailServerDomain types.String `tfsdk:"mail_server_domain"`
	Priority         types.Int64  `tfsdk:"priority"`
	Text             types.String `tfsdk:"text"`
	ServerDomain     types.String `tfsdk:"server_domain"`
	Service          types.String `tfsdk:"service"`
	Protocol         types.String `tfsdk:"protocol"`
	Port             types.Int64  `tfsdk:"port"`
	Weight           types.Int64  `tfsdk:"weight"`
	IPAddress        types.String `tfsdk:"ip_address"`
	TTLSeconds       types.Int64  `tfsdk:"ttl_seconds"`
}

func NewDNSPolicyDataSource() datasource.DataSource {
	return &dnsPolicyDataSource{}
}

func (d *dnsPolicyDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_dns_policy"
}

func (d *dnsPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi DNS policy by `id` or `domain` within a site. Set `type` to disambiguate shared domains.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"site_id": schema.StringAttribute{Required: true},
			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"enabled":            schema.BoolAttribute{Computed: true},
			"domain":             schema.StringAttribute{Optional: true, Computed: true},
			"ipv4_address":       schema.StringAttribute{Computed: true},
			"ipv6_address":       schema.StringAttribute{Computed: true},
			"target_domain":      schema.StringAttribute{Computed: true},
			"mail_server_domain": schema.StringAttribute{Computed: true},
			"priority":           schema.Int64Attribute{Computed: true},
			"text":               schema.StringAttribute{Computed: true},
			"server_domain":      schema.StringAttribute{Computed: true},
			"service":            schema.StringAttribute{Computed: true},
			"protocol":           schema.StringAttribute{Computed: true},
			"port":               schema.Int64Attribute{Computed: true},
			"weight":             schema.Int64Attribute{Computed: true},
			"ip_address":         schema.StringAttribute{Computed: true},
			"ttl_seconds":        schema.Int64Attribute{Computed: true},
		},
	}
}

func (d *dnsPolicyDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *dnsPolicyDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state dnsPolicyDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	lookupByID := !state.ID.IsNull() && state.ID.ValueString() != ""
	lookupByDomain := !state.Domain.IsNull() && state.Domain.ValueString() != ""
	if lookupByID == lookupByDomain {
		response.Diagnostics.AddError("Invalid DNS policy lookup arguments", "Exactly one of `id` or `domain` must be set.")
		return
	}

	policies, err := d.clientProvider.client.ListDNSPolicies(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list DNS policies", err.Error())
		return
	}

	var matches []client.DNSPolicy
	for _, policy := range policies {
		switch {
		case lookupByID:
			if policy.ID == state.ID.ValueString() {
				matches = append(matches, policy)
			}
		case lookupByDomain:
			if policy.Domain != nil && *policy.Domain == state.Domain.ValueString() && (state.Type.IsNull() || state.Type.ValueString() == "" || policy.Type == state.Type.ValueString()) {
				matches = append(matches, policy)
			}
		}
	}

	switch len(matches) {
	case 0:
		response.Diagnostics.AddError("DNS policy not found", "No DNS policy matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple DNS policies matched", fmt.Sprintf("%d DNS policies matched the given selector", len(matches)))
		return
	}

	policy := matches[0]
	model := dnsPolicyDataSourceModel{
		ID:               types.StringValue(policy.ID),
		SiteID:           state.SiteID,
		Type:             types.StringValue(policy.Type),
		Enabled:          types.BoolValue(policy.Enabled),
		Domain:           nullableString(policy.Domain),
		IPv4Address:      nullableString(policy.IPv4Address),
		IPv6Address:      nullableString(policy.IPv6Address),
		TargetDomain:     nullableString(policy.TargetDomain),
		MailServerDomain: nullableString(policy.MailServerDomain),
		Priority:         nullableInt64(policy.Priority),
		Text:             nullableString(policy.Text),
		ServerDomain:     nullableString(policy.ServerDomain),
		Service:          nullableString(policy.Service),
		Protocol:         nullableString(policy.Protocol),
		Port:             nullableInt64(policy.Port),
		Weight:           nullableInt64(policy.Weight),
		IPAddress:        nullableString(policy.IPAddress),
		TTLSeconds:       nullableInt64(policy.TTLSeconds),
	}

	response.Diagnostics.Append(response.State.Set(ctx, &model)...)
}
