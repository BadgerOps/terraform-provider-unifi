data "unifi_site" "default" {
  name = "Default"
}

# Send device, client and firewall logs from every UniFi device to one syslog server.
resource "unifi_setting_remote_syslog" "default" {
  site_id  = data.unifi_site.default.id
  enabled  = true
  ip       = "192.0.2.10"
  port     = 514
  contents = ["device", "client", "firewall_default_policy", "security_detections"]
}
