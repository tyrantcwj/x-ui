package accesslog

import (
	"net"
	"strconv"
	"strings"
)

// Record 是一条已接受的转发记录。Target 是域名或 IP 加端口，不含网页路径。
type Record struct {
	Time       string
	InboundTag string
	Email      string
	Source     string
	Network    string
	Target     string
}

// ParseLine 解析 Xray 访问日志里的一行。拒绝的连接返回 false。
func ParseLine(line string) (Record, bool) {
	line = strings.TrimSpace(line)
	if len(line) < 20 || line[4] != '/' || line[7] != '/' || line[10] != ' ' {
		return Record{}, false
	}
	timeStr := line[:19]
	rest := strings.TrimSpace(line[19:])
	if strings.HasPrefix(rest, ".") {
		i := 1
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}
		rest = strings.TrimSpace(rest[i:])
	}
	const mark = " accepted "
	idx := strings.Index(rest, mark)
	if idx <= 0 {
		return Record{}, false
	}
	from := rest[:idx]
	rest = rest[idx+len(mark):]
	dest, tail := rest, ""
	if sp := strings.IndexByte(rest, ' '); sp >= 0 {
		dest = rest[:sp]
		tail = strings.TrimSpace(rest[sp+1:])
	}
	network, target, ok := splitDest(dest)
	if !ok {
		return Record{}, false
	}
	detour := ""
	if strings.HasPrefix(tail, "[") {
		if end := strings.IndexByte(tail, ']'); end > 1 {
			detour = tail[1:end]
			tail = strings.TrimSpace(tail[end+1:])
		}
	}
	email := ""
	if i := strings.LastIndex(tail, "email: "); i >= 0 {
		email = strings.TrimSpace(tail[i+len("email: "):])
		if fields := strings.Fields(email); len(fields) > 0 {
			email = fields[0]
		}
	}
	source, _, _ := splitHostPort(from)
	if source == "" {
		source = from
	}
	return Record{
		Time:       timeStr,
		InboundTag: inboundTag(detour),
		Email:      email,
		Source:     source,
		Network:    network,
		Target:     target,
	}, true
}

func inboundTag(detour string) string {
	for _, sep := range []string{" ==> ", " >> ", " -> "} {
		if i := strings.Index(detour, sep); i >= 0 {
			return strings.TrimSpace(detour[:i])
		}
	}
	return strings.TrimSpace(detour)
}

func splitDest(dest string) (network, target string, ok bool) {
	i := strings.IndexByte(dest, ':')
	if i <= 0 || i == len(dest)-1 {
		return "", "", false
	}
	network = dest[:i]
	host, port, ok := splitHostPort(dest[i+1:])
	if !ok {
		return "", "", false
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return network, host + ":" + strconv.Itoa(port), true
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
	if z := strings.IndexByte(host, '%'); z >= 0 {
		host = host[:z]
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, false
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), port, true
	}
	if host == "" || strings.ContainsAny(host, " []") {
		return "", 0, false
	}
	return host, port, true
}
