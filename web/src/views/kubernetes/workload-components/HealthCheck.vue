<template>
  <div class="health-check-config">
    <div class="check-section">
      <!-- 存活探针 -->
      <div class="probe-item">
        <div class="probe-header">
          <div class="probe-title">
            <el-icon><Monitor /></el-icon>
            <span>存活探针 (Liveness Probe)</span>
          </div>
          <el-switch v-model="livenessProbe.enabled" @change="handleLivenessChange" />
        </div>
        <div v-if="livenessProbe.enabled" class="probe-config">
          <ProbeConfig v-model="livenessProbe" @update="updateLiveness" />
        </div>
      </div>

      <!-- 就绪探针 -->
      <div class="probe-item">
        <div class="probe-header">
          <div class="probe-title">
            <el-icon><CircleCheck /></el-icon>
            <span>就绪探针 (Readiness Probe)</span>
          </div>
          <el-switch v-model="readinessProbe.enabled" @change="handleReadinessChange" />
        </div>
        <div v-if="readinessProbe.enabled" class="probe-config">
          <ProbeConfig v-model="readinessProbe" @update="updateReadiness" />
        </div>
      </div>

      <!-- 启动探针 -->
      <div class="probe-item">
        <div class="probe-header">
          <div class="probe-title">
            <el-icon><Odometer /></el-icon>
            <span>启动探针 (Startup Probe)</span>
          </div>
          <el-switch v-model="startupProbe.enabled" @change="handleStartupChange" />
        </div>
        <div v-if="startupProbe.enabled" class="probe-config">
          <ProbeConfig v-model="startupProbe" @update="updateStartup" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { Monitor, CircleCheck, Odometer } from '@element-plus/icons-vue'
import ProbeConfig from './ProbeConfig.vue'

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
  livenessProbe?: Probe
  readinessProbe?: Probe
  startupProbe?: Probe
}>()

const emit = defineEmits<{
  updateLiveness: [probe: Probe]
  updateReadiness: [probe: Probe]
  updateStartup: [probe: Probe]
}>()

// 初始化探针数据
const livenessProbe = reactive<Probe>(props.livenessProbe || {
  enabled: false,
  type: 'httpGet',
  path: '/',
  port: 80,
  scheme: 'HTTP',
  initialDelaySeconds: 0,
  timeoutSeconds: 3,
  periodSeconds: 10,
  successThreshold: 1,
  failureThreshold: 3
})

const readinessProbe = reactive<Probe>(props.readinessProbe || {
  enabled: false,
  type: 'httpGet',
  path: '/',
  port: 80,
  scheme: 'HTTP',
  initialDelaySeconds: 0,
  timeoutSeconds: 3,
  periodSeconds: 10,
  successThreshold: 1,
  failureThreshold: 3
})

const startupProbe = reactive<Probe>(props.startupProbe || {
  enabled: false,
  type: 'httpGet',
  path: '/',
  port: 80,
  scheme: 'HTTP',
  initialDelaySeconds: 0,
  timeoutSeconds: 3,
  periodSeconds: 10,
  successThreshold: 1,
  failureThreshold: 3
})

const updateLiveness = () => {
  emit('updateLiveness', { ...livenessProbe })
}

const updateReadiness = () => {
  emit('updateReadiness', { ...readinessProbe })
}

const updateStartup = () => {
  emit('updateStartup', { ...startupProbe })
}

const handleLivenessChange = (enabled: boolean) => {
  if (!enabled) {
    emit('updateLiveness', { ...livenessProbe, enabled: false })
  } else {
    emit('updateLiveness', { ...livenessProbe })
  }
}

const handleReadinessChange = (enabled: boolean) => {
  if (!enabled) {
    emit('updateReadiness', { ...readinessProbe, enabled: false })
  } else {
    emit('updateReadiness', { ...readinessProbe })
  }
}

const handleStartupChange = (enabled: boolean) => {
  if (!enabled) {
    emit('updateStartup', { ...startupProbe, enabled: false })
  } else {
    emit('updateStartup', { ...startupProbe })
  }
}
</script>

<style scoped>
.health-check-config {
  padding: 0;
}

.check-section {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.probe-item {
  background: #f8f9fa;
  border: 1px solid #e4e7ed;
  border-radius: 0;
  padding: 20px;
  transition: all 0.3s;
}

.probe-item:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.1);
}

.probe-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e4e7ed;
}

.probe-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 500;
  color: #303133;
}

.probe-title .el-icon {
  font-size: 18px;
  color: #409eff;
}

.probe-config {
  margin-top: 16px;
}
</style>
