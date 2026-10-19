package reality

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewKey(t *testing.T) {
	key, err := NewKey()
	if err != nil {
		t.Fatal(err)
	}
	priv, err := base64.RawURLEncoding.DecodeString(key.Private)
	if err != nil || len(priv) != 32 {
		t.Fatalf("private key %q", key.Private)
	}
	pub, err := base64.RawURLEncoding.DecodeString(key.Public)
	if err != nil || len(pub) != 32 {
		t.Fatalf("public key %q", key.Public)
	}
	if key.Private == key.Public {
		t.Fatal("public key should differ from private key")
	}
}

func TestMaterial(t *testing.T) {
	m, err := NewMaterial()
	if err != nil {
		t.Fatal(err)
	}
	if m.SNI != DefaultSNI || m.FP != DefaultFP {
		t.Fatalf("defaults %s %s", m.SNI, m.FP)
	}
	if len(m.ShortID) != 8 {
		t.Fatalf("short id %s", m.ShortID)
	}
	var stream map[string]interface{}
	if err := json.Unmarshal([]byte(m.Stream), &stream); err != nil {
		t.Fatal(err)
	}
	if stream["security"] != "reality" || stream["network"] != "tcp" {
		t.Fatalf("stream %#v", stream)
	}
	if !strings.Contains(m.Settings, m.UUID) {
		t.Fatal("client id missing from settings")
	}
}
