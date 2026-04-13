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
	_ datasource.DataSource              = (*wifiBroadcastDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*wifiBroadcastDataSource)(nil)
)

type wifiBroadcastDataSource struct {
	clientProvider *providerData
}

func NewWifiBroadcastDataSource() datasource.DataSource {
	return &wifiBroadcastDataSource{}
}

func (d *wifiBroadcastDataSource) Metadata(_ context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_wifi_broadcast"
}

func (d *wifiBroadcastDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Look up a UniFi WiFi broadcast by `id` or `name` within a site.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"site_id": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"enabled": schema.BoolAttribute{
				Computed: true,
			},
			"network": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Computed: true,
					},
					"network_id": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"security_configuration": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Computed: true,
					},
					"passphrase": schema.StringAttribute{
						Computed:  true,
						Sensitive: true,
					},
					"pmf_mode": schema.StringAttribute{
						Computed: true,
					},
					"fast_roaming_enabled": schema.BoolAttribute{
						Computed: true,
					},
					"group_rekey_interval_seconds": schema.Int64Attribute{
						Computed: true,
					},
					"wpa3_fast_roaming_enabled": schema.BoolAttribute{
						Computed: true,
					},
					"sae_configuration": schema.SingleNestedAttribute{
						Computed: true,
						Attributes: map[string]schema.Attribute{
							"anticlogging_threshold_seconds": schema.Int64Attribute{
								Computed: true,
							},
							"sync_time_seconds": schema.Int64Attribute{
								Computed: true,
							},
						},
					},
				},
			},
			"client_isolation_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"hide_name": schema.BoolAttribute{
				Computed: true,
			},
			"uapsd_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"multicast_to_unicast_conversion_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"broadcasting_frequencies_ghz": schema.SetAttribute{
				Computed:    true,
				ElementType: types.Float64Type,
			},
			"broadcasting_device_filter": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Computed: true,
					},
					"device_tag_ids": schema.SetAttribute{
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
			"advertise_device_name": schema.BoolAttribute{
				Computed: true,
			},
			"arp_proxy_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"band_steering_enabled": schema.BoolAttribute{
				Computed: true,
			},
			"bss_transition_enabled": schema.BoolAttribute{
				Computed: true,
			},
		},
	}
}

func (d *wifiBroadcastDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	d.clientProvider = request.ProviderData.(*providerData)
}

func (d *wifiBroadcastDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var state wifiBroadcastResourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := validateIDOrNameLookup(state.ID, state.Name); err != nil {
		response.Diagnostics.AddError("Invalid WiFi broadcast lookup arguments", err.Error())
		return
	}

	var broadcast *client.WifiBroadcast
	if !state.ID.IsNull() && state.ID.ValueString() != "" {
		var err error
		broadcast, err = d.clientProvider.client.GetWifiBroadcast(ctx, state.SiteID.ValueString(), state.ID.ValueString())
		if err != nil {
			if client.IsNotFound(err) {
				response.Diagnostics.AddError("WiFi broadcast not found", "No WiFi broadcast matched the given selector.")
				return
			}
			response.Diagnostics.AddError("Unable to read WiFi broadcast", err.Error())
			return
		}
	} else {
		broadcasts, err := d.clientProvider.client.ListWifiBroadcasts(ctx, state.SiteID.ValueString())
		if err != nil {
			response.Diagnostics.AddError("Unable to list WiFi broadcasts", err.Error())
			return
		}

		var matches []client.WifiBroadcast
		for _, candidate := range broadcasts {
			if candidate.Name == state.Name.ValueString() {
				matches = append(matches, candidate)
			}
		}

		switch len(matches) {
		case 0:
			response.Diagnostics.AddError("WiFi broadcast not found", "No WiFi broadcast matched the given selector.")
			return
		case 1:
		default:
			response.Diagnostics.AddError("Multiple WiFi broadcasts matched", fmt.Sprintf("%d WiFi broadcasts matched the given selector", len(matches)))
			return
		}

		broadcast, err = d.clientProvider.client.GetWifiBroadcast(ctx, state.SiteID.ValueString(), matches[0].ID)
		if err != nil {
			response.Diagnostics.AddError("Unable to read WiFi broadcast", err.Error())
			return
		}
	}

	model, diagnostics := buildWifiBroadcastStateModel(ctx, state.SiteID, broadcast)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &model)...)
}
