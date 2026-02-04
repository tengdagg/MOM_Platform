<template>
  <div class="list-container">
    <div class="search-bar">
      <div class="search-bar-left">
        <el-input
          v-model="searchName"
          placeholder="搜索 Lease 名称..."
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
      <div class="search-bar-right">
        <el-button class="blue-button" type="primary" @click="handleCreate">
            <el-icon style="margin-right: 6px;"><Plus /></el-icon>
            新增 Lease
        </el-button>
      </div>
    </div>

    <div class="table-wrapper">
      <el-table
        :data="paginatedData"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="名称" prop="name" min-width="200" fixed>
           <template #header>
            <span class="header-with-icon">
              名称
            </span>
          </template>
           <template #default="{ row }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <el-icon class="name-icon" :size="18"><Timer /></el-icon>
              </div>
              <div class="name-content">
                <div class="name-text">{{ row.name }}</div>
                <div class="namespace-text">{{ row.namespace }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="Holder Identity" prop="holderIdentity" min-width="200" show-overflow-tooltip />
        <el-table-column label="Lease Duration(s)" prop="leaseDuration" width="160" />
        <el-table-column label="Renew Time" prop="renewTime" width="180" />
        <el-table-column label="存活时间" prop="age" width="140" />
        <el-table-column label="操作" width="120" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑 YAML" placement="top">
                <el-button link class="action-btn" @click="handleEdit(row)">
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

       <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredData.length"
          layout="total, sizes, prev, pager, next"
        />
      </div>
    </div>

    <!-- YAML Dialog -->
    <el-dialog v-model="yamlDialogVisible" :title="yamlDialogTitle" width="800px">
      <YamlEditor
        v-if="yamlDialogVisible"
        v-model="yamlContent"
        language="yaml"
        theme="vs-dark"
        style="height: 500px"
      />
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="yamlDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSaveYAML" :loading="saving">保存</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import axios from 'axios'
import { Search, Timer, Plus, Document, Delete, Edit } from '@element-plus/icons-vue'
import { getNamespaces } from '@/api/kubernetes'
import { useKubernetesStore } from '@/stores/kubernetes'
import { ElMessage, ElMessageBox } from 'element-plus'
import YamlEditor from '@/components/YamlEditor.vue'
import * as yaml from 'js-yaml'

interface LeaseInfo {
  name: string
  namespace: string
  holderIdentity: string
  leaseDuration: number
  renewTime: string
  age: string
  createdAt: string
}

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['count-update'])
const loading = ref(false)
const listData = ref<LeaseInfo[]>([])
const namespaces = ref<{ name: string }[]>([])
const searchName = ref('')
const currentPage = ref(1)
const pageSize = ref(10)

const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const saving = ref(false)
const isCreateMode = ref(false)
const selectedItem = ref<LeaseInfo | null>(null)

const kubernetesStore = useKubernetesStore()

const selectedNamespaces = computed({
  get: () => kubernetesStore.selectedNamespaces,
  set: (val: string[]) => kubernetesStore.setNamespaces(val)
})

const yamlDialogTitle = computed(() => {
  return isCreateMode.value ? '创建 Lease' : `编辑 Lease - ${selectedItem.value?.name}`
})

const filteredData = computed(() => {
  let result = listData.value
  if (searchName.value) {
    result = result.filter(item => item.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  // 与 SecretList 保持一致的过滤逻辑
  if (selectedNamespaces.value.length > 0 && !selectedNamespaces.value.includes('')) {
    result = result.filter(item => selectedNamespaces.value.includes(item.namespace))
  }
  return result
})

const paginatedData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredData.value.slice(start, end)
})

const loadNamespacesData = async () => {
  if (!props.clusterId) return
  try {
    const data = await getNamespaces(props.clusterId)
    namespaces.value = data || []
  } catch (error) {
  }
}

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
  // 前端过滤，不需要重新加载数据
}

const loadData = async () => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const token = localStorage.getItem('token')
    // 与 SecretList 保持一致：总是获取所有数据，前端过滤
    const response = await axios.get('/api/v1/plugins/kubernetes/resources/leases', {
      params: { clusterId: props.clusterId },
      headers: { Authorization: `Bearer ${token}` }
    })
    listData.value = response.data.data || []
  } catch (error) {
    listData.value = []
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
}

const handleCreate = () => {
  isCreateMode.value = true
  selectedItem.value = null
  yamlContent.value = `apiVersion: coordination.k8s.io/v1
kind: Lease
metadata:
  name: example-lease
  namespace: default
spec:
  holderIdentity: "example-holder"
  leaseDurationSeconds: 15
  renewTime: "${new Date().toISOString()}"
`
  yamlDialogVisible.value = true
}

const handleEdit = async (row: LeaseInfo) => {
  isCreateMode.value = false
  selectedItem.value = row
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(`/api/v1/plugins/kubernetes/resources/leases/${row.namespace}/${row.name}/yaml`, {
      params: { clusterId: props.clusterId },
      headers: { Authorization: `Bearer ${token}` }
    })
    
    // Check structure of response using ConfigMapList pattern
    // Usually response.data.data.yaml is the string
    const data = response.data.data
    if (data.yaml) {
      yamlContent.value = data.yaml
    } else {
      // Fallback if returned as object
      yamlContent.value = yaml.dump(data)
    }
    
    yamlDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error('获取 YAML 失败: ' + (error.response?.data?.message || error.message))
  }
}

const handleSaveYAML = async () => {
  saving.value = true
  try {
    const token = localStorage.getItem('token')
    const yamlObj: any = yaml.load(yamlContent.value)
    if (!yamlObj || !yamlObj.metadata || !yamlObj.metadata.name) {
      ElMessage.error('Invalid YAML: missing metadata.name')
      saving.value = false
      return
    }
    
    const name = yamlObj.metadata.name
    const namespace = yamlObj.metadata.namespace || 'default'

    if (isCreateMode.value) {
      await axios.post(`/api/v1/plugins/kubernetes/resources/leases/${namespace}/yaml`, {
        clusterId: props.clusterId,
        yaml: yamlContent.value
      }, {
        headers: { Authorization: `Bearer ${token}` }
      })
      ElMessage.success('创建成功')
    } else {
      await axios.put(`/api/v1/plugins/kubernetes/resources/leases/${namespace}/${name}/yaml`, {
        clusterId: props.clusterId,
        yaml: yamlContent.value
      }, {
        headers: { Authorization: `Bearer ${token}` }
      })
      ElMessage.success('更新成功')
    }
    yamlDialogVisible.value = false
    loadData()
  } catch (error: any) {
    ElMessage.error('保存失败: ' + (error.response?.data?.message || error.message))
  } finally {
    saving.value = false
  }
}

const handleDelete = (row: LeaseInfo) => {
  ElMessageBox.confirm(
    `确定要删除 Lease ${row.name} 吗？`,
    '删除确认',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    try {
      const token = localStorage.getItem('token')
      await axios.delete(`/api/v1/plugins/kubernetes/resources/leases/${row.namespace}/${row.name}`, {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      })
      ElMessage.success('删除成功')
      loadData()
    } catch (error: any) {
      ElMessage.error('删除失败: ' + (error.response?.data?.message || error.message))
    }
  })
}

watch(() => props.clusterId, (newVal) => {
  if (newVal) {
      loadNamespacesData()
      loadData()
  }
}, { immediate: true })

// 监听过滤后的数据变化，更新计数（与 SecretList 一致）
watch(filteredData, (newData) => {
  emit('count-update', newData.length)
})

defineExpose({ loadLeases: loadData })
</script>

<style scoped>
.list-container {
  padding: 0;
}

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
  width: 280px;
}

.filter-select {
  width: 200px;
}

.search-icon {
  color: #909399;
}

.table-wrapper {
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__row) {
  height: 56px !important;
}

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
  color: #ffffff;
}

.name-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.name-text {
  font-weight: 600;
  color: #303133;
}

.namespace-text {
  font-size: 12px;
  color: #909399;
}

.header-with-icon {
  display: flex;
  align-items: center;
  gap: 6px;
}

.action-buttons {
  display: flex;
  justify-content: center;
  gap: 4px;
}

.action-btn {
  padding: 4px;
  color: #ffffff;
  transition: all 0.3s;
}

.action-btn:hover {
  color: #bfa13f;
}

.action-btn.danger {
  color: #f56c6c;
}

.action-btn.danger:hover {
  color: #f78989;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}
</style>
