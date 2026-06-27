package ccr

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestDiagnosticsReportsObjectCountAndSize(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	first, err := store.Put([]byte("first"))
	if err != nil {
		t.Fatalf("Put(first) error = %v", err)
	}
	second, err := store.Put([]byte("second object"))
	if err != nil {
		t.Fatalf("Put(second) error = %v", err)
	}

	diagnostics, err := store.Diagnostics()
	if err != nil {
		t.Fatalf("Diagnostics() error = %v", err)
	}

	if diagnostics.Objects != 2 {
		t.Fatalf("objects = %d, want 2", diagnostics.Objects)
	}
	if diagnostics.Bytes != int64(first.Bytes+second.Bytes) {
		t.Fatalf("bytes = %d, want %d", diagnostics.Bytes, first.Bytes+second.Bytes)
	}
	if diagnostics.Root == "" {
		t.Fatal("diagnostics root is empty")
	}
}

func TestDiagnosticsHandlesEmptyStore(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	diagnostics, err := store.Diagnostics()
	if err != nil {
		t.Fatalf("Diagnostics() error = %v", err)
	}
	if diagnostics.Objects != 0 || diagnostics.Bytes != 0 {
		t.Fatalf("diagnostics = %#v, want empty store", diagnostics)
	}
}

func TestCleanupDryRunReportsExpiredObjectsWithoutRemoving(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 16, 20, 0, 0, 0, time.UTC)
	store := newTestStore(t)
	old := putWithModTime(t, store, []byte("old"), now.Add(-2*time.Hour))
	putWithModTime(t, store, []byte("new"), now)

	result, err := store.Cleanup(CleanupOptions{
		Now:     now,
		Policy:  RetentionPolicy{MaxAge: time.Hour},
		Confirm: false,
	})
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	if !result.DryRun || result.WouldRemove != 1 || result.Removed != 0 {
		t.Fatalf("cleanup result = %#v, want dry-run one would-remove", result)
	}
	if _, err := os.Stat(store.ObjectPath(old)); err != nil {
		t.Fatalf("old object removed during dry run: %v", err)
	}
}

func TestCleanupConfirmedRemovesExpiredObjects(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 16, 20, 0, 0, 0, time.UTC)
	store := newTestStore(t)
	old := putWithModTime(t, store, []byte("old"), now.Add(-2*time.Hour))
	kept := putWithModTime(t, store, []byte("new"), now)

	result, err := store.Cleanup(CleanupOptions{
		Now:     now,
		Policy:  RetentionPolicy{MaxAge: time.Hour},
		Confirm: true,
	})
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	if result.DryRun || result.Removed != 1 || result.Kept != 1 {
		t.Fatalf("cleanup result = %#v, want one removed and one kept", result)
	}
	if _, err := os.Stat(store.ObjectPath(old)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old object stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(store.ObjectPath(kept)); err != nil {
		t.Fatalf("kept object missing: %v", err)
	}
}

func TestCleanupTrimsOldestObjectsToMaxBytes(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 16, 20, 0, 0, 0, time.UTC)
	store := newTestStore(t)
	oldest := putWithModTime(t, store, []byte("oldest"), now.Add(-3*time.Hour))
	middle := putWithModTime(t, store, []byte("middle"), now.Add(-2*time.Hour))
	newest := putWithModTime(t, store, []byte("newest"), now.Add(-time.Hour))

	result, err := store.Cleanup(CleanupOptions{
		Now:     now,
		Policy:  RetentionPolicy{MaxBytes: int64(newest.Bytes + 1)},
		Confirm: true,
	})
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	if result.Removed != 2 || result.Kept != 1 {
		t.Fatalf("cleanup result = %#v, want two removed and one kept", result)
	}
	for _, removed := range []Address{oldest, middle} {
		if _, err := os.Stat(store.ObjectPath(removed)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("removed object stat error = %v, want not exist", err)
		}
	}
	if _, err := os.Stat(store.ObjectPath(newest)); err != nil {
		t.Fatalf("newest object missing: %v", err)
	}
}

func TestCleanupReportsDeleteFailures(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 16, 20, 0, 0, 0, time.UTC)
	store := newTestStore(t)
	old := putWithModTime(t, store, []byte("old"), now.Add(-2*time.Hour))
	store.remove = func(string) error {
		return errors.New("delete failed")
	}

	result, err := store.Cleanup(CleanupOptions{
		Now:     now,
		Policy:  RetentionPolicy{MaxAge: time.Hour},
		Confirm: true,
	})
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	if result.Failed != 1 || result.Removed != 0 || result.WouldRemove != 1 {
		t.Fatalf("cleanup result = %#v, want one failed removal", result)
	}
	if _, err := os.Stat(store.ObjectPath(old)); err != nil {
		t.Fatalf("old object should remain after failed delete: %v", err)
	}
}

func putWithModTime(t *testing.T, store *Store, content []byte, modTime time.Time) Address {
	t.Helper()

	address, err := store.Put(content)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if err := os.Chtimes(store.ObjectPath(address), modTime, modTime); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}
	return address
}
