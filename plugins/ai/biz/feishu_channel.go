package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkauth "github.com/larksuite/oapi-sdk-go/v3/service/auth/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"gorm.io/gorm"
)

type feishuTextContent struct {
	Text string `json:"text"`
}

type feishuToolState struct {
	Name      string
	Status    string
	RiskLevel string
}

type FeishuChannelAdapter struct {
	db         *gorm.DB
	agent      *Agent
	convMgr    *ConversationManager
	channelSvc *ChannelService
	bridge     *ChannelAgentBridge
	channel    *AIChannelConfig
	apiClient  *lark.Client
}

type feishuIncomingMessage struct {
	MessageID       string
	ChatID          string
	Content         string
	ExternalChatID  string
	ExternalUserID  string
	ExternalOpenID  string
	ExternalUnionID string
}

var feishuNewSessionCommands = map[string]struct{}{
	"新对话":   {},
	"重置上下文": {},
	"清空上下文": {},
	"开始新会话": {},
}

func NewFeishuChannelAdapter(
	db *gorm.DB,
	agent *Agent,
	convMgr *ConversationManager,
	channelSvc *ChannelService,
	channel *AIChannelConfig,
) *FeishuChannelAdapter {
	return &FeishuChannelAdapter{
		db:         db,
		agent:      agent,
		convMgr:    convMgr,
		channelSvc: channelSvc,
		bridge:     NewChannelAgentBridge(agent, channelSvc),
		channel:    channel,
		apiClient: lark.NewClient(
			channel.AppID,
			channel.AppSecret,
			lark.WithLogLevel(larkcore.LogLevelInfo),
		),
	}
}

func (a *FeishuChannelAdapter) Provider() string {
	return ChannelTypeFeishu
}

func (a *FeishuChannelAdapter) Start(ctx context.Context) error {
	if err := a.Test(ctx); err != nil {
		return err
	}

	dispatcher := larkdispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(a.handleMessage).
		OnP2MessageReadV1(func(ctx context.Context, event *larkim.P2MessageReadV1) error {
			return nil
		})

	wsClient := larkws.NewClient(
		a.channel.AppID,
		a.channel.AppSecret,
		larkws.WithEventHandler(dispatcher),
		larkws.WithLogLevel(larkcore.LogLevelInfo),
	)

	go func() {
		if err := wsClient.Start(ctx); err != nil && ctx.Err() == nil {
			_ = a.channelSvc.UpdateChannelRuntime(a.channel.ID, ChannelStatusError, err.Error(), nil)
		}
	}()
	return nil
}

func (a *FeishuChannelAdapter) Stop() error {
	return nil
}

func (a *FeishuChannelAdapter) Test(ctx context.Context) error {
	body := larkauth.NewInternalAppAccessTokenPathReqBodyBuilder().
		AppId(a.channel.AppID).
		AppSecret(a.channel.AppSecret)
	reqBody, _ := body.Build()
	resp, err := a.apiClient.Auth.V3.AppAccessToken.Internal(
		ctx,
		larkauth.NewInternalAppAccessTokenReqBuilder().
			Body(reqBody).
			Build(),
	)
	if err != nil {
		return err
	}
	if !resp.Success() {
		return fmt.Errorf("%s", resp.Msg)
	}
	return nil
}

func (a *FeishuChannelAdapter) handleMessage(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}
	message := event.Event.Message
	if message.ChatType == nil || *message.ChatType != "p2p" {
		return nil
	}
	if message.MessageType == nil || *message.MessageType != "text" {
		return a.sendTextByChatID(ctx, safeString(message.ChatId), "当前聊天渠道第一版仅支持文本消息。")
	}

	content := extractFeishuText(message.Content)
	content = strings.TrimSpace(content)
	if content == "" {
		return a.sendTextByChatID(ctx, safeString(message.ChatId), "未识别到文本内容，请重新发送。")
	}

	req := feishuIncomingMessage{
		MessageID:      safeString(message.MessageId),
		ChatID:         safeString(message.ChatId),
		Content:        content,
		ExternalChatID: safeString(message.ChatId),
	}
	if event.Event.Sender != nil && event.Event.Sender.SenderId != nil {
		req.ExternalUserID = safeString(event.Event.Sender.SenderId.UserId)
		req.ExternalOpenID = safeString(event.Event.Sender.SenderId.OpenId)
		req.ExternalUnionID = safeString(event.Event.Sender.SenderId.UnionId)
	}

	acquired, err := a.channelSvc.TryAcquireInboundMessage(a.channel, req.MessageID, req.ExternalChatID, req.ExternalOpenID)
	if err != nil {
		return err
	}
	if !acquired {
		return nil
	}

	go a.processIncomingMessage(req)
	return nil
}

func (a *FeishuChannelAdapter) processIncomingMessage(req feishuIncomingMessage) {
	runCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	identity := ChannelExternalIdentity{
		ExternalChatID:  req.ExternalChatID,
		ExternalUserID:  req.ExternalUserID,
		ExternalOpenID:  req.ExternalOpenID,
		ExternalUnionID: req.ExternalUnionID,
	}

	forceNewSession := isFeishuNewSessionCommand(req.Content)
	resolution, err := a.channelSvc.ResolveBindingSession(a.channel, identity, forceNewSession)
	if err != nil {
		_ = a.sendTextByChatID(runCtx, req.ChatID, "创建 AI 会话失败: "+err.Error())
		return
	}
	session := resolution.Session

	if forceNewSession {
		notice := resolution.NoticeMessage
		if notice == "" {
			notice = "已为你开启新的 MOM Claw 会话，请直接发送新的问题。"
		}
		if _, createErr := a.createCardMessage(runCtx, req.ChatID, notice, nil, false); createErr != nil {
			_ = a.sendTextByChatID(runCtx, req.ChatID, notice)
		}
		return
	}

	sessionNotice := strings.TrimSpace(resolution.NoticeMessage)

	GlobalSessionStreamBus.Publish(session.UserID, SessionStreamPayload{
		SessionID: session.ID,
		Type:      "external_user_message",
		Content:   req.Content,
		Source:    "external_channel",
	})

	placeholderText := "正在处理中..."
	if sessionNotice != "" {
		placeholderText = sessionNotice + "\n\n" + placeholderText
	}
	var toolStates []feishuToolState
	messageID, err := a.createCardMessage(runCtx, req.ChatID, placeholderText, toolStates, true)
	if err != nil {
		messageID = ""
	}

	lastPushedText := placeholderText
	lastPushAt := time.Now()
	var streamedText strings.Builder

	replyText, events, err := a.bridge.StreamChannelMessage(runCtx, a.channel, session.ID, req.Content, func(event AgentEvent) {
		GlobalSessionStreamBus.Publish(session.UserID, SessionStreamPayload{
			SessionID: session.ID,
			Type:      event.Type,
			Event:     event,
			Source:    "external_channel",
		})

		switch event.Type {
		case "tool_call_start":
			toolStates = append(toolStates, feishuToolState{
				Name:      event.ToolName,
				Status:    "running",
				RiskLevel: event.RiskLevel,
			})
		case "tool_call_result":
			toolStates = updateFeishuToolState(toolStates, event.ToolName, parseFeishuToolResultStatus(event.ToolResult), event.RiskLevel)
		}

		if event.Type == "text_delta" && event.Content != "" {
			streamedText.WriteString(event.Content)
		}

		if messageID == "" {
			return
		}

		nextText := strings.TrimSpace(streamedText.String())
		if event.Type == "error" && event.Error != "" {
			nextText = "AI 处理失败: " + event.Error
		}
		if nextText == "" {
			nextText = placeholderText
		} else if sessionNotice != "" {
			nextText = sessionNotice + "\n\n" + nextText
		}

		shouldFlush := event.Type == "message_end" ||
			event.Type == "error" ||
			event.Type == "tool_call_start" ||
			event.Type == "tool_call_result" ||
			time.Since(lastPushAt) >= 700*time.Millisecond
		if shouldFlush && nextText != lastPushedText {
			if updateErr := a.updateCardMessage(runCtx, messageID, nextText, toolStates, event.Type != "message_end" && event.Type != "error"); updateErr == nil {
				lastPushAt = time.Now()
				lastPushedText = nextText
			}
		} else if shouldFlush && nextText == lastPushedText {
			if updateErr := a.updateCardMessage(runCtx, messageID, nextText, toolStates, event.Type != "message_end" && event.Type != "error"); updateErr == nil {
				lastPushAt = time.Now()
			}
		}
	})
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		finalErrText := "AI 处理失败: " + err.Error()
		if messageID != "" {
			_ = a.updateCardMessage(runCtx, messageID, finalErrText, toolStates, false)
		} else {
			_ = a.sendTextByChatID(runCtx, req.ChatID, finalErrText)
		}
		return
	}

	if replyText == "" {
		replyText = buildFeishuFallbackReply(events)
	}
	if replyText == "" {
		replyText = "消息已处理完成。"
	}
	if sessionNotice != "" {
		replyText = sessionNotice + "\n\n" + replyText
	}

	if messageID != "" {
		if strings.TrimSpace(replyText) != strings.TrimSpace(lastPushedText) {
			_ = a.updateCardMessage(runCtx, messageID, replyText, toolStates, false)
		} else {
			_ = a.updateCardMessage(runCtx, messageID, replyText, toolStates, false)
		}
		return
	}
	_ = a.sendTextByChatID(runCtx, req.ChatID, replyText)
}

func isFeishuNewSessionCommand(content string) bool {
	normalized := strings.TrimSpace(content)
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "　", "")
	normalized = strings.Trim(normalized, "。.!！?？")
	_, ok := feishuNewSessionCommands[normalized]
	return ok
}

func (a *FeishuChannelAdapter) sendTextByChatID(ctx context.Context, chatID string, text string) error {
	_, err := a.createTextMessage(ctx, chatID, text)
	return err
}

func (a *FeishuChannelAdapter) createTextMessage(ctx context.Context, chatID string, text string) (string, error) {
	if chatID == "" {
		return "", fmt.Errorf("缺少 chat_id")
	}
	contentBytes, _ := json.Marshal(feishuTextContent{Text: text})
	body := larkim.NewCreateMessageReqBodyBuilder().
		ReceiveId(chatID).
		MsgType("text").
		Content(string(contentBytes)).
		Uuid(uuid.NewString()).
		Build()
	resp, err := a.apiClient.Im.V1.Message.Create(
		ctx,
		larkim.NewCreateMessageReqBuilder().
			ReceiveIdType("chat_id").
			Body(body).
			Build(),
	)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", fmt.Errorf("%s", resp.Msg)
	}
	if resp.Data == nil {
		return "", nil
	}
	return safeString(resp.Data.MessageId), nil
}

func (a *FeishuChannelAdapter) updateTextMessage(ctx context.Context, messageID string, text string) error {
	if messageID == "" {
		return fmt.Errorf("缺少 message_id")
	}
	contentBytes, _ := json.Marshal(feishuTextContent{Text: text})
	body := larkim.NewUpdateMessageReqBodyBuilder().
		MsgType("text").
		Content(string(contentBytes)).
		Build()
	resp, err := a.apiClient.Im.V1.Message.Update(
		ctx,
		larkim.NewUpdateMessageReqBuilder().
			MessageId(messageID).
			Body(body).
			Build(),
	)
	if err != nil {
		return err
	}
	if !resp.Success() {
		return fmt.Errorf("%s", resp.Msg)
	}
	return nil
}

func (a *FeishuChannelAdapter) createCardMessage(ctx context.Context, chatID string, text string, toolStates []feishuToolState, processing bool) (string, error) {
	if chatID == "" {
		return "", fmt.Errorf("缺少 chat_id")
	}
	contentBytes, err := buildFeishuCardPayload(text, toolStates, processing)
	if err != nil {
		return "", err
	}
	body := larkim.NewCreateMessageReqBodyBuilder().
		ReceiveId(chatID).
		MsgType("interactive").
		Content(string(contentBytes)).
		Uuid(uuid.NewString()).
		Build()
	resp, err := a.apiClient.Im.V1.Message.Create(
		ctx,
		larkim.NewCreateMessageReqBuilder().
			ReceiveIdType("chat_id").
			Body(body).
			Build(),
	)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", fmt.Errorf("%s", resp.Msg)
	}
	if resp.Data == nil {
		return "", nil
	}
	return safeString(resp.Data.MessageId), nil
}

func (a *FeishuChannelAdapter) updateCardMessage(ctx context.Context, messageID string, text string, toolStates []feishuToolState, processing bool) error {
	if messageID == "" {
		return fmt.Errorf("缺少 message_id")
	}
	contentBytes, err := buildFeishuCardPayload(text, toolStates, processing)
	if err != nil {
		return err
	}
	body := larkim.NewPatchMessageReqBodyBuilder().
		Content(string(contentBytes)).
		Build()
	resp, err := a.apiClient.Im.V1.Message.Patch(
		ctx,
		larkim.NewPatchMessageReqBuilder().
			MessageId(messageID).
			Body(body).
			Build(),
	)
	if err != nil {
		return err
	}
	if !resp.Success() {
		return fmt.Errorf("%s", resp.Msg)
	}
	return nil
}

func extractFeishuText(raw *string) string {
	if raw == nil || *raw == "" {
		return ""
	}
	var payload feishuTextContent
	if err := json.Unmarshal([]byte(*raw), &payload); err == nil && payload.Text != "" {
		return payload.Text
	}
	return ""
}

func buildFeishuFallbackReply(events []AgentEvent) string {
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.Type == "error" && event.Error != "" {
			return "AI 处理失败: " + event.Error
		}
		if event.Type == "tool_call_result" && event.ToolResult != "" {
			return "操作已执行，工具已返回结果，请在 MOM Web 端查看完整详情。"
		}
	}
	return ""
}

func buildFeishuCardPayload(text string, toolStates []feishuToolState, processing bool) ([]byte, error) {
	title := "MOM Claw"
	template := "green"
	statusText := "已完成"
	if processing {
		title = "MOM Claw · 处理中"
		template = "blue"
		statusText = "生成中"
	}

	normalized := normalizeFeishuReplyText(text)
	if normalized == "" {
		normalized = "正在处理中..."
	}
	if len(normalized) > 6000 {
		normalized = normalized[:6000] + "\n\n...... 内容较长，剩余部分请到 MOM Claw 查看"
	}

	skillMarkdown := "暂未调用 Skill"
	if len(toolStates) > 0 {
		lines := make([]string, 0, len(toolStates))
		for idx, tool := range toolStates {
			lines = append(lines, fmt.Sprintf("%d. `%s` %s%s", idx+1, tool.Name, formatFeishuToolStatus(tool.Status), formatFeishuToolRisk(tool.RiskLevel)))
		}
		skillMarkdown = strings.Join(lines, "\n")
	}

	card := map[string]any{
		"config": map[string]any{
			"wide_screen_mode": true,
			"update_multi":     true,
		},
		"header": map[string]any{
			"title": map[string]any{
				"tag":     "plain_text",
				"content": title,
			},
			"template": template,
		},
		"elements": []map[string]any{
			{
				"tag": "note",
				"elements": []map[string]any{
					{"tag": "plain_text", "content": "多元运维平台外部渠道机器人"},
					{"tag": "plain_text", "content": statusText},
				},
			},
			{
				"tag":     "markdown",
				"content": "**Skill 调用**\n" + skillMarkdown,
			},
			{
				"tag": "hr",
			},
			{
				"tag":     "markdown",
				"content": "**回复内容**\n" + normalized,
			},
		},
	}

	return json.Marshal(card)
}

func normalizeFeishuReplyText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "<think>", "")
	text = strings.ReplaceAll(text, "</think>", "")
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "__", "")
	text = strings.ReplaceAll(text, "```bash", "")
	text = strings.ReplaceAll(text, "```json", "")
	text = strings.ReplaceAll(text, "```go", "")
	text = strings.ReplaceAll(text, "```", "")

	headingPattern := regexp.MustCompile(`(?m)^#{1,6}\s*`)
	text = headingPattern.ReplaceAllString(text, "")

	lines := strings.Split(text, "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			normalized = append(normalized, "")
			continue
		}
		line = strings.ReplaceAll(line, "|", " | ")
		normalized = append(normalized, line)
	}
	return strings.TrimSpace(strings.Join(normalized, "\n"))
}

func updateFeishuToolState(toolStates []feishuToolState, toolName string, status string, riskLevel string) []feishuToolState {
	for i := len(toolStates) - 1; i >= 0; i-- {
		if toolStates[i].Name == toolName && toolStates[i].Status == "running" {
			toolStates[i].Status = status
			if riskLevel != "" {
				toolStates[i].RiskLevel = riskLevel
			}
			return toolStates
		}
	}
	if toolName != "" {
		return append(toolStates, feishuToolState{Name: toolName, Status: status, RiskLevel: riskLevel})
	}
	return toolStates
}

func parseFeishuToolResultStatus(toolResult string) string {
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(toolResult), &payload); err == nil && payload.Status != "" {
		return payload.Status
	}
	return "success"
}

func formatFeishuToolStatus(status string) string {
	switch status {
	case "running":
		return "[执行中]"
	case "pending_confirmation":
		return "[待确认]"
	case "error":
		return "[失败]"
	default:
		return "[成功]"
	}
}

func formatFeishuToolRisk(riskLevel string) string {
	switch riskLevel {
	case "critical":
		return " [危险]"
	case "high":
		return " [高风险]"
	case "medium":
		return " [中风险]"
	case "low":
		return " [安全]"
	default:
		return ""
	}
}

func safeString(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}
