package xraycfg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// WithEmails 给没有 email 的账号补上稳定标识，访问日志才能分到具体用户。
// 已有 email 保持不变。密码本身不会写进标识。
func WithEmails(settings string, inboundID int) string {
	if settings == "" {
		return settings
	}
	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(settings), &doc); err != nil || doc == nil {
		return settings
	}
	changed := false
	if clients, ok := doc["clients"].([]interface{}); ok {
		for _, item := range clients {
			client, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if email, _ := client["email"].(string); email != "" {
				continue
			}
			if id, _ := client["id"].(string); id != "" {
				client["email"] = id
				changed = true
				continue
			}
			if password, _ := client["password"].(string); password != "" {
				client["email"] = passwordEmail(inboundID, password)
				changed = true
			}
		}
	} else if password, _ := doc["password"].(string); password != "" {
		if email, _ := doc["email"].(string); email == "" {
			doc["email"] = passwordEmail(inboundID, password)
			changed = true
		}
	}
	if !changed {
		return settings
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		return settings
	}
	return string(encoded)
}

// WithSniffing 在用户没有关闭嗅探时打开 http/tls/quic，这样访问记录能拿到域名。
func WithSniffing(sniffing string) string {
	doc := map[string]interface{}{}
	if sniffing != "" {
		if err := json.Unmarshal([]byte(sniffing), &doc); err != nil || doc == nil {
			return sniffing
		}
	}
	if enabled, ok := doc["enabled"].(bool); ok && !enabled {
		return sniffing
	}
	doc["enabled"] = true
	override, _ := doc["destOverride"].([]interface{})
	if len(override) == 0 {
		doc["destOverride"] = []string{"http", "tls", "quic"}
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		return sniffing
	}
	return string(encoded)
}

// UserLabel 把访问日志里的 email 转成面板上显示的用户名。
func UserLabel(inboundID int, settings, email string) string {
	if email == "" {
		return ""
	}
	var doc map[string]interface{}
	if settings != "" {
		_ = json.Unmarshal([]byte(settings), &doc)
	}
	if clients, ok := doc["clients"].([]interface{}); ok {
		for i, item := range clients {
			client, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if id, _ := client["id"].(string); id != "" && id == email {
				if len(id) > 8 {
					return id[:8]
				}
				return id
			}
			if own, _ := client["email"].(string); own != "" && own == email {
				return own
			}
			if password, _ := client["password"].(string); password != "" && passwordEmail(inboundID, password) == email {
				return fmt.Sprintf("用户%d", i+1)
			}
		}
	}
	if password, _ := doc["password"].(string); password != "" && passwordEmail(inboundID, password) == email {
		return "默认"
	}
	return email
}

func passwordEmail(inboundID int, password string) string {
	sum := sha256.Sum256([]byte(password))
	return fmt.Sprintf("u%d-%s", inboundID, hex.EncodeToString(sum[:])[:8])
}
