package connlimit

import (
	"strings"
	"testing"
)

func TestParseSSLine(t *testing.T) {
	cases := []struct {
		line      string
		localIP   string
		localPort int
		peerIP    string
		ok        bool
	}{
		{"ESTAB 0 0 10.0.22.15:443 1.2.3.4:54321", "10.0.22.15", 443, "1.2.3.4", true},
		{"0 0 10.0.22.15:8443 8.8.8.8:1", "10.0.22.15", 8443, "8.8.8.8", true},
		{"ESTAB 0 0 [::]:443 [2001:db8::1]:54321", "::", 443, "2001:db8::1", true},
		{"ESTAB 0 0 127.0.0.1:62789 127.0.0.1:40000", "127.0.0.1", 62789, "127.0.0.1", true},
		{"LISTEN 0 128 0.0.0.0:22 0.0.0.0:*", "", 0, "", false},
		{"not a socket line", "", 0, "", false},
	}
	for _, tc := range cases {
		got, ok := ParseSSLine(tc.line)
		if ok != tc.ok {
			t.Fatalf("%q ok=%v want %v", tc.line, ok, tc.ok)
		}
		if !ok {
			continue
		}
		if got.LocalIP != tc.localIP || got.LocalPort != tc.localPort || got.PeerIP != tc.peerIP {
			t.Fatalf("%q got %+v", tc.line, got)
		}
	}
}

func TestDecideKeepsExistingAndFillsSlots(t *testing.T) {
	keep, kill := Decide(2, []string{"c", "a", "b"}, []string{"b", "a"})
	if join(keep) != "a,b" || join(kill) != "c" {
		t.Fatalf("keep=%v kill=%v", keep, kill)
	}
}

func TestDecideDropsPreviouslyAllowedWhenLimitShrinks(t *testing.T) {
	keep, kill := Decide(1, []string{"b", "a"}, []string{"a", "b"})
	if join(keep) != "a" || join(kill) != "b" {
		t.Fatalf("keep=%v kill=%v", keep, kill)
	}
}

func TestDecideUnlimited(t *testing.T) {
	keep, kill := Decide(0, []string{"b", "a", "a"}, nil)
	if join(keep) != "a,b" || len(kill) != 0 {
		t.Fatalf("keep=%v kill=%v", keep, kill)
	}
}

func TestDecideReplacesGonePeer(t *testing.T) {
	keep, kill := Decide(1, []string{"c", "b"}, []string{"a"})
	if join(keep) != "b" || join(kill) != "c" {
		t.Fatalf("keep=%v kill=%v", keep, kill)
	}
}

func TestPeersRespectsListen(t *testing.T) {
	sockets := []Socket{
		{LocalIP: "10.0.0.8", LocalPort: 443, PeerIP: "1.1.1.1"},
		{LocalIP: "10.0.0.9", LocalPort: 443, PeerIP: "2.2.2.2"},
		{LocalIP: "10.0.0.8", LocalPort: 443, PeerIP: "127.0.0.1"},
		{LocalIP: "10.0.0.8", LocalPort: 443, PeerIP: "1.1.1.1"},
	}
	if got := Peers(sockets, 443, "10.0.0.8"); join(got) != "1.1.1.1" {
		t.Fatalf("got %v", got)
	}
	if got := Peers(sockets, 443, ""); join(got) != "1.1.1.1,2.2.2.2" {
		t.Fatalf("any listen got %v", got)
	}
}

func join(items []string) string {
	return strings.Join(items, ",")
}
