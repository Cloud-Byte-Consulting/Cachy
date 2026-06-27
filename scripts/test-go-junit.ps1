[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

New-Item -ItemType Directory -Path 'reports/go' -Force | Out-Null

# Prevent Windows installer-detection/UAC heuristics from intercepting package
# test binaries such as internal/install/install.test.
$env:__COMPAT_LAYER = 'RunAsInvoker'

go run github.com/onsi/ginkgo/v2/ginkgo `
  -r `
  --keep-going `
  --junit-report=go-junit.xml `
  --output-dir=reports/go
