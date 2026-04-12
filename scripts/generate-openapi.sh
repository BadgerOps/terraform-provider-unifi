#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SPEC_PATH="${ROOT_DIR}/internal/openapi/spec/integration.json"
CONFIG_PATH="${ROOT_DIR}/internal/openapi/oapi-codegen.yaml"
OUTPUT_PATH="${ROOT_DIR}/internal/openapi/generated/integration.gen.go"
GENERATOR_MODULE="github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
GENERATOR_VERSION="v2.6.0"

mkdir -p "$(dirname "${OUTPUT_PATH}")"
cd "${ROOT_DIR}"

go run "${GENERATOR_MODULE}@${GENERATOR_VERSION}" \
  -config "${CONFIG_PATH}" \
  "${SPEC_PATH}"

gofmt -w "${OUTPUT_PATH}"
