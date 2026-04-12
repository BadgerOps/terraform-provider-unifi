terraform {
  required_version = ">= 1.5.0"

  required_providers {
    unifi = {
      source  = "badgerops/unifi"
      version = "0.0.1"
    }
  }
}

provider "unifi" {
  api_url        = var.api_url
  api_key        = var.api_key
  allow_insecure = var.allow_insecure
}

data "unifi_site" "main" {
  name = var.site_name
}

resource "unifi_network" "trusted" {
  site_id                 = data.unifi_site.main.id
  management              = "GATEWAY"
  name                    = "trusted"
  enabled                 = true
  vlan_id                 = 20
  isolation_enabled       = false
  cellular_backup_enabled = false
  internet_access_enabled = true
  mdns_forwarding_enabled = true

  ipv4_configuration = {
    auto_scale_enabled = false
    host_ip_address    = "10.20.0.1"
    prefix_length      = 24

    dhcp_configuration = {
      mode                            = "SERVER"
      start_ip_address                = "10.20.0.100"
      end_ip_address                  = "10.20.0.240"
      lease_time_seconds              = 86400
      dns_server_ip_addresses         = ["10.20.0.10", "1.1.1.1"]
      ping_conflict_detection_enabled = false
    }
  }
}

resource "unifi_network" "iot" {
  site_id                 = data.unifi_site.main.id
  management              = "GATEWAY"
  name                    = "iot"
  enabled                 = true
  vlan_id                 = 30
  isolation_enabled       = true
  cellular_backup_enabled = false
  internet_access_enabled = true
  mdns_forwarding_enabled = false

  ipv4_configuration = {
    auto_scale_enabled = false
    host_ip_address    = "10.30.0.1"
    prefix_length      = 24

    dhcp_configuration = {
      mode                            = "SERVER"
      start_ip_address                = "10.30.0.100"
      end_ip_address                  = "10.30.0.240"
      lease_time_seconds              = 86400
      dns_server_ip_addresses         = ["10.30.0.10", "1.1.1.1"]
      ping_conflict_detection_enabled = false
    }
  }
}

resource "unifi_wifi_broadcast" "trusted" {
  site_id                                 = data.unifi_site.main.id
  type                                    = "STANDARD"
  name                                    = "trusted"
  enabled                                 = true
  client_isolation_enabled                = false
  hide_name                               = false
  uapsd_enabled                           = true
  multicast_to_unicast_conversion_enabled = false
  broadcasting_frequencies_ghz            = [2.4, 5]
  advertise_device_name                   = false
  arp_proxy_enabled                       = false
  band_steering_enabled                   = true
  bss_transition_enabled                  = true

  network = {
    type       = "SPECIFIC"
    network_id = unifi_network.trusted.id
  }

  security_configuration = {
    type                      = "WPA2_WPA3_PERSONAL"
    passphrase                = var.wifi_passphrase
    pmf_mode                  = "OPTIONAL"
    wpa3_fast_roaming_enabled = false

    sae_configuration = {
      anticlogging_threshold_seconds = 5
      sync_time_seconds              = 5
    }
  }
}

resource "unifi_firewall_zone" "trusted" {
  site_id     = data.unifi_site.main.id
  name        = "trusted"
  network_ids = [unifi_network.trusted.id]
}

resource "unifi_firewall_zone" "iot" {
  site_id     = data.unifi_site.main.id
  name        = "iot"
  network_ids = [unifi_network.iot.id]
}

resource "unifi_firewall_policy" "trusted_to_iot" {
  site_id                 = data.unifi_site.main.id
  enabled                 = true
  name                    = "trusted-to-iot"
  action                  = "ALLOW"
  source_zone_id          = unifi_firewall_zone.trusted.id
  destination_zone_id     = unifi_firewall_zone.iot.id
  ip_version              = "IPV4_AND_IPV6"
  logging_enabled         = false
  destination_network_ids = [unifi_network.iot.id]
}

resource "unifi_dns_policy" "printer" {
  site_id      = data.unifi_site.main.id
  type         = "A_RECORD"
  enabled      = true
  domain       = "printer.iot.internal"
  ipv4_address = "10.30.0.50"
  ttl_seconds  = 300
}

resource "unifi_acl_rule" "block_iot_dns" {
  site_id         = data.unifi_site.main.id
  type            = "IPV4"
  enabled         = true
  name            = "block-iot-dns"
  action          = "BLOCK"
  protocol_filter = ["TCP", "UDP"]

  source_ip_filter = {
    type        = "NETWORKS"
    network_ids = [unifi_network.iot.id]
  }

  destination_ip_filter = {
    type  = "PORTS"
    ports = [53]
  }
}
