<template>
  <div class="scaling-strategy-wrapper">
    <div class="strategy-section">
      <label class="section-label">更新策略</label>
      <el-radio-group v-model="localFormData.strategyType" @change="handleUpdate" class="strategy-radio">
        <el-radio value="RollingUpdate" class="strategy-radio-item">
          <span class="radio-label">滚动升级 (RollingUpdate)</span>
        </el-radio>
        <el-radio value="Recreate" class="strategy-radio-item">
          <span class="radio-label">重新创建 (Recreate)</span>
        </el-radio>
      </el-radio-group>
    </div>

    <!-- 滚动升级配置 -->
    <div v-if="localFormData.strategyType === 'RollingUpdate'" class="strategy-form-grid">
      <div class="form-grid-row">
        <div class="form-grid-item">
          <label class="form-grid-label">最大激增 Pod 数</label>
          <el-input
            v-model="localFormData.maxSurge"
            placeholder="例如: 3 或 25%"
            class="grid-input"
            @input="handleUpdate"
          />
          <div class="form-tip">升级过程中最多可以比期望副本数多的 Pod 数量</div>
        </div>
        <div class="form-grid-item">
          <label class="form-grid-label">最大不可用 Pod 数</label>
          <el-input
            v-model="localFormData.maxUnavailable"
            placeholder="例如: 1 或 25%"
            class="grid-input"
            @input="handleUpdate"
          />
          <div class="form-tip">升级过程中最多可以有多少个 Pod 不可用</div>
        </div>
      </div>
    </div>

    <!-- 通用配置 -->
    <div class="strategy-form-grid">
      <div class="form-grid-row">
        <div class="form-grid-item">
          <label class="form-grid-label">最小就绪时间(秒)</label>
          <el-input-number
            v-model="localFormData.minReadySeconds"
            :min="0"
            :max="3600"
            placeholder="0"
            class="grid-input"
            @change="handleUpdate"
          />
          <div class="form-tip">新 Pod 就绪后最少等待多少秒才继续升级</div>
        </div>
        <div class="form-grid-item">
          <label class="form-grid-label">进度截止时间(秒)</label>
          <el-input-number
            v-model="localFormData.progressDeadlineSeconds"
            :min="60"
            :max="3600"
            placeholder="600"
            class="grid-input"
            @change="handleUpdate"
          />
          <div class="form-tip">升级失败前等待的最长时间</div>
        </div>
      </div>
      <div class="form-grid-row">
        <div class="form-grid-item">
          <label class="form-grid-label">修订历史限制</label>
          <el-input-number
            v-model="localFormData.revisionHistoryLimit"
            :min="0"
            :max="100"
            placeholder="10"
            class="grid-input"
            @change="handleUpdate"
          />
          <div class="form-tip">保留多少个旧版本的 ReplicaSet</div>
        </div>
        <div class="form-grid-item">
          <label class="form-grid-label">超时时间(秒)</label>
          <el-input-number
            v-model="localFormData.timeoutSeconds"
            :min="1"
            :max="3600"
            placeholder="600"
            class="grid-input"
            @change="handleUpdate"
          />
          <div class="form-tip">升级操作的默认超时时间</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'

interface FormData {
  strategyType: string
  maxSurge: string | number
  maxUnavailable: string | number
  minReadySeconds: number
  progressDeadlineSeconds: number
  revisionHistoryLimit: number
  timeoutSeconds: number
}

const props = defineProps<{
  formData: FormData
}>()

const emit = defineEmits<{
  update: [data: FormData]
}>()

// 创建本地响应式数据
const localFormData = reactive<FormData>({
  strategyType: props.formData.strategyType ?? 'RollingUpdate',
  maxSurge: props.formData.maxSurge !== undefined ? props.formData.maxSurge : '25%',
  maxUnavailable: props.formData.maxUnavailable !== undefined ? props.formData.maxUnavailable : '25%',
  minReadySeconds: props.formData.minReadySeconds ?? 0,
  progressDeadlineSeconds: props.formData.progressDeadlineSeconds ?? 600,
  revisionHistoryLimit: props.formData.revisionHistoryLimit ?? 10,
  timeoutSeconds: props.formData.timeoutSeconds ?? 600
})

const handleUpdate = () => {
  emit('update', { ...localFormData })
}

// 监听 props 变化
watch(() => props.formData, (newVal) => {
  Object.assign(localFormData, newVal)
}, { deep: true })
</script>

<style scoped>
.scaling-strategy-wrapper {
  padding: 0;
  background: transparent;
}

.strategy-section {
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e4e7ed;
}

.strategy-section:last-child {
  border-bottom: none;
}

.section-label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  margin-bottom: 16px;
}

.strategy-radio {
  display: flex;
  gap: 16px;
}

.strategy-radio :deep(.el-radio-group) {
  display: flex;
  gap: 16px;
}

.strategy-radio-item {
  margin: 0 !important;
  padding: 12px 24px;
  border: 1px solid #e4e7ed;
  border-radius: 0;
  background: #fff;
  transition: all 0.3s;
  display: flex;
  align-items: center;
}

.strategy-radio-item:hover {
  border-color: #409eff;
  background: #ecf5ff;
}

.strategy-radio-item.is-checked {
  border-color: #409eff;
  background: #ecf5ff;
}

.strategy-radio-item :deep(.el-radio__label) {
  padding-left: 8px;
}

.strategy-radio-item :deep(.el-radio__input) {
  transform: scale(1.15);
}

.strategy-radio-item :deep(.el-radio__input.is-checked .el-radio__inner) {
  background: #409eff;
  border-color: #409eff;
}

.strategy-radio-item :deep(.el-radio__inner) {
  border-color: #dcdfe6;
  transition: all 0.3s;
}

.radio-label {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.strategy-form-grid {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-grid-row {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}

.form-grid-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-grid-item.full-width {
  grid-column: 1 / -1;
}

.form-grid-label {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  line-height: 1.5;
}

.grid-input {
  width: 100%;
}

.grid-input :deep(.el-input__wrapper) {
  border-radius: 0;
  border: 1px solid #e4e7ed;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  transition: all 0.3s;
  background-color: #fff;
  padding: 8px 15px;
}

.grid-input :deep(.el-input__wrapper:hover) {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.2);
}

.grid-input :deep(.el-input__wrapper.is-focus) {
  border-color: #409eff;
  box-shadow: 0 2px 12px rgba(64, 158, 255, 0.3);
}

.grid-input :deep(.el-input__inner) {
  font-size: 14px;
  color: #303133;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
  margin-top: 4px;
}
</style>
