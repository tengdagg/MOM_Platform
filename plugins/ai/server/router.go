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

	// 从 SKILL.md 加载并注册所有内置 Skills
	skills.RegisterAllBuiltinSkills(registry)

	// 加载自定义 Skills（从数据库）
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
		skillGroup := ai.Group("/skills")
		{
			skillGroup.GET("", handler.ListSkills)
			skillGroup.GET("/stats", handler.GetSkillStats)
			skillGroup.PUT("/:id/toggle", handler.ToggleSkill)
			skillGroup.PUT("/toggle-builtin", handler.ToggleBuiltinSkill)
			skillGroup.POST("/upload", handler.UploadSkill)
			skillGroup.DELETE("/:id", handler.DeleteSkill)
		}

		// 对话模板
		ai.GET("/templates", handler.ListTemplates)
	}
}
