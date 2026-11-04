data "unifi_site" "default" {
  name = "Default"
}

# Keep SNMP off on every device. The community and v3 credentials are not managed.
resource "unifi_setting_snmp" "default" {
  site_id    = data.unifi_site.default.id
  enabled    = false
  enabled_v3 = false
}
