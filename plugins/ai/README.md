# MOM AI 助手 - 架构与处理流程

## 主 Agent

**你的主 Agent 是 `plugins/ai/biz/agent.go` 中的 `Agent` 结构体。**

```go
type Agent struct {
    db             *gorm.DB           // 数据库连接
    registry       *ToolRegistry      // 工具注册中心（28 个内置 + N 个自定义 Skills）
    conversation   *ConversationManager // 对话管理（历史消息、会话持久化）
    contextBuilder *ContextBuilder    // 上下文构建（注入用户/角色/平台信息到 SystemPrompt）
}
```

Agent 是整个 AI 助手的核心引擎，负责：
- 接收用户消息，构建完整的上下文
- 驱动 ReAct（Reasoning + Acting）循环
- 调度 LLM 决策 → 执行 Skill → 收集结果 → 再次调用 LLM
- 管理对话持久化和操作审计

---

## 系统架构总览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              前端 (Vue.js)                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │ AIChat   │  │AIModelCfg│  │ AISkills │  │ Layout   │  │  Menu    │    │
│  │  .vue    │  │  .vue    │  │  .vue    │  │  .vue    │  │  管理    │    │
│  └────┬─────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
│       │ WebSocket / HTTP                                                   │
└───────┼───────────────────────────────────────────────────────────────────┘
        │
        ▼
┌───────────────────────────────────────────────────────────────────────────┐
│                            后端 (Go / Gin)                                │
│                                                                           │
│  ┌─────────────────┐    ┌──────────────────────────────────────────────┐  │
│  │  ChatWebSocket   │───▶│               Agent (主引擎)                 │  │
│  │  chat_handler.go │    │                agent.go                      │  │
│  └─────────────────┘    │                                              │  │
│                          │  ┌────────────┐  ┌──────────────────────┐   │  │
│                          │  │ Context    │  │    ToolRegistry      │   │  │
│                          │  │ Builder    │  │  28 内置 + N 自定义   │   │  │
│                          │  └────────────┘  └──────────┬───────────┘   │  │
│                          │                             │               │  │
│                          │  ┌───────────────┐  ┌───────▼───────────┐   │  │
│                          │  │ Conversation  │  │   Skill Engine    │   │  │
│                          │  │   Manager     │  │  (Go 内置/JS/Py)  │   │  │
│                          │  └───────────────┘  └───────────────────┘   │  │
│                          └──────────────────────────────────────────────┘  │
│                                      │                                     │
│  ┌───────────────────┐    ┌──────────▼───────────┐   ┌────────────────┐   │
│  │   ModelAdapter     │    │     审计日志          │   │   数据库 (DB)  │   │
│  │  model_adapter.go  │    │  sys_operation_log    │   │  MySQL/TiDB   │   │
│  └────────┬──────────┘    └──────────────────────┘   └────────────────┘   │
│           │                                                                │
└───────────┼────────────────────────────────────────────────────────────────┘
            │ HTTPS (SSE Stream)
            ▼
┌───────────────────────────┐
│     LLM API Provider      │
│  OpenAI / Qwen / DeepSeek │
│  Doubao / Gemini / Ollama │
└───────────────────────────┘
```

---

## 完整处理流程（7 个阶段）

以用户输入 **"有哪些主机 CPU 使用率超过 80%？"** 为例：

```
用户                  前端 AIChat.vue          后端 ChatWebSocket          Agent 主引擎           LLM (大模型)           Skill 执行
 │                        │                         │                        │                       │                      │
 │  ① 输入问题            │                         │                        │                       │                      │
 │───────────────────────▶│                         │                        │                       │                      │
 │                        │                         │                        │                       │                      │
 │                        │  ② WebSocket JSON       │                        │                       │                      │
 │                        │────────────────────────▶│                        │                       │                      │
 │                        │                         │                        │                       │                      │
 │                        │                         │  ③ 创建 ModelAdapter   │                       │                      │
 │                        │                         │  启动 RunStream()      │                       │                      │
 │                        │                         │───────────────────────▶│                       │                      │
 │                        │                         │                        │                       │                      │
 │                        │                         │                        │  ④ 构建上下文          │                      │
 │                        │                         │                        │  加载历史消息          │                      │
 │                        │                         │                        │  注入 SystemPrompt     │                      │
 │                        │                         │                        │  获取 Tools 定义       │                      │
 │                        │                         │                        │                       │                      │
 │                        │                         │                        │  ⑤ 第 1 轮: 调用 LLM  │                      │
 │                        │                         │                        │──────────────────────▶│                      │
 │                        │                         │                        │                       │  LLM 决策:            │
 │                        │                         │                        │                       │  调用 host-analyze    │
 │                        │                         │                        │◁ ─ ─ tool_calls ─ ─ ─│                      │
 │                        │                         │                        │                       │                      │
 │                        │  ◁ ─ tool_call_start ─ │◁ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│                       │                      │
 │  显示"正在调用          │                         │                        │                       │                      │
 │  host.analyze..."      │                         │                        │  ⑥ 执行 Skill         │                      │
 │◁ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│                         │                        │─────────────────────────────────────────────▶│
 │                        │                         │                        │                       │   SQL 查询 hosts 表   │
 │                        │                         │                        │                       │   返回 JSON 结果      │
 │                        │                         │                        │◁ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│
 │                        │                         │                        │                       │                      │
 │                        │  ◁ ─ tool_call_result ─│◁ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│  写入审计日志          │                      │
 │  显示"执行完成"        │                         │                        │                       │                      │
 │◁ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│                         │                        │                       │                      │
 │                        │                         │                        │                       │                      │
 │                        │                         │                        │  ⑦ 第 2 轮: 再调 LLM  │                      │
 │                        │                         │                        │  (带上 tool 结果)      │                      │
 │                        │                         │                        │──────────────────────▶│                      │
 │                        │                         │                        │                       │  LLM 整合数据         │
 │                        │                         │                        │                       │  生成自然语言回复      │
 │                        │                         │                        │◁ ─ 流式文本 ─ ─ ─ ─ ─│                      │
 │                        │                         │                        │                       │                      │
 │                        │  ◁ ─ text_delta (多次)─│◁ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│  保存消息到 DB         │                      │
 │  逐字渲染 Markdown     │                         │                        │  保存 tool_calls JSON  │                      │
 │◁ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│                         │                        │                       │                      │
 │                        │  ◁ ─ message_end ─ ─ ─ │◁ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│                       │                      │
 │  渲染完成              │                         │                        │                       │                      │
 │◁ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─│                         │                        │                       │                      │
 ▼                        ▼                         ▼                        ▼                       ▼                      ▼
```

---

## 阶段详解

### 阶段一：前端发送消息

**文件**: `web/src/views/ai/AIChat.vue` → `sendMessage()`

```
用户输入 "有哪些主机 CPU 使用率超过 80%？"
    │
    ├── 1. 显示用户消息到聊天界面
    ├── 2. 如果没有会话，先创建一个新会话
    └── 3. 通过 WebSocket 发送 JSON:
         {
           type: "message",
           sessionId: 5,
           content: "有哪些主机CPU使用率超过80%？",
           modelId: 1
         }
         如果 WebSocket 不可用，回退到 HTTP POST /api/v1/plugins/ai/chat/send
```

### 阶段二：后端接收并初始化

**文件**: `plugins/ai/server/chat_handler.go` → `ChatWebSocket()`

```
WebSocket 消息到达
    │
    ├── 1. 解析 JSON（sessionId, content, modelId）
    ├── 2. 从 JWT Token 获取用户身份（userID, username）
    ├── 3. 从数据库加载 AI 模型配置（BaseURL, APIKey, ModelName）
    ├── 4. 创建 ModelAdapter（封装 LLM HTTP 客户端）
    ├── 5. 创建 eventCh 事件通道（容量 64）
    ├── 6. 启动 goroutine: agent.RunStream(ctx, adapter, sessionID, content, ...)
    └── 7. 主循环：从 eventCh 读取事件 → 实时转发给前端 WebSocket
```

### 阶段三：Agent 构建上下文

**文件**: `plugins/ai/biz/agent.go` → `RunStream()`

Agent 是主引擎，启动后首先构建完整的 LLM 请求上下文：

```
RunStream() 启动
    │
    ├── 1. 加载历史消息（最近 20 条）
    │      ← ConversationManager.GetRecentMessages()
    │
    ├── 2. 构建 SystemPrompt
    │      ← ContextBuilder.BuildSystemContext()
    │      注入内容：
    │        当前用户: admin (ID: 1)
    │        当前时间: 2026-02-14 10:30:00
    │        用户角色: 管理员
    │        平台概况: 管理 50 台主机, 3 个 K8s 集群
    │
    ├── 3. 组装 messages 数组
    │      [
    │        { role: "system",    content: "你是 MOM 运维管理平台的 AI 助手..." },
    │        { role: "user",      content: "（历史消息1）..." },
    │        { role: "assistant", content: "（历史回复1）..." },
    │        { role: "user",      content: "有哪些主机CPU使用率超过80%？" }
    │      ]
    │
    ├── 4. 保存用户消息到 DB
    ├── 5. 自动设置会话标题（首条消息时）
    │
    └── 6. 获取可用工具定义
           ← ToolRegistry.GetToolDefinitionsFiltered(db)
           ← 排除被用户禁用的 Skills
           ← 生成 OpenAI Function Calling 格式的 tools[]
           ← 名称清洗: host.analyze → host-analyze（LLM API 兼容）
```

### 阶段四：ReAct 循环 — 调用 LLM（第 1 轮）

**文件**: `plugins/ai/biz/model_adapter.go` → `ChatCompletionStream()`

```
Agent 发起 LLM 调用
    │
    ├── POST → https://api.xxx.com/v1/chat/completions
    │   请求体:
    │   {
    │     model: "qwen-plus",
    │     messages: [...],    ← 系统提示 + 历史 + 用户消息
    │     tools: [            ← 28 个内置 Skill 定义
    │       {
    │         type: "function",
    │         function: {
    │           name: "host-analyze",
    │           description: "分析主机健康状态...",
    │           parameters: { type: "object", properties: { metric: {...}, threshold: {...} } }
    │         }
    │       },
    │       ... (共 28 个)
    │     ],
    │     stream: true
    │   }
    │
    └── LLM 分析后决策:
        "用户想知道 CPU 超过 80% 的主机，我应该调用 host-analyze"

        返回 (SSE 流):
        {
          choices: [{
            message: {
              content: "",
              tool_calls: [{
                id: "call_abc123",
                function: {
                  name: "host-analyze",
                  arguments: '{"metric":"cpu","threshold":80}'
                }
              }]
            },
            finish_reason: "tool_calls"
          }]
        }
```

### 阶段五：执行 Skill

**文件**: `plugins/ai/biz/agent.go` → `executeTool()`

```
Agent 收到 LLM 的 tool_calls
    │
    ├── 1. 解析工具名称: "host-analyze" → 还原为 "host.analyze"
    │      ← ToolRegistry.ResolveName()
    │
    ├── 2. 发送事件给前端: { type: "tool_call_start", toolName: "host.analyze", riskLevel: "low" }
    │
    ├── 3. 查找并执行 Skill
    │      ToolRegistry.Get("host-analyze")
    │        → BuiltinSkill {
    │            元数据: host.analyze/SKILL.md (YAML 前置元数据)
    │            执行函数: executeHostAnalyze (host_skills.go)
    │          }
    │        → executeHostAnalyze(SkillContext{
    │            UserID:   1,
    │            Username: "admin",
    │            Params:   {"metric": "cpu", "threshold": 80},
    │            DB:       gorm.DB,
    │          })
    │
    │      实际 SQL:
    │      SELECT name, ip, cpu_usage as `usage`, group_name
    │      FROM hosts LEFT JOIN asset_groups ON ...
    │      WHERE deleted_at IS NULL AND status = 1 AND cpu_usage > 80
    │      ORDER BY cpu_usage DESC LIMIT 20
    │
    ├── 4. 返回结果:
    │      {
    │        "threshold": 80,
    │        "alertCount": 3,
    │        "healthLevel": "warning",
    │        "alerts": [
    │          {"name":"web-01","ip":"10.0.1.5","metric":"CPU","usage":95.2,"groupName":"生产环境"},
    │          {"name":"db-02","ip":"10.0.1.10","metric":"CPU","usage":88.7,"groupName":"数据库"},
    │          {"name":"app-03","ip":"10.0.1.15","metric":"CPU","usage":82.1,"groupName":"应用服务"}
    │        ],
    │        "overallStats": {"avgCpu":45.3,"maxCpu":95.2,...},
    │        "groupStats": [...]
    │      }
    │
    ├── 5. 写入审计日志（异步）
    │      → sys_operation_log: admin(AI) | AI-主机管理 | AI-Skill:host.analyze | low
    │
    ├── 6. 记录 toolCallRecord（用于持久化到消息历史）
    │
    └── 7. 发送事件给前端: { type: "tool_call_result", toolName: "host.analyze", toolResult: "{...}" }
```

### 阶段六：ReAct 循环 — 再次调用 LLM（第 2 轮）

```
Agent 将 Skill 结果追加到 messages，再次调用 LLM
    │
    ├── messages 现在是:
    │   [
    │     { role: "system",    content: "你是 MOM 运维管理平台的 AI 助手..." },
    │     { role: "user",      content: "有哪些主机CPU使用率超过80%？" },
    │     { role: "assistant", tool_calls: [{name:"host-analyze",...}] },
    │     { role: "tool",      content: '{"threshold":80,"alertCount":3,...}', tool_call_id: "call_abc123" }
    │   ]
    │
    ├── LLM 看到真实数据，生成最终自然语言回复:
    │
    │   "目前有 **3 台主机** CPU 使用率超过 80%：
    │    | 主机名 | IP | CPU 使用率 | 分组 |
    │    |--------|-----|-----------|------|
    │    | web-01 | 10.0.1.5  | 95.2% | 生产环境 |
    │    | db-02  | 10.0.1.10 | 88.7% | 数据库   |
    │    | app-03 | 10.0.1.15 | 82.1% | 应用服务 |
    │    建议重点关注 web-01，CPU 使用率已达 95%..."
    │
    ├── finish_reason: "stop" ← 不再需要调用工具
    │
    └── Agent 流式转发 text_delta 事件给前端
```

### 阶段七：前端渲染与持久化

**文件**: `web/src/views/ai/AIChat.vue` → `handleWSEvent()`

```
前端接收事件流
    │
    ├── tool_call_start  → 显示"正在调用 host.analyze..."卡片（在回复文字上方）
    ├── tool_call_result → 更新卡片为"执行完成"，可展开查看参数和结果
    ├── text_delta (多次) → streamingContent 逐字追加，实时 Markdown 渲染
    └── message_end      → 合并为正式消息，isLoading = false

后端持久化:
    ├── ChatMessage: role="assistant", content="目前有 3 台主机..."
    └── ChatMessage.ToolCalls: JSON 序列化的 tool 调用记录（切换会话后仍可查看）
```

---

## 高风险操作的两步确认流程

对于扩缩容、重启、远程命令等危险操作，Agent 执行的是**两步确认**模式：

```
用户: "把 order-service 扩容到 5 个副本"
    │
    │  ┌─── 第 1 轮 LLM 调用 ───┐
    │  │ LLM → tool_calls:       │
    │  │ k8s-scale(replicas=5)   │
    │  └──────────┬──────────────┘
    │             │
    │  ┌─── Skill 执行 ──────────┐
    │  │ 检测: confirmed 未传     │
    │  │ 返回: status =           │
    │  │   "pending_confirmation" │
    │  │ warning: "⚠️ 将把..."   │
    │  └──────────┬──────────────┘
    │             │
    │  ┌─── 第 2 轮 LLM 调用 ───┐
    │  │ LLM 看到 pending，      │
    │  │ 生成确认提问文本        │
    │  │ finish_reason: "stop"   │
    │  └──────────┬──────────────┘
    │             │
    ▼             ▼
AI: "即将把 Deployment/order-service 的副本数调整为 5，确认执行吗？"
    │
用户: "确认"
    │
    │  ┌─── 第 1 轮 LLM 调用 ───┐
    │  │ LLM → tool_calls:       │
    │  │ k8s-scale(replicas=5,   │
    │  │   confirmed=true)       │
    │  └──────────┬──────────────┘
    │             │
    │  ┌─── Skill 真正执行 ──────┐
    │  │ confirmed=true → 执行！ │
    │  │ K8s API: Scale → 5      │
    │  │ 返回: status="success"  │
    │  │ 写入审计日志             │
    │  └──────────┬──────────────┘
    │             │
    ▼             ▼
AI: "✅ 已成功将 order-service 的副本数调整为 5"
```

---

## 核心组件说明

| 组件 | 文件 | 职责 |
|------|------|------|
| **Agent** (主引擎) | `biz/agent.go` | ReAct 循环、工具调度、消息流转、审计日志 |
| **ModelAdapter** | `biz/model_adapter.go` | 封装 LLM API 调用（OpenAI 兼容协议），支持流式/非流式 |
| **ToolRegistry** | `biz/agent.go` | 工具注册中心，管理所有 Skill，名称清洗/还原，过滤禁用 |
| **ConversationManager** | `biz/conversation.go` | 会话管理、历史消息持久化、toolCalls JSON 存储 |
| **ContextBuilder** | `biz/context_builder.go` | 注入用户身份、角色、平台规模到 SystemPrompt |
| **SkillEngine** | `biz/skill_engine.go` | 加载数据库中的自定义 Skills（JS/Python 脚本） |
| **ScriptSandbox** | `biz/sandbox.go` | 沙箱执行自定义 JavaScript/Python 脚本 |
| **BuiltinSkill** | `skills/loader.go` | 从 SKILL.md 加载元数据 + 绑定 Go 执行函数 |
| **ChatWebSocket** | `server/chat_handler.go` | WebSocket 连接管理、事件转发 |

---

## 内置 Skills 清单（28 个）

### 主机管理 (6)

| Skill | 风险 | 说明 |
|-------|------|------|
| `host.list` | low | 查询主机列表，支持搜索/筛选/排序 |
| `host.detail` | low | 查询单台主机详情（资源/系统/云平台信息） |
| `host.analyze` | low | 分析主机健康状态（按指标/分组/阈值） |
| `host.collect` | medium | SSH 采集主机系统信息 |
| `host.exec_command` | critical | SSH 远程执行命令（安全黑名单 + 两步确认） |
| `host.file_manage` | high | 远程文件列表/读取/管理 |

### Kubernetes (7)

| Skill | 风险 | 说明 |
|-------|------|------|
| `k8s.kubectl` | medium | 通用 K8s 操作（get/describe/logs/scale/delete/events 等 20+ 资源类型） |
| `k8s.scale` | high | 扩缩容 Deployment/StatefulSet |
| `k8s.restart` | high | 滚动重启工作负载 |
| `k8s.diagnose` | low | 诊断 Pod/Node 问题 |
| `k8s.node_manage` | critical | 节点 cordon/uncordon/drain |
| `k8s.log_query` | low | 查询 Pod 日志 |
| `k8s.helm_manage` | high | Helm Release 管理 |

### 任务中心 (3)

| Skill | 风险 | 说明 |
|-------|------|------|
| `task.history` | low | 查询任务执行历史 |
| `task.execute` | critical | 在指定主机/分组上执行 Ad-hoc 命令 |
| `task.ansible` | high | 执行 Ansible Playbook |

### 监控告警 (3)

| Skill | 风险 | 说明 |
|-------|------|------|
| `monitor.domain_status` | low | 域名监控状态查询 |
| `monitor.alert_summary` | low | 告警汇总分析 |
| `monitor.alert_config` | medium | 告警规则 CRUD |

### 审计分析 (4)

| Skill | 风险 | 说明 |
|-------|------|------|
| `audit.operation_summary` | low | 操作日志统计（含 AI 操作统计） |
| `audit.login_analysis` | low | 登录行为分析（暴力破解检测） |
| `audit.session_summary` | low | 终端会话汇总 |
| `audit.data_changes` | low | 数据变更追踪 |

### 云账号 (3)

| Skill | 风险 | 说明 |
|-------|------|------|
| `cloud.list_accounts` | low | 查询云账号（支持按厂商/状态筛选） |
| `cloud.list_instances` | low | 查询云主机实例 |
| `cloud.import_hosts` | medium | 从云平台导入主机 |

### 综合分析 (3)

| Skill | 风险 | 说明 |
|-------|------|------|
| `analysis.infra_report` | low | 基础设施综合报告（一键全面巡检） |
| `analysis.security_audit` | low | 安全态势分析（7 类风险检测） |
| `analysis.capacity_plan` | low | 容量规划建议（按分组/指标） |

---

## Skill 的两种来源

```
内置 Skill（Go 编译时嵌入）             自定义 Skill（用户上传 .zip）
┌────────────────────────────┐      ┌────────────────────────────┐
│  host.analyze/             │      │  my-custom-skill/          │
│  ├── SKILL.md  ← 元数据    │      │  ├── SKILL.md  ← 元数据    │
│  │   (name, description,   │      │  │   (name, description,   │
│  │    parameters, risk)    │      │  │    parameters, risk)    │
│  │                         │      │  └── scripts/              │
│  └── 执行逻辑: Go 函数      │      │      └── script.js ← 逻辑  │
│      host_skills.go        │      │          (或 script.py)    │
│      executeHostAnalyze()  │      │                            │
└────────────────────────────┘      └────────────────────────────┘
         │                                    │
         │ go:embed SKILL.md                  │ 上传 → 解析 → 存 DB
         │ loader.go 解析                     │ skill_engine.go 加载
         ▼                                    ▼
     ToolRegistry.Register(skill)         ToolRegistry.Register(skill)
```

---

## 事件流协议（Agent ↔ 前端）

Agent 通过 `eventCh` 向前端 WebSocket 推送以下事件类型：

| 事件类型 | 触发时机 | 关键字段 |
|----------|----------|----------|
| `text_delta` | LLM 流式输出文本 | `content`: 文本片段 |
| `tool_call_start` | 开始执行 Skill | `toolName`, `toolParams`, `riskLevel` |
| `tool_call_result` | Skill 执行完成 | `toolName`, `toolResult` |
| `error` | 任何环节出错 | `error`: 错误信息 |
| `message_end` | 本轮对话完成 | `usage`: Token 使用量 |

---

## 关键设计要点

1. **双通道通信**: WebSocket 流式（优先）+ HTTP 非流式（回退）
2. **ReAct 循环**: 最多 10 轮，LLM 自主决定调用哪些 tools，直到 `finish_reason=stop`
3. **SKILL.md 驱动**: 元数据从 SKILL.md YAML 加载，执行逻辑在 Go 函数中
4. **事件驱动**: Agent 通过 eventCh 异步推送，后端实时转发给前端
5. **上下文注入**: ContextBuilder 自动注入用户/角色/平台规模到 SystemPrompt
6. **两步确认**: 高风险操作 → pending_confirmation → 用户确认 → confirmed=true → 真正执行
7. **审计全覆盖**: 每次 Skill 执行后异步写入 `sys_operation_log`，标记为 AI 操作
8. **Skills 可控**: 用户可在 Skills 管理页面禁用/启用任意内置 Skill，Agent 运行时自动过滤
9. **可扩展**: 支持上传自定义 JS/Python Skill，同名可覆盖内置 Skill
