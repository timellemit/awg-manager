package hydraroute

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestGeoStore(t *testing.T) *GeoDataStore {
	t.Helper()
	tmp := t.TempDir()
	return NewGeoDataStore(tmp)
}

func TestAdoptExternalFiles_AddsUnknownFiles(t *testing.T) {
	store := newTestGeoStore(t)

	geositePath := filepath.Join(store.geoDir, "geosite.dat")
	geoipPath := filepath.Join(store.geoDir, "geoip.dat")
	if err := os.WriteFile(geositePath, []byte("fake-content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(geoipPath, []byte("fake-content"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		GeoSiteFiles: []string{geositePath},
		GeoIPFiles:   []string{geoipPath},
	}

	n, err := store.AdoptExternalFiles(cfg)
	if err != nil {
		t.Fatalf("AdoptExternalFiles: %v", err)
	}
	if n != 2 {
		t.Fatalf("adopted count = %d, want 2", n)
	}

	entries := store.List()
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	for _, e := range entries {
		if !e.External {
			t.Errorf("entry %q: External=false, want true", e.Path)
		}
		want := ""
		switch e.Type {
		case "geoip":
			want = GroundZerroGeoIPURL
		case "geosite":
			want = GroundZerroGeoSiteURL
		}
		if e.URL != want {
			t.Errorf("entry %q (type=%s): URL=%q, want %q", e.Path, e.Type, e.URL, want)
		}
	}
}

func TestAdoptExternalFiles_SkipsAlreadyTracked(t *testing.T) {
	store := newTestGeoStore(t)
	existingPath := filepath.Join(store.geoDir, "existing.dat")
	if err := os.WriteFile(existingPath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	store.mu.Lock()
	store.entries = []GeoFileEntry{
		{Type: "geosite", Path: existingPath, URL: "https://example.com/f.dat"},
	}
	store.mu.Unlock()

	cfg := &Config{
		GeoSiteFiles: []string{existingPath},
	}

	n, err := store.AdoptExternalFiles(cfg)
	if err != nil {
		t.Fatalf("AdoptExternalFiles: %v", err)
	}
	if n != 0 {
		t.Fatalf("adopted = %d, want 0 (path already tracked)", n)
	}
	if len(store.List()) != 1 {
		t.Fatalf("entries = %d, want 1 (no duplicate)", len(store.List()))
	}
}

func TestAdoptExternalFiles_SkipsMissingFiles(t *testing.T) {
	store := newTestGeoStore(t)
	cfg := &Config{
		GeoSiteFiles: []string{filepath.Join(store.geoDir, "does-not-exist.dat")},
	}

	n, err := store.AdoptExternalFiles(cfg)
	if err != nil {
		t.Fatalf("AdoptExternalFiles: %v", err)
	}
	if n != 0 {
		t.Fatalf("adopted = %d, want 0 (file missing)", n)
	}
	if len(store.List()) != 0 {
		t.Fatalf("entries = %d, want 0", len(store.List()))
	}
}

func TestAdoptExternalFiles_NilConfig(t *testing.T) {
	store := newTestGeoStore(t)
	n, err := store.AdoptExternalFiles(nil)
	if err != nil {
		t.Fatalf("AdoptExternalFiles(nil): %v", err)
	}
	if n != 0 {
		t.Fatalf("adopted = %d, want 0", n)
	}
}

func TestAdoptExternalFiles_SkipsUnmanagedPaths(t *testing.T) {
	tmp := t.TempDir()
	store := NewGeoDataStore(tmp)

	outsidePath := filepath.Join(tmp, "outside.dat")
	if err := os.WriteFile(outsidePath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	insidePath := filepath.Join(store.geoDir, "inside.dat")
	if err := os.WriteFile(insidePath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		GeoSiteFiles: []string{outsidePath, insidePath},
	}

	n, err := store.AdoptExternalFiles(cfg)
	if err != nil {
		t.Fatalf("AdoptExternalFiles: %v", err)
	}
	if n != 1 {
		t.Fatalf("adopted = %d, want 1 (only path under geoDir)", n)
	}
	entries := store.List()
	if len(entries) != 1 || entries[0].Path != insidePath {
		t.Fatalf("entries = %+v, want only %q", entries, insidePath)
	}
}

func TestUpdate_RejectsExternalEntryWithoutURL(t *testing.T) {
	store := newTestGeoStore(t)
	path := filepath.Join(store.geoDir, "adopted.dat")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	store.entries = []GeoFileEntry{
		{Type: "geosite", Path: path, URL: "", External: true},
	}
	store.mu.Unlock()

	_, err := store.Update(path)
	if err == nil {
		t.Fatal("Update returned nil, expected error for external entry")
	}
	want := "cannot update external file: no source URL on record"
	if err.Error() != want {
		t.Fatalf("err = %q, want %q", err, want)
	}
}

func TestNewGeoDataStore_UsesGeoSubdir(t *testing.T) {
	tmp := t.TempDir()
	store := NewGeoDataStore(tmp)
	want := filepath.Join(tmp, "geo")
	if store.geoDir != want {
		t.Fatalf("geoDir = %q, want %q", store.geoDir, want)
	}
	if st, err := os.Stat(want); err != nil || !st.IsDir() {
		t.Fatalf("geo dir not created: %v", err)
	}
}
