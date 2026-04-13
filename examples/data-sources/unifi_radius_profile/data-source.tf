# Look up a RADIUS profile by name.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_radius_profile" "corp_radius" {
  site_id = data.unifi_site.main.id
  name    = "Corp RADIUS"
}
