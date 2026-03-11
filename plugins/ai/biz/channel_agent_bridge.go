package biz

import (
	"context"
	"strings"
	"sync"
	"time"
)

type ChannelAgentBridge struct {
	agent      *Agent
	channelSvc *ChannelService
	mu         sync.Mutex
	cancelFn   map[uint]context.CancelFunc
}

func NewChannelAgentBridge(agent *Agent, channelSvc *ChannelService) *ChannelAgentBridge {
	return &ChannelAgentBridge{
		agent:      agent,
		channelSvc: channelSvc,
		cancelFn:   make(map[uint]context.CancelFunc),
	}
}

func (b *ChannelAgentBridge) StreamChannelMessage(
	ctx context.Context,
	channel *AIChannelConfig,
	sessionID uint,
	content string,
	onEvent func(AgentEvent),
) (string, []AgentEvent, error) {
	user, err := b.channelSvc.ResolveExecuteUser(channel)
	if err != nil {
		return "", nil, err
	}
	model, err := b.channelSvc.ResolveModel(channel)
	if err != nil {
		return "", nil, err
	}

	adapter := NewModelAdapter(model)
	eventCh := make(chan AgentEvent, 64)
	runCtx, cancel := context.WithCancel(ctx)
	b.registerSessionRun(sessionID, cancel)
	defer func() {
		cancel()
		b.unregisterSessionRun(sessionID, cancel)
	}()
	go b.agent.RunStream(runCtx, adapter, sessionID, content, user.ID, user.Username, model.MaxToolCalls, eventCh)

	var textBuilder strings.Builder
	var events []AgentEvent
	cancelled := false
	var cancelDrain <-chan time.Time
	for {
		select {
		case <-ctx.Done():
			if !cancelled {
				cancelled = true
				cancelDrain = time.After(1500 * time.Millisecond)
				continue
			}
			return strings.TrimSpace(textBuilder.String()), events, ctx.Err()
		case <-cancelDrain:
			return strings.TrimSpace(textBuilder.String()), events, ctx.Err()
		case event, ok := <-eventCh:
			if !ok {
				return strings.TrimSpace(textBuilder.String()), events, nil
			}
			events = append(events, event)
			if onEvent != nil {
				onEvent(event)
			}
			if event.Type == "text_delta" {
				textBuilder.WriteString(event.Content)
			}
		}
	}
}

func (b *ChannelAgentBridge) StopSession(sessionID uint) bool {
	b.mu.Lock()
	cancel := b.cancelFn[sessionID]
	b.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (b *ChannelAgentBridge) registerSessionRun(sessionID uint, cancel context.CancelFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cancelFn[sessionID] = cancel
}

func (b *ChannelAgentBridge) unregisterSessionRun(sessionID uint, cancel context.CancelFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.cancelFn, sessionID)
}
