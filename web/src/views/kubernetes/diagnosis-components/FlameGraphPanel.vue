<template>
  <div class="flame-graph-panel">
    <div v-if="!attached" class="not-attached">
      <el-empty description="请先选择Pod并连接到进程">
        <template #image>
          <el-icon :size="60" color="#909399"><Connection /></el-icon>
        </template>
      </el-empty>
    </div>

    <div v-else class="panel-content">
      <!-- 工具栏 -->
      <div class="toolbar">
        <div class="toolbar-row">
          <div class="input-group">
            <span class="input-label">采样事件</span>
            <el-select v-model="eventType" size="small" style="width: 120px">
              <el-option label="CPU" value="cpu" />
              <el-option label="内存分配" value="alloc" />
              <el-option label="锁竞争" value="lock" />
              <el-option label="Wall Clock" value="wall" />
            </el-select>
          </div>

          <div class="input-group">
            <span class="input-label">采样时长</span>
            <el-input-number
              v-model="duration"
              :min="5"
              :max="300"
              size="small"
              style="width: 120px"
            />
            <span class="input-suffix">秒</span>
          </div>

          <div class="input-group">
            <span class="input-label">线程ID</span>
            <el-select
              v-model="threadId"
              placeholder="可选，采样指定线程"
              size="small"
              style="width: 200px"
              clearable
              filterable
              :loading="loadingThreads"
              @focus="loadThreadListForSelect"
            >
              <el-option
                v-for="thread in threadOptions"
                :key="thread.id"
                :label="`${thread.id} - ${thread.name}`"
                :value="thread.id"
              >
                <span style="float: left">{{ thread.id }}</span>
                <span style="float: right; color: #8492a6; font-size: 12px; max-width: 150px; overflow: hidden; text-overflow: ellipsis;">{{ thread.name }}</span>
              </el-option>
            </el-select>
          </div>

          <el-checkbox v-model="includeThreads" size="small">
            按线程分组
          </el-checkbox>

          <el-button
            type="primary"
            size="small"
            @click="startProfiling"
            :loading="loading"
            :disabled="loading"
          >
            <el-icon><VideoPlay /></el-icon>
            {{ loading ? `采样中 (${countdown}s)` : '开始采样' }}
          </el-button>

          <el-button
            size="small"
            @click="stopProfiling"
            :disabled="!loading"
            type="danger"
          >
            <el-icon><VideoPause /></el-icon>
            停止
          </el-button>
        </div>

        <div class="toolbar-tips">
          <el-icon><InfoFilled /></el-icon>
          <span>提示：火焰图采样期间会对应用产生轻微性能影响，建议采样时长 10-60 秒</span>
        </div>
      </div>

      <!-- 采样进度 -->
      <div class="progress-section" v-if="loading">
        <el-progress
          :percentage="progressPercent"
          :stroke-width="8"
          :format="formatProgress"
          status="warning"
        />
        <p class="progress-text">正在采样分析，请稍候...</p>
      </div>

      <!-- 火焰图显示 -->
      <div class="flame-graph-section" v-if="flameGraphHtml && !loading">
        <div class="section-header">
          <span>火焰图结果</span>
          <div class="header-actions">
            <el-tag :type="isEmptyFlameGraph ? 'warning' : 'info'" size="small">
              {{ eventTypeLabel }} | {{ lastDuration }}秒
              <span v-if="isEmptyFlameGraph"> (无数据)</span>
            </el-tag>
            <el-button size="small" @click="openInNewWindow">
              <el-icon><FullScreen /></el-icon>
              新窗口打开
            </el-button>
            <el-button size="small" @click="downloadHtml">
              <el-icon><Download /></el-icon>
              下载 HTML
            </el-button>
          </div>
        </div>
        <!-- 空火焰图警告 -->
        <el-alert
          v-if="isEmptyFlameGraph"
          title="采样数据为空"
          type="warning"
          :closable="false"
          style="margin: 12px"
        >
          <template #default>
            <div>采样期间没有捕获到 CPU 活动，可能的原因：</div>
            <ul style="margin: 8px 0 0 20px; padding: 0;">
              <li>指定的线程在采样期间处于空闲状态</li>
              <li>采样时长太短</li>
              <li>应用在采样期间没有活动</li>
            </ul>
            <div style="margin-top: 8px;">建议：清空「线程ID」字段，采样所有线程试试</div>
          </template>
        </el-alert>
        <div class="flame-graph-container">
          <iframe
            ref="flameIframe"
            :srcdoc="flameGraphHtml"
            frameborder="0"
            width="100%"
            height="600"
            sandbox="allow-scripts allow-same-origin"
          ></iframe>
        </div>
      </div>

      <!-- 空状态 -->
      <div class="empty-state" v-if="!flameGraphHtml && !loading">
        <el-empty description="点击「开始采样」生成火焰图">
          <template #image>
            <div class="flame-icon">
              <el-icon :size="80" color="#f56c6c"><Histogram /></el-icon>
            </div>
          </template>
        </el-empty>
        <div class="event-type-guide">
          <h4>采样事件类型说明</h4>
          <div class="guide-items">
            <div class="guide-item">
              <el-tag type="danger" effect="plain">CPU</el-tag>
              <span>分析 CPU 热点，查看哪些方法消耗最多 CPU 时间</span>
            </div>
            <div class="guide-item">
              <el-tag type="warning" effect="plain">内存分配</el-tag>
              <span>分析内存分配热点，查看哪些方法分配最多内存</span>
            </div>
            <div class="guide-item">
              <el-tag type="primary" effect="plain">锁竞争</el-tag>
              <span>分析锁竞争情况，查看线程阻塞等待的位置</span>
            </div>
            <div class="guide-item">
              <el-tag type="info" effect="plain">Wall Clock</el-tag>
              <span>分析墙钟时间，包括 I/O 等待、sleep 等</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 错误信息 -->
      <el-alert
        v-if="errorMessage"
        :title="errorMessage"
        type="error"
        show-icon
        closable
        @close="errorMessage = ''"
        style="margin-top: 16px"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Connection,
  VideoPlay,
  VideoPause,
  Download,
  FullScreen,
  InfoFilled,
  Histogram
} from '@element-plus/icons-vue'
import { generateFlameGraph, getThreadList } from '@/api/arthas'

const props = defineProps<{
  clusterId: number | null
  namespace: string
  pod: string
  container: string
  processId: string
  attached: boolean
}>()

const loading = ref(false)
const duration = ref(30)
const eventType = ref('cpu')
const threadId = ref('')
const includeThreads = ref(false)
const flameGraphHtml = ref('')
const errorMessage = ref('')
const countdown = ref(0)
const lastDuration = ref(0)
const flameIframe = ref<HTMLIFrameElement | null>(null)
const isEmptyFlameGraph = ref(false)
const loadingThreads = ref(false)
const threadOptions = ref<{id: string; name: string}[]>([])

let countdownTimer: ReturnType<typeof setInterval> | null = null

// 进度百分比
const progressPercent = computed(() => {
  if (!loading.value || duration.value === 0) return 0
  return Math.min(100, Math.round(((duration.value - countdown.value) / duration.value) * 100))
})

// 事件类型标签
const eventTypeLabel = computed(() => {
  const labels: Record<string, string> = {
    'cpu': 'CPU采样',
    'alloc': '内存分配',
    'lock': '锁竞争',
    'wall': 'Wall Clock'
  }
  return labels[eventType.value] || eventType.value
})

// 格式化进度
const formatProgress = (percentage: number) => {
  return `${percentage}%`
}

// 加载线程列表供选择
const loadThreadListForSelect = async () => {
  if (threadOptions.value.length > 0 || loadingThreads.value) {
    return // 已加载或正在加载
  }

  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    return
  }

  loadingThreads.value = true
  try {
    const res = await getThreadList({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId
    })

    const output = typeof res === 'string' ? res : (res?.data || '')
    threadOptions.value = parseThreadListOutput(output)
  } catch (error) {
  } finally {
    loadingThreads.value = false
  }
}

// 解析线程列表输出
const parseThreadListOutput = (output: string): {id: string; name: string}[] => {
  const threads: {id: string; name: string}[] = []
  const lines = output.split('\n')

  let headerFound = false
  for (const line of lines) {
    const trimmedLine = line.trim()
      .replace(/\x1b\[[0-9;]*m/g, '')
      .replace(/\033\[[0-9;]*m/g, '')

    if (!trimmedLine || trimmedLine.startsWith('[INFO]') || trimmedLine.startsWith('[arthas@')) {
      continue
    }

    if (trimmedLine.startsWith('ID') && trimmedLine.includes('NAME')) {
      headerFound = true
      continue
    }

    if (headerFound) {
      const parts = trimmedLine.split(/\s+/)
      if (parts.length >= 5) {
        const id = parts[0]
        if (!/^\d+$/.test(id)) continue

        // 从后往前解析，获取线程名
        const n = parts.length
        const name = parts.slice(1, n - 8).join(' ') || parts[1] || `Thread-${id}`

        threads.push({ id, name })
      }
    }
  }

  return threads
}

// 开始采样
const startProfiling = async () => {
  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    ElMessage.warning('请先选择完整的Pod信息')
    return
  }

  loading.value = true
  errorMessage.value = ''
  flameGraphHtml.value = ''
  isEmptyFlameGraph.value = false
  countdown.value = duration.value
  lastDuration.value = duration.value

  // 启动倒计时
  countdownTimer = setInterval(() => {
    if (countdown.value > 0) {
      countdown.value--
    }
  }, 1000)

  try {
    const params: any = {
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId,
      duration: String(duration.value),
      event: eventType.value
    }

    if (threadId.value) {
      params.threadId = threadId.value
    }

    if (includeThreads.value) {
      params.includeThreads = 'true'
    }

    const res = await generateFlameGraph(params)
    const output = typeof res === 'string' ? res : (res?.data || '')

    // 提取 HTML 内容
    const htmlContent = extractHtmlContent(output)

    if (htmlContent) {
      // 检查火焰图是否有实际数据（不只是空的头部）
      // async-profiler 生成的空火焰图只有头部，没有实际的火焰堆栈数据
      const hasData = htmlContent.includes('<rect') ||
                      htmlContent.includes('samples') ||
                      (htmlContent.includes('title') && htmlContent.length > 5000)

      flameGraphHtml.value = htmlContent

      if (!hasData && htmlContent.length < 3000) {
        // 火焰图可能是空的
        isEmptyFlameGraph.value = true
        ElMessage.warning('火焰图生成成功，但采样数据为空')
      } else {
        isEmptyFlameGraph.value = false
        ElMessage.success('火焰图生成成功')
      }
    } else {
      // 检查输出中是否有错误信息
      const cleanedOutput = output
        .replace(/\x1b\[[0-9;]*m/g, '')
        .replace(/\033\[[0-9;]*m/g, '')

      if (cleanedOutput.includes('[ERROR]')) {
        // 提取错误信息
        const errorMatch = cleanedOutput.match(/\[ERROR\][^\n]+/)
        errorMessage.value = errorMatch ? errorMatch[0] : '生成火焰图失败，请查看控制台日志'
      } else if (cleanedOutput.includes('profiler not started')) {
        errorMessage.value = 'Profiler 未启动，请检查进程是否支持 async-profiler'
      } else if (cleanedOutput.includes('Cannot attach to process')) {
        errorMessage.value = '无法附加到进程，请检查进程 ID 是否正确'
      } else {
        errorMessage.value = '未能生成火焰图，可能采样时间太短或进程活动不足'
      }
    }
  } catch (error: any) {
    errorMessage.value = '生成火焰图失败: ' + (error.message || '未知错误')
  } finally {
    loading.value = false
    if (countdownTimer) {
      clearInterval(countdownTimer)
      countdownTimer = null
    }
  }
}

// 停止采样
const stopProfiling = () => {
  // 目前只是清理状态，实际的profiler停止需要后端支持
  loading.value = false
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
  ElMessage.info('采样已中断')
}

// 从输出中提取 HTML 内容
const extractHtmlContent = (output: string): string => {
  // 移除 ANSI 转义码
  let cleaned = output
    .replace(/\x1b\[[0-9;]*m/g, '')
    .replace(/\033\[[0-9;]*m/g, '')
    .replace(/\[\d+;\d+m/g, '')
    .replace(/\[\d+m/g, '')

  // 首先尝试从标记中提取
  const startMarker = '---FLAMEGRAPH_START---'
  const endMarker = '---FLAMEGRAPH_END---'

  const markerStart = cleaned.indexOf(startMarker)
  const markerEnd = cleaned.indexOf(endMarker)

  if (markerStart !== -1 && markerEnd !== -1 && markerEnd > markerStart) {
    const htmlContent = cleaned.substring(markerStart + startMarker.length, markerEnd).trim()
    if (htmlContent) {
      return htmlContent
    }
  }

  // 回退：查找 HTML 内容
  // profiler 输出的 HTML 可能以 <!DOCTYPE html> 或 <html> 开头
  const htmlStart = cleaned.indexOf('<!DOCTYPE html>')
  const htmlStartAlt = cleaned.indexOf('<html')

  let startIndex = -1
  if (htmlStart !== -1) {
    startIndex = htmlStart
  } else if (htmlStartAlt !== -1) {
    startIndex = htmlStartAlt
  }

  if (startIndex === -1) {
    return ''
  }

  // 查找 HTML 结束
  const htmlEnd = cleaned.lastIndexOf('</html>')
  if (htmlEnd === -1) {
    return ''
  }

  return cleaned.substring(startIndex, htmlEnd + 7)
}

// 在新窗口打开
const openInNewWindow = () => {
  if (!flameGraphHtml.value) return

  const newWindow = window.open('', '_blank')
  if (newWindow) {
    newWindow.document.write(flameGraphHtml.value)
    newWindow.document.close()
  }
}

// 下载 HTML
const downloadHtml = () => {
  if (!flameGraphHtml.value) return

  const blob = new Blob([flameGraphHtml.value], { type: 'text/html' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `flamegraph-${eventType.value}-${Date.now()}.html`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

// 清理
onUnmounted(() => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
  }
})

watch(() => props.attached, (newVal) => {
  if (!newVal) {
    flameGraphHtml.value = ''
    errorMessage.value = ''
    isEmptyFlameGraph.value = false
    threadOptions.value = []
  }
})
</script>

<style scoped>
.flame-graph-panel {
  min-height: 400px;
}

.not-attached {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}

.panel-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 工具栏 */
.toolbar {
  padding: 16px;
  background: #f8f9fa;
  border-radius: 0;
  border: 1px solid #e9ecef;
}

.toolbar-row {
  display: flex;
  gap: 16px;
  align-items: center;
  flex-wrap: wrap;
}

.input-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.input-label {
  font-size: 13px;
  color: #606266;
  white-space: nowrap;
}

.input-suffix {
  font-size: 13px;
  color: #909399;
}

.toolbar-tips {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #e9ecef;
  font-size: 12px;
  color: #909399;
}

/* 进度 */
.progress-section {
  padding: 24px;
  background: #fff;
  border-radius: 0;
  border: 1px solid #ebeef5;
  text-align: center;
}

.progress-text {
  margin-top: 12px;
  font-size: 14px;
  color: #606266;
}

/* 火焰图区域 */
.flame-graph-section {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 0;
  overflow: hidden;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #f5f7fa;
  border-bottom: 1px solid #ebeef5;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.flame-graph-container {
  width: 100%;
  height: 600px;
  overflow: hidden;
}

.flame-graph-container iframe {
  width: 100%;
  height: 100%;
  border: none;
}

/* 空状态 */
.empty-state {
  padding: 40px 20px;
  background: #fff;
  border-radius: 0;
  border: 1px solid #ebeef5;
}

.flame-icon {
  display: flex;
  justify-content: center;
  opacity: 0.6;
}

.event-type-guide {
  max-width: 600px;
  margin: 24px auto 0;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 0;
}

.event-type-guide h4 {
  margin: 0 0 16px 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.guide-items {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.guide-item {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 13px;
  color: #606266;
}

.guide-item .el-tag {
  flex-shrink: 0;
  width: 80px;
  text-align: center;
}
</style>
