package hydraroute

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hoaxisr/awg-manager/internal/storage"
)

const geoSubdir = "geo"

// hasPathPrefix reports whether clean is equal to prefix or is a child of prefix.
func hasPathPrefix(clean, prefix string) bool {
	clean = filepath.Clean(clean)
	prefix = filepath.Clean(prefix)
	if clean == prefix {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(clean, prefix+sep)
}

func (s *GeoDataStore) isManagedPath(path string) bool {
	if s == nil || s.geoDir == "" {
		return false
	}
	return hasPathPrefix(filepath.Clean(path), s.geoDir)
}

// relocateIntoGeoDir moves or copies src into geoDir, returning the final path.
func (s *GeoDataStore) relocateIntoGeoDir(src string) (string, error) {
	src = filepath.Clean(src)
	if hasPathPrefix(src, s.geoDir) {
		return src, nil
	}

	base := filepath.Base(src)
	if base == "" || base == "." {
		return "", fmt.Errorf("invalid source path: %s", src)
	}

	dest := s.resolveConflict(filepath.Join(s.geoDir, base))
	if err := os.MkdirAll(s.geoDir, storage.DirPermission); err != nil {
		return "", fmt.Errorf("create geo dir: %w", err)
	}

	if err := os.Rename(src, dest); err == nil {
		return dest, nil
	}

	if err := copyFile(src, dest); err != nil {
		return "", err
	}
	_ = os.Remove(src)
	return dest, nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, storage.FilePermission)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dest)
		return fmt.Errorf("copy to %s: %w", dest, err)
	}
	if err := out.Close(); err != nil {
		os.Remove(dest)
		return fmt.Errorf("close %s: %w", dest, err)
	}
	return nil
}

// migrateLegacyPaths moves tracked files and metadata paths from the legacy
// HydraRoute directory into awg-manager's geoDir.
func (s *GeoDataStore) migrateLegacyPaths() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.geoDir, storage.DirPermission); err != nil {
		return fmt.Errorf("create geo dir: %w", err)
	}

	changed := false
	next := make([]GeoFileEntry, 0, len(s.entries))
	for _, e := range s.entries {
		path := filepath.Clean(e.Path)
		if hasPathPrefix(path, s.geoDir) {
			next = append(next, e)
			continue
		}
		if !hasPathPrefix(path, hrDir) {
			next = append(next, e)
			continue
		}
		if _, err := os.Stat(path); err != nil {
			delete(s.tagCache, path)
			changed = true
			continue
		}
		dest, err := s.relocateIntoGeoDir(path)
		if err != nil {
			return fmt.Errorf("migrate %s: %w", path, err)
		}
		delete(s.tagCache, path)
		e.Path = dest
		next = append(next, e)
		changed = true
	}
	s.entries = next

	if !changed {
		return nil
	}
	return s.saveUnlocked()
}
