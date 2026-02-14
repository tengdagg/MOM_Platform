<template>
  <div class="ai-chat-container">
    <!-- 左侧会话列表 -->
    <div class="session-sidebar">
      <div class="sidebar-header">
        <el-button type="primary" :icon="Plus" @click="createNewSession" style="width: 100%">
          新建对话
        </el-button>
      </div>
      <div class="session-list">
        <div
          v-for="session in sessions"
          :key="session.id"
          class="session-item"
          :class="{ active: currentSessionId === session.id }"
          @click="switchSession(session.id)"
        >
          <el-icon><ChatLineRound /></el-icon>
          <span class="session-title">{{ session.title }}</span>
          <el-icon class="session-delete" @click.stop="handleDeleteSession(session.id)">
            <Delete />
          </el-icon>
        </div>
        <div v-if="sessions.length === 0" class="no-sessions">
          <el-empty description="暂无对话" :image-size="60" />
        </div>
      </div>
    </div>

    <!-- 主对话区域 -->
    <div class="chat-main">
      <!-- 顶部栏 -->
      <div class="chat-header">
        <div class="chat-title">
          {{ currentSession?.title || 'AI 助手' }}
        </div>
        <div class="chat-actions">
          <el-select
            v-model="selectedModelId"
            placeholder="选择模型"
            size="small"
            style="width: 200px"
          >
            <el-option
              v-for="model in models"
              :key="model.id"
              :label="model.name"
              :value="model.id"
            >
              <span>{{ model.name }}</span>
              <span style="color: #999; font-size: 12px; margin-left: 8px;">{{ model.modelName }}</span>
            </el-option>
          </el-select>
        </div>
      </div>

      <!-- 消息区域 -->
      <div class="messages-container" ref="messagesContainer">
        <div v-if="messages.length === 0 && !isLoading" class="welcome-screen">
          <div class="welcome-icon">
            <el-icon :size="64" color="#409eff"><ChatDotRound /></el-icon>
          </div>
          <h2>MOM AI 助手</h2>
          <p>我可以帮你管理和分析平台中的运维资源</p>

          <!-- 对话模板 -->
          <div v-if="templates.length > 0" class="template-section">
            <div class="template-title">场景模板</div>
            <div class="template-cards">
              <div
                v-for="tpl in templates"
                :key="tpl.id"
                class="template-card"
                @click="useTemplate(tpl)"
              >
                <span class="tpl-icon">{{ tpl.icon }}</span>
                <div class="tpl-info">
                  <div class="tpl-name">{{ tpl.name }}</div>
                  <div class="tpl-desc">{{ tpl.description }}</div>
                </div>
              </div>
            </div>
          </div>

          <div class="quick-actions">
            <el-tag
              v-for="action in quickActions"
              :key="action"
              class="quick-action-tag"
              @click="sendQuickAction(action)"
              effect="plain"
              size="large"
            >
              {{ action }}
            </el-tag>
          </div>
        </div>

        <div v-for="msg in messages" :key="msg.id" class="message-wrapper" :class="msg.role">
          <div class="message-avatar">
            <el-avatar v-if="msg.role === 'user'" :size="36" style="background: #409eff">
              <el-icon><User /></el-icon>
            </el-avatar>
            <el-avatar v-else :size="36" style="background: #67c23a">
              <el-icon><MagicStick /></el-icon>
            </el-avatar>
          </div>
          <div class="message-content">
            <div class="message-role">{{ msg.role === 'user' ? '你' : 'AI 助手' }}</div>
            <!-- 工具调用卡片（显示在文字上方） -->
            <div v-if="msg.toolCalls && msg.toolCalls.length > 0" class="tool-calls">
              <div class="tool-calls-title">
                <el-icon><Operation /></el-icon>
                <span>调用了 {{ msg.toolCalls.length }} 个 Skill</span>
              </div>
              <div v-for="(tc, idx) in msg.toolCalls" :key="idx" class="tool-call-card" :class="'status-' + tc.status">
                <div class="tool-call-header" @click="tc._expanded = !tc._expanded">
                  <span class="tool-call-step">{{ idx + 1 }}</span>
                  <span class="tool-name">{{ tc.toolName }}</span>
                  <el-tag :type="tc.status === 'success' ? 'success' : 'danger'" size="small" class="status-tag">
                    {{ tc.status === 'success' ? '成功' : '失败' }}
                    {{ tc.duration ? `(${tc.duration}ms)` : '' }}
                  </el-tag>
                  <el-icon class="expand-icon">
                    <ArrowDown v-if="!tc._expanded" /><ArrowUp v-else />
                  </el-icon>
                </div>
                <div v-if="tc._expanded" class="tool-call-body">
                  <div class="tool-section">
                    <div class="tool-section-title">📥 调用参数</div>
                    <pre>{{ formatJSON(tc.params) }}</pre>
                  </div>
                  <div class="tool-section" v-if="tc.result">
                    <div class="tool-section-title">📤 返回结果</div>
                    <pre>{{ formatJSON(tc.result) }}</pre>
                  </div>
                </div>
              </div>
            </div>
            <div class="message-body" v-html="renderMarkdown(msg.content)"></div>
          </div>
        </div>

        <!-- 正在输入 -->
        <div v-if="isLoading" class="message-wrapper assistant">
          <div class="message-avatar">
            <el-avatar :size="36" style="background: #67c23a">
              <el-icon><MagicStick /></el-icon>
            </el-avatar>
          </div>
          <div class="message-content">
            <div class="message-role">AI 助手</div>
            <!-- 流式工具调用（显示在文字上方） -->
            <div v-if="currentToolCalls.length > 0" class="tool-calls streaming-tools">
              <div class="tool-calls-title">
                <el-icon><Operation /></el-icon>
                <span>Skill 调用过程</span>
              </div>
              <div v-for="(tc, idx) in currentToolCalls" :key="idx" class="tool-call-card" :class="'status-' + tc.status">
                <div class="tool-call-header" @click="tc._expanded = !tc._expanded">
                  <span class="tool-call-step">{{ idx + 1 }}</span>
                  <span class="tool-name">{{ tc.toolName }}</span>
                  <el-tag v-if="tc.riskLevel === 'high' || tc.riskLevel === 'critical'" type="danger" size="small" effect="plain">
                    {{ tc.riskLevel === 'critical' ? '危险' : '高风险' }}
                  </el-tag>
                  <el-tag v-if="tc.status === 'running'" type="warning" size="small" class="status-tag">
                    <span class="running-dot"></span> 执行中...
                  </el-tag>
                  <el-tag v-else-if="tc.status === 'success'" type="success" size="small" class="status-tag">
                    完成 {{ tc.duration ? `(${tc.duration}ms)` : '' }}
                  </el-tag>
                  <el-tag v-else-if="tc.status === 'error'" type="danger" size="small" class="status-tag">失败</el-tag>
                  <el-icon class="expand-icon" v-if="tc.params || tc.result">
                    <ArrowDown v-if="!tc._expanded" /><ArrowUp v-else />
                  </el-icon>
                </div>
                <div v-if="tc._expanded || tc.status === 'running'" class="tool-call-body">
                  <div class="tool-section" v-if="tc.params">
                    <div class="tool-section-title">📥 调用参数</div>
                    <pre>{{ formatJSON(tc.params) }}</pre>
                  </div>
                  <div class="tool-section" v-if="tc.result">
                    <div class="tool-section-title">📤 返回结果</div>
                    <pre>{{ formatJSON(tc.result) }}</pre>
                  </div>
                </div>
              </div>
            </div>
            <div class="message-body">
              <span v-if="streamingContent" v-html="renderMarkdown(streamingContent)"></span>
              <span v-else-if="currentToolCalls.length === 0" class="typing-indicator">
                <span class="dot"></span>
                <span class="dot"></span>
                <span class="dot"></span>
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- 输入区域 -->
      <div class="input-container">
        <!-- 快捷指令面板 -->
        <div v-if="showQuickCommands" class="quick-commands-panel">
          <div class="quick-commands-title">快捷指令</div>
          <div
            v-for="(cmd, idx) in filteredQuickCommands"
            :key="cmd.command"
            class="quick-command-item"
            :class="{ active: quickCommandIndex === idx }"
            @click="selectQuickCommand(cmd)"
          >
            <span class="cmd-icon">{{ cmd.icon }}</span>
            <div class="cmd-info">
              <div class="cmd-name">{{ cmd.command }}</div>
              <div class="cmd-desc">{{ cmd.description }}</div>
            </div>
          </div>
        </div>
        <div class="input-wrapper">
          <el-input
            v-model="inputMessage"
            type="textarea"
            :rows="2"
            :autosize="{ minRows: 1, maxRows: 6 }"
            placeholder="输入消息... (/ 快捷指令，Enter 发送，Shift+Enter 换行)"
            @keydown="handleKeydown"
            @input="handleInputChange"
            :disabled="isLoading"
          />
          <el-button
            class="send-btn"
            type="primary"
            :icon="Promotion"
            @click="sendMessage"
            :disabled="!inputMessage.trim() || isLoading"
            :loading="isLoading"
            circle
          />
        </div>
        <div class="input-footer">
          <span class="hint-text">输入 / 使用快捷指令 · AI 可能会产生不准确的信息，请注意核实</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onBeforeUnmount, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Delete, User, MagicStick, ChatLineRound, ChatDotRound,
  Promotion, Operation, ArrowDown, ArrowUp
} from '@element-plus/icons-vue'
import {
  getSessions, createSession, deleteSession, getSessionMessages,
  getModelList, sendMessage as apiSendMessage, getTemplates
} from '@/api/ai'

// 状态
const sessions = ref<any[]>([])
const currentSessionId = ref<number>(0)
const messages = ref<any[]>([])
const models = ref<any[]>([])
const selectedModelId = ref<number>(0)
const inputMessage = ref('')
const isLoading = ref(false)
const streamingContent = ref('')
const currentToolCalls = ref<any[]>([])
const messagesContainer = ref<HTMLElement>()
const templates = ref<any[]>([])
let ws: WebSocket | null = null

const currentSession = computed(() =>
  sessions.value.find(s => s.id === currentSessionId.value)
)

const quickActions = [
  '列出所有主机状态',
  '查看 K8s 集群健康状态',
  '最近有哪些告警？',
  '生成基础设施运营报告',
  '查看今天的操作日志统计',
]

// 快捷指令
const quickCommands = [
  { command: '/巡检', description: '生成基础设施综合巡检报告', icon: '📋', prompt: '帮我做一次完整的基础设施巡检，包括主机状态、K8s 集群、域名监控、安全风险分析' },
  { command: '/主机状态', description: '查看所有主机的运行状态', icon: '🖥', prompt: '列出所有主机的状态，包括在线/离线数量和资源使用情况' },
  { command: '/K8s诊断', description: '检查 Kubernetes 集群健康状态', icon: '☸', prompt: '检查所有 K8s 集群的健康状态，有没有异常的集群？' },
  { command: '/安全检查', description: '分析系统安全态势和风险', icon: '🔒', prompt: '帮我做一次安全态势检查，包括登录失败记录、离线主机、SSL 证书到期等' },
  { command: '/容量分析', description: '资源使用率分析和扩容建议', icon: '📊', prompt: '分析当前资源使用情况，哪些主机需要扩容？给出容量规划建议' },
  { command: '/操作日志', description: '今日操作日志统计分析', icon: '📝', prompt: '统计分析今天的操作日志，按模块和用户分类' },
  { command: '/告警汇总', description: '查看告警和监控状况', icon: '🔔', prompt: '汇总今天的告警情况和域名监控状态' },
  { command: '/云账号', description: '查看云平台账号和实例', icon: '☁', prompt: '列出所有云平台账号和已导入的云主机实例' },
]

const showQuickCommands = ref(false)
const quickCommandIndex = ref(0)
const filteredQuickCommands = computed(() => {
  const input = inputMessage.value
  if (!input.startsWith('/')) return quickCommands
  const search = input.slice(1).toLowerCase()
  return quickCommands.filter(c => c.command.toLowerCase().includes(search) || c.description.includes(search))
})

// 初始化
onMounted(async () => {
  await loadModels()
  await loadSessions()
  await loadTemplates()
  connectWebSocket()
})

onBeforeUnmount(() => {
  if (ws) {
    ws.close()
    ws = null
  }
})

// 加载模型列表
async function loadModels() {
  try {
    const data = await getModelList()
    // request.ts 拦截器成功时直接返回 response.data.data
    models.value = Array.isArray(data) ? data : (data?.list || [])
    const defaultModel = models.value.find((m: any) => m.isDefault)
    if (defaultModel) {
      selectedModelId.value = defaultModel.id
    } else if (models.value.length > 0) {
      selectedModelId.value = models.value[0].id
    }
  } catch (e) {
    // 静默处理
  }
}

// 加载会话列表
async function loadSessions() {
  try {
    const data = await getSessions()
    sessions.value = Array.isArray(data) ? data : []
  } catch (e) {
    // 静默处理
  }
}

// 创建新会话
async function createNewSession() {
  try {
    const session = await createSession({ modelId: selectedModelId.value })
    if (session && session.id) {
      sessions.value.unshift(session)
      currentSessionId.value = session.id
      messages.value = []
    }
  } catch (e) {
    ElMessage.error('创建会话失败')
  }
}

// 切换会话
async function switchSession(id: number) {
  currentSessionId.value = id
  await loadMessages(id)
}

// 加载会话消息
async function loadMessages(sessionId: number) {
  try {
    const data = await getSessionMessages(sessionId)
    const msgList = Array.isArray(data) ? data : []
    messages.value = msgList.map((m: any) => {
      let toolCalls: any[] = []
      if (m.toolCalls) {
        try {
          toolCalls = typeof m.toolCalls === 'string' ? JSON.parse(m.toolCalls) : m.toolCalls
        } catch { toolCalls = [] }
      }
      // 给每个 tool call 添加 _expanded 属性
      toolCalls = (toolCalls || []).map((tc: any) => ({ ...tc, _expanded: false }))
      return { ...m, toolCalls }
    })
    scrollToBottom()
  } catch (e) {
    // 静默处理
  }
}

// 删除会话
async function handleDeleteSession(id: number) {
  try {
    await ElMessageBox.confirm('确定删除该对话？', '提示', {
      type: 'warning'
    })
    await deleteSession(id)
    sessions.value = sessions.value.filter(s => s.id !== id)
    if (currentSessionId.value === id) {
      currentSessionId.value = 0
      messages.value = []
    }
  } catch (e) {
    // 取消或错误
  }
}

// WebSocket 连接
function connectWebSocket() {
  const token = localStorage.getItem('token') || ''
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  ws = new WebSocket(`${protocol}//${host}/api/v1/plugins/ai/chat/ws?token=${token}`)

  ws.onmessage = (event) => {
    const data = JSON.parse(event.data)
    handleWSEvent(data)
  }

  ws.onerror = () => {
    // WebSocket 连接失败，回退到 HTTP 模式
    ws = null
  }

  ws.onclose = () => {
    ws = null
  }
}

// 处理 WebSocket 事件
function handleWSEvent(event: any) {
  switch (event.type) {
    case 'session_created':
      sessions.value.unshift(event.session)
      currentSessionId.value = event.session.id
      break

    case 'text_delta':
      streamingContent.value += event.content
      scrollToBottom()
      break

    case 'tool_call_start':
      currentToolCalls.value.push({
        toolName: event.toolName,
        params: event.toolParams,
        riskLevel: event.riskLevel || 'low',
        status: 'running',
        startTime: Date.now(),
      })
      scrollToBottom()
      break

    case 'tool_call_result': {
      // 按名称匹配最后一个 running 状态的 tool call
      const tc = [...currentToolCalls.value].reverse().find(
        t => t.toolName === event.toolName && t.status === 'running'
      ) || currentToolCalls.value[currentToolCalls.value.length - 1]
      if (tc) {
        tc.result = event.toolResult
        tc.status = event.toolResult?.includes('"error"') ? 'error' : 'success'
        tc.duration = Date.now() - (tc.startTime || Date.now())
      }
      scrollToBottom()
      break
    }

    case 'message_end':
      // 将流式内容合并为正式消息
      if (streamingContent.value) {
        messages.value.push({
          id: Date.now(),
          role: 'assistant',
          content: streamingContent.value,
          toolCalls: currentToolCalls.value.map(tc => ({ ...tc, _expanded: false })),
        })
      }
      streamingContent.value = ''
      currentToolCalls.value = []
      isLoading.value = false
      scrollToBottom()
      loadSessions() // 刷新会话标题
      break

    case 'error':
      ElMessage.error(event.error || '请求处理失败')
      isLoading.value = false
      streamingContent.value = ''
      currentToolCalls.value = []
      break
  }
}

// 发送消息
async function sendMessage() {
  const content = inputMessage.value.trim()
  if (!content || isLoading.value) return

  // 如果没有选中会话，先创建一个
  if (!currentSessionId.value) {
    await createNewSession()
    if (!currentSessionId.value) return
  }

  // 添加用户消息到界面
  messages.value.push({
    id: Date.now(),
    role: 'user',
    content,
  })
  inputMessage.value = ''
  isLoading.value = true
  streamingContent.value = ''
  scrollToBottom()

  // 优先使用 WebSocket
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({
      type: 'message',
      sessionId: currentSessionId.value,
      content,
      modelId: selectedModelId.value,
    }))
  } else {
    // 回退到 HTTP
    try {
      const data = await apiSendMessage({
        sessionId: currentSessionId.value,
        content,
        modelId: selectedModelId.value,
      })
      messages.value.push({
        id: Date.now(),
        role: 'assistant',
        content: data?.content || '',
        toolCalls: [],
      })
    } catch (e: any) {
      ElMessage.error(e?.message || '请求失败')
    }
    isLoading.value = false
    scrollToBottom()
    loadSessions()
  }
}

// 加载对话模板
async function loadTemplates() {
  try {
    const data = await getTemplates()
    templates.value = Array.isArray(data) ? data : []
  } catch (e) {
    // 静默处理
  }
}

// 使用模板
function useTemplate(tpl: any) {
  inputMessage.value = tpl.prompt
  sendMessage()
}

// 快捷操作
function sendQuickAction(action: string) {
  inputMessage.value = action
  sendMessage()
}

// 快捷指令输入检测
function handleInputChange() {
  if (inputMessage.value === '/') {
    showQuickCommands.value = true
    quickCommandIndex.value = 0
  } else if (inputMessage.value.startsWith('/') && inputMessage.value.length > 1) {
    showQuickCommands.value = filteredQuickCommands.value.length > 0
    quickCommandIndex.value = 0
  } else {
    showQuickCommands.value = false
  }
}

function selectQuickCommand(cmd: any) {
  inputMessage.value = cmd.prompt
  showQuickCommands.value = false
  sendMessage()
}

// 键盘事件
function handleKeydown(e: KeyboardEvent) {
  // 快捷指令导航
  if (showQuickCommands.value) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      quickCommandIndex.value = Math.min(quickCommandIndex.value + 1, filteredQuickCommands.value.length - 1)
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      quickCommandIndex.value = Math.max(quickCommandIndex.value - 1, 0)
      return
    }
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      if (filteredQuickCommands.value.length > 0) {
        selectQuickCommand(filteredQuickCommands.value[quickCommandIndex.value])
      }
      return
    }
    if (e.key === 'Escape') {
      showQuickCommands.value = false
      return
    }
  }

  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

// 滚动到底部
function scrollToBottom() {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

// Markdown 渲染（简单实现）
function renderMarkdown(content: string): string {
  if (!content) return ''
  let html = content
  // 代码块
  html = html.replace(/```(\w*)\n([\s\S]*?)```/g, '<pre class="code-block"><code class="language-$1">$2</code></pre>')
  // 行内代码
  html = html.replace(/`([^`]+)`/g, '<code class="inline-code">$1</code>')
  // 粗体
  html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
  // 斜体
  html = html.replace(/\*(.*?)\*/g, '<em>$1</em>')
  // 表格
  html = html.replace(/\n\|(.+)\|\n\|[-| :]+\|\n((?:\|.+\|\n?)+)/g, (_match, header, body) => {
    const headers = header.split('|').map((h: string) => `<th>${h.trim()}</th>`).join('')
    const rows = body.trim().split('\n').map((row: string) => {
      const cells = row.replace(/^\||\|$/g, '').split('|').map((c: string) => `<td>${c.trim()}</td>`).join('')
      return `<tr>${cells}</tr>`
    }).join('')
    return `<table class="md-table"><thead><tr>${headers}</tr></thead><tbody>${rows}</tbody></table>`
  })
  // 标题
  html = html.replace(/^### (.+)$/gm, '<h4>$1</h4>')
  html = html.replace(/^## (.+)$/gm, '<h3>$1</h3>')
  html = html.replace(/^# (.+)$/gm, '<h2>$1</h2>')
  // 列表
  html = html.replace(/^- (.+)$/gm, '<li>$1</li>')
  html = html.replace(/(<li>.*<\/li>\n?)+/g, '<ul>$&</ul>')
  // 有序列表
  html = html.replace(/^\d+\. (.+)$/gm, '<li>$1</li>')
  // 换行
  html = html.replace(/\n/g, '<br>')
  return html
}

// 格式化 JSON
function formatJSON(data: any): string {
  if (typeof data === 'string') {
    try {
      return JSON.stringify(JSON.parse(data), null, 2)
    } catch {
      return data
    }
  }
  return JSON.stringify(data, null, 2)
}
</script>

<style scoped>
.ai-chat-container {
  display: flex;
  height: calc(100vh - 60px);
  background: #f5f7fa;
}

/* 左侧会话列表 */
.session-sidebar {
  width: 260px;
  background: #fff;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  padding: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.session-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.session-item {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  margin-bottom: 4px;
  transition: background 0.2s;
}

.session-item:hover {
  background: #f5f7fa;
}

.session-item.active {
  background: #ecf5ff;
  color: #409eff;
}

.session-title {
  flex: 1;
  margin-left: 8px;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-delete {
  opacity: 0;
  color: #909399;
  transition: opacity 0.2s;
}

.session-item:hover .session-delete {
  opacity: 1;
}

.session-delete:hover {
  color: #f56c6c;
}

/* 主对话区域 */
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
}

.chat-title {
  font-size: 16px;
  font-weight: 600;
}

/* 消息容器 */
.messages-container {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.welcome-screen {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #606266;
}

.welcome-screen h2 {
  margin: 16px 0 8px;
  color: #303133;
}

.welcome-screen p {
  color: #909399;
  margin-bottom: 24px;
}

/* 对话模板 */
.template-section {
  margin-bottom: 24px;
  width: 100%;
  max-width: 680px;
}

.template-title {
  font-size: 14px;
  color: #909399;
  margin-bottom: 12px;
  text-align: center;
}

.template-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 10px;
}

.template-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
}

.template-card:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.15);
  transform: translateY(-1px);
}

.tpl-icon {
  font-size: 24px;
}

.tpl-name {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.tpl-desc {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}

.quick-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
  max-width: 600px;
}

.quick-action-tag {
  cursor: pointer;
  border-radius: 16px !important;
  transition: all 0.2s;
}

.quick-action-tag:hover {
  background: #409eff !important;
  color: #fff !important;
  border-color: #409eff !important;
}

/* 消息 */
.message-wrapper {
  display: flex;
  margin-bottom: 20px;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.message-wrapper.user {
  flex-direction: row-reverse;
}

.message-avatar {
  flex-shrink: 0;
}

.message-content {
  max-width: 75%;
  margin: 0 12px;
}

.message-role {
  font-size: 12px;
  color: #909399;
  margin-bottom: 4px;
}

.message-wrapper.user .message-role {
  text-align: right;
}

.message-body {
  background: #fff;
  padding: 12px 16px;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  line-height: 1.6;
  word-break: break-word;
}

.message-wrapper.user .message-body {
  background: #409eff;
  color: #fff;
}

.message-body :deep(pre.code-block) {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 8px 0;
  font-size: 13px;
}

.message-body :deep(code.inline-code) {
  background: #f0f0f0;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
}

.message-body :deep(table.md-table) {
  border-collapse: collapse;
  width: 100%;
  margin: 8px 0;
}

.message-body :deep(table.md-table th),
.message-body :deep(table.md-table td) {
  border: 1px solid #dcdfe6;
  padding: 8px 12px;
  text-align: left;
}

.message-body :deep(table.md-table th) {
  background: #f5f7fa;
  font-weight: 600;
}

/* 工具调用卡片 */
.tool-calls {
  margin-top: 12px;
  background: #f8f9fb;
  border: 1px solid #e8ecf0;
  border-radius: 10px;
  padding: 12px;
}

.tool-calls-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: #606266;
  margin-bottom: 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e8ecf0;
}

.tool-call-card {
  background: #ffffff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  margin-bottom: 6px;
  overflow: hidden;
  transition: all 0.2s ease;
}

.tool-call-card:last-child {
  margin-bottom: 0;
}

.tool-call-card.status-running {
  border-color: #e6a23c;
  box-shadow: 0 0 0 1px rgba(230, 162, 60, 0.15);
}

.tool-call-card.status-success {
  border-color: #67c23a;
}

.tool-call-card.status-error {
  border-color: #f56c6c;
}

.tool-call-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  cursor: pointer;
  font-size: 13px;
  transition: background 0.2s;
}

.tool-call-header:hover {
  background: #f5f7fa;
}

.tool-call-step {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: #409eff;
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.tool-call-card.status-success .tool-call-step {
  background: #67c23a;
}

.tool-call-card.status-error .tool-call-step {
  background: #f56c6c;
}

.tool-call-card.status-running .tool-call-step {
  background: #e6a23c;
}

.tool-name {
  font-weight: 600;
  flex: 1;
  font-family: 'Monaco', 'Menlo', monospace;
  color: #303133;
}

.status-tag {
  font-size: 11px !important;
}

.running-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #e6a23c;
  margin-right: 4px;
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.expand-icon {
  color: #909399;
  transition: transform 0.2s;
}

.tool-call-body {
  padding: 0 12px 12px;
  border-top: 1px solid #f0f0f0;
}

.tool-section {
  margin-top: 8px;
}

.tool-section-title {
  font-size: 12px;
  color: #606266;
  font-weight: 500;
  margin-bottom: 4px;
}

.tool-section pre {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 10px 12px;
  border-radius: 6px;
  font-size: 12px;
  overflow-x: auto;
  max-height: 200px;
  overflow-y: auto;
  line-height: 1.5;
  margin: 0;
}

/* 流式工具调用区域特殊样式 */
.streaming-tools {
  border-color: #e6a23c;
  background: #fdf6ec;
}

/* 正在输入 */
.typing-indicator {
  display: inline-flex;
  gap: 4px;
  align-items: center;
}

.typing-indicator .dot {
  width: 8px;
  height: 8px;
  background: #c0c4cc;
  border-radius: 50%;
  animation: bounce 1.4s infinite ease-in-out;
}

.typing-indicator .dot:nth-child(1) { animation-delay: -0.32s; }
.typing-indicator .dot:nth-child(2) { animation-delay: -0.16s; }

@keyframes bounce {
  0%, 80%, 100% { transform: scale(0); }
  40% { transform: scale(1); }
}

/* 输入区域 */
.input-container {
  position: relative;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #e4e7ed;
}

.input-wrapper {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  max-width: 800px;
  margin: 0 auto;
}

.input-wrapper :deep(.el-textarea__inner) {
  border-radius: 12px;
  padding: 10px 16px;
  resize: none;
}

.send-btn {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
}

.input-footer {
  text-align: center;
  margin-top: 8px;
}

.hint-text {
  font-size: 12px;
  color: #c0c4cc;
}

/* 快捷指令面板 */
.quick-commands-panel {
  position: absolute;
  bottom: 100%;
  left: 0;
  right: 0;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  box-shadow: 0 -4px 12px rgba(0, 0, 0, 0.1);
  padding: 8px;
  max-height: 300px;
  overflow-y: auto;
  margin-bottom: 8px;
  z-index: 10;
}

.quick-commands-title {
  font-size: 12px;
  color: #909399;
  padding: 4px 8px 8px;
  border-bottom: 1px solid #f0f0f0;
  margin-bottom: 4px;
}

.quick-command-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
}

.quick-command-item:hover,
.quick-command-item.active {
  background: #ecf5ff;
}

.cmd-icon {
  font-size: 18px;
  width: 28px;
  text-align: center;
}

.cmd-info {
  flex: 1;
}

.cmd-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.cmd-desc {
  font-size: 12px;
  color: #909399;
}

/* 空会话 */
.no-sessions {
  padding: 40px 0;
}

</style>
