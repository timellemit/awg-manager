package hydraroute

import (
	"os"
	"path/filepath"
	"testing"
)

func buildDomainItem(domainType int, value string) []byte {
	var domain []byte
	domain = append(domain, varintField(1, uint64(domainType))...)
	domain = append(domain, field(2, []byte(value))...)
	return field(2, domain)
}

func buildCidrItem(ip []byte, prefix uint32) []byte {
	var cidr []byte
	cidr = append(cidr, field(1, ip)...)
	if prefix > 0 {
		cidr = append(cidr, varintField(2, uint64(prefix))...)
	}
	return field(2, cidr)
}

func buildGeoEntryWithItems(ccField int, name string, items [][]byte) []byte {
	var entry []byte
	entry = append(entry, field(ccField, []byte(name))...)
	for _, item := range items {
		entry = append(entry, item...)
	}
	return entry
}

func TestExtractGeoSiteTagLines(t *testing.T) {
	entries := [][]byte{
		buildGeoEntryWithItems(1, "GOOGLE", [][]byte{
			buildDomainItem(0, "google.com"),
			buildDomainItem(2, "googlevideo.com"),
			buildDomainItem(1, `^ads\.google\.`),
		}),
		buildGeoEntryWithItems(1, "TELEGRAM", [][]byte{
			buildDomainItem(3, "t.me"),
		}),
	}
	dat := buildGeoDAT(entries)
	tmp := filepath.Join(t.TempDir(), "geosite.dat")
	if err := os.WriteFile(tmp, dat, 0o644); err != nil {
		t.Fatal(err)
	}

	lines, err := ExtractGeoSiteTagLines(tmp, "google")
	if err != nil {
		t.Fatalf("ExtractGeoSiteTagLines: %v", err)
	}
	want := []string{
		"google.com",
		".googlevideo.com",
		`domain_regex:^ads\.google\.`,
	}
	if len(lines) != len(want) {
		t.Fatalf("lines = %v, want %v", lines, want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("lines[%d] = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestExtractGeoIPTagLines(t *testing.T) {
	entries := [][]byte{
		buildGeoEntryWithItems(1, "RU", [][]byte{
			buildCidrItem([]byte{5, 8, 0, 0}, 21),
			buildCidrItem([]byte{1, 1, 1, 1}, 32),
		}),
	}
	dat := buildGeoDAT(entries)
	tmp := filepath.Join(t.TempDir(), "geoip.dat")
	if err := os.WriteFile(tmp, dat, 0o644); err != nil {
		t.Fatal(err)
	}

	lines, err := ExtractGeoIPTagLines(tmp, "ru")
	if err != nil {
		t.Fatalf("ExtractGeoIPTagLines: %v", err)
	}
	want := []string{"5.8.0.0/21", "1.1.1.1/32"}
	if len(lines) != len(want) {
		t.Fatalf("lines = %v, want %v", lines, want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("lines[%d] = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestExtractGeoSiteTagLines_NotFound(t *testing.T) {
	entries := [][]byte{buildGeoEntryWithItems(1, "A", nil)}
	dat := buildGeoDAT(entries)
	tmp := filepath.Join(t.TempDir(), "geosite.dat")
	if err := os.WriteFile(tmp, dat, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ExtractGeoSiteTagLines(tmp, "MISSING")
	if err == nil {
		t.Fatal("expected error for missing tag")
	}
}
