<template>
  <div class="feishu-page">
    <div class="toolbar">
      <div class="toolbar-text">
        <h3>飞书渠道</h3>
        <p>配置飞书应用 `App ID/App Secret`，复用现有 AI 会话、模型和高风险确认链路。</p>
        <div class="toolbar-tips">
          <span>会话控制口令：`新对话` / `重置上下文` / `清空上下文` / `开始新会话`</span>
          <span>辅助命令：`状态` 查看当前会话、`压缩上下文` 手动生成摘要、`停止` 中断当前生成任务。</span>
          <span>自动轮换规则可在下方渠道配置中调整，达到阈值后会自动切换到新会话。</span>
        </div>
      </div>
      <el-button type="primary" :icon="Plus" @click="openCreateDialog">新增飞书渠道</el-button>
    </div>

    <div v-if="feishuChannels.length > 0" class="channel-grid">
      <el-card v-for="channel in feishuChannels" :key="channel.id" class="channel-card" shadow="hover">
        <template #header>
          <div class="card-header">
            <div>
              <div class="card-title">
                {{ channel.name }}
                <el-tag size="small" :type="getStatusTagType(channel.status)">{{ getStatusLabel(channel.status) }}</el-tag>
                <el-tag size="small" :type="channel.enabled ? 'success' : 'info'">
                  {{ channel.enabled ? '已启用' : '未启用' }}
                </el-tag>
              </div>
              <div class="card-subtitle">App ID: {{ channel.appId || '-' }}</div>
            </div>
            <el-dropdown trigger="click">
              <el-icon class="more-btn"><MoreFilled /></el-icon>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="editChannel(channel)">
                    <el-icon><Edit /></el-icon> 编辑
                  </el-dropdown-item>
                  <el-dropdown-item @click="handleTest(channel)" :disabled="actionLoading[channel.id || 0]">
                    <el-icon><Connection /></el-icon> 测试连接
                  </el-dropdown-item>
                  <el-dropdown-item @click="handleStart(channel)" :disabled="actionLoading[channel.id || 0]">
                    <el-icon><VideoPlay /></el-icon> 启动
                  </el-dropdown-item>
                  <el-dropdown-item @click="handleStop(channel)" :disabled="actionLoading[channel.id || 0]">
                    <el-icon><VideoPause /></el-icon> 停止
                  </el-dropdown-item>
                  <el-dropdown-item @click="handleReconnect(channel)" :disabled="actionLoading[channel.id || 0]">
                    <el-icon><Refresh /></el-icon> 重连
                  </el-dropdown-item>
                  <el-dropdown-item divided style="color: #f56c6c" @click="handleDelete(channel)">
                    <el-icon><Delete /></el-icon> 删除
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </template>

        <div class="info-list">
          <div class="info-item">
            <span class="label">默认模型</span>
            <span class="value">{{ getModelName(channel.defaultModelId) }}</span>
          </div>
          <div class="info-item">
            <span class="label">执行身份</span>
            <span class="value">{{ getUserName(channel.executeAsUserId) }}</span>
          </div>
          <div class="info-item">
            <span class="label">连接时间</span>
            <span class="value">{{ formatDateTime(channel.lastConnectedAt) }}</span>
          </div>
          <div class="info-item">
            <span class="label">自动轮换</span>
            <span class="value">{{ formatConversationPolicy(channel.config) }}</span>
          </div>
          <div class="info-item info-item-column">
            <span class="label">控制命令</span>
            <span class="value command-summary">{{ formatCommandSummary(channel.config) }}</span>
          </div>
        </div>

        <el-alert
          v-if="channel.lastError"
          class="error-alert"
          type="error"
          :closable="false"
          :title="channel.lastError"
          show-icon
        />

        <div class="card-actions">
          <el-button size="small" @click="handleTest(channel)" :loading="actionLoading[channel.id || 0]">测试</el-button>
          <el-button size="small" type="primary" @click="handleStart(channel)" :loading="actionLoading[channel.id || 0]">启动</el-button>
          <el-button size="small" @click="handleStop(channel)" :loading="actionLoading[channel.id || 0]">停止</el-button>
          <el-button size="small" @click="handleReconnect(channel)" :loading="actionLoading[channel.id || 0]">重连</el-button>
        </div>
      </el-card>
    </div>

    <el-card v-else shadow="never">
      <el-empty description="暂未配置飞书聊天渠道">
        <el-button type="primary" @click="openCreateDialog">立即配置</el-button>
      </el-empty>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="isEdit ? '编辑飞书渠道' : '新增飞书渠道'" width="620px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item label="渠道名称" prop="name">
          <el-input v-model="form.name" placeholder="例如：飞书应用机器人" />
        </el-form-item>
        <el-form-item label="App ID" prop="appId">
          <el-input v-model="form.appId" placeholder="输入飞书应用 App ID" />
        </el-form-item>
        <el-form-item label="App Secret" prop="appSecret">
          <el-input
            v-model="form.appSecret"
            type="password"
            show-password
            :placeholder="isEdit ? '留空表示保持不变' : '输入飞书应用 App Secret'"
          />
        </el-form-item>
        <el-form-item label="默认模型">
          <el-select v-model="form.defaultModelId" clearable filterable placeholder="不选则使用系统默认模型" style="width: 100%">
            <el-option v-for="model in modelOptions" :key="model.id" :label="model.name" :value="model.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="执行身份" prop="executeAsUserId">
          <el-select v-model="form.executeAsUserId" filterable placeholder="选择飞书消息映射到哪个 MOM 用户执行" style="width: 100%">
            <el-option v-for="user in userOptions" :key="user.id" :label="user.label" :value="user.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用配置">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-divider content-position="left">对话轮换策略</el-divider>
        <el-form-item label="消息数阈值">
          <div class="policy-row">
            <el-input-number v-model="conversationPolicy.sessionMaxMessages" :min="0" :max="500" :step="10" />
            <span class="policy-tip">单会话消息数达到该值时自动开启新会话，`0` 表示关闭</span>
          </div>
        </el-form-item>
        <el-form-item label="空闲阈值">
          <div class="policy-row">
            <el-input-number v-model="conversationPolicy.sessionIdleHours" :min="0" :max="720" :step="1" />
            <span class="policy-tip">距离上次对话超过多少小时后自动开启新会话，`0` 表示关闭</span>
          </div>
        </el-form-item>
        <el-form-item label="会话寿命">
          <div class="policy-row">
            <el-input-number v-model="conversationPolicy.sessionMaxAgeDays" :min="0" :max="365" :step="1" />
            <span class="policy-tip">同一飞书会话连续使用多少天后自动轮换，`0` 表示关闭</span>
          </div>
        </el-form-item>
        <el-divider content-position="left">会话控制命令</el-divider>
        <el-form-item label="新会话命令">
          <div class="command-row">
            <el-input
              v-model="commandText.reset"
              type="textarea"
              :rows="2"
              placeholder="每行一个命令，例如：新对话"
            />
            <span class="policy-tip">补充手动开启新会话的命令，默认命令仍然保留。</span>
          </div>
        </el-form-item>
        <el-form-item label="状态命令">
          <div class="command-row">
            <el-input
              v-model="commandText.status"
              type="textarea"
              :rows="2"
              placeholder="每行一个命令，例如：状态"
            />
            <span class="policy-tip">触发查看当前会话状态的命令，默认命令仍然保留。</span>
          </div>
        </el-form-item>
        <el-form-item label="压缩命令">
          <div class="command-row">
            <el-input
              v-model="commandText.compact"
              type="textarea"
              :rows="2"
              placeholder="每行一个命令，例如：压缩上下文"
            />
            <span class="policy-tip">触发手动摘要压缩的命令，默认命令仍然保留。</span>
          </div>
        </el-form-item>
        <el-form-item label="停止命令">
          <div class="command-row">
            <el-input
              v-model="commandText.stop"
              type="textarea"
              :rows="2"
              placeholder="每行一个命令，例如：停止"
            />
            <span class="policy-tip">触发停止当前生成任务的命令，默认命令仍然保留。</span>
          </div>
        </el-form-item>
        <el-alert
          class="policy-alert"
          type="info"
          :closable="false"
          show-icon
          title="飞书支持会话控制命令：新对话 / 重置上下文 / 清空上下文 / 开始新会话；同时支持 状态 / 压缩上下文 / 停止。"
        />
        <el-form-item label="扩展配置">
          <el-input
            v-model="configText"
            type="textarea"
            :rows="4"
            placeholder='预留给后续企业微信/钉钉统一扩展，不包含上面的会话轮换字段，例如 {"tenantKey":"xxx"}'
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  MoreFilled,
  Edit,
  Connection,
  Delete,
  VideoPlay,
  VideoPause,
  Refresh,
} from '@element-plus/icons-vue'
import type { AIChannelConfig, AIChannelExtraConfig } from '@/api/ai-channel'
import {
  createChannel,
  deleteChannel,
  getChannelList,
  reconnectChannel,
  startChannel,
  stopChannel,
  testChannel,
  updateChannel,
} from '@/api/ai-channel'
import { getModelList } from '@/api/ai'
import { getUserList } from '@/api/user'

const channels = ref<AIChannelConfig[]>([])
const modelOptions = ref<any[]>([])
const userOptions = ref<{ id: number; label: string }[]>([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref<number | null>(null)
const submitting = ref(false)
const actionLoading = ref<Record<number, boolean>>({})
const formRef = ref()
const configText = ref('{}')
const defaultConversationPolicy = {
  sessionMaxMessages: 60,
  sessionIdleHours: 24,
  sessionMaxAgeDays: 7,
}
const conversationPolicy = ref({
  ...defaultConversationPolicy,
})
const commandText = ref({
  reset: '',
  status: '',
  compact: '',
  stop: '',
})

const form = ref<AIChannelConfig>({
  name: '',
  channelType: 'feishu',
  enabled: true,
  appId: '',
  appSecret: '',
  defaultModelId: undefined,
  executeAsUserId: undefined,
  config: {},
})

const rules = {
  name: [{ required: true, message: '请输入渠道名称', trigger: 'blur' }],
  appId: [{ required: true, message: '请输入 App ID', trigger: 'blur' }],
  executeAsUserId: [{ required: true, message: '请选择执行身份', trigger: 'change' }],
  appSecret: [
    {
      validator: (_rule: any, value: string, callback: (error?: Error) => void) => {
        if (!isEdit.value && !value) {
          callback(new Error('请输入 App Secret'))
          return
        }
        callback()
      },
      trigger: 'blur',
    },
  ],
}

const feishuChannels = computed(() => channels.value.filter(item => item.channelType === 'feishu'))

onMounted(async () => {
  await Promise.all([loadChannels(), loadModels(), loadUsers()])
})

async function loadChannels() {
  const data: any = await getChannelList()
  channels.value = Array.isArray(data) ? data : []
}

async function loadModels() {
  const data: any = await getModelList()
  modelOptions.value = Array.isArray(data) ? data : (data?.list || [])
}

async function loadUsers() {
  const data: any = await getUserList({ page: 1, pageSize: 200 })
  const list = Array.isArray(data) ? data : (data?.list || [])
  userOptions.value = list
    .map((item: any) => {
      const id = item.ID || item.id
      return {
        id,
        label: `${item.realName || item.username} (${item.username})`,
      }
    })
    .filter((item: { id: number; label: string }) => !!item.id)
}

function openCreateDialog() {
  isEdit.value = false
  editId.value = null
  form.value = {
    name: '',
    channelType: 'feishu',
    enabled: true,
    appId: '',
    appSecret: '',
    defaultModelId: undefined,
    executeAsUserId: undefined,
    config: {},
  }
  conversationPolicy.value = { ...defaultConversationPolicy }
  commandText.value = {
    reset: '',
    status: '',
    compact: '',
    stop: '',
  }
  configText.value = '{}'
  dialogVisible.value = true
}

function editChannel(channel: AIChannelConfig) {
  isEdit.value = true
  editId.value = channel.id || null
  form.value = {
    ...channel,
    channelType: 'feishu',
    appSecret: '',
    config: channel.config || {},
  }
  conversationPolicy.value = normalizeConversationPolicy(channel.config)
  commandText.value = normalizeCommandText(channel.config)
  configText.value = JSON.stringify(extractExtraConfig(channel.config), null, 2)
  dialogVisible.value = true
}

async function submitForm() {
  if (!formRef.value) return
  await formRef.value.validate()

  let parsedConfig: AIChannelExtraConfig = {}
  try {
    parsedConfig = configText.value.trim() ? JSON.parse(configText.value) : {}
  } catch {
    ElMessage.error('扩展配置必须是合法 JSON')
    return
  }

  submitting.value = true
  try {
    const finalConfig: AIChannelExtraConfig = {
      ...parsedConfig,
      sessionMaxMessages: normalizePolicyValue(conversationPolicy.value.sessionMaxMessages),
      sessionIdleHours: normalizePolicyValue(conversationPolicy.value.sessionIdleHours),
      sessionMaxAgeDays: normalizePolicyValue(conversationPolicy.value.sessionMaxAgeDays),
      sessionResetCommands: parseCommandText(commandText.value.reset),
      sessionStatusCommands: parseCommandText(commandText.value.status),
      sessionCompactCommands: parseCommandText(commandText.value.compact),
      sessionStopCommands: parseCommandText(commandText.value.stop),
    }
    const payload: AIChannelConfig = {
      ...form.value,
      channelType: 'feishu',
      config: finalConfig,
    }
    if (isEdit.value && editId.value) {
      await updateChannel(editId.value, payload)
      ElMessage.success('飞书渠道已更新')
    } else {
      await createChannel(payload)
      ElMessage.success('飞书渠道已创建')
    }
    dialogVisible.value = false
    await loadChannels()
  } finally {
    submitting.value = false
  }
}

async function withAction(channel: AIChannelConfig, action: () => Promise<any>) {
  const id = channel.id || 0
  actionLoading.value[id] = true
  try {
    await action()
    await loadChannels()
  } finally {
    actionLoading.value[id] = false
  }
}

async function handleTest(channel: AIChannelConfig) {
  await withAction(channel, async () => {
    await testChannel(channel.id || 0)
    ElMessage.success('飞书渠道测试成功')
  })
}

async function handleStart(channel: AIChannelConfig) {
  await withAction(channel, async () => {
    await startChannel(channel.id || 0)
    ElMessage.success('飞书渠道已启动')
  })
}

async function handleStop(channel: AIChannelConfig) {
  await withAction(channel, async () => {
    await stopChannel(channel.id || 0)
    ElMessage.success('飞书渠道已停止')
  })
}

async function handleReconnect(channel: AIChannelConfig) {
  await withAction(channel, async () => {
    await reconnectChannel(channel.id || 0)
    ElMessage.success('飞书渠道已重连')
  })
}

async function handleDelete(channel: AIChannelConfig) {
  await ElMessageBox.confirm(`确认删除渠道「${channel.name}」吗？`, '提示', { type: 'warning' })
  await withAction(channel, async () => {
    await deleteChannel(channel.id || 0)
    ElMessage.success('飞书渠道已删除')
  })
}

function getStatusLabel(status?: string) {
  if (status === 'running') return '运行中'
  if (status === 'error') return '异常'
  return '已停止'
}

function getStatusTagType(status?: string) {
  if (status === 'running') return 'success'
  if (status === 'error') return 'danger'
  return 'info'
}

function getModelName(modelId?: number) {
  if (!modelId) return '系统默认模型'
  return modelOptions.value.find(item => item.id === modelId)?.name || `模型 #${modelId}`
}

function getUserName(userId?: number) {
  if (!userId) return '-'
  return userOptions.value.find(item => item.id === userId)?.label || `用户 #${userId}`
}

function formatDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function normalizeConversationPolicy(config?: AIChannelExtraConfig) {
  return {
    sessionMaxMessages: normalizePolicyValue(config?.sessionMaxMessages, defaultConversationPolicy.sessionMaxMessages),
    sessionIdleHours: normalizePolicyValue(config?.sessionIdleHours, defaultConversationPolicy.sessionIdleHours),
    sessionMaxAgeDays: normalizePolicyValue(config?.sessionMaxAgeDays, defaultConversationPolicy.sessionMaxAgeDays),
  }
}

function normalizeCommandText(config?: AIChannelExtraConfig) {
  return {
    reset: joinCommands(config?.sessionResetCommands),
    status: joinCommands(config?.sessionStatusCommands),
    compact: joinCommands(config?.sessionCompactCommands),
    stop: joinCommands(config?.sessionStopCommands),
  }
}

function extractExtraConfig(config?: AIChannelExtraConfig) {
  const extraConfig: AIChannelExtraConfig = { ...(config || {}) }
  delete extraConfig.sessionMaxMessages
  delete extraConfig.sessionIdleHours
  delete extraConfig.sessionMaxAgeDays
  delete extraConfig.sessionResetCommands
  delete extraConfig.sessionStatusCommands
  delete extraConfig.sessionCompactCommands
  delete extraConfig.sessionStopCommands
  return extraConfig
}

function normalizePolicyValue(value: unknown, fallback = 0) {
  const num = Number(value)
  if (!Number.isFinite(num)) return fallback
  return Math.max(0, Math.trunc(num))
}

function formatConversationPolicy(config?: AIChannelExtraConfig) {
  const policy = normalizeConversationPolicy(config)
  const parts: string[] = []
  if (policy.sessionMaxMessages > 0) parts.push(`${policy.sessionMaxMessages} 条消息`)
  if (policy.sessionIdleHours > 0) parts.push(`${policy.sessionIdleHours} 小时空闲`)
  if (policy.sessionMaxAgeDays > 0) parts.push(`${policy.sessionMaxAgeDays} 天寿命`)
  return parts.length > 0 ? parts.join(' / ') : '已关闭'
}

function formatCommandSummary(config?: AIChannelExtraConfig) {
  const parts: string[] = []
  if ((config?.sessionResetCommands || []).length > 0) parts.push(`新会话 ${config?.sessionResetCommands?.length} 条`)
  if ((config?.sessionStatusCommands || []).length > 0) parts.push(`状态 ${config?.sessionStatusCommands?.length} 条`)
  if ((config?.sessionCompactCommands || []).length > 0) parts.push(`压缩 ${config?.sessionCompactCommands?.length} 条`)
  if ((config?.sessionStopCommands || []).length > 0) parts.push(`停止 ${config?.sessionStopCommands?.length} 条`)
  return parts.length > 0 ? parts.join(' / ') : '使用默认命令'
}

function parseCommandText(value: string) {
  return value
    .split('\n')
    .map(item => item.trim())
    .filter(Boolean)
}

function joinCommands(list?: string[]) {
  return Array.isArray(list) ? list.join('\n') : ''
}
</script>

<style scoped>
.feishu-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.toolbar-text h3 {
  margin: 0;
  font-size: 20px;
  color: #1f2937;
}

.toolbar-text p {
  margin: 6px 0 0;
  color: #6b7280;
}

.toolbar-tips {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 10px;
  font-size: 12px;
  color: #6b7280;
}

.channel-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 16px;
}

.channel-card {
  border-radius: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.card-title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #111827;
}

.card-subtitle {
  margin-top: 6px;
  font-size: 12px;
  color: #6b7280;
}

.more-btn {
  cursor: pointer;
  color: #6b7280;
}

.info-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 14px;
}

.info-item-column {
  align-items: flex-start;
}

.info-item .label {
  color: #6b7280;
}

.info-item .value {
  color: #111827;
  text-align: right;
}

.command-summary {
  white-space: normal;
  line-height: 1.5;
}

.error-alert {
  margin-top: 16px;
}

.card-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 16px;
}

.policy-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
}

.command-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.policy-tip {
  color: #6b7280;
  font-size: 12px;
  line-height: 1.4;
}

.policy-alert {
  margin-bottom: 18px;
}
</style>
