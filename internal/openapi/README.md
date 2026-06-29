# OpenAPI

This directory is the generated-code boundary for the provider.

- `spec/` contains the committed upstream OpenAPI snapshot and manifest metadata.
- `generated/` contains code produced from that snapshot.
- `../translate/` is where conversions between generated DTOs and the handwritten Terraform/provider models belong.

## Current Scope

The committed generator lane now targets the full committed snapshot.

The shipped UniFi Network `10.3.58` OpenAPI document is `3.1.0`, and `oapi-codegen` `v2.6.0` does not yet advertise OpenAPI `3.1` support. To keep the vendor snapshot untouched while still generating a usable client, the repo uses:

- `oapi-codegen.yaml` for pinned generator configuration
- `overlay.yaml` to downgrade the declared document version to `3.0.3` before generation
- `resolve-type-name-collisions` to guard against duplicate generated names across component sections

The Terraform provider still uses an explicit translation boundary in `internal/translate/` instead of coupling provider logic directly to generated DTOs.

The regeneration command is:

```bash
make openapi-generate
```

To check whether the committed snapshot has fallen behind the current stable UniFi package feed, run:

```bash
make openapi-check-upstream
```

GitHub Actions also runs that check weekly and opens or updates a tracking issue when the stable upstream artifact includes a newer `api-docs/integration.json` than the committed snapshot. If the UniFi application package version moves ahead before an OpenAPI snapshot is published in the artifact, the check reports the package metadata but does not raise provider drift because this provider is generated from the OpenAPI document, not the application package version alone.
