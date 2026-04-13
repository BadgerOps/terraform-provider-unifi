# Look up an existing DNS policy by domain and type.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_dns_policy" "printer" {
  site_id = data.unifi_site.main.id
  domain  = "printer.iot.internal"
  type    = "A_RECORD"
}
