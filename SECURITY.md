# Security policy

## Supported versions

Security fixes are made on the default branch and released in the newest provider version. Users should upgrade to the latest available release; older releases do not receive separate security patches unless maintainers announce an exception.

## Reporting a vulnerability

Use [GitHub private vulnerability reporting](https://github.com/BadgerOps/terraform-provider-unifi/security/advisories/new) to report a suspected vulnerability. Do not open a public issue, discussion, or pull request before coordinated disclosure.

Include, when possible:

- Affected provider and Terraform versions.
- A description of the impact and affected configurations.
- Minimal reproduction steps or a proof of concept.
- Whether credentials, Terraform state, release artifacts, or a UniFi controller are exposed.
- Any suggested mitigation.

Remove real API keys, credentials, customer data, controller addresses, and Terraform state from the report. If sensitive material is necessary to reproduce the problem, first ask the maintainers how to transfer it safely.

Maintainers aim to acknowledge a report within three business days and provide an initial assessment within seven business days. Remediation and disclosure timing depend on severity and coordination with affected upstream projects.

The maintainers will coordinate publication of an advisory and fixed release with the reporter. Please allow a reasonable remediation period before public disclosure.

Vulnerabilities in UniFi products or Terraform itself should also be reported to the relevant upstream vendor. If provider behavior increases the impact, report it here as well.
