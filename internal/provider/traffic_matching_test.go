package provider

import (
	"reflect"
	"testing"

	"github.com/badgerops/terraform-provider-unifi/internal/client"
)

func TestExpandTrafficMatchingListItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		listType string
		values   []string
		want     []client.TrafficMatchingItem
	}{
		{
			name:     "ports",
			listType: "PORTS",
			values:   []string{"443", "8443-8444"},
			want: []client.TrafficMatchingItem{
				{Type: "PORT_NUMBER", Value: int64(443)},
				{Type: "PORT_NUMBER_RANGE", Start: int64(8443), Stop: int64(8444)},
			},
		},
		{
			name:     "ipv4",
			listType: "IPV4_ADDRESSES",
			values:   []string{"192.168.1.10", "192.168.1.0/24", "192.168.1.20-192.168.1.30"},
			want: []client.TrafficMatchingItem{
				{Type: "IP_ADDRESS", Value: "192.168.1.10"},
				{Type: "SUBNET", Value: "192.168.1.0/24"},
				{Type: "IP_ADDRESS_RANGE", Start: "192.168.1.20", Stop: "192.168.1.30"},
			},
		},
		{
			name:     "ipv6",
			listType: "IPV6_ADDRESSES",
			values:   []string{"2001:db8::10", "2001:db8::/64"},
			want: []client.TrafficMatchingItem{
				{Type: "IP_ADDRESS", Value: "2001:db8::10"},
				{Type: "SUBNET", Value: "2001:db8::/64"},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := expandTrafficMatchingListItems(test.listType, test.values)
			if err != nil {
				t.Fatalf("expandTrafficMatchingListItems returned error: %v", err)
			}

			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("expandTrafficMatchingListItems mismatch\nwant: %#v\ngot:  %#v", test.want, got)
			}
		})
	}
}

func TestFlattenTrafficMatchingListItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		listType string
		items    []client.TrafficMatchingItem
		want     []string
	}{
		{
			name:     "ports",
			listType: "PORTS",
			items: []client.TrafficMatchingItem{
				{Type: "PORT_NUMBER_RANGE", Start: float64(8443), Stop: float64(8444)},
				{Type: "PORT_NUMBER", Value: float64(443)},
			},
			want: []string{"443", "8443-8444"},
		},
		{
			name:     "ipv4",
			listType: "IPV4_ADDRESSES",
			items: []client.TrafficMatchingItem{
				{Type: "IP_ADDRESS_RANGE", Start: "192.168.1.20", Stop: "192.168.1.30"},
				{Type: "SUBNET", Value: "192.168.1.0/24"},
				{Type: "IP_ADDRESS", Value: "192.168.1.10"},
			},
			want: []string{"192.168.1.0/24", "192.168.1.10", "192.168.1.20-192.168.1.30"},
		},
		{
			name:     "ipv6",
			listType: "IPV6_ADDRESSES",
			items: []client.TrafficMatchingItem{
				{Type: "SUBNET", Value: "2001:db8::/64"},
				{Type: "IP_ADDRESS", Value: "2001:db8::10"},
			},
			want: []string{"2001:db8::/64", "2001:db8::10"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := flattenTrafficMatchingListItems(test.listType, test.items)
			if err != nil {
				t.Fatalf("flattenTrafficMatchingListItems returned error: %v", err)
			}

			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("flattenTrafficMatchingListItems mismatch\nwant: %#v\ngot:  %#v", test.want, got)
			}
		})
	}
}
