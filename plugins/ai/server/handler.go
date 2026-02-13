package server

import (
	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/mom/pkg/response"
	"github.com/ydcloud-dy/mom/plugins/ai/biz"
	"gorm.io/gorm"
)

// Handler AI API 处理器
type Handler struct {
	db       *gorm.DB
	agent    *biz.Agent
	registry *biz.ToolRegistry
	convMgr  *biz.ConversationManager
}

// ListTemplates 获取对话模板
func (h *Handler) ListTemplates(c *gin.Context) {
	templates := biz.GetDefaultTemplates()
	response.Success(c, templates)
}
