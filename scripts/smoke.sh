#!/usr/bin/env bash
set -euo pipefail
base="${REGISTRY_BASE_URL:-http://127.0.0.1:8083}"
curl -fsS "$base/healthz" >/dev/null
curl -fsS "$base/v2/" >/dev/null
curl -fsS -X POST "$base/api/v1/repositories" -H 'content-type: application/json' -d '{"name":"smoke/app","tenant":"smoke"}' >/dev/null
echo "artifact registry smoke checks passed"
