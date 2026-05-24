package nwg

import (
	"testing"

	"github.com/hoaxisr/awg-manager/internal/tunnel"
)

func TestClassifyNWGState(t *testing.T) {
	slotPresent := func(int) bool { return true }
	slotAbsent := func(int) bool { return false }

	cases := []struct {
		name        string
		rci         NWGState
		supportsASC bool
		hasSlot     func(int) bool
		want        tunnel.State
	}{
		{"running+online -> Running",
			NWGState{ConfLayer: "running", PeerOnline: true}, false, slotAbsent, tunnel.StateRunning},
		{"proxy running+offline, no slot -> Broken",
			NWGState{ConfLayer: "running", PeerOnline: false, PeerRemoteAddr: "127.0.0.1", PeerRemotePort: 51958}, false, slotAbsent, tunnel.StateBroken},
		{"proxy running+offline, remote not localhost -> Broken",
			NWGState{ConfLayer: "running", PeerOnline: false, PeerRemoteAddr: "46.149.74.35", PeerRemotePort: 443}, false, slotPresent, tunnel.StateBroken},
		{"proxy running+offline, coherent -> Starting",
			NWGState{ConfLayer: "running", PeerOnline: false, PeerRemoteAddr: "127.0.0.1", PeerRemotePort: 51958}, false, slotPresent, tunnel.StateStarting},
		{"ASC running+offline -> Starting (no kmod)",
			NWGState{ConfLayer: "running", PeerOnline: false, PeerRemoteAddr: "1.2.3.4", PeerRemotePort: 51820}, true, slotAbsent, tunnel.StateStarting},
		{"disabled -> Stopped",
			NWGState{ConfLayer: "disabled"}, false, slotAbsent, tunnel.StateStopped},
		{"unknown conf -> Unknown",
			NWGState{ConfLayer: ""}, false, slotAbsent, tunnel.StateUnknown},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := classifyNWGState(c.rci, c.supportsASC, c.hasSlot)
			if got != c.want {
				t.Errorf("classifyNWGState = %v, want %v", got, c.want)
			}
		})
	}
}

func TestClassifyNWGState_RunningSkipsSlotCheck(t *testing.T) {
	called := false
	probe := func(int) bool { called = true; return false }
	got := classifyNWGState(NWGState{ConfLayer: "running", PeerOnline: true}, false, probe)
	if got != tunnel.StateRunning {
		t.Fatalf("got %v, want Running", got)
	}
	if called {
		t.Error("hasProxySlot must NOT be called when peer is online")
	}
}
