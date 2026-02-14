package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// SkillContext Skill 执行上下文
type SkillContext struct {
	Ctx      context.Context
	UserID   uint
	Username string
	Params   map[string]any
	DB       *gorm.DB
}

// Context 返回上下文（如果未设置则返回 Background）
func (sc SkillContext) Context() context.Context {
	if sc.Ctx != nil {
		return sc.Ctx
	}
	return context.Background()
}

// Skill 技能接口
type Skill interface {
	Name() string
	Description() string
	Parameters() json.RawMessage // JSON Schema
	Execute(ctx SkillContext) (any, error)
	RiskLevel() string // low / medium / high / critical
}

// SanitizeToolName 将 Skill 名称转换为 LLM 兼容格式
// LLM API 要求名称匹配 ^[a-zA-Z0-9_-]+$ (不允许包含点号)
// 例如: host.list -> host-list, k8s.cluster_status -> k8s-cluster_status
func SanitizeToolName(name string) string {
	return strings.ReplaceAll(name, ".", "-")
}

// ToolRegistry 工具注册中心
type ToolRegistry struct {
	mu       sync.RWMutex
	skills   map[string]Skill
	aliasMap map[string]string // LLM 清洗后的名称 -> 原始名称
}

// NewToolRegistry 创建工具注册中心
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		skills:   make(map[string]Skill),
		aliasMap: make(map[string]string),
	}
}

// Register 注册 Skill
func (r *ToolRegistry) Register(skill Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()
	originalName := skill.Name()
	r.skills[originalName] = skill
	// 建立 LLM 清洗名称到原始名称的映射
	sanitized := SanitizeToolName(originalName)
	if sanitized != originalName {
		r.aliasMap[sanitized] = originalName
	}
}

// Get 获取 Skill（支持原始名称和 LLM 清洗后的名称）
func (r *ToolRegistry) Get(name string) (Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// 先用原始名称查找
	if s, ok := r.skills[name]; ok {
		return s, ok
	}
	// 再用别名查找（LLM 返回的是清洗后的名称）
	if original, ok := r.aliasMap[name]; ok {
		if s, ok := r.skills[original]; ok {
			return s, ok
		}
	}
	return nil, false
}

// ResolveName 将 LLM 返回的清洗名称解析为原始 Skill 名称
func (r *ToolRegistry) ResolveName(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.skills[name]; ok {
		return name
	}
	if original, ok := r.aliasMap[name]; ok {
		return original
	}
	return name
}

// GetAll 获取所有 Skill
func (r *ToolRegistry) GetAll() []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Skill, 0, len(r.skills))
	for _, s := range r.skills {
		result = append(result, s)
	}
	return result
}

// GetToolDefinitions 获取所有工具定义（发送给 LLM）
// 名称会被清洗为 LLM 兼容格式（点号替换为连字符）
func (r *ToolRegistry) GetToolDefinitions() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDefinition, 0, len(r.skills))
	for _, s := range r.skills {
		defs = append(defs, ToolDefinition{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        SanitizeToolName(s.Name()),
				Description: s.Description(),
				Parameters:  s.Parameters(),
			},
		})
	}
	return defs
}

// GetToolDefinitionsFiltered 获取工具定义（排除被禁用的 Skills）
func (r *ToolRegistry) GetToolDefinitionsFiltered(db *gorm.DB) []ToolDefinition {
	// 查询被禁用的 Skills
	disabledSet := make(map[string]bool)
	var disabledSkills []SkillDefinition
	if db != nil {
		db.Select("name").Where("is_enabled = ?", false).Find(&disabledSkills)
		for _, s := range disabledSkills {
			disabledSet[s.Name] = true
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDefinition, 0, len(r.skills))
	for _, s := range r.skills {
		if disabledSet[s.Name()] {
			continue // 跳过禁用的 Skill
		}
		defs = append(defs, ToolDefinition{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        SanitizeToolName(s.Name()),
				Description: s.Description(),
				Parameters:  s.Parameters(),
			},
		})
	}
	return defs
}

// AgentEvent Agent 事件（发送给前端）
type AgentEvent struct {
	Type         string `json:"type"` // text_delta / tool_call_start / tool_call_result / action_confirm / message_end / error
	Content      string `json:"content,omitempty"`
	ToolName     string `json:"toolName,omitempty"`
	ToolParams   string `json:"toolParams,omitempty"`
	ToolResult   string `json:"toolResult,omitempty"`
	ActionID     string `json:"actionId,omitempty"`
	Description  string `json:"description,omitempty"`
	RiskLevel    string `json:"riskLevel,omitempty"`
	FinishReason string `json:"finishReason,omitempty"`
	Error        string `json:"error,omitempty"`
	Usage        *Usage `json:"usage,omitempty"`
}

// Usage Token 使用量
type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
}

// Agent AI Agent 核心引擎
type Agent struct {
	db             *gorm.DB
	registry       *ToolRegistry
	conversation   *ConversationManager
	contextBuilder *ContextBuilder
}

// NewAgent 创建 Agent
func NewAgent(db *gorm.DB, registry *ToolRegistry) *Agent {
	return &Agent{
		db:             db,
		registry:       registry,
		conversation:   NewConversationManager(db),
		contextBuilder: NewContextBuilder(db),
	}
}

// systemPrompt 系统提示词
const systemPrompt = `你是 MOM 运维管理平台的 AI 助手。你可以帮助用户管理和分析平台中的各类运维资源，包括主机管理、Kubernetes 集群、任务执行、监控告警、审计日志等。

你的能力：
1. 查询和分析平台中的资源信息
2. 执行运维操作（高风险操作需要用户确认后才执行）
3. 生成运维报告和分析建议
4. 回答运维相关的技术问题

工作原则：
- 优先使用可用的工具来获取准确信息，不要编造数据
- 回答要清晰、结构化，善用表格和列表
- 如果工具返回错误，如实告知用户并给出建议

⚠️ 高风险操作确认机制（非常重要）：
对于高风险操作（扩缩容、重启、远程命令执行、节点管理等），执行流程如下：
1. 第一次调用工具时，不传 confirmed 参数。工具会返回 status="pending_confirmation" 和操作详情。
2. 你必须把操作详情清晰地展示给用户，询问用户是否确认执行。
3. 如果用户明确表示"确认"、"执行"、"好的"、"可以"等肯定回复，则第二次调用同一个工具，带上 confirmed=true 参数，这样工具才会真正执行操作。
4. 如果用户说"取消"或"不要"，则不再调用工具，告知用户操作已取消。

示例流程：
- 用户: "把 order-service 扩展到 5 个副本"
- 你: 调用 k8s-scale(resource_name="order-service", replicas=5, namespace="default") → 返回 pending_confirmation
- 你: "即将把 Deployment/order-service 的副本数从当前调整为 5，确认执行吗？"
- 用户: "确认"
- 你: 调用 k8s-scale(resource_name="order-service", replicas=5, namespace="default", confirmed=true) → 返回 success
- 你: "✅ 已成功将 order-service 的副本数调整为 5"`

// Run 执行 Agent（非流式，简单的 ReAct 循环）
func (a *Agent) Run(ctx context.Context, adapter *ModelAdapter, sessionID uint, userMessage string, userID uint, username string, eventCh chan<- AgentEvent) {
	defer func() {
		if r := recover(); r != nil {
			eventCh <- AgentEvent{Type: "error", Error: fmt.Sprintf("Agent 异常: %v", r)}
		}
		eventCh <- AgentEvent{Type: "message_end"}
	}()

	// 获取历史消息
	historyMsgs, _ := a.conversation.GetRecentMessages(sessionID, 20)

	// 构建系统上下文
	userContext := a.contextBuilder.BuildSystemContext(userID, username)
	fullSystemPrompt := systemPrompt + "\n\n--- 当前上下文 ---\n" + userContext

	// 构建消息列表
	messages := []ChatCompletionMessage{
		{Role: "system", Content: fullSystemPrompt},
	}
	for _, msg := range historyMsgs {
		messages = append(messages, ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	messages = append(messages, ChatCompletionMessage{
		Role:    "user",
		Content: userMessage,
	})

	// 保存用户消息
	a.conversation.AddMessage(sessionID, "user", userMessage)
	a.conversation.AutoTitleFromFirstMessage(sessionID, userMessage)

	// 获取工具定义
	tools := a.registry.GetToolDefinitionsFiltered(a.db)

	// 收集所有工具调用记录（用于持久化）
	var allToolCallRecords []map[string]any

	// ReAct 循环（最多 10 轮工具调用）
	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		// 调用 LLM
		resp, err := adapter.ChatCompletion(ctx, messages, tools)
		if err != nil {
			eventCh <- AgentEvent{Type: "error", Error: fmt.Sprintf("调用模型失败: %v", err)}
			return
		}

		if len(resp.Choices) == 0 {
			eventCh <- AgentEvent{Type: "error", Error: "模型未返回任何响应"}
			return
		}

		choice := resp.Choices[0]
		assistantMsg := choice.Message

		// 如果有文本内容，发送给前端
		if assistantMsg.Content != "" {
			eventCh <- AgentEvent{Type: "text_delta", Content: assistantMsg.Content}
		}

		// 如果没有工具调用，结束循环
		if len(assistantMsg.ToolCalls) == 0 {
			// 保存助手消息（附带工具调用记录）
			a.saveAssistantMessage(sessionID, assistantMsg.Content, allToolCallRecords)
			eventCh <- AgentEvent{
				Type: "message_end",
				Usage: &Usage{
					PromptTokens:     resp.Usage.PromptTokens,
					CompletionTokens: resp.Usage.CompletionTokens,
				},
			}
			return
		}

		// 处理工具调用
		messages = append(messages, ChatCompletionMessage{
			Role:      "assistant",
			Content:   assistantMsg.Content,
			ToolCalls: assistantMsg.ToolCalls,
		})

		for _, tc := range assistantMsg.ToolCalls {
			llmName := tc.Function.Name
			toolArgs := tc.Function.Arguments
			displayName := a.registry.ResolveName(llmName)

			riskLevel := ""
			if skill, ok := a.registry.Get(llmName); ok {
				riskLevel = skill.RiskLevel()
			}

			eventCh <- AgentEvent{
				Type:       "tool_call_start",
				ToolName:   displayName,
				ToolParams: toolArgs,
				RiskLevel:  riskLevel,
			}

			result := a.executeTool(llmName, toolArgs, userID, username)
			resultJSON, _ := json.Marshal(result)

			// 记录工具调用
			hasError := false
			if errMsg, ok := result["error"]; ok && errMsg != nil {
				hasError = true
			}
			allToolCallRecords = append(allToolCallRecords, map[string]any{
				"toolName":  displayName,
				"params":    toolArgs,
				"result":    string(resultJSON),
				"riskLevel": riskLevel,
				"status":    map[bool]string{true: "error", false: "success"}[hasError],
			})

			eventCh <- AgentEvent{
				Type:       "tool_call_result",
				ToolName:   displayName,
				ToolResult: string(resultJSON),
			}

			messages = append(messages, ChatCompletionMessage{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
			})
		}
	}

	// 超过最大迭代次数
	a.saveAssistantMessage(sessionID, "\n\n[已达到最大工具调用次数，结束处理]", allToolCallRecords)
	eventCh <- AgentEvent{Type: "text_delta", Content: "\n\n[已达到最大工具调用次数，结束处理]"}
	eventCh <- AgentEvent{Type: "message_end"}
}

// RunStream 流式执行 Agent
func (a *Agent) RunStream(ctx context.Context, adapter *ModelAdapter, sessionID uint, userMessage string, userID uint, username string, eventCh chan<- AgentEvent) {
	defer func() {
		if r := recover(); r != nil {
			eventCh <- AgentEvent{Type: "error", Error: fmt.Sprintf("Agent 异常: %v", r)}
		}
		close(eventCh)
	}()

	// 获取历史消息
	historyMsgs, _ := a.conversation.GetRecentMessages(sessionID, 20)

	// 构建系统上下文
	userCtx := a.contextBuilder.BuildSystemContext(userID, username)
	fullSysPrompt := systemPrompt + "\n\n--- 当前上下文 ---\n" + userCtx

	// 构建消息列表
	messages := []ChatCompletionMessage{
		{Role: "system", Content: fullSysPrompt},
	}
	for _, msg := range historyMsgs {
		messages = append(messages, ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	messages = append(messages, ChatCompletionMessage{
		Role:    "user",
		Content: userMessage,
	})

	// 保存用户消息
	a.conversation.AddMessage(sessionID, "user", userMessage)
	a.conversation.AutoTitleFromFirstMessage(sessionID, userMessage)

	// 获取工具定义（排除被禁用的 Skills）
	tools := a.registry.GetToolDefinitionsFiltered(a.db)

	// 收集所有工具调用记录（用于持久化）
	var allToolCallRecords []map[string]any

	// ReAct 循环
	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		// 流式调用 LLM
		streamCh, err := adapter.ChatCompletionStream(ctx, messages, tools)
		if err != nil {
			eventCh <- AgentEvent{Type: "error", Error: fmt.Sprintf("调用模型失败: %v", err)}
			return
		}

		// 收集完整的响应
		var contentBuilder strings.Builder
		var toolCalls []ToolCall
		toolCallArgsBuilders := make(map[int]*strings.Builder)
		var finishReason string

		for event := range streamCh {
			switch event.Type {
			case "text_delta":
				contentBuilder.WriteString(event.Content)
				eventCh <- AgentEvent{Type: "text_delta", Content: event.Content}

			case "tool_call_delta":
				for _, tc := range event.ToolCalls {
					idx := 0
					if tc.ID != "" {
						toolCalls = append(toolCalls, ToolCall{
							ID:   tc.ID,
							Type: "function",
							Function: FunctionCall{
								Name: tc.Function.Name,
							},
						})
						idx = len(toolCalls) - 1
						toolCallArgsBuilders[idx] = &strings.Builder{}
					} else {
						idx = len(toolCalls) - 1
					}
					if builder, ok := toolCallArgsBuilders[idx]; ok {
						builder.WriteString(tc.Function.Arguments)
					}
				}

			case "finish":
				finishReason = event.FinishReason

			case "error":
				eventCh <- AgentEvent{Type: "error", Error: event.Error}
				return

			case "done":
				// 流结束
			}
		}

		// 组装完整的工具调用参数
		for idx, builder := range toolCallArgsBuilders {
			if idx < len(toolCalls) {
				toolCalls[idx].Function.Arguments = builder.String()
			}
		}

		content := contentBuilder.String()

		// 如果没有工具调用，结束循环
		if len(toolCalls) == 0 || finishReason == "stop" {
			a.saveAssistantMessage(sessionID, content, allToolCallRecords)
			eventCh <- AgentEvent{Type: "message_end"}
			return
		}

		// 处理工具调用
		messages = append(messages, ChatCompletionMessage{
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			llmName := tc.Function.Name
			toolArgs := tc.Function.Arguments
			displayName := a.registry.ResolveName(llmName)

			riskLevel := ""
			if skill, ok := a.registry.Get(llmName); ok {
				riskLevel = skill.RiskLevel()
			}

			eventCh <- AgentEvent{
				Type:       "tool_call_start",
				ToolName:   displayName,
				ToolParams: toolArgs,
				RiskLevel:  riskLevel,
			}

			result := a.executeTool(llmName, toolArgs, userID, username)
			resultJSON, _ := json.Marshal(result)

			hasError := false
			if errMsg, ok := result["error"]; ok && errMsg != nil {
				hasError = true
			}
			allToolCallRecords = append(allToolCallRecords, map[string]any{
				"toolName":  displayName,
				"params":    toolArgs,
				"result":    string(resultJSON),
				"riskLevel": riskLevel,
				"status":    map[bool]string{true: "error", false: "success"}[hasError],
			})

			eventCh <- AgentEvent{
				Type:       "tool_call_result",
				ToolName:   displayName,
				ToolResult: string(resultJSON),
			}

			messages = append(messages, ChatCompletionMessage{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
			})
		}
	}

	a.saveAssistantMessage(sessionID, "\n\n[已达到最大工具调用次数]", allToolCallRecords)
	eventCh <- AgentEvent{Type: "text_delta", Content: "\n\n[已达到最大工具调用次数]"}
	eventCh <- AgentEvent{Type: "message_end"}
}

// saveAssistantMessage 保存助手消息（附带工具调用记录）
func (a *Agent) saveAssistantMessage(sessionID uint, content string, toolCallRecords []map[string]any) {
	if len(toolCallRecords) > 0 {
		toolCallsJSON, _ := json.Marshal(toolCallRecords)
		a.conversation.AddMessageWithTools(sessionID, "assistant", content, string(toolCallsJSON))
	} else {
		a.conversation.AddMessage(sessionID, "assistant", content)
	}
}

// executeTool 执行工具
func (a *Agent) executeTool(name string, argsJSON string, userID uint, username string) map[string]any {
	skill, ok := a.registry.Get(name)
	if !ok {
		return map[string]any{"error": fmt.Sprintf("工具 %s 不存在", name)}
	}

	// 解析参数
	var params map[string]any
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
			return map[string]any{"error": fmt.Sprintf("参数解析失败: %v", err)}
		}
	}

	startTime := time.Now()

	// 执行
	result, err := skill.Execute(SkillContext{
		UserID:   userID,
		Username: username,
		Params:   params,
		DB:       a.db,
	})

	costTime := time.Since(startTime).Milliseconds()
	displayName := a.registry.ResolveName(name)

	if err != nil {
		log.Printf("[agent] Skill %s 执行失败: %v", name, err)
		// 记录失败的操作审计
		a.writeAuditLog(userID, username, displayName, argsJSON, err.Error(), skill.RiskLevel(), costTime, false)
		return map[string]any{"error": err.Error()}
	}

	// 将结果转为 map
	var resultMap map[string]any
	switch v := result.(type) {
	case map[string]any:
		resultMap = v
	default:
		resultMap = map[string]any{"result": v}
	}

	// 记录操作审计（只记录非 pending_confirmation 的实际执行，以及查询操作）
	status, _ := resultMap["status"].(string)
	if status != "pending_confirmation" {
		resultBrief := a.buildAuditDescription(displayName, params, resultMap)
		a.writeAuditLog(userID, username, displayName, argsJSON, resultBrief, skill.RiskLevel(), costTime, true)
	}

	return resultMap
}

// writeAuditLog 写入 AI 操作审计日志
func (a *Agent) writeAuditLog(userID uint, username, skillName, params, description, riskLevel string, costTime int64, success bool) {
	status := 200
	errMsg := ""
	if !success {
		status = 500
		errMsg = description
		description = "执行失败: " + description
	}

	// 模块名称根据 Skill 前缀确定
	module := "AI 助手"
	if strings.HasPrefix(skillName, "k8s.") || strings.HasPrefix(skillName, "k8s-") {
		module = "AI-Kubernetes"
	} else if strings.HasPrefix(skillName, "host.") || strings.HasPrefix(skillName, "host-") {
		module = "AI-主机管理"
	} else if strings.HasPrefix(skillName, "task.") || strings.HasPrefix(skillName, "task-") {
		module = "AI-任务中心"
	} else if strings.HasPrefix(skillName, "monitor.") || strings.HasPrefix(skillName, "monitor-") {
		module = "AI-监控告警"
	} else if strings.HasPrefix(skillName, "cloud.") || strings.HasPrefix(skillName, "cloud-") {
		module = "AI-云账号"
	} else if strings.HasPrefix(skillName, "audit.") || strings.HasPrefix(skillName, "audit-") {
		module = "AI-审计分析"
	} else if strings.HasPrefix(skillName, "analysis.") || strings.HasPrefix(skillName, "analysis-") {
		module = "AI-综合分析"
	}

	// 截断过长的参数
	if len(params) > 500 {
		params = params[:500] + "..."
	}
	if len(description) > 200 {
		description = description[:200] + "..."
	}

	// 异步写入审计日志
	go func() {
		a.db.Exec(`INSERT INTO sys_operation_log 
			(user_id, username, module, action, description, method, path, params, status, error_msg, cost_time, ip, user_agent, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
			userID, username+"(AI)", module, "AI-Skill:"+skillName, description,
			"SKILL", "/ai/skill/"+skillName, params, status, errMsg, costTime, "AI-Agent", "MOM-AI-Agent/1.0")
	}()
}

// buildAuditDescription 构建审计描述
func (a *Agent) buildAuditDescription(skillName string, params map[string]any, result map[string]any) string {
	// 优先使用结果中的 message 字段
	if msg, ok := result["message"].(string); ok && msg != "" {
		return msg
	}
	// 使用结果中的 status 字段
	if status, ok := result["status"].(string); ok {
		return fmt.Sprintf("Skill %s 执行: %s", skillName, status)
	}
	return fmt.Sprintf("Skill %s 执行完成", skillName)
}
