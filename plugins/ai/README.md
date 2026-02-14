完整处理流程
假设用户在 AI 助手中输入 "有哪些主机CPU使用率超过80%？"，整个链路如下：
阶段一：前端发送消息
AIChat.vue → sendMessage()
AIChat.vue
Lines 455-511
// 发送消息async function sendMessage() {  const content = inputMessage.value.trim()  if (!content || isLoading.value) return  // 如果没有选中会话，先创建一个  if (!currentSessionId.value) {    await createNewSession()    if (!currentSessionId.value) return  }  // 添加用户消息到界面  messages.value.push({    id: Date.now(),    role: 'user',    content,  })  inputMessage.value = ''  isLoading.value = true  streamingContent.value = ''  scrollToBottom()  // 优先使用 WebSocket  if (ws && ws.readyState === WebSocket.OPEN) {    ws.send(JSON.stringify({      type: 'message',      sessionId: currentSessionId.value,      content,      modelId: selectedModelId.value,    }))  } else {    // 回退到 HTTP    // ...  }}
先在界面显示用户消息
通过 WebSocket 发送 JSON：{type: "message", sessionId: 5, content: "有哪些主机CPU使用率超过80%？", modelId: 1}
如果 WebSocket 不可用，回退到 HTTP POST /api/v1/plugins/ai/chat/send
阶段二：后端接收消息
chat_handler.go → ChatWebSocket()
chat_handler.go
Lines 177-233
    for {        _, message, err := conn.ReadMessage()        // ...        var req struct {            Type      string `json:"type"`            SessionID uint   `json:"sessionId"`            Content   string `json:"content"`            ModelID   uint   `json:"modelId"`        }        // ...        // 获取模型配置        model := h.getModel(req.ModelID)        // ...        adapter := biz.NewModelAdapter(model)        eventCh := make(chan biz.AgentEvent, 64)        go h.agent.RunStream(c.Request.Context(), adapter, req.SessionID, req.Content, uid, uname, eventCh)        for event := range eventCh {            if err := h.writeWSEvent(conn, event); err != nil {                return            }        }    }
从 WebSocket 读取消息
从数据库获取 AI 模型配置（AIModelConfig：BaseURL、APIKey、ModelName）
创建 ModelAdapter（HTTP 客户端，负责与 LLM API 通信）
创建 eventCh 事件通道
启动 goroutine 执行 agent.RunStream()
主循环从 eventCh 读取事件，实时转发给前端 WebSocket
阶段三：Agent ReAct 引擎
agent.go → RunStream()
这是核心。执行 ReAct（Reasoning + Acting）循环：
Step 3.1 - 构建上下文
agent.go
Lines 269-296
    // 获取历史消息    historyMsgs, _ := a.conversation.GetRecentMessages(sessionID, 20)    // 构建系统上下文    userCtx := a.contextBuilder.BuildSystemContext(userID, username)    fullSysPrompt := systemPrompt + "\n\n--- 当前上下文 ---\n" + userCtx    // 构建消息列表    messages := []ChatCompletionMessage{        {Role: "system", Content: fullSysPrompt},    }    for _, msg := range historyMsgs {        messages = append(messages, ChatCompletionMessage{            Role:    msg.Role,            Content: msg.Content,        })    }    messages = append(messages, ChatCompletionMessage{        Role:    "user",        Content: userMessage,    })    // 保存用户消息    a.conversation.AddMessage(sessionID, "user", userMessage)    a.conversation.AutoTitleFromFirstMessage(sessionID, userMessage)
ContextBuilder 注入的内容（context_builder.go）：
当前用户: admin (ID: 1)当前时间: 2026-02-14 10:30:00用户角色: 管理员平台概况: 管理 50 台主机, 3 个 K8s 集群
最终发给 LLM 的 messages 数组长这样：
[  { role: "system",    content: "你是 MOM 运维管理平台的 AI 助手...\n--- 当前上下文 ---\n当前用户: admin..." },  { role: "user",      content: "之前的对话消息1..." },        // 历史  { role: "assistant", content: "之前的回复1..." },            // 历史  { role: "user",      content: "有哪些主机CPU使用率超过80%？" }  // 本次]
Step 3.2 - 获取工具定义
agent.go
Lines 296-296
    tools := a.registry.GetToolDefinitions()
ToolRegistry 从所有注册的 Skills（17 个内置 + N 个自定义）生成 OpenAI Function Calling 格式的 tools 数组。每个 tool 的元数据来自 SKILL.md 的 YAML 前置元数据：
[  {    "type": "function",    "function": {      "name": "host.list",      "description": "查询主机列表，支持按关键词搜索...",      "parameters": { "type": "object", "properties": { "keyword": {...}, "os_type": {...} } }    }  },  {    "type": "function",    "function": {      "name": "host.analyze",      "description": "分析主机健康状态和资源使用情况...",      "parameters": { "type": "object", "properties": { "metric": {...}, "threshold": {...} } }    }  },  // ... 共 17 个内置 + N 个自定义 skills]
Step 3.3 - ReAct 循环（最多 10 轮）
agent.go
Lines 298-402
    maxIterations := 10    for i := 0; i < maxIterations; i++ {        // 流式调用 LLM        streamCh, err := adapter.ChatCompletionStream(ctx, messages, tools)        // ... 收集响应 ...        // 如果没有工具调用，结束循环        if len(toolCalls) == 0 || finishReason == "stop" {            a.conversation.AddMessage(sessionID, "assistant", content)            eventCh <- AgentEvent{Type: "message_end"}            return        }        // 处理工具调用        for _, tc := range toolCalls {            // 执行工具 → 结果加回 messages → 进入下一轮循环        }    }
阶段四：调用 LLM（第 1 轮）
model_adapter.go → ChatCompletionStream()
model_adapter.go
Lines 184-289
func (a *ModelAdapter) ChatCompletionStream(ctx context.Context, messages []ChatCompletionMessage, tools []ToolDefinition) (<-chan StreamEvent, error) {    req := ChatCompletionRequest{        Model:    a.config.ModelName,  // 例: "gpt-4o" 或 "qwen-plus"        Messages: messages,        Tools:    tools,        Stream:   true,    }    // POST → https://api.openai.com/v1/chat/completions (SSE 流)    // ...}
发送给 LLM 的请求包含 messages + tools。
LLM 分析用户问题后，判断需要调用工具，返回 tool_calls：
{  "choices": [{    "message": {      "content": "",      "tool_calls": [{        "id": "call_abc123",        "type": "function",        "function": {          "name": "host.analyze",          "arguments": "{\"metric\": \"cpu\", \"threshold\": 80}"        }      }]    },    "finish_reason": "tool_calls"  }]}
LLM 决定调用 host.analyze，参数为 metric=cpu, threshold=80。
阶段五：执行 Skill
agent.go → executeTool() → host_skills.go → executeHostAnalyze()
agent.go
Lines 408-442
func (a *Agent) executeTool(name string, argsJSON string, userID uint, username string) map[string]any {    skill, ok := a.registry.Get(name)  // 从 ToolRegistry 获取 "host.analyze"    // ...    result, err := skill.Execute(SkillContext{        UserID:   userID,        Username: username,        Params:   params,     // {"metric": "cpu", "threshold": 80}        DB:       a.db,       // GORM 数据库连接    })    // ...}
执行过程中经过的层：
ToolRegistry.Get("host.analyze")  → BuiltinSkill (loader.go)        // SKILL.md 提供元数据    → executeFn = executeHostAnalyze  // Go 函数提供执行逻辑      → ctx.DB.Table("hosts")...     // 直接查数据库
executeHostAnalyze 的实际查询（host_skills.go）：
SELECT name, ip, cpu_usage as `usage`FROM hostsWHERE deleted_at IS NULL AND status = 1 AND cpu_usage > 80ORDER BY cpu_usage DESCLIMIT 20
返回结果：
{  "threshold": 80,  "alertCount": 3,  "alerts": [    {"name": "web-server-01", "ip": "10.0.1.5",  "metric": "CPU", "usage": 95.2},    {"name": "db-server-02",  "ip": "10.0.1.10", "metric": "CPU", "usage": 88.7},    {"name": "app-server-03", "ip": "10.0.1.15", "metric": "CPU", "usage": 82.1}  ]}
同时向前端发送两个事件：
{type: "tool_call_start", toolName: "host.analyze", toolParams: "{...}"}
{type: "tool_call_result", toolName: "host.analyze", toolResult: "{...}"}
阶段六：LLM 整合结果（第 2 轮）
工具结果被追加到 messages 中，再次调用 LLM：
messages 现在变成:[  { role: "system",    content: "你是 MOM 运维管理平台..." },  { role: "user",      content: "有哪些主机CPU使用率超过80%？" },  { role: "assistant", content: "", tool_calls: [{name: "host.analyze", ...}] },  { role: "tool",      content: '{"threshold":80,"alertCount":3,"alerts":[...]}', tool_call_id: "call_abc123" }]
LLM 看到工具返回的真实数据后，生成最终的自然语言回复（这次没有 tool_calls，finish_reason = "stop"）：
目前有 **3 台主机** CPU 使用率超过 80%：| 主机名 | IP | CPU 使用率 ||--------|-----|-----------|| web-server-01 | 10.0.1.5 | 95.2% || db-server-02 | 10.0.1.10 | 88.7% || app-server-03 | 10.0.1.15 | 82.1% |建议关注 web-server-01，使用率已达 95%，建议排查...
Agent 向前端流式发送：
多个 {type: "text_delta", content: "目前有 **3..."}
最后 {type: "message_end"}
同时保存助手消息到数据库：conversation.AddMessage(sessionID, "assistant", content)
阶段七：前端渲染
AIChat.vue
Lines 399-444
function handleWSEvent(event: any) {  switch (event.type) {    case 'text_delta':      streamingContent.value += event.content   // 逐字追加      scrollToBottom()      break    case 'tool_call_start':      currentToolCalls.value.push({...})        // 显示"执行中"卡片      break    case 'tool_call_result':      last.status = 'success'                    // 更新为"完成"      break    case 'message_end':      messages.value.push({                      // 合并为正式消息        role: 'assistant',        content: streamingContent.value,        toolCalls: currentToolCalls.value,      })      isLoading.value = false      break  }}
完整流程图
用户输入 "有哪些主机CPU使用率超过80%？"│├─ ① 前端 AIChat.vue│   ├── 显示用户消息到界面│   └── WebSocket 发送 JSON {type:"message", content:"...", sessionId:5, modelId:1}│├─ ② 后端 ChatWebSocket (chat_handler.go)│   ├── 解析消息│   ├── 获取 AIModelConfig (数据库)│   ├── 创建 ModelAdapter (HTTP 客户端)│   └── 启动 goroutine → agent.RunStream()│├─ ③ Agent ReAct 引擎 (agent.go)│   ├── 加载历史消息 ← conversation.GetRecentMessages() ← DB│   ├── 构建 SystemPrompt ← ContextBuilder.BuildSystemContext() ← DB│   ├── 获取 Tools 定义 ← ToolRegistry.GetToolDefinitions() ← SKILL.md 元数据│   ││   ├── 【第 1 轮】调用 LLM│   │   ├── ④ ModelAdapter.ChatCompletionStream()│   │   │   └── POST → LLM API (OpenAI/Ollama/自定义)│   │   │       请求: messages + tools (17个 function 定义)│   │   ││   │   └── LLM 返回: tool_calls: [{name:"host.analyze", args:{"metric":"cpu","threshold":80}}]│   ││   ├── ⑤ 执行 Skill│   │   ├── eventCh ← {type:"tool_call_start"} → WebSocket → 前端显示"执行中"│   │   ├── ToolRegistry.Get("host.analyze") → BuiltinSkill│   │   │   ├── 元数据来自 host.analyze/SKILL.md│   │   │   └── 执行函数 executeHostAnalyze() → SQL 查询 → 返回 JSON 结果│   │   └── eventCh ← {type:"tool_call_result"} → WebSocket → 前端显示"完成"│   ││   ├── 【第 2 轮】再次调用 LLM│   │   ├── messages 加入 tool 结果│   │   ├── ⑥ ModelAdapter.ChatCompletionStream()│   │   │   └── POST → LLM API│   │   ││   │   └── LLM 返回: 自然语言回复 (流式), finish_reason="stop"│   │       └── eventCh ← 多个 {type:"text_delta"} → WebSocket → 前端逐字显示│   ││   ├── 保存消息到 DB ← conversation.AddMessage()│   └── eventCh ← {type:"message_end"} → WebSocket → 前端合并为正式消息│└─ ⑦ 前端 handleWSEvent()    ├── text_delta: streamingContent 逐字追加，Markdown 渲染    ├── tool_call_start/result: 显示工具调用卡片    └── message_end: 将流式内容合并，isLoading=false
关键设计点
双通道通信: WebSocket 流式（优先）+ HTTP 非流式（回退）
ReAct 循环: 最多 10 轮，LLM 自主决定调用哪些 tools，直到 finish_reason=stop
SKILL.md 驱动: 元数据（name/description/parameters/riskLevel）从 SKILL.md 加载，执行逻辑在 Go 函数中
事件驱动: Agent 通过 eventCh 异步推送事件，后端实时转发给前端
上下文注入: ContextBuilder 自动注入当前用户、角色、平台规模等信息到 system prompt