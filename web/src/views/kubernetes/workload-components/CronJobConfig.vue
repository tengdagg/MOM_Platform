<template>
  <div class="info-panel cronjob-panel">
    <div class="panel-header">
      <span class="panel-icon">📅</span>
      <span class="panel-title">CronJob 调度配置</span>
    </div>
    <div class="panel-content">
      <div class="form-row">
        <label>调度规则</label>
        <el-input v-model="formData.schedule" size="small" placeholder="例如: */5 * * * * (每5分钟执行一次)" />
        <div class="form-tip">Cron 表达式格式: 分 时 日 月 周</div>
      </div>

      <div class="form-row">
        <label>并发策略</label>
        <el-select v-model="formData.concurrencyPolicy" size="small" style="width: 100%">
          <el-option label="Allow (允许并发运行)" value="Allow" />
          <el-option label="Forbid (禁止并发运行)" value="Forbid" />
          <el-option label="Replace (替换旧任务)" value="Replace" />
        </el-select>
        <div class="form-tip">
          Allow: 允许同时运行多个任务 | Forbid: 跳过新任务如果上次任务还在运行 | Replace: 替换正在运行的任务
        </div>
      </div>

      <div class="form-row">
        <label>时区</label>
        <el-input v-model="formData.timeZone" size="small" placeholder="例如: Asia/Shanghai" />
        <div class="form-tip">留空使用集群默认时区</div>
      </div>

      <div class="form-section">
        <div class="form-section-header">
          <label>历史记录限制</label>
        </div>
        <div class="form-row">
          <label>成功任务保留数</label>
          <el-input-number v-model="formData.successfulJobsHistoryLimit" :min="0" :max="100" size="small" style="width: 100%" />
        </div>
        <div class="form-row">
          <label>失败任务保留数</label>
          <el-input-number v-model="formData.failedJobsHistoryLimit" :min="0" :max="100" size="small" style="width: 100%" />
        </div>
        <div class="form-tip">控制保留多少个已完成和失败的 Job 记录</div>
      </div>

      <div class="form-section">
        <div class="form-section-header">
          <label>任务截止时间</label>
        </div>
        <div class="form-row">
          <label>启动截止时间 (秒)</label>
          <el-input-number v-model="formData.startingDeadlineSeconds" :min="0" size="small" style="width: 100%" />
          <div class="form-tip">如果任务错过调度时间超过此秒数，将不再执行。设置为0表示不限制</div>
        </div>
      </div>

      <div class="form-section">
        <div class="form-section-header">
          <label>暂停调度</label>
        </div>
        <div class="form-row">
          <el-switch v-model="formData.suspend" active-text="暂停" inactive-text="启用" />
          <div class="form-tip">暂停后不会创建新的 Job，但正在运行的 Job 不会受影响</div>
        </div>
      </div>

      <div class="form-section">
        <div class="form-section-header">
          <label>常用调度示例</label>
        </div>
        <div class="schedule-examples">
          <div class="example-item" @click="applySchedule('*/5 * * * *')">
            <span class="example-label">每 5 分钟</span>
            <span class="example-value">*/5 * * * *</span>
          </div>
          <div class="example-item" @click="applySchedule('0 * * * *')">
            <span class="example-label">每小时</span>
            <span class="example-value">0 * * * *</span>
          </div>
          <div class="example-item" @click="applySchedule('0 0 * * *')">
            <span class="example-label">每天零点</span>
            <span class="example-value">0 0 * * *</span>
          </div>
          <div class="example-item" @click="applySchedule('0 2 * * *')">
            <span class="example-label">每天凌晨 2 点</span>
            <span class="example-value">0 2 * * *</span>
          </div>
          <div class="example-item" @click="applySchedule('0 0 * * 0')">
            <span class="example-label">每周日零点</span>
            <span class="example-value">0 0 * * 0</span>
          </div>
          <div class="example-item" @click="applySchedule('0 0 1 * *')">
            <span class="example-label">每月 1 号零点</span>
            <span class="example-value">0 0 1 * *</span>
          </div>
          <div class="example-item" @click="applySchedule('0 0 * * 1-5')">
            <span class="example-label">工作日零点</span>
            <span class="example-value">0 0 * * 1-5</span>
          </div>
          <div class="example-item" @click="applySchedule('*/30 9-17 * * 1-5')">
            <span class="example-label">工作日上班时间每30分钟</span>
            <span class="example-value">*/30 9-17 * * 1-5</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface CronJobConfig {
  schedule: string
  concurrencyPolicy: string
  timeZone: string
  successfulJobsHistoryLimit: number
  failedJobsHistoryLimit: number
  startingDeadlineSeconds: number | null
  suspend: boolean
}

const props = defineProps<{
  formData: CronJobConfig
}>()

const emit = defineEmits<{
  'update:formData': [value: CronJobConfig]
}>()

const applySchedule = (schedule: string) => {
  emit('update:formData', {
    ...props.formData,
    schedule
  })
}
</script>

<style scoped>
.info-panel {
  background: #ffffff;
  border-radius: 0;
  border: 1px solid #e8e8e8;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  overflow: hidden;
}

.cronjob-panel {
  border-right: 1px solid #f0f0f0;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 20px;
  border-bottom: 2px solid #d4af37;
  background: linear-gradient(135deg, #fafafa 0%, #ffffff 100%);
  position: sticky;
  top: 0;
  z-index: 10;
}

.panel-icon {
  font-size: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: #d4af37;
  border-radius: 0;
  color: #ffffff;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  flex: 1;
  letter-spacing: 0.3px;
}

.panel-content {
  padding: 20px;
  background: #ffffff;
}

.form-row {
  margin-bottom: 20px;
}

.form-row label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: #333;
  margin-bottom: 8px;
  letter-spacing: 0.3px;
}

.form-row .el-input :deep(.el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.form-row .el-input :deep(.el-input__wrapper:hover) {
  border-color: #ffffff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 3px rgba(212, 175, 55, 0.1);
}

.form-row .el-input :deep(.el-input__wrapper.is-focus) {
  border-color: #ffffff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 4px rgba(212, 175, 55, 0.15);
}

.form-row .el-input-number {
  width: 100%;
}

.form-row .el-input-number :deep(.el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 0;
}

.form-row .el-select {
  width: 100%;
}

.form-tip {
  font-size: 12px;
  color: #999;
  margin-top: 6px;
  line-height: 1.5;
}

.form-section {
  margin-bottom: 24px;
  padding: 16px;
  background: linear-gradient(135deg, #fafafa 0%, #f5f5f5 100%);
  border-radius: 0;
  border: 1px solid #e8e8e8;
}

.form-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.form-section-header label {
  font-size: 14px;
  font-weight: 600;
  color: #333;
  letter-spacing: 0.3px;
}

.form-section .form-row {
  margin-bottom: 12px;
}

.form-section .form-row:last-child {
  margin-bottom: 0;
}

.schedule-examples {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
}

.example-item {
  display: flex;
  flex-direction: column;
  padding: 12px;
  background: #ffffff;
  border: 1px solid #e8e8e8;
  border-radius: 0;
  cursor: pointer;
  transition: all 0.3s ease;
}

.example-item:hover {
  border-color: #ffffff;
  background: #fffef5;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.2);
  transform: translateY(-2px);
}

.example-label {
  font-size: 12px;
  color: #666;
  margin-bottom: 4px;
}

.example-value {
  font-size: 13px;
  font-weight: 600;
  color: #ffffff;
  font-family: 'Courier New', monospace;
}

.form-row .el-switch {
  margin-top: 8px;
}
</style>
