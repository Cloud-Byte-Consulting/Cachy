package ccr

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	AlgorithmSHA256 = "sha256"
	MarkerVersion   = "v1"

	SHA256HexLength = 64
	MaxMarkerLength = 128

	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
)

var (
	ErrInvalidAddress = errors.New("invalid CCR address")
	ErrInvalidMarker  = errors.New("invalid CCR marker")
)

type Address struct {
	Algorithm string
	Hex       string
	Bytes     int
}

type Marker struct {
	Version string
	Address Address
}

type MarkerContext struct {
	Provider string
	Selected bool
	Stable   bool
}

func AddressForContent(content []byte) Address {
	sum := sha256.Sum256(content)
	return Address{
		Algorithm: AlgorithmSHA256,
		Hex:       hex.EncodeToString(sum[:]),
		Bytes:     len(content),
	}
}

func RenderMarker(address Address) (string, error) {
	if err := validateAddress(address); err != nil {
		return "", err
	}
	marker := fmt.Sprintf("[[cachy-ccr:%s %s:%s bytes:%d]]", MarkerVersion, address.Algorithm, address.Hex, address.Bytes)
	if len(marker) > MaxMarkerLength {
		return "", ErrInvalidMarker
	}
	return marker, nil
}

func ParseMarker(text string) (Marker, error) {
	if len(text) == 0 || len(text) > MaxMarkerLength {
		return Marker{}, ErrInvalidMarker
	}
	if !strings.HasPrefix(text, "[[cachy-ccr:") || !strings.HasSuffix(text, "]]") {
		return Marker{}, ErrInvalidMarker
	}

	inner := strings.TrimSuffix(strings.TrimPrefix(text, "[[cachy-ccr:"), "]]")
	parts := strings.Fields(inner)
	if len(parts) != 3 {
		return Marker{}, ErrInvalidMarker
	}
	if parts[0] != MarkerVersion {
		return Marker{}, ErrInvalidMarker
	}

	algorithm, hash, ok := strings.Cut(parts[1], ":")
	if !ok {
		return Marker{}, ErrInvalidMarker
	}
	sizeLabel, sizeText, ok := strings.Cut(parts[2], ":")
	if !ok || sizeLabel != "bytes" {
		return Marker{}, ErrInvalidMarker
	}
	size, err := strconv.Atoi(sizeText)
	if err != nil {
		return Marker{}, ErrInvalidMarker
	}

	address := Address{Algorithm: algorithm, Hex: hash, Bytes: size}
	if err := validateAddress(address); err != nil {
		return Marker{}, err
	}
	return Marker{Version: MarkerVersion, Address: address}, nil
}

func CanEmitMarker(ctx MarkerContext) bool {
	if !ctx.Selected || ctx.Stable {
		return false
	}
	return ctx.Provider == ProviderOpenAI || ctx.Provider == ProviderAnthropic
}

func validateAddress(address Address) error {
	if address.Algorithm != AlgorithmSHA256 {
		return ErrInvalidAddress
	}
	if address.Bytes < 0 {
		return ErrInvalidAddress
	}
	if len(address.Hex) != SHA256HexLength {
		return ErrInvalidAddress
	}
	if _, err := hex.DecodeString(address.Hex); err != nil {
		return ErrInvalidAddress
	}
	return nil
}
