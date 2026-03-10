package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	rbacbiz "github.com/ydcloud-dy/mom/internal/biz/rbac"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChannelExternalIdentity struct {
	ExternalUserID  string
	ExternalOpenID  string
	ExternalUnionID string
	ExternalChatID  string
}

type ChannelService struct {
	db      *gorm.DB
	convMgr *ConversationManager
}

const (
	defaultInboundMessageRetentionDays = 15
	inboundMessageCleanupInterval      = 12 * time.Hour
	inboundMessageCleanupInitialDelay  = 3 * time.Minute
	channelSessionMaxMessages          = 60
	channelSessionIdleThreshold        = 24 * time.Hour
	channelSessionMaxAge               = 7 * 24 * time.Hour
)

type ChannelSessionResolution struct {
	Binding       *AIChannelBinding
	Session       *ChatSession
	CreatedNew    bool
	NoticeMessage string
}

type ChannelConversationPolicy struct {
	MaxMessages int
	IdleHours   int
	MaxAgeDays  int
}

type channelConversationPolicyConfig struct {
	SessionMaxMessages *int `json:"sessionMaxMessages,omitempty"`
	SessionIdleHours   *int `json:"sessionIdleHours,omitempty"`
	SessionMaxAgeDays  *int `json:"sessionMaxAgeDays,omitempty"`
}

func NewChannelService(db *gorm.DB) *ChannelService {
	return &ChannelService{
		db:      db,
		convMgr: NewConversationManager(db),
	}
}

func NormalizeChannelType(channelType string) string {
	return strings.ToLower(strings.TrimSpace(channelType))
}

func IsSupportedChannelType(channelType string) bool {
	switch NormalizeChannelType(channelType) {
	case ChannelTypeFeishu, ChannelTypeWeCom, ChannelTypeDingTalk:
		return true
	default:
		return false
	}
}

func (s *ChannelService) ListChannels() ([]AIChannelConfig, error) {
	var channels []AIChannelConfig
	if err := s.db.Order("channel_type ASC, id ASC").Find(&channels).Error; err != nil {
		return nil, err
	}
	for i := range channels {
		channels[i].AppSecretSet = channels[i].AppSecret != ""
	}
	return channels, nil
}

func (s *ChannelService) GetChannel(id uint) (*AIChannelConfig, error) {
	var channel AIChannelConfig
	if err := s.db.First(&channel, id).Error; err != nil {
		return nil, err
	}
	channel.AppSecretSet = channel.AppSecret != ""
	return &channel, nil
}

func (s *ChannelService) CreateChannel(channel *AIChannelConfig) error {
	channel.ChannelType = NormalizeChannelType(channel.ChannelType)
	if !IsSupportedChannelType(channel.ChannelType) {
		return fmt.Errorf("不支持的渠道类型: %s", channel.ChannelType)
	}
	if channel.Name == "" {
		return errors.New("渠道名称不能为空")
	}
	if channel.AppID == "" {
		return errors.New("App ID 不能为空")
	}
	if channel.AppSecret == "" {
		return errors.New("App Secret 不能为空")
	}
	if channel.ExecuteAsUserID == 0 {
		return errors.New("执行身份不能为空")
	}
	if channel.Status == "" {
		channel.Status = ChannelStatusStopped
	}
	if len(channel.ConfigJSON) == 0 {
		channel.ConfigJSON = datatypes.JSON([]byte(`{}`))
	}
	return s.db.Create(channel).Error
}

func (s *ChannelService) UpdateChannel(id uint, updates *AIChannelConfig) (*AIChannelConfig, error) {
	existing, err := s.GetChannel(id)
	if err != nil {
		return nil, err
	}

	existing.Name = updates.Name
	existing.ChannelType = NormalizeChannelType(updates.ChannelType)
	existing.Enabled = updates.Enabled
	existing.AppID = updates.AppID
	existing.DefaultModelID = updates.DefaultModelID
	existing.ExecuteAsUserID = updates.ExecuteAsUserID
	existing.ConfigJSON = updates.ConfigJSON
	if len(existing.ConfigJSON) == 0 {
		existing.ConfigJSON = datatypes.JSON([]byte(`{}`))
	}
	if strings.TrimSpace(updates.AppSecret) != "" {
		existing.AppSecret = updates.AppSecret
	}
	if !IsSupportedChannelType(existing.ChannelType) {
		return nil, fmt.Errorf("不支持的渠道类型: %s", existing.ChannelType)
	}
	if existing.Name == "" {
		return nil, errors.New("渠道名称不能为空")
	}
	if existing.AppID == "" {
		return nil, errors.New("App ID 不能为空")
	}
	if existing.ExecuteAsUserID == 0 {
		return nil, errors.New("执行身份不能为空")
	}
	if err := s.db.Save(existing).Error; err != nil {
		return nil, err
	}
	existing.AppSecretSet = existing.AppSecret != ""
	return existing, nil
}

func (s *ChannelService) DeleteChannel(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("channel_config_id = ?", id).Delete(&AIChannelBinding{}).Error; err != nil {
			return err
		}
		return tx.Delete(&AIChannelConfig{}, id).Error
	})
}

func (s *ChannelService) SetChannelEnabled(id uint, enabled bool) error {
	return s.db.Model(&AIChannelConfig{}).Where("id = ?", id).Update("enabled", enabled).Error
}

func (s *ChannelService) UpdateChannelRuntime(id uint, status string, lastError string, connectedAt *time.Time) error {
	updates := map[string]any{
		"status":     status,
		"last_error": lastError,
	}
	if connectedAt != nil {
		updates["last_connected_at"] = *connectedAt
	}
	return s.db.Model(&AIChannelConfig{}).Where("id = ?", id).Updates(updates).Error
}

func (s *ChannelService) TryAcquireInboundMessage(
	channel *AIChannelConfig,
	externalMessageID string,
	externalChatID string,
	externalOpenID string,
) (bool, error) {
	externalMessageID = strings.TrimSpace(externalMessageID)
	if externalMessageID == "" {
		return true, nil
	}

	record := &AIChannelInboundMessage{
		ChannelType:       channel.ChannelType,
		ChannelConfigID:   channel.ID,
		ExternalMessageID: externalMessageID,
		ExternalChatID:    strings.TrimSpace(externalChatID),
		ExternalOpenID:    strings.TrimSpace(externalOpenID),
	}

	result := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(record)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (s *ChannelService) CleanupOldInboundMessages(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = defaultInboundMessageRetentionDays
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result := s.db.Unscoped().
		Where("created_at < ?", cutoff).
		Delete(&AIChannelInboundMessage{})
	return result.RowsAffected, result.Error
}

func (s *ChannelService) StartInboundMessageCleanupScheduler(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

	go func() {
		timer := time.NewTimer(inboundMessageCleanupInitialDelay)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.doInboundMessageCleanup()
		}

		ticker := time.NewTicker(inboundMessageCleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.doInboundMessageCleanup()
			}
		}
	}()
}

func (s *ChannelService) doInboundMessageCleanup() {
	deleted, err := s.CleanupOldInboundMessages(defaultInboundMessageRetentionDays)
	if err != nil {
		log.Printf("[ai-channel-cleanup] 清理外部渠道幂等记录失败: %v", err)
		return
	}
	if deleted > 0 {
		log.Printf("[ai-channel-cleanup] 已清理 %d 条超过 %d 天的外部渠道幂等记录", deleted, defaultInboundMessageRetentionDays)
	}
}

func (s *ChannelService) ResolveExecuteUser(channel *AIChannelConfig) (*rbacbiz.SysUser, error) {
	if channel.ExecuteAsUserID == 0 {
		return nil, errors.New("未配置执行身份")
	}
	var user rbacbiz.SysUser
	if err := s.db.Where("id = ? AND status = 1", channel.ExecuteAsUserID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("执行身份不存在或已禁用")
	}
	return &user, nil
}

func (s *ChannelService) ResolveModel(channel *AIChannelConfig) (*AIModelConfig, error) {
	var model AIModelConfig
	if channel.DefaultModelID > 0 {
		if err := s.db.Where("id = ? AND status = 1", channel.DefaultModelID).First(&model).Error; err == nil {
			return &model, nil
		}
	}
	if err := s.db.Where("is_default = ? AND status = 1", true).First(&model).Error; err == nil {
		return &model, nil
	}
	if err := s.db.Where("status = 1").First(&model).Error; err == nil {
		return &model, nil
	}
	return nil, errors.New("未配置可用 AI 模型")
}

func (s *ChannelService) FindOrCreateBindingSession(channel *AIChannelConfig, identity ChannelExternalIdentity) (*AIChannelBinding, *ChatSession, error) {
	resolution, err := s.ResolveBindingSession(channel, identity, false)
	if err != nil {
		return nil, nil, err
	}
	return resolution.Binding, resolution.Session, nil
}

func (s *ChannelService) ResolveBindingSession(
	channel *AIChannelConfig,
	identity ChannelExternalIdentity,
	forceNewSession bool,
) (*ChannelSessionResolution, error) {
	lookupField := ""
	lookupValue := ""
	switch {
	case strings.TrimSpace(identity.ExternalOpenID) != "":
		lookupField = "external_open_id"
		lookupValue = strings.TrimSpace(identity.ExternalOpenID)
	case strings.TrimSpace(identity.ExternalUserID) != "":
		lookupField = "external_user_id"
		lookupValue = strings.TrimSpace(identity.ExternalUserID)
	case strings.TrimSpace(identity.ExternalUnionID) != "":
		lookupField = "external_union_id"
		lookupValue = strings.TrimSpace(identity.ExternalUnionID)
	default:
		return nil, errors.New("缺少外部用户标识")
	}

	var binding AIChannelBinding
	query := s.db.Where("channel_config_id = ? AND channel_type = ?", channel.ID, channel.ChannelType)
	query = query.Where(fmt.Sprintf("%s = ?", lookupField), lookupValue)
	if strings.TrimSpace(identity.ExternalChatID) != "" {
		query = query.Where("external_chat_id = ?", strings.TrimSpace(identity.ExternalChatID))
	}
	if err := query.First(&binding).Error; err == nil {
		var session ChatSession
		if err := s.db.First(&session, binding.SessionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				newBinding, newSession, recreateErr := s.recreateBindingSession(channel, identity, &binding, lookupValue)
				if recreateErr != nil {
					return nil, recreateErr
				}
				return &ChannelSessionResolution{
					Binding:    newBinding,
					Session:    newSession,
					CreatedNew: true,
				}, nil
			}
			return nil, err
		}

		if forceNewSession {
			newBinding, newSession, recreateErr := s.recreateBindingSession(channel, identity, &binding, lookupValue)
			if recreateErr != nil {
				return nil, recreateErr
			}
			return &ChannelSessionResolution{
				Binding:       newBinding,
				Session:       newSession,
				CreatedNew:    true,
				NoticeMessage: "已为你开启新的 MOM Claw 会话，请直接发送新的问题。",
			}, nil
		}

		if notice := s.evaluateSessionRotation(channel, &session); notice != "" {
			newBinding, newSession, recreateErr := s.recreateBindingSession(channel, identity, &binding, lookupValue)
			if recreateErr != nil {
				return nil, recreateErr
			}
			return &ChannelSessionResolution{
				Binding:       newBinding,
				Session:       newSession,
				CreatedNew:    true,
				NoticeMessage: notice,
			}, nil
		}

		return &ChannelSessionResolution{
			Binding: &binding,
			Session: &session,
		}, nil
	}

	user, err := s.ResolveExecuteUser(channel)
	if err != nil {
		return nil, err
	}
	modelID := channel.DefaultModelID
	title := fmt.Sprintf("%s-%s", channel.Name, lookupValue)
	session, err := s.convMgr.CreateSession(user.ID, user.Username, title, modelID)
	if err != nil {
		return nil, err
	}

	binding = AIChannelBinding{
		ChannelType:     channel.ChannelType,
		ChannelConfigID: channel.ID,
		ExternalUserID:  strings.TrimSpace(identity.ExternalUserID),
		ExternalOpenID:  strings.TrimSpace(identity.ExternalOpenID),
		ExternalUnionID: strings.TrimSpace(identity.ExternalUnionID),
		ExternalChatID:  strings.TrimSpace(identity.ExternalChatID),
		SessionID:       session.ID,
	}
	if err := s.db.Create(&binding).Error; err != nil {
		return nil, err
	}
	return &ChannelSessionResolution{
		Binding:    &binding,
		Session:    session,
		CreatedNew: true,
	}, nil
}

func (s *ChannelService) evaluateSessionRotation(channel *AIChannelConfig, session *ChatSession) string {
	policy := s.getChannelConversationPolicy(channel)
	messageCount := s.convMgr.CountMessages(session.ID)
	if policy.MaxMessages > 0 && messageCount >= int64(policy.MaxMessages) {
		return fmt.Sprintf("检测到当前对话历史已累计 %d 条消息，已自动为你开启新的 MOM Claw 会话。本次问题将按新上下文处理。", messageCount)
	}

	if policy.IdleHours > 0 && time.Since(session.UpdatedAt) >= time.Duration(policy.IdleHours)*time.Hour {
		return fmt.Sprintf("检测到当前对话距离上次交流已超过 %d 小时，已自动为你开启新的 MOM Claw 会话。本次问题将按新上下文处理。", policy.IdleHours)
	}

	if policy.MaxAgeDays > 0 && time.Since(session.CreatedAt) >= time.Duration(policy.MaxAgeDays)*24*time.Hour {
		return fmt.Sprintf("检测到当前对话已连续使用超过 %d 天，已自动为你开启新的 MOM Claw 会话。本次问题将按新上下文处理。", policy.MaxAgeDays)
	}

	return ""
}

func (s *ChannelService) getChannelConversationPolicy(channel *AIChannelConfig) ChannelConversationPolicy {
	policy := ChannelConversationPolicy{
		MaxMessages: channelSessionMaxMessages,
		IdleHours:   int(channelSessionIdleThreshold / time.Hour),
		MaxAgeDays:  int(channelSessionMaxAge / (24 * time.Hour)),
	}
	if channel == nil || len(channel.ConfigJSON) == 0 {
		return policy
	}

	var cfg channelConversationPolicyConfig
	if err := json.Unmarshal(channel.ConfigJSON, &cfg); err != nil {
		return policy
	}

	if cfg.SessionMaxMessages != nil {
		policy.MaxMessages = normalizeChannelPolicyNumber(*cfg.SessionMaxMessages)
	}
	if cfg.SessionIdleHours != nil {
		policy.IdleHours = normalizeChannelPolicyNumber(*cfg.SessionIdleHours)
	}
	if cfg.SessionMaxAgeDays != nil {
		policy.MaxAgeDays = normalizeChannelPolicyNumber(*cfg.SessionMaxAgeDays)
	}
	return policy
}

func normalizeChannelPolicyNumber(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func (s *ChannelService) recreateBindingSession(
	channel *AIChannelConfig,
	identity ChannelExternalIdentity,
	binding *AIChannelBinding,
	lookupValue string,
) (*AIChannelBinding, *ChatSession, error) {
	user, err := s.ResolveExecuteUser(channel)
	if err != nil {
		return nil, nil, err
	}

	title := fmt.Sprintf("%s-%s", channel.Name, lookupValue)
	session, err := s.convMgr.CreateSession(user.ID, user.Username, title, channel.DefaultModelID)
	if err != nil {
		return nil, nil, err
	}

	binding.ExternalUserID = strings.TrimSpace(identity.ExternalUserID)
	binding.ExternalOpenID = strings.TrimSpace(identity.ExternalOpenID)
	binding.ExternalUnionID = strings.TrimSpace(identity.ExternalUnionID)
	binding.ExternalChatID = strings.TrimSpace(identity.ExternalChatID)
	binding.SessionID = session.ID

	if err := s.db.Save(binding).Error; err != nil {
		return nil, nil, err
	}
	return binding, session, nil
}
