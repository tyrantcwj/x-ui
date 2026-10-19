package job

import "x-ui/web/service"

// AccessLogJob 读取 Xray 访问日志，供面板显示最近访问的域名。
type AccessLogJob struct{}

func NewAccessLogJob() *AccessLogJob {
	return &AccessLogJob{}
}

func (j *AccessLogJob) Run() {
	service.PollAccessLog()
}
