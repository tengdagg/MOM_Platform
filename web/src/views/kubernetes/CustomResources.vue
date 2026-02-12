<template>
  <div class="workloads-container">
    <!-- Page Header -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Document /></el-icon>
        </div>
        <div>
          <h2 class="page-title">自定义资源 (CRD)</h2>
          <p class="page-subtitle">管理 Kubernetes 自定义资源定义</p>
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
        <el-button class="black-button" @click="getList">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- Action Bar -->
    <div class="action-bar">
      <div class="search-section">
        <el-input
          v-model="listQuery.name"
          placeholder="搜索 CRD 名称..."
          clearable
          class="search-input"
          @clear="handleFilter"
          @keyup.enter="handleFilter"
          @input="handleFilter"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>
        <el-button type="primary" class="add-button" @click="handleCreateCRD">
          <el-icon><Plus /></el-icon>
          创建 CRD
        </el-button>
      </div>
    </div>

    <!-- Table -->
    <div class="table-wrapper">
      <el-table
        v-loading="listLoading"
        :data="paginatedData"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        :row-style="{ height: '56px' }"
        :cell-style="{ padding: '8px 0' }"
      >
        <el-table-column label="名称" prop="name" align="left" min-width="280" fixed="left">
          <template #default="{ row }">
             <div class="name-cell">
               <div class="resource-icon-box">
                  <el-icon class="resource-icon-gold"><Document /></el-icon>
               </div>
               <div class="name-content">
                 <el-link type="primary" :underline="false" @click="handleViewCustomResources(row)" class="name-text">{{ row.name }}</el-link>
                 <div class="group-text">{{ row.group }}</div>
               </div>
             </div>
          </template>
        </el-table-column>
        
        <el-table-column 
            label="Group" 
            prop="group" 
            align="center" 
            width="220"
            :filters="groupFilters"
            :filter-method="filterGroup" 
        />
        
        <el-table-column label="Version" prop="version" align="center" width="120">
           <template #default="{ row }">
             <el-tag size="small" type="info" effect="plain" class="version-tag">{{ row.version }}</el-tag>
           </template>
        </el-table-column>
        <el-table-column label="Kind" prop="kind" align="center" width="180" />
        <el-table-column label="Scope" prop="scope" align="center" width="120">
           <template #default="{ row }">
             <el-tag size="small" :type="row.scope === 'Namespaced' ? 'success' : 'warning'">{{ row.scope }}</el-tag>
           </template>
        </el-table-column>
        <el-table-column label="存活时间" align="center" width="180">
          <template #default="{ row }">
            <div class="time-cell">
               <span>{{ formatAge(row.creationTimestamp) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" align="center" width="230" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="查看 YAML" placement="top">
                <el-button link class="action-btn" @click="handleViewYAML(row)">
                  <el-icon :size="18"><Document /></el-icon>
                 </el-button>
                </el-tooltip>
              <el-tooltip content="删除" placement="top">
              <el-button link class="action-btn danger" @click="handleDelete(row)">
                <el-icon :size="18"><Delete /></el-icon>
              </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>

       <!-- Pagination -->
       <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredData.length"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- YAML Dialog -->
    <el-dialog
      :title="dialogStatus === 'update' ? '编辑 YAML' : '查看 YAML'"
      v-model="dialogFormVisible"
      width="70%"
      :close-on-click-modal="false"
      class="yaml-dialog"
    >
      <div class="yaml-editor-container" style="height: 500px; border: 1px solid #dcdfe6;">
        <monaco-editor
          v-if="dialogFormVisible"
          v-model="tempYaml"
          language="yaml"
        />
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogFormVisible = false">关闭</el-button>
          <el-button type="primary" class="black-button" @click="createData" v-if="dialogStatus === 'create'">
             创建
          </el-button>
          <el-button type="primary" class="black-button" @click="updateData" v-if="dialogStatus === 'update'">
             更新
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getCRDs, deleteCRD, getCRD, createCRD, getClusterList, type Cluster } from '@/api/kubernetes'
import type { CRDInfo } from '@/api/kubernetes'
import MonacoEditor from '@/components/YamlEditor.vue'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import * as yaml from 'js-yaml'
import { useKubernetesStore } from '@/stores/kubernetes'

dayjs.extend(relativeTime)
import {
  Document,
  Platform,
  Refresh,
  Search,
  Clock,
  Delete,
  Tools,
  Plus
} from '@element-plus/icons-vue'


interface LeaseInfo {
  age: string
  createdAt: string
}

const router = useRouter()
const route = useRoute()
const kubernetesStore = useKubernetesStore()

const list = ref<CRDInfo[]>([])
const listLoading = ref(false)
const listQuery = reactive({
  name: ''
})
const dialogFormVisible = ref(false)
const dialogStatus = ref('')
const tempYaml = ref('')
const currentRow = ref<CRDInfo | null>(null)

// Pagination refs
const currentPage = ref(1)
const pageSize = ref(10)

// 集群列表相关
const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()

const currentClusterId = computed(() => {
  return selectedClusterId.value || Number(route.query.clusterId) || kubernetesStore.selectedClusterId
})

const filteredData = computed(() => {
  if (!listQuery.name) return list.value
  const lowerName = listQuery.name.toLowerCase()
  return list.value.filter(item => 
    item.name.toLowerCase().includes(lowerName) || 
    item.group.toLowerCase().includes(lowerName) ||
    item.kind.toLowerCase().includes(lowerName)
  )
})

const paginatedData = computed(() => {
    const start = (currentPage.value - 1) * pageSize.value
    const end = start + pageSize.value
    return filteredData.value.slice(start, end)
})

const groupFilters = computed(() => {
    const groups = new Set(filteredData.value.map(item => item.group))
    return Array.from(groups).map(group => ({ text: group, value: group })).sort((a, b) => a.text.localeCompare(b.text))
})

const filterGroup = (value: string, row: CRDInfo) => {
    return row.group === value
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      // 优先使用 URL 参数中的 clusterId
      const queryClusterId = Number(route.query.clusterId)
      // 其次使用 store 中保存的 clusterId
      const storeClusterId = kubernetesStore.selectedClusterId
      // 再次使用 localStorage 中保存的
      const savedClusterId = localStorage.getItem('crd_selected_cluster_id')
      
      if (queryClusterId && clusterList.value.some(c => c.id === queryClusterId)) {
        selectedClusterId.value = queryClusterId
      } else if (storeClusterId && clusterList.value.some(c => c.id === storeClusterId)) {
        selectedClusterId.value = storeClusterId
      } else if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
      await getList()
    }
  } catch (error) {
    ElMessage.error('获取集群列表失败')
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('crd_selected_cluster_id', selectedClusterId.value.toString())
    // 同步到全局 store
    kubernetesStore.setCluster(selectedClusterId.value)
  }
  await getList()
}

onMounted(() => {
  loadClusters()
})

// Watch for cluster ID changes from route
watch(() => route.query.clusterId, (newVal) => {
  if (newVal) {
    const clusterId = Number(newVal)
    if (clusterId && clusterList.value.some(c => c.id === clusterId)) {
      selectedClusterId.value = clusterId
      getList()
    }
  }
})

const getList = async () => {
  if (!currentClusterId.value) return
  listLoading.value = true
  try {
    const res = await getCRDs(currentClusterId.value)
    list.value = res || []
  } catch (error) {
    // optional
  } finally {
    listLoading.value = false
  }
}

const handleFilter = () => {
  currentPage.value = 1 // Reset to first page on search
}

const handlePageChange = (page: number) => {
  currentPage.value = page
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
}


const formatTime = (time: string) => {
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

const formatAge = (time: string) => {
  const diff = dayjs().diff(dayjs(time), 'second')
  const days = Math.floor(diff / 86400)
  const hours = Math.floor((diff % 86400) / 3600)
  const minutes = Math.floor((diff % 3600) / 60)
  const seconds = diff % 60

  if (days > 0) return `${days}d`
  if (hours > 0) return `${hours}h`
  if (minutes > 0) return `${minutes}m`
  return `${seconds}s`
}

const handleCreateCRD = () => {
    dialogStatus.value = 'create'
    const template = {
        apiVersion: "apiextensions.k8s.io/v1",
        kind: "CustomResourceDefinition",
        metadata: {
            name: "crontabs.stable.example.com"
        },
        spec: {
            group: "stable.example.com",
            versions: [
                {
                    name: "v1",
                    served: true,
                    storage: true,
                    schema: {
                        openAPIV3Schema: {
                            type: "object",
                            properties: {
                                spec: {
                                    type: "object",
                                    properties: {
                                        cronSpec: {
                                            type: "string"
                                        },
                                        image: {
                                            type: "string"
                                        },
                                        replicas: {
                                            type: "integer"
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            ],
            scope: "Namespaced",
            names: {
                plural: "crontabs",
                singular: "crontab",
                kind: "CronTab",
                shortNames: ["ex"]
            }
        }
    }
    tempYaml.value = yaml.dump(template)
    dialogFormVisible.value = true
}

const handleViewCustomResources = (row: CRDInfo) => {
  const plural = row.name.substring(0, row.name.length - row.group.length - 1)
  
  router.push({
    name: 'K8sCustomResourceList',
    params: {
        crdName: row.name
    },
    query: {
      clusterId: currentClusterId.value,
      group: row.group,
      version: row.version,
      kind: row.kind,
      plural: plural, 
      scope: row.scope
    }
  })
}

const handleViewYAML = async (row: CRDInfo) => {
  if (!currentClusterId.value) return
  try {
    const res = await getCRD(currentClusterId.value, row.name)
    tempYaml.value = yaml.dump(res)
    currentRow.value = row
    dialogStatus.value = 'view'

    dialogFormVisible.value = true
  } catch (error) {
    // console.error(error)
  }
}

const handleDelete = (row: CRDInfo) => {
  ElMessageBox.confirm(`确定要删除 CRD "${row.name}" 吗? 这将同时删除所有该类型的自定义资源！`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    if (!currentClusterId.value) return
    try {
      await deleteCRD(currentClusterId.value, row.name)
      ElMessage.success('删除成功')
      getList()
    } catch (error) {
      console.error(error)
    }
  })
}

const updateData = () => {
    // Implement CRD update if needed
    ElMessage.info('功能开发中')
}

const createData = async () => {
    if (!currentClusterId.value) return
    try {
        await createCRD(currentClusterId.value, tempYaml.value)
        ElMessage.success('创建成功')
        dialogFormVisible.value = false
        getList()
    } catch (error) {
        // console.error(error)
    }
}
</script>

<style scoped>
.workloads-container {
  padding: 0;
  background-color: transparent;
}

/* Page Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
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
  background: #000;
  border-radius: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  font-size: 22px;
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
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

/* Action Bar */
.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.search-section {
  display: flex;
  gap: 12px;
  align-items: center;
  flex: 1;
}

.search-input {
  width: 280px;
}

/* Buttons */
.black-button {
  background: #1a1a1a;
  border-color: #1a1a1a;
  color: #ffffff;
}

.black-button:hover {
  background: #333333;
  border-color: #0d5a87;
  color: #ffffff;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.25);
}

.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  font-weight: 500;
}

.view-btn {
  color: #409eff;
}

/* Table */
.table-wrapper {
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table :deep(.el-table__header th) {
  background: #fafbfc;
  color: #606266;
  font-weight: 600;
  height: 50px;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* Styled Icon Box */
.resource-icon-box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, #0a466a 0%, #0d5a87 100%);
  border-radius: 0;
  color: #ffffff;
  flex-shrink: 0;
}

.resource-icon-gold {
  font-size: 16px;
  color: #ffffff;
}

.name-content {
    overflow: hidden;
}

.name-text {
  font-weight: 600;
  color: #303133;
}

.group-text {
  font-size: 12px;
  color: #909399;
}


.time-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  color: #606266;
}

.search-icon {
  color: #909399;
}


/* Pagination */
.pagination-wrapper {
  padding: 16px 24px;
  display: flex;
  justify-content: flex-end;
  background: #fff;
  border-top: 1px solid #ebeef5;
}

/* Dialog */
.yaml-dialog :deep(.el-dialog__body) {
  padding: 20px;
}
</style>
