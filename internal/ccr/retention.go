package ccr

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"time"
)

type RetentionPolicy struct {
	MaxAge   time.Duration
	MaxBytes int64
}

type CleanupOptions struct {
	Now     time.Time
	Policy  RetentionPolicy
	Confirm bool
}

type CleanupResult struct {
	DryRun           bool  `json:"dry_run"`
	WouldRemove      int   `json:"would_remove"`
	Removed          int   `json:"removed"`
	Failed           int   `json:"failed"`
	Kept             int   `json:"kept"`
	BytesWouldRemove int64 `json:"bytes_would_remove"`
	BytesRemoved     int64 `json:"bytes_removed"`
	BytesKept        int64 `json:"bytes_kept"`
}

type Diagnostics struct {
	Root    string `json:"root"`
	Objects int    `json:"objects"`
	Bytes   int64  `json:"bytes"`
}

type objectInfo struct {
	path    string
	size    int64
	modTime time.Time
}

func (s *Store) Diagnostics() (Diagnostics, error) {
	objects, err := s.objects()
	if err != nil {
		return Diagnostics{}, err
	}
	var bytes int64
	for _, object := range objects {
		bytes += object.size
	}
	return Diagnostics{
		Root:    s.root,
		Objects: len(objects),
		Bytes:   bytes,
	}, nil
}

func (s *Store) Cleanup(options CleanupOptions) (CleanupResult, error) {
	objects, err := s.objects()
	if err != nil {
		return CleanupResult{}, err
	}
	if options.Now.IsZero() {
		options.Now = time.Now()
	}

	removeSet := cleanupCandidates(objects, options.Policy, options.Now)
	result := CleanupResult{DryRun: !options.Confirm}
	for _, object := range objects {
		if removeSet[object.path] {
			result.WouldRemove++
			result.BytesWouldRemove += object.size
			if options.Confirm {
				if err := s.remove(object.path); err != nil {
					result.Failed++
					result.Kept++
					result.BytesKept += object.size
					continue
				}
				result.Removed++
				result.BytesRemoved += object.size
			}
			continue
		}
		result.Kept++
		result.BytesKept += object.size
	}
	return result, nil
}

func cleanupCandidates(objects []objectInfo, policy RetentionPolicy, now time.Time) map[string]bool {
	remove := map[string]bool{}
	var kept []objectInfo
	for _, object := range objects {
		if policy.MaxAge > 0 && object.modTime.Before(now.Add(-policy.MaxAge)) {
			remove[object.path] = true
			continue
		}
		kept = append(kept, object)
	}

	if policy.MaxBytes > 0 {
		sort.Slice(kept, func(i, j int) bool {
			return kept[i].modTime.Before(kept[j].modTime)
		})
		var total int64
		for _, object := range kept {
			total += object.size
		}
		for _, object := range kept {
			if total <= policy.MaxBytes {
				break
			}
			remove[object.path] = true
			total -= object.size
		}
	}
	return remove
}

func (s *Store) objects() ([]objectInfo, error) {
	root := filepath.Join(s.root, "objects")
	var objects []objectInfo
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		objects = append(objects, objectInfo{
			path:    path,
			size:    info.Size(),
			modTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return objects, nil
}
