package connlimit

import (
	"net"
	"sort"
	"strconv"
	"strings"
)

// Socket 是一条已建立连接的本地地址和来源 IP。
type Socket struct {
	LocalIP   string
	LocalPort int
	PeerIP    string
}

// ParseSS 解析 ss 的输出。本地地址在前，对端地址在后。
func ParseSS(out string) []Socket {
	var sockets []Socket
	for _, line := range strings.Split(out, "\n") {
		item, ok := ParseSSLine(line)
		if ok {
			sockets = append(sockets, item)
		}
	}
	return sockets
}

// ParseSSLine 解析 ss 的一行。监听套接字和对端为 * 的行会被跳过。
func ParseSSLine(line string) (Socket, bool) {
	var addrs []string
	for _, field := range strings.Fields(line) {
		if _, _, ok := splitHostPort(field); ok {
			addrs = append(addrs, field)
		}
	}
	if len(addrs) < 2 {
		return Socket{}, false
	}
	localIP, localPort, ok1 := splitHostPort(addrs[0])
	peerIP, _, ok2 := splitHostPort(addrs[1])
	if !ok1 || !ok2 {
		return Socket{}, false
	}
	return Socket{LocalIP: localIP, LocalPort: localPort, PeerIP: peerIP}, true
}

func splitHostPort(s string) (string, int, bool) {
	s = strings.TrimSpace(s)
	var host, portStr string
	if strings.HasPrefix(s, "[") {
		i := strings.LastIndex(s, "]:")
		if i < 0 {
			return "", 0, false
		}
		host = s[1:i]
		portStr = s[i+2:]
	} else {
		i := strings.LastIndex(s, ":")
		if i < 0 {
			return "", 0, false
		}
		host = s[:i]
		portStr = s[i+1:]
	}
	if z := strings.Index(host, "%"); z >= 0 {
		host = host[:z]
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 || net.ParseIP(host) == nil {
		return "", 0, false
	}
	return net.ParseIP(host).String(), port, true
}

// Peers 返回匹配该入站端口和监听地址的来源 IP。回环地址不计入。
func Peers(sockets []Socket, port int, listen string) []string {
	seen := map[string]struct{}{}
	for _, item := range sockets {
		if !match(item, port, listen) || isLoopback(item.PeerIP) {
			continue
		}
		seen[item.PeerIP] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for ip := range seen {
		out = append(out, ip)
	}
	sort.Strings(out)
	return out
}

func match(item Socket, port int, listen string) bool {
	if item.LocalPort != port {
		return false
	}
	listen = strings.TrimSpace(listen)
	if listen == "" || listen == "0.0.0.0" || listen == "::" || listen == "::0" {
		return true
	}
	ip := net.ParseIP(listen)
	if ip == nil {
		return false
	}
	return item.LocalIP == ip.String()
}

func isLoopback(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.IsLoopback()
}

// Decide 保留已经接入且仍在线的来源。空位按 IP 顺序接纳新来源，其余返回为需要断开的来源。
// limit 小于等于 0 时全部保留。
func Decide(limit int, observed, previouslyAllowed []string) (keep, kill []string) {
	obs := map[string]struct{}{}
	for _, ip := range observed {
		if ip != "" {
			obs[ip] = struct{}{}
		}
	}
	if limit <= 0 {
		return uniqueSorted(observed), nil
	}

	var stayed []string
	for _, ip := range previouslyAllowed {
		if _, ok := obs[ip]; ok {
			stayed = append(stayed, ip)
		}
	}
	stayed = uniqueSorted(stayed)
	killed := map[string]struct{}{}
	if len(stayed) > limit {
		for _, ip := range stayed[limit:] {
			kill = append(kill, ip)
			killed[ip] = struct{}{}
		}
		stayed = append([]string(nil), stayed[:limit]...)
	}
	kept := map[string]struct{}{}
	for _, ip := range stayed {
		kept[ip] = struct{}{}
	}
	var fresh []string
	for ip := range obs {
		if _, ok := kept[ip]; ok {
			continue
		}
		if _, ok := killed[ip]; ok {
			continue
		}
		fresh = append(fresh, ip)
	}
	sort.Strings(fresh)
	for _, ip := range fresh {
		if len(stayed) >= limit {
			kill = append(kill, ip)
			continue
		}
		stayed = append(stayed, ip)
	}
	return stayed, kill
}

func uniqueSorted(ips []string) []string {
	seen := map[string]struct{}{}
	for _, ip := range ips {
		if ip != "" {
			seen[ip] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for ip := range seen {
		out = append(out, ip)
	}
	sort.Strings(out)
	return out
}
