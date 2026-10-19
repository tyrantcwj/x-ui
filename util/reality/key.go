package reality

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// Key is an Xray Reality x25519 key pair, encoded the same way Xray stores it.
type Key struct {
	Private string
	Public  string
}

// NewKey creates a clamped x25519 key pair. The public key is what clients put in pbk.
func NewKey() (Key, error) {
	var private [32]byte
	if _, err := rand.Read(private[:]); err != nil {
		return Key{}, err
	}
	private[0] &= 248
	private[31] &= 127
	private[31] |= 64

	public, err := curve25519.X25519(private[:], curve25519.Basepoint)
	if err != nil {
		return Key{}, err
	}
	return Key{
		Private: base64.RawURLEncoding.EncodeToString(private[:]),
		Public:  base64.RawURLEncoding.EncodeToString(public),
	}, nil
}

// ShortID is 8 hex characters, matching the panel's random short id.
func ShortID() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// UUID returns a random version-4 UUID.
func UUID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:]), nil
}
