<template>
  <div class="probe-config">
    <div class="config-grid">
      <!-- 探针类型 -->
      <div class="config-item">
        <label class="config-label">检测类型</label>
        <el-select v-model="localProbe.type" placeholder="选择检测类型" @change="handleTypeChange">
          <el-option label="HTTP GET" value="httpGet" />
          <el-option label="TCP Socket" value="tcpSocket" />
          <el-option label="Exec 执行命令" value="exec" />
        </el-select>
      </div>

      <!-- HTTP GET 配置 -->
      <template v-if="localProbe.type === 'httpGet'">
        <div class="config-item">
          <label class="config-label">路径</label>
          <el-input v-model="localProbe.path" placeholder="例如: /health" />
        </div>
        <div class="config-item">
          <label class="config-label">端口</label>
          <el-input-number v-model="localProbe.port" :min="1" :max="65535" controls-position="right" />
        </div>
        <div class="config-item">
          <label class="config-label">协议</label>
          <el-select v-model="localProbe.scheme">
            <el-option label="HTTP" value="HTTP" />
            <el-option label="HTTPS" value="HTTPS" />
          </el-select>
        </div>
      </template>

      <!-- TCP Socket 配置 -->
      <template v-else-if="localProbe.type === 'tcpSocket'">
        <div class="config-item">
          <label class="config-label">端口</label>
          <el-input-number v-model="localProbe.port" :min="1" :max="65535" controls-position="right" />
        </div>
      </template>

      <!-- Exec 配置 -->
      <template v-else-if="localProbe.type === 'exec'">
        <div class="config-item full-width">
          <label class="config-label">执行命令</label>
          <el-input
            v-model="commandStr"
            type="textarea"
            :rows="3"
            placeholder="例如: cat /tmp/healthy"
            @input="handleCommandChange"
          />
          <div class="config-tip">多个命令用空格分隔，例如: cat /tmp/healthy</div>
        </div>
      </template>
    </div>

    <!-- 高级设置 -->
    <div class="advanced-settings">
      <div class="advanced-title" @click="toggleAdvanced">
        <span>高级设置</span>
        <el-icon class="arrow-icon" :class="{ expanded: advancedExpanded }">
          <ArrowDown />
        </el-icon>
      </div>
      <div v-show="advancedExpanded" class="advanced-grid">
        <div class="config-item">
          <label class="config-label">初始延迟(秒)</label>
          <el-input-number v-model="localProbe.initialDelaySeconds" :min="0" :max="300" controls-position="right" />
          <div class="config-tip">容器启动后等待多久开始检测</div>
        </div>
        <div class="config-item">
          <label class="config-label">超时时间(秒)</label>
          <el-input-number v-model="localProbe.timeoutSeconds" :min="1" :max="60" controls-position="right" />
          <div class="config-tip">检测超时时间</div>
        </div>
        <div class="config-item">
          <label class="config-label">检测周期(秒)</label>
          <el-input-number v-model="localProbe.periodSeconds" :min="1" :max="300" controls-position="right" />
          <div class="config-tip">每隔多少秒进行一次检测</div>
        </div>
        <div class="config-item">
          <label class="config-label">成功阈值</label>
          <el-input-number v-model="localProbe.successThreshold" :min="1" :max="10" controls-position="right" />
          <div class="config-tip">连续成功多少次才算成功</div>
        </div>
        <div class="config-item">
          <label class="config-label">失败阈值</label>
          <el-input-number v-model="localProbe.failureThreshold" :min="1" :max="30" controls-position="right" />
          <div class="config-tip">连续失败多少次才算失败</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, computed } from 'vue'
import { ArrowDown } from '@element-plus/icons-vue'

interface Probe {
  enabled: boolean
  type: string
  path?: string
  port?: number
  scheme?: string
  command?: string[]
  initialDelaySeconds?: number
  timeoutSeconds?: number
  periodSeconds?: number
  successThreshold?: number
  failureThreshold?: number
}

const props = defineProps<{
  modelValue: Probe
}>()

const emit = defineEmits<{
  update: [value: Probe]
}>()

const localProbe = reactive<Probe>({ ...props.modelValue })
const advancedExpanded = ref(false)

// 命令字符串（用于显示）
const commandStr = computed({
  get: () => {
    if (!localProbe.command || localProbe.command.length === 0) {
      return ''
    }
    return localProbe.command.join(' ')
  },
  set: (value: string) => {
    if (!value || value.trim() === '') {
      localProbe.command = []
    } else {
      localProbe.command = value.trim().split(/\s+/)
    }
  }
})

const handleCommandChange = () => {
  emitUpdate()
}

const handleTypeChange = () => {
  // 切换类型时重置相关字段
  if (localProbe.type === 'httpGet') {
    localProbe.path = localProbe.path || '/'
    localProbe.port = localProbe.port || 80
    localProbe.scheme = localProbe.scheme || 'HTTP'
  } else if (localProbe.type === 'tcpSocket') {
    localProbe.port = localProbe.port || 80
  } else if (localProbe.type === 'exec') {
    localProbe.command = localProbe.command || []
  }
  emitUpdate()
}

const emitUpdate = () => {
  emit('update', { ...localProbe })
}

const toggleAdvanced = () => {
  advancedExpanded.value = !advancedExpanded.value
}

// 监听 props 变化
watch(() => props.modelValue, (newVal) => {
  Object.assign(localProbe, newVal)
}, { deep: true })
</script>

<style scoped>
.probe-config {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
}

.config-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.config-item.full-width {
  grid-column: 1 / -1;
}

.config-label {
  font-size: 13px;
  font-weight: 500;
  color: #606266;
}

.config-tip {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
}

.advanced-settings {
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 0;
  padding: 16px;
}

.advanced-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 16px;
  padding: 8px;
  border-radius: 0;
  transition: background-color 0.3s;
}

.advanced-title:hover {
  background-color: #f5f7fa;
}

.arrow-icon {
  transition: transform 0.3s;
  cursor: pointer;
}

.arrow-icon.expanded {
  transform: rotate(180deg);
}

.advanced-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
}

.el-input-number,
.el-select,
.el-input {
  width: 100%;
}
</style>
