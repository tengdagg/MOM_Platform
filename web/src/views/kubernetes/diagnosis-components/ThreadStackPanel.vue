<template>
  <div class="thread-stack-panel">
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
        <div class="input-group">
          <span class="input-label">线程ID</span>
          <el-input
            v-model="threadId"
            placeholder="输入线程ID"
            style="width: 120px"
            size="small"
            clearable
            @keyup.enter="getThreadStackData"
          />
        </div>
        <el-button type="primary" size="small" @click="getThreadStackData" :loading="loading" :disabled="!threadId">
          <el-icon><Search /></el-icon> 查看堆栈
        </el-button>
        <el-divider direction="vertical" />
        <el-button size="small" @click="loadThreadList" :loading="loadingThreads">
          <el-icon><List /></el-icon> 加载线程列表
        </el-button>
        <el-button size="small" @click="getBlockedThreads" :loading="loadingBlocked">
          <el-icon><Warning /></el-icon> 阻塞线程
        </el-button>
        <el-button size="small" @click="getBusyThreads" :loading="loadingBusy">
          <el-icon><Cpu /></el-icon> 繁忙线程
        </el-button>
      </div>

      <div class="main-content">
        <!-- 左侧线程列表 -->
        <div class="thread-list-section" v-if="threads.length > 0">
          <div class="section-header">
            <span>线程列表</span>
            <el-input
              v-model="searchText"
              placeholder="搜索线程"
              size="small"
              clearable
              style="width: 150px"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
          </div>
          <div class="thread-list">
            <div
              v-for="thread in filteredThreads"
              :key="thread.id"
              class="thread-item"
              :class="{ active: threadId === thread.id, [thread.state?.toLowerCase()]: true }"
              @click="selectThread(thread)"
            >
              <span class="thread-id">{{ thread.id }}</span>
              <span class="thread-name" :title="thread.name">{{ thread.name }}</span>
              <el-tag :type="getStateType(thread.state)" size="small" effect="plain">
                {{ formatState(thread.state) }}
              </el-tag>
            </div>
          </div>
        </div>

        <!-- 右侧堆栈显示 -->
        <div class="stack-section">
          <div class="section-header" v-if="stackOutput">
            <span>堆栈信息</span>
            <span class="thread-info" v-if="currentThread">
              线程 #{{ currentThread.id }} - {{ currentThread.name }}
              <el-tag :type="getStateType(currentThread.state)" size="small" style="margin-left: 8px">
                {{ currentThread.state }}
              </el-tag>
            </span>
          </div>
          <div class="stack-output" v-if="stackOutput" v-loading="loading">
            <pre>{{ cleanStackOutput }}</pre>
          </div>
          <el-empty v-else-if="!loading" description="请输入线程ID或从左侧列表选择线程查看堆栈信息">
            <template #image>
              <el-icon :size="60" color="#c0c4cc"><Document /></el-icon>
            </template>
          </el-empty>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection, Search, List, Warning, Cpu, Document } from '@element-plus/icons-vue'
import { getThreadList, getThreadStack } from '@/api/arthas'

const props = defineProps<{
  clusterId: number | null
  namespace: string
  pod: string
  container: string
  processId: string
  attached: boolean
}>()

interface ThreadInfo {
  id: string
  name: string
  state: string
  cpu?: string
  group?: string
}

const loading = ref(false)
const loadingThreads = ref(false)
const loadingBlocked = ref(false)
const loadingBusy = ref(false)
const threadId = ref('')
const stackOutput = ref('')
const searchText = ref('')
const threads = ref<ThreadInfo[]>([])
const currentThread = ref<ThreadInfo | null>(null)

// 过滤后的线程列表
const filteredThreads = computed(() => {
  if (!searchText.value) {
    return threads.value
  }
  const keyword = searchText.value.toLowerCase()
  return threads.value.filter(t =>
    t.id.includes(keyword) ||
    t.name.toLowerCase().includes(keyword) ||
    t.state?.toLowerCase().includes(keyword)
  )
})

// 清理堆栈输出（移除 ANSI 转义码）
const cleanStackOutput = computed(() => {
  if (!stackOutput.value) return ''
  return stackOutput.value
    .replace(/\x1b\[[0-9;]*m/g, '')
    .replace(/\033\[[0-9;]*m/g, '')
    .replace(/\[\d+;\d+m/g, '')
    .replace(/\[\d+m/g, '')
    .replace(/\[0m/g, '')
    .replace(/\[m/g, '')
})

// 获取状态标签类型
const getStateType = (state: string): string => {
  if (!state) return 'info'
  const types: Record<string, string> = {
    'RUNNABLE': 'success',
    'RUNNING': 'success',
    'BLOCKED': 'danger',
    'WAITING': 'warning',
    'TIMED_WAITING': 'primary',
    'TIMED_WAIT': 'primary',
    'T_WAIT': 'primary',
    'NEW': 'info',
    'TERMINATED': 'info'
  }
  return types[state.toUpperCase()] || 'info'
}

// 格式化状态
const formatState = (state: string): string => {
  if (!state) return '-'
  const stateMap: Record<string, string> = {
    'TIMED_WAITING': 'T_WAIT',
    'TIMED_WAIT': 'T_WAIT'
  }
  return stateMap[state] || state
}

// 选择线程
const selectThread = (thread: ThreadInfo) => {
  threadId.value = thread.id
  currentThread.value = thread
  getThreadStackData()
}

// 解析线程列表输出
const parseThreadListOutput = (output: string): ThreadInfo[] => {
  const threadList: ThreadInfo[] = []
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

        // 从后往前解析
        const n = parts.length
        const state = normalizeState(parts[n - 6] || '')
        const name = parts.slice(1, n - 8).join(' ') || parts[1]

        threadList.push({
          id,
          name,
          state,
          cpu: parts[n - 5] || ''
        })
      }
    }
  }

  return threadList
}

// 标准化状态
const normalizeState = (state: string): string => {
  const stateMap: Record<string, string> = {
    'TIMED_': 'TIMED_WAITING',
    'WAITIN': 'WAITING',
    'RUNNAB': 'RUNNABLE',
    'BLOCKE': 'BLOCKED',
    'TERMIN': 'TERMINATED'
  }
  return stateMap[state] || state
}

// 加载线程列表
const loadThreadList = async () => {
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
    threads.value = parseThreadListOutput(output)

    if (threads.value.length === 0) {
      ElMessage.warning('未获取到线程列表')
    } else {
      ElMessage.success(`获取到 ${threads.value.length} 个线程`)
    }
  } catch (error: any) {
    ElMessage.error('获取线程列表失败: ' + (error.message || '未知错误'))
  } finally {
    loadingThreads.value = false
  }
}

// 获取线程堆栈
const getThreadStackData = async () => {
  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    return
  }

  if (!threadId.value) {
    ElMessage.warning('请输入线程ID')
    return
  }

  loading.value = true
  try {
    const res = await getThreadStack({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId,
      threadId: threadId.value
    })

    stackOutput.value = typeof res === 'string' ? res : (res?.data || '暂无堆栈信息')

    // 如果当前没有选中线程信息，尝试从线程列表中查找
    if (!currentThread.value) {
      currentThread.value = threads.value.find(t => t.id === threadId.value) || null
    }
  } catch (error: any) {
    ElMessage.error('获取线程堆栈失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 获取阻塞线程
const getBlockedThreads = async () => {
  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    return
  }

  loadingBlocked.value = true
  try {
    const res = await getThreadStack({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId,
      threadId: '-b' // Arthas thread -b 显示阻塞线程
    })

    stackOutput.value = typeof res === 'string' ? res : (res?.data || '暂无阻塞线程')
    currentThread.value = null
    threadId.value = ''
  } catch (error: any) {
    ElMessage.error('获取阻塞线程失败: ' + (error.message || '未知错误'))
  } finally {
    loadingBlocked.value = false
  }
}

// 获取繁忙线程
const getBusyThreads = async () => {
  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    return
  }

  loadingBusy.value = true
  try {
    const res = await getThreadStack({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId,
      threadId: '-n 5' // Arthas thread -n 5 显示最繁忙的5个线程
    })

    stackOutput.value = typeof res === 'string' ? res : (res?.data || '暂无繁忙线程数据')
    currentThread.value = null
    threadId.value = ''
  } catch (error: any) {
    ElMessage.error('获取繁忙线程失败: ' + (error.message || '未知错误'))
  } finally {
    loadingBusy.value = false
  }
}

watch(() => props.attached, (newVal) => {
  if (newVal) {
    loadThreadList()
  } else {
    threads.value = []
    stackOutput.value = ''
    currentThread.value = null
  }
})
</script>

<style scoped>
.thread-stack-panel {
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
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 0;
  border: 1px solid #e9ecef;
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

/* 主内容区 */
.main-content {
  display: flex;
  gap: 16px;
  min-height: 500px;
}

/* 线程列表区域 */
.thread-list-section {
  width: 320px;
  flex-shrink: 0;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 0;
  display: flex;
  flex-direction: column;
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

.thread-info {
  font-size: 12px;
  font-weight: normal;
  color: #606266;
}

.thread-list {
  flex: 1;
  overflow-y: auto;
  max-height: 450px;
}

.thread-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  cursor: pointer;
  border-bottom: 1px solid #f0f0f0;
  transition: all 0.2s;
}

.thread-item:hover {
  background: #f5f7fa;
}

.thread-item.active {
  background: #ecf5ff;
  border-left: 3px solid #409eff;
}

.thread-item.blocked {
  background: #fef0f0;
}

.thread-item.waiting {
  background: #fdf6ec;
}

.thread-id {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  color: #409eff;
  font-weight: 600;
  min-width: 35px;
}

.thread-name {
  flex: 1;
  font-size: 12px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 堆栈区域 */
.stack-section {
  flex: 1;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.stack-output {
  flex: 1;
  overflow: auto;
}

.stack-output pre {
  margin: 0;
  padding: 16px;
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.5;
  min-height: 100%;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 空状态 */
.stack-section :deep(.el-empty) {
  padding: 60px 0;
}

/* 响应式 */
@media (max-width: 992px) {
  .main-content {
    flex-direction: column;
  }

  .thread-list-section {
    width: 100%;
    max-height: 250px;
  }

  .thread-list {
    max-height: 180px;
  }
}
</style>
