package hydraroute

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLegacyPaths_MovesFromHRDir(t *testing.T) {
	tmp := t.TempDir()
	origHR := hrDir
	hrDir = filepath.Join(tmp, "HydraRoute")
	if err := os.MkdirAll(hrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	defer func() { hrDir = origHR }()

	legacyPath := filepath.Join(hrDir, "geosite_GA.dat")
	if err := os.WriteFile(legacyPath, []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := &GeoDataStore{
		storagePath: filepath.Join(tmp, "hydraroute-geodata.json"),
		geoDir:      filepath.Join(tmp, "geo"),
		tagCache:    make(map[string][]GeoTag),
		entries: []GeoFileEntry{
			{Type: "geosite", Path: legacyPath, URL: "https://example.com/f.dat"},
		},
	}
	if err := os.MkdirAll(store.geoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := store.migrateLegacyPaths(); err != nil {
		t.Fatalf("migrateLegacyPaths: %v", err)
	}

	if len(store.entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(store.entries))
	}
	newPath := store.entries[0].Path
	if !hasPathPrefix(newPath, store.geoDir) {
		t.Fatalf("path %q not under geoDir %q", newPath, store.geoDir)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("migrated file missing: %v", err)
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("legacy file still at %q", legacyPath)
	}
}

func TestAdoptExternalFiles_MigratesLegacyHRPath(t *testing.T) {
	tmp := t.TempDir()
	origHR := hrDir
	hrDir = filepath.Join(tmp, "HydraRoute")
	if err := os.MkdirAll(hrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	defer func() { hrDir = origHR }()

	legacyPath := filepath.Join(hrDir, "geoip_GA.dat")
	if err := os.WriteFile(legacyPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := &GeoDataStore{
		storagePath: filepath.Join(tmp, "hydraroute-geodata.json"),
		geoDir:      filepath.Join(tmp, "geo"),
		tagCache:    make(map[string][]GeoTag),
	}
	if err := os.MkdirAll(store.geoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	n, err := store.AdoptExternalFiles(&Config{GeoIPFiles: []string{legacyPath}})
	if err != nil {
		t.Fatalf("AdoptExternalFiles: %v", err)
	}
	if n != 1 {
		t.Fatalf("adopted = %d, want 1", n)
	}
	if !hasPathPrefix(store.entries[0].Path, store.geoDir) {
		t.Fatalf("adopted path %q not under geoDir", store.entries[0].Path)
	}
}
