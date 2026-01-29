<template>
  <div class="info-panel scaling-panel">
    <div class="panel-header">
      <span class="panel-icon">📊</span>
      <span class="panel-title">{{ panelTitle }}</span>
    </div>
    <div class="panel-content">
      <!-- Deployment/StatefulSet 扩容配置 -->
      <template v-if="workloadType === 'Deployment' || workloadType === 'StatefulSet'">
        <div class="form-row">
          <label>当前副本数</label>
          <el-input-number :model-value="formData.replicas" @update:model-value="updateReplicas" :min="0" :max="100" size="small" style="width: 100%" />
          <div class="form-tip" v-if="workloadType === 'Deployment'">
            Deployment 会持续维护指定数量的 Pod 副本，确保应用的高可用性
          </div>
          <div class="form-tip" v-else-if="workloadType === 'StatefulSet'">
            StatefulSet 会维护指定数量的有序 Pod 副本，每个 Pod 都有唯一标识
          </div>
        </div>

        <div class="form-section">
          <div class="form-section-header">
            <label>更新策略</label>
          </div>
          <div class="form-row">
            <label>策略类型</label>
            <el-select :model-value="scalingStrategy.strategyType" @update:model-value="updateScalingStrategy('strategyType', $event)" size="small" style="width: 100%">
              <el-option label="RollingUpdate (滚动更新)" value="RollingUpdate" />
              <el-option label="OnDelete (删除时更新)" value="OnDelete" v-if="workloadType === 'StatefulSet'" />
              <el-option label="Recreate (重建)" value="Recreate" v-if="workloadType === 'Deployment'" />
            </el-select>
          </div>

          <template v-if="scalingStrategy.strategyType === 'RollingUpdate'">
            <div class="form-row">
              <label>最大激增 (Max Surge)</label>
              <el-input :model-value="scalingStrategy.maxSurge" @update:model-value="updateScalingStrategy('maxSurge', $event)" size="small" placeholder="例如: 25%" />
              <div class="form-tip">滚动更新期间最多可以超出期望副本数的数量，可以是数量或百分比</div>
            </div>

            <div class="form-row">
              <label>最大不可用 (Max Unavailable)</label>
              <el-input :model-value="scalingStrategy.maxUnavailable" @update:model-value="updateScalingStrategy('maxUnavailable', $event)" size="small" placeholder="例如: 25%" />
              <div class="form-tip">滚动更新期间最多可以不可用的 Pod 数量，可以是数量或百分比</div>
            </div>
          </template>

          <div class="form-row">
            <label>最小就绪时间 (秒)</label>
            <el-input-number :model-value="scalingStrategy.minReadySeconds" @update:model-value="updateScalingStrategy('minReadySeconds', $event)" :min="0" :max="3600" size="small" style="width: 100%" />
            <div class="form-tip">新 Pod 就绪后至少保持多久才认为可用，默认 0 秒</div>
          </div>

          <div class="form-row">
            <label>进度截止时间 (秒)</label>
            <el-input-number :model-value="scalingStrategy.progressDeadlineSeconds" @update:model-value="updateScalingStrategy('progressDeadlineSeconds', $event)" :min="0" :max="3600" size="small" style="width: 100%" />
            <div class="form-tip">滚动更新的超时时间，超时后会标记为失败，默认 600 秒</div>
          </div>

          <div class="form-row">
            <label>版本历史限制</label>
            <el-input-number :model-value="scalingStrategy.revisionHistoryLimit" @update:model-value="updateScalingStrategy('revisionHistoryLimit', $event)" :min="0" :max="100" size="small" style="width: 100%" />
            <div class="form-tip">保留的历史版本数量，用于回滚操作，默认 10 个</div>
          </div>
        </div>
      </template>

      <!-- DaemonSet 扩容配置 -->
      <template v-else-if="workloadType === 'DaemonSet'">
        <div class="form-section">
          <div class="form-section-header">
            <label>DaemonSet 说明</label>
          </div>
          <div class="daemonset-info">
            <div class="info-item">
              <span class="info-label">副本数模式</span>
              <span class="info-value">每个节点一个 Pod</span>
            </div>
            <div class="info-item">
              <span class="info-label">自动扩缩容</span>
              <span class="info-value">随节点数量自动调整</span>
            </div>
            <div class="info-item">
              <span class="info-label">典型用途</span>
              <span class="info-value">日志收集、监控代理、存储插件</span>
            </div>
            <div class="form-tip">
              DaemonSet 会在每个符合条件的节点上运行一个 Pod 副本。当节点添加到集群中时，Pod 会自动添加；
              当节点从集群移除时，Pod 也会自动回收。无需手动设置副本数。
            </div>
          </div>

          <div class="form-section">
            <div class="form-section-header">
              <label>更新策略</label>
            </div>
            <div class="form-row">
              <label>策略类型</label>
              <el-select :model-value="scalingStrategy.strategyType" @update:model-value="updateScalingStrategy('strategyType', $event)" size="small" style="width: 100%">
                <el-option label="RollingUpdate (滚动更新)" value="RollingUpdate" />
                <el-option label="OnDelete (删除时更新)" value="OnDelete" />
              </el-select>
            </div>

            <template v-if="scalingStrategy.strategyType === 'RollingUpdate'">
              <div class="form-row">
                <label>最大不可用 (Max Unavailable)</label>
                <el-input :model-value="scalingStrategy.maxUnavailable" @update:model-value="updateScalingStrategy('maxUnavailable', $event)" size="small" placeholder="例如: 1" />
                <div class="form-tip">滚动更新期间最多可以不可用的 Pod 数量</div>
              </div>
            </template>
          </div>
        </div>
      </template>

      <!-- Job 扩容配置 -->
      <template v-else-if="workloadType === 'Job'">
        <div class="form-section">
          <div class="form-section-header">
            <label>Job 任务配置</label>
          </div>
          <div class="form-row">
            <label>完成次数 (Completions)</label>
            <el-input-number :model-value="jobConfig.completions" @update:model-value="updateJobConfig('completions', $event)" :min="1" :max="1000" size="small" style="width: 100%" />
            <div class="form-tip">需要成功完成的 Pod 数量。设置为 1 表示只需要一个 Pod 成功完成任务</div>
          </div>

          <div class="form-row">
            <label>并行度 (Parallelism)</label>
            <el-input-number :model-value="jobConfig.parallelism" @update:model-value="updateJobConfig('parallelism', $event)" :min="1" :max="100" size="small" style="width: 100%" />
            <div class="form-tip">同时运行的 Pod 最大数量。设置为 1 表示串行执行</div>
          </div>

          <div class="form-row">
            <label>失败重试次数 (Backoff Limit)</label>
            <el-input-number :model-value="jobConfig.backoffLimit" @update:model-value="updateJobConfig('backoffLimit', $event)" :min="0" :max="20" size="small" style="width: 100%" />
            <div class="form-tip">Pod 失败后的重试次数。设置为 0 表示不重试，6 表示最多重试 6 次</div>
          </div>

          <div class="form-row">
            <label>活跃终止时间 (Active Deadline Seconds)</label>
            <el-input-number :model-value="jobConfig.activeDeadlineSeconds" @update:model-value="updateJobConfig('activeDeadlineSeconds', $event)" :min="0" :max="86400" size="small" style="width: 100%" />
            <div class="form-tip">Job 的最长运行时间（秒）。超过此时间 Job 将被标记为失败并终止所有 Pod。设置为 0 表示无限制</div>
          </div>

          <div class="job-examples">
            <div class="example-header">常见场景示例</div>
            <div class="example-item" @click="applyJobExample('one-time')">
              <span class="example-title">一次性任务</span>
              <span class="example-config">completions: 1, parallelism: 1</span>
            </div>
            <div class="example-item" @click="applyJobExample('parallel')">
              <span class="example-title">并行处理</span>
              <span class="example-config">completions: 10, parallelism: 5</span>
            </div>
            <div class="example-item" @click="applyJobExample('sequential')">
              <span class="example-title">串行队列</span>
              <span class="example-config">completions: 5, parallelism: 1</span>
            </div>
            <div class="example-item" @click="applyJobExample('work-queue')">
              <span class="example-title">工作队列</span>
              <span class="example-config">completions: 1, parallelism: 多个</span>
            </div>
          </div>
        </div>
      </template>

      <!-- CronJob 扩容配置 -->
      <template v-else-if="workloadType === 'CronJob'">
        <div class="form-section">
          <div class="form-section-header">
            <label>调度规则</label>
          </div>
          <div class="form-row">
            <label>Cron 表达式</label>
            <el-input :model-value="cronJobConfig.schedule" @update:model-value="updateCronJobConfig('schedule', $event)" size="small" placeholder="例如: */5 * * * *" />
            <div class="form-tip">Cron 表达式格式: 分 时 日 月 周</div>
          </div>

          <div class="schedule-examples">
            <div class="example-item" @click="applyScheduleExample('*/5 * * * *')">
              <span class="example-label">每 5 分钟</span>
              <span class="example-value">*/5 * * * *</span>
            </div>
            <div class="example-item" @click="applyScheduleExample('0 * * * *')">
              <span class="example-label">每小时</span>
              <span class="example-value">0 * * * *</span>
            </div>
            <div class="example-item" @click="applyScheduleExample('0 0 * * *')">
              <span class="example-label">每天零点</span>
              <span class="example-value">0 0 * * *</span>
            </div>
            <div class="example-item" @click="applyScheduleExample('0 2 * * *')">
              <span class="example-label">每天凌晨 2 点</span>
              <span class="example-value">0 2 * * *</span>
            </div>
            <div class="example-item" @click="applyScheduleExample('0 0 * * 0')">
              <span class="example-label">每周日零点</span>
              <span class="example-value">0 0 * * 0</span>
            </div>
            <div class="example-item" @click="applyScheduleExample('0 0 1 * *')">
              <span class="example-label">每月 1 号零点</span>
              <span class="example-value">0 0 1 * *</span>
            </div>
            <div class="example-item" @click="applyScheduleExample('0 0 * * 1-5')">
              <span class="example-label">工作日零点</span>
              <span class="example-value">0 0 * * 1-5</span>
            </div>
            <div class="example-item" @click="applyScheduleExample('*/30 9-17 * * 1-5')">
              <span class="example-label">工作日上班时间每30分钟</span>
              <span class="example-value">*/30 9-17 * * 1-5</span>
            </div>
          </div>
        </div>

        <div class="form-section">
          <div class="form-section-header">
            <label>并发策略</label>
          </div>
          <div class="form-row">
            <label>策略类型</label>
            <el-select :model-value="cronJobConfig.concurrencyPolicy" @update:model-value="updateCronJobConfig('concurrencyPolicy', $event)" size="small" style="width: 100%">
              <el-option label="Allow (允许并发运行)" value="Allow" />
              <el-option label="Forbid (禁止并发运行)" value="Forbid" />
              <el-option label="Replace (替换旧任务)" value="Replace" />
            </el-select>
            <div class="form-tip">
              Allow: 允许同时运行多个任务 | Forbid: 跳过新任务如果上次任务还在运行 | Replace: 替换正在运行的任务
            </div>
          </div>

          <div class="form-row">
            <label>暂停调度</label>
            <el-switch :model-value="cronJobConfig.suspend" @update:model-value="updateCronJobConfig('suspend', $event)" active-text="暂停" inactive-text="启用" />
            <div class="form-tip">暂停后不会创建新的 Job，但正在运行的 Job 不会受影响</div>
          </div>
        </div>

        <div class="form-section">
          <div class="form-section-header">
            <label>Job 任务配置</label>
          </div>
          <div class="form-row">
            <label>完成次数 (Completions)</label>
            <el-input-number :model-value="jobConfig.completions" @update:model-value="updateJobConfig('completions', $event)" :min="1" :max="1000" size="small" style="width: 100%" />
            <div class="form-tip">每次调度执行需要成功完成的 Pod 数量</div>
          </div>

          <div class="form-row">
            <label>并行度 (Parallelism)</label>
            <el-input-number :model-value="jobConfig.parallelism" @update:model-value="updateJobConfig('parallelism', $event)" :min="1" :max="100" size="small" style="width: 100%" />
            <div class="form-tip">每次调度同时运行的 Pod 最大数量</div>
          </div>

          <div class="form-row">
            <label>失败重试次数 (Backoff Limit)</label>
            <el-input-number :model-value="jobConfig.backoffLimit" @update:model-value="updateJobConfig('backoffLimit', $event)" :min="0" :max="20" size="small" style="width: 100%" />
            <div class="form-tip">Pod 失败后的重试次数</div>
          </div>

          <div class="form-row">
            <label>活跃终止时间 (Active Deadline Seconds)</label>
            <el-input-number :model-value="jobConfig.activeDeadlineSeconds" @update:model-value="updateJobConfig('activeDeadlineSeconds', $event)" :min="0" :max="86400" size="small" style="width: 100%" />
            <div class="form-tip">单次 Job 的最长运行时间（秒）</div>
          </div>
        </div>

        <div class="form-section">
          <div class="form-section-header">
            <label>历史记录限制</label>
          </div>
          <div class="form-row">
            <label>成功任务保留数</label>
            <el-input-number :model-value="cronJobConfig.successfulJobsHistoryLimit" @update:model-value="updateCronJobConfig('successfulJobsHistoryLimit', $event)" :min="0" :max="100" size="small" style="width: 100%" />
            <div class="form-tip">控制保留多少个已完成的 Job 记录</div>
          </div>

          <div class="form-row">
            <label>失败任务保留数</label>
            <el-input-number :model-value="cronJobConfig.failedJobsHistoryLimit" @update:model-value="updateCronJobConfig('failedJobsHistoryLimit', $event)" :min="0" :max="100" size="small" style="width: 100%" />
            <div class="form-tip">控制保留多少个失败的 Job 记录</div>
          </div>
        </div>

        <div class="form-section">
          <div class="form-section-header">
            <label>其他配置</label>
          </div>
          <div class="form-row">
            <label>启动截止时间 (秒)</label>
            <el-input-number :model-value="cronJobConfig.startingDeadlineSeconds" @update:model-value="updateCronJobConfig('startingDeadlineSeconds', $event)" :min="0" :max="300" size="small" style="width: 100%" />
            <div class="form-tip">如果任务错过调度时间超过此秒数，将不再执行。设置为 0 表示不限制</div>
          </div>

          <div class="form-row">
            <label>时区</label>
            <el-input :model-value="cronJobConfig.timeZone" @update:model-value="updateCronJobConfig('timeZone', $event)" size="small" placeholder="例如: Asia/Shanghai" />
            <div class="form-tip">留空使用集群默认时区</div>
          </div>
        </div>
      </template>

      <!-- Pod 扩容配置 -->
      <template v-else-if="workloadType === 'Pod'">
        <div class="form-section">
          <div class="form-section-header">
            <label>Pod 说明</label>
          </div>
          <div class="pod-info">
            <div class="info-item">
              <span class="info-label">类型</span>
              <span class="info-value">单个 Pod（无副本）</span>
            </div>
            <div class="info-item">
              <span class="info-label">生命周期</span>
              <span class="info-value">独立管理，不受控制器管理</span>
            </div>
            <div class="form-tip">
              Pod 是 Kubernetes 中最小的部署单元。此处创建的是独立 Pod，不支持副本数配置。
              如需自动管理副本，请使用 Deployment、StatefulSet 或 DaemonSet。
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface ScalingStrategy {
  strategyType: string
  maxSurge: string
  maxUnavailable: string
  minReadySeconds: number
  progressDeadlineSeconds: number
  revisionHistoryLimit: number
  timeoutSeconds?: number
}

interface JobConfig {
  completions: number
  parallelism: number
  backoffLimit: number
  activeDeadlineSeconds: number | null
}

interface CronJobConfig {
  schedule: string
  concurrencyPolicy: string
  timeZone: string
  successfulJobsHistoryLimit: number
  failedJobsHistoryLimit: number
  startingDeadlineSeconds: number | null
  suspend: boolean
}

interface FormData {
  replicas?: number
}

const props = defineProps<{
  workloadType: string
  formData: FormData
  scalingStrategy: ScalingStrategy
  jobConfig: JobConfig
  cronJobConfig: CronJobConfig
}>()

const emit = defineEmits<{
  'update:formData': [value: FormData]
  'update:scalingStrategy': [value: ScalingStrategy]
  'update:jobConfig': [value: JobConfig]
  'update:cronJobConfig': [value: CronJobConfig]
}>()

const panelTitle = computed(() => {
  switch (props.workloadType) {
    case 'Deployment':
    case 'StatefulSet':
      return '扩容配置'
    case 'DaemonSet':
      return 'DaemonSet 配置'
    case 'Job':
      return 'Job 任务配置'
    case 'CronJob':
      return 'CronJob 配置'
    case 'Pod':
      return 'Pod 说明'
    default:
      return '配置'
  }
})

// 更新副本数
const updateReplicas = (value: number) => {
  emit('update:formData', { ...props.formData, replicas: value })
}

// 更新扩缩容策略
const updateScalingStrategy = (field: string, value: any) => {
  emit('update:scalingStrategy', { ...props.scalingStrategy, [field]: value })
}

// 更新 Job 配置
const updateJobConfig = (field: string, value: any) => {
  emit('update:jobConfig', { ...props.jobConfig, [field]: value })
}

// 更新 CronJob 配置
const updateCronJobConfig = (field: string, value: any) => {
  emit('update:cronJobConfig', { ...props.cronJobConfig, [field]: value })
}

// 应用 Job 示例
const applyJobExample = (type: string) => {
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

  emit('update:jobConfig', config)
}

// 应用调度示例
const applyScheduleExample = (schedule: string) => {
  emit('update:cronJobConfig', {
    ...props.cronJobConfig,
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

.scaling-panel {
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

.form-row .el-input,
.form-row .el-input-number,
.form-row .el-select {
  width: 100%;
}

.form-row .el-input :deep(.el-input__wrapper),
.form-row .el-input-number :deep(.el-input__wrapper),
.form-row .el-select :deep(.el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.form-row .el-input :deep(.el-input__wrapper:hover),
.form-row .el-input-number :deep(.el-input__wrapper:hover),
.form-row .el-select :deep(.el-input__wrapper:hover) {
  border-color: #ffffff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 3px rgba(212, 175, 55, 0.1);
}

.form-row .el-input :deep(.el-input__wrapper.is-focus),
.form-row .el-input-number :deep(.el-input__wrapper.is-focus),
.form-row .el-select :deep(.el-input__wrapper.is-focus) {
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

.daemonset-info,
.pod-info {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: #ffffff;
  border-radius: 0;
  border: 1px solid #e8e8e8;
}

.info-label {
  font-size: 13px;
  font-weight: 600;
  color: #333;
}

.info-value {
  font-size: 13px;
  color: #666;
}

.schedule-examples {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
  margin-top: 12px;
}

.example-item {
  padding: 12px;
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

.example-label {
  font-size: 12px;
  color: #666;
}

.example-value {
  font-size: 13px;
  font-weight: 600;
  color: #ffffff;
  font-family: 'Courier New', monospace;
}

.job-examples {
  margin-top: 16px;
}

.example-header {
  font-size: 13px;
  font-weight: 600;
  color: #333;
  margin-bottom: 10px;
}

.job-examples .example-item {
  flex-direction: row;
  justify-content: space-between;
  align-items: center;
}

.job-examples .example-title {
  font-size: 13px;
  font-weight: 600;
  color: #333;
}

.job-examples .example-config {
  font-size: 11px;
  color: #ffffff;
  font-family: 'Courier New', monospace;
}
</style>
