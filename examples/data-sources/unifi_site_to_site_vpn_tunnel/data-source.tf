# Look up an existing site-to-site VPN tunnel by name.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_site_to_site_vpn_tunnel" "branch" {
  site_id = data.unifi_site.main.id
  name    = "Branch Office Tunnel"
}
