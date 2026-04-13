# Look up an adopted switch by name and required feature.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_device" "switch" {
  site_id          = data.unifi_site.main.id
  name             = "core-switch-a"
  required_feature = "switching"
}
