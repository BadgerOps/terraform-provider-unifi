package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type DHCPReservation struct {
	ClientID                  string  `json:"_id,omitempty"`
	MACAddress                string  `json:"mac"`
	FixedIP                   *string `json:"fixed_ip,omitempty"`
	Enabled                   bool    `json:"use_fixedip"`
	Hostname                  *string `json:"hostname,omitempty"`
	Name                      *string `json:"name,omitempty"`
	LastIP                    *string `json:"last_ip,omitempty"`
	LastConnectionNetworkName *string `json:"last_connection_network_name,omitempty"`
}

type legacyResponse[T any] struct {
	Data T `json:"data"`
}

type dhcpReservationUpdateRequest struct {
	ClientID string  `json:"_id"`
	FixedIP  *string `json:"fixed_ip,omitempty"`
	Enabled  bool    `json:"use_fixedip"`
}

func (c *Client) ListDHCPReservations(ctx context.Context, siteID string) ([]DHCPReservation, error) {
	siteReference, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("list dhcp reservations site id: %w", err)
	}

	var response legacyResponse[[]DHCPReservation]
	if err := c.doLegacyRequest(ctx, http.MethodGet, []string{"s", siteReference, "rest", "user"}, nil, &response); err != nil {
		return nil, fmt.Errorf("list dhcp reservations: %w", err)
	}

	return response.Data, nil
}

func (c *Client) GetDHCPReservation(ctx context.Context, siteID, macAddress string) (*DHCPReservation, error) {
	reservations, err := c.ListDHCPReservations(ctx, siteID)
	if err != nil {
		return nil, err
	}

	for _, reservation := range reservations {
		if strings.EqualFold(reservation.MACAddress, macAddress) {
			return &reservation, nil
		}
	}

	return nil, &MissingClientError{
		SiteID:     siteID,
		MACAddress: macAddress,
	}
}

func (c *Client) UpsertDHCPReservation(ctx context.Context, siteID, macAddress, fixedIP string, enabled bool) (*DHCPReservation, error) {
	existing, err := c.GetDHCPReservation(ctx, siteID, macAddress)
	if err != nil {
		return nil, err
	}

	siteReference, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("update dhcp reservation site id: %w", err)
	}

	fixedIPValue := strings.TrimSpace(fixedIP)
	request := dhcpReservationUpdateRequest{
		ClientID: existing.ClientID,
		FixedIP:  &fixedIPValue,
		Enabled:  enabled,
	}

	if err := c.doLegacyRequest(
		ctx,
		http.MethodPut,
		[]string{"s", siteReference, "rest", "user", existing.ClientID},
		request,
		nil,
	); err != nil {
		return nil, fmt.Errorf("update dhcp reservation: %w", err)
	}

	return c.GetDHCPReservation(ctx, siteID, macAddress)
}

func (c *Client) DeleteDHCPReservation(ctx context.Context, siteID, macAddress string) error {
	existing, err := c.GetDHCPReservation(ctx, siteID, macAddress)
	if err != nil {
		if IsMissingClient(err) {
			return nil
		}
		return err
	}

	siteReference, err := c.resolveLegacySiteReference(ctx, siteID)
	if err != nil {
		return fmt.Errorf("delete dhcp reservation site id: %w", err)
	}

	request := dhcpReservationUpdateRequest{
		ClientID: existing.ClientID,
		FixedIP:  existing.FixedIP,
		Enabled:  false,
	}

	if err := c.doLegacyRequest(
		ctx,
		http.MethodPut,
		[]string{"s", siteReference, "rest", "user", existing.ClientID},
		request,
		nil,
	); err != nil {
		return fmt.Errorf("delete dhcp reservation: %w", err)
	}

	return nil
}

func (c *Client) resolveLegacySiteReference(ctx context.Context, siteID string) (string, error) {
	sites, err := c.ListSites(ctx)
	if err != nil {
		return "", err
	}

	for _, site := range sites {
		if site.ID == siteID {
			return site.InternalReference, nil
		}
	}

	return "", fmt.Errorf("site %s not found", siteID)
}

func (c *Client) doLegacyRequest(ctx context.Context, method string, pathElements []string, payload any, target any) error {
	var bodyReader io.Reader
	if payload != nil {
		var err error
		bodyReader, err = jsonBodyReader(payload)
		if err != nil {
			return err
		}
	}

	endpoint := *c.legacyBaseURL
	endpoint.Path = joinURLPath(c.legacyBaseURL.Path, pathElements...)

	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("build legacy request: %w", err)
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-API-KEY", c.apiKey)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.userAgent != "" {
		request.Header.Set("User-Agent", c.userAgent)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read legacy response body: %w", err)
	}

	if err := requireStatus(response.StatusCode, body, http.StatusOK); err != nil {
		return err
	}

	if target == nil || len(body) == 0 {
		return nil
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode legacy response body: %w", err)
	}

	return nil
}
