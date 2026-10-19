package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"x-ui/config"
	"x-ui/logger"
	"x-ui/util/json_util"
	"x-ui/xray"

	"go.uber.org/atomic"
)

var p *xray.Process
var lock sync.Mutex
var isNeedXrayRestart atomic.Bool
var result string

type XrayService struct {
	inboundService InboundService
	settingService SettingService
}

func (s *XrayService) IsXrayRunning() bool {
	return p != nil && p.IsRunning()
}

func (s *XrayService) GetXrayErr() error {
	if p == nil {
		return nil
	}
	return p.GetErr()
}

func (s *XrayService) GetXrayResult() string {
	if result != "" {
		return result
	}
	if s.IsXrayRunning() {
		return ""
	}
	if p == nil {
		return ""
	}
	result = p.GetResult()
	return result
}

func (s *XrayService) GetXrayVersion() string {
	if p == nil {
		return "Unknown"
	}
	return p.GetVersion()
}

func (s *XrayService) GetXrayConfig() (*xray.Config, error) {
	templateConfig, err := s.settingService.GetXrayConfigTemplate()
	if err != nil {
		return nil, err
	}

	xrayConfig := &xray.Config{}
	err = json.Unmarshal([]byte(templateConfig), xrayConfig)
	if err != nil {
		return nil, err
	}
	if err = ensureAccessLog(xrayConfig); err != nil {
		return nil, err
	}
	if err = ensureDirectOutbound(xrayConfig); err != nil {
		return nil, err
	}

	inbounds, err := s.inboundService.GetAllInbounds()
	if err != nil {
		return nil, err
	}
	for _, inbound := range inbounds {
		if !inbound.Enable {
			continue
		}
		inboundConfig := inbound.GenXrayInboundConfig()
		xrayConfig.InboundConfigs = append(xrayConfig.InboundConfigs, *inboundConfig)
	}
	return xrayConfig, nil
}

func ensureAccessLog(cfg *xray.Config) error {
	if err := os.MkdirAll(filepath.Dir(config.GetAccessLogPath()), 0755); err != nil {
		return err
	}
	logCfg := map[string]interface{}{}
	if len(cfg.LogConfig) > 0 && string(cfg.LogConfig) != "null" {
		if err := json.Unmarshal(cfg.LogConfig, &logCfg); err != nil {
			return err
		}
	}
	logCfg["access"] = config.GetAccessLogPath()
	encoded, err := json.Marshal(logCfg)
	if err != nil {
		return err
	}
	cfg.LogConfig = json_util.RawMessage(encoded)
	return nil
}

// ensureDirectOutbound 给默认出站补上 tag。否则访问日志里没有入站名称，无法对应到用户。
func ensureDirectOutbound(cfg *xray.Config) error {
	if len(cfg.OutboundConfigs) == 0 || string(cfg.OutboundConfigs) == "null" {
		return nil
	}
	var outbounds []map[string]interface{}
	if err := json.Unmarshal(cfg.OutboundConfigs, &outbounds); err != nil || len(outbounds) == 0 {
		return err
	}
	tag, _ := outbounds[0]["tag"].(string)
	if tag != "" {
		return nil
	}
	outbounds[0]["tag"] = "direct"
	encoded, err := json.Marshal(outbounds)
	if err != nil {
		return err
	}
	cfg.OutboundConfigs = json_util.RawMessage(encoded)
	return nil
}

func (s *XrayService) GetXrayTraffic() ([]*xray.Traffic, error) {
	if !s.IsXrayRunning() {
		return nil, errors.New("xray is not running")
	}
	return p.GetTraffic(true)
}

func (s *XrayService) RestartXray(isForce bool) error {
	lock.Lock()
	defer lock.Unlock()
	logger.Debug("restart xray, force:", isForce)

	xrayConfig, err := s.GetXrayConfig()
	if err != nil {
		return err
	}

	if p != nil && p.IsRunning() {
		if !isForce && p.GetConfig().Equals(xrayConfig) {
			logger.Debug("not need to restart xray")
			return nil
		}
		p.Stop()
	}

	p = xray.NewProcess(xrayConfig)
	result = ""
	return p.Start()
}

func (s *XrayService) StopXray() error {
	lock.Lock()
	defer lock.Unlock()
	logger.Debug("stop xray")
	if s.IsXrayRunning() {
		return p.Stop()
	}
	return errors.New("xray is not running")
}

func (s *XrayService) SetToNeedRestart() {
	isNeedXrayRestart.Store(true)
}

func (s *XrayService) IsNeedRestartAndSetFalse() bool {
	return isNeedXrayRestart.CompareAndSwap(true, false)
}
