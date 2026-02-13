package server

import (
	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/mom/plugins/ai/biz"
	"github.com/ydcloud-dy/mom/plugins/ai/skills"
	"gorm.io/gorm"
)

// RegisterRoutes 注册 AI 插件路由
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	registry := biz.NewToolRegistry()

	// 注册内置 Skills
	skills.RegisterHostSkills(registry)
	skills.RegisterK8sSkills(registry)
	skills.RegisterAuditSkills(registry)
	skills.RegisterTaskSkills(registry)
	skills.RegisterMonitorSkills(registry)
	skills.RegisterCloudSkills(registry)
	skills.RegisterAnalysisSkills(registry)

	// 加载自定义 Skills
	skillEngine := biz.NewSkillEngine(db, registry)
	skillEngine.LoadCustomSkills()

	agent := biz.NewAgent(db, registry)
	convMgr := biz.NewConversationManager(db)

	handler := &Handler{
		db:       db,
		agent:    agent,
		registry: registry,
		convMgr:  convMgr,
	}

	ai := router.Group("/ai")
	{
		// 模型配置
		models := ai.Group("/models")
		{
			models.GET("", handler.ListModels)
			models.POST("", handler.CreateModel)
			models.PUT("/:id", handler.UpdateModel)
			models.DELETE("/:id", handler.DeleteModel)
			models.POST("/:id/test", handler.TestModel)
			models.PUT("/:id/default", handler.SetDefaultModel)
		}

		// 对话管理
		chat := ai.Group("/chat")
		{
			chat.GET("/ws", handler.ChatWebSocket)
			chat.POST("/sessions", handler.CreateSession)
			chat.GET("/sessions", handler.ListSessions)
			chat.DELETE("/sessions/:id", handler.DeleteSession)
			chat.GET("/sessions/:id/messages", handler.GetMessages)
			chat.POST("/send", handler.SendMessage)
		}

		// Skill 管理
		skills := ai.Group("/skills")
		{
			skills.GET("", handler.ListSkills)
			skills.PUT("/:id/toggle", handler.ToggleSkill)
			skills.POST("/upload", handler.UploadSkill)
			skills.DELETE("/:id", handler.DeleteSkill)
		}

		// 对话模板
		ai.GET("/templates", handler.ListTemplates)
	}
}
