<template>
  <div class="thread-list-panel">
    <div v-if="!attached" class="not-attached">
      <el-empty description="请先选择Pod并连接到进程">
        <template #image>
          <el-icon :size="60" color="#909399"><Connection /></el-icon>
        </template>
      </el-empty>
    </div>

    <div v-else class="panel-content" v-loading="loading">
      <!-- 状态统计栏 -->
      <div class="status-bar">
        <div class="status-info">
          <span class="total-label">总量:</span>
          <span class="total-value">{{ threads.length }}</span>
          <span class="divider">|</span>
          <span class="status-label">状态分布</span>
          <el-tag
            v-for="(count, state) in stateStats"
            :key="state"
            :type="getStateTagType(state as string)"
            size="small"
            :class="['state-tag', { active: selectedState === state || selectedState === '' }]"
            @click="filterByState(state as string)"
          >
            {{ state }} {{ count }}
          </el-tag>
        </div>
        <el-button type="primary" size="small" @click="loadThreads" :loading="loading">
          <el-icon><Refresh /></el-icon> 更新
        </el-button>
      </div>

      <!-- 线程表格 -->
      <div class="table-section">
        <el-table
          :data="displayThreads"
          stripe
          size="small"
          :header-cell-style="{ background: '#f5f7fa', color: '#606266' }"
          style="width: 100%"
          @row-click="handleRowClick"
          :row-class-name="() => 'clickable-row'"
        >
          <el-table-column prop="id" label="ID" width="60" align="center" />
          <el-table-column prop="name" label="名称" show-overflow-tooltip />
          <el-table-column prop="group" label="Group" width="80" align="center" />
          <el-table-column prop="priority" label="Priority" width="80" align="center" />
          <el-table-column prop="state" label="State" width="130" align="center">
            <template #default="{ row }">
              <el-tag :type="getStateTagType(row.state)" size="small" effect="light">{{ row.state }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="cpu" label="CPU(%)" width="80" align="right" />
          <el-table-column prop="time" label="Time(秒)" width="90" align="right" />
          <el-table-column prop="interrupted" label="Interrupted" width="100" align="center">
            <template #default="{ row }">
              <span :class="row.interrupted ? 'text-danger' : 'text-muted'">{{ row.interrupted ? 'true' : 'false' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="daemon" label="Daemon" width="80" align="center">
            <template #default="{ row }">
              <span :class="row.daemon ? 'text-primary' : 'text-muted'">{{ row.daemon ? 'true' : 'false' }}</span>
            </template>
          </el-table-column>
        </el-table>

        <!-- 空状态 -->
        <el-empty v-if="displayThreads.length === 0 && !loading" description="暂无数据" :image-size="60">
          <template #default>
            <el-link type="primary" @click="loadThreads">查看更多</el-link>
          </template>
        </el-empty>

        <!-- 查看更多 -->
        <div v-if="!showAll && filteredThreads.length > pageSize" class="load-more">
          <el-link type="primary" @click="showAll = true">查看更多 (共 {{ filteredThreads.length }} 条)</el-link>
        </div>
      </div>

      <!-- 线程堆栈详情对话框 -->
      <el-dialog v-model="stackDialogVisible" :title="`线程堆栈 - ${currentThread?.name || ''}`" width="900px" top="5vh">
        <div class="stack-header" v-if="currentThread">
          <el-descriptions :column="4" size="small" border>
            <el-descriptions-item label="ID">{{ currentThread.id }}</el-descriptions-item>
            <el-descriptions-item label="名称">{{ currentThread.name }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="getStateTagType(currentThread.state)" size="small">{{ currentThread.state }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="CPU">{{ currentThread.cpu }}%</el-descriptions-item>
          </el-descriptions>
        </div>
        <div class="stack-content" v-loading="stackLoading">
          <pre>{{ currentStack || '暂无堆栈信息' }}</pre>
        </div>
      </el-dialog>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection, Refresh } from '@element-plus/icons-vue'
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
  group: string
  priority: string
  state: string
  cpu: string
  time: string
  interrupted: boolean
  daemon: boolean
}

const loading = ref(false)
const stackLoading = ref(false)
const threads = ref<ThreadInfo[]>([])
const rawOutput = ref('')
const selectedState = ref('')
const showAll = ref(false)
const pageSize = 20

const stackDialogVisible = ref(false)
const currentThread = ref<ThreadInfo | null>(null)
const currentStack = ref('')

// 状态统计
const stateStats = computed(() => {
  const stats: Record<string, number> = {
    NEW: 0,
    RUNNABLE: 0,
    BLOCKED: 0,
    WAITING: 0,
    TIMED_WAITING: 0,
    TERMINATED: 0
  }
  threads.value.forEach(t => {
    if (stats[t.state] !== undefined) {
      stats[t.state]++
    } else {
      stats[t.state] = 1
    }
  })
  // 只返回有数据的状态
  return Object.fromEntries(Object.entries(stats).filter(([_, v]) => v > 0))
})

// 按状态过滤的线程
const filteredThreads = computed(() => {
  if (!selectedState.value) {
    return threads.value
  }
  return threads.value.filter(t => t.state === selectedState.value)
})

// 显示的线程（分页）
const displayThreads = computed(() => {
  if (showAll.value) {
    return filteredThreads.value
  }
  return filteredThreads.value.slice(0, pageSize)
})

// 获取状态标签类型
const getStateTagType = (state: string): string => {
  const types: Record<string, string> = {
    'RUNNABLE': 'success',
    'BLOCKED': 'danger',
    'WAITING': 'warning',
    'TIMED_WAITING': 'primary',
    'NEW': 'info',
    'TERMINATED': 'info'
  }
  return types[state] || 'info'
}

// 按状态过滤
const filterByState = (state: string) => {
  if (selectedState.value === state) {
    selectedState.value = ''
  } else {
    selectedState.value = state
  }
  showAll.value = false
}

// 解析线程输出
const parseThreadOutput = (output: string): ThreadInfo[] => {
  const threads: ThreadInfo[] = []
  const lines = output.split('\n')

  // 查找表头行
  let headerFound = false
  for (const line of lines) {
    const trimmedLine = line.trim()

    // 跳过空行和信息行
    if (!trimmedLine || trimmedLine.startsWith('[INFO]') || trimmedLine.startsWith('[arthas@')) {
      continue
    }

    // 检查是否是表头
    if (trimmedLine.startsWith('ID') && trimmedLine.includes('NAME')) {
      headerFound = true
      continue
    }

    // 解析数据行
    if (headerFound) {
      // 线程行格式: ID NAME GROUP PRIORITY STATE CPU DELTA_TIME TIME INTERRUPTED DAEMON
      const parts = trimmedLine.split(/\s+/)
      if (parts.length >= 8) {
        const id = parts[0]
        // 验证ID是数字
        if (!/^\d+$/.test(id)) continue

        // 从后往前解析固定字段
        const n = parts.length
        const daemon = parts[n - 1] === 'true'
        const interrupted = parts[n - 2] === 'true'
        const time = parts[n - 3]
        const deltaTime = parts[n - 4]
        const cpu = parts[n - 5]
        const state = parts[n - 6]
        const priority = parts[n - 7]
        const group = parts[n - 8]

        // 名称是ID和group之间的部分
        const nameEndIdx = n - 8
        const name = parts.slice(1, nameEndIdx).join(' ')

        threads.push({
          id,
          name,
          group,
          priority,
          state: normalizeState(state),
          cpu,
          time,
          interrupted,
          daemon
        })
      }
    }
  }

  return threads
}

// 标准化状态名
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
const loadThreads = async () => {
  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    return
  }

  loading.value = true
  showAll.value = false
  selectedState.value = ''

  try {
    const res = await getThreadList({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId
    })

    const output = typeof res === 'string' ? res : (res?.data || '')
    rawOutput.value = output
    threads.value = parseThreadOutput(output)

    if (threads.value.length === 0 && output) {
    }
  } catch (error: any) {
    ElMessage.error('获取线程列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 点击行查看堆栈
const handleRowClick = async (row: ThreadInfo) => {
  currentThread.value = row
  stackDialogVisible.value = true
  await loadThreadStack(row.id)
}

// 加载线程堆栈
const loadThreadStack = async (threadId: string) => {
  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    return
  }

  stackLoading.value = true
  try {
    const res = await getThreadStack({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId,
      threadId
    })
    currentStack.value = typeof res === 'string' ? res : (res?.data || '暂无堆栈信息')
  } catch (error: any) {
    ElMessage.error('获取线程堆栈失败: ' + (error.message || '未知错误'))
    currentStack.value = '获取失败: ' + (error.message || '未知错误')
  } finally {
    stackLoading.value = false
  }
}

watch(() => props.attached, (newVal) => {
  if (newVal) {
    loadThreads()
  } else {
    threads.value = []
    rawOutput.value = ''
  }
})
</script>

<style scoped>
.thread-list-panel {
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

/* 状态栏 */
.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 0;
  border: 1px solid #e9ecef;
}

.status-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.total-label {
  font-size: 13px;
  color: #606266;
}

.total-value {
  font-size: 16px;
  font-weight: 600;
  color: #409eff;
  min-width: 30px;
}

.divider {
  color: #dcdfe6;
  margin: 0 4px;
}

.status-label {
  font-size: 13px;
  color: #606266;
  margin-right: 4px;
}

.state-tag {
  cursor: pointer;
  transition: all 0.2s;
}

.state-tag:hover {
  transform: scale(1.05);
}

/* 表格 */
.table-section {
  background: #fff;
  border-radius: 0;
  border: 1px solid #ebeef5;
  overflow: hidden;
}

.text-danger { color: #f56c6c; }
.text-primary { color: #409eff; }
.text-muted { color: #c0c4cc; }

:deep(.clickable-row) {
  cursor: pointer;
}

:deep(.clickable-row:hover) {
  background-color: #f5f7fa !important;
}

/* 查看更多 */
.load-more {
  text-align: center;
  padding: 16px;
  border-top: 1px solid #ebeef5;
}

/* 堆栈对话框 */
.stack-header {
  margin-bottom: 16px;
}

.stack-content {
  background: #1e1e1e;
  border-radius: 0;
  padding: 16px;
  max-height: 500px;
  overflow: auto;
}

.stack-content pre {
  margin: 0;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
