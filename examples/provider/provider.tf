# Configure the UniFi provider with an integration API endpoint and API key.
terraform {
  required_providers {
    unifi = {
      source  = "badgerops/unifi"
      version = "0.4.0"
    }
  }
}

provider "unifi" {
  api_url        = "https://unifi.example.com"
  api_key        = "replace-me"
  allow_insecure = false

  # Raise this when the controller reprovisions devices mid-apply and requests
  # time out. Defaults to 30 seconds.
  # request_timeout_seconds = 300
}
