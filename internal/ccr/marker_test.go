package ccr

import (
	"strings"
	"testing"
)

func TestAddressForContentUsesSHA256AndSize(t *testing.T) {
	t.Parallel()

	address := AddressForContent([]byte("original tool output"))

	if address.Algorithm != AlgorithmSHA256 {
		t.Fatalf("algorithm = %q, want %q", address.Algorithm, AlgorithmSHA256)
	}
	if len(address.Hex) != SHA256HexLength {
		t.Fatalf("hex length = %d, want %d", len(address.Hex), SHA256HexLength)
	}
	if address.Bytes != len("original tool output") {
		t.Fatalf("bytes = %d, want original size", address.Bytes)
	}
	if again := AddressForContent([]byte("original tool output")); again != address {
		t.Fatalf("address not deterministic: %#v != %#v", again, address)
	}
}

func TestRenderAndParseMarkerRoundTripsBoundedAddress(t *testing.T) {
	t.Parallel()

	address := AddressForContent([]byte(strings.Repeat("diagnostic ", 12)))
	marker, err := RenderMarker(address)
	if err != nil {
		t.Fatalf("RenderMarker() error = %v", err)
	}
	if len(marker) > MaxMarkerLength {
		t.Fatalf("marker length = %d, want <= %d", len(marker), MaxMarkerLength)
	}

	parsed, err := ParseMarker(marker)
	if err != nil {
		t.Fatalf("ParseMarker() error = %v", err)
	}
	if parsed.Address != address {
		t.Fatalf("parsed address = %#v, want %#v", parsed.Address, address)
	}
	if parsed.Version != MarkerVersion {
		t.Fatalf("version = %q, want %q", parsed.Version, MarkerVersion)
	}
}

func TestParseMarkerRejectsMalformedMarkers(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		"not a marker",
		"[[cachy-ccr:v2 sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bytes:1]]",
		"[[cachy-ccr:v1 md5:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bytes:1]]",
		"[[cachy-ccr:v1 sha256:not-hex bytes:1]]",
		"[[cachy-ccr:v1 sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bytes:-1]]",
		strings.Repeat("x", MaxMarkerLength+1),
	}

	for _, marker := range tests {
		t.Run(marker, func(t *testing.T) {
			t.Parallel()

			if _, err := ParseMarker(marker); err == nil {
				t.Fatalf("ParseMarker(%q) error = nil, want error", marker)
			}
		})
	}
}

func TestCanEmitMarkerAllowsOnlySelectedLiveBlocksForSupportedProviders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  MarkerContext
		want bool
	}{
		{name: "openai live selected", ctx: MarkerContext{Provider: ProviderOpenAI, Selected: true, Stable: false}, want: true},
		{name: "anthropic live selected", ctx: MarkerContext{Provider: ProviderAnthropic, Selected: true, Stable: false}, want: true},
		{name: "stable blocked", ctx: MarkerContext{Provider: ProviderOpenAI, Selected: true, Stable: true}, want: false},
		{name: "not selected blocked", ctx: MarkerContext{Provider: ProviderOpenAI, Selected: false, Stable: false}, want: false},
		{name: "unsupported provider blocked", ctx: MarkerContext{Provider: "bedrock", Selected: true, Stable: false}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := CanEmitMarker(tt.ctx); got != tt.want {
				t.Fatalf("CanEmitMarker(%#v) = %v, want %v", tt.ctx, got, tt.want)
			}
		})
	}
}
