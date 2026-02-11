<template>
  <div class="workloads-container">
    <!-- Page Header -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Document /></el-icon>
        </div>
        <div>
          <h2 class="page-title">{{ crdName }}</h2>
          <p class="page-subtitle">{{ group }}/{{ version }} ({{ kind }})</p>
        </div>
      </div>
      <div class="header-actions">
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
          placeholder="搜索资源名称..."
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

        <el-select
          v-if="isNamespaced"
          v-model="listQuery.namespace"
          placeholder="所有命名空间"
          clearable
          filterable
          class="namespace-select"
          @change="getList"
        >
          <template #prefix>
             <el-icon class="search-icon"><FolderOpened /></el-icon>
          </template>
          <el-option v-if="kubernetesStore.fullNamespaceAccess" label="所有命名空间" value="" />
          <el-option v-for="item in namespaces" :key="item" :label="item" :value="item" />
        </el-select>
        <el-button class="black-button" @click="handleBack">
          <el-icon style="margin-right: 4px;"><Back /></el-icon>
          返回
        </el-button>
      </div>

       <div class="action-buttons">
         <!-- Create button removed as requested -->
      </div>
    </div>

    <!-- Table -->
    <div class="table-wrapper">
      <el-table
        v-loading="listLoading"
        :data="filteredData"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        :row-style="{ height: '56px' }"
        :cell-style="{ padding: '8px 0' }"
      >
        <el-table-column label="名称" prop="metadata.name" align="left" min-width="260">
          <template #default="{ row }">
             <div class="name-cell">
               <div class="resource-icon-box">
                  <el-icon class="resource-icon-gold"><Document /></el-icon>
               </div>
               <div class="name-content">
                   <div class="resource-name">{{ row.metadata.name }}</div>
               </div>
             </div>
          </template>
        </el-table-column>
        <el-table-column label="命名空间" prop="metadata.namespace" align="center" width="150" v-if="isNamespaced" />
        <el-table-column label="存活时间" align="center" width="150">
          <template #default="{ row }">
            <div class="time-cell">
               <span>{{ calculateAge(row.metadata.creationTimestamp) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" align="center" width="180" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons-cell">
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn" @click="handleUpdate(row)">
                  <el-icon :size="18"><Edit /></el-icon>
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
    </div>

    <!-- Create/Update Dialog -->
    <el-dialog
      :title="dialogStatus === 'create' ? '创建 Custom Resource' : '编辑 Custom Resource'"
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
          <el-button @click="dialogFormVisible = false">取消</el-button>
          <el-button type="primary" class="black-button" @click="dialogStatus === 'create' ? createData() : updateData()">
            确定
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
import { getCustomResources, createCustomResource, updateCustomResource, deleteCustomResource, getCustomResource, getNamespaces } from '@/api/kubernetes'
import MonacoEditor from '@/components/YamlEditor.vue'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import * as yaml from 'js-yaml'
import { useKubernetesStore } from '@/stores/kubernetes'
import {
  Back,
  Refresh,
  Search,
  FolderOpened,
  Document,
  Clock,
  Edit,
  Delete,
  Plus
} from '@element-plus/icons-vue'

dayjs.extend(relativeTime)

const router = useRouter()
const route = useRoute()
const kubernetesStore = useKubernetesStore()

const list = ref<any[]>([])
const listLoading = ref(false)
const namespaces = ref<string[]>([])
const listQuery = reactive({
  name: '',
  namespace: ''
})
const dialogFormVisible = ref(false)
const dialogStatus = ref('')
const tempYaml = ref('')

const crdName = computed(() => route.params.crdName as string)
const currentClusterId = computed(() => Number(route.query.clusterId) || kubernetesStore.selectedClusterId)
const group = computed(() => route.query.group as string)
const version = computed(() => route.query.version as string)
const resource = computed(() => route.query.plural as string)
const kind = computed(() => route.query.kind as string)
const isNamespaced = computed(() => route.query.scope === 'Namespaced')

// Client-side filtering
const filteredData = computed(() => {
    let data = list.value
    // If we have client-side data, filtering by namespace locally is redundant if API already filtered,
    // but if we fetched ALL (namespace=''), we can filter locally OR re-fetch.
    // The previous implementation filtered locally. 
    // To ensure consistency with "Manage Resources" issue, we rely on API fetching mostly, 
    // but if we stick to local filtering, we must ensure we fetched enough data.
    // For now, getList handles API filtering.
    // But if listQuery.namespace is set, getList fetches that namespace.
    // So usually data contains only that namespace.
    
    // Name filtering is typically client side for small lists
    if (listQuery.name) {
        data = data.filter(item => item.metadata.name.toLowerCase().includes(listQuery.name.toLowerCase()))
    }
    return data
})

onMounted(async () => {
    if (currentClusterId.value) {
        if (isNamespaced.value) {
           await loadNamespaces()
        }
        getList()
    }
})

watch(currentClusterId, async (newVal) => {
  if (newVal) {
    if (isNamespaced.value) {
       await loadNamespaces()
    }
    getList()
  }
})

const loadNamespaces = async () => {
    if (!currentClusterId.value) return
    try {
        const res = await getNamespaces(currentClusterId.value)
        namespaces.value = res.data.map((ns: any) => ns.name)
    } catch (e) {
        // console.error(e)
    }
}

const getList = async () => {
  if (!currentClusterId.value) return
  listLoading.value = true
  try {
    // FIX: Pass undefined if listQuery.namespace is empty string, so API can handle "all namespaces"
    const namespaceParam = listQuery.namespace || undefined
    
    const res = await getCustomResources(
        currentClusterId.value,
        group.value,
        version.value,
        resource.value,
        namespaceParam
    )
    list.value = res || []
  } catch (error) {
    ElMessage.error('获取资源列表失败')
  } finally {
    listLoading.value = false
  }
}

const handleFilter = () => {
    // computed
}

const handleBack = () => {
    if (!currentClusterId.value) {
        router.back() // Fallback
        return
    }
    router.push({
        name: 'K8sCustomResources',
        query: { clusterId: currentClusterId.value.toString() }
    })
}

const calculateAge = (timeStr: string) => {
    const diff = dayjs().diff(dayjs(timeStr), 'second')
    const days = Math.floor(diff / 86400)
    const hours = Math.floor((diff % 86400) / 3600)
    const minutes = Math.floor((diff % 3600) / 60)
    const seconds = diff % 60

    if (days > 0) return `${days}d`
    if (hours > 0) return `${hours}h`
    if (minutes > 0) return `${minutes}m`
    return `${seconds}s`
}

const handleCreate = () => {
    dialogStatus.value = 'create'
    const template = {
        apiVersion: `${group.value}/${version.value}`,
        kind: kind.value,
        metadata: {
            name: 'example',
            ...(isNamespaced.value ? { namespace: 'default' } : {})
        },
        spec: {}
    }
    tempYaml.value = yaml.dump(template)
    dialogFormVisible.value = true
}

const handleUpdate = async (row: any) => {
    if (!currentClusterId.value) return
    try {
         const res = await getCustomResource(
             currentClusterId.value,
             group.value,
             version.value,
             resource.value,
             row.metadata.namespace || 'cluster', // generic placeholder logic
             row.metadata.name
         )
         tempYaml.value = yaml.dump(res)
         dialogStatus.value = 'update'
         dialogFormVisible.value = true
    } catch (e) {
        // console.error(e)
    }
}

const createData = async () => {
    if (!currentClusterId.value) return
    try {
        await createCustomResource(
            currentClusterId.value,
            group.value,
            version.value,
            resource.value,
            tempYaml.value
        )
        ElMessage.success('创建成功')
        dialogFormVisible.value = false
        getList()
    } catch (e) {
        console.error(e)
    }
}

const updateData = async () => {
    if (!currentClusterId.value) return
    try {
        await updateCustomResource(
            currentClusterId.value,
            group.value,
            version.value,
            resource.value,
            tempYaml.value
        )
         ElMessage.success('更新成功')
        dialogFormVisible.value = false
        getList()
    } catch (e) {
        console.error(e)
    }
}

const handleDelete = (row: any) => {
  ElMessageBox.confirm(`确定要删除 ${row.metadata.name} 吗?`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    if (!currentClusterId.value) return
    try {
      await deleteCustomResource(
          currentClusterId.value,
          group.value,
          version.value,
          resource.value,
          row.metadata.namespace || 'cluster',
          row.metadata.name
      )
      ElMessage.success('删除成功')
      getList()
    } catch (error) {
      console.error(error)
    }
  })
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
  transition: all 0.3s;
}

.page-title-icon:hover {
    background: #333;
    transform: scale(1.05);
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

.namespace-select {
  width: 200px;
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

.add-button {
  background: #0f69a6;
  border: none;
  border-radius: 0;
}

.add-button:hover {
    background: #0d5a87;
}

.back-button {
  background: #ffffff;
  border: 1px solid #dcdfe6;
  color: #606266;
  border-radius: 0;
}

.back-button:hover {
  color: #409eff;
  border-color: #c6e2ff;
  background-color: #ecf5ff;
}

.action-buttons-cell {
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

.resource-name {
  font-weight: 600;
  color: #303133;
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

/* Dialog */
.yaml-dialog :deep(.el-dialog__body) {
  padding: 20px;
}
</style>
