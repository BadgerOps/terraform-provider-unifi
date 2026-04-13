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
	_ datasource.DataSource              = (*aclRuleDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*aclRuleDataSource)(nil)
)

type aclRuleDataSource struct {
	clientProvider *providerData
}

type aclRuleDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	SiteID               types.String `tfsdk:"site_id"`
	Type                 types.String `tfsdk:"type"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Action               types.String `tfsdk:"action"`
	EnforcingDeviceIDs   types.Set    `tfsdk:"enforcing_device_ids"`
	ProtocolFilter       types.Set    `tfsdk:"protocol_filter"`
	NetworkIDFilter      types.String `tfsdk:"network_id_filter"`
	SourceIPFilter       types.Object `tfsdk:"source_ip_filter"`
	DestinationIPFilter  types.Object `tfsdk:"destination_ip_filter"`
	SourceMACFilter      types.Object `tfsdk:"source_mac_filter"`
	DestinationMACFilter types.Object `tfsdk:"destination_mac_filter"`
	Index                types.Int64  `tfsdk:"index"`
}

func NewACLRuleDataSource() datasource.DataSource {
	return &aclRuleDataSource{}
}

func (d *aclRuleDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_acl_rule"
}

func (d *aclRuleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi ACL rule by `id` or `name` within a site.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"site_id": schema.StringAttribute{Required: true},
			"type":    schema.StringAttribute{Computed: true},
			"enabled": schema.BoolAttribute{Computed: true},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"description": schema.StringAttribute{Computed: true},
			"action":      schema.StringAttribute{Computed: true},
			"enforcing_device_ids": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"protocol_filter": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"network_id_filter": schema.StringAttribute{Computed: true},
			"source_ip_filter": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{Computed: true},
					"ip_addresses_or_subnets": schema.SetAttribute{
						Computed:    true,
						ElementType: types.StringType,
					},
					"network_ids": schema.SetAttribute{
						Computed:    true,
						ElementType: types.StringType,
					},
					"ports": schema.SetAttribute{
						Computed:    true,
						ElementType: types.Int64Type,
					},
				},
			},
			"destination_ip_filter": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{Computed: true},
					"ip_addresses_or_subnets": schema.SetAttribute{
						Computed:    true,
						ElementType: types.StringType,
					},
					"network_ids": schema.SetAttribute{
						Computed:    true,
						ElementType: types.StringType,
					},
					"ports": schema.SetAttribute{
						Computed:    true,
						ElementType: types.Int64Type,
					},
				},
			},
			"source_mac_filter": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{Computed: true},
					"mac_addresses": schema.SetAttribute{
						Computed:    true,
						ElementType: types.StringType,
					},
					"prefix_length": schema.Int64Attribute{Computed: true},
				},
			},
			"destination_mac_filter": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{Computed: true},
					"mac_addresses": schema.SetAttribute{
						Computed:    true,
						ElementType: types.StringType,
					},
					"prefix_length": schema.Int64Attribute{Computed: true},
				},
			},
			"index": schema.Int64Attribute{Computed: true},
		},
	}
}

func (d *aclRuleDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *aclRuleDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state aclRuleDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateIDOrNameLookup(state.ID, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid ACL rule lookup arguments", err.Error())
		return
	}

	rules, err := d.clientProvider.client.ListACLRules(ctx, state.SiteID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to list ACL rules", err.Error())
		return
	}

	var matches []client.ACLRule
	for _, rule := range rules {
		switch {
		case !state.ID.IsNull() && state.ID.ValueString() != "":
			if rule.ID == state.ID.ValueString() {
				matches = append(matches, rule)
			}
		case !state.Name.IsNull() && state.Name.ValueString() != "":
			if rule.Name == state.Name.ValueString() {
				matches = append(matches, rule)
			}
		}
	}

	switch len(matches) {
	case 0:
		response.Diagnostics.AddError("ACL rule not found", "No ACL rule matched the given selector.")
		return
	case 1:
	default:
		response.Diagnostics.AddError("Multiple ACL rules matched", fmt.Sprintf("%d ACL rules matched the given selector", len(matches)))
		return
	}

	model, diagnostics := buildACLRuleStateModel(ctx, state.SiteID, &matches[0])
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &model)...)
}
