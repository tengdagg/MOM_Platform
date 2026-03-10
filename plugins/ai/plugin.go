package ai

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ydcloud-dy/mom/internal/plugin"
	"github.com/ydcloud-dy/mom/plugins/ai/biz"
	"github.com/ydcloud-dy/mom/plugins/ai/server"
)

// Plugin AI 助手插件实现
type Plugin struct {
	db        *gorm.DB
	name      string
	ctx       context.Context
	cancelCtx context.CancelFunc
}

// New 创建 AI 插件实例
func New() *Plugin {
	return &Plugin{
		name: "ai",
	}
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "ai"
}

// Description 返回插件描述
func (p *Plugin) Description() string {
	return "AI 智能助手插件，提供 Agent + Skills 运维管理能力"
}

// Version 返回插件版本
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Author 返回插件作者
func (p *Plugin) Author() string {
	return "dat"
}

// Enable 启用插件
func (p *Plugin) Enable(db *gorm.DB) error {
	p.db = db

	// 自动迁移所有 AI 相关表
	models := []interface{}{
		&biz.AIModelConfig{},
		&biz.ChatSession{},
		&biz.ChatMessage{},
		&biz.SkillDefinition{},
		&biz.AIChannelConfig{},
		&biz.AIChannelBinding{},
		&biz.AIChannelInboundMessage{},
	}

	for _, m := range models {
		if err := db.AutoMigrate(m); err != nil {
			return err
		}
	}

	p.ctx, p.cancelCtx = context.WithCancel(context.Background())

	return nil
}

// Disable 禁用插件
func (p *Plugin) Disable(db *gorm.DB) error {
	if p.cancelCtx != nil {
		p.cancelCtx()
	}
	return nil
}

// RegisterRoutes 注册路由
func (p *Plugin) RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
	server.RegisterRoutes(router, db, p.ctx)
}

// GetMenus 获取插件菜单配置
func (p *Plugin) GetMenus() []plugin.MenuConfig {
	return []plugin.MenuConfig{
		{
			Name:   "AI 助手",
			Path:   "/ai",
			Icon:   "ChatDotRound",
			Sort:   5,
			Hidden: false,
		},
		{
			Name:       "MOM Claw",
			Path:       "/ai/chat",
			Icon:       "ChatLineRound",
			Sort:       1,
			Hidden:     false,
			ParentPath: "/ai",
		},
		{
			Name:       "Skill 管理",
			Path:       "/ai/skills",
			Icon:       "MagicStick",
			Sort:       2,
			Hidden:     false,
			ParentPath: "/ai",
		},
		{
			Name:       "模型配置",
			Path:       "/ai/models",
			Icon:       "Setting",
			Sort:       3,
			Hidden:     false,
			ParentPath: "/ai",
		},
		{
			Name:       "聊天渠道",
			Path:       "/ai/channels",
			Icon:       "Connection",
			Sort:       4,
			Hidden:     false,
			ParentPath: "/ai",
		},
	}
}
