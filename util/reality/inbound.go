package reality

import "encoding/json"

const (
	DefaultDest = "www.sega.com:443"
	DefaultSNI  = "www.sega.com"
	DefaultFP   = "random"
)

// Material is everything NOC needs to write a client, plus the JSON stored in the panel.
type Material struct {
	UUID     string
	Private  string
	Public   string
	ShortID  string
	SNI      string
	FP       string
	Settings string
	Stream   string
	Sniffing string
}

// NewMaterial builds a vless + reality inbound the panel UI can still open and edit.
func NewMaterial() (Material, error) {
	id, err := UUID()
	if err != nil {
		return Material{}, err
	}
	key, err := NewKey()
	if err != nil {
		return Material{}, err
	}
	sid, err := ShortID()
	if err != nil {
		return Material{}, err
	}
	return MaterialFrom(id, key.Private, key.Public, sid, DefaultSNI, DefaultFP)
}

// MaterialFrom fills the JSON from already chosen values. Used when reading an existing inbound too.
func MaterialFrom(id, privateKey, publicKey, shortID, sni, fp string) (Material, error) {
	if sni == "" {
		sni = DefaultSNI
	}
	if fp == "" {
		fp = DefaultFP
	}
	dest := sni + ":443"
	if sni == DefaultSNI {
		dest = DefaultDest
	}
	settings, err := json.Marshal(map[string]interface{}{
		"clients": []map[string]string{{
			"id":    id,
			"flow":  "xtls-rprx-vision",
			"email": id,
		}},
		"decryption": "none",
		"fallbacks":  []interface{}{},
	})
	if err != nil {
		return Material{}, err
	}
	stream, err := json.Marshal(map[string]interface{}{
		"network":  "tcp",
		"security": "reality",
		"realitySettings": map[string]interface{}{
			"show":         false,
			"dest":         dest,
			"xver":         0,
			"fingerprint":  fp,
			"serverNames":  []string{sni},
			"privateKey":   privateKey,
			"publicKey":    publicKey,
			"minClientVer": "",
			"maxClientVer": "",
			"maxTimeDiff":  0,
			"shortIds":     []string{shortID},
		},
		"tcpSettings": map[string]interface{}{
			"header": map[string]string{"type": "none"},
		},
	})
	if err != nil {
		return Material{}, err
	}
	sniffing, err := json.Marshal(map[string]interface{}{
		"enabled":      true,
		"destOverride": []string{"http", "tls", "quic"},
	})
	if err != nil {
		return Material{}, err
	}
	return Material{
		UUID:     id,
		Private:  privateKey,
		Public:   publicKey,
		ShortID:  shortID,
		SNI:      sni,
		FP:       fp,
		Settings: string(settings),
		Stream:   string(stream),
		Sniffing: string(sniffing),
	}, nil
}
