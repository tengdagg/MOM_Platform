<template>
  <div class="jvm-info-panel">
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
        <el-button type="primary" size="small" @click="loadJvmInfo" :loading="loading">
          <el-icon><Refresh /></el-icon> 刷新
        </el-button>
      </div>

      <!-- JVM 信息卡片 -->
      <div class="info-grid" v-if="Object.keys(jvmInfo).length > 0">
        <!-- RUNTIME -->
        <div class="info-card" v-if="jvmInfo.RUNTIME">
          <div class="card-header">
            <el-icon><Timer /></el-icon>
            <span>RUNTIME</span>
          </div>
          <div class="card-body">
            <div class="info-row" v-for="(value, key) in jvmInfo.RUNTIME" :key="key">
              <span class="info-label">{{ key }}</span>
              <span class="info-value" :title="String(value)">{{ formatValue(value) }}</span>
            </div>
          </div>
        </div>

        <!-- CLASS-LOADING -->
        <div class="info-card" v-if="jvmInfo['CLASS-LOADING']">
          <div class="card-header">
            <el-icon><Files /></el-icon>
            <span>CLASS-LOADING</span>
          </div>
          <div class="card-body">
            <div class="info-row" v-for="(value, key) in jvmInfo['CLASS-LOADING']" :key="key">
              <span class="info-label">{{ key }}</span>
              <span class="info-value">{{ value }}</span>
            </div>
          </div>
        </div>

        <!-- COMPILATION -->
        <div class="info-card" v-if="jvmInfo.COMPILATION">
          <div class="card-header">
            <el-icon><Cpu /></el-icon>
            <span>COMPILATION</span>
          </div>
          <div class="card-body">
            <div class="info-row" v-for="(value, key) in jvmInfo.COMPILATION" :key="key">
              <span class="info-label">{{ key }}</span>
              <span class="info-value">{{ value }}</span>
            </div>
          </div>
        </div>

        <!-- THREAD -->
        <div class="info-card" v-if="jvmInfo.THREAD">
          <div class="card-header">
            <el-icon><Operation /></el-icon>
            <span>THREAD</span>
          </div>
          <div class="card-body">
            <div class="info-row" v-for="(value, key) in jvmInfo.THREAD" :key="key">
              <span class="info-label">{{ key }}</span>
              <span class="info-value">{{ value }}</span>
            </div>
          </div>
        </div>

        <!-- GARBAGE-COLLECTORS -->
        <div class="info-card wide" v-if="jvmInfo['GARBAGE-COLLECTORS']">
          <div class="card-header">
            <el-icon><Delete /></el-icon>
            <span>GARBAGE-COLLECTORS</span>
          </div>
          <div class="card-body">
            <div class="info-row" v-for="(value, key) in jvmInfo['GARBAGE-COLLECTORS']" :key="key">
              <span class="info-label">{{ key }}</span>
              <span class="info-value">{{ formatArrayValue(value) }}</span>
            </div>
          </div>
        </div>

        <!-- MEMORY-MANAGERS -->
        <div class="info-card wide" v-if="jvmInfo['MEMORY-MANAGERS']">
          <div class="card-header">
            <el-icon><Coin /></el-icon>
            <span>MEMORY-MANAGERS</span>
          </div>
          <div class="card-body">
            <div class="info-row" v-for="(value, key) in jvmInfo['MEMORY-MANAGERS']" :key="key">
              <span class="info-label">{{ key }}</span>
              <span class="info-value">{{ formatArrayValue(value) }}</span>
            </div>
          </div>
        </div>

        <!-- MEMORY -->
        <div class="info-card wide" v-if="jvmInfo.MEMORY">
          <div class="card-header">
            <el-icon><Histogram /></el-icon>
            <span>MEMORY</span>
          </div>
          <div class="card-body">
            <div class="info-row" v-for="(value, key) in jvmInfo.MEMORY" :key="key">
              <span class="info-label">{{ key }}</span>
              <span class="info-value">{{ value }}</span>
            </div>
          </div>
        </div>

        <!-- OPERATING-SYSTEM -->
        <div class="info-card wide" v-if="jvmInfo['OPERATING-SYSTEM']">
          <div class="card-header">
            <el-icon><Monitor /></el-icon>
            <span>OPERATING-SYSTEM</span>
          </div>
          <div class="card-body">
            <div class="info-row" v-for="(value, key) in jvmInfo['OPERATING-SYSTEM']" :key="key">
              <span class="info-label">{{ key }}</span>
              <span class="info-value">{{ value }}</span>
            </div>
          </div>
        </div>

        <!-- FILE-DESCRIPTOR -->
        <div class="info-card" v-if="jvmInfo['FILE-DESCRIPTOR']">
          <div class="card-header">
            <el-icon><Document /></el-icon>
            <span>FILE-DESCRIPTOR</span>
          </div>
          <div class="card-body">
            <div class="info-row" v-for="(value, key) in jvmInfo['FILE-DESCRIPTOR']" :key="key">
              <span class="info-label">{{ key }}</span>
              <span class="info-value">{{ value }}</span>
            </div>
          </div>
        </div>
      </div>

      <el-empty v-else-if="!loading" description="暂无JVM信息" />

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
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection, Refresh, Timer, Files, Cpu, Operation, Delete, Coin, Histogram, Monitor, Document } from '@element-plus/icons-vue'
import { getJvmInfo } from '@/api/arthas'

const props = defineProps<{
  clusterId: number | null
  namespace: string
  pod: string
  container: string
  processId: string
  attached: boolean
}>()

const loading = ref(false)
const rawOutput = ref('')
const jvmInfo = ref<Record<string, Record<string, any>>>({})

// 解析 JVM 输出
const parseJvmOutput = (output: string): Record<string, Record<string, any>> => {
  const result: Record<string, Record<string, any>> = {}
  const lines = output.split('\n')

  let currentSection = ''

  for (const line of lines) {
    const trimmedLine = line.trim()

    // 跳过空行和信息行
    if (!trimmedLine || trimmedLine.startsWith('[INFO]') || trimmedLine.startsWith('[arthas@')) {
      continue
    }

    // 检查是否是分隔线
    if (trimmedLine.startsWith('-----') || trimmedLine.startsWith('=====')) {
      continue
    }

    // 检查是否是新的 section
    if (/^[A-Z][A-Z\-]+$/.test(trimmedLine)) {
      currentSection = trimmedLine
      result[currentSection] = {}
      continue
    }

    // 解析键值对
    if (currentSection && trimmedLine.includes(':')) {
      // 处理带有多个冒号的情况（如时间戳）
      const colonIndex = trimmedLine.indexOf(':')
      // 检查是否是类似 "key   value" 的格式（有多个空格分隔）
      const spaceMatch = trimmedLine.match(/^(\S+)\s{2,}(.+)$/)

      if (spaceMatch) {
        const key = spaceMatch[1].trim()
        const value = spaceMatch[2].trim()
        if (key && result[currentSection]) {
          result[currentSection][key] = value
        }
      } else {
        const key = trimmedLine.substring(0, colonIndex).trim()
        const value = trimmedLine.substring(colonIndex + 1).trim()
        if (key && result[currentSection]) {
          result[currentSection][key] = value
        }
      }
    } else if (currentSection && trimmedLine.includes('  ')) {
      // 处理 "key    value" 格式
      const parts = trimmedLine.split(/\s{2,}/)
      if (parts.length >= 2) {
        const key = parts[0].trim()
        const value = parts.slice(1).join(' ').trim()
        if (key && result[currentSection]) {
          result[currentSection][key] = value
        }
      }
    }
  }

  return result
}

// 格式化值
const formatValue = (value: any): string => {
  if (typeof value === 'string' && value.length > 80) {
    return value.substring(0, 80) + '...'
  }
  return String(value)
}

// 格式化数组值
const formatArrayValue = (value: any): string => {
  if (Array.isArray(value)) {
    return value.join(', ')
  }
  return String(value)
}

const loadJvmInfo = async () => {
  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    return
  }

  loading.value = true
  try {
    const res = await getJvmInfo({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId
    })

    const output = typeof res === 'string' ? res : (res?.data || '')
    rawOutput.value = output
    jvmInfo.value = parseJvmOutput(output)

    if (Object.keys(jvmInfo.value).length === 0 && output) {
    }
  } catch (error: any) {
    ElMessage.error('获取JVM信息失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

watch(() => props.attached, (newVal) => {
  if (newVal) {
    loadJvmInfo()
  } else {
    rawOutput.value = ''
    jvmInfo.value = {}
  }
})
</script>

<style scoped>
.jvm-info-panel {
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
}

/* 信息卡片网格 */
.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.info-card {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 0;
  overflow: hidden;
}

.info-card.wide {
  grid-column: span 2;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #f5f7fa;
  border-bottom: 1px solid #ebeef5;
  color: #303133;
  font-size: 14px;
  font-weight: 600;
}

.card-header :deep(.el-icon) {
  font-size: 18px;
  color: #409eff;
}

.card-body {
  padding: 12px 16px;
  max-height: 300px;
  overflow-y: auto;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
  gap: 16px;
}

.info-row:last-child {
  border-bottom: none;
}

.info-label {
  font-size: 13px;
  color: #606266;
  flex-shrink: 0;
  max-width: 40%;
}

.info-value {
  font-size: 13px;
  color: #303133;
  font-weight: 500;
  text-align: right;
  word-break: break-all;
  max-width: 60%;
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

/* 响应式 */
@media (max-width: 768px) {
  .info-grid {
    grid-template-columns: 1fr;
  }

  .info-card.wide {
    grid-column: span 1;
  }
}
</style>
