<template>
  <div class="spec-content-wrapper">
    <div class="info-panel">
      <div class="panel-header">
        <span class="panel-icon">⚙️</span>
        <span class="panel-title">其他配置</span>
      </div>
      <div class="panel-content">
      <div class="network-config-form">
        <div class="config-form-section">
          <div class="form-section-title">
            <span class="section-icon">⏱️</span>
            <span class="section-text">优雅终止</span>
          </div>
          <div class="form-item-row">
            <label class="form-label">优雅终止期限(秒)</label>
            <el-input-number v-model="formData.terminationGracePeriodSeconds" :min="0" :max="3600" size="small" style="width: 200px;" />
            <span class="form-hint">Pod 删除时等待优雅终止的时间（秒），默认 30 秒</span>
          </div>
          <div class="form-item-row" v-if="formData.type !== 'Job' && formData.type !== 'CronJob'">
            <label class="form-label">活动期限(秒)</label>
            <el-input-number v-model="formData.activeDeadlineSeconds" :min="0" :max="86400" size="small" style="width: 200px;" controls-position="right" />
            <span class="form-hint">可选，Pod 可运行的最长时间（秒），超时将被终止</span>
          </div>
        </div>

        <div class="config-form-section">
          <div class="form-section-title">
            <span class="section-icon">🔐</span>
            <span class="section-text">服务账户</span>
          </div>
          <div class="form-item-row">
            <label class="form-label">服务账户名</label>
            <el-input v-model="formData.serviceAccountName" placeholder="默认: default" style="width: 300px;" />
            <span class="form-hint">指定运行 Pod 的服务账户</span>
          </div>
          <div class="form-item-row">
            <label class="form-label">自动挂载令牌</label>
            <el-switch
              v-model="formData.automountServiceAccountToken"
              active-text="开启"
              inactive-text="关闭"
            />
            <span class="form-hint">是否自动挂载服务账户令牌到 Pod</span>
          </div>
        </div>

        <div class="config-form-section">
          <div class="form-section-title">
            <span class="section-icon">⭐</span>
            <span class="section-text">优先级</span>
          </div>
          <div class="form-item-row">
            <label class="form-label">优先级类名称</label>
            <el-input v-model="formData.priorityClassName" placeholder="可选，如: high-priority" style="width: 300px;" />
            <span class="form-hint">指定 Pod 的优先级类，影响调度优先级</span>
          </div>
        </div>

        <div class="config-form-section">
          <div class="form-section-title">
            <span class="section-icon">🔄</span>
            <span class="section-text">重启策略</span>
          </div>
          <div class="form-item-row">
            <label class="form-label">重启策略</label>
            <el-select v-model="formData.restartPolicy" placeholder="选择重启策略" style="width: 200px;" :disabled="formData.type === 'Deployment'">
              <el-option label="Always" value="Always" />
              <el-option label="OnFailure" value="OnFailure" />
              <el-option label="Never" value="Never" />
            </el-select>
            <span class="form-hint" v-if="formData.type === 'Deployment'">Deployment 固定为 Always</span>
            <span class="form-hint" v-else>容器退出时的重启策略</span>
          </div>
        </div>
      </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface FormData {
  terminationGracePeriodSeconds: string
  activeDeadlineSeconds: string
  priorityClassName: string
  serviceAccountName: string
  automountServiceAccountToken: boolean
  restartPolicy: string
  type: string
}

const props = defineProps<{
  formData: FormData
}>()
</script>

<style scoped>
.spec-content-wrapper {
  padding: 0;
  background: transparent;
}

.info-panel {
  background: #fff;
  border-radius: 0;
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  padding: 12px 20px;
  background: #0f69a6;
  border-bottom: 1px solid #0f69a6;
}

.panel-icon {
  font-size: 18px;
  margin-right: 8px;
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  color: #ffffff;
  flex: 1;
}

.panel-content {
  padding: 16px;
  background: #ffffff;
}

.spec-content-header {
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 2px solid #f0f0f0;
}

.spec-content-header h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.spec-content-header p {
  margin: 0;
  font-size: 13px;
  color: #909399;
}

.spec-content {
  background: #fff;
}

.network-config-form {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.config-form-section {
  padding: 20px;
  background: #f8f9fa;
  border-radius: 0;
  border: 1px solid #e4e7ed;
}

.form-section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.section-icon {
  font-size: 18px;
}

.section-text {
  font-size: 15px;
  font-weight: 600;
}

.form-item-row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}

.form-item-row:last-child {
  border-bottom: none;
}

.form-label {
  width: 120px;
  font-size: 14px;
  font-weight: 500;
  color: #606266;
  flex-shrink: 0;
}

.form-hint {
  font-size: 12px;
  color: #909399;
  margin-left: 8px;
}
</style>
