package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	tfstate "github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

type liveAcceptanceConfig struct {
	APIURL        string
	APIKey        string
	AllowInsecure bool
	SiteID        string
	SiteName      string
	NamePrefix    string
}

func TestAccLiveSiteDataSource(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)

	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttrSet("data.unifi_site.target", "id"),
		resource.TestCheckResourceAttrSet("data.unifi_site.target", "name"),
		resource.TestCheckResourceAttrSet("data.unifi_site.target", "internal_reference"),
	}
	if config.SiteID != "" {
		checks = append(checks, resource.TestCheckResourceAttr("data.unifi_site.target", "id", config.SiteID))
	}
	if config.SiteName != "" {
		checks = append(checks, resource.TestCheckResourceAttr("data.unifi_site.target", "name", config.SiteName))
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config),
				Check:  resource.ComposeAggregateTestCheckFunc(checks...),
			},
		},
	})
}

func TestAccLiveResourceNetwork(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	resourceName := "unifi_network.test"
	networkName := liveAcceptanceName(config, "network")
	updatedName := networkName + "-updated"
	vlanID := liveAcceptanceVLAN()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyNetwork(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveNetworkResourceConfig(config, networkName, vlanID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "management", "UNMANAGED"),
					resource.TestCheckResourceAttr(resourceName, "name", networkName),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "vlan_id", strconv.FormatInt(vlanID, 10)),
				),
			},
			{
				Config: liveNetworkResourceConfig(config, updatedName, vlanID+1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckResourceAttr(resourceName, "vlan_id", strconv.FormatInt(vlanID+1, 10)),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateIdFunc:       liveImportCompositeID(resourceName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"default"},
			},
		},
	})
}

func TestAccLiveDataSourceNetwork(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	resourceName := "unifi_network.test"
	dataSourceName := "data.unifi_network.lookup"
	networkName := liveAcceptanceName(config, "netds")
	vlanID := liveAcceptanceVLAN()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyNetwork(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_network" "test" {
  site_id    = data.unifi_site.target.id
  management = "UNMANAGED"
  name       = %q
  enabled    = true
  vlan_id    = %d
}

data "unifi_network" "lookup" {
  site_id = data.unifi_site.target.id
  id      = unifi_network.test.id
}
`, networkName, vlanID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "id", resourceName, "id"),
					resource.TestCheckResourceAttr(dataSourceName, "name", networkName),
					resource.TestCheckResourceAttr(dataSourceName, "management", "UNMANAGED"),
					resource.TestCheckResourceAttr(dataSourceName, "vlan_id", strconv.FormatInt(vlanID, 10)),
				),
			},
		},
	})
}

func TestAccLiveResourceTrafficMatchingList(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	requireZoneFirewallConfigured(t)
	resourceName := "unifi_traffic_matching_list.test"
	listName := liveAcceptanceName(config, "ports")
	updatedName := liveAcceptanceName(config, "portsu")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyTrafficMatchingList(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_traffic_matching_list" "test" {
  site_id = data.unifi_site.target.id
  type    = "PORTS"
  name    = %q
  ports   = ["80", "443", "8443-8444"]
}
`, listName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "type", "PORTS"),
					resource.TestCheckResourceAttr(resourceName, "name", listName),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "80"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "443"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "8443-8444"),
				),
			},
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_traffic_matching_list" "test" {
  site_id = data.unifi_site.target.id
  type    = "PORTS"
  name    = %q
  ports   = ["53", "443", "10000-10010"]
}
`, updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "53"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "443"),
					resource.TestCheckTypeSetElemAttr(resourceName, "ports.*", "10000-10010"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: liveImportCompositeID(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLiveResourceFirewallZone(t *testing.T) {
	t.Parallel()

	config := requireLiveAcceptanceConfig(t)
	requireZoneFirewallConfigured(t)
	resourceName := "unifi_firewall_zone.test"
	networkName := liveAcceptanceName(config, "zone-net")
	zoneName := liveAcceptanceName(config, "zone")
	updatedZoneName := liveAcceptanceName(config, "zoneu")
	vlanID := liveAcceptanceVLAN()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             liveCheckDestroyFirewallZone(config, resourceName),
		Steps: []resource.TestStep{
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_network" "test" {
  site_id    = data.unifi_site.target.id
  management = "UNMANAGED"
  name       = %q
  enabled    = true
  vlan_id    = %d
}

resource "unifi_firewall_zone" "test" {
  site_id     = data.unifi_site.target.id
  name        = %q
  network_ids = [unifi_network.test.id]
}
`, networkName, vlanID, zoneName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", zoneName),
					resource.TestCheckResourceAttr(resourceName, "network_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "network_ids.*", "unifi_network.test", "id"),
				),
			},
			{
				Config: liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_network" "test" {
  site_id    = data.unifi_site.target.id
  management = "UNMANAGED"
  name       = %q
  enabled    = true
  vlan_id    = %d
}

resource "unifi_firewall_zone" "test" {
  site_id     = data.unifi_site.target.id
  name        = %q
  network_ids = [unifi_network.test.id]
}
`, networkName, vlanID, updatedZoneName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedZoneName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: liveImportCompositeID(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func requireLiveAcceptanceConfig(t *testing.T) liveAcceptanceConfig {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run live acceptance tests")
	}

	config := liveAcceptanceConfig{
		APIURL:        strings.TrimSpace(os.Getenv("UNIFI_API_URL")),
		APIKey:        strings.TrimSpace(os.Getenv("UNIFI_API_KEY")),
		SiteID:        strings.TrimSpace(os.Getenv("UNIFI_TEST_SITE_ID")),
		SiteName:      strings.TrimSpace(os.Getenv("UNIFI_TEST_SITE_NAME")),
		NamePrefix:    strings.TrimSpace(os.Getenv("UNIFI_TEST_NAME_PREFIX")),
		AllowInsecure: parseEnvBool("UNIFI_ALLOW_INSECURE"),
	}

	if config.APIURL == "" {
		t.Fatal("UNIFI_API_URL must be set for live acceptance tests")
	}
	if config.APIKey == "" {
		t.Fatal("UNIFI_API_KEY must be set for live acceptance tests")
	}

	selectorCount := 0
	if config.SiteID != "" {
		selectorCount++
	}
	if config.SiteName != "" {
		selectorCount++
	}
	if selectorCount != 1 {
		t.Fatal("set exactly one of UNIFI_TEST_SITE_ID or UNIFI_TEST_SITE_NAME for live acceptance tests")
	}

	if config.NamePrefix == "" {
		config.NamePrefix = "acctest-"
	}

	return config
}

func parseEnvBool(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func requireZoneFirewallConfigured(t *testing.T) {
	t.Helper()

	if !parseEnvBool("UNIFI_TEST_ENABLE_ZONE_FIREWALL") {
		t.Skip("set UNIFI_TEST_ENABLE_ZONE_FIREWALL=1 to run zone firewall live acceptance tests")
	}
}

func liveProviderConfig(config liveAcceptanceConfig) string {
	return fmt.Sprintf(`
provider "unifi" {
  api_url        = %q
  api_key        = %q
  allow_insecure = %t
}
`, config.APIURL, config.APIKey, config.AllowInsecure)
}

func liveSiteLookupDataSource(config liveAcceptanceConfig) string {
	switch {
	case config.SiteID != "":
		return fmt.Sprintf(`
data "unifi_site" "target" {
  id = %q
}
`, config.SiteID)
	default:
		return fmt.Sprintf(`
data "unifi_site" "target" {
  name = %q
}
`, config.SiteName)
	}
}

func liveNetworkResourceConfig(config liveAcceptanceConfig, name string, vlanID int64) string {
	return liveProviderConfig(config) + liveSiteLookupDataSource(config) + fmt.Sprintf(`
resource "unifi_network" "test" {
  site_id    = data.unifi_site.target.id
  management = "UNMANAGED"
  name       = %q
  enabled    = true
  vlan_id    = %d
}
`, name, vlanID)
}

func liveAcceptanceName(config liveAcceptanceConfig, suffix string) string {
	token := strconv.FormatInt(time.Now().UnixNano()%1_000_000_000, 36)
	name := fmt.Sprintf("%s%s-%s", config.NamePrefix, suffix, token)
	if len(name) <= 32 {
		return name
	}

	excess := len(name) - 32
	if excess >= len(config.NamePrefix) {
		return name[len(name)-32:]
	}

	return fmt.Sprintf("%s%s-%s", config.NamePrefix[:len(config.NamePrefix)-excess], suffix, token)
}

func liveAcceptanceVLAN() int64 {
	return 2000 + time.Now().UnixNano()%1000
}

func liveImportCompositeID(resourceName string) resource.ImportStateIdFunc {
	return func(state *tfstate.State) (string, error) {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}

		siteID := state.RootModule().Resources["data.unifi_site.target"].Primary.ID
		if siteID == "" {
			return "", fmt.Errorf("site lookup data source missing from state")
		}

		return fmt.Sprintf("%s/%s", siteID, resourceState.Primary.ID), nil
	}
}

func liveCheckDestroyNetwork(config liveAcceptanceConfig, resourceName string) resource.TestCheckFunc {
	return func(state *tfstate.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}

		siteState, ok := state.RootModule().Resources["data.unifi_site.target"]
		if !ok || siteState.Primary.ID == "" {
			return fmt.Errorf("site lookup data source missing from state during destroy check")
		}

		apiClient, err := client.New(client.Config{
			BaseURL:       config.APIURL,
			APIKey:        config.APIKey,
			AllowInsecure: config.AllowInsecure,
			UserAgent:     "terraform-provider-unifi/testacc",
		})
		if err != nil {
			return fmt.Errorf("build client for destroy check: %w", err)
		}

		_, err = apiClient.GetNetwork(context.Background(), siteState.Primary.ID, resourceState.Primary.ID)
		if client.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("verify network destroy: %w", err)
		}

		return fmt.Errorf("network %s still exists", resourceState.Primary.ID)
	}
}

func liveCheckDestroyFirewallZone(config liveAcceptanceConfig, resourceName string) resource.TestCheckFunc {
	return func(state *tfstate.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}

		siteState, ok := state.RootModule().Resources["data.unifi_site.target"]
		if !ok || siteState.Primary.ID == "" {
			return fmt.Errorf("site lookup data source missing from state during destroy check")
		}

		apiClient, err := newLiveDestroyCheckClient(config)
		if err != nil {
			return err
		}

		_, err = apiClient.GetFirewallZone(context.Background(), siteState.Primary.ID, resourceState.Primary.ID)
		if client.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("verify firewall zone destroy: %w", err)
		}

		return fmt.Errorf("firewall zone %s still exists", resourceState.Primary.ID)
	}
}

func liveCheckDestroyTrafficMatchingList(config liveAcceptanceConfig, resourceName string) resource.TestCheckFunc {
	return func(state *tfstate.State) error {
		resourceState, ok := state.RootModule().Resources[resourceName]
		if !ok || resourceState.Primary.ID == "" {
			return nil
		}

		siteState, ok := state.RootModule().Resources["data.unifi_site.target"]
		if !ok || siteState.Primary.ID == "" {
			return fmt.Errorf("site lookup data source missing from state during destroy check")
		}

		apiClient, err := newLiveDestroyCheckClient(config)
		if err != nil {
			return err
		}

		_, err = apiClient.GetTrafficMatchingList(context.Background(), siteState.Primary.ID, resourceState.Primary.ID)
		if client.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("verify traffic matching list destroy: %w", err)
		}

		return fmt.Errorf("traffic matching list %s still exists", resourceState.Primary.ID)
	}
}

func newLiveDestroyCheckClient(config liveAcceptanceConfig) (*client.Client, error) {
	apiClient, err := client.New(client.Config{
		BaseURL:       config.APIURL,
		APIKey:        config.APIKey,
		AllowInsecure: config.AllowInsecure,
		UserAgent:     "terraform-provider-unifi/testacc",
	})
	if err != nil {
		return nil, fmt.Errorf("build client for destroy check: %w", err)
	}

	return apiClient, nil
}
