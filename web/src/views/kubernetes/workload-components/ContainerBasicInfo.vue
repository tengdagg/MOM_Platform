<template>
  <el-row :gutter="20">
    <el-col :span="12">
      <el-form label-position="top" size="default">
        <el-form-item label="容器名称">
          <el-input v-model="localContainer.name" placeholder="输入容器名称" @input="update">
            <template #prefix>
              <el-icon><Box /></el-icon>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="容器镜像">
          <el-input v-model="localContainer.image" placeholder="例如: nginx:latest" @input="update">
            <template #prefix>
              <el-icon><Picture /></el-icon>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
    </el-col>
    <el-col :span="12">
      <el-form label-position="top" size="default">
        <el-form-item label="拉取策略">
          <el-select v-model="localContainer.imagePullPolicy" style="width: 100%" placeholder="选择拉取策略" @change="update">
            <el-option label="IfNotPresent - 本地不存在时拉取" value="IfNotPresent" />
            <el-option label="Always - 总是拉取最新" value="Always" />
            <el-option label="Never - 从不拉取" value="Never" />
          </el-select>
        </el-form-item>
      </el-form>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { Box, Picture } from '@element-plus/icons-vue'

interface Container {
  name: string
  image: string
  imagePullPolicy?: string
}

const props = defineProps<{
  container: Container
}>()

const emit = defineEmits<{
  update: [container: Container]
}>()

const localContainer = reactive({ ...props.container })

watch(() => props.container, (newVal) => {
  Object.assign(localContainer, newVal)
}, { deep: true })

const update = () => {
  emit('update', { ...localContainer })
}
</script>

<style scoped>
.el-row {
  padding: 20px;
  background: #ffffff;
  border-radius: 0;
  border: 1px solid #e8e8e8;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.el-col {
  padding: 0 8px;
}

.el-form-item {
  margin-bottom: 0;
}

.el-form-item :deep(.el-form-item__label) {
  font-weight: 600;
  color: #333;
  font-size: 13px;
  letter-spacing: 0.3px;
  margin-bottom: 8px;
}

.el-input :deep(.el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.el-input :deep(.el-input__wrapper:hover) {
  border-color: #ffffff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 3px rgba(212, 175, 55, 0.1);
}

.el-input :deep(.el-input__wrapper.is-focus) {
  border-color: #ffffff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 4px rgba(212, 175, 55, 0.15);
}

.el-input :deep(.el-input__prefix) {
  color: #ffffff;
}

.el-input :deep(.el-input__prefix-inner) {
  color: #ffffff;
}

.el-select :deep(.el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
}
</style>
