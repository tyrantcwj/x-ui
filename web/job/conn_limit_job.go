package job

import "x-ui/web/service"

// ConnLimitJob 统计入站当前连入的来源 IP，并执行连入上限。
type ConnLimitJob struct{}

func NewConnLimitJob() *ConnLimitJob {
	return &ConnLimitJob{}
}

func (j *ConnLimitJob) Run() {
	service.RefreshConnLimits()
}
