<template>
  <div class="ai-chat-container">
    <!-- 左侧会话列表 -->
    <div class="session-sidebar">
      <div class="sidebar-header">
        <el-button type="primary" :icon="Plus" @click="createNewSession" style="width: 100%">
          新建对话
        </el-button>
        <div class="retention-bar" @click="showRetentionDialog = true">
          <el-icon><Timer /></el-icon>
          <span>历史保留 {{ retentionDays }} 天</span>
        </div>
      </div>
      <div class="session-list">
        <template v-for="group in groupedSessions" :key="group.label">
          <div class="session-date-label">{{ group.label }}</div>
          <div
            v-for="session in group.sessions"
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
        </template>
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
              <el-icon><CustomIcons name="Aibot" /></el-icon>
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
                  <el-tag v-if="tc.riskLevel === 'high' || tc.riskLevel === 'critical'" type="danger" size="small" effect="plain">
                    {{ tc.riskLevel === 'critical' ? '危险' : '高风险' }}
                  </el-tag>
                  <el-tag v-else-if="tc.riskLevel === 'medium'" type="warning" size="small" effect="plain">中风险</el-tag>
                  <el-tag v-else type="success" size="small" effect="plain">安全</el-tag>
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
              <el-icon><CustomIcons name="Aibot" /></el-icon>
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
                  <el-tag v-else-if="tc.riskLevel === 'medium'" type="warning" size="small" effect="plain">中风险</el-tag>
                  <el-tag v-else type="success" size="small" effect="plain">安全</el-tag>
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
              <div v-if="streamingContent" v-html="renderMarkdown(streamingContent)"></div>
              <div v-else-if="currentToolCalls.length === 0" class="typing-indicator">
                <span class="dot"></span>
                <span class="dot"></span>
                <span class="dot"></span>
              </div>
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
            v-if="isLoading"
            class="stop-btn"
            type="danger"
            @click="stopGeneration"
            circle
          >
            <el-icon><VideoPause /></el-icon>
          </el-button>
          <el-button
            v-else
            class="send-btn"
            type="primary"
            :icon="Promotion"
            @click="sendMessage"
            :disabled="!inputMessage.trim()"
            circle
          />
        </div>
        <div class="input-footer">
          <span class="hint-text">输入 / 使用快捷指令 · AI 可能会产生不准确的信息，请注意核实</span>
        </div>
      </div>
    </div>

    <!-- 历史保留设置弹窗 -->
    <el-dialog v-model="showRetentionDialog" title="历史会话保留设置" width="400px" destroy-on-close>
      <div class="retention-dialog-body">
        <p class="retention-desc">超过保留天数的历史会话将被自动清理，释放存储空间。</p>
        <el-form label-width="100px">
          <el-form-item label="保留天数">
            <el-input-number
              v-model="retentionDaysInput"
              :min="1"
              :max="365"
              :step="1"
              style="width: 180px"
            />
            <span style="margin-left: 8px; font-size: 12px; color: #909399;">天</span>
          </el-form-item>
          <el-form-item label="快捷选择">
            <el-radio-group v-model="retentionDaysInput" size="small">
              <el-radio-button :value="7">7 天</el-radio-button>
              <el-radio-button :value="14">14 天</el-radio-button>
              <el-radio-button :value="30">30 天</el-radio-button>
              <el-radio-button :value="90">90 天</el-radio-button>
              <el-radio-button :value="180">180 天</el-radio-button>
              <el-radio-button :value="365">365 天</el-radio-button>
            </el-radio-group>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="handleCleanupNow" type="warning" plain :loading="cleaningUp">立即清理</el-button>
        <el-button @click="showRetentionDialog = false">取消</el-button>
        <el-button type="primary" @click="saveRetention" :loading="savingRetention">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onBeforeUnmount, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, Delete, User, MagicStick, ChatLineRound, ChatDotRound,
  Promotion, Operation, ArrowDown, ArrowUp, Timer, VideoPause
} from '@element-plus/icons-vue'
import {
  getSessions, createSession, deleteSession, getSessionMessages,
  getModelList, sendMessage as apiSendMessage, getTemplates,
  getRetentionSettings, setRetentionSettings, cleanupSessions
} from '@/api/ai'
import CustomIcons from '@/components/icons/CustomIcons.vue'

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
const streamingSessionId = ref<number>(0) // 正在流式输出的会话 ID
const stoppedByUser = ref(false) // 用户是否主动停止了生成
// 按会话缓冲流式内容，切换回来时可恢复
const streamingBuffers = new Map<number, { content: string; toolCalls: any[] }>()
const messagesContainer = ref<HTMLElement>()
const templates = ref<any[]>([])
const retentionDays = ref(30)
const retentionDaysInput = ref(30)
const showRetentionDialog = ref(false)
const savingRetention = ref(false)
const cleaningUp = ref(false)
let ws: WebSocket | null = null

const currentSession = computed(() =>
  sessions.value.find(s => s.id === currentSessionId.value)
)

// 按日期分组的会话列表
const groupedSessions = computed(() => {
  if (sessions.value.length === 0) return []

  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)

  const groups: Map<string, { label: string; date: Date; sessions: any[] }> = new Map()

  for (const session of sessions.value) {
    const createdAt = new Date(session.createdAt || session.created_at || Date.now())
    const sessionDate = new Date(createdAt.getFullYear(), createdAt.getMonth(), createdAt.getDate())

    let label: string
    if (sessionDate.getTime() === today.getTime()) {
      label = '今天'
    } else if (sessionDate.getTime() === yesterday.getTime()) {
      label = '昨天'
    } else {
      const y = sessionDate.getFullYear()
      const m = String(sessionDate.getMonth() + 1).padStart(2, '0')
      const d = String(sessionDate.getDate()).padStart(2, '0')
      label = `${y}/${m}/${d}`
    }

    if (!groups.has(label)) {
      groups.set(label, { label, date: sessionDate, sessions: [] })
    }
    groups.get(label)!.sessions.push(session)
  }

  // 按日期降序排列
  return Array.from(groups.values()).sort((a, b) => b.date.getTime() - a.date.getTime())
})

const quickActions = [
  '列出所有主机状态',
  '查看域名监控状态',
  '查看 K8s 集群健康状态',
  '最近有哪些告警？',
  '列出所有网络设备',
  '生成基础设施运营报告',
  '查看今天的操作日志统计',
  '有哪些可用的凭证？',
]

// 快捷指令
const quickCommands = [
  { command: '/巡检', description: '生成基础设施综合巡检报告', icon: '📋', prompt: '帮我做一次完整的基础设施巡检，包括主机状态、K8s 集群、域名监控、网络设备、安全风险分析' },
  { command: '/主机状态', description: '查看所有主机的运行状态', icon: '🖥', prompt: '列出所有主机的状态，包括在线/离线数量和资源使用情况' },
  { command: '/添加主机', description: '通过对话添加新主机', icon: '➕', prompt: '我想添加一台新主机，请先帮我查看有哪些可用的凭证和分组' },
  { command: '/网络设备', description: '查看所有网络设备状态', icon: '🌐', prompt: '列出所有网络设备的状态，包括交换机、路由器、防火墙等' },
  { command: '/添加设备', description: '通过对话添加网络设备', icon: '🔌', prompt: '我想添加一台新网络设备，请先帮我查看有哪些可用的凭证和分组' },
  { command: '/域名监控', description: '查看域名监控和SSL证书', icon: '🌍', prompt: '查看所有域名的监控状态，有没有异常或者 SSL 即将过期的？' },
  { command: '/添加域名', description: '添加新的域名监控', icon: '📡', prompt: '我想添加一个新的域名监控' },
  { command: '/K8s诊断', description: '检查 Kubernetes 集群健康状态', icon: '☸', prompt: '检查所有 K8s 集群的健康状态，有没有异常的集群？' },
  { command: '/安全检查', description: '分析系统安全态势和风险', icon: '🔒', prompt: '帮我做一次安全态势检查，包括登录失败记录、离线主机、SSL 证书到期等' },
  { command: '/容量分析', description: '资源使用率分析和扩容建议', icon: '📊', prompt: '分析当前资源使用情况，哪些主机需要扩容？给出容量规划建议' },
  { command: '/操作日志', description: '今日操作日志统计分析', icon: '📝', prompt: '统计分析今天的操作日志，按模块和用户分类' },
  { command: '/告警汇总', description: '查看告警和监控状况', icon: '🔔', prompt: '汇总今天的告警情况和域名监控状态' },
  { command: '/云账号', description: '查看云平台账号和实例', icon: '☁', prompt: '列出所有云平台账号和已导入的云主机实例' },
  { command: '/任务历史', description: '查看最近的任务执行记录', icon: '📜', prompt: '查看最近执行过的任务和命令历史' },
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
  await loadRetention()
  connectWebSocket()
})

onBeforeUnmount(() => {
  wsManualClose = true
  if (wsReconnectTimer) {
    clearTimeout(wsReconnectTimer)
    wsReconnectTimer = null
  }
  if (ws) {
    ws.close()
    ws = null
  }
})

// 加载模型列表
async function loadModels() {
  try {
    const data = await getModelList()
    const allModels = Array.isArray(data) ? data : (data?.list || [])
    // 只显示启用的模型
    models.value = allModels.filter((m: any) => m.status === 1)
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
  streamingContent.value = ''
  currentToolCalls.value = []

  const isTargetStreaming = streamingSessionId.value === id && streamingSessionId.value !== 0
  isLoading.value = isTargetStreaming
  currentSessionId.value = id
  await loadMessages(id)

  // 如果目标会话正在流式中，从缓冲恢复已有内容
  // 缓冲始终保有完整的流式内容，直接用缓冲覆盖即可
  if (isTargetStreaming) {
    const buf = streamingBuffers.get(id)
    if (buf) {
      streamingContent.value = buf.content
      currentToolCalls.value = buf.toolCalls.map(tc => ({ ...tc }))
      scrollToBottom()
    }
  }
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

// WebSocket 连接（支持自动重连）
let wsReconnectTimer: ReturnType<typeof setTimeout> | null = null
let wsReconnectDelay = 1000 // 初始重连间隔 1 秒
const WS_MAX_RECONNECT_DELAY = 30000 // 最大重连间隔 30 秒
let wsManualClose = false // 是否为主动关闭（组件卸载）

function connectWebSocket() {
  if (wsManualClose) return
  const token = localStorage.getItem('token') || ''
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  ws = new WebSocket(`${protocol}//${host}/api/v1/plugins/ai/chat/ws?token=${token}`)

  ws.onopen = () => {
    // 连接成功，重置重连间隔
    wsReconnectDelay = 1000
  }

  ws.onmessage = (event) => {
    const data = JSON.parse(event.data)
    handleWSEvent(data)
  }

  ws.onerror = () => {
    // WebSocket 连接失败
    ws = null
  }

  ws.onclose = () => {
    ws = null
    // 非主动关闭时自动重连
    if (!wsManualClose) {
      wsReconnectTimer = setTimeout(() => {
        connectWebSocket()
        // 指数退避，最大 30 秒
        wsReconnectDelay = Math.min(wsReconnectDelay * 2, WS_MAX_RECONNECT_DELAY)
      }, wsReconnectDelay)
    }
  }
}

// 获取或创建会话的流式缓冲
function getBuffer(sid: number) {
  if (!streamingBuffers.has(sid)) {
    streamingBuffers.set(sid, { content: '', toolCalls: [] })
  }
  return streamingBuffers.get(sid)!
}

// 处理 WebSocket 事件
function handleWSEvent(event: any) {
  const eventSessionId = event.sessionId as number
  const isCurrentSession = !eventSessionId || eventSessionId === currentSessionId.value

  switch (event.type) {
    case 'session_created':
      sessions.value.unshift(event.session)
      currentSessionId.value = event.session.id
      streamingSessionId.value = event.session.id
      break

    case 'text_delta': {
      if (stoppedByUser.value) break
      // 无论是否当前会话，都写入缓冲
      if (eventSessionId) {
        getBuffer(eventSessionId).content += event.content
      }
      if (isCurrentSession) {
        streamingContent.value += event.content
        scrollToBottom()
      }
      break
    }

    case 'tool_call_start': {
      if (stoppedByUser.value) break
      const tcItem = {
        toolName: event.toolName,
        params: event.toolParams,
        riskLevel: event.riskLevel || 'low',
        status: 'running',
        startTime: Date.now(),
      }
      if (eventSessionId) {
        getBuffer(eventSessionId).toolCalls.push({ ...tcItem })
      }
      if (isCurrentSession) {
        currentToolCalls.value.push(tcItem)
        scrollToBottom()
      }
      break
    }

    case 'tool_call_result': {
      if (stoppedByUser.value) break
      // 更新缓冲中的 tool call 状态
      if (eventSessionId) {
        const buf = getBuffer(eventSessionId)
        const bufTc = [...buf.toolCalls].reverse().find(
          t => t.toolName === event.toolName && t.status === 'running'
        ) || buf.toolCalls[buf.toolCalls.length - 1]
        if (bufTc) {
          bufTc.result = event.toolResult
          bufTc.status = event.toolResult?.includes('"error"') ? 'error' : 'success'
          bufTc.duration = Date.now() - (bufTc.startTime || Date.now())
          if (event.riskLevel) bufTc.riskLevel = event.riskLevel
        }
      }
      if (isCurrentSession) {
        const tc = [...currentToolCalls.value].reverse().find(
          t => t.toolName === event.toolName && t.status === 'running'
        ) || currentToolCalls.value[currentToolCalls.value.length - 1]
        if (tc) {
          tc.result = event.toolResult
          tc.status = event.toolResult?.includes('"error"') ? 'error' : 'success'
          tc.duration = Date.now() - (tc.startTime || Date.now())
          if (event.riskLevel) tc.riskLevel = event.riskLevel
        }
        scrollToBottom()
      }
      break
    }

    case 'message_end': {
      const buf = eventSessionId ? streamingBuffers.get(eventSessionId) : null
      if (!stoppedByUser.value && isCurrentSession && streamingContent.value) {
        messages.value.push({
          id: Date.now(),
          role: 'assistant',
          content: streamingContent.value,
          toolCalls: currentToolCalls.value.map(tc => ({ ...tc, _expanded: false })),
        })
      } else if (!stoppedByUser.value && !isCurrentSession && buf && buf.content) {
        // 非当前会话完成：内容已保存到后端数据库，清理缓冲即可
        // 用户切回时 loadMessages 会从后端拉取完整消息
      }
      streamingContent.value = ''
      currentToolCalls.value = []
      if (eventSessionId) {
        streamingBuffers.delete(eventSessionId)
      }
      streamingSessionId.value = 0
      stoppedByUser.value = false
      if (isCurrentSession) {
        isLoading.value = false
      }
      scrollToBottom()
      loadSessions()
      break
    }

    case 'error':
      if (isCurrentSession && !stoppedByUser.value) {
        ElMessage.error(event.error || '请求处理失败')
      }
      if (isCurrentSession) {
        isLoading.value = false
      }
      streamingContent.value = ''
      currentToolCalls.value = []
      if (eventSessionId) {
        streamingBuffers.delete(eventSessionId)
      }
      streamingSessionId.value = 0
      stoppedByUser.value = false
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
  streamingSessionId.value = currentSessionId.value
  stoppedByUser.value = false
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

// 停止 AI 生成
function stopGeneration() {
  stoppedByUser.value = true
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'stop' }))
  }
  // 立即将已有的流式内容保存为消息，不等后端 message_end
  if (streamingContent.value) {
    messages.value.push({
      id: Date.now(),
      role: 'assistant',
      content: streamingContent.value + '\n\n[已停止]',
      toolCalls: currentToolCalls.value.map(tc => ({ ...tc, _expanded: false })),
    })
  }
  streamingContent.value = ''
  currentToolCalls.value = []
  if (streamingSessionId.value) {
    streamingBuffers.delete(streamingSessionId.value)
  }
  streamingSessionId.value = 0
  isLoading.value = false
  scrollToBottom()
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

// 历史保留设置
async function loadRetention() {
  try {
    const data: any = await getRetentionSettings()
    retentionDays.value = data?.retentionDays || 30
    retentionDaysInput.value = retentionDays.value
  } catch {
    // 静默
  }
}

async function saveRetention() {
  savingRetention.value = true
  try {
    await setRetentionSettings(retentionDaysInput.value)
    retentionDays.value = retentionDaysInput.value
    showRetentionDialog.value = false
    ElMessage.success(`已设置保留 ${retentionDays.value} 天`)
  } catch {
    ElMessage.error('保存失败')
  }
  savingRetention.value = false
}

async function handleCleanupNow() {
  try {
    await ElMessageBox.confirm(
      `将清理 ${retentionDaysInput.value} 天前的所有历史会话，此操作不可恢复，确认？`,
      '清理历史会话',
      { type: 'warning' }
    )
  } catch { return }

  cleaningUp.value = true
  try {
    // 先保存设置
    await setRetentionSettings(retentionDaysInput.value)
    retentionDays.value = retentionDaysInput.value

    const data: any = await cleanupSessions()
    const deleted = data?.deleted || 0
    if (deleted > 0) {
      ElMessage.success(`已清理 ${deleted} 个过期会话`)
      await loadSessions()
    } else {
      ElMessage.info('没有需要清理的过期会话')
    }
    showRetentionDialog.value = false
  } catch {
    ElMessage.error('清理失败')
  }
  cleaningUp.value = false
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

// Markdown 渲染（带 HTML 转义和深度思考标签修复）
function renderMarkdown(content: string): string {
  if (!content) return ''
  let html = content

  // 1. HTML 转义（防止 XSS 并且修正代码块中的 <TAG> 渲染）
  html = html.replace(/&/g, '&amp;')
             .replace(/</g, '&lt;')
             .replace(/>/g, '&gt;')

  // 2. 处理深度思考标签 (<think> ... </think>)
  html = html.replace(/&lt;think&gt;/g, '<div class="think-block"><div class="think-title">🤔 思考过程</div><div class="think-content">')
  html = html.replace(/&lt;\/think&gt;/g, '</div></div>')
  
  // 处理流式输出过程中只有头没有尾的情况，自动闭合
  const thinkOpen = (html.match(/<div class="think-block">/g) || []).length
  const thinkClose = (html.match(/<\/div><\/div>/g) || []).length
  if (thinkOpen > thinkClose) {
    html += '</div></div>'
  }

  // 3. 代码块
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
  html = html.replace(/^[-*] (.+)$/gm, '<li class="ul-item">$1</li>')
  // 有序列表
  html = html.replace(/^\d+\. (.+)$/gm, '<li class="ol-item">$1</li>')
  // 包装列表
  html = html.replace(/(<li class="ul-item">.*<\/li>\n?)+/g, '<ul>$&</ul>')
  html = html.replace(/(<li class="ol-item">.*<\/li>\n?)+/g, '<ol>$&</ol>')
  // 清理内部标签
  html = html.replace(/ class="ul-item"/g, '')
  html = html.replace(/ class="ol-item"/g, '')
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
/* 思考过程块样式 */
.think-block {
  margin: 12px 0;
  border-left: 3px solid #dcdfe6;
  background-color: #f8f9fa;
  border-radius: 4px;
  overflow: hidden;
}

.think-title {
  padding: 8px 12px;
  font-size: 13px;
  font-weight: 600;
  color: #606266;
  background-color: #ebeef5;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  align-items: center;
  gap: 6px;
}

.think-content {
  padding: 12px;
  font-size: 13px;
  color: #909399;
  line-height: 1.6;
}

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

.retention-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  padding: 6px 10px;
  font-size: 12px;
  color: #909399;
  background: #f5f7fa;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.retention-bar:hover {
  background: #ecf5ff;
  color: #409eff;
}

.retention-dialog-body .retention-desc {
  margin: 0 0 16px 0;
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
}

.session-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.session-date-label {
  font-size: 12px;
  color: #909399;
  padding: 10px 12px 4px;
  font-weight: 600;
  letter-spacing: 0.5px;
  position: sticky;
  top: 0;
  background: #fff;
  z-index: 1;
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
  padding: 12px 15px;
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

.message-body :deep(ul),
.message-body :deep(ol) {
  padding-left: 30px;
  margin: 12px 0;
}

.message-body :deep(li) {
  margin-bottom: 6px;
  line-height: 1.6;
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

.stop-btn {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  animation: pulse-stop 1.5s ease-in-out infinite;
}

@keyframes pulse-stop {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
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
