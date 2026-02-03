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
        :namespace="namespaceParam"
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
import { ref, computed, onMounted } from 'vue'
import { Refresh, FolderOpened, Platform, Coin, Folder, Box } from '@element-plus/icons-vue'
import { getClusterList, getNamespaces, type Cluster } from '@/api/kubernetes'
import { useKubernetesStore } from '@/stores/kubernetes'
import axios from 'axios'
import PVCList from './storage-components/PVCList.vue'
import PVList from './storage-components/PVList.vue'
import StorageClassList from './storage-components/StorageClassList.vue'

// 使用全局 Kubernetes store
const kubernetesStore = useKubernetesStore()

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
const namespaceList = ref<any[]>([])
const selectedClusterId = ref<number>()
const activeTab = ref('pvcs')

// 使用 computed 双向绑定 store 的 selectedNamespaces
const selectedNamespaces = computed({
  get: () => kubernetesStore.selectedNamespaces,
  set: (val: string[]) => kubernetesStore.setNamespaces(val)
})

// 命名空间选择变化处理
const handleNamespaceChange = (namespaces: string[]) => {
  kubernetesStore.setNamespaces(namespaces)
  loadCurrentResources()
}

// 为子组件提供的命名空间参数
const namespaceParam = computed(() => {
  if (selectedNamespaces.value.length === 0) return ''
  return selectedNamespaces.value.join(',')
})

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

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!selectedClusterId.value) return
  try {
    const data = await getNamespaces(selectedClusterId.value)
    namespaceList.value = data || []
  } catch (error) {
    console.error('获取命名空间列表失败', error)
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('storage_selected_cluster_id', selectedClusterId.value.toString())
    await loadNamespaces()
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
  await loadNamespaces()
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
  width: 200px;
}

.namespace-select {
  width: 250px;
}

/* 按钮样式 - 使用全局样式 .black-button */

.search-icon {
  color: #909399;
}

/* 存储类型标签栏 - 使用全局样式 .type-tab */

/* 内容区域 */
.content-wrapper {
  background: transparent;
}

.cluster-select :deep(.el-input__wrapper) {
  border-radius: 0;
}
</style>
