package biz

import (
	"time"

	"gorm.io/gorm"
)

// AIModelConfig AI 模型配置
type AIModelConfig struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Provider    string         `gorm:"size:50;not null" json:"provider"` // openai / ollama / custom
	BaseURL     string         `gorm:"size:500" json:"baseUrl"`
	APIKey      string         `gorm:"size:500" json:"-"`         // 加密存储，不返回给前端
	APIKeySet   bool           `gorm:"-" json:"apiKeySet"`        // 前端判断是否已设置
	ModelName   string         `gorm:"size:100" json:"modelName"` // gpt-4o / qwen-plus / llama3
	MaxTokens   int            `gorm:"default:4096" json:"maxTokens"`
	Temperature float64        `gorm:"type:decimal(3,2);default:0.70" json:"temperature"`
	IsDefault   bool           `gorm:"default:false" json:"isDefault"`
	Status      int            `gorm:"default:1" json:"status"` // 1=启用 0=禁用
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AIModelConfig) TableName() string {
	return "ai_model_configs"
}

// ChatSession 对话会话
type ChatSession struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	UserID        uint           `gorm:"index;not null" json:"userId"`
	Username      string         `gorm:"size:100" json:"username"`
	Title         string         `gorm:"size:500;default:'新对话'" json:"title"`
	ModelID       uint           `gorm:"default:0" json:"modelId"`
	Summary       string         `gorm:"type:text" json:"summary,omitempty"`       // 历史消息摘要
	SummaryUpToID uint           `gorm:"default:0" json:"summaryUpToId,omitempty"` // 摘要覆盖到的最后一条消息 ID
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ChatSession) TableName() string {
	return "ai_chat_sessions"
}

// ChatMessage 对话消息
type ChatMessage struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SessionID  uint      `gorm:"index;not null" json:"sessionId"`
	Role       string    `gorm:"size:20;not null" json:"role"` // user / assistant / system / tool
	Content    string    `gorm:"type:longtext" json:"content"`
	ToolCalls  string    `gorm:"type:text" json:"toolCalls,omitempty"`  // JSON: 工具调用请求
	ToolResult string    `gorm:"type:text" json:"toolResult,omitempty"` // JSON: 工具调用结果
	TokensUsed int       `gorm:"default:0" json:"tokensUsed"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (ChatMessage) TableName() string {
	return "ai_chat_messages"
}

// SkillDefinition Skill 技能定义
// 每个 Skill 遵循标准目录结构:
//
//	skill-name/
//	├── SKILL.md          (必需) YAML 前置元数据 + Markdown 指令
//	└── 打包资源           (可选)
//	    ├── scripts/      可执行代码
//	    ├── references/   上下文文档
//	    └── assets/       输出文件（模板等）
type SkillDefinition struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100;uniqueIndex" json:"name"`
	DisplayName string    `gorm:"size:200" json:"displayName"`
	Description string    `gorm:"size:1000" json:"description"`
	Category    string    `gorm:"size:50" json:"category"`     // host / device / k8s / task / monitor / cloud / audit / analysis
	Parameters  string    `gorm:"type:text" json:"parameters"` // JSON Schema
	IsBuiltin   bool      `gorm:"default:false" json:"isBuiltin"`
	ScriptType  string    `gorm:"size:20;default:'builtin'" json:"scriptType"` // builtin / javascript / python
	ScriptBody  string    `gorm:"type:longtext" json:"scriptBody,omitempty"`
	Markdown    string    `gorm:"type:longtext" json:"markdown,omitempty"` // SKILL.md 中的 Markdown 指令部分
	IsEnabled   bool      `gorm:"default:true" json:"isEnabled"`
	RiskLevel   string    `gorm:"size:20;default:'low'" json:"riskLevel"` // low / medium / high / critical
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (SkillDefinition) TableName() string {
	return "ai_skill_definitions"
}

// PendingAction 待确认操作
type PendingAction struct {
	ID          string    `json:"id"`
	SessionID   uint      `json:"sessionId"`
	SkillName   string    `json:"skillName"`
	Description string    `json:"description"`
	Params      string    `json:"params"` // JSON
	RiskLevel   string    `json:"riskLevel"`
	Status      string    `json:"status"` // pending / confirmed / rejected / expired
	CreatedAt   time.Time `json:"createdAt"`
}
