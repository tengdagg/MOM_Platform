<template>
  <div class="perf-counter-panel">
    <div v-if="!attached" class="not-attached">
      <el-empty description="请先选择Pod并连接到进程">
        <template #image>
          <el-icon :size="60" color="#909399"><Connection /></el-icon>
        </template>
      </el-empty>
    </div>

    <div v-else class="panel-content" v-loading="loading">
      <!-- 工具栏 -->
      <div class="toolbar">
        <el-button type="primary" size="small" @click="loadPerfCounters" :loading="loading">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
        <el-input
          v-model="searchText"
          placeholder="搜索计数器名称..."
          style="width: 280px"
          clearable
          size="small"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <span class="counter-count" v-if="perfCounters.length > 0">
          共 {{ filteredCounters.length }} / {{ perfCounters.length }} 项
        </span>
      </div>

      <!-- 计数器表格 -->
      <div class="table-section" v-if="filteredCounters.length > 0">
        <el-table
          :data="displayCounters"
          stripe
          size="small"
          :header-cell-style="{ background: '#f5f7fa', color: '#606266' }"
          style="width: 100%"
        >
          <el-table-column prop="name" label="计数器名称" min-width="300" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="counter-name">{{ row.name }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="value" label="值" width="200" align="right">
            <template #default="{ row }">
              <span class="counter-value" :class="{ 'is-number': isNumber(row.value) }">
                {{ formatValue(row.value) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="unit" label="单位" width="120" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.unit" size="small" type="info" effect="plain">{{ row.unit }}</el-tag>
              <span v-else class="text-muted">-</span>
            </template>
          </el-table-column>
        </el-table>

        <!-- 查看更多 -->
        <div v-if="!showAll && filteredCounters.length > pageSize" class="load-more">
          <el-link type="primary" @click="showAll = true">
            查看更多 (共 {{ filteredCounters.length }} 条)
          </el-link>
        </div>
      </div>

      <el-empty v-else-if="!loading" description="暂无性能计数器数据" />

      <!-- 原始输出（可折叠） -->
      <el-collapse v-if="rawOutput" class="raw-output-collapse">
        <el-collapse-item title="原始输出" name="raw">
          <div class="output-content">
            <pre>{{ rawOutput }}</pre>
          </div>
        </el-collapse-item>
      </el-collapse>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection, Refresh, Search } from '@element-plus/icons-vue'
import { getPerfCounter } from '@/api/arthas'

const props = defineProps<{
  clusterId: number | null
  namespace: string
  pod: string
  container: string
  processId: string
  attached: boolean
}>()

interface PerfCounter {
  name: string
  value: string
  unit: string
}

const loading = ref(false)
const rawOutput = ref('')
const perfCounters = ref<PerfCounter[]>([])
const searchText = ref('')
const showAll = ref(false)
const pageSize = 50

// 过滤后的计数器
const filteredCounters = computed(() => {
  if (!searchText.value) {
    return perfCounters.value
  }
  const keyword = searchText.value.toLowerCase()
  return perfCounters.value.filter(c =>
    c.name.toLowerCase().includes(keyword) ||
    c.value.toLowerCase().includes(keyword)
  )
})

// 显示的计数器（分页）
const displayCounters = computed(() => {
  if (showAll.value) {
    return filteredCounters.value
  }
  return filteredCounters.value.slice(0, pageSize)
})

// 移除 ANSI 转义码
const stripAnsi = (str: string): string => {
  // 匹配 ANSI 转义序列: \x1b[...m 或 \033[...m 或 [数字;数字m 格式
  return str
    .replace(/\x1b\[[0-9;]*m/g, '')
    .replace(/\033\[[0-9;]*m/g, '')
    .replace(/\[\d+;\d+m/g, '')
    .replace(/\[\d+m/g, '')
    .replace(/\[0m/g, '')
    .replace(/\[m/g, '')
}

// 检查是否是有效的计数器名称
const isValidCounterName = (name: string): boolean => {
  // 有效的计数器名称应该：
  // 1. 以字母开头
  // 2. 包含 . 分隔的包名格式，如 java.xxx 或 sun.xxx
  // 3. 不包含 ASCII art 字符
  if (!name || name.length < 3) return false

  // 排除 ASCII art 和无效字符
  if (/^[,\-`'|\\\/\s.]+$/.test(name)) return false
  if (/[`'\\|]/.test(name)) return false
  if (name.startsWith(',') || name.startsWith('-') || name.startsWith('.')) return false

  // 有效的计数器名通常是 java.xxx 或 sun.xxx 格式
  // 或者是简单的字母数字下划线组合
  return /^[a-zA-Z][a-zA-Z0-9._]+$/.test(name)
}

// 解析性能计数器输出
const parsePerfCounterOutput = (output: string): PerfCounter[] => {
  const counters: PerfCounter[] = []
  // 先移除 ANSI 转义码
  const cleanOutput = stripAnsi(output)
  const lines = cleanOutput.split('\n')

  // 需要跳过的行（Arthas 启动信息、logo等）
  const skipPatterns = [
    /^\[INFO\]/,
    /^\[arthas@/,
    /^-----/,
    /^=====/,
    /^NAME\s/i,
    /Affect/,
    /wiki/i,
    /tutorials/i,
    /^version\s/i,
    /^main_class\s/i,
    /^pid\s/i,
    /^start_time\s/i,
    /^current_time\s/i,
    /please check arthas/i,
    /ARTHAS/,
    /aliyun\.com/,
    /Process ends/i,
    /^[,\-`'|\\\/\s.]+$/,  // 纯 ASCII art 符号行
  ]

  for (const line of lines) {
    const trimmedLine = line.trim()

    // 跳过空行
    if (!trimmedLine) continue

    // 检查是否匹配跳过模式
    let shouldSkip = false
    for (const pattern of skipPatterns) {
      if (pattern.test(trimmedLine)) {
        shouldSkip = true
        break
      }
    }
    if (shouldSkip) continue

    // 解析计数器行
    // perfcounter 输出格式: name    value
    const parts = trimmedLine.split(/\s{2,}/)

    if (parts.length >= 2) {
      const name = parts[0].trim()
      const value = parts[1].trim()
      const unit = parts[2]?.trim() || ''

      // 验证是否是有效的计数器名称
      if (isValidCounterName(name) && value) {
        counters.push({ name, value, unit })
      }
    }
  }

  return counters
}

// 判断是否是数字
const isNumber = (value: string): boolean => {
  return /^-?\d+(\.\d+)?$/.test(value)
}

// 格式化值
const formatValue = (value: string): string => {
  if (isNumber(value)) {
    const num = parseFloat(value)
    if (num >= 1000000000) {
      return (num / 1000000000).toFixed(2) + 'G'
    } else if (num >= 1000000) {
      return (num / 1000000).toFixed(2) + 'M'
    } else if (num >= 1000) {
      return (num / 1000).toFixed(2) + 'K'
    }
  }
  return value
}

const loadPerfCounters = async () => {
  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    return
  }

  loading.value = true
  showAll.value = false

  try {
    const res = await getPerfCounter({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId
    })

    const output = typeof res === 'string' ? res : (res?.data || '')
    rawOutput.value = output
    perfCounters.value = parsePerfCounterOutput(output)

    if (perfCounters.value.length === 0 && output) {
    }
  } catch (error: any) {
    ElMessage.error('获取性能计数器失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

watch(() => props.attached, (newVal) => {
  if (newVal) {
    loadPerfCounters()
  } else {
    rawOutput.value = ''
    perfCounters.value = []
  }
})

// 搜索时重置分页
watch(searchText, () => {
  showAll.value = false
})
</script>

<style scoped>
.perf-counter-panel {
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

.toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.counter-count {
  font-size: 13px;
  color: #909399;
  margin-left: auto;
}

/* 表格 */
.table-section {
  background: #fff;
  border-radius: 0;
  border: 1px solid #ebeef5;
  overflow: hidden;
}

.counter-name {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  color: #303133;
}

.counter-value {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  color: #606266;
}

.counter-value.is-number {
  color: #409eff;
  font-weight: 600;
}

.text-muted {
  color: #c0c4cc;
}

/* 查看更多 */
.load-more {
  text-align: center;
  padding: 16px;
  border-top: 1px solid #ebeef5;
}

/* 原始输出 */
.raw-output-collapse {
  margin-top: 8px;
}

.output-content {
  padding: 12px;
  background: #1e1e1e;
  max-height: 300px;
  overflow: auto;
  border-radius: 0;
}

.output-content pre {
  margin: 0;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  line-height: 1.4;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
