# Provider Example

This directory contains the minimal provider configuration example used to render the provider landing page in generated Registry docs.

It is an example embedded in the provider source repository, not a standalone Terraform module.

Its primary purpose is:

- provide a minimal `provider "unifi"` example for generated documentation
- keep the documented source address and version syntax exercised in version drift checks

Use the provider from the Terraform Registry in normal configurations:

```hcl
terraform {
  required_providers {
    unifi = {
      source  = "badgerops/unifi"
      version = "0.2.9"
    }
  }
}
```
