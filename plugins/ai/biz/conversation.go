package biz

import (
	"fmt"

	"gorm.io/gorm"
)

// ConversationManager 对话管理
type ConversationManager struct {
	db *gorm.DB
}

// NewConversationManager 创建对话管理器
func NewConversationManager(db *gorm.DB) *ConversationManager {
	return &ConversationManager{db: db}
}

// CreateSession 创建新会话
func (m *ConversationManager) CreateSession(userID uint, username string, title string, modelID uint) (*ChatSession, error) {
	if title == "" {
		title = "新对话"
	}
	session := &ChatSession{
		UserID:   userID,
		Username: username,
		Title:    title,
		ModelID:  modelID,
	}
	if err := m.db.Create(session).Error; err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}
	return session, nil
}

// GetSession 获取会话
func (m *ConversationManager) GetSession(sessionID uint, userID uint) (*ChatSession, error) {
	var session ChatSession
	if err := m.db.Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error; err != nil {
		return nil, fmt.Errorf("会话不存在")
	}
	return &session, nil
}

// ListSessions 获取用户的会话列表
func (m *ConversationManager) ListSessions(userID uint) ([]ChatSession, error) {
	var sessions []ChatSession
	if err := m.db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// DeleteSession 删除会话
func (m *ConversationManager) DeleteSession(sessionID uint, userID uint) error {
	// 先删除会话消息
	if err := m.db.Where("session_id = ?", sessionID).Delete(&ChatMessage{}).Error; err != nil {
		return fmt.Errorf("删除会话消息失败: %w", err)
	}
	// 再删除会话
	result := m.db.Where("id = ? AND user_id = ?", sessionID, userID).Delete(&ChatSession{})
	if result.Error != nil {
		return fmt.Errorf("删除会话失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("会话不存在")
	}
	return nil
}

// UpdateSessionTitle 更新会话标题
func (m *ConversationManager) UpdateSessionTitle(sessionID uint, userID uint, title string) error {
	return m.db.Model(&ChatSession{}).Where("id = ? AND user_id = ?", sessionID, userID).Update("title", title).Error
}

// AddMessage 添加消息
func (m *ConversationManager) AddMessage(sessionID uint, role string, content string) (*ChatMessage, error) {
	msg := &ChatMessage{
		SessionID: sessionID,
		Role:      role,
		Content:   content,
	}
	if err := m.db.Create(msg).Error; err != nil {
		return nil, fmt.Errorf("保存消息失败: %w", err)
	}
	// 更新会话时间
	m.db.Model(&ChatSession{}).Where("id = ?", sessionID).Update("updated_at", msg.CreatedAt)
	return msg, nil
}

// GetMessages 获取会话消息
func (m *ConversationManager) GetMessages(sessionID uint) ([]ChatMessage, error) {
	var messages []ChatMessage
	if err := m.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// GetRecentMessages 获取最近 N 条消息（用于构建上下文）
func (m *ConversationManager) GetRecentMessages(sessionID uint, limit int) ([]ChatMessage, error) {
	var messages []ChatMessage
	if err := m.db.Where("session_id = ?", sessionID).Order("created_at DESC").Limit(limit).Find(&messages).Error; err != nil {
		return nil, err
	}
	// 反转为正序
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

// AutoTitleFromFirstMessage 根据第一条消息自动生成标题
func (m *ConversationManager) AutoTitleFromFirstMessage(sessionID uint, content string) {
	title := content
	if len([]rune(title)) > 30 {
		title = string([]rune(title)[:30]) + "..."
	}
	m.db.Model(&ChatSession{}).Where("id = ? AND title = ?", sessionID, "新对话").Update("title", title)
}
