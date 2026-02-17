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

// AddMessageWithTools 添加带工具调用记录的消息
func (m *ConversationManager) AddMessageWithTools(sessionID uint, role string, content string, toolCallsJSON string) (*ChatMessage, error) {
	msg := &ChatMessage{
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		ToolCalls: toolCallsJSON,
	}
	if err := m.db.Create(msg).Error; err != nil {
		return nil, fmt.Errorf("保存消息失败: %w", err)
	}
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

// CleanupOldSessions 清理过期会话（根据保留天数）
func (m *ConversationManager) CleanupOldSessions(userID uint, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	cutoff := fmt.Sprintf("DATE_SUB(NOW(), INTERVAL %d DAY)", retentionDays)

	// 查找过期会话 ID
	var sessionIDs []uint
	m.db.Model(&ChatSession{}).Where("user_id = ? AND updated_at < "+cutoff, userID).Pluck("id", &sessionIDs)

	if len(sessionIDs) == 0 {
		return 0, nil
	}

	// 删除过期会话的消息
	m.db.Where("session_id IN ?", sessionIDs).Delete(&ChatMessage{})

	// 删除过期会话
	result := m.db.Where("id IN ? AND user_id = ?", sessionIDs, userID).Delete(&ChatSession{})
	return result.RowsAffected, result.Error
}

// GetRetentionDays 获取用户的会话保留天数设置
func (m *ConversationManager) GetRetentionDays(userID uint) int {
	var setting struct {
		Value string `gorm:"column:value"`
	}
	err := m.db.Table("ai_user_settings").
		Where("user_id = ? AND `key` = 'session_retention_days'", userID).
		First(&setting).Error
	if err != nil {
		return 30 // 默认 30 天
	}
	days := 30
	fmt.Sscanf(setting.Value, "%d", &days)
	if days <= 0 {
		days = 30
	}
	return days
}

// SetRetentionDays 设置用户的会话保留天数
func (m *ConversationManager) SetRetentionDays(userID uint, days int) error {
	if days < 1 {
		days = 1
	}
	if days > 365 {
		days = 365
	}
	// Upsert
	sql := `INSERT INTO ai_user_settings (user_id, ` + "`key`" + `, value, created_at, updated_at)
		VALUES (?, 'session_retention_days', ?, NOW(), NOW())
		ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = NOW()`
	return m.db.Exec(sql, userID, fmt.Sprintf("%d", days)).Error
}

// EnsureSettingsTable 确保 ai_user_settings 表存在
func (m *ConversationManager) EnsureSettingsTable() {
	m.db.Exec(`CREATE TABLE IF NOT EXISTS ai_user_settings (
		id bigint unsigned NOT NULL AUTO_INCREMENT,
		user_id bigint unsigned NOT NULL,
		` + "`key`" + ` varchar(100) NOT NULL,
		value varchar(500) NOT NULL DEFAULT '',
		created_at datetime DEFAULT CURRENT_TIMESTAMP,
		updated_at datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		PRIMARY KEY (id),
		UNIQUE KEY uk_user_key (user_id, ` + "`key`" + `)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
}

// --- 摘要相关方法 ---

// GetSessionByID 通过 ID 获取会话（内部使用，不校验 userID）
func (m *ConversationManager) GetSessionByID(sessionID uint) (*ChatSession, error) {
	var session ChatSession
	if err := m.db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		return nil, fmt.Errorf("会话不存在")
	}
	return &session, nil
}

// CountMessages 统计会话消息数量
func (m *ConversationManager) CountMessages(sessionID uint) int64 {
	var count int64
	m.db.Model(&ChatMessage{}).Where("session_id = ?", sessionID).Count(&count)
	return count
}

// GetOldMessages 获取指定 ID 之后、最近 N 条之前的旧消息（用于生成摘要）
// 返回的是需要被摘要压缩的消息
func (m *ConversationManager) GetOldMessages(sessionID uint, afterID uint, recentLimit int) ([]ChatMessage, error) {
	// 先获取最近 N 条消息的最小 ID
	var recentMsgs []ChatMessage
	m.db.Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(recentLimit).
		Find(&recentMsgs)

	if len(recentMsgs) == 0 {
		return nil, nil
	}

	// 最近消息中的最小 ID
	minRecentID := recentMsgs[len(recentMsgs)-1].ID

	// 获取在 afterID 之后、minRecentID 之前的消息
	var oldMsgs []ChatMessage
	query := m.db.Where("session_id = ? AND id < ?", sessionID, minRecentID)
	if afterID > 0 {
		query = query.Where("id > ?", afterID)
	}
	if err := query.Order("created_at ASC").Find(&oldMsgs).Error; err != nil {
		return nil, err
	}

	return oldMsgs, nil
}

// UpdateSummary 更新会话摘要
func (m *ConversationManager) UpdateSummary(sessionID uint, summary string, upToID uint) error {
	return m.db.Model(&ChatSession{}).Where("id = ?", sessionID).
		Updates(map[string]any{
			"summary":          summary,
			"summary_up_to_id": upToID,
		}).Error
}
