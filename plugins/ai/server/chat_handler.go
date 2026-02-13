package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
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

	go h.agent.Run(c.Request.Context(), adapter, req.SessionID, req.Content, uid, uname, eventCh)

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

	// Ping 保活
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[ai-chat] WebSocket read error: %v", err)
			}
			return
		}

		var req struct {
			Type      string `json:"type"`
			SessionID uint   `json:"sessionId"`
			Content   string `json:"content"`
			ModelID   uint   `json:"modelId"`
		}
		if err := json.Unmarshal(message, &req); err != nil {
			h.writeWSEvent(conn, biz.AgentEvent{Type: "error", Error: "消息格式错误"})
			continue
		}

		if req.Type != "message" || req.Content == "" {
			continue
		}

		// 获取模型
		model := h.getModel(req.ModelID)
		if model == nil {
			h.writeWSEvent(conn, biz.AgentEvent{Type: "error", Error: "未配置 AI 模型"})
			continue
		}

		// 如果没有 session，创建一个
		if req.SessionID == 0 {
			session, err := h.convMgr.CreateSession(uid, uname, "", req.ModelID)
			if err != nil {
				h.writeWSEvent(conn, biz.AgentEvent{Type: "error", Error: "创建会话失败"})
				continue
			}
			req.SessionID = session.ID
			// 发送 session 创建事件
			h.writeWSJSON(conn, map[string]any{
				"type":    "session_created",
				"session": session,
			})
		}

		adapter := biz.NewModelAdapter(model)
		eventCh := make(chan biz.AgentEvent, 64)

		go h.agent.RunStream(c.Request.Context(), adapter, req.SessionID, req.Content, uid, uname, eventCh)

		for event := range eventCh {
			if err := h.writeWSEvent(conn, event); err != nil {
				return
			}
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
