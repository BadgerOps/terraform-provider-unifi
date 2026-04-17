package translate

import (
	"encoding/json"
	"testing"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
	"github.com/badgerops/terraform-provider-unifi/internal/openapi/generated"
)

func TestSiteOverviewToClient(t *testing.T) {
	var site generated.SiteOverview
	if err := json.Unmarshal([]byte(`{"id":"11111111-1111-1111-1111-111111111111","internalReference":"default","name":"Default"}`), &site); err != nil {
		t.Fatalf("unmarshal generated site: %v", err)
	}

	got := SiteOverviewToClient(site)

	if got.ID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected site id: %s", got.ID)
	}
	if got.InternalReference != "default" {
		t.Fatalf("unexpected internal reference: %s", got.InternalReference)
	}
	if got.Name != "Default" {
		t.Fatalf("unexpected site name: %s", got.Name)
	}
}

func TestNetworkRoundTripSpikeMapping(t *testing.T) {
	var generatedNetwork generated.NetworkDetails
	if err := json.Unmarshal([]byte(`{
		"id":"22222222-2222-2222-2222-222222222222",
		"management":"GATEWAY",
		"name":"trusted",
		"enabled":true,
		"default":false,
		"vlanId":20,
		"metadata":{"origin":"USER"},
		"dhcpGuarding":{"trustedDhcpServerIpAddresses":["10.20.0.10"]}
	}`), &generatedNetwork); err != nil {
		t.Fatalf("unmarshal generated network: %v", err)
	}

	clientNetwork := NetworkDetailsToClient(generatedNetwork)
	if clientNetwork.ID != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("unexpected network id: %s", clientNetwork.ID)
	}
	if clientNetwork.VLANID != 20 {
		t.Fatalf("unexpected vlan id: %d", clientNetwork.VLANID)
	}
	if clientNetwork.DHCPGuarding == nil || len(clientNetwork.DHCPGuarding.TrustedDHCPServerIPAddresses) != 1 {
		t.Fatalf("expected DHCP guarding to be mapped")
	}

	backToGenerated := NetworkToGeneratedCreateUpdate(client.Network{
		Management: "GATEWAY",
		Name:       "trusted",
		Enabled:    true,
		VLANID:     20,
		DHCPGuarding: &client.DHCPGuarding{
			TrustedDHCPServerIPAddresses: []string{"10.20.0.10"},
		},
	})

	if backToGenerated.VlanId != 20 {
		t.Fatalf("unexpected generated vlan id: %d", backToGenerated.VlanId)
	}
	if backToGenerated.DhcpGuarding == nil || len(backToGenerated.DhcpGuarding.TrustedDhcpServerIpAddresses) != 1 {
		t.Fatalf("expected generated DHCP guarding to be mapped")
	}
}
