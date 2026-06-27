// Package platform exposes Cachy's cross-platform config, state, and cache
// path resolution.
package platform

import internalplatform "truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/internal/platform"

type Options = internalplatform.Options
type Paths = internalplatform.Paths

func ResolvePaths(options Options) (Paths, error) {
	return internalplatform.ResolvePaths(options)
}
