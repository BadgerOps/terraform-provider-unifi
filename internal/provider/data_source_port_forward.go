package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

var (
	_ datasource.DataSource              = (*portForwardDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*portForwardDataSource)(nil)
)

type portForwardDataSource struct{ providerData *providerData }

func NewPortForwardDataSource() datasource.DataSource { return &portForwardDataSource{} }

func (d *portForwardDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_forward"
}

func (d *portForwardDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi port forwarding rule by exactly one of `id` (legacy `_id`) or `name` within a site, using the legacy local Network API. Names must identify a unique rule.",
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Legacy rule ID."},
			"site_id":          schema.StringAttribute{Required: true, MarkdownDescription: "Integration site UUID."},
			"name":             schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Rule name."},
			"enabled":          schema.BoolAttribute{Computed: true},
			"wan_interface":    schema.StringAttribute{Computed: true},
			"protocol":         schema.StringAttribute{Computed: true},
			"source":           schema.StringAttribute{Computed: true},
			"destination_port": schema.StringAttribute{Computed: true},
			"forward_ip":       schema.StringAttribute{Computed: true},
			"forward_port":     schema.StringAttribute{Computed: true},
			"logging_enabled":  schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *portForwardDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	d.providerData, ok = req.ProviderData.(*providerData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *providerData, got %T", req.ProviderData))
	}
}

func (d *portForwardDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config portForwardModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, name := config.ID.ValueString(), config.Name.ValueString()
	if (id == "") == (name == "") {
		resp.Diagnostics.AddError("Invalid port forward lookup arguments", "Exactly one of `id` or `name` must be set.")
		return
	}
	rules, err := d.providerData.client.ListPortForwards(ctx, config.SiteID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to list port forwards", err.Error())
		return
	}
	var matches []client.PortForward
	for _, rule := range rules {
		if (id != "" && rule.ID == id) || (id == "" && rule.Name == name) {
			matches = append(matches, rule)
		}
	}
	if len(matches) == 0 {
		resp.Diagnostics.AddError("Port forward not found", "No port forward matched the given selector.")
		return
	}
	if len(matches) > 1 {
		resp.Diagnostics.AddError("Multiple port forwards matched", fmt.Sprintf("%d port forwards matched the given selector; use an id instead.", len(matches)))
		return
	}
	writePortForwardState(ctx, &resp.State, &resp.Diagnostics, config.SiteID, &matches[0])
}
