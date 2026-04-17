# Basic Site Example

This directory contains a larger end-to-end example configuration for standing up common UniFi objects with the `badgerops/unifi` provider.

It is an operator reference example inside the provider repository, not a published Terraform module.

The configuration demonstrates:

- provider configuration and site lookup
- managed networks and WiFi broadcasts
- firewall zones and firewall policies
- DNS policies and ACL rules

If BadgerOps publishes reusable Terraform modules built on top of this provider, those modules should live in separate repositories rather than under this provider repository.
