data "unifi_site" "default" {
  name = "Default"
}

# Fail over when a quarter of the pings to 1.1.1.1 are lost, or they take
# 500 ms, over one minute: a single lost ping doesn't switch the WAN.
resource "unifi_wan_sla" "primary" {
  site_id             = data.unifi_site.default.id
  name                = "ping 1.1.1.1"
  monitor_health_type = "ALL"
  monitors = [{
    type              = "ICMP"
    target            = "1.1.1.1"
    interval          = 1
    time_period       = 60
    loss_threshold    = 25
    latency_threshold = 500
    threshold_policy  = "any"
  }]
}
