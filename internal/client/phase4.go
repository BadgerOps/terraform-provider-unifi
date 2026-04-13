package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/badgerops/terraform-provider-unifi/internal/openapi/generated"
)

type ReferenceMetadata struct {
	Origin string `json:"origin"`
}

type VPNServer struct {
	ID       string            `json:"id,omitempty"`
	Enabled  bool              `json:"enabled"`
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Metadata ReferenceMetadata `json:"metadata"`
}

type SiteToSiteVPNTunnel struct {
	ID       string            `json:"id,omitempty"`
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Metadata ReferenceMetadata `json:"metadata"`
}

type DPIApplication struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type DPIApplicationCategory struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Country struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (c *Client) ListVPNServers(ctx context.Context, siteID string) ([]VPNServer, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list vpn servers site id: %w", err)
	}

	var servers []VPNServer
	offset := 0

	for {
		response, err := c.apiClient.GetVpnServerPageWithResponse(ctx, siteUUID, &generated.GetVpnServerPageParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list vpn servers: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]VPNServer](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate vpn server page: %w", err)
		}

		servers = append(servers, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return servers, nil
}

func (c *Client) ListSiteToSiteVPNTunnels(ctx context.Context, siteID string) ([]SiteToSiteVPNTunnel, error) {
	siteUUID, err := parseUUID(siteID)
	if err != nil {
		return nil, fmt.Errorf("list site-to-site vpn tunnels site id: %w", err)
	}

	var tunnels []SiteToSiteVPNTunnel
	offset := 0

	for {
		response, err := c.apiClient.GetSiteToSiteVpnTunnelPageWithResponse(ctx, siteUUID, &generated.GetSiteToSiteVpnTunnelPageParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list site-to-site vpn tunnels: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]SiteToSiteVPNTunnel](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate site-to-site vpn tunnel page: %w", err)
		}

		tunnels = append(tunnels, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return tunnels, nil
}

func (c *Client) ListDPIApplications(ctx context.Context) ([]DPIApplication, error) {
	var applications []DPIApplication
	offset := 0

	for {
		response, err := c.apiClient.GetDpiApplicationsWithResponse(ctx, &generated.GetDpiApplicationsParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list dpi applications: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]DPIApplication](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate dpi application page: %w", err)
		}

		applications = append(applications, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return applications, nil
}

func (c *Client) ListDPIApplicationCategories(ctx context.Context) ([]DPIApplicationCategory, error) {
	var categories []DPIApplicationCategory
	offset := 0

	for {
		response, err := c.apiClient.GetDpiApplicationCategoriesWithResponse(ctx, &generated.GetDpiApplicationCategoriesParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list dpi application categories: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]DPIApplicationCategory](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate dpi application category page: %w", err)
		}

		categories = append(categories, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return categories, nil
}

func (c *Client) ListCountries(ctx context.Context) ([]Country, error) {
	var countries []Country
	offset := 0

	for {
		response, err := c.apiClient.GetCountriesWithResponse(ctx, &generated.GetCountriesParams{
			Limit:  pageParam(defaultPageLimit),
			Offset: pageParam(offset),
		})
		if err != nil {
			return nil, fmt.Errorf("list countries: %w", err)
		}

		page, err := requireJSON(response.StatusCode(), response.Body, response.JSON200, http.StatusOK)
		if err != nil {
			return nil, err
		}

		batch, err := transcode[[]Country](page.Data)
		if err != nil {
			return nil, fmt.Errorf("translate country page: %w", err)
		}

		countries = append(countries, batch...)
		offset += len(batch)

		if len(batch) == 0 || int64(offset) >= page.TotalCount {
			break
		}
	}

	return countries, nil
}
