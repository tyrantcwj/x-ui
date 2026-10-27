package hysteria

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"
)

const DefaultSNI = "www.bing.com"

// Material 是写一个 hysteria2 入站和客户端链接需要的全部东西。
type Material struct {
	Auth     string
	Email    string
	SNI      string
	Pin      string // 证书 sha256, 冒号分隔大写十六进制, 客户端用 pinSHA256 校验
	Settings string
	Stream   string
	Sniffing string
}

// NewMaterial 生成自签证书 (10 年) 和随机密码。自签证书靠客户端固定指纹校验, 不需要域名。
func NewMaterial(sni string) (Material, error) {
	if strings.TrimSpace(sni) == "" {
		sni = DefaultSNI
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return Material{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 100))
	if err != nil {
		return Material{}, err
	}
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: sni},
		DNSNames:     []string{sni},
		NotBefore:    time.Now().Add(-24 * time.Hour),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return Material{}, err
	}
	keyDer, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return Material{}, err
	}
	certPem := strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})))
	keyPem := strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDer})))

	raw := make([]byte, 12)
	if _, err = rand.Read(raw); err != nil {
		return Material{}, err
	}
	auth := hex.EncodeToString(raw)
	email := "hy2-" + auth[:6]
	return From(auth, email, sni, certPem, keyPem)
}

// From 用已有的值拼 JSON (读已有入站时也用)。
func From(auth, email, sni, certPem, keyPem string) (Material, error) {
	pin := Pin(certPem)
	settings, err := json.Marshal(map[string]interface{}{
		"version": 2,
		"users":   []map[string]interface{}{{"auth": auth, "level": 0, "email": email}},
	})
	if err != nil {
		return Material{}, err
	}
	stream, err := json.Marshal(map[string]interface{}{
		"network":  "hysteria",
		"security": "tls",
		"tlsSettings": map[string]interface{}{
			"serverName":       sni,
			"rejectUnknownSni": false,
			"minVersion":       "1.2",
			"maxVersion":       "1.3",
			"cipherSuites":     "",
			"allowInsecure":    false,
			"fingerprint":      "random",
			"certificates": []map[string]interface{}{{
				"ocspStapling": 3600,
				"certificate":  strings.Split(certPem, "\n"),
				"key":          strings.Split(keyPem, "\n"),
			}},
			"alpn": []string{"h3"},
		},
		"hysteriaSettings": map[string]interface{}{
			"version":        2,
			"auth":           auth,
			"udpIdleTimeout": 60,
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
	return Material{Auth: auth, Email: email, SNI: sni, Pin: pin, Settings: string(settings), Stream: string(stream), Sniffing: string(sniffing)}, nil
}

// Pin 算 PEM 证书的 sha256 指纹 (冒号分隔大写); 解析不了返回空。
func Pin(certPem string) string {
	block, _ := pem.Decode([]byte(certPem))
	if block == nil {
		return ""
	}
	sum := sha256.Sum256(block.Bytes)
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(parts, ":")
}
