#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SPEC_PATH="${ROOT_DIR}/internal/openapi/spec/integration.json"
OUTPUT_PATH="${ROOT_DIR}/internal/openapi/generated/network_spike.gen.go"
GENERATOR_MODULE="github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
GENERATOR_VERSION="v2.6.0"

mkdir -p "$(dirname "${OUTPUT_PATH}")"

go run "${GENERATOR_MODULE}@${GENERATOR_VERSION}" \
  -generate types,client \
  -include-operation-ids getSiteOverviewPage,getNetworksOverviewPage,createNetwork,getNetworkDetails,updateNetwork,deleteNetwork \
  -package generated \
  -o "${OUTPUT_PATH}" \
  "${SPEC_PATH}"

gofmt -w "${OUTPUT_PATH}"
