package controller

import (
	"crypto/subtle"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"x-ui/config"
	"x-ui/database"
	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

type NocV1Controller struct {
	nocService     service.NocService
	settingService service.SettingService
}

type nocBody struct {
	Ok   bool        `json:"ok"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type ensureBody struct {
	Remark string `json:"remark"`
	Port   int    `json:"port"`
}

type ensureBatchBody struct {
	Items   []service.NocEnsureItem `json:"items"`
	Remarks []string                `json:"remarks"`
}

func NewNocV1Controller(g *gin.RouterGroup) *NocV1Controller {
	a := &NocV1Controller{}
	g = g.Group("/api/noc/v1")
	g.Use(a.auth)
	g.GET("/info", a.info)
	g.GET("/status", a.status)
	g.GET("/inbounds", a.inbounds)
	g.POST("/inbounds/ensure", a.ensure)
	g.POST("/inbounds/ensure-batch", a.ensureBatch)
	g.DELETE("/inbounds", a.deleteInbound)
	g.GET("/backup", a.backup)
	return a
}

func (a *NocV1Controller) auth(c *gin.Context) {
	expect, err := a.settingService.GetApiToken()
	got := bearerToken(c)
	if err != nil || !sameToken(got, expect) {
		nocFail(c, http.StatusUnauthorized, "令牌无效或已吊销")
		c.Abort()
		return
	}
	c.Next()
}

func (a *NocV1Controller) info(c *gin.Context) {
	nocOK(c, "", a.nocService.Info())
}

func (a *NocV1Controller) status(c *gin.Context) {
	nocOK(c, "", a.nocService.Status())
}

func (a *NocV1Controller) inbounds(c *gin.Context) {
	inbounds, err := a.nocService.Inbounds()
	if err != nil {
		nocFail(c, http.StatusInternalServerError, "读取入站失败："+err.Error())
		return
	}
	nocOK(c, "", inbounds)
}

func (a *NocV1Controller) ensure(c *gin.Context) {
	body := &ensureBody{}
	if err := c.ShouldBindJSON(body); err != nil {
		nocFail(c, http.StatusBadRequest, "请求体不是 JSON")
		return
	}
	result, err := a.nocService.Ensure(body.Remark, body.Port)
	if err != nil {
		data := interface{}(nil)
		if result != nil {
			data = result
		}
		nocFailData(c, ensureHTTPStatus(err, result != nil), err.Error(), data)
		return
	}
	msg := "已创建"
	if result.Existed {
		msg = "已存在，返回原来的入站"
	}
	if strings.Contains(result.Firewall, "失败") {
		msg += "，" + result.Firewall
	}
	nocOK(c, msg, result)
}

func (a *NocV1Controller) ensureBatch(c *gin.Context) {
	body := &ensureBatchBody{}
	if err := c.ShouldBindJSON(body); err != nil {
		nocFail(c, http.StatusBadRequest, "请求体不是 JSON")
		return
	}
	items := body.Items
	for _, remark := range body.Remarks {
		items = append(items, service.NocEnsureItem{Remark: remark})
	}
	results, err := a.nocService.EnsureBatch(items)
	if err != nil {
		status := http.StatusConflict
		if strings.Contains(err.Error(), "重启失败") {
			status = http.StatusInternalServerError
		} else if len(results) == 0 {
			status = ensureHTTPStatus(err, false)
		}
		c.JSON(status, nocBody{Ok: false, Msg: err.Error(), Data: gin.H{"items": results}})
		return
	}
	nocOK(c, "已处理", gin.H{"items": results})
}

func (a *NocV1Controller) deleteInbound(c *gin.Context) {
	port, err := strconv.Atoi(c.Query("port"))
	if err != nil {
		nocFail(c, http.StatusBadRequest, "需要同时提供 remark 和 port")
		return
	}
	n, err := a.nocService.DeleteInbound(c.Query("remark"), port)
	if err != nil {
		status := http.StatusBadRequest
		if !strings.Contains(err.Error(), "需要同时提供") {
			status = http.StatusInternalServerError
		}
		nocFail(c, status, err.Error(), gin.H{"deleted": n})
		return
	}
	msg := "已删除"
	if n == 0 {
		msg = "入站不存在"
	}
	nocOK(c, msg, gin.H{"deleted": n})
}

func (a *NocV1Controller) backup(c *gin.Context) {
	dir := filepath.Dir(config.GetDBPath())
	f, err := os.CreateTemp(dir, "x-ui-backup-*.db")
	if err != nil {
		nocFail(c, http.StatusInternalServerError, "创建备份文件失败："+err.Error())
		return
	}
	name := f.Name()
	_ = f.Close()
	defer os.Remove(name)
	if err = database.Backup(name); err != nil {
		nocFail(c, http.StatusInternalServerError, "备份数据库失败："+err.Error())
		return
	}
	c.FileAttachment(name, "x-ui.db")
}

func ensureHTTPStatus(err error, saved bool) int {
	if saved || strings.Contains(err.Error(), "重启失败") || strings.Contains(err.Error(), "保存入站失败") || strings.Contains(err.Error(), "密钥失败") {
		return http.StatusInternalServerError
	}
	if strings.Contains(err.Error(), "已被") {
		return http.StatusConflict
	}
	if strings.Contains(err.Error(), "不能为空") || strings.Contains(err.Error(), "必须在") || strings.Contains(err.Error(), "没有找到空闲") {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func bearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if len(h) < 7 || !strings.EqualFold(h[:7], "bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}

func sameToken(got, expect string) bool {
	if expect == "" || len(got) != len(expect) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expect)) == 1
}

func nocOK(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, nocBody{Ok: true, Msg: msg, Data: data})
}

func nocFail(c *gin.Context, status int, msg string) {
	nocFailData(c, status, msg, nil)
}

func nocFailData(c *gin.Context, status int, msg string, data interface{}) {
	c.JSON(status, nocBody{Ok: false, Msg: msg, Data: data})
}
