<template>
  <div class="info-panel job-panel">
    <div class="panel-header">
      <span class="panel-icon">📝</span>
      <span class="panel-title">Job 任务配置</span>
    </div>
    <div class="panel-content">
      <div class="form-row">
        <label>完成次数 (Completions)</label>
        <el-input-number v-model="formData.completions" :min="1" :max="100" size="small" style="width: 100%" />
        <div class="form-tip">需要成功完成的Pod数量。设置为1表示只需要一个Pod成功完成任务</div>
      </div>

      <div class="form-row">
        <label>并行度 (Parallelism)</label>
        <el-input-number v-model="formData.parallelism" :min="1" :max="50" size="small" style="width: 100%" />
        <div class="form-tip">同时运行的Pod最大数量。设置为1表示串行执行</div>
      </div>

      <div class="form-row">
        <label>失败重试次数 (Backoff Limit)</label>
        <el-input-number v-model="formData.backoffLimit" :min="0" :max="20" size="small" style="width: 100%" />
        <div class="form-tip">Pod失败后的重试次数。设置为0表示不重试，6表示最多重试6次</div>
      </div>

      <div class="form-row">
        <label>活跃 deadline 秒数 (Active Deadline Seconds)</label>
        <el-input-number v-model="formData.activeDeadlineSeconds" :min="0" :max="86400" size="small" style="width: 100%" />
        <div class="form-tip">Job的最长运行时间（秒）。超过此时间Job将被标记为失败并终止所有Pod。设置为0表示无限制</div>
      </div>

      <div class="form-section">
        <div class="form-section-header">
          <label>常见场景示例</label>
        </div>
        <div class="job-examples">
          <div class="example-item" @click="applyExample('one-time')">
            <span class="example-title">一次性任务</span>
            <span class="example-desc">执行一次即完成</span>
            <div class="example-config">completions: 1, parallelism: 1</div>
          </div>
          <div class="example-item" @click="applyExample('parallel')">
            <span class="example-title">并行处理任务</span>
            <span class="example-desc">多个Pod同时工作</span>
            <div class="example-config">completions: 10, parallelism: 5</div>
          </div>
          <div class="example-item" @click="applyExample('sequential')">
            <span class="example-title">串行队列任务</span>
            <span class="example-desc">按顺序逐个完成</span>
            <div class="example-config">completions: 5, parallelism: 1</div>
          </div>
          <div class="example-item" @click="applyExample('work-queue')">
            <span class="example-title">工作队列模式</span>
            <span class="example-desc">Pod自行协调任务</span>
            <div class="example-config">completions: 1, parallelism: 多个</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface JobConfig {
  completions: number
  parallelism: number
  backoffLimit: number
  activeDeadlineSeconds: number | null
}

const props = defineProps<{
  formData: JobConfig
}>()

const emit = defineEmits<{
  'update:formData': [value: JobConfig]
}>()

const applyExample = (type: string) => {
  let config: JobConfig

  switch (type) {
    case 'one-time':
      config = {
        completions: 1,
        parallelism: 1,
        backoffLimit: 6,
        activeDeadlineSeconds: null
      }
      break
    case 'parallel':
      config = {
        completions: 10,
        parallelism: 5,
        backoffLimit: 6,
        activeDeadlineSeconds: null
      }
      break
    case 'sequential':
      config = {
        completions: 5,
        parallelism: 1,
        backoffLimit: 6,
        activeDeadlineSeconds: null
      }
      break
    case 'work-queue':
      config = {
        completions: 1,
        parallelism: 3,
        backoffLimit: 6,
        activeDeadlineSeconds: null
      }
      break
    default:
      return
  }

  emit('update:formData', config)
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

.job-panel {
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

.form-row .el-input-number {
  width: 100%;
}

.form-row .el-input-number :deep(.el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.form-row .el-input-number :deep(.el-input__wrapper:hover) {
  border-color: #ffffff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 3px rgba(212, 175, 55, 0.1);
}

.form-row .el-input-number :deep(.el-input__wrapper.is-focus) {
  border-color: #ffffff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 4px rgba(212, 175, 55, 0.15);
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

.job-examples {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.example-item {
  padding: 14px;
  background: #ffffff;
  border: 1px solid #e8e8e8;
  border-radius: 0;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.example-item:hover {
  border-color: #ffffff;
  background: #fffef5;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.2);
  transform: translateY(-2px);
}

.example-title {
  font-size: 14px;
  font-weight: 600;
  color: #333;
}

.example-desc {
  font-size: 12px;
  color: #666;
}

.example-config {
  font-size: 11px;
  color: #ffffff;
  font-family: 'Courier New', monospace;
  margin-top: 4px;
}
</style>
