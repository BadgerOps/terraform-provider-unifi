---
page_title: "Testing release candidates"
description: |-
  Install an opt-in UniFi provider release candidate, validate your configuration, and report results before a stable release.
---

# Testing release candidates

Release candidates (RCs) let volunteers test changes on different UniFi controllers before a stable release. They use the same `badgerops/unifi` provider address, platform packages, and signed checksums as stable releases. A version such as `0.6.0-rc.1` is a candidate for `0.6.0`; these version numbers are examples, not an announcement of an available release.

Terraform only selects a prerelease when you specify its exact version. Existing stable constraints such as `~> 0.5` do not opt users into RCs. See HashiCorp's [version constraint documentation](https://developer.hashicorp.com/terraform/language/expressions/version-constraints#specify-a-pre-release-version).

## Install an RC

1. Choose a published candidate from the [GitHub releases](https://github.com/BadgerOps/terraform-provider-unifi/releases). Read its changes and known issues. Allow time for the Terraform Registry to ingest the signed release; do not substitute an unrelated version if the candidate is not available yet.
2. Use a test site where possible. Save your current configuration and dependency lock file, and take a controller backup before testing changes that write configuration. State and saved plan files can contain credentials and must stay private.
3. Remove any `dev_overrides` entry for `badgerops/unifi` from your Terraform CLI configuration. A local development override would test your local binary instead of the published RC. Check both `TF_CLI_CONFIG_FILE` and your default CLI configuration. If using a filesystem mirror, make the RC available there first.
4. Change the provider version constraint in your existing configuration. Keep the provider connection settings unchanged:

```hcl
terraform {
  required_providers {
    unifi = {
      source  = "badgerops/unifi"
      version = "= 0.6.0-rc.1"
    }
  }
}
```

Run:

```shell
terraform init -upgrade
terraform validate
terraform plan
```

`init -upgrade` can update other providers and modules too. Review the lock-file diff and confirm `registry.terraform.io/badgerops/unifi` has the exact RC version you selected. Terraform must not print a development-override warning for this provider.

## Test and report results

Start with a plan against your existing state. Compare it with a stable-provider baseline and distinguish existing drift from new differences. A successful plan tests reads and planning; it does not prove create, update, import, or delete behavior.

For changes affecting resources you use, exercise the relevant lifecycle on disposable test resources. Review a plan before applying it. Re-plan afterward to check for unexpected drift. Never run a broad destroy against your normal site to test an RC.

Report both successful tests and bugs on the [GitHub issue tracker](https://github.com/BadgerOps/terraform-provider-unifi/issues), referencing the candidate version. Include:

- Exact provider RC and Terraform versions, plus your OS/architecture.
- Controller model, UniFi OS version (if applicable), and Network application version.
- Local-console or Site Manager connection, affected resources, and operations tested.
- Expected behavior, actual behavior, and a minimal sanitized configuration or diagnostic.

Do not attach API keys, WiFi passphrases, raw Terraform state, or saved plans. A useful success report is: “RC version …, controller …, Network …: refreshed existing resources, tested create/update/import/delete of …, and the final plan was empty.”

## Move to another RC or stable

Change the exact constraint to the next candidate or the published stable version, then repeat `terraform init -upgrade`, `terraform validate`, and `terraform plan`. RC tags are never replaced with new builds.

To return to your previous stable provider, restore its version constraint and review initialization and planning before proceeding. Downgrades are not guaranteed after applying an RC: a newer provider can write state or controller changes an older provider cannot understand. Restoring an old state file alone does not undo controller changes. Describe the problem in an issue if rollback requires help.

## Mirrored installs

Each RC includes a `terraform-provider-unifi_<version>_terraform-mirror.tar.gz` asset. The normal [filesystem mirror instructions](https://github.com/BadgerOps/terraform-provider-unifi#filesystem-mirror-installs) also apply to RC versions. Use the exact RC constraint with the mirror and verify the signed checksum manifest and platform archive before installation. The mirror bundle itself is not included in that checksum manifest.
