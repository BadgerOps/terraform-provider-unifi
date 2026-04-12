variable "api_url" {
  description = "Base URL for the UniFi Network controller."
  type        = string
}

variable "api_key" {
  description = "API key created in the UniFi Network integrations UI."
  type        = string
  sensitive   = true
}

variable "allow_insecure" {
  description = "Disable TLS verification for lab controllers."
  type        = bool
  default     = false
}

variable "site_name" {
  description = "Site name to look up."
  type        = string
  default     = "Default"
}

variable "wifi_passphrase" {
  description = "WiFi passphrase for the example SSID."
  type        = string
  sensitive   = true
}
