package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/badgerops/terraform-provider-unifi/internal/openapi/generated"
)

type Device struct {
	ID                string   `json:"id,omitempty"`
	Name              string   `json:"name"`
	Model             string   `json:"model"`
	MacAddress        string   `json:"macAddress"`
	IPAddress         string   `json:"ipAddress"`
	State             string   `json:"state"`
	Supported         bool     `json:"supported"`
	FirmwareUpdatable bool     `json:"firmwareUpdatable"`
	FirmwareVersion   *string  `json:"firmwareVersion,omitempty"`
	Features          []string `json:"features,omitempty"`
	Interfaces        []string `json:"interfaces,omitempty"`
}

func (c *Client) ListDevices(ctx context.Context, siteID string) ([]Device, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list devices site id: %w", err)
	}

	var devices []Device
	offset := 0

	for {
		response, err := c.apiClient.GetAdoptedDeviceOverviewPageWithResponse(ctx, siteUUID, &generated.GetAdoptedDeviceOverviewPageParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list devices: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]Device](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate device page: %w", err)
		}

		devices = append(devices, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return devices, nil
}
