package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ydcloud-dy/mom/pkg/response"
	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// CreateSession 创建对话会话
func (h *Handler) CreateSession(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	var req struct {
		Title   string `json:"title"`
		ModelID uint   `json:"modelId"`
	}
	c.ShouldBindJSON(&req)

	uid, _ := userID.(uint)
	uname, _ := username.(string)

	session, err := h.convMgr.CreateSession(uid, uname, req.Title, req.ModelID)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, session)
}

// ListSessions 获取会话列表
func (h *Handler) ListSessions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)

	sessions, err := h.convMgr.ListSessions(uid)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取会话列表失败")
		return
	}
	response.Success(c, sessions)
}

// DeleteSession 删除会话
func (h *Handler) DeleteSession(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := h.convMgr.DeleteSession(uint(id), uid); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(c, nil)
}

// GetMessages 获取会话消息
func (h *Handler) GetMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// 验证会话归属
	_, err := h.convMgr.GetSession(uint(id), uid)
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "会话不存在")
		return
	}

	messages, err := h.convMgr.GetMessages(uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取消息失败")
		return
	}
	response.Success(c, messages)
}

// SendMessage HTTP 方式发送消息（非流式，返回完整结果）
func (h *Handler) SendMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	uid, _ := userID.(uint)
	uname, _ := username.(string)

	var req struct {
		SessionID uint   `json:"sessionId" binding:"required"`
		Content   string `json:"content" binding:"required"`
		ModelID   uint   `json:"modelId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}

	// 验证会话
	_, err := h.convMgr.GetSession(req.SessionID, uid)
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "会话不存在")
		return
	}

	// 获取模型
	model := h.getModel(req.ModelID)
	if model == nil {
		response.ErrorCode(c, http.StatusBadRequest, "未配置 AI 模型，请先在模型配置中添加模型")
		return
	}

	adapter := biz.NewModelAdapter(model)
	eventCh := make(chan biz.AgentEvent, 64)

	go h.agent.Run(c.Request.Context(), adapter, req.SessionID, req.Content, uid, uname, model.MaxToolCalls, eventCh)

	// 收集所有事件
	var contentBuilder string
	var events []biz.AgentEvent
	for event := range eventCh {
		events = append(events, event)
		if event.Type == "text_delta" {
			contentBuilder += event.Content
		}
		if event.Type == "message_end" {
			break
		}
	}

	response.Success(c, map[string]any{
		"content": contentBuilder,
		"events":  events,
	})
}

// ChatWebSocket WebSocket 方式对话（流式）
func (h *Handler) ChatWebSocket(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	uid, _ := userID.(uint)
	uname, _ := username.(string)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ai-chat] WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
		return nil
	})
	_, busCh, unsubscribe := biz.GlobalSessionStreamBus.Subscribe(uid)
	defer unsubscribe()

	// 写锁：WebSocket 不支持并发写
	var writeMu sync.Mutex
	safeWriteEvent := func(event biz.AgentEvent) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return h.writeWSEvent(conn, event)
	}
	safeWriteJSON := func(v any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return h.writeWSJSON(conn, v)
	}

	// Ping 保活
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			writeMu.Lock()
			err := conn.WriteMessage(websocket.PingMessage, nil)
			writeMu.Unlock()
			if err != nil {
				return
			}
		}
	}()

	// 用于取消当前正在运行的 Agent
	var currentCancel context.CancelFunc
	var cancelMu sync.Mutex

	// 收到的 WebSocket 消息通过 channel 传递
	type wsMsg struct {
		Type      string `json:"type"`
		SessionID uint   `json:"sessionId"`
		Content   string `json:"content"`
		ModelID   uint   `json:"modelId"`
	}
	msgCh := make(chan wsMsg, 8)
	doneCh := make(chan struct{})

	// 独立的读取 goroutine：持续读取 WebSocket 消息
	go func() {
		defer close(doneCh)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("[ai-chat] WebSocket read error: %v", err)
				}
				// 连接断开时取消正在运行的 Agent
				cancelMu.Lock()
				if currentCancel != nil {
					currentCancel()
				}
				cancelMu.Unlock()
				return
			}

			var req wsMsg
			if err := json.Unmarshal(message, &req); err != nil {
				safeWriteEvent(biz.AgentEvent{Type: "error", Error: "消息格式错误"})
				continue
			}

			// 停止请求直接在读取 goroutine 中处理（不走 channel），确保立即响应
			if req.Type == "stop" {
				cancelMu.Lock()
				if currentCancel != nil {
					currentCancel()
					currentCancel = nil
				}
				cancelMu.Unlock()
				continue
			}

			msgCh <- req
		}
	}()

	// 主循环：处理消息请求
	for {
		select {
		case req, ok := <-msgCh:
			if !ok {
				return
			}

			if req.Type != "message" || req.Content == "" {
				continue
			}

			// 获取模型
			model := h.getModel(req.ModelID)
			if model == nil {
				safeWriteEvent(biz.AgentEvent{Type: "error", Error: "未配置 AI 模型"})
				continue
			}

			// 如果没有 session，创建一个
			if req.SessionID == 0 {
				session, err := h.convMgr.CreateSession(uid, uname, "", req.ModelID)
				if err != nil {
					safeWriteEvent(biz.AgentEvent{Type: "error", Error: "创建会话失败"})
					continue
				}
				req.SessionID = session.ID
				safeWriteJSON(map[string]any{
					"type":    "session_created",
					"session": session,
				})
			}

			// 取消之前的 Agent（如果有）
			cancelMu.Lock()
			if currentCancel != nil {
				currentCancel()
			}
			ctx, cancel := context.WithCancel(c.Request.Context())
			currentCancel = cancel
			cancelMu.Unlock()

			adapter := biz.NewModelAdapter(model)
			eventCh := make(chan biz.AgentEvent, 64)
			streamSessionID := req.SessionID

			go h.agent.RunStream(ctx, adapter, req.SessionID, req.Content, uid, uname, model.MaxToolCalls, eventCh)

			// 转发事件给前端，附带 sessionId 让前端区分会话
			for event := range eventCh {
				wrappedEvent := map[string]any{
					"type":      event.Type,
					"sessionId": streamSessionID,
				}
				if event.Content != "" {
					wrappedEvent["content"] = event.Content
				}
				if event.ToolName != "" {
					wrappedEvent["toolName"] = event.ToolName
				}
				if event.ToolParams != "" {
					wrappedEvent["toolParams"] = event.ToolParams
				}
				if event.ToolResult != "" {
					wrappedEvent["toolResult"] = event.ToolResult
				}
				if event.ActionID != "" {
					wrappedEvent["actionId"] = event.ActionID
				}
				if event.Description != "" {
					wrappedEvent["description"] = event.Description
				}
				if event.RiskLevel != "" {
					wrappedEvent["riskLevel"] = event.RiskLevel
				}
				if event.FinishReason != "" {
					wrappedEvent["finishReason"] = event.FinishReason
				}
				if event.Error != "" {
					wrappedEvent["error"] = event.Error
				}
				if event.Usage != nil {
					wrappedEvent["usage"] = event.Usage
				}
				if err := safeWriteJSON(wrappedEvent); err != nil {
					cancel()
					return
				}
			}

		case payload, ok := <-busCh:
			if !ok {
				return
			}
			wrappedEvent := map[string]any{
				"type":      payload.Type,
				"sessionId": payload.SessionID,
				"source":    payload.Source,
			}
			if payload.RunID != "" {
				wrappedEvent["runId"] = payload.RunID
			}
			if payload.Type == "external_user_message" {
				wrappedEvent["content"] = payload.Content
			} else {
				event := payload.Event
				if event.Content != "" {
					wrappedEvent["content"] = event.Content
				}
				if event.ToolName != "" {
					wrappedEvent["toolName"] = event.ToolName
				}
				if event.ToolParams != "" {
					wrappedEvent["toolParams"] = event.ToolParams
				}
				if event.ToolResult != "" {
					wrappedEvent["toolResult"] = event.ToolResult
				}
				if event.ActionID != "" {
					wrappedEvent["actionId"] = event.ActionID
				}
				if event.Description != "" {
					wrappedEvent["description"] = event.Description
				}
				if event.RiskLevel != "" {
					wrappedEvent["riskLevel"] = event.RiskLevel
				}
				if event.RiskMode != "" {
					wrappedEvent["riskMode"] = event.RiskMode
				}
				if event.RiskHint != "" {
					wrappedEvent["riskHint"] = event.RiskHint
				}
				if event.FinishReason != "" {
					wrappedEvent["finishReason"] = event.FinishReason
				}
				if event.Error != "" {
					wrappedEvent["error"] = event.Error
				}
				if event.Usage != nil {
					wrappedEvent["usage"] = event.Usage
				}
			}
			if err := safeWriteJSON(wrappedEvent); err != nil {
				return
			}
		case <-doneCh:
			return
		}
	}
}

// getModel 获取指定模型或默认模型
func (h *Handler) getModel(modelID uint) *biz.AIModelConfig {
	var model biz.AIModelConfig

	if modelID > 0 {
		if err := h.db.Where("id = ? AND status = 1", modelID).First(&model).Error; err == nil {
			return &model
		}
	}

	// 获取默认模型
	if err := h.db.Where("is_default = ? AND status = 1", true).First(&model).Error; err == nil {
		return &model
	}

	// 获取第一个启用的模型
	if err := h.db.Where("status = 1").First(&model).Error; err == nil {
		return &model
	}

	return nil
}

func (h *Handler) writeWSEvent(conn *websocket.Conn, event biz.AgentEvent) error {
	data, _ := json.Marshal(event)
	return conn.WriteMessage(websocket.TextMessage, data)
}

func (h *Handler) writeWSJSON(conn *websocket.Conn, v any) error {
	data, _ := json.Marshal(v)
	return conn.WriteMessage(websocket.TextMessage, data)
}

// GetRetentionSettings 获取会话保留设置
func (h *Handler) GetRetentionSettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	days := h.convMgr.GetRetentionDays(userID.(uint))
	response.Success(c, gin.H{"retentionDays": days})
}

// SetRetentionSettings 设置会话保留天数
func (h *Handler) SetRetentionSettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var req struct {
		RetentionDays int `json:"retentionDays" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := h.convMgr.SetRetentionDays(userID.(uint), req.RetentionDays); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存失败")
		return
	}
	response.Success(c, gin.H{"retentionDays": req.RetentionDays})
}

// CleanupSessions 手动清理过期会话
func (h *Handler) CleanupSessions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)
	days := h.convMgr.GetRetentionDays(uid)
	deleted, err := h.convMgr.CleanupOldSessions(uid, days)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "清理失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{
		"deleted":       deleted,
		"retentionDays": days,
	})
}
