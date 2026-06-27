// Package ccr exposes Cachy's content-addressed retrieval marker and local
// object store primitives.
package ccr

import (
	internalccr "github.com/cloud-byte-consulting/cachy/internal/ccr"
	"github.com/cloud-byte-consulting/cachy/pkg/platform"
)

const (
	AlgorithmSHA256   = internalccr.AlgorithmSHA256
	MarkerVersion     = internalccr.MarkerVersion
	SHA256HexLength   = internalccr.SHA256HexLength
	MaxMarkerLength   = internalccr.MaxMarkerLength
	ProviderOpenAI    = internalccr.ProviderOpenAI
	ProviderAnthropic = internalccr.ProviderAnthropic
)

var (
	ErrNotFound       = internalccr.ErrNotFound
	ErrCorrupt        = internalccr.ErrCorrupt
	ErrInvalidAddress = internalccr.ErrInvalidAddress
	ErrInvalidMarker  = internalccr.ErrInvalidMarker
)

type Address = internalccr.Address
type Marker = internalccr.Marker
type MarkerContext = internalccr.MarkerContext
type Store = internalccr.Store
type RetentionPolicy = internalccr.RetentionPolicy
type CleanupOptions = internalccr.CleanupOptions
type CleanupResult = internalccr.CleanupResult
type Diagnostics = internalccr.Diagnostics

func AddressForContent(content []byte) Address {
	return internalccr.AddressForContent(content)
}

func RenderMarker(address Address) (string, error) {
	return internalccr.RenderMarker(address)
}

func ParseMarker(text string) (Marker, error) {
	return internalccr.ParseMarker(text)
}

func CanEmitMarker(ctx MarkerContext) bool {
	return internalccr.CanEmitMarker(ctx)
}

func StoreRoot(paths platform.Paths) string {
	return internalccr.StoreRoot(paths)
}

func NewStore(root string) (*Store, error) {
	return internalccr.NewStore(root)
}
