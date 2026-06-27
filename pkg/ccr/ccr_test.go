package ccr_test

import (
	"testing"

	"truenas-scale-1.tail5a208d.ts.net/Cloud-Byte-Consulting/Cachy/pkg/ccr"
)

func TestPublicCCRMarkerRoundTripCanBeEmbedded(t *testing.T) {
	t.Parallel()

	address := ccr.AddressForContent([]byte("recoverable content"))
	marker, err := ccr.RenderMarker(address)
	if err != nil {
		t.Fatalf("RenderMarker() error = %v", err)
	}

	parsed, err := ccr.ParseMarker(marker)
	if err != nil {
		t.Fatalf("ParseMarker() error = %v", err)
	}
	if parsed.Address != address {
		t.Fatalf("address = %#v, want %#v", parsed.Address, address)
	}
}
