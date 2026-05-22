package singbox

import (
	"context"
	"testing"
)

type fakeScanner struct {
	name  string
	rules []DependencyRule
}

func (f *fakeScanner) StoreName() string                                       { return f.name }
func (f *fakeScanner) ListRulesWithInterface(_ context.Context) []DependencyRule { return f.rules }

func TestFindNDMSProxyDependencies_DetectsAcrossStores(t *testing.T) {
	scanners := []DependencyScanner{
		&fakeScanner{name: "clientroute", rules: []DependencyRule{
			{ID: "cr-1", Label: "Devices → US", Interface: "Proxy0"},
			{ID: "cr-2", Label: "к AWG", Interface: "awg0"}, // не singbox-proxy
		}},
		&fakeScanner{name: "accesspolicy", rules: []DependencyRule{
			{ID: "ap-1", Label: "Streaming", Interface: "Proxy1"},
		}},
		&fakeScanner{name: "dnsroute", rules: nil},
		&fakeScanner{name: "hydraroute", rules: nil},
		&fakeScanner{name: "staticroute", rules: nil},
	}
	singboxProxies := map[string]string{
		"Proxy0": "us-vless",
		"Proxy1": "jp-hysteria",
	}
	got := FindNDMSProxyDependencies(context.Background(), singboxProxies, scanners)
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2; got=%+v", len(got), got)
	}
	wantIDs := map[string]bool{"cr-1": true, "ap-1": true}
	for _, d := range got {
		if !wantIDs[d.RecordID] {
			t.Errorf("неожиданная зависимость: %+v", d)
		}
		if d.TunnelTag == "" || d.ProxyInterface == "" {
			t.Errorf("неполная зависимость: %+v", d)
		}
	}
}

func TestFindNDMSProxyDependencies_IgnoresNonSingboxRefs(t *testing.T) {
	scanners := []DependencyScanner{
		&fakeScanner{name: "clientroute", rules: []DependencyRule{
			{ID: "x", Label: "к AWG", Interface: "awg0"},
		}},
	}
	deps := FindNDMSProxyDependencies(context.Background(), map[string]string{"Proxy0": "tag"}, scanners)
	if len(deps) != 0 {
		t.Errorf("не singbox-iface не должен помечаться: %+v", deps)
	}
}

func TestFindNDMSProxyDependencies_EmptyProxiesMap_ReturnsNil(t *testing.T) {
	scanners := []DependencyScanner{
		&fakeScanner{name: "clientroute", rules: []DependencyRule{
			{ID: "x", Label: "foo", Interface: "Proxy0"},
		}},
	}
	if deps := FindNDMSProxyDependencies(context.Background(), nil, scanners); deps != nil {
		t.Errorf("ожидался nil при пустых singboxProxies, got %+v", deps)
	}
}
