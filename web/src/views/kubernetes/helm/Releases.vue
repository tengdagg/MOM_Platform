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
            <el-tag v-if="uninstallingSet.has(row.namespace + '/' + row.name)" type="danger" size="small">
              <el-icon class="is-loading" style="margin-right: 4px;"><Loading /></el-icon>卸载中
            </el-tag>
            <el-tag v-else :type="getStatusType(row.status)" size="small">{{ row.status }}</el-tag>
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
                <el-button 
                  link 
                  class="action-btn danger" 
                  :disabled="uninstallingSet.has(row.namespace + '/' + row.name)"
                  @click.stop="uninstallRelease(row)"
                >卸载
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Detail Drawer -->
    <el-drawer v-model="drawerVisible" title="Release 详情" size="65%" destroy-on-close>
      <div v-loading="detailLoading" class="release-detail">
        <div v-if="currentRelease">
          <!-- 基本信息 -->
          <el-descriptions title="基本信息" :column="2" border>
            <el-descriptions-item label="名称">{{ currentRelease.name }}</el-descriptions-item>
            <el-descriptions-item label="命名空间">{{ currentRelease.namespace }}</el-descriptions-item>
            <el-descriptions-item label="版本">{{ currentRelease.revision }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="getStatusType(currentRelease.status)" size="small">{{ currentRelease.status }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Chart">{{ currentRelease.chart }}</el-descriptions-item>
            <el-descriptions-item label="App 版本">{{ currentRelease.appVersion }}</el-descriptions-item>
            <el-descriptions-item label="更新时间">{{ formatDate(currentRelease.updated) }}</el-descriptions-item>
          </el-descriptions>

          <!-- Tabs -->
          <el-tabs v-model="activeDetailTab" class="detail-tabs">
            <!-- Values Tab -->
            <el-tab-pane label="Values" name="values">
              <div class="tab-toolbar">
                <el-button v-if="!upgradeMode" type="primary" size="small" @click="startUpgrade">
                  <el-icon style="margin-right: 4px;"><Upload /></el-icon>Upgrade
                </el-button>
                <template v-if="upgradeMode">
                  <el-button type="success" size="small" :loading="upgrading" @click="confirmUpgrade">
                    <el-icon style="margin-right: 4px;"><Check /></el-icon>保存并升级
                  </el-button>
                  <el-button size="small" @click="cancelUpgrade">取消</el-button>
                </template>
              </div>
              <div class="editor-container">
                <MonacoEditor
                  v-model="valuesContent"
                  language="yaml"
                  :read-only="!upgradeMode"
                  :theme="upgradeMode ? 'vs-dark' : 'vs-dark'"
                />
              </div>
            </el-tab-pane>

            <!-- Resources Tab -->
            <el-tab-pane label="Resources" name="resources">
              <div class="resources-section" v-if="currentRelease.resources && Object.keys(currentRelease.resources).length > 0">
                <div v-for="(items, kind) in sortedResources" :key="kind" class="resource-group">
                  <h4 class="resource-kind-title">{{ kind }}</h4>
                  <el-table :data="items" size="small" class="resource-table" :show-header="true">
                    <el-table-column label="Name" min-width="250">
                      <template #default="{ row }">
                        <el-link type="primary" @click="showResourceDetail(kind as string, row)">{{ row.name }}</el-link>
                      </template>
                    </el-table-column>
                    <el-table-column label="Namespace" prop="namespace" width="200">
                      <template #default="{ row }">
                        {{ row.namespace || currentRelease?.namespace || '-' }}
                      </template>
                    </el-table-column>
                  </el-table>
                </div>
              </div>
              <el-empty v-else description="No Resources" />
            </el-tab-pane>

            <!-- Notes Tab -->
            <el-tab-pane label="Notes" name="notes">
              <pre class="notes-content">{{ currentRelease.notes || 'No notes available.' }}</pre>
            </el-tab-pane>
          </el-tabs>
        </div>
      </div>
    </el-drawer>

    <!-- Resource Detail Drawer -->
    <ResourceDetailDrawer
      v-model="resourceDrawerVisible"
      :cluster-id="props.clusterId"
      :kind="resourceKind"
      :name="resourceName"
      :namespace="resourceNamespace"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, reactive } from 'vue'
import { Search, Ship, Loading, Upload, Check } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import request from '@/utils/request'
import { getNamespaces } from '@/api/kubernetes'
import { useKubernetesStore } from '@/stores/kubernetes'
import MonacoEditor from '@/components/YamlEditor.vue'
import ResourceDetailDrawer from '@/components/ResourceDetailDrawer.vue'

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

// Values / Upgrade 相关
const valuesContent = ref('')
const upgradeMode = ref(false)
const upgrading = ref(false)
const originalUserValues = ref('')

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

// 按 Kind 排序的资源
const sortedResources = computed(() => {
  if (!currentRelease.value?.resources) return {}
  const order = [
    'ServiceAccount', 'ConfigMap', 'Secret', 'PersistentVolumeClaim',
    'ClusterRole', 'ClusterRoleBinding', 'Role', 'RoleBinding',
    'Service', 'Deployment', 'StatefulSet', 'DaemonSet', 'Job', 'CronJob',
    'Ingress', 'NetworkPolicy', 'HorizontalPodAutoscaler', 'PodDisruptionBudget'
  ]
  const res = currentRelease.value.resources as Record<string, any[]>
  const sorted: Record<string, any[]> = {}
  // 先按预定义顺序
  for (const kind of order) {
    if (res[kind]) {
      sorted[kind] = res[kind]
    }
  }
  // 再添加未在预定义顺序中的
  for (const kind of Object.keys(res)) {
    if (!sorted[kind]) {
      sorted[kind] = res[kind]
    }
  }
  return sorted
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
  upgradeMode.value = false
  try {
    const res = await request.get(`/api/v1/plugins/kubernetes/helm/clusters/${props.clusterId}/releases/${row.namespace}/${row.name}`)
    currentRelease.value = res
    // 默认显示所有合并后的 values
    valuesContent.value = res?.values || ''
    originalUserValues.value = res?.userValues || ''
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 Release 详情失败')
  } finally {
    detailLoading.value = false
  }
}

// ========== Upgrade 功能 ==========
const startUpgrade = () => {
  upgradeMode.value = true
  // 保持显示完整的 computed values 供用户编辑修改
  // valuesContent 此时已经是完整的 values，无需切换
}

const cancelUpgrade = () => {
  upgradeMode.value = false
  // 恢复为所有合并后的 values
  valuesContent.value = currentRelease.value?.values || ''
}

const confirmUpgrade = async () => {
  if (!currentRelease.value || !props.clusterId) return

  try {
    await ElMessageBox.confirm(
      `确定要升级 Release "${currentRelease.value.name}" 吗？`,
      '确认升级',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return // 用户取消
  }

  upgrading.value = true
  try {
    await request.put(
      `/api/v1/plugins/kubernetes/helm/clusters/${props.clusterId}/releases/${currentRelease.value.namespace}/${currentRelease.value.name}`,
      { values: valuesContent.value }
    )
    ElMessage.success('升级成功')
    upgradeMode.value = false
    // 刷新详情
    await showDetail(currentRelease.value)
    // 刷新列表
    fetchReleases()
  } catch (error: any) {
    ElMessage.error('升级失败: ' + (error?.message || '未知错误'))
  } finally {
    upgrading.value = false
  }
}

// ========== 资源详情 Drawer ==========
const resourceDrawerVisible = ref(false)
const resourceKind = ref('')
const resourceName = ref('')
const resourceNamespace = ref('')

const showResourceDetail = (kind: string, item: any) => {
  resourceKind.value = kind
  resourceName.value = item.name
  resourceNamespace.value = item.namespace || currentRelease.value?.namespace || ''
  resourceDrawerVisible.value = true
}

// ========== 卸载 ==========
const uninstallingSet = reactive(new Set<string>())

const uninstallRelease = async (row: any) => {
  const releaseKey = `${row.namespace}/${row.name}`
  try {
    await ElMessageBox.confirm(`确定卸载 Release ${row.name}？`, '警告', {
      confirmButtonText: '卸载',
      cancelButtonText: '取消',
      type: 'warning',
    })

    uninstallingSet.add(releaseKey)

    const notification = ElNotification({
      title: `卸载 ${row.name}`,
      message: '正在删除 Release 资源...',
      type: 'info',
      duration: 0,
      showClose: false,
    })

    try {
      await request.delete(`/api/v1/plugins/kubernetes/helm/clusters/${props.clusterId}/releases/${row.namespace}/${row.name}`)
      
      notification.close()
      ElNotification({
        title: `卸载 ${row.name}`,
        message: 'Release 已成功卸载',
        type: 'success',
        duration: 3000,
      })
      fetchReleases()
    } catch (error: any) {
      notification.close()
      const errorMsg = error?.response?.data?.error || error?.message || '卸载失败'
      ElNotification({
        title: `卸载 ${row.name} 失败`,
        message: errorMsg,
        type: 'error',
        duration: 5000,
      })
    } finally {
      uninstallingSet.delete(releaseKey)
    }
  } catch (error) {
    // User cancelled
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

/* 搜索栏样式 */
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

/* ========== Detail Drawer ========== */
.release-detail {
  padding: 0 4px;
}

.detail-tabs {
  margin-top: 20px;
}

/* Tab 工具栏 */
.tab-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 10px;
  gap: 8px;
}

/* Monaco Editor 容器 */
.editor-container {
  height: 500px;
  border: 1px solid #dcdfe6;
  border-radius: 2px;
  overflow: hidden;
}

/* ========== Resources Tab ========== */
.resources-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.resource-group {
  background: #1e1e1e;
  border-radius: 4px;
  overflow: hidden;
}

.resource-kind-title {
  margin: 0;
  padding: 10px 16px;
  font-size: 14px;
  font-weight: 600;
  color: #e0e0e0;
  background: #2d2d2d;
  border-bottom: 1px solid #3a3a3a;
}

.resource-table {
  --el-table-bg-color: #1e1e1e;
  --el-table-tr-bg-color: #1e1e1e;
  --el-table-header-bg-color: #2a2a2a;
  --el-table-header-text-color: #b0b0b0;
  --el-table-text-color: #d0d0d0;
  --el-table-border-color: #3a3a3a;
  --el-table-row-hover-bg-color: #2a2a2a;
}

.resource-table :deep(.el-table__header th) {
  background: #2a2a2a !important;
  color: #b0b0b0;
  font-weight: 600;
  font-size: 13px;
}

.resource-table :deep(.el-table__body td) {
  border-bottom-color: #3a3a3a;
}

.resource-table :deep(.el-link--primary) {
  color: #58a6ff;
}

.resource-table :deep(.el-link--primary:hover) {
  color: #79b8ff;
}

/* Notes 内容样式 */
.notes-content {
  background-color: #1e1e1e;
  color: #d0d0d0;
  padding: 16px;
  border-radius: 4px;
  overflow-x: auto;
  white-space: pre-wrap;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  border: 1px solid #3a3a3a;
}

/* Uninstalling state */
.is-loading {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

</style>
