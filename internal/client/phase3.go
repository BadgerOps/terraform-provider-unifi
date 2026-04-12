package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	var wans []WAN
	offset := 0

	for {
		var response page[WAN]
		query := url.Values{}
		query.Set("limit", fmt.Sprintf("%d", defaultPageLimit))
		query.Set("offset", fmt.Sprintf("%d", offset))

		if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/wans", siteID), query, nil, &response); err != nil {
			return nil, err
		}

		wans = append(wans, response.Data...)
		offset += len(response.Data)

		if len(response.Data) == 0 || int64(offset) >= response.TotalCount {
			break
		}
	}

	return wans, nil
}

func (c *Client) ListSwitchStacks(ctx context.Context, siteID string) ([]SwitchStack, error) {
	var stacks []SwitchStack
	offset := 0

	for {
		var response page[SwitchStack]
		query := url.Values{}
		query.Set("limit", fmt.Sprintf("%d", defaultPageLimit))
		query.Set("offset", fmt.Sprintf("%d", offset))

		if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/switching/switch-stacks", siteID), query, nil, &response); err != nil {
			return nil, err
		}

		stacks = append(stacks, response.Data...)
		offset += len(response.Data)

		if len(response.Data) == 0 || int64(offset) >= response.TotalCount {
			break
		}
	}

	return stacks, nil
}

func (c *Client) GetLag(ctx context.Context, siteID, lagID string) (*Lag, error) {
	var response Lag
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/switching/lags/%s", siteID, lagID), nil, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ListMcLagDomains(ctx context.Context, siteID string) ([]McLagDomain, error) {
	var domains []McLagDomain
	offset := 0

	for {
		var response page[McLagDomain]
		query := url.Values{}
		query.Set("limit", fmt.Sprintf("%d", defaultPageLimit))
		query.Set("offset", fmt.Sprintf("%d", offset))

		if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/switching/mc-lag-domains", siteID), query, nil, &response); err != nil {
			return nil, err
		}

		domains = append(domains, response.Data...)
		offset += len(response.Data)

		if len(response.Data) == 0 || int64(offset) >= response.TotalCount {
			break
		}
	}

	return domains, nil
}

func (c *Client) ListLags(ctx context.Context, siteID string) ([]Lag, error) {
	var lags []Lag
	offset := 0

	for {
		var response page[Lag]
		query := url.Values{}
		query.Set("limit", fmt.Sprintf("%d", defaultPageLimit))
		query.Set("offset", fmt.Sprintf("%d", offset))

		if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/sites/%s/switching/lags", siteID), query, nil, &response); err != nil {
			return nil, err
		}

		lags = append(lags, response.Data...)
		offset += len(response.Data)

		if len(response.Data) == 0 || int64(offset) >= response.TotalCount {
			break
		}
	}

	return lags, nil
}
