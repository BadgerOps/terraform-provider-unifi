package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/badgerops/terraform-provider-unifi/internal/openapi/generated"
)

type WAN struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

type SwitchStackMember struct {
	DeviceID string `json:"deviceId"`
}

type LagMember struct {
	DeviceID string  `json:"deviceId"`
	PortIdxs []int64 `json:"portIdxs"`
}

type SwitchStackLag struct {
	ID      string      `json:"id,omitempty"`
	Members []LagMember `json:"members"`
}

type SwitchStack struct {
	ID      string              `json:"id,omitempty"`
	Name    string              `json:"name"`
	Members []SwitchStackMember `json:"members"`
	Lags    []SwitchStackLag    `json:"lags,omitempty"`
}

type McLagPeer struct {
	Role         string  `json:"role"`
	DeviceID     string  `json:"deviceId"`
	LinkPortIdxs []int64 `json:"linkPortIdxs"`
}

type McLagLocalLag struct {
	ID      string      `json:"id,omitempty"`
	Members []LagMember `json:"members"`
}

type McLagDomain struct {
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name"`
	Peers []McLagPeer     `json:"peers"`
	Lags  []McLagLocalLag `json:"lags,omitempty"`
}

type Lag struct {
	ID            string      `json:"id,omitempty"`
	Type          string      `json:"type"`
	SwitchStackID *string     `json:"switchStackId,omitempty"`
	McLagDomainID *string     `json:"mcLagDomainId,omitempty"`
	Members       []LagMember `json:"members"`
}

func (c *Client) ListWANs(ctx context.Context, siteID string) ([]WAN, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list wans site id: %w", err)
	}

	var wans []WAN
	offset := 0

	for {
		response, err := c.apiClient.GetWansOverviewPageWithResponse(ctx, siteUUID, &generated.GetWansOverviewPageParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list wans: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]WAN](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate wan page: %w", err)
		}

		wans = append(wans, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return wans, nil
}

func (c *Client) ListSwitchStacks(ctx context.Context, siteID string) ([]SwitchStack, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list switch stacks site id: %w", err)
	}

	var stacks []SwitchStack
	offset := 0

	for {
		response, err := c.apiClient.GetSwitchStackPageWithResponse(ctx, siteUUID, &generated.GetSwitchStackPageParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list switch stacks: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]SwitchStack](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate switch stack page: %w", err)
		}

		stacks = append(stacks, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return stacks, nil
}

func (c *Client) GetLag(ctx context.Context, siteID, lagID string) (*Lag, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("get lag site id: %w", err)
	}
	lagUUID, err := parseUUID(lagID)
	if err != nil {
		return nil, fmt.Errorf("get lag id: %w", err)
	}

	response, err := c.apiClient.GetLagWithResponse(ctx, siteUUID, lagUUID)
	if err != nil {
		return nil, fmt.Errorf("get lag: %w", err)
	}

	details, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
	if err != nil {
		return nil, err
	}

	lag, err := transcode[Lag](details)
	if err != nil {
		return nil, fmt.Errorf("translate lag: %w", err)
	}

	return &lag, nil
}

func (c *Client) ListMcLagDomains(ctx context.Context, siteID string) ([]McLagDomain, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list mc-lag domains site id: %w", err)
	}

	var domains []McLagDomain
	offset := 0

	for {
		response, err := c.apiClient.GetMcLagDomainPageWithResponse(ctx, siteUUID, &generated.GetMcLagDomainPageParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list mc-lag domains: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]McLagDomain](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate mc-lag domain page: %w", err)
		}

		domains = append(domains, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return domains, nil
}

func (c *Client) ListLags(ctx context.Context, siteID string) ([]Lag, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list lags site id: %w", err)
	}

	var lags []Lag
	offset := 0

	for {
		response, err := c.apiClient.GetLagPageWithResponse(ctx, siteUUID, &generated.GetLagPageParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list lags: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]Lag](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate lag page: %w", err)
		}

		lags = append(lags, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return lags, nil
}
