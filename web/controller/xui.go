package controller

import (
	"crypto/subtle"
	"net/http"
	"x-ui/web/entity"
	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

type XUIController struct {
	BaseController

	inboundController *InboundController
	settingController *SettingController
}

func NewXUIController(g *gin.RouterGroup) *XUIController {
	a := &XUIController{}
	a.initRouter(g)
	return a
}

func (a *XUIController) initRouter(g *gin.RouterGroup) {
	g.GET("/api/noc", a.noc)
	g.POST("/api/noc", a.noc)
	NewNocV1Controller(g)

	g = g.Group("/xui")
	g.Use(a.checkLogin)

	g.GET("/", a.index)
	g.GET("/inbounds", a.inbounds)
	g.GET("/access", a.access)
	g.GET("/setting", a.setting)

	a.inboundController = NewInboundController(g)
	a.settingController = NewSettingController(g)
}

func (a *XUIController) noc(c *gin.Context) {
	token := c.GetHeader("X-Api-Token")
	if token == "" {
		token = c.Query("token")
	}
	expect, err := (&service.SettingService{}).GetApiToken()
	if err != nil || token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(expect)) != 1 {
		c.JSON(http.StatusUnauthorized, entity.Msg{Success: false, Msg: "token 无效"})
		return
	}
	inbounds, err := (&service.InboundService{}).GetAllInbounds()
	if err != nil {
		jsonMsg(c, "获取", err)
		return
	}
	jsonObj(c, service.NocSnapshot(inbounds), nil)
}

func (a *XUIController) index(c *gin.Context) {
	html(c, "index.html", "系统状态", nil)
}

func (a *XUIController) inbounds(c *gin.Context) {
	html(c, "inbounds.html", "入站列表", nil)
}

func (a *XUIController) access(c *gin.Context) {
	html(c, "access.html", "访问记录", nil)
}

func (a *XUIController) setting(c *gin.Context) {
	html(c, "setting.html", "设置", nil)
}
