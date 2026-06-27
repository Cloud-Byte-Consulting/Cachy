// Package platform exposes Cachy's cross-platform config, state, and cache
// path resolution.
package platform

import internalplatform "github.com/cloud-byte-consulting/cachy/internal/platform"

type Options = internalplatform.Options
type Paths = internalplatform.Paths

func ResolvePaths(options Options) (Paths, error) {
	return internalplatform.ResolvePaths(options)
}
