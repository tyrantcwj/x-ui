package accesslog

import "testing"

func TestParseAcceptedLine(t *testing.T) {
	line := "2026/10/19 16:31:05 1.2.3.4:54321 accepted tcp:www.youtube.com:443 [inbound-443 -> direct] email: abcdef1234567890"
	got, ok := ParseLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if got.Time != "2026/10/19 16:31:05" || got.Source != "1.2.3.4" || got.Target != "www.youtube.com:443" || got.Network != "tcp" {
		t.Fatalf("%+v", got)
	}
	if got.InboundTag != "inbound-443" || got.Email != "abcdef1234567890" {
		t.Fatalf("%+v", got)
	}
}

func TestParseIPv6AndNoEmail(t *testing.T) {
	line := "2026/10/19 16:31:05 [2001:db8::1]:54321 accepted tcp:[2001:db8::2]:443 [inbound-8443 >> freedom]"
	got, ok := ParseLine(line)
	if !ok {
		t.Fatal("expected ok")
	}
	if got.Source != "2001:db8::1" || got.Target != "[2001:db8::2]:443" || got.InboundTag != "inbound-8443" || got.Email != "" {
		t.Fatalf("%+v", got)
	}
}

func TestParseRejectedSkipped(t *testing.T) {
	if _, ok := ParseLine("2026/10/19 16:31:05 1.2.3.4:1 rejected tcp:example.com:80"); ok {
		t.Fatal("rejected should be skipped")
	}
}
