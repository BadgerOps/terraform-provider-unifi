# Look up an existing VPN server by name.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_vpn_server" "remote_access" {
  site_id = data.unifi_site.main.id
  name    = "Remote Access"
}
