<template>
  <div class="method-monitor-panel">
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
            <span class="input-label">类名</span>
            <el-input
              v-model="classPattern"
              placeholder="类名表达式 (如: com.example.service.*)"
              style="width: 300px"
              size="default"
              clearable
              :disabled="monitoring"
            >
              <template #prefix>
                <el-icon><Folder /></el-icon>
              </template>
            </el-input>
          </div>
          <div class="input-group">
            <span class="input-label">方法名</span>
            <el-input
              v-model="methodPattern"
              placeholder="方法名 (如: doSomething)"
              style="width: 200px"
              size="default"
              clearable
              :disabled="monitoring"
            >
              <template #prefix>
                <el-icon><Promotion /></el-icon>
              </template>
            </el-input>
          </div>
        </div>
        <div class="toolbar-row">
          <div class="input-group">
            <span class="input-label">统计周期</span>
            <el-input-number
              v-model="interval"
              :min="1"
              :max="60"
              size="default"
              :disabled="monitoring"
              style="width: 120px"
            />
            <span class="input-suffix">秒</span>
          </div>
          <div class="input-group">
            <span class="input-label">最大周期数</span>
            <el-input-number
              v-model="maxCycles"
              :min="1"
              :max="100"
              size="default"
              :disabled="monitoring"
              style="width: 120px"
            />
          </div>
          <div class="input-group">
            <span class="input-label">条件表达式</span>
            <el-input
              v-model="condition"
              placeholder="OGNL 条件 (可选)"
              style="width: 200px"
              size="default"
              clearable
              :disabled="monitoring"
            >
              <template #prefix>
                <el-icon><Filter /></el-icon>
              </template>
            </el-input>
          </div>
        </div>
        <div class="toolbar-row actions">
          <el-button
            type="primary"
            @click="startMonitor"
            :loading="starting"
            :disabled="monitoring || !classPattern || !methodPattern"
          >
            <el-icon><VideoPlay /></el-icon>
            {{ starting ? '启动中...' : '开始监控' }}
          </el-button>
          <el-button
            @click="stopMonitor"
            :disabled="!monitoring"
            type="danger"
          >
            <el-icon><VideoPause /></el-icon> 停止监控
          </el-button>
          <el-button @click="clearData">
            <el-icon><Delete /></el-icon> 清空数据
          </el-button>
          <el-divider direction="vertical" />
          <el-tag v-if="monitoring" type="success" effect="dark">
            <el-icon class="is-loading"><Loading /></el-icon>
            监控中...
          </el-tag>
          <el-tag v-else type="info">未监控</el-tag>
          <span class="cycle-count" v-if="monitorData.length > 0">
            已统计: {{ monitorData.length }} 个周期
          </span>
        </div>
      </div>

      <!-- 使用说明 -->
      <el-collapse v-model="showHelp" class="help-collapse">
        <el-collapse-item title="使用说明" name="help">
          <div class="help-content">
            <p><strong>monitor 命令</strong> 可以监控方法的执行统计信息，包括调用次数、成功率、平均响应时间等。</p>
            <ul>
              <li><strong>统计周期</strong>: 每隔多少秒输出一次统计结果</li>
              <li><strong>最大周期数</strong>: 统计多少个周期后自动停止</li>
              <li><strong>条件表达式</strong>: OGNL 表达式，只统计满足条件的调用</li>
            </ul>
            <div class="help-section">
              <h4>统计指标说明</h4>
              <ul>
                <li><strong>调用次数</strong>: 该周期内方法被调用的总次数</li>
                <li><strong>成功</strong>: 正常返回的次数</li>
                <li><strong>失败</strong>: 抛出异常的次数</li>
                <li><strong>平均RT</strong>: 平均响应时间 (毫秒)</li>
                <li><strong>RT范围</strong>: 最小/最大响应时间</li>
              </ul>
            </div>
            <p class="tip">提示: 监控高频调用的方法时，建议适当增大统计周期，减少输出频率。</p>
          </div>
        </el-collapse-item>
      </el-collapse>

      <!-- 统计数据表格 -->
      <div class="data-section">
        <div class="section-header">
          <span>监控数据</span>
          <span class="data-info">
            <el-tag size="small" type="info">{{ monitorData.length }} 条记录</el-tag>
          </span>
        </div>
        <el-table
          :data="monitorData"
          border
          stripe
          max-height="400"
          v-loading="starting"
          empty-text="等待监控数据..."
          class="monitor-table"
        >
          <el-table-column prop="timestamp" label="时间" width="180" fixed>
            <template #default="{ row }">
              <span class="timestamp">{{ row.timestamp }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="class" label="类名" min-width="250" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="class-name">{{ row.class }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="method" label="方法名" width="150" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="method-name">{{ row.method }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="total" label="调用次数" width="100" align="center">
            <template #default="{ row }">
              <span class="total-count">{{ row.total }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="success" label="成功" width="80" align="center">
            <template #default="{ row }">
              <span class="success-count">{{ row.success }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="fail" label="失败" width="80" align="center">
            <template #default="{ row }">
              <span :class="['fail-count', { 'has-fail': row.fail > 0 }]">{{ row.fail }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="failRate" label="失败率" width="90" align="center">
            <template #default="{ row }">
              <el-tag :type="getFailRateType(row.failRate)" size="small">{{ row.failRate }}%</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="avgRt" label="平均RT(ms)" width="110" align="center">
            <template #default="{ row }">
              <span :class="['avg-rt', { 'slow': row.avgRt > 1000 }]">{{ row.avgRt }}</span>
            </template>
          </el-table-column>
          <el-table-column label="RT范围(ms)" width="140" align="center">
            <template #default="{ row }">
              <span class="rt-range">{{ row.minRt }} ~ {{ row.maxRt }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- 原始输出 -->
      <el-collapse v-model="showRawOutput" class="raw-output-collapse">
        <el-collapse-item title="原始输出" name="raw">
          <div class="raw-output">
            <pre>{{ cleanOutput }}</pre>
          </div>
        </el-collapse-item>
      </el-collapse>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection, VideoPlay, VideoPause, Delete, Folder, Promotion, Filter, Loading } from '@element-plus/icons-vue'
import { createArthasWebSocket, type ArthasWSMessage } from '@/api/arthas'

const props = defineProps<{
  clusterId: number | null
  namespace: string
  pod: string
  container: string
  processId: string
  attached: boolean
}>()

// 表单数据
const classPattern = ref('')
const methodPattern = ref('')
const interval = ref(5)
const maxCycles = ref(20)
const condition = ref('')

// 状态
const monitoring = ref(false)
const starting = ref(false)
const showHelp = ref<string[]>([])
const showRawOutput = ref<string[]>([])
const rawOutput = ref('')  // 使用原始字符串

// 统计数据
interface MonitorRecord {
  timestamp: string
  class: string
  method: string
  total: number
  success: number
  fail: number
  failRate: string
  avgRt: number
  minRt: number
  maxRt: number
}

const monitorData = ref<MonitorRecord[]>([])

// WebSocket 连接
let ws: WebSocket | null = null

// 清理输出中的 ANSI 转义码
const cleanOutput = computed(() => {
  return rawOutput.value
    .replace(/\x1b\[[0-9;]*m/g, '')
    .replace(/\033\[[0-9;]*m/g, '')
    .replace(/\[\d+;\d+m/g, '')
    .replace(/\[\d+m/g, '')
    .replace(/\[0m/g, '')
    .replace(/\[m/g, '')
})

// 获取失败率标签类型
const getFailRateType = (rate: string): string => {
  const rateNum = parseFloat(rate)
  if (rateNum === 0) return 'success'
  if (rateNum < 5) return 'warning'
  return 'danger'
}

// 构建 monitor 命令
const buildMonitorCommand = (): string => {
  let cmd = `monitor ${classPattern.value} ${methodPattern.value}`

  // 添加条件表达式
  if (condition.value) {
    cmd += ` '${condition.value}'`
  }

  // 添加选项
  cmd += ` -c ${interval.value}` // 统计周期
  cmd += ` -n ${maxCycles.value}` // 最大周期数

  return cmd
}

// 解析 monitor 输出
const parseMonitorOutput = (content: string) => {
  // monitor 输出格式:
  // timestamp    class           method  total  success  fail  avg-rt(ms)  fail-rate  rt-min  rt-max
  // 2024-01-01 12:00:00  com.example.Service  method  100    99       1     10.5        1%       5      100

  const lines = content.split('\n')
  for (const line of lines) {
    const trimmedLine = line.trim()
      .replace(/\x1b\[[0-9;]*m/g, '')
      .replace(/\033\[[0-9;]*m/g, '')

    // 跳过表头和空行
    if (!trimmedLine || trimmedLine.startsWith('timestamp') ||
        trimmedLine.startsWith('[INFO]') || trimmedLine.startsWith('[arthas@') ||
        trimmedLine.includes('Affect(') || trimmedLine.includes('Press Q')) {
      continue
    }

    // 尝试解析数据行
    // 格式: 时间戳(2部分) 类名 方法名 total success fail avg-rt fail-rate
    const parts = trimmedLine.split(/\s+/)
    if (parts.length >= 8) {
      // 检查第一部分是否像日期
      if (/^\d{4}-\d{2}-\d{2}$/.test(parts[0])) {
        const record: MonitorRecord = {
          timestamp: `${parts[0]} ${parts[1]}`,
          class: parts[2] || '',
          method: parts[3] || '',
          total: parseInt(parts[4]) || 0,
          success: parseInt(parts[5]) || 0,
          fail: parseInt(parts[6]) || 0,
          avgRt: parseFloat(parts[7]) || 0,
          failRate: (parts[8] || '0%').replace('%', ''),
          minRt: 0,
          maxRt: 0
        }

        // 如果有更多字段，尝试解析 rt 范围
        if (parts.length >= 10) {
          record.minRt = parseFloat(parts[9]) || 0
          record.maxRt = parseFloat(parts[10]) || 0
        }

        // 计算失败率（如果没有提供）
        if (record.total > 0 && record.failRate === '0') {
          record.failRate = ((record.fail / record.total) * 100).toFixed(2)
        }

        monitorData.value.push(record)
      }
    }
  }
}

// 开始监控
const startMonitor = async () => {
  if (!classPattern.value || !methodPattern.value) {
    ElMessage.warning('请输入类名和方法名')
    return
  }

  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    ElMessage.warning('请先选择 Pod 和容器')
    return
  }

  starting.value = true
  monitorData.value = []
  rawOutput.value = ''

  try {
    // 创建 WebSocket 连接
    ws = createArthasWebSocket({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId
    })

    ws.onopen = () => {
      // 发送 monitor 命令
      const command = buildMonitorCommand()
      rawOutput.value = `[INFO] 执行命令: ${command}\n\n`

      const msg: ArthasWSMessage = {
        type: 'command',
        command: command
      }
      ws?.send(JSON.stringify(msg))
      monitoring.value = true
      starting.value = false
      ElMessage.success('开始监控')
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'output') {
          // 监控输出 - 直接追加到原始输出
          const content = data.content
          rawOutput.value += content

          // 解析并添加监控数据
          parseMonitorOutput(content)
        } else if (data.type === 'error') {
          rawOutput.value += `\n[ERROR] ${data.content}\n`
          ElMessage.error(data.content)
        } else if (data.type === 'info') {
          rawOutput.value += `[INFO] ${data.content}\n`
        }
      } catch (e) {
        // 如果不是 JSON，直接追加原始数据
        rawOutput.value += event.data
      }
    }

    ws.onerror = (error) => {
      rawOutput.value += '\n[ERROR] WebSocket 连接错误\n'
      monitoring.value = false
      starting.value = false
      ElMessage.error('WebSocket 连接失败')
    }

    ws.onclose = () => {
      monitoring.value = false
      starting.value = false
      rawOutput.value += '\n[INFO] 监控已停止\n'
    }

  } catch (error: any) {
    ElMessage.error('启动监控失败: ' + (error.message || '未知错误'))
    starting.value = false
  }
}

// 停止监控
const stopMonitor = () => {
  if (ws) {
    const msg: ArthasWSMessage = {
      type: 'stop'
    }
    ws.send(JSON.stringify(msg))
    ws.close()
    ws = null
  }
  monitoring.value = false
  rawOutput.value += '\n[INFO] 用户停止监控\n'
  ElMessage.info('已停止监控')
}

// 清空数据
const clearData = () => {
  monitorData.value = []
  rawOutput.value = ''
}

// 组件卸载时清理
onUnmounted(() => {
  if (ws) {
    ws.close()
    ws = null
  }
})

// 监听 attached 状态变化
watch(() => props.attached, (newVal) => {
  if (!newVal && ws) {
    ws.close()
    ws = null
    monitoring.value = false
  }
})
</script>

<style scoped>
.method-monitor-panel {
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
  background: #f8f9fa;
  border-radius: 0;
  padding: 16px;
  border: 1px solid #e9ecef;
}

.toolbar-row {
  display: flex;
  gap: 16px;
  align-items: center;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.toolbar-row:last-child {
  margin-bottom: 0;
}

.toolbar-row.actions {
  padding-top: 12px;
  border-top: 1px solid #e9ecef;
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
  font-weight: 500;
}

.input-suffix {
  font-size: 13px;
  color: #909399;
}

.cycle-count {
  font-size: 13px;
  color: #67c23a;
  font-weight: 500;
  margin-left: 8px;
}

/* 帮助折叠面板 */
.help-collapse {
  border: 1px solid #e4e7ed;
  border-radius: 0;
}

.help-collapse :deep(.el-collapse-item__header) {
  padding: 0 16px;
  font-size: 13px;
  color: #606266;
}

.help-content {
  padding: 0 8px;
  font-size: 13px;
  color: #606266;
  line-height: 1.8;
}

.help-section {
  margin: 12px 0;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 0;
}

.help-section h4 {
  margin: 0 0 8px 0;
  font-size: 13px;
  color: #303133;
}

.help-content ul {
  margin: 8px 0;
  padding-left: 20px;
}

.help-content .tip {
  background: #fdf6ec;
  padding: 8px 12px;
  border-radius: 0;
  border-left: 3px solid #e6a23c;
  margin-top: 12px;
}

/* 数据区域 */
.data-section {
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

.data-info {
  display: flex;
  gap: 8px;
  align-items: center;
}

/* 表格样式 */
.monitor-table {
  font-size: 13px;
}

.timestamp {
  font-family: 'Consolas', 'Monaco', monospace;
  color: #909399;
  font-size: 12px;
}

.class-name {
  font-family: 'Consolas', 'Monaco', monospace;
  color: #606266;
  font-size: 12px;
}

.method-name {
  font-family: 'Consolas', 'Monaco', monospace;
  color: #409eff;
  font-weight: 500;
}

.total-count {
  font-weight: 600;
  color: #303133;
}

.success-count {
  color: #67c23a;
  font-weight: 500;
}

.fail-count {
  color: #909399;
  font-weight: 500;
}

.fail-count.has-fail {
  color: #f56c6c;
}

.avg-rt {
  font-weight: 500;
  color: #303133;
}

.avg-rt.slow {
  color: #e6a23c;
}

.rt-range {
  font-size: 12px;
  color: #909399;
}

/* 原始输出折叠面板 */
.raw-output-collapse {
  border: 1px solid #e4e7ed;
  border-radius: 0;
}

.raw-output-collapse :deep(.el-collapse-item__header) {
  padding: 0 16px;
  font-size: 13px;
  color: #909399;
}

.raw-output {
  max-height: 300px;
  overflow: auto;
}

.raw-output pre {
  margin: 0;
  padding: 12px;
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 加载动画 */
.is-loading {
  animation: rotating 1s linear infinite;
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* 响应式 */
@media (max-width: 992px) {
  .toolbar-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .input-group {
    width: 100%;
  }

  .input-group :deep(.el-input) {
    width: 100% !important;
  }
}
</style>
