<template>
  <div class="pdb-list">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-bar-left">
        <el-input
          v-model="searchName"
          placeholder="搜索 PodDisruptionBudget 名称..."
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
          <el-option v-if="kubernetesStore.fullNamespaceAccess" label="所有命名空间" value="" />
          <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </el-select>
      </div>

      <div class="search-bar-right">
        <el-button type="primary" class="black-button create-btn" @click="handleCreate">
          <el-icon style="margin-right: 4px;"><Plus /></el-icon>
          新增 PodDisruptionBudget
        </el-button>
      </div>
    </div>

    <!-- PodDisruptionBudget 列表 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedPDBs"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="名称" prop="name" min-width="180" fixed>
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-blue"><Lock /></el-icon>
              名称
            </span>
          </template>
          <template #default="{ row }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <el-icon class="name-icon" :size="18"><Lock /></el-icon>
              </div>
              <div class="name-content">
                <div class="name-text">{{ row.name }}</div>
                <div class="namespace-text">{{ row.namespace }}</div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="Min Available" prop="minAvailable" width="140" align="center">
          <template #default="{ row }">
            <span class="resource-value">{{ row.minAvailable || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Max Unavailable" prop="maxUnavailable" width="150" align="center">
          <template #default="{ row }">
            <span class="resource-value">{{ row.maxUnavailable || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Allowed Disruptions" prop="allowedDisruptions" width="170" align="center">
          <template #default="{ row }">
            <el-tag type="success" size="small">{{ row.allowedDisruptions ?? '-' }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Current Healthy" prop="currentHealthy" width="150" align="center">
          <template #default="{ row }">
            <el-tag :type="getHealthyTagType(row.currentHealthy, row.desiredHealthy)" size="small">
              {{ row.currentHealthy ?? '-' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Desired Healthy" prop="desiredHealthy" width="150" align="center">
          <template #default="{ row }">
            <el-tag type="info" size="small">{{ row.desiredHealthy ?? '-' }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" prop="createdAt" width="180">
          <template #default="{ row }">
            {{ row.createdAt || '-' }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="120" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑 YAML" placement="top">
                <el-button link class="action-btn" @click="handleEditYAML(row)">
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

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredPDBs.length"
          layout="total, sizes, prev, pager, next"
        />
      </div>
    </div>

    <!-- YAML 弹窗 -->
    <el-dialog v-model="yamlDialogVisible" :title="yamlDialogTitle" width="900px" class="yaml-dialog">
      <YamlEditor
        v-if="yamlDialogVisible"
        v-model="yamlContent"
        language="yaml"
        theme="vs-dark"
        style="height: 500px"
      />
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="yamlDialogVisible = false">关闭</el-button>
          <el-button type="primary" @click="handleSaveYAML" :loading="saving" class="black-button">保存</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Lock, Document, Delete, Plus, Edit } from '@element-plus/icons-vue'
import { getPodDisruptionBudgets, getPodDisruptionBudgetYAML, updatePodDisruptionBudgetYAML, createPodDisruptionBudgetFromYAML, deletePodDisruptionBudget, getNamespaces, type PodDisruptionBudgetInfo } from '@/api/kubernetes'
import { useKubernetesStore } from '@/stores/kubernetes'
import YamlEditor from '@/components/YamlEditor.vue'

interface PDBInfo {
  name: string
  namespace: string
  minAvailable?: string
  maxUnavailable?: string
  allowedDisruptions?: number
  currentHealthy?: number
  desiredHealthy?: number
  age: string
  createdAt?: string
}

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['edit', 'yaml', 'refresh', 'count-update'])

const loading = ref(false)
const pdbList = ref<PodDisruptionBudgetInfo[]>([])
const namespaces = ref<{ name: string }[]>([])

// 搜索和筛选
const searchName = ref('')
// const filterNamespace = ref('')
const kubernetesStore = useKubernetesStore()
const selectedNamespaces = computed({
  get: () => kubernetesStore.selectedNamespaces,
  set: (val: string[]) => kubernetesStore.setNamespaces(val)
})

// 分页
const currentPage = ref(1)
const pageSize = ref(10)

// YAML 编辑
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedPDB = ref<PodDisruptionBudgetInfo | null>(null)
const saving = ref(false)
const isCreateMode = ref(false)

// YAML对话框标题
const yamlDialogTitle = computed(() => {
  if (isCreateMode.value) {
    return '新增 PodDisruptionBudget'
  }
  return `PodDisruptionBudget YAML - ${selectedPDB.value?.name || ''}`
})

// 默认 PodDisruptionBudget YAML 模板
const getDefaultPDBYAML = () => `apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: example-pdb
  namespace: default
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: example
`

// 计算YAML行数


// 获取健康状态标签类型
const getHealthyTagType = (current: number | undefined, desired: number | undefined) => {
  if (current === undefined || desired === undefined) return 'info'
  if (current < desired) return 'danger'
  if (current === desired) return 'success'
  return 'info'
}

// 过滤后的列表
const filteredPDBs = computed(() => {
  let result = pdbList.value

  if (searchName.value) {
    result = result.filter(p =>
      p.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  if (selectedNamespaces.value.length > 0 && !selectedNamespaces.value.includes('')) {
    result = result.filter(p => selectedNamespaces.value.includes(p.namespace))
  }

  return result
})

// 分页后的列表
const paginatedPDBs = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredPDBs.value.slice(start, end)
})

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!props.clusterId) return
  try {
    const data = await getNamespaces(props.clusterId)
    namespaces.value = data || []
  } catch (error) {
    console.error('Failed to load namespaces:', error)
  }
}

// 加载 PodDisruptionBudget 列表
const loadPDBs = async () => {
  if (!props.clusterId) return

  loading.value = true
  try {
    let nsParam: string | undefined = undefined
    if (selectedNamespaces.value.length === 1 && selectedNamespaces.value[0] !== '') {
      nsParam = selectedNamespaces.value[0]
    } else if (selectedNamespaces.value.length > 1) {
      // If multiple specific namespaces are selected, filter client-side
      // The API currently only supports one namespace or all.
      // For now, we fetch all and filter later.
      nsParam = undefined
    }
    const response = await getPodDisruptionBudgets(props.clusterId, nsParam)
    pdbList.value = response || []
  } catch (error) {
    console.error('Failed to load PDBs:', error)
    pdbList.value = []
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  currentPage.value = 1
}

// 编辑 YAML
const handleEditYAML = async (row: PodDisruptionBudgetInfo) => {
  selectedPDB.value = row
  isCreateMode.value = false

  try {
    const response = await getPodDisruptionBudgetYAML(props.clusterId!, row.namespace, row.name)
    yamlContent.value = response || ''
    yamlDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
  }
}

// 新增 PodDisruptionBudget
const handleCreate = () => {
  isCreateMode.value = true
  selectedPDB.value = null
  yamlContent.value = getDefaultPDBYAML()
  yamlDialogVisible.value = true
}

// 保存 YAML
const handleSaveYAML = async () => {
  if (isCreateMode.value) {
    // 创建模式
    const nameMatch = yamlContent.value.match(/name:\s*(.+)/)
    const nsMatch = yamlContent.value.match(/namespace:\s*(.+)/)
    if (!nameMatch || !nsMatch) {
      ElMessage.error('YAML中缺少name或namespace字段')
      return
    }
    const namespace = nsMatch[1].trim()

    saving.value = true
    try {
      await createPodDisruptionBudgetFromYAML(props.clusterId!, namespace, yamlContent.value)
      ElMessage.success('创建成功')
      yamlDialogVisible.value = false
      await loadPDBs()
      emit('refresh')
    } catch (error: any) {
      ElMessage.error(`创建失败: ${error.response?.data?.message || error.message}`)
    } finally {
      saving.value = false
    }
  } else {
    // 编辑模式
    if (!selectedPDB.value) return

    saving.value = true
    try {
      await updatePodDisruptionBudgetYAML(props.clusterId!, selectedPDB.value.namespace, selectedPDB.value.name, yamlContent.value)

      ElMessage.success('保存成功')
      yamlDialogVisible.value = false
      await loadPDBs()
      emit('refresh')
    } catch (error: any) {
      ElMessage.error(`保存失败: ${error.response?.data?.message || error.message}`)
    } finally {
      saving.value = false
    }
  }
}

// 删除 PodDisruptionBudget
const handleDelete = async (row: PodDisruptionBudgetInfo) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除 PodDisruptionBudget ${row.name} 吗？此操作不可恢复！`,
      '删除 PodDisruptionBudget 确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error'
      }
    )

    await deletePodDisruptionBudget(props.clusterId!, row.namespace, row.name)

    ElMessage.success('删除成功')
    await loadPDBs()
    emit('refresh')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`删除失败: ${error.response?.data?.message || error.message}`)
    }
  }
}



// 监听 clusterId 变化
watch(() => props.clusterId, (newVal) => {
  if (newVal) {
    currentPage.value = 1
    loadNamespaces()
    loadPDBs()
  }
})

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
  loadPDBs()
}

// 监听筛选后的数据变化，更新计数
watch(filteredPDBs, (newData) => {
  emit('count-update', newData.length)
})

onMounted(() => {
  if (props.clusterId) {
    loadNamespaces()
    loadPDBs()
  }
})

// 暴露方法给父组件
defineExpose({
  loadPDBs
})
</script>

<style scoped>
.pdb-list {
  padding: 0;
}

/* 搜索栏 */
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

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__body-wrapper) {
  border-radius: 0;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
  height: 56px !important;
}

.modern-table :deep(.el-table__row td) {
  height: 56px !important;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.resource-value {
  font-size: 13px;
  color: #606266;
}

/* 名称单元格 */
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

/* 表头图标 */
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

.namespace-text {
  font-size: 12px;
  color: #909399;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 4px;
  justify-content: center;
}

.action-btn {
  color: #ffffff;
  padding: 4px;
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

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}

/* YAML 编辑弹窗 */
.yaml-dialog :deep(.el-dialog__header) {
  background: linear-gradient(135deg, #0a466a 0%, #0d5a87 100%);
  color: #ffffff;
  border-radius: 0;
  padding: 20px 24px;
}

.yaml-dialog :deep(.el-dialog__title) {
  color: #ffffff;
  font-size: 16px;
  font-weight: 600;
}

.yaml-dialog :deep(.el-dialog__body) {
  padding: 24px;
  background-color: #1a1a1a;
}



.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 按钮样式 - 使用全局样式 .black-button */
</style>
