package biz

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

type ChannelRuntimeManager struct {
	rootCtx    context.Context
	db         *gorm.DB
	agent      *Agent
	convMgr    *ConversationManager
	channelSvc *ChannelService

	mu       sync.RWMutex
	cancelFn map[uint]context.CancelFunc
	adapters map[uint]ChatChannelAdapter
}

func NewChannelRuntimeManager(rootCtx context.Context, db *gorm.DB, agent *Agent, convMgr *ConversationManager, channelSvc *ChannelService) *ChannelRuntimeManager {
	if rootCtx == nil {
		rootCtx = context.Background()
	}
	return &ChannelRuntimeManager{
		rootCtx:    rootCtx,
		db:         db,
		agent:      agent,
		convMgr:    convMgr,
		channelSvc: channelSvc,
		cancelFn:   make(map[uint]context.CancelFunc),
		adapters:   make(map[uint]ChatChannelAdapter),
	}
}

func (m *ChannelRuntimeManager) StartEnabledChannels() {
	var channels []AIChannelConfig
	if err := m.db.Where("enabled = ?", true).Find(&channels).Error; err != nil {
		return
	}
	for _, channel := range channels {
		_ = m.StartChannel(channel.ID)
	}
}

func (m *ChannelRuntimeManager) StartChannel(id uint) error {
	channel, err := m.channelSvc.GetChannel(id)
	if err != nil {
		return err
	}
	adapter, err := m.buildAdapter(channel)
	if err != nil {
		_ = m.channelSvc.UpdateChannelRuntime(id, ChannelStatusError, err.Error(), nil)
		return err
	}

	m.StopChannel(id)

	runCtx, cancel := context.WithCancel(m.rootCtx)
	if err := adapter.Start(runCtx); err != nil {
		cancel()
		_ = m.channelSvc.UpdateChannelRuntime(id, ChannelStatusError, err.Error(), nil)
		return err
	}

	now := time.Now()
	m.mu.Lock()
	m.cancelFn[id] = cancel
	m.adapters[id] = adapter
	m.mu.Unlock()
	_ = m.channelSvc.UpdateChannelRuntime(id, ChannelStatusRunning, "", &now)
	return nil
}

func (m *ChannelRuntimeManager) StopChannel(id uint) error {
	m.mu.Lock()
	cancel := m.cancelFn[id]
	adapter := m.adapters[id]
	delete(m.cancelFn, id)
	delete(m.adapters, id)
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if adapter != nil {
		_ = adapter.Stop()
	}
	return m.channelSvc.UpdateChannelRuntime(id, ChannelStatusStopped, "", nil)
}

func (m *ChannelRuntimeManager) ReconnectChannel(id uint) error {
	_ = m.StopChannel(id)
	return m.StartChannel(id)
}

func (m *ChannelRuntimeManager) TestChannel(ctx context.Context, id uint) error {
	channel, err := m.channelSvc.GetChannel(id)
	if err != nil {
		return err
	}
	adapter, err := m.buildAdapter(channel)
	if err != nil {
		return err
	}
	return adapter.Test(ctx)
}

func (m *ChannelRuntimeManager) IsRunning(id uint) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.adapters[id]
	return ok
}

func (m *ChannelRuntimeManager) buildAdapter(channel *AIChannelConfig) (ChatChannelAdapter, error) {
	switch channel.ChannelType {
	case ChannelTypeFeishu:
		return NewFeishuChannelAdapter(m.db, m.agent, m.convMgr, m.channelSvc, channel), nil
	default:
		return nil, fmt.Errorf("暂不支持的渠道类型: %s", channel.ChannelType)
	}
}
