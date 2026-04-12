package provider

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

func expandPortMatches(values []string) ([]client.PortMatch, error) {
	matches := make([]client.PortMatch, 0, len(values))

	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			return nil, fmt.Errorf("port entries must not be empty")
		}

		if strings.Contains(value, "-") {
			parts := strings.Split(value, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid port range %q", raw)
			}

			start, err := parsePortNumber(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid port range %q: %w", raw, err)
			}
			stop, err := parsePortNumber(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid port range %q: %w", raw, err)
			}
			if start > stop {
				return nil, fmt.Errorf("invalid port range %q: start must be less than or equal to stop", raw)
			}

			startCopy := start
			stopCopy := stop
			matches = append(matches, client.PortMatch{
				Type:  "PORT_NUMBER_RANGE",
				Start: &startCopy,
				Stop:  &stopCopy,
			})
			continue
		}

		number, err := parsePortNumber(value)
		if err != nil {
			return nil, fmt.Errorf("invalid port %q: %w", raw, err)
		}

		numberCopy := number
		matches = append(matches, client.PortMatch{
			Type:  "PORT_NUMBER",
			Value: &numberCopy,
		})
	}

	return matches, nil
}

func flattenPortMatches(matches []client.PortMatch) []string {
	values := make([]string, 0, len(matches))

	for _, match := range matches {
		switch match.Type {
		case "PORT_NUMBER":
			if match.Value != nil {
				values = append(values, strconv.FormatInt(*match.Value, 10))
			}
		case "PORT_NUMBER_RANGE":
			if match.Start != nil && match.Stop != nil {
				values = append(values, fmt.Sprintf("%d-%d", *match.Start, *match.Stop))
			}
		}
	}

	sort.Strings(values)
	return values
}

func parsePortNumber(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, err
	}
	if value < 1 || value > 65535 {
		return 0, fmt.Errorf("port must be between 1 and 65535")
	}

	return value, nil
}
