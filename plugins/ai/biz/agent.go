package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"gorm.io/gorm"
)

// SkillContext Skill 执行上下文
type SkillContext struct {
	UserID   uint
	Username string
	Params   map[string]any
	DB       *gorm.DB
}

// Skill 技能接口
type Skill interface {
	Name() string
	Description() string
	Parameters() json.RawMessage // JSON Schema
	Execute(ctx SkillContext) (any, error)
	RiskLevel() string // low / medium / high / critical
}

// ToolRegistry 工具注册中心
type ToolRegistry struct {
	mu     sync.RWMutex
	skills map[string]Skill
}

// NewToolRegistry 创建工具注册中心
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		skills: make(map[string]Skill),
	}
}

// Register 注册 Skill
func (r *ToolRegistry) Register(skill Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.skills[skill.Name()] = skill
}

// Get 获取 Skill
func (r *ToolRegistry) Get(name string) (Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.skills[name]
	return s, ok
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
func (r *ToolRegistry) GetToolDefinitions() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDefinition, 0, len(r.skills))
	for _, s := range r.skills {
		defs = append(defs, ToolDefinition{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        s.Name(),
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
2. 执行运维操作（需要用户确认高风险操作）
3. 生成运维报告和分析建议
4. 回答运维相关的技术问题

工作原则：
- 优先使用可用的工具来获取准确信息，不要编造数据
- 对于高风险操作（删除、重启、执行命令等），务必提醒用户确认
- 回答要清晰、结构化，善用表格和列表
- 如果工具返回错误，如实告知用户并给出建议`

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
	tools := a.registry.GetToolDefinitions()

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
			// 保存助手消息
			a.conversation.AddMessage(sessionID, "assistant", assistantMsg.Content)
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
			toolName := tc.Function.Name
			toolArgs := tc.Function.Arguments

			// 通知前端工具调用开始
			eventCh <- AgentEvent{
				Type:       "tool_call_start",
				ToolName:   toolName,
				ToolParams: toolArgs,
			}

			// 执行工具
			result := a.executeTool(toolName, toolArgs, userID, username)

			// 通知前端工具调用结果
			resultJSON, _ := json.Marshal(result)
			eventCh <- AgentEvent{
				Type:       "tool_call_result",
				ToolName:   toolName,
				ToolResult: string(resultJSON),
			}

			// 将工具结果添加到消息列表
			messages = append(messages, ChatCompletionMessage{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
			})
		}
	}

	// 超过最大迭代次数
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

	// 获取工具定义
	tools := a.registry.GetToolDefinitions()

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
					idx := 0 // 默认索引
					if tc.ID != "" {
						// 新的工具调用
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
			a.conversation.AddMessage(sessionID, "assistant", content)
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
			toolName := tc.Function.Name
			toolArgs := tc.Function.Arguments

			eventCh <- AgentEvent{
				Type:       "tool_call_start",
				ToolName:   toolName,
				ToolParams: toolArgs,
			}

			result := a.executeTool(toolName, toolArgs, userID, username)
			resultJSON, _ := json.Marshal(result)

			eventCh <- AgentEvent{
				Type:       "tool_call_result",
				ToolName:   toolName,
				ToolResult: string(resultJSON),
			}

			messages = append(messages, ChatCompletionMessage{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
			})
		}
	}

	eventCh <- AgentEvent{Type: "text_delta", Content: "\n\n[已达到最大工具调用次数]"}
	eventCh <- AgentEvent{Type: "message_end"}
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

	// 执行
	result, err := skill.Execute(SkillContext{
		UserID:   userID,
		Username: username,
		Params:   params,
		DB:       a.db,
	})
	if err != nil {
		log.Printf("[agent] Skill %s 执行失败: %v", name, err)
		return map[string]any{"error": err.Error()}
	}

	// 将结果转为 map
	switch v := result.(type) {
	case map[string]any:
		return v
	default:
		return map[string]any{"result": v}
	}
}
