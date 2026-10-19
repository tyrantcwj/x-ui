package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"x-ui/config"
	"x-ui/database"
	"x-ui/database/model"
	"x-ui/util/reality"
)

// NocFeatures 是 GET /info 公布的能力。没做出来的功能不要写在这里，NOC 会改走 SSH。
var NocFeatures = []string{"status", "inbound", "backup"}

var nocEnsureMu sync.Mutex

type NocService struct {
	inboundService InboundService
	serverService  ServerService
	xrayService    XrayService
}

type NocClient struct {
	ID    string `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
	Flow  string `json:"flow,omitempty"`
	Up    int64  `json:"up"`
	Down  int64  `json:"down"`
}

type NocInbound struct {
	ID           int         `json:"id"`
	Remark       string      `json:"remark"`
	Port         int         `json:"port"`
	Protocol     string      `json:"protocol"`
	Enable       bool        `json:"enable"`
	Up           int64       `json:"up"`
	Down         int64       `json:"down"`
	Total        int64       `json:"total"`
	ExpiryTime   int64       `json:"expiry_time"`
	TrafficScope string      `json:"traffic_scope"`
	Clients      []NocClient `json:"clients"`
}

type NocEnsureResult struct {
	ID       int    `json:"id"`
	Remark   string `json:"remark"`
	Port     int    `json:"port"`
	UUID     string `json:"uuid"`
	Pbk      string `json:"pbk"`
	Sid      string `json:"sid"`
	Sni      string `json:"sni"`
	Fp       string `json:"fp"`
	Existed  bool   `json:"existed"`
	Firewall string `json:"firewall,omitempty"`
	Msg      string `json:"msg,omitempty"`
}

type NocEnsureItem struct {
	Remark string `json:"remark"`
	Port   int    `json:"port"`
}

type NocStatus struct {
	PanelVersion string  `json:"panel_version"`
	Uptime       uint64  `json:"uptime"`
	Cpu          float64 `json:"cpu"`
	Mem          struct {
		Used  uint64 `json:"used"`
		Total uint64 `json:"total"`
	} `json:"mem"`
	Disk struct {
		Used  uint64 `json:"used"`
		Total uint64 `json:"total"`
	} `json:"disk"`
	NetIO struct {
		Up   uint64 `json:"up"`
		Down uint64 `json:"down"`
	} `json:"net_io"`
	NetTraffic struct {
		Sent uint64 `json:"sent"`
		Recv uint64 `json:"recv"`
	} `json:"net_traffic"`
	TcpCount int `json:"tcp_count"`
	UdpCount int `json:"udp_count"`
	Xray     struct {
		Running bool   `json:"running"`
		Version string `json:"version"`
		Error   string `json:"error"`
	} `json:"xray"`
}

func (s *NocService) Info() map[string]interface{} {
	return map[string]interface{}{
		"panel":    "x-ui",
		"version":  config.GetVersion(),
		"features": NocFeatures,
	}
}

// Status 连续采样两次，这样一次请求也能算出网卡速率。速率单位是字节/秒。
func (s *NocService) Status() *NocStatus {
	first := s.serverService.GetStatus(nil)
	time.Sleep(300 * time.Millisecond)
	st := s.serverService.GetStatus(first)
	out := &NocStatus{
		PanelVersion: config.GetVersion(),
		Uptime:       st.Uptime,
		Cpu:          st.Cpu,
		TcpCount:     st.TcpCount,
		UdpCount:     st.UdpCount,
	}
	out.Mem.Used = st.Mem.Current
	out.Mem.Total = st.Mem.Total
	out.Disk.Used = st.Disk.Current
	out.Disk.Total = st.Disk.Total
	out.NetIO.Up = st.NetIO.Up
	out.NetIO.Down = st.NetIO.Down
	out.NetTraffic.Sent = st.NetTraffic.Sent
	out.NetTraffic.Recv = st.NetTraffic.Recv
	out.Xray.Running = st.Xray.State == Running
	out.Xray.Version = st.Xray.Version
	out.Xray.Error = st.Xray.ErrorMsg
	return out
}

func (s *NocService) Inbounds() ([]*NocInbound, error) {
	inbounds, err := s.inboundService.GetAllInbounds()
	if err != nil {
		return nil, err
	}
	out := make([]*NocInbound, 0, len(inbounds))
	for _, in := range inbounds {
		item := &NocInbound{
			ID:           in.Id,
			Remark:       in.Remark,
			Port:         in.Port,
			Protocol:     string(in.Protocol),
			Enable:       in.Enable,
			Up:           in.Up,
			Down:         in.Down,
			Total:        in.Total,
			ExpiryTime:   in.ExpiryTime,
			TrafficScope: "inbound",
			Clients:      clientsOf(in),
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *NocService) Ensure(remark string, port int) (*NocEnsureResult, error) {
	nocEnsureMu.Lock()
	defer nocEnsureMu.Unlock()
	result, created, err := s.ensureLocked(remark, port)
	if err != nil {
		return nil, err
	}
	if err = s.applyXray(created); err != nil {
		return result, err
	}
	return result, nil
}

func (s *NocService) EnsureBatch(items []NocEnsureItem) ([]*NocEnsureResult, error) {
	nocEnsureMu.Lock()
	defer nocEnsureMu.Unlock()
	if len(items) == 0 {
		return nil, errors.New("items 不能为空")
	}
	out := make([]*NocEnsureResult, 0, len(items))
	var first error
	created := false
	for _, item := range items {
		result, made, err := s.ensureLocked(item.Remark, item.Port)
		if err != nil {
			if first == nil {
				first = fmt.Errorf("%s：%s", strings.TrimSpace(item.Remark), err.Error())
			}
			out = append(out, &NocEnsureResult{Remark: strings.TrimSpace(item.Remark), Msg: err.Error()})
			continue
		}
		if made {
			created = true
		}
		out = append(out, result)
	}
	if err := s.applyXray(created); err != nil && first == nil {
		first = err
	}
	return out, first
}

// DeleteInbound 按名称和端口删除。已经不存在时也算成功，方便 NOC 重试。
func (s *NocService) DeleteInbound(remark string, port int) (int, error) {
	remark = strings.TrimSpace(remark)
	if remark == "" || port <= 0 {
		return 0, errors.New("需要同时提供 remark 和 port")
	}
	nocEnsureMu.Lock()
	defer nocEnsureMu.Unlock()
	var inbound model.Inbound
	err := database.GetDB().Where("remark = ? AND port = ?", remark, port).First(&inbound).Error
	if database.IsNotFound(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if err = s.inboundService.DelInbound(inbound.Id); err != nil {
		return 0, err
	}
	if err = s.applyXray(true); err != nil {
		return 1, err
	}
	return 1, nil
}

func (s *NocService) ensureLocked(remark string, port int) (*NocEnsureResult, bool, error) {
	remark = strings.TrimSpace(remark)
	if remark == "" {
		return nil, false, errors.New("remark 不能为空")
	}
	if port < 0 || port > 65535 {
		return nil, false, errors.New("端口必须在 1 到 65535 之间")
	}
	var existing []model.Inbound
	if err := database.GetDB().Where("remark = ?", remark).Order("id").Find(&existing).Error; err != nil {
		return nil, false, err
	}
	if len(existing) > 0 {
		result := linkOf(&existing[0])
		result.Firewall = allowPort(result.Port)
		return result, false, nil
	}
	chosen, err := s.choosePort(port)
	if err != nil {
		return nil, false, err
	}
	material, err := reality.NewMaterial()
	if err != nil {
		return nil, false, fmt.Errorf("生成 reality 密钥失败：%s", err.Error())
	}
	var user model.User
	if err = database.GetDB().First(&user).Error; err != nil {
		return nil, false, errors.New("面板还没有管理员账号")
	}
	inbound := &model.Inbound{
		UserId:         user.Id,
		Remark:         remark,
		Enable:         true,
		Listen:         "",
		Port:           chosen,
		Protocol:       model.VLESS,
		Settings:       material.Settings,
		StreamSettings: material.Stream,
		Tag:            fmt.Sprintf("inbound-%d", chosen),
		Sniffing:       material.Sniffing,
	}
	if err = s.inboundService.AddInbound(inbound); err != nil {
		return nil, false, fmt.Errorf("保存入站失败：%s", err.Error())
	}
	return &NocEnsureResult{
		ID:       inbound.Id,
		Remark:   remark,
		Port:     chosen,
		UUID:     material.UUID,
		Pbk:      material.Public,
		Sid:      material.ShortID,
		Sni:      material.SNI,
		Fp:       material.FP,
		Existed:  false,
		Firewall: allowPort(chosen),
	}, true, nil
}

func (s *NocService) choosePort(port int) (int, error) {
	inbounds, err := s.inboundService.GetAllInbounds()
	if err != nil {
		return 0, err
	}
	used := map[int]bool{}
	for _, in := range inbounds {
		used[in.Port] = true
	}
	if port > 0 {
		if used[port] {
			return 0, fmt.Errorf("端口 %d 已被其他入站使用", port)
		}
		if !portFree(port) {
			return 0, fmt.Errorf("端口 %d 已被系统占用", port)
		}
		return port, nil
	}
	for i := 0; i < 40; i++ {
		p := 20000 + rand.Intn(30000)
		if used[p] || !portFree(p) {
			continue
		}
		return p, nil
	}
	return 0, errors.New("没有找到空闲端口")
}

func (s *NocService) applyXray(force bool) error {
	if !force && s.xrayService.IsXrayRunning() {
		return nil
	}
	if err := s.xrayService.RestartXray(force); err != nil {
		s.xrayService.SetToNeedRestart()
		return fmt.Errorf("入站已写入数据库，但 xray 重启失败：%s", err.Error())
	}
	return nil
}

func clientsOf(in *model.Inbound) []NocClient {
	var doc struct {
		Clients []struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Flow  string `json:"flow"`
		} `json:"clients"`
	}
	_ = json.Unmarshal([]byte(in.Settings), &doc)
	out := make([]NocClient, 0, len(doc.Clients))
	for _, c := range doc.Clients {
		out = append(out, NocClient{ID: c.ID, Email: c.Email, Flow: c.Flow})
	}
	if len(out) == 1 {
		out[0].Up = in.Up
		out[0].Down = in.Down
	}
	return out
}

func linkOf(in *model.Inbound) *NocEnsureResult {
	var settings struct {
		Clients []struct {
			ID string `json:"id"`
		} `json:"clients"`
	}
	_ = json.Unmarshal([]byte(in.Settings), &settings)
	var stream struct {
		Reality struct {
			PublicKey   string   `json:"publicKey"`
			Fingerprint string   `json:"fingerprint"`
			ServerNames []string `json:"serverNames"`
			ShortIds    []string `json:"shortIds"`
		} `json:"realitySettings"`
	}
	_ = json.Unmarshal([]byte(in.StreamSettings), &stream)
	result := &NocEnsureResult{
		ID:      in.Id,
		Remark:  in.Remark,
		Port:    in.Port,
		Pbk:     stream.Reality.PublicKey,
		Fp:      stream.Reality.Fingerprint,
		Existed: true,
	}
	if len(settings.Clients) > 0 {
		result.UUID = settings.Clients[0].ID
	}
	if len(stream.Reality.ServerNames) > 0 {
		result.Sni = stream.Reality.ServerNames[0]
	}
	if len(stream.Reality.ShortIds) > 0 {
		result.Sid = stream.Reality.ShortIds[0]
	}
	return result
}

func portFree(port int) bool {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// allowPort 在防火墙开着时放行端口。安装脚本默认会关掉防火墙，这时跳过，不把整次创建判失败。
func allowPort(port int) string {
	if runtime.GOOS != "linux" {
		return "非 Linux，未修改防火墙"
	}
	if ufwActive() {
		out, err := exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", port)).CombinedOutput()
		if err != nil {
			return "ufw 放行失败：" + strings.TrimSpace(string(out))
		}
		return "ufw 已放行"
	}
	if firewalldActive() {
		spec := fmt.Sprintf("%d/tcp", port)
		if out, err := exec.Command("firewall-cmd", "--add-port="+spec).CombinedOutput(); err != nil {
			return "firewalld 放行失败：" + strings.TrimSpace(string(out))
		}
		_, _ = exec.Command("firewall-cmd", "--permanent", "--add-port="+spec).CombinedOutput()
		return "firewalld 已放行"
	}
	return "未检测到启用的防火墙，跳过"
}

func ufwActive() bool {
	if _, err := exec.LookPath("ufw"); err != nil {
		return false
	}
	out, err := exec.Command("ufw", "status").CombinedOutput()
	if err != nil {
		return false
	}
	text := string(out)
	return strings.Contains(text, "Status: active") || strings.Contains(text, "状态：激活") || strings.Contains(text, "状态: 激活")
}

func firewalldActive() bool {
	if _, err := exec.LookPath("firewall-cmd"); err != nil {
		return false
	}
	return exec.Command("firewall-cmd", "--state").Run() == nil
}
