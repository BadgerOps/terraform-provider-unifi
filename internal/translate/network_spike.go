package translate

import (
	"github.com/badgerops/terraform-provider-unifi/internal/client"
	"github.com/badgerops/terraform-provider-unifi/internal/openapi/generated"
)

// SiteOverviewToClient maps the generated site overview model into the
// provider's handwritten client shape.
func SiteOverviewToClient(site generated.SiteOverview) client.Site {
	return client.Site{
		ID:                site.Id.String(),
		InternalReference: site.InternalReference,
		Name:              site.Name,
	}
}

// NetworkDetailsToClient maps the generated network details model into the
// provider's handwritten client shape.
func NetworkDetailsToClient(network generated.NetworkDetails) client.Network {
	result := client.Network{
		ID:         network.Id.String(),
		Management: network.Management,
		Name:       network.Name,
		Enabled:    network.Enabled,
		VLANID:     int64(network.VlanId),
		Default:    network.Default,
	}

	if network.DhcpGuarding != nil {
		result.DHCPGuarding = &client.DHCPGuarding{
			TrustedDHCPServerIPAddresses: network.DhcpGuarding.TrustedDhcpServerIpAddresses,
		}
	}

	return result
}

// NetworkToGeneratedCreateUpdate maps the handwritten network model into the
// generated create/update DTO.
func NetworkToGeneratedCreateUpdate(network client.Network) generated.CreateOrUpdateNetwork {
	result := generated.CreateOrUpdateNetwork{
		Enabled:    network.Enabled,
		Management: network.Management,
		Name:       network.Name,
		VlanId:     int32(network.VLANID),
	}

	if network.DHCPGuarding != nil {
		result.DhcpGuarding = &generated.NetworkDHCPGuarding{
			TrustedDhcpServerIpAddresses: network.DHCPGuarding.TrustedDHCPServerIPAddresses,
		}
	}

	return result
}
