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
