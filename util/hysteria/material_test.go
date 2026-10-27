package hysteria

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewMaterial(t *testing.T) {
	m, err := NewMaterial("")
	if err != nil {
		t.Fatal(err)
	}
	if m.SNI != DefaultSNI || len(m.Auth) != 24 || !strings.HasPrefix(m.Email, "hy2-") {
		t.Fatalf("unexpected material %+v", m)
	}
	if len(strings.Split(m.Pin, ":")) != 32 {
		t.Fatalf("pin should be 32 hex pairs, got %q", m.Pin)
	}
	var stream struct {
		Network     string `json:"network"`
		Security    string `json:"security"`
		TlsSettings struct {
			Certs []struct {
				Certificate []string `json:"certificate"`
				Key         []string `json:"key"`
			} `json:"certificates"`
			Alpn []string `json:"alpn"`
		} `json:"tlsSettings"`
	}
	if err := json.Unmarshal([]byte(m.Stream), &stream); err != nil {
		t.Fatal(err)
	}
	if stream.Network != "hysteria" || stream.Security != "tls" || len(stream.TlsSettings.Alpn) != 1 || stream.TlsSettings.Alpn[0] != "h3" {
		t.Fatalf("bad stream %s", m.Stream)
	}
	cert := strings.Join(stream.TlsSettings.Certs[0].Certificate, "\n")
	if Pin(cert) != m.Pin {
		t.Fatal("pin does not match the stored certificate")
	}
	var settings struct {
		Version int `json:"version"`
		Users   []struct {
			Auth  string `json:"auth"`
			Email string `json:"email"`
		} `json:"users"`
	}
	if err := json.Unmarshal([]byte(m.Settings), &settings); err != nil || settings.Version != 2 || settings.Users[0].Auth != m.Auth {
		t.Fatalf("bad settings %s (%v)", m.Settings, err)
	}
}
