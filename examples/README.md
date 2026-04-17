# Examples

This directory is the checked-in input set for generated provider documentation.

This repository is a Terraform provider repository, not a module repository. Any Terraform configuration under `examples/` exists to support generated provider docs or operator reference workflows. Reusable modules that BadgerOps intends to publish should live in separate repositories that follow the Registry module naming convention `terraform-<provider>-<name>`.

`tfplugindocs` reads these paths by convention:

- `provider/provider.tf` for the provider index example
- `data-sources/<full_name>/data-source.tf` for each data source example
- `resources/<full_name>/resource.tf` for each resource example
- `resources/<full_name>/import.sh` for each resource import example

The generation flow is:

1. Update the provider schema or examples.
2. Run `make docs-generate`.
3. Review changes under `docs/`.
4. Run `make docs-check` before committing.

Example files should stay minimal, valid, and focused on the intended public workflow for that resource or data source.

Example subdirectories should include a local `README.md` when they contain runnable Terraform configuration so accidental Registry module indexing describes them as examples instead of undocumented internal-only submodules.
