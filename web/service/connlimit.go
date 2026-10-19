package service

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/util/connlimit"
)

var (
	connMu      sync.Mutex
	connCounts  = map[int]int{}
	connAllowed = map[int]map[string]struct{}{}

	refreshMu       sync.Mutex
	ssMissingLogged bool
	killFailLogged  bool
)

// GetConnCounts 返回最近一次统计的入站来源 IP 数。
func GetConnCounts() map[int]int {
	connMu.Lock()
	defer connMu.Unlock()
	out := make(map[int]int, len(connCounts))
	for id, n := range connCounts {
		out[id] = n
	}
	return out
}

// ApplyConnCounts 把缓存中的连入数写进入站，供列表接口返回。
func ApplyConnCounts(inbounds []*model.Inbound) {
	counts := GetConnCounts()
	for _, inbound := range inbounds {
		inbound.ConnCount = counts[inbound.Id]
	}
}

// RefreshConnLimits 统计每个入站的来源 IP，并断开超过上限的新来源。
// 上限为 0 时只统计，不踢连接。同一时刻只跑一轮。
func RefreshConnLimits() {
	if !refreshMu.TryLock() {
		return
	}
	defer refreshMu.Unlock()
	if runtime.GOOS != "linux" {
		return
	}

	inbounds, err := (&InboundService{}).GetAllInbounds()
	if err != nil {
		logger.Warning("conn limit list inbounds:", err)
		return
	}
	sockets, err := listEstablished()
	if err != nil {
		if !ssMissingLogged {
			ssMissingLogged = true
			logger.Warning("统计连入需要 ss（iproute2）:", err)
		}
		return
	}

	alive := make(map[int]struct{}, len(inbounds))
	next := make(map[int]int, len(inbounds))
	for _, inbound := range inbounds {
		if inbound == nil {
			continue
		}
		alive[inbound.Id] = struct{}{}
		if inbound.Port <= 0 {
			continue
		}
		observed := connlimit.Peers(sockets, inbound.Port, inbound.Listen)
		next[inbound.Id] = len(observed)
		if !inbound.Enable || inbound.LimitConn <= 0 {
			delete(connAllowed, inbound.Id)
			continue
		}
		keep, kill := connlimit.Decide(inbound.LimitConn, observed, allowedIPs(connAllowed[inbound.Id]))
		kept := make(map[string]struct{}, len(keep))
		for _, ip := range keep {
			kept[ip] = struct{}{}
		}
		connAllowed[inbound.Id] = kept
		if len(kill) == 0 {
			continue
		}
		failed := 0
		n := len(kill)
		if n > 64 {
			n = 64
		}
		for _, ip := range kill[:n] {
			if err := disconnect(inbound.Port, ip); err != nil {
				failed++
				if !killFailLogged {
					killFailLogged = true
					logger.Warning("断开超出上限的连入失败:", err)
				}
			}
		}
		if failed < n {
			logger.Infof("入站 %d 连入上限 %d，断开 %d 个新来源", inbound.Port, inbound.LimitConn, n-failed)
		}
	}
	for id := range connAllowed {
		if _, ok := alive[id]; !ok {
			delete(connAllowed, id)
		}
	}
	connMu.Lock()
	connCounts = next
	connMu.Unlock()
}

func allowedIPs(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for ip := range set {
		out = append(out, ip)
	}
	return out
}

func listEstablished() ([]connlimit.Socket, error) {
	tcp, err := runSS(3*time.Second, "-H", "-tn", "state", "established")
	if err != nil {
		return nil, err
	}
	all := connlimit.ParseSS(tcp)
	udp, err := runSS(3*time.Second, "-H", "-un", "state", "established")
	if err == nil {
		all = append(all, connlimit.ParseSS(udp)...)
	}
	return all, nil
}

func disconnect(port int, ip string) error {
	if net.ParseIP(ip) == nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid disconnect target")
	}
	portArg := ":" + strconv.Itoa(port)
	_, err := runSS(3*time.Second, "-K", "-tn", "state", "established", "sport", "=", portArg, "dst", ip)
	if err != nil {
		return err
	}
	_, _ = runSS(3*time.Second, "-K", "-un", "state", "established", "sport", "=", portArg, "dst", ip)
	return nil
}

func runSS(timeout time.Duration, args ...string) (string, error) {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ss", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, msg)
	}
	return string(out), nil
}
