<template>
  <el-alert
    v-if="kubernetesStore.isReadOnly"
    type="warning"
    :closable="false"
    show-icon
    class="readonly-banner"
  >
    <template #title>
      <span class="readonly-title">只读模式</span>
    </template>
    <span class="readonly-desc">
      当前用户仅有查看权限，无法执行创建、编辑、删除等操作。如需写入权限，请联系管理员在「集群授权」中为您分配
      <strong>namespace-owner</strong> 或 <strong>cluster-owner</strong> 角色。
    </span>
  </el-alert>
</template>

<script setup lang="ts">
import { useKubernetesStore } from '@/stores/kubernetes'

const kubernetesStore = useKubernetesStore()
</script>

<style scoped>
.readonly-banner {
  margin-bottom: 12px;
  border-radius: 8px;
}
.readonly-title {
  font-weight: 600;
  font-size: 14px;
}
.readonly-desc {
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
}
</style>
