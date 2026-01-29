<template>
  <div class="resource-section">
    <div class="resource-group">
      <div class="group-header">
        <span class="group-title">CPU 限制</span>
      </div>
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="请求 (Request)">
            <el-input v-model="localResources.requests.cpu" placeholder="例如: 100m" @input="update" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="限制 (Limit)">
            <el-input v-model="localResources.limits.cpu" placeholder="例如: 500m 或 1" @input="update" />
          </el-form-item>
        </el-col>
      </el-row>
    </div>

    <div class="resource-group">
      <div class="group-header">
        <span class="group-title">内存限制</span>
      </div>
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="请求 (Request)">
            <el-input v-model="localResources.requests.memory" placeholder="例如: 128Mi" @input="update" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="限制 (Limit)">
            <el-input v-model="localResources.limits.memory" placeholder="例如: 512Mi" @input="update" />
          </el-form-item>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'

interface ResourceRequest {
  cpu?: string
  memory?: string
}

interface Resources {
  requests: ResourceRequest
  limits: ResourceRequest
}

const props = defineProps<{
  resources: Resources
}>()

const emit = defineEmits<{
  update: [resources: Resources]
}>()

const localResources = reactive<Resources>({
  requests: {
    cpu: '',
    memory: ''
  },
  limits: {
    cpu: '',
    memory: ''
  }
})

watch(() => props.resources, (newVal) => {
  if (newVal) {
    localResources.requests.cpu = newVal.requests?.cpu || ''
    localResources.requests.memory = newVal.requests?.memory || ''
    localResources.limits.cpu = newVal.limits?.cpu || ''
    localResources.limits.memory = newVal.limits?.memory || ''
  }
}, { deep: true, immediate: true })

const update = () => {
  emit('update', {
    requests: { ...localResources.requests },
    limits: { ...localResources.limits }
  })
}
</script>

<style scoped>
.resource-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.resource-group {
  padding: 20px;
  background: linear-gradient(135deg, #ffffff 0%, #fafafa 100%);
  border-radius: 0;
  border: 1px solid #e8e8e8;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.resource-group:hover {
  border-color: #ffffff;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
}

.group-header {
  margin-bottom: 16px;
}

.group-title {
  font-weight: 600;
  font-size: 15px;
  color: #1a1a1a;
  letter-spacing: 0.3px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.group-title::before {
  content: '';
  width: 4px;
  height: 18px;
  background: #d4af37;
  border-radius: 0;
  box-shadow: 0 0 8px rgba(212, 175, 55, 0.4);
}

.resource-group :deep(.el-form-item) {
  margin-bottom: 0;
}

.resource-group :deep(.el-form-item__label) {
  font-weight: 500;
  color: #666;
  font-size: 13px;
}

.resource-group :deep(.el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.resource-group :deep(.el-input__wrapper:hover) {
  border-color: #ffffff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 3px rgba(212, 175, 55, 0.1);
}

.resource-group :deep(.el-input__wrapper.is-focus) {
  border-color: #ffffff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 4px rgba(212, 175, 55, 0.15);
}
</style>
