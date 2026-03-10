package biz

import (
	"context"
	"strings"
)

type ChannelAgentBridge struct {
	agent      *Agent
	channelSvc *ChannelService
}

func NewChannelAgentBridge(agent *Agent, channelSvc *ChannelService) *ChannelAgentBridge {
	return &ChannelAgentBridge{
		agent:      agent,
		channelSvc: channelSvc,
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
	go b.agent.RunStream(ctx, adapter, sessionID, content, user.ID, user.Username, model.MaxToolCalls, eventCh)

	var textBuilder strings.Builder
	var events []AgentEvent
	for {
		select {
		case <-ctx.Done():
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
