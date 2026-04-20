# OpenAPI

This directory is the generated-code boundary for the provider.

- `spec/` contains the committed upstream OpenAPI snapshot and manifest metadata.
- `generated/` contains code produced from that snapshot.
- `../translate/` is where conversions between generated DTOs and the handwritten Terraform/provider models belong.

## Current Scope

The committed generator lane now targets the full committed snapshot.

The shipped UniFi Network `10.2.105` OpenAPI document is `3.1.0`, and `oapi-codegen` `v2.6.0` does not yet advertise OpenAPI `3.1` support. To keep the vendor snapshot untouched while still generating a usable client, the repo uses:

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

GitHub Actions also runs that check weekly and opens or updates a tracking issue when the stable upstream package version moves ahead of the committed snapshot. If a packaged `integration.json` is available in the upstream artifact, the same check also compares the snapshot contents directly.
