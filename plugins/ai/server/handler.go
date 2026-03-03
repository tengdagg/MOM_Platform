package server

import (
	"time"

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

// GetModelCallStats 获取最近 7 天的模型调用统计（按模型分组、按天分组）
func (h *Handler) GetModelCallStats(c *gin.Context) {
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -6)
	startOfDay := time.Date(sevenDaysAgo.Year(), sevenDaysAgo.Month(), sevenDaysAgo.Day(), 0, 0, 0, 0, now.Location())

	// 按天 + 模型分组统计 assistant 消息数（每条 assistant 消息 = 一次模型调用）
	type DayModelCount struct {
		Day       string `json:"day"`
		ModelID   uint   `json:"modelId"`
		ModelName string `json:"modelName"`
		Count     int64  `json:"count"`
	}
	var rows []DayModelCount
	h.db.Raw(`
		SELECT
			DATE_FORMAT(m.created_at, '%Y-%m-%d') AS day,
			COALESCE(s.model_id, 0)               AS model_id,
			COALESCE(mc.name, '默认模型')          AS model_name,
			COUNT(*)                               AS count
		FROM ai_chat_messages m
		JOIN ai_chat_sessions s  ON s.id = m.session_id AND s.deleted_at IS NULL
		LEFT JOIN ai_model_configs mc ON mc.id = s.model_id AND mc.deleted_at IS NULL
		WHERE m.role = 'assistant'
		  AND m.created_at >= ?
		GROUP BY day, model_id, model_name
		ORDER BY day, model_name
	`, startOfDay).Scan(&rows)

	// 今日总调用量
	var todayTotal int64
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	h.db.Raw(`
		SELECT COUNT(*) FROM ai_chat_messages m
		JOIN ai_chat_sessions s ON s.id = m.session_id AND s.deleted_at IS NULL
		WHERE m.role = 'assistant' AND m.created_at >= ?
	`, todayStart).Scan(&todayTotal)

	// 总调用量
	var total int64
	h.db.Raw(`
		SELECT COUNT(*) FROM ai_chat_messages m
		JOIN ai_chat_sessions s ON s.id = m.session_id AND s.deleted_at IS NULL
		WHERE m.role = 'assistant'
	`).Scan(&total)

	response.Success(c, gin.H{
		"daily":      rows,
		"todayTotal": todayTotal,
		"total":      total,
	})
}
