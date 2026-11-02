data "unifi_site" "default" {
  name = "Default"
}

# Let the devices use a local time server first, then a public pool.
resource "unifi_setting_ntp" "default" {
  site_id = data.unifi_site.default.id
  mode    = "manual"
  servers = ["192.0.2.1", "pool.ntp.org"]
}
