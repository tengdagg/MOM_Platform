package server

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/mom/plugins/ai/biz"
	"github.com/ydcloud-dy/mom/plugins/ai/skills"
	"gorm.io/gorm"
)

// RegisterRoutes 注册 AI 插件路由
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB, rootCtx context.Context) {
	registry := biz.NewToolRegistry()

	// 从 SKILL.md 加载并注册所有内置 Skills
	skills.RegisterAllBuiltinSkills(registry)
	if err := skills.SyncBuiltinSkillDefinitions(db, registry); err != nil {
		panic("同步内置 Skills 失败: " + err.Error())
	}

	// 加载自定义 Skills（从数据库）
	skillEngine := biz.NewSkillEngine(db, registry)
	skillEngine.LoadCustomSkills()

	agent := biz.NewAgent(db, registry)
	convMgr := biz.NewConversationManager(db)
	convMgr.EnsureSettingsTable()
	convMgr.StartSessionCleanupScheduler()
	channelSvc := biz.NewChannelService(db)
	channelSvc.StartInboundMessageCleanupScheduler(rootCtx)
	channelRuntime := biz.NewChannelRuntimeManager(rootCtx, db, agent, convMgr, channelSvc)
	channelRuntime.StartEnabledChannels()

	handler := &Handler{
		db:                 db,
		agent:              agent,
		registry:           registry,
		convMgr:            convMgr,
		channelSvc:         channelSvc,
		channelRuntime:     channelRuntime,
		channelTestTimeout: 8 * time.Second,
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
			chat.GET("/settings/retention", handler.GetRetentionSettings)
			chat.PUT("/settings/retention", handler.SetRetentionSettings)
			chat.POST("/cleanup", handler.CleanupSessions)
		}

		// 聊天渠道
		channels := ai.Group("/channels")
		{
			channels.GET("", handler.ListChannels)
			channels.POST("", handler.CreateChannel)
			channels.GET("/:id", handler.GetChannel)
			channels.PUT("/:id", handler.UpdateChannel)
			channels.DELETE("/:id", handler.DeleteChannel)
			channels.POST("/:id/test", handler.TestChannel)
			channels.POST("/:id/start", handler.StartChannel)
			channels.POST("/:id/stop", handler.StopChannel)
			channels.POST("/:id/reconnect", handler.ReconnectChannel)
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

		// 统计
		ai.GET("/stats/model-calls", handler.GetModelCallStats)

		// 对话模板
		ai.GET("/templates", handler.ListTemplates)
	}
}
