package service

import (
	"os"
	"strings"
	"sync"

	"x-ui/config"
	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/util/accesslog"
	"x-ui/util/xraycfg"
)

const (
	maxAccessRecords = 500
	accessLogLimit   = 8 << 20
	accessTailWindow = 256 << 10
)

// Visit 是面板上显示的一条访问记录。
type Visit struct {
	Time    string `json:"time"`
	Inbound string `json:"inbound"`
	Remark  string `json:"remark"`
	User    string `json:"user"`
	Source  string `json:"source"`
	Network string `json:"network"`
	Target  string `json:"target"`
}

var accessState = struct {
	sync.Mutex
	offset  int64
	started bool
	records []accesslog.Record
}{}

// PollAccessLog 读入新的访问日志。日志超过上限后截断，Xray 以追加方式写入，截断后仍可继续写。
func PollAccessLog() {
	path := config.GetAccessLogPath()
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return
	}
	size := info.Size()

	accessState.Lock()
	defer accessState.Unlock()
	if size < accessState.offset {
		accessState.offset = 0
		accessState.started = false
	}
	skipPartial := false
	if !accessState.started && size > accessTailWindow {
		accessState.offset = size - accessTailWindow
		skipPartial = true
	}
	accessState.started = true
	if size == accessState.offset {
		return
	}
	buf := make([]byte, size-accessState.offset)
	n, err := f.ReadAt(buf, accessState.offset)
	if n <= 0 {
		if err != nil && !os.IsNotExist(err) {
			logger.Debug("read access log:", err)
		}
		return
	}
	text := string(buf[:n])
	lines := strings.Split(text, "\n")
	end := len(lines)
	consumed := n
	if !strings.HasSuffix(text, "\n") && end > 0 {
		consumed -= len(lines[end-1])
		end--
	}
	start := 0
	if skipPartial && end > 0 {
		start = 1
	}
	for i := start; i < end; i++ {
		record, ok := accesslog.ParseLine(lines[i])
		if !ok || record.InboundTag == "" || record.InboundTag == "api" {
			continue
		}
		accessState.records = append(accessState.records, record)
	}
	if extra := len(accessState.records) - maxAccessRecords; extra > 0 {
		accessState.records = append([]accesslog.Record(nil), accessState.records[extra:]...)
	}
	accessState.offset += int64(consumed)
	if size <= accessLogLimit {
		return
	}
	if err := os.Truncate(path, 0); err != nil {
		logger.Warning("truncate access log:", err)
		return
	}
	accessState.offset = 0
}

// NocOnline 是 NOC 接口里的一个入站及其当前连入来源。
type NocOnline struct {
	Id       int          `json:"id"`
	Remark   string       `json:"remark"`
	Port     int          `json:"port"`
	Protocol string       `json:"protocol"`
	Tag      string       `json:"tag"`
	Limit    int          `json:"limit"`
	Count    int          `json:"count"`
	Peers    []model.Peer `json:"peers"`
}

// NocSnapshotData 给外部 NOC 面板一次取走访问记录和在线来源。
type NocSnapshotData struct {
	Access []Visit     `json:"access"`
	Online []NocOnline `json:"online"`
}

// NocSnapshot 汇总访问记录和当前连入来源。
func NocSnapshot(inbounds []*model.Inbound) NocSnapshotData {
	ApplyConnCounts(inbounds)
	online := make([]NocOnline, 0, len(inbounds))
	for _, inbound := range inbounds {
		if inbound == nil {
			continue
		}
		peers := inbound.Peers
		if peers == nil {
			peers = []model.Peer{}
		}
		online = append(online, NocOnline{
			Id:       inbound.Id,
			Remark:   inbound.Remark,
			Port:     inbound.Port,
			Protocol: string(inbound.Protocol),
			Tag:      inbound.Tag,
			Limit:    inbound.LimitConn,
			Count:    inbound.ConnCount,
			Peers:    peers,
		})
	}
	access := RecentVisits(inbounds)
	if access == nil {
		access = []Visit{}
	}
	return NocSnapshotData{Access: access, Online: online}
}

func copyAccessRecords() []accesslog.Record {
	accessState.Lock()
	defer accessState.Unlock()
	return append([]accesslog.Record(nil), accessState.records...)
}

func userForIP(records []accesslog.Record, inbound *model.Inbound, ip string) string {
	if inbound == nil || ip == "" {
		return ""
	}
	for i := len(records) - 1; i >= 0; i-- {
		record := records[i]
		if record.InboundTag == inbound.Tag && record.Source == ip && record.Email != "" {
			return xraycfg.UserLabel(inbound.Id, inbound.Settings, record.Email)
		}
	}
	return ""
}

// RecentVisits 返回当前用户入站的最近访问，新的在前。
func RecentVisits(inbounds []*model.Inbound) []Visit {
	byTag := make(map[string]*model.Inbound, len(inbounds))
	for _, inbound := range inbounds {
		if inbound != nil && inbound.Tag != "" {
			byTag[inbound.Tag] = inbound
		}
	}
	records := copyAccessRecords()

	out := make([]Visit, 0, len(records))
	for i := len(records) - 1; i >= 0; i-- {
		record := records[i]
		inbound := byTag[record.InboundTag]
		if inbound == nil {
			continue
		}
		out = append(out, Visit{
			Time:    record.Time,
			Inbound: record.InboundTag,
			Remark:  inbound.Remark,
			User:    xraycfg.UserLabel(inbound.Id, inbound.Settings, record.Email),
			Source:  record.Source,
			Network: record.Network,
			Target:  record.Target,
		})
		if len(out) >= 200 {
			break
		}
	}
	return out
}
