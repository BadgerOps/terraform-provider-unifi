# Look up a device tag by name.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_device_tag" "staff_aps" {
  site_id = data.unifi_site.main.id
  name    = "staff-aps"
}
