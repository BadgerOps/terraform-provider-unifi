package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// PortOverride is one entry of a switch's legacy `port_overrides` list: the
// settings of one port. Only the fields below are modelled; every other field
// of the entry (PoE, STP, port security, rate limits, ...) is kept from the
// last read and written back unchanged.
//
// Network references are legacy `networkconf` IDs, not the integration API's
// network UUIDs; see LegacyNetworkIDs.
type PortOverride struct {
	PortIdx                int64    `json:"port_idx"`
	Name                   *string  `json:"name,omitempty"`
	NativeNetworkconfID    string   `json:"native_networkconf_id,omitempty"`
	TaggedVLANMgmt         string   `json:"tagged_vlan_mgmt,omitempty"`
	ExcludedNetworkconfIDs []string `json:"excluded_networkconf_ids,omitempty"`
	rawFields              map[string]json.RawMessage
}

func (o *PortOverride) UnmarshalJSON(data []byte) error {
	type wirePortOverride PortOverride
	var decoded wirePortOverride
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &decoded.rawFields); err != nil {
		return err
	}
	*o = PortOverride(decoded)
	return nil
}

// MarshalJSON writes the unmodelled fields from the last read, overlaid with
// the modelled ones. A nil name or exclusion list preserves the controller's
// value; a non-nil empty value explicitly clears it. Other empty fields are
// left as the controller had them.
func (o PortOverride) MarshalJSON() ([]byte, error) {
	out := make(map[string]any, len(o.rawFields)+5)
	for key, value := range o.rawFields {
		out[key] = value
	}
	out["port_idx"] = o.PortIdx
	if o.Name != nil {
		out["name"] = *o.Name
	}
	if o.NativeNetworkconfID != "" {
		out["native_networkconf_id"] = o.NativeNetworkconfID
	}
	if o.TaggedVLANMgmt != "" {
		out["tagged_vlan_mgmt"] = o.TaggedVLANMgmt
	}
	if o.ExcludedNetworkconfIDs != nil {
		out["excluded_networkconf_ids"] = o.ExcludedNetworkconfIDs
	}
	return json.Marshal(out)
}

type legacyDevice struct {
	ID            string         `json:"_id"`
	MAC           string         `json:"mac"`
	PortOverrides []PortOverride `json:"port_overrides"`
}

type legacyNetworkconf struct {
	ID         string `json:"_id"`
	ExternalID string `json:"external_id"`
}

// devicePortLocks serialises the read-modify-write of one device's
// port_overrides, so that two port resources of the same switch applied in
// parallel can't overwrite each other's change.
var devicePortLocks sync.Map

func lockDevicePorts(mac string) func() {
	value, _ := devicePortLocks.LoadOrStore(strings.ToLower(mac), &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}

func (c *Client) getLegacyDevice(ctx context.Context, site, mac string) (*legacyDevice, error) {
	var response legacyResponse[[]legacyDevice]
	if err := c.doLegacyRequest(ctx, http.MethodGet, []string{"s", site, "stat", "device"}, nil, &response); err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	for _, device := range response.Data {
		if strings.EqualFold(device.MAC, mac) {
			return &device, nil
		}
	}
	return nil, &Error{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("device %s not found in site %s", mac, site)}
}

// GetDevicePortOverride returns the override of one port; a not-found error if
// the device or the port's entry doesn't exist.
func (c *Client) GetDevicePortOverride(ctx context.Context, siteID, mac string, portIdx int64) (*PortOverride, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("read port override site: %w", err)
	}
	device, err := c.getLegacyDevice(ctx, site, mac)
	if err != nil {
		return nil, err
	}
	for _, override := range device.PortOverrides {
		if override.PortIdx == portIdx {
			return &override, nil
		}
	}
	return nil, &Error{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("device %s has no override for port %d", mac, portIdx)}
}

// SetDevicePortOverride creates or changes the override of one port. The
// device's list is read first; only the entry for override.PortIdx changes,
// keeping its unmodelled fields, and every other entry is written back as read.
func (c *Client) SetDevicePortOverride(ctx context.Context, siteID, mac string, override PortOverride) (*PortOverride, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("update port override site: %w", err)
	}
	unlock := lockDevicePorts(mac)
	defer unlock()
	device, err := c.getLegacyDevice(ctx, site, mac)
	if err != nil {
		return nil, err
	}
	overrides := make([]PortOverride, 0, len(device.PortOverrides)+1)
	replaced := false
	for _, existing := range device.PortOverrides {
		if existing.PortIdx == override.PortIdx {
			override.rawFields = existing.rawFields
			existing = override
			replaced = true
		}
		overrides = append(overrides, existing)
	}
	if !replaced {
		overrides = append(overrides, override)
	}
	payload := map[string]any{"port_overrides": overrides}
	if err := c.doLegacyRequest(ctx, http.MethodPut, []string{"s", site, "rest", "device", device.ID}, payload, nil); err != nil {
		return nil, fmt.Errorf("update port overrides of %s: %w", mac, err)
	}
	return c.GetDevicePortOverride(ctx, siteID, mac, override.PortIdx)
}

// LegacyNetworkIDs maps the integration API's network UUIDs to the legacy
// networkconf IDs that port overrides use. The legacy object carries the UUID
// as `external_id`.
func (c *Client) LegacyNetworkIDs(ctx context.Context, siteID string) (map[string]string, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("list networks site: %w", err)
	}
	var response legacyResponse[[]legacyNetworkconf]
	if err := c.doLegacyRequest(ctx, http.MethodGet, []string{"s", site, "rest", "networkconf"}, nil, &response); err != nil {
		return nil, fmt.Errorf("list legacy networks: %w", err)
	}
	ids := make(map[string]string, len(response.Data))
	for _, network := range response.Data {
		if network.ExternalID != "" {
			ids[network.ExternalID] = network.ID
		}
	}
	return ids, nil
}

type legacyDeviceManagement struct {
	ID                string `json:"_id"`
	MAC               string `json:"mac"`
	MgmtNetworkconfID string `json:"mgmt_network_id"`
}

// GetDeviceManagementNetwork returns the legacy networkconf ID of the network
// the device itself lives in (`mgmt_network_id`); empty when the device has
// none set and uses its uplink's untagged network.
func (c *Client) GetDeviceManagementNetwork(ctx context.Context, siteID, mac string) (string, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return "", fmt.Errorf("read management network site: %w", err)
	}
	device, err := c.getLegacyDeviceManagement(ctx, site, mac)
	if err != nil {
		return "", err
	}
	return device.MgmtNetworkconfID, nil
}

// SetDeviceManagementNetwork moves the device into another network. Only
// `mgmt_network_id` is sent; the controller re-provisions the device, which
// then takes an address there.
func (c *Client) SetDeviceManagementNetwork(ctx context.Context, siteID, mac, networkconfID string) (string, error) {
	site, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return "", fmt.Errorf("update management network site: %w", err)
	}
	unlock := lockDevicePorts(mac)
	defer unlock()
	device, err := c.getLegacyDeviceManagement(ctx, site, mac)
	if err != nil {
		return "", err
	}
	payload := map[string]any{"mgmt_network_id": networkconfID}
	if err := c.doLegacyRequest(ctx, http.MethodPut, []string{"s", site, "rest", "device", device.ID}, payload, nil); err != nil {
		return "", fmt.Errorf("update management network of %s: %w", mac, err)
	}
	return c.GetDeviceManagementNetwork(ctx, siteID, mac)
}

func (c *Client) getLegacyDeviceManagement(ctx context.Context, site, mac string) (*legacyDeviceManagement, error) {
	var response legacyResponse[[]legacyDeviceManagement]
	if err := c.doLegacyRequest(ctx, http.MethodGet, []string{"s", site, "stat", "device"}, nil, &response); err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	for _, device := range response.Data {
		if strings.EqualFold(device.MAC, mac) {
			return &device, nil
		}
	}
	return nil, &Error{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("device %s not found in site %s", mac, site)}
}
