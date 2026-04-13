package provider

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

func expandTrafficMatchingListItems(listType string, values []string) ([]client.TrafficMatchingItem, error) {
	switch listType {
	case "PORTS":
		return expandPortTrafficMatchingItems(values)
	case "IPV4_ADDRESSES":
		return expandIPv4TrafficMatchingItems(values)
	case "IPV6_ADDRESSES":
		return expandIPv6TrafficMatchingItems(values)
	default:
		return nil, fmt.Errorf("unsupported traffic matching list type %q", listType)
	}
}

func flattenTrafficMatchingListItems(listType string, items []client.TrafficMatchingItem) ([]string, error) {
	switch listType {
	case "PORTS":
		return flattenPortTrafficMatchingItems(items)
	case "IPV4_ADDRESSES":
		return flattenIPv4TrafficMatchingItems(items)
	case "IPV6_ADDRESSES":
		return flattenIPv6TrafficMatchingItems(items)
	default:
		return nil, fmt.Errorf("unsupported traffic matching list type %q", listType)
	}
}

func expandPortTrafficMatchingItems(values []string) ([]client.TrafficMatchingItem, error) {
	portMatches, err := expandPortMatches(values)
	if err != nil {
		return nil, err
	}

	items := make([]client.TrafficMatchingItem, 0, len(portMatches))
	for _, match := range portMatches {
		item := client.TrafficMatchingItem{Type: match.Type}
		if match.Value != nil {
			item.Value = *match.Value
		}
		if match.Start != nil {
			item.Start = *match.Start
		}
		if match.Stop != nil {
			item.Stop = *match.Stop
		}
		items = append(items, item)
	}

	return items, nil
}

func flattenPortTrafficMatchingItems(items []client.TrafficMatchingItem) ([]string, error) {
	values := make([]string, 0, len(items))

	for _, item := range items {
		switch item.Type {
		case "PORT_NUMBER":
			value, ok := int64FromAny(item.Value)
			if !ok {
				return nil, fmt.Errorf("port number match is missing a numeric value")
			}
			values = append(values, strconv.FormatInt(value, 10))
		case "PORT_NUMBER_RANGE":
			start, ok := int64FromAny(item.Start)
			if !ok {
				return nil, fmt.Errorf("port range match is missing a numeric start")
			}
			stop, ok := int64FromAny(item.Stop)
			if !ok {
				return nil, fmt.Errorf("port range match is missing a numeric stop")
			}
			values = append(values, fmt.Sprintf("%d-%d", start, stop))
		default:
			return nil, fmt.Errorf("unsupported port traffic matching item type %q", item.Type)
		}
	}

	sort.Strings(values)
	return values, nil
}

func expandIPv4TrafficMatchingItems(values []string) ([]client.TrafficMatchingItem, error) {
	items := make([]client.TrafficMatchingItem, 0, len(values))

	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			return nil, fmt.Errorf("IPv4 entries must not be empty")
		}

		switch {
		case strings.Contains(value, "/"):
			if err := validateCIDR(value, true); err != nil {
				return nil, fmt.Errorf("invalid IPv4 subnet %q: %w", raw, err)
			}
			items = append(items, client.TrafficMatchingItem{
				Type:  "SUBNET",
				Value: value,
			})
		case strings.Contains(value, "-"):
			startRaw, stopRaw, ok := strings.Cut(value, "-")
			if !ok {
				return nil, fmt.Errorf("invalid IPv4 range %q", raw)
			}
			start := strings.TrimSpace(startRaw)
			stop := strings.TrimSpace(stopRaw)
			if err := validateIPAddress(start, true); err != nil {
				return nil, fmt.Errorf("invalid IPv4 range %q: %w", raw, err)
			}
			if err := validateIPAddress(stop, true); err != nil {
				return nil, fmt.Errorf("invalid IPv4 range %q: %w", raw, err)
			}
			items = append(items, client.TrafficMatchingItem{
				Type:  "IP_ADDRESS_RANGE",
				Start: start,
				Stop:  stop,
			})
		default:
			if err := validateIPAddress(value, true); err != nil {
				return nil, fmt.Errorf("invalid IPv4 address %q: %w", raw, err)
			}
			items = append(items, client.TrafficMatchingItem{
				Type:  "IP_ADDRESS",
				Value: value,
			})
		}
	}

	return items, nil
}

func flattenIPv4TrafficMatchingItems(items []client.TrafficMatchingItem) ([]string, error) {
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
				return nil, fmt.Errorf("IPv4 range item is missing a string start")
			}
			stop, ok := stringFromAny(item.Stop)
			if !ok {
				return nil, fmt.Errorf("IPv4 range item is missing a string stop")
			}
			values = append(values, start+"-"+stop)
		default:
			return nil, fmt.Errorf("unsupported IPv4 traffic matching item type %q", item.Type)
		}
	}

	sort.Strings(values)
	return values, nil
}

func expandIPv6TrafficMatchingItems(values []string) ([]client.TrafficMatchingItem, error) {
	items := make([]client.TrafficMatchingItem, 0, len(values))

	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			return nil, fmt.Errorf("IPv6 entries must not be empty")
		}

		if strings.Contains(value, "-") {
			return nil, fmt.Errorf("invalid IPv6 entry %q: IPv6 address ranges are not supported", raw)
		}

		if strings.Contains(value, "/") {
			if err := validateCIDR(value, false); err != nil {
				return nil, fmt.Errorf("invalid IPv6 subnet %q: %w", raw, err)
			}
			items = append(items, client.TrafficMatchingItem{
				Type:  "SUBNET",
				Value: value,
			})
			continue
		}

		if err := validateIPAddress(value, false); err != nil {
			return nil, fmt.Errorf("invalid IPv6 address %q: %w", raw, err)
		}
		items = append(items, client.TrafficMatchingItem{
			Type:  "IP_ADDRESS",
			Value: value,
		})
	}

	return items, nil
}

func flattenIPv6TrafficMatchingItems(items []client.TrafficMatchingItem) ([]string, error) {
	values := make([]string, 0, len(items))

	for _, item := range items {
		switch item.Type {
		case "IP_ADDRESS", "SUBNET":
			value, ok := stringFromAny(item.Value)
			if !ok {
				return nil, fmt.Errorf("%s item is missing a string value", item.Type)
			}
			values = append(values, value)
		default:
			return nil, fmt.Errorf("unsupported IPv6 traffic matching item type %q", item.Type)
		}
	}

	sort.Strings(values)
	return values, nil
}

func validateIPAddress(value string, ipv4 bool) error {
	ip := net.ParseIP(value)
	if ip == nil {
		return fmt.Errorf("not a valid IP address")
	}
	if ipv4 && ip.To4() == nil {
		return fmt.Errorf("not a valid IPv4 address")
	}
	if !ipv4 && ip.To4() != nil {
		return fmt.Errorf("not a valid IPv6 address")
	}

	return nil
}

func validateCIDR(value string, ipv4 bool) error {
	ip, _, err := net.ParseCIDR(value)
	if err != nil {
		return err
	}
	if ipv4 && ip.To4() == nil {
		return fmt.Errorf("not a valid IPv4 subnet")
	}
	if !ipv4 && ip.To4() != nil {
		return fmt.Errorf("not a valid IPv6 subnet")
	}

	return nil
}

func int64FromAny(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return int64(typed), true
	default:
		return 0, false
	}
}

func stringFromAny(value any) (string, bool) {
	typed, ok := value.(string)
	return typed, ok
}
