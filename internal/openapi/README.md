# OpenAPI

This directory is the generated-code boundary for the provider.

- `spec/` contains the committed upstream OpenAPI snapshot and manifest metadata.
- `generated/` contains code produced from that snapshot.
- `../translate/` is where conversions between generated DTOs and the handwritten Terraform/provider models belong.

## Current Scope

The committed generator lane currently targets a narrow spike:

- site listing
- network listing
- network CRUD

This is intentional. The shipped UniFi Network `10.2.105` OpenAPI document is `3.1.0`, and `oapi-codegen` `v2.6.0` does not yet handle the full document cleanly because of both OpenAPI `3.1` limitations and duplicate schema names in the upstream spec.

The regeneration command is:

```bash
make openapi-generate
```
