# Look up an existing ACL rule by name.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_acl_rule" "block_iot_dns" {
  site_id = data.unifi_site.main.id
  name    = "block-iot-dns"
}
