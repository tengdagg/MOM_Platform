<template>
  <div class="releases-list">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-bar-left">
        <el-input
          v-model="searchQuery"
          placeholder="搜索 Release 名称..."
          clearable
          class="search-input"
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select 
          v-model="selectedNamespaces" 
          placeholder="命名空间" 
          multiple
          collapse-tags
          collapse-tags-tooltip
          clearable 
          filterable
          @change="handleNamespaceChange" 
          class="filter-select"
        >
          <el-option label="所有命名空间" value="" />
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
      </div>
    </div>

    <!-- Release 列表 -->
    <div class="table-wrapper">
      <el-table
        :data="filteredReleases"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        @row-click="showDetail"
        highlight-current-row
      >
        <el-table-column label="名称" prop="name" min-width="200" fixed>
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-blue"><Ship /></el-icon>
              名称
            </span>
          </template>
          <template #default="{ row }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <el-icon class="name-icon" :size="18"><Ship /></el-icon>
              </div>
              <div class="name-content">
                <div class="name-text">{{ row.name }}</div>
                <div class="namespace-text">{{ row.namespace }}</div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="状态" prop="status" width="120" align="center">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="版本" prop="revision" width="80" align="center">
          <template #default="{ row }">
            <el-tag type="info" size="small">{{ row.revision }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Chart" prop="chart" min-width="150" />

        <el-table-column label="App 版本" prop="appVersion" width="120" />

        <el-table-column label="更新时间" prop="updated" width="180">
          <template #default="{ row }">
            {{ formatDate(row.updated) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="100" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="卸载" placement="top">
                <el-button link class="action-btn danger" @click.stop="uninstallRelease(row)">Uninstall
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Detail Drawer -->
    <el-drawer v-model="drawerVisible" title="Release 详情" size="60%">
      <div v-loading="detailLoading" class="release-detail">
        <div v-if="currentRelease">
          <el-descriptions title="基本信息" :column="2" border>
            <el-descriptions-item label="名称">{{ currentRelease.name }}</el-descriptions-item>
            <el-descriptions-item label="命名空间">{{ currentRelease.namespace }}</el-descriptions-item>
            <el-descriptions-item label="版本">{{ currentRelease.revision }}</el-descriptions-item>
            <el-descriptions-item label="状态">{{ currentRelease.status }}</el-descriptions-item>
            <el-descriptions-item label="Chart">{{ currentRelease.chart }}</el-descriptions-item>
            <el-descriptions-item label="App 版本">{{ currentRelease.appVersion }}</el-descriptions-item>
            <el-descriptions-item label="更新时间">{{ formatDate(currentRelease.updated) }}</el-descriptions-item>
          </el-descriptions>

          <el-tabs v-model="activeDetailTab" style="margin-top: 20px">
            <el-tab-pane label="Values" name="values">
              <el-input
                v-model="currentRelease.values"
                type="textarea"
                :rows="20"
                readonly
                style="font-family: monospace;"
              />
            </el-tab-pane>
            <el-tab-pane label="Manifest" name="manifest">
               <el-input
                v-model="currentRelease.manifest"
                type="textarea"
                :rows="20"
                readonly
                style="font-family: monospace;"
              />
            </el-tab-pane>
            <el-tab-pane label="Notes" name="notes">
               <pre class="notes-content">{{ currentRelease.notes }}</pre>
            </el-tab-pane>
          </el-tabs>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { Refresh, Search, Delete, Ship } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'
import { getNamespaces } from '@/api/kubernetes'
import { useKubernetesStore } from '@/stores/kubernetes'

const props = defineProps<{
  clusterId: number | undefined
}>()

const kubernetesStore = useKubernetesStore()

const releases = ref<any[]>([])
const loading = ref(false)
const searchQuery = ref('')
const namespaces = ref<{ name: string }[]>([])

// 使用全局 store 的命名空间选择
const selectedNamespaces = computed({
  get: () => kubernetesStore.selectedNamespaces,
  set: (val: string[]) => kubernetesStore.setNamespaces(val)
})

const drawerVisible = ref(false)
const detailLoading = ref(false)
const currentRelease = ref<any>(null)
const activeDetailTab = ref('values')

// 处理命名空间变化
const handleNamespaceChange = (val: string[]) => {
  if (val.includes('')) {
    if (val[val.length - 1] === '') {
      kubernetesStore.setNamespaces([''])
    } else {
      const newVal = val.filter(v => v !== '')
      kubernetesStore.setNamespaces(newVal)
    }
  } else {
    if (val.length === 0) {
      kubernetesStore.setNamespaces([''])
    } else {
      kubernetesStore.setNamespaces(val)
    }
  }
}

// 处理搜索
const handleSearch = () => {
  // 客户端过滤，无需额外操作
}

// Filter releases by search query and namespace
const filteredReleases = computed(() => {
  return releases.value.filter(release => {
    const matchesSearch = !searchQuery.value || 
      release.name.toLowerCase().includes(searchQuery.value.toLowerCase())
    
    let matchesNamespace = true
    if (selectedNamespaces.value.length > 0 && !selectedNamespaces.value.includes('')) {
      matchesNamespace = selectedNamespaces.value.includes(release.namespace)
    }
    
    return matchesSearch && matchesNamespace
  })
})

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!props.clusterId) return
  try {
    const data = await getNamespaces(props.clusterId)
    namespaces.value = data || []
  } catch (error) {
    console.error('获取命名空间列表失败', error)
  }
}

const fetchReleases = async () => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const res = await request.get(`/api/v1/plugins/kubernetes/helm/clusters/${props.clusterId}/releases`)
    releases.value = res || []
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 Releases 失败')
  } finally {
    loading.value = false
  }
}

watch(() => props.clusterId, (newVal) => {
  if (newVal) {
    searchQuery.value = ''
    loadNamespaces()
    fetchReleases()
  }
}, { immediate: true })

const getStatusType = (status: string) => {
  switch (status) {
    case 'deployed':
      return 'success'
    case 'failed':
      return 'danger'
    case 'pending-install':
    case 'pending-upgrade':
    case 'pending-rollback':
      return 'warning'
    default:
      return 'info'
  }
}

const formatDate = (dateStr: string) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  const h = String(date.getHours()).padStart(2, '0')
  const min = String(date.getMinutes()).padStart(2, '0')
  const s = String(date.getSeconds()).padStart(2, '0')
  return `${y}-${m}-${d} ${h}:${min}:${s}`
}

const showDetail = async (row: any) => {
  drawerVisible.value = true
  detailLoading.value = true
  activeDetailTab.value = 'values'
  try {
    const res = await request.get(`/api/v1/plugins/kubernetes/helm/clusters/${props.clusterId}/releases/${row.namespace}/${row.name}`)
    currentRelease.value = res
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 Release 详情失败')
  } finally {
    detailLoading.value = false
  }
}

const uninstallRelease = async (row: any) => {
  try {
      await ElMessageBox.confirm(`确定卸载 Release ${row.name}?`, '警告', {
          confirmButtonText: '卸载',
          cancelButtonText: '取消',
          type: 'warning',
      })
      
      await request.delete(`/api/v1/plugins/kubernetes/helm/clusters/${props.clusterId}/releases/${row.namespace}/${row.name}`)
      ElMessage.success('卸载已开始')
      fetchReleases()
  } catch (error) {
      if (error !== 'cancel') {
          console.error(error)
          ElMessage.error('卸载失败')
      }
  }
}

defineExpose({
  fetchReleases
})
</script>

<style scoped>
.releases-list {
  height: 100%;
  display: flex;
  flex-direction: column;
}

/* 搜索栏样式 - 与 ConfigMapList 保持一致 */
.search-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.search-bar-left {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-bar-right {
  display: flex;
  gap: 12px;
}

.search-input {
  width: 250px;
}

.filter-select {
  width: 200px;
}

.search-icon {
  color: #909399;
}

/* 表格容器 */
.table-wrapper {
  flex: 1;
  overflow: auto;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

/* 表头图标样式 */
.header-with-icon {
  display: flex;
  align-items: center;
  gap: 6px;
}

.header-icon {
  font-size: 16px;
}

.header-icon-blue {
  color: #ffffff;
}

/* 名称单元格样式 */
.name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.name-icon-wrapper {
  width: 36px;
  height: 36px;
  border-radius: 0;
  background: linear-gradient(135deg, #0a466a 0%, #0d5a87 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  flex-shrink: 0;
}

.name-icon {
  color: #fff;
}

.name-content {
  overflow: hidden;
}

.name-text {
  font-weight: 500;
  color: #303133;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.namespace-text {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}

/* 操作按钮样式 */
.action-buttons {
  display: flex;
  justify-content: center;
  gap: 8px;
}

.action-btn {
  padding: 4px;
  color: #606266;
}

.action-btn:hover {
  color: #409eff;
}

.action-btn.danger:hover {
  color: #f56c6c;
}

/* Notes 内容样式 */
.notes-content {
  background-color: var(--el-fill-color-light);
  padding: 15px;
  border-radius: 4px;
  overflow-x: auto;
  white-space: pre-wrap;
  font-family: monospace;
}
</style>
