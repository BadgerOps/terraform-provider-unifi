# Configure the UniFi provider with an integration API endpoint and API key.
terraform {
  required_providers {
    unifi = {
      source  = "badgerops/unifi"
      version = "0.2.5"
    }
  }
}

provider "unifi" {
  api_url        = "https://unifi.example.com"
  api_key        = "replace-me"
  allow_insecure = false
}
