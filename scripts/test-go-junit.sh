#!/usr/bin/env bash
set -euo pipefail

mkdir -p reports/go

go run github.com/onsi/ginkgo/v2/ginkgo \
  -r \
  --keep-going \
  --junit-report=go-junit.xml \
  --output-dir=reports/go
