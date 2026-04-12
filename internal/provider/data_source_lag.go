package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ datasource.DataSource              = (*lagDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*lagDataSource)(nil)
)

type lagDataSource struct {
	clientProvider *providerData
}

type lagDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	SiteID          types.String `tfsdk:"site_id"`
	Type            types.String `tfsdk:"type"`
	SwitchStackID   types.String `tfsdk:"switch_stack_id"`
	McLagDomainID   types.String `tfsdk:"mc_lag_domain_id"`
	MemberDeviceIDs types.Set    `tfsdk:"member_device_ids"`
}

func NewLagDataSource() datasource.DataSource {
	return &lagDataSource{}
}

func (d *lagDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_lag"
}

func (d *lagDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi LAG by `id` within a site.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required: true,
			},
			"site_id": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Computed: true,
			},
			"switch_stack_id": schema.StringAttribute{
				Computed: true,
			},
			"mc_lag_domain_id": schema.StringAttribute{
				Computed: true,
			},
			"member_device_ids": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *lagDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *lagDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state lagDataSourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	lag, err := d.clientProvider.client.GetLag(ctx, state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to read UniFi LAG", err.Error())
		return
	}

	var diagnostics diag.Diagnostics
	state.Type = types.StringValue(lag.Type)
	state.SwitchStackID = nullableString(lag.SwitchStackID)
	state.McLagDomainID = nullableString(lag.McLagDomainID)
	state.MemberDeviceIDs, diagnostics = stringSetValue(ctx, flattenLagMemberDeviceIDs(lag.Members))
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func flattenLagMemberDeviceIDs(members []client.LagMember) []string {
	deviceIDs := make([]string, 0, len(members))
	for _, member := range members {
		deviceIDs = append(deviceIDs, member.DeviceID)
	}

	return deviceIDs
}
