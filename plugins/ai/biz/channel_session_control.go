package biz

import "strings"

type ChannelSessionControlAction string

const (
	ChannelSessionControlNone       ChannelSessionControlAction = ""
	ChannelSessionControlNewSession ChannelSessionControlAction = "new_session"
	ChannelSessionControlStatus     ChannelSessionControlAction = "status"
	ChannelSessionControlCompact    ChannelSessionControlAction = "compact"
	ChannelSessionControlStop       ChannelSessionControlAction = "stop"
)

type ChannelInboundSessionDecision struct {
	Action            ChannelSessionControlAction
	Resolution        *ChannelSessionResolution
	NormalizedContent string
}

var defaultChannelNewSessionCommands = []string{
	"新对话",
	"重置上下文",
	"清空上下文",
	"开始新会话",
}

var defaultChannelStatusCommands = []string{
	"状态",
	"/status",
	"会话状态",
}

var defaultChannelCompactCommands = []string{
	"压缩上下文",
	"压缩会话",
	"/compact",
}

var defaultChannelStopCommands = []string{
	"停止",
	"/stop",
	"停止生成",
	"停止回答",
}

func (s *ChannelService) ResolveInboundSession(
	channel *AIChannelConfig,
	identity ChannelExternalIdentity,
	content string,
) (*ChannelInboundSessionDecision, error) {
	normalizedContent := strings.TrimSpace(content)
	action := s.DetectSessionControlAction(channel, normalizedContent)
	resolution, err := s.ResolveBindingSession(channel, identity, action == ChannelSessionControlNewSession)
	if err != nil {
		return nil, err
	}
	return &ChannelInboundSessionDecision{
		Action:            action,
		Resolution:        resolution,
		NormalizedContent: normalizedContent,
	}, nil
}

func (s *ChannelService) DetectSessionControlAction(channel *AIChannelConfig, content string) ChannelSessionControlAction {
	normalized := normalizeChannelCommandText(content)
	if normalized == "" {
		return ChannelSessionControlNone
	}

	for _, cmd := range s.getChannelNewSessionCommands(channel) {
		if normalized == normalizeChannelCommandText(cmd) {
			return ChannelSessionControlNewSession
		}
	}
	for _, cmd := range s.getChannelStatusCommands(channel) {
		if normalized == normalizeChannelCommandText(cmd) {
			return ChannelSessionControlStatus
		}
	}
	for _, cmd := range s.getChannelCompactCommands(channel) {
		if normalized == normalizeChannelCommandText(cmd) {
			return ChannelSessionControlCompact
		}
	}
	for _, cmd := range s.getChannelStopCommands(channel) {
		if normalized == normalizeChannelCommandText(cmd) {
			return ChannelSessionControlStop
		}
	}
	return ChannelSessionControlNone
}

func (s *ChannelService) getChannelNewSessionCommands(channel *AIChannelConfig) []string {
	policy := s.getChannelConversationPolicy(channel)
	return mergeChannelCommands(defaultChannelNewSessionCommands, policy.ResetCommands)
}

func (s *ChannelService) getChannelStatusCommands(channel *AIChannelConfig) []string {
	policy := s.getChannelConversationPolicy(channel)
	return mergeChannelCommands(defaultChannelStatusCommands, policy.StatusCommands)
}

func (s *ChannelService) getChannelCompactCommands(channel *AIChannelConfig) []string {
	policy := s.getChannelConversationPolicy(channel)
	return mergeChannelCommands(defaultChannelCompactCommands, policy.CompactCommands)
}

func (s *ChannelService) getChannelStopCommands(channel *AIChannelConfig) []string {
	policy := s.getChannelConversationPolicy(channel)
	return mergeChannelCommands(defaultChannelStopCommands, policy.StopCommands)
}

func mergeChannelCommands(defaults []string, custom []string) []string {
	commands := append([]string{}, defaults...)
	if len(custom) == 0 {
		return commands
	}

	seen := make(map[string]struct{}, len(commands)+len(custom))
	merged := make([]string, 0, len(commands)+len(custom))
	for _, cmd := range append(commands, custom...) {
		normalized := normalizeChannelCommandText(cmd)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		merged = append(merged, cmd)
	}
	return merged
}

func normalizeChannelCommandText(content string) string {
	normalized := strings.TrimSpace(content)
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "　", "")
	normalized = strings.Trim(normalized, "。.!！?？")
	return normalized
}

func BuildChannelNewSessionNotice(resolution *ChannelSessionResolution) string {
	if resolution == nil || strings.TrimSpace(resolution.NoticeMessage) == "" {
		return "已为你开启新的 MOM Claw 会话，请直接发送新的问题。"
	}
	return strings.TrimSpace(resolution.NoticeMessage)
}
