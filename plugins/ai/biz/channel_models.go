package biz

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	ChannelTypeFeishu   = "feishu"
	ChannelTypeWeCom    = "wecom"
	ChannelTypeDingTalk = "dingtalk"
)

const (
	ChannelStatusStopped = "stopped"
	ChannelStatusRunning = "running"
	ChannelStatusError   = "error"
)

// AIChannelConfig 外部聊天渠道配置
type AIChannelConfig struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Name            string         `gorm:"size:100;not null" json:"name"`
	ChannelType     string         `gorm:"size:20;index;not null" json:"channelType"`
	Enabled         bool           `gorm:"default:false" json:"enabled"`
	Status          string         `gorm:"size:20;default:'stopped'" json:"status"`
	AppID           string         `gorm:"size:200" json:"appId"`
	AppSecret       string         `gorm:"size:500" json:"-"`
	AppSecretSet    bool           `gorm:"-" json:"appSecretSet"`
	DefaultModelID  uint           `gorm:"default:0" json:"defaultModelId"`
	ExecuteAsUserID uint           `gorm:"default:0;index" json:"executeAsUserId"`
	ConfigJSON      datatypes.JSON `gorm:"type:json" json:"config,omitempty"`
	LastError       string         `gorm:"type:text" json:"lastError,omitempty"`
	LastConnectedAt *time.Time     `json:"lastConnectedAt,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AIChannelConfig) TableName() string {
	return "ai_channel_configs"
}

// AIChannelBinding 外部渠道会话与内部 AI 会话绑定
type AIChannelBinding struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	ChannelType     string         `gorm:"size:20;index;not null" json:"channelType"`
	ChannelConfigID uint           `gorm:"index;not null" json:"channelConfigId"`
	ExternalUserID  string         `gorm:"size:200;index" json:"externalUserId"`
	ExternalOpenID  string         `gorm:"size:200;index" json:"externalOpenId"`
	ExternalUnionID string         `gorm:"size:200;index" json:"externalUnionId"`
	ExternalChatID  string         `gorm:"size:200;index" json:"externalChatId"`
	SessionID       uint           `gorm:"index;not null" json:"sessionId"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AIChannelBinding) TableName() string {
	return "ai_channel_bindings"
}

// AIChannelInboundMessage 外部渠道入站消息幂等记录
type AIChannelInboundMessage struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	ChannelType       string         `gorm:"size:20;index;not null" json:"channelType"`
	ChannelConfigID   uint           `gorm:"index;not null" json:"channelConfigId"`
	ExternalMessageID string         `gorm:"size:200;uniqueIndex:uk_channel_message;not null" json:"externalMessageId"`
	ExternalChatID    string         `gorm:"size:200;index" json:"externalChatId"`
	ExternalOpenID    string         `gorm:"size:200;index" json:"externalOpenId"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AIChannelInboundMessage) TableName() string {
	return "ai_channel_inbound_messages"
}
