# Look up an MC-LAG domain by name.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_mc_lag_domain" "leaf_pair" {
  site_id = data.unifi_site.main.id
  name    = "leaf-domain"
}
