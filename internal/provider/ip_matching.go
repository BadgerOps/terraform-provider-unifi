package provider

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

func expandGenericIPMatches(values []string) ([]client.TrafficMatchingItem, error) {
	items := make([]client.TrafficMatchingItem, 0, len(values))

	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			return nil, fmt.Errorf("IP entries must not be empty")
		}

		switch {
		case strings.Contains(value, "/"):
			if _, _, err := net.ParseCIDR(value); err != nil {
				return nil, fmt.Errorf("invalid IP subnet %q: %w", raw, err)
			}
			items = append(items, client.TrafficMatchingItem{
				Type:  "SUBNET",
				Value: value,
			})
		case strings.Contains(value, "-"):
			startRaw, stopRaw, ok := strings.Cut(value, "-")
			if !ok {
				return nil, fmt.Errorf("invalid IP range %q", raw)
			}

			start := strings.TrimSpace(startRaw)
			stop := strings.TrimSpace(stopRaw)
			startIP := net.ParseIP(start)
			stopIP := net.ParseIP(stop)
			if startIP == nil || stopIP == nil {
				return nil, fmt.Errorf("invalid IP range %q", raw)
			}
			if (startIP.To4() == nil) != (stopIP.To4() == nil) {
				return nil, fmt.Errorf("invalid IP range %q: start and stop must use the same IP family", raw)
			}

			items = append(items, client.TrafficMatchingItem{
				Type:  "IP_ADDRESS_RANGE",
				Start: start,
				Stop:  stop,
			})
		default:
			if net.ParseIP(value) == nil {
				return nil, fmt.Errorf("invalid IP address %q", raw)
			}
			items = append(items, client.TrafficMatchingItem{
				Type:  "IP_ADDRESS",
				Value: value,
			})
		}
	}

	return items, nil
}

func flattenGenericIPMatches(items []client.TrafficMatchingItem) ([]string, error) {
	values := make([]string, 0, len(items))

	for _, item := range items {
		switch item.Type {
		case "IP_ADDRESS", "SUBNET":
			value, ok := stringFromAny(item.Value)
			if !ok {
				return nil, fmt.Errorf("%s item is missing a string value", item.Type)
			}
			values = append(values, value)
		case "IP_ADDRESS_RANGE":
			start, ok := stringFromAny(item.Start)
			if !ok {
				return nil, fmt.Errorf("IP range item is missing a string start")
			}
			stop, ok := stringFromAny(item.Stop)
			if !ok {
				return nil, fmt.Errorf("IP range item is missing a string stop")
			}
			values = append(values, start+"-"+stop)
		default:
			return nil, fmt.Errorf("unsupported IP matching item type %q", item.Type)
		}
	}

	sort.Strings(values)
	return values, nil
}
