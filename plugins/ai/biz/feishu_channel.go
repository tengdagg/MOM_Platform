package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"
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

type feishuTimelineBlock struct {
	Type           string
	Content        string
	ToolName       string
	Status         string
	RiskLevel      string
	IsConfirmation bool
}

type FeishuChannelAdapter struct {
	db         *gorm.DB
	agent      *Agent
	convMgr    *ConversationManager
	channelSvc *ChannelService
	bridge     *ChannelAgentBridge
	channel    *AIChannelConfig
	apiClient  *lark.Client
	lockMu     sync.Mutex
	chatLocks  map[string]*sync.Mutex
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
		chatLocks: make(map[string]*sync.Mutex),
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
	unlock := a.lockChatConversation(req)
	defer unlock()

	identity := ChannelExternalIdentity{
		ExternalChatID:  req.ExternalChatID,
		ExternalUserID:  req.ExternalUserID,
		ExternalOpenID:  req.ExternalOpenID,
		ExternalUnionID: req.ExternalUnionID,
	}

	decision, err := a.channelSvc.ResolveInboundSession(a.channel, identity, req.Content)
	if err != nil {
		_ = a.sendTextByChatID(runCtx, req.ChatID, "创建 AI 会话失败: "+err.Error())
		return
	}
	resolution := decision.Resolution
	session := resolution.Session

	if decision.Action == ChannelSessionControlNewSession {
		notice := BuildChannelNewSessionNotice(resolution)
		if _, createErr := a.createCardMessage(runCtx, req.ChatID, replaceFeishuTimelineWithText("", notice), nil, false); createErr != nil {
			_ = a.sendTextByChatID(runCtx, req.ChatID, notice)
		}
		return
	}

	switch decision.Action {
	case ChannelSessionControlStatus:
		statusText := a.channelSvc.BuildSessionStatusText(a.channel, session)
		if _, createErr := a.createCardMessage(runCtx, req.ChatID, replaceFeishuTimelineWithText("", statusText), nil, false); createErr != nil {
			_ = a.sendTextByChatID(runCtx, req.ChatID, statusText)
		}
		return
	case ChannelSessionControlCompact:
		compacted, compactErr := a.agent.CompactSession(session.ID)
		reply := "当前会话的历史消息较少，暂时不需要压缩上下文。"
		if compactErr != nil {
			reply = "压缩上下文失败: " + compactErr.Error()
		} else if compacted {
			reply = "已压缩当前上下文，后续对话会优先参考摘要与最近消息继续进行。"
		}
		if _, createErr := a.createCardMessage(runCtx, req.ChatID, replaceFeishuTimelineWithText("", reply), nil, false); createErr != nil {
			_ = a.sendTextByChatID(runCtx, req.ChatID, reply)
		}
		return
	case ChannelSessionControlStop:
		reply := "当前没有正在运行的生成任务。"
		if a.bridge.StopSession(session.ID) {
			reply = "已停止当前会话的生成任务。"
		}
		if _, createErr := a.createCardMessage(runCtx, req.ChatID, replaceFeishuTimelineWithText("", reply), nil, false); createErr != nil {
			_ = a.sendTextByChatID(runCtx, req.ChatID, reply)
		}
		return
	}

	sessionNotice := strings.TrimSpace(resolution.NoticeMessage)
	runID := uuid.NewString()

	GlobalSessionStreamBus.Publish(session.UserID, SessionStreamPayload{
		SessionID: session.ID,
		RunID:     runID,
		Type:      "external_user_message",
		Content:   req.Content,
		Source:    "external_channel",
	})

	placeholderText := "正在处理中..."
	if sessionNotice != "" {
		placeholderText = sessionNotice + "\n\n" + placeholderText
	}
	var toolStates []feishuToolState
	var timelineBlocks []feishuTimelineBlock
	if sessionNotice != "" {
		timelineBlocks = appendFeishuTimelineText(timelineBlocks, sessionNotice)
	}
	messageID, err := a.createCardMessage(runCtx, req.ChatID, timelineBlocks, toolStates, true)
	if err != nil {
		messageID = ""
	}

	lastPushedText := placeholderText
	lastPushAt := time.Now()
	var streamedText strings.Builder
	nextTextIsConfirmation := false

	replyText, events, err := a.bridge.StreamChannelMessage(runCtx, a.channel, session.ID, req.Content, func(event AgentEvent) {
		GlobalSessionStreamBus.Publish(session.UserID, SessionStreamPayload{
			SessionID: session.ID,
			RunID:     runID,
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
			timelineBlocks = appendFeishuTimelineTool(timelineBlocks, event.ToolName, "running", event.RiskLevel)
		case "tool_call_result":
			toolStates = updateFeishuToolState(toolStates, event.ToolName, parseFeishuToolResultStatus(event.ToolResult), event.RiskLevel)
			timelineBlocks = updateFeishuTimelineTool(timelineBlocks, event.ToolName, parseFeishuToolResultStatus(event.ToolResult), event.RiskLevel)
			if parseFeishuToolResultStatus(event.ToolResult) == "pending_confirmation" {
				nextTextIsConfirmation = true
			}
		}

		if event.Type == "text_delta" && event.Content != "" {
			decodedContent := html.UnescapeString(event.Content)
			streamedText.WriteString(decodedContent)
			if nextTextIsConfirmation {
				timelineBlocks = appendFeishuConfirmationText(timelineBlocks, decodedContent)
				nextTextIsConfirmation = false
			} else {
				timelineBlocks = appendFeishuTimelineText(timelineBlocks, decodedContent)
			}
		}

		if messageID == "" {
			return
		}

		nextText := strings.TrimSpace(streamedText.String())
		if event.Type == "error" && event.Error != "" {
			nextText = "AI 处理失败: " + event.Error
			timelineBlocks = replaceFeishuTimelineWithText(sessionNotice, nextText)
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
			if updateErr := a.updateCardMessage(runCtx, messageID, timelineBlocks, toolStates, event.Type != "message_end" && event.Type != "error"); updateErr == nil {
				lastPushAt = time.Now()
				lastPushedText = nextText
			}
		} else if shouldFlush && nextText == lastPushedText {
			if updateErr := a.updateCardMessage(runCtx, messageID, timelineBlocks, toolStates, event.Type != "message_end" && event.Type != "error"); updateErr == nil {
				lastPushAt = time.Now()
			}
		}
	})
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		finalErrText := "AI 处理失败: " + err.Error()
		if messageID != "" {
			_ = a.updateCardMessage(runCtx, messageID, replaceFeishuTimelineWithText(sessionNotice, finalErrText), toolStates, false)
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
		finalTimeline := timelineBlocks
		if len(finalTimeline) == 0 {
			finalTimeline = replaceFeishuTimelineWithText(sessionNotice, replyText)
		}
		if strings.TrimSpace(replyText) != strings.TrimSpace(lastPushedText) {
			_ = a.updateCardMessage(runCtx, messageID, finalTimeline, toolStates, false)
		} else {
			_ = a.updateCardMessage(runCtx, messageID, finalTimeline, toolStates, false)
		}
		return
	}
	_ = a.sendTextByChatID(runCtx, req.ChatID, replyText)
}

func (a *FeishuChannelAdapter) lockChatConversation(req feishuIncomingMessage) func() {
	key := strings.TrimSpace(req.ExternalChatID)
	if key == "" {
		key = strings.TrimSpace(req.ChatID)
	}
	if key == "" {
		key = strings.TrimSpace(req.ExternalOpenID)
	}
	if key == "" {
		return func() {}
	}

	a.lockMu.Lock()
	mu := a.chatLocks[key]
	if mu == nil {
		mu = &sync.Mutex{}
		a.chatLocks[key] = mu
	}
	a.lockMu.Unlock()

	mu.Lock()
	return mu.Unlock
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

func (a *FeishuChannelAdapter) createCardMessage(ctx context.Context, chatID string, timelineBlocks []feishuTimelineBlock, toolStates []feishuToolState, processing bool) (string, error) {
	if chatID == "" {
		return "", fmt.Errorf("缺少 chat_id")
	}
	contentBytes, err := buildFeishuCardPayload(timelineBlocks, toolStates, processing)
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

func (a *FeishuChannelAdapter) updateCardMessage(ctx context.Context, messageID string, timelineBlocks []feishuTimelineBlock, toolStates []feishuToolState, processing bool) error {
	if messageID == "" {
		return fmt.Errorf("缺少 message_id")
	}
	contentBytes, err := buildFeishuCardPayload(timelineBlocks, toolStates, processing)
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

func buildFeishuCardPayload(timelineBlocks []feishuTimelineBlock, toolStates []feishuToolState, processing bool) ([]byte, error) {
	title := "MOM Claw"
	template := "green"
	statusText := "已完成"
	if processing {
		title = "MOM Claw · 处理中"
		template = "blue"
		statusText = "生成中"
	}

	elements := []map[string]any{
		{
			"tag": "note",
			"elements": []map[string]any{
				{"tag": "plain_text", "content": "多元运维平台外部渠道机器人"},
				{"tag": "plain_text", "content": statusText},
			},
		},
	}

	elements = append(elements, buildFeishuTimelineElements(timelineBlocks, toolStates, processing)...)

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
		"elements": elements,
	}

	return json.Marshal(card)
}

func buildFeishuTimelineElements(timelineBlocks []feishuTimelineBlock, toolStates []feishuToolState, processing bool) []map[string]any {
	blocks := trimFeishuTimelineBlocks(timelineBlocks)
	if len(blocks) == 0 {
		if len(toolStates) == 0 {
			return []map[string]any{
				{
					"tag":     "markdown",
					"content": "正在处理中...",
				},
			}
		}
		lines := make([]string, 0, len(toolStates))
		for idx, tool := range toolStates {
			lines = append(lines, fmt.Sprintf("%d. `%s` %s%s", idx+1, tool.Name, formatFeishuToolStatus(tool.Status), formatFeishuToolRisk(tool.RiskLevel)))
		}
		return []map[string]any{
			{
				"tag":     "markdown",
				"content": strings.Join(lines, "\n"),
			},
		}
	}

	elements := make([]map[string]any, 0, len(blocks)*2)
	toolIdx := 0
	for _, block := range blocks {
		if block.Type == "text" {
			if pendingElements, ok := buildFeishuPendingConfirmationElements(block.Content); ok {
				elements = append(elements, pendingElements...)
				continue
			}
			if block.IsConfirmation {
				elements = append(elements, splitFeishuConfirmationFallback(block.Content)...)
				continue
			}
			if strings.Contains(block.Content, "### 待确认操作") {
				elements = append(elements, splitFeishuConfirmationFallback(block.Content)...)
				continue
			}
		}
		currentToolIdx := toolIdx
		if block.Type == "tool" {
			currentToolIdx++
		}
		content := renderFeishuTimelineBlock(block, currentToolIdx)
		if strings.TrimSpace(content) == "" {
			continue
		}
		elements = append(elements, map[string]any{
			"tag":     "markdown",
			"content": content,
		})
		if block.Type == "tool" {
			toolIdx = currentToolIdx
		}
	}
	if len(elements) == 0 && processing {
		elements = append(elements, map[string]any{
			"tag":     "markdown",
			"content": "正在处理中...",
		})
	}
	return elements
}

func splitFeishuConfirmationFallback(content string) []map[string]any {
	normalized := normalizeFeishuPendingContent(content)
	titleLoc := regexp.MustCompile(`###\s*待确认操作`).FindStringIndex(normalized)
	if titleLoc == nil {
		return []map[string]any{
			{"tag": "markdown", "content": normalizeFeishuReplyText(content)},
		}
	}

	var elements []map[string]any

	prefix := strings.TrimSpace(normalized[:titleLoc[0]])
	if prefix != "" {
		elements = append(elements, map[string]any{
			"tag":     "markdown",
			"content": normalizeFeishuReplyText(prefix),
		})
	}

	confirmationPart := strings.TrimSpace(normalized[titleLoc[0]:])
	if elems, ok := buildFeishuPendingConfirmationElements(confirmationPart); ok {
		elements = append(elements, elems...)
	} else {
		elements = append(elements, map[string]any{
			"tag":     "markdown",
			"content": confirmationPart,
		})
	}

	return elements
}

func buildFeishuPendingConfirmationElements(content string) ([]map[string]any, bool) {
	prefixText, sections, actionText, ok := parseFeishuPendingConfirmationSections(content)
	if !ok {
		return nil, false
	}
	elements := make([]map[string]any, 0, len(sections)+3)
	if strings.TrimSpace(prefixText) != "" {
		elements = append(elements, map[string]any{
			"tag":     "markdown",
			"content": normalizeFeishuReplyText(prefixText),
		})
	}
	elements = append(elements, map[string]any{
		"tag":     "markdown",
		"content": "⚠️ **待确认操作**",
	})
	for _, section := range sections {
		body := strings.TrimSpace(section.Body)
		if body == "" {
			continue
		}
		if section.Title == "执行命令" {
			body = extractFeishuCommandBlock(body)
			if body == "" {
				continue
			}
			elements = append(elements, map[string]any{
				"tag":     "markdown",
				"content": fmt.Sprintf("**%s**\n```\n%s\n```", section.Title, body),
			})
			continue
		}
		elements = append(elements, map[string]any{
			"tag":     "markdown",
			"content": fmt.Sprintf("**%s**\n%s", section.Title, normalizeFeishuSectionBody(body)),
		})
	}
	if actionText != "" {
		elements = append(elements, map[string]any{
			"tag": "note",
			"elements": []map[string]any{
				{"tag": "plain_text", "content": actionText},
			},
		})
	}
	return elements, true
}

type feishuPendingSection struct {
	Title string
	Body  string
}

func parseFeishuPendingConfirmationSections(content string) (string, []feishuPendingSection, string, bool) {
	normalized := normalizeFeishuPendingContent(content)
	titleLoc := regexp.MustCompile(`###\s*待确认操作`).FindStringIndex(normalized)
	if titleLoc == nil {
		return "", nil, "", false
	}

	prefixText := strings.TrimSpace(normalized[:titleLoc[0]])
	body := strings.TrimSpace(normalized[titleLoc[1]:])
	if body == "" {
		return prefixText, nil, "", false
	}

	lines := strings.Split(body, "\n")
	sectionTitleRe := regexp.MustCompile(`^\*\*(.+?)\*\*$`)
	sections := make([]feishuPendingSection, 0, 4)
	actionText := ""
	currentTitle := ""
	currentBody := make([]string, 0, 6)
	flushSection := func() {
		if strings.TrimSpace(currentTitle) == "" {
			currentBody = currentBody[:0]
			return
		}
		sections = append(sections, feishuPendingSection{
			Title: strings.TrimSpace(currentTitle),
			Body:  strings.TrimSpace(strings.Join(currentBody, "\n")),
		})
		currentTitle = ""
		currentBody = currentBody[:0]
	}

	for idx, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			if currentTitle != "" && len(currentBody) > 0 && currentBody[len(currentBody)-1] != "" {
				currentBody = append(currentBody, "")
			}
			continue
		}

		if strings.HasPrefix(line, "请回复") {
			flushSection()
			actionText = strings.TrimSpace(strings.Join(lines[idx:], "\n"))
			break
		}

		titleMatch := sectionTitleRe.FindStringSubmatch(line)
		if len(titleMatch) == 2 {
			flushSection()
			currentTitle = titleMatch[1]
			continue
		}

		if currentTitle == "" {
			continue
		}
		currentBody = append(currentBody, line)
	}
	flushSection()

	if len(sections) == 0 {
		return prefixText, nil, actionText, false
	}
	return prefixText, sections, actionText, true
}

func normalizeFeishuPendingContent(content string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	replacements := []struct {
		pattern string
		target  string
	}{
		{`(?s)\s*###\s*待确认操作\s*`, "\n### 待确认操作\n"},
		{`(?s)\s*\*\*目标主机\*\*\s*`, "\n**目标主机**\n"},
		{`(?s)\s*\*\*超时设置\*\*\s*`, "\n**超时设置**\n"},
		{`(?s)\s*\*\*风险说明\*\*\s*`, "\n**风险说明**\n"},
		{`(?s)\s*\*\*执行命令\*\*\s*`, "\n**执行命令**\n"},
		{`(?s)\s*\*\*操作说明\*\*\s*`, "\n**操作说明**\n"},
		{`(?m)\s*请回复`, "\n请回复"},
	}
	for _, item := range replacements {
		normalized = regexp.MustCompile(item.pattern).ReplaceAllString(normalized, item.target)
	}

	normalized = regexp.MustCompile(`\n{3,}`).ReplaceAllString(normalized, "\n\n")
	return strings.TrimSpace(normalized)
}

func extractFeishuCommandBlock(body string) string {
	body = strings.TrimSpace(body)
	body = html.UnescapeString(body)

	re := regexp.MustCompile("(?s)```(?:\\w+)?\\n(.*?)```")
	matches := re.FindStringSubmatch(body)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}

	body = regexp.MustCompile("^```(?:bash|sh|shell|zsh)?").ReplaceAllString(body, "")
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimSuffix(body, "```")
	return strings.TrimSpace(body)
}

func normalizeFeishuSectionBody(body string) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r", "\n"), "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		normalized = append(normalized, line)
	}
	return strings.Join(normalized, "\n")
}

func renderFeishuTimelineBlock(block feishuTimelineBlock, toolIdx int) string {
	switch block.Type {
	case "tool":
		return fmt.Sprintf("`%d` `%s` %s%s", toolIdx, block.ToolName, formatFeishuToolStatus(block.Status), formatFeishuToolRisk(block.RiskLevel))
	default:
		text := normalizeFeishuReplyText(block.Content)
		return text
	}
}

func trimFeishuTimelineBlocks(blocks []feishuTimelineBlock) []feishuTimelineBlock {
	if len(blocks) == 0 {
		return nil
	}
	const maxChars = 5500
	trimmed := make([]feishuTimelineBlock, 0, len(blocks))
	used := 0
	toolIdx := 0
	for _, block := range blocks {
		currentToolIdx := toolIdx
		if block.Type == "tool" {
			currentToolIdx++
		}
		rendered := renderFeishuTimelineBlock(block, currentToolIdx)
		if strings.TrimSpace(rendered) == "" {
			continue
		}
		if used+len(rendered) > maxChars {
			if block.IsConfirmation || strings.Contains(block.Content, "### 待确认操作") {
				trimmed = append(trimmed, block)
				break
			}
			remaining := maxChars - used
			if remaining > 40 {
				if block.Type == "text" {
					short := rendered[:remaining]
					trimmed = append(trimmed, feishuTimelineBlock{
						Type:    "text",
						Content: short + "\n\n...... 内容较长，剩余部分请到 MOM Claw 查看",
					})
				}
			}
			break
		}
		trimmed = append(trimmed, block)
		used += len(rendered)
		if block.Type == "tool" {
			toolIdx = currentToolIdx
		}
	}
	return trimmed
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

func appendFeishuTimelineText(blocks []feishuTimelineBlock, content string) []feishuTimelineBlock {
	if strings.TrimSpace(content) == "" {
		return blocks
	}
	if len(blocks) > 0 && blocks[len(blocks)-1].Type == "text" && !blocks[len(blocks)-1].IsConfirmation {
		blocks[len(blocks)-1].Content += content
		return blocks
	}
	return append(blocks, feishuTimelineBlock{
		Type:    "text",
		Content: content,
	})
}

func appendFeishuConfirmationText(blocks []feishuTimelineBlock, content string) []feishuTimelineBlock {
	if strings.TrimSpace(content) == "" {
		return blocks
	}
	return append(blocks, feishuTimelineBlock{
		Type:           "text",
		Content:        content,
		IsConfirmation: true,
	})
}

func appendFeishuTimelineTool(blocks []feishuTimelineBlock, toolName string, status string, riskLevel string) []feishuTimelineBlock {
	if strings.TrimSpace(toolName) == "" {
		return blocks
	}
	return append(blocks, feishuTimelineBlock{
		Type:      "tool",
		ToolName:  toolName,
		Status:    status,
		RiskLevel: riskLevel,
	})
}

func updateFeishuTimelineTool(blocks []feishuTimelineBlock, toolName string, status string, riskLevel string) []feishuTimelineBlock {
	for i := len(blocks) - 1; i >= 0; i-- {
		if blocks[i].Type == "tool" && blocks[i].ToolName == toolName && blocks[i].Status == "running" {
			blocks[i].Status = status
			if riskLevel != "" {
				blocks[i].RiskLevel = riskLevel
			}
			return blocks
		}
	}
	return appendFeishuTimelineTool(blocks, toolName, status, riskLevel)
}

func replaceFeishuTimelineWithText(prefix string, content string) []feishuTimelineBlock {
	var blocks []feishuTimelineBlock
	if strings.TrimSpace(prefix) != "" {
		blocks = appendFeishuTimelineText(blocks, prefix)
	}
	return appendFeishuTimelineText(blocks, content)
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
	raw := strings.TrimSpace(toolResult)
	var payload struct {
		Status string `json:"status"`
		Error  any    `json:"error"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err == nil {
		if payload.Status == "pending_confirmation" {
			return "pending_confirmation"
		}
		if payload.Error != nil {
			return "error"
		}
		if payload.Status != "" {
			return payload.Status
		}
	}
	if strings.Contains(raw, `"status":"pending_confirmation"`) || strings.Contains(raw, `"status": "pending_confirmation"`) {
		return "pending_confirmation"
	}
	if strings.Contains(raw, `"error"`) {
		return "error"
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
