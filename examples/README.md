# Examples

This directory is the checked-in input set for generated provider documentation.

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
