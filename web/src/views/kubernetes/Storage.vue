<template>
  <div class="storage-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><FolderOpened /></el-icon>
        </div>
        <div>
          <h2 class="page-title">存储管理</h2>
          <p class="page-subtitle">管理 Kubernetes PersistentVolumes、PersistentVolumeClaims 和 StorageClasses</p>
        </div>
      </div>
      <div class="header-actions">
        <el-select
          v-model="selectedClusterId"
          placeholder="选择集群"
          class="cluster-select"
          @change="handleClusterChange"
        >
          <template #prefix>
            <el-icon class="search-icon"><Platform /></el-icon>
          </template>
          <el-option
            v-for="cluster in clusterList"
            :key="cluster.id"
            :label="cluster.alias || cluster.name"
            :value="cluster.id"
          />
        </el-select>
        <el-button class="black-button" @click="loadCurrentResources">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 存储类型标签 -->
    <div class="storage-types-bar">
      <div
        v-for="type in storageTypes"
        :key="type.value"
        :class="['type-tab', { active: activeTab === type.value }]"
        @click="handleTabChange(type.value)"
      >
        <el-icon class="type-icon">
          <component :is="type.icon" />
        </el-icon>
        <span class="type-label">{{ type.label }}</span>
        <span v-if="type.count !== undefined" class="type-count">({{ type.count }})</span>
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="content-wrapper">
      <!-- PersistentVolumeClaims -->
      <PVCList
        v-show="activeTab === 'pvcs' && selectedClusterId"
        ref="pvcListRef"
        :clusterId="selectedClusterId"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('pvcs', count)"
      />

      <!-- PersistentVolumes -->
      <PVList
        v-show="activeTab === 'pvs' && selectedClusterId"
        ref="pvListRef"
        :clusterId="selectedClusterId"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('pvs', count)"
      />

      <!-- StorageClasses -->
      <StorageClassList
        v-show="activeTab === 'storageclasses' && selectedClusterId"
        ref="storageClassListRef"
        :clusterId="selectedClusterId"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('storageclasses', count)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Refresh, FolderOpened, Platform, Coin, Folder, Box } from '@element-plus/icons-vue'
import { getClusterList, type Cluster } from '@/api/kubernetes'
import axios from 'axios'
import PVCList from './storage-components/PVCList.vue'
import PVList from './storage-components/PVList.vue'
import StorageClassList from './storage-components/StorageClassList.vue'

// 存储类型定义
interface StorageType {
  label: string
  value: string
  icon: any
  count: number
}

const storageTypes = ref<StorageType[]>([
  { label: 'PersistentVolumeClaims', value: 'pvcs', icon: Coin, count: 0 },
  { label: 'PersistentVolumes', value: 'pvs', icon: Folder, count: 0 },
  { label: 'StorageClasses', value: 'storageclasses', icon: Box, count: 0 },
])

const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()
const activeTab = ref('pvcs')

// 子组件引用
const pvcListRef = ref()
const pvListRef = ref()
const storageClassListRef = ref()

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      const savedClusterId = localStorage.getItem('storage_selected_cluster_id')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
    }
  } catch (error) {
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('storage_selected_cluster_id', selectedClusterId.value.toString())
  }
}

// Tab 切换
const handleTabChange = (tab: string) => {
  activeTab.value = tab
  localStorage.setItem('storage_active_tab', tab)
}

// 更新资源数量
const updateCount = (type: string, count: number) => {
  const storageType = storageTypes.value.find(t => t.value === type)
  if (storageType) {
    storageType.count = count
  }
}

// 加载当前标签页的资源
const loadCurrentResources = () => {
  switch (activeTab.value) {
    case 'pvcs':
      pvcListRef.value?.loadData()
      break
    case 'pvs':
      pvListRef.value?.loadData()
      break
    case 'storageclasses':
      storageClassListRef.value?.loadData()
      break
  }
}

onMounted(async () => {
  await loadClusters()
  const savedTab = localStorage.getItem('storage_active_tab')
  if (savedTab) {
    activeTab.value = savedTab
  }
})
</script>

<style scoped>
.storage-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 48px;
  height: 48px;
  background: #0a466a;
  border-radius: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  font-size: 22px;
  flex-shrink: 0;
  border: none;
}

.page-title-icon .el-icon {
  font-size: 22px;
  color: #ffffff;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.cluster-select {
  width: 280px;
}

.black-button {
  background-color: #0a466a !important;
  color: #ffffff !important;
  border-color: #0a466a !important;
  border-radius: 0;
  padding: 10px 20px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #0d5a87 !important;
}

.search-icon {
  color: #909399;
}

/* 存储类型标签栏 */
.storage-types-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  flex-wrap: wrap;
}

.type-tab {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: #1a1a1a;
  color: #fff;
  border-radius: 0;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 14px;
  font-weight: 500;
}

.type-tab:hover {
  background: #2a2a2a;
}

.type-tab.active {
  background: #d4af37;
  color: #000;
  border: none;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
}

.type-icon {
  font-size: 16px;
}

.type-label {
  white-space: nowrap;
}

.type-count {
  font-size: 12px;
  opacity: 0.8;
  margin-left: 2px;
}

/* 内容区域 */
.content-wrapper {
  background: transparent;
}

.cluster-select :deep(.el-input__wrapper) {
  border-radius: 0;
}
</style>
