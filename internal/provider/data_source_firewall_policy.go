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
	_ datasource.DataSource              = (*firewallPolicyDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*firewallPolicyDataSource)(nil)
)

type firewallPolicyDataSource struct {
	clientProvider *providerData
}

func NewFirewallPolicyDataSource() datasource.DataSource {
	return &firewallPolicyDataSource{}
}

func (d *firewallPolicyDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_firewall_policy"
}

func (d *firewallPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	resourceSchema := &resourceFirewallPolicySchemaHolder{}
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi firewall policy by `id` or `name` within a site.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"site_id":                 schema.StringAttribute{Required: true},
			"enabled":                 resourceSchema.enabled(),
			"name":                    schema.StringAttribute{Optional: true, Computed: true},
			"description":             resourceSchema.description(),
			"action":                  resourceSchema.action(),
			"allow_return_traffic":    resourceSchema.allowReturnTraffic(),
			"source_zone_id":          resourceSchema.sourceZoneID(),
			"source_filter":           resourceSchema.sourceFilter(),
			"destination_zone_id":     resourceSchema.destinationZoneID(),
			"destination_filter":      resourceSchema.destinationFilter(),
			"ip_version":              resourceSchema.ipVersion(),
			"protocol_filter":         resourceSchema.protocolFilter(),
			"connection_state_filter": resourceSchema.connectionStateFilter(),
			"ipsec_filter":            resourceSchema.ipsecFilter(),
			"logging_enabled":         resourceSchema.loggingEnabled(),
			"schedule":                resourceSchema.schedule(),
			"index":                   resourceSchema.index(),
		},
	}
}

func (d *firewallPolicyDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *firewallPolicyDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state firewallPolicyModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateIDOrNameLookup(state.ID, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid firewall policy lookup arguments", err.Error())
		return
	}

	policies, err := d.clientProvider.client.ListFirewallPolicies(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list firewall policies", err.Error())
		return
	}

	var matches []client.FirewallPolicy
	for _, policy := range policies {
		switch {
		case !state.ID.IsNull() && state.ID.ValueString() != "":
			if policy.ID == state.ID.ValueString() {
				matches = append(matches, policy)
			}
		case !state.Name.IsNull() && state.Name.ValueString() != "":
			if policy.Name == state.Name.ValueString() {
				matches = append(matches, policy)
			}
		}
	}

	switch len(matches) {
	case 0:
		response.Diagnostics.AddError("Firewall policy not found", "No firewall policy matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple firewall policies matched", fmt.Sprintf("%d firewall policies matched the given selector", len(matches)))
		return
	}

	model, diagnostics := buildFirewallPolicyStateModel(ctx, state.SiteID, &matches[0])
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &model)...)
}

type resourceFirewallPolicySchemaHolder struct{}

func (resourceFirewallPolicySchemaHolder) enabled() schema.Attribute {
	return schema.BoolAttribute{Computed: true}
}
func (resourceFirewallPolicySchemaHolder) description() schema.Attribute {
	return schema.StringAttribute{Computed: true}
}
func (resourceFirewallPolicySchemaHolder) action() schema.Attribute {
	return schema.StringAttribute{Computed: true}
}
func (resourceFirewallPolicySchemaHolder) allowReturnTraffic() schema.Attribute {
	return schema.BoolAttribute{Computed: true}
}
func (resourceFirewallPolicySchemaHolder) sourceZoneID() schema.Attribute {
	return schema.StringAttribute{Computed: true}
}
func (resourceFirewallPolicySchemaHolder) sourceFilter() schema.Attribute {
	return firewallPolicyEndpointFilterComputedSchema("source")
}
func (resourceFirewallPolicySchemaHolder) destinationZoneID() schema.Attribute {
	return schema.StringAttribute{Computed: true}
}
func (resourceFirewallPolicySchemaHolder) destinationFilter() schema.Attribute {
	return firewallPolicyEndpointFilterComputedSchema("destination")
}
func (resourceFirewallPolicySchemaHolder) ipVersion() schema.Attribute {
	return schema.StringAttribute{Computed: true}
}
func (resourceFirewallPolicySchemaHolder) protocolFilter() schema.Attribute {
	return schema.SingleNestedAttribute{
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"type":            schema.StringAttribute{Computed: true},
			"named_protocol":  schema.StringAttribute{Computed: true},
			"match_opposite":  schema.BoolAttribute{Computed: true},
			"protocol_number": schema.Int64Attribute{Computed: true},
			"preset_name":     schema.StringAttribute{Computed: true},
		},
	}
}
func (resourceFirewallPolicySchemaHolder) connectionStateFilter() schema.Attribute {
	return schema.SetAttribute{Computed: true, ElementType: types.StringType}
}
func (resourceFirewallPolicySchemaHolder) ipsecFilter() schema.Attribute {
	return schema.StringAttribute{Computed: true}
}
func (resourceFirewallPolicySchemaHolder) loggingEnabled() schema.Attribute {
	return schema.BoolAttribute{Computed: true}
}
func (resourceFirewallPolicySchemaHolder) schedule() schema.Attribute {
	return schema.SingleNestedAttribute{
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"mode":           schema.StringAttribute{Computed: true},
			"start_time":     schema.StringAttribute{Computed: true},
			"stop_time":      schema.StringAttribute{Computed: true},
			"repeat_on_days": schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"date":           schema.StringAttribute{Computed: true},
			"start_date":     schema.StringAttribute{Computed: true},
			"stop_date":      schema.StringAttribute{Computed: true},
		},
	}
}
func (resourceFirewallPolicySchemaHolder) index() schema.Attribute {
	return schema.Int64Attribute{Computed: true}
}
