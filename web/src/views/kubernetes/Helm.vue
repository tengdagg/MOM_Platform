<template>
  <div class="helm-container">
    <!-- Header -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><CustomIcons name="Helm" /></el-icon>
        </div>
        <div>
          <h2 class="page-title">Helm</h2>
          <p class="page-subtitle">Helm Charts and Releases Management</p>
        </div>
      </div>
      <div class="header-actions">
        <el-select
          v-model="selectedClusterId"
          placeholder="Select Cluster"
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
        <el-button class="black-button" @click="refreshCurrentView">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- Tab Bar -->
    <div class="network-types-bar">
      <div
        class="type-tab"
        :class="{ active: activeTab === 'charts' }"
        @click="activeTab = 'charts'"
      >
        <el-icon class="type-icon"><Shop /></el-icon>
        <span class="type-label">Charts</span>
      </div>
      <div
        class="type-tab"
        :class="{ active: activeTab === 'releases' }"
        @click="activeTab = 'releases'"
      >
        <el-icon class="type-icon"><List /></el-icon>
        <span class="type-label">Releases</span>
      </div>
    </div>

    <!-- Content -->
    <div class="content-wrapper">
      <div v-show="activeTab === 'charts'" class="tab-content">
        <Charts ref="chartsRef" :clusterId="selectedClusterId" />
      </div>
      <div v-show="activeTab === 'releases'" class="tab-content">
        <Releases ref="releasesRef" :clusterId="selectedClusterId" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, nextTick } from 'vue'
import { 
  Shop, 
  List, 
  Platform, 
  Refresh 
} from '@element-plus/icons-vue'
import CustomIcons from '@/components/icons/CustomIcons.vue'
import { ElMessage } from 'element-plus'
import { getClusterList, type Cluster } from '@/api/kubernetes'
import Charts from './helm/Charts.vue'
import Releases from './helm/Releases.vue'

const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()
const activeTab = ref('charts')
const chartsRef = ref()
const releasesRef = ref()

const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      // Restore selected cluster from local storage if needed, or default to first
      const savedClusterId = localStorage.getItem('helm_selected_cluster_id')
      if (savedClusterId) {
         const savedId = parseInt(savedClusterId)
         const exists = clusterList.value.some(c => c.id === savedId)
         selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
    }
  } catch (error) {
    ElMessage.error('Failed to load clusters')
  }
}

const handleClusterChange = () => {
  if (selectedClusterId.value) {
    localStorage.setItem('helm_selected_cluster_id', selectedClusterId.value.toString())
  }
}

const refreshCurrentView = () => {
  if (activeTab.value === 'charts') {
    chartsRef.value?.fetchRepos?.()
  } else if (activeTab.value === 'releases') {
    releasesRef.value?.fetchReleases?.()
  }
}

// Auto-refresh when switching tabs
watch(activeTab, (val) => {
  if (val === 'releases') {
    nextTick(() => {
      releasesRef.value?.fetchReleases?.()
    })
  }
})

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.helm-container {
  padding: 0;
  background-color: transparent;
  height: 100%;
  display: flex;
  flex-direction: column;
}

/* Page Header - Consistent with Network.vue */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  padding: 16px 20px;
  background: #fff;
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
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title-icon .el-icon {
  font-size: 22px;
  color: #ffffff;
}
/* Update icon usage in template */

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

.cluster-select :deep(.el-input__wrapper) {
  border-radius: 0;
}

/* Tab Bar - Use global styles .network-types-bar and .type-tab */

/* Content */
.content-wrapper {
  flex: 1;
  background: transparent;
  padding: 0;
  overflow: hidden; 
  display: flex;
  flex-direction: column;
}

.tab-content {
  height: 100%;
  overflow: hidden;
}
</style>
