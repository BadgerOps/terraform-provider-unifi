# Look up a switch stack by name.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_switch_stack" "core" {
  site_id = data.unifi_site.main.id
  name    = "core-stack"
}
