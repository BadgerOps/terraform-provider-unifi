data "unifi_site" "default" {
  name = "Default"
}

# SSH to the devices with keys only; firmware updates at 03:00. The SSH username, password and
# keys are set in the UI and never reach Terraform state.
resource "unifi_setting_device_management" "default" {
  site_id                   = data.unifi_site.default.id
  ssh_enabled               = true
  ssh_password_auth_enabled = false
  auto_upgrade              = true
  auto_upgrade_hour         = 3
}
