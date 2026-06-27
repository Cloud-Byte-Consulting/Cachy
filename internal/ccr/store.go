package ccr

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/cloud-byte-consulting/cachy/internal/platform"
)

var (
	ErrNotFound = errors.New("CCR content not found")
	ErrCorrupt  = errors.New("CCR content hash mismatch")
)

type Store struct {
	root   string
	remove func(string) error
}

func StoreRoot(paths platform.Paths) string {
	return filepath.Join(paths.StateDir, "ccr")
}

func NewStore(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("CCR store root is required")
	}
	return &Store{root: root, remove: os.Remove}, nil
}

func (s *Store) Put(content []byte) (Address, error) {
	address := AddressForContent(content)
	path := s.ObjectPath(address)
	if _, err := os.Stat(path); err == nil {
		return address, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Address{}, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return Address{}, err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return Address{}, err
	}
	tempName := temp.Name()
	defer func() {
		_ = os.Remove(tempName)
	}()

	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return Address{}, err
	}
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return Address{}, err
	}
	if err := temp.Close(); err != nil {
		return Address{}, err
	}
	if err := os.Rename(tempName, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return address, nil
		}
		return Address{}, err
	}
	return address, nil
}

func (s *Store) Get(address Address) ([]byte, error) {
	if err := validateAddress(address); err != nil {
		return nil, err
	}
	content, err := os.ReadFile(s.ObjectPath(address))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if AddressForContent(content) != address {
		return nil, ErrCorrupt
	}
	return content, nil
}

func (s *Store) ObjectPath(address Address) string {
	if err := validateAddress(address); err != nil {
		return ""
	}
	return filepath.Join(s.root, "objects", address.Algorithm, address.Hex[:2], address.Hex)
}
