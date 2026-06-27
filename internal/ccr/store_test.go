package ccr

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloud-byte-consulting/cachy/internal/platform"
)

func TestStoreWritesAndReadsContentByAddress(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	original := []byte("sensitive original tool output")

	address, err := store.Put(original)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if address != AddressForContent(original) {
		t.Fatalf("address = %#v, want content address", address)
	}

	got, err := store.Get(address)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("content = %q, want original", got)
	}
}

func TestStoreDuplicateWritesAreIdempotent(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	original := []byte("duplicate original")

	first, err := store.Put(original)
	if err != nil {
		t.Fatalf("first Put() error = %v", err)
	}
	path := store.ObjectPath(first)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat object: %v", err)
	}

	second, err := store.Put(original)
	if err != nil {
		t.Fatalf("second Put() error = %v", err)
	}
	again, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat object again: %v", err)
	}

	if second != first {
		t.Fatalf("second address = %#v, want %#v", second, first)
	}
	if !again.ModTime().Equal(info.ModTime()) {
		t.Fatalf("duplicate write changed object mtime: %s != %s", again.ModTime(), info.ModTime())
	}
}

func TestStoreRejectsMissingAndInvalidAddresses(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	missing := AddressForContent([]byte("missing"))

	if _, err := store.Get(missing); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(missing) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := store.Get(Address{Algorithm: AlgorithmSHA256, Hex: "../escape", Bytes: 1}); !errors.Is(err, ErrInvalidAddress) {
		t.Fatalf("Get(invalid) error = %v, want %v", err, ErrInvalidAddress)
	}
}

func TestStoreDetectsCorruptContent(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	address, err := store.Put([]byte("original"))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if err := os.WriteFile(store.ObjectPath(address), []byte("tampered"), 0o600); err != nil {
		t.Fatalf("tamper object: %v", err)
	}

	if _, err := store.Get(address); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Get(corrupt) error = %v, want %v", err, ErrCorrupt)
	}
}

func TestStoreRootUsesPlatformStateDir(t *testing.T) {
	t.Parallel()

	paths := platform.Paths{StateDir: filepath.Join("state", "cachy")}
	root := StoreRoot(paths)

	if root != filepath.Join("state", "cachy", "ccr") {
		t.Fatalf("StoreRoot() = %q, want state/cachy/ccr", root)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	return store
}
