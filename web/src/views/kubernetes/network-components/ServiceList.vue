```html
<template>
  <div class="service-list">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-bar-left">
        <el-input
          v-model="searchName"
          placeholder="搜索服务名称..."
          clearable
          class="search-input"
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select v-model="filterType" placeholder="服务类型" clearable @change="handleSearch" class="filter-select">
          <el-option label="全部" value="" />
          <el-option label="ClusterIP" value="ClusterIP" />
          <el-option label="NodePort" value="NodePort" />
          <el-option label="LoadBalancer" value="LoadBalancer" />
        </el-select>

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
        <el-button class="black-button" @click="handleCreate">创建服务</el-button>
        <el-button class="black-button" @click="handleCreateYAML">
          <el-icon><Document /></el-icon> YAML创建
        </el-button>
      </div>
    </div>

    <!-- 服务列表 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedServices"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="名称" prop="name" min-width="180" fixed>
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-blue"><Connection /></el-icon>
              名称
            </span>
          </template>
          <template #default="{ row }">
            <div class="name-cell" @click="handleShowDetail(row)" style="cursor: pointer;">
              <el-icon class="name-icon"><Connection /></el-icon>
              <div>
                <div class="name-text">{{ row.name }}</div>
                <div class="namespace-text">{{ row.namespace }}</div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="类型" prop="type" width="130">
          <template #default="{ row }">
            <el-tag :type="getTypeTagType(row.type)" size="small">{{ row.type }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Cluster IP" prop="clusterIP" width="140" />

        <el-table-column label="外部 IP" prop="externalIP" width="140">
          <template #default="{ row }">
            {{ row.externalIP || '-' }}
          </template>
        </el-table-column>

        <el-table-column label="端口" min-width="200">
          <template #default="{ row }">
            <div v-for="port in row.ports" :key="port.port" class="port-item">
              {{ port.protocol }}: {{ port.port }}
              <span v-if="port.targetPort">→ {{ port.targetPort }}</span>
              <span v-if="port.nodePort"> ({{ port.nodePort }})</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="端点" prop="endpoints" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.endpoints > 0" type="success" size="small">{{ row.endpoints }}</el-tag>
            <el-tag v-else type="info" size="small">0</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="存活时间" prop="age" width="120" />

        <el-table-column label="操作" width="160" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑 YAML" placement="top">
                <el-button link class="action-btn" @click="handleEditYAML(row)">
                  <el-icon :size="18"><Document /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn" @click="handleEdit(row)">
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

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="filteredServices.length"
          layout="total, sizes, prev, pager, next"
        />
      </div>
    </div>

    <!-- YAML 弹窗 -->
    <el-dialog v-model="yamlDialogVisible" :title="`Service YAML - ${selectedService?.name}`" width="900px" :lock-scroll="false" class="yaml-dialog">
      <div class="yaml-editor-wrapper">
        <YamlEditor
          v-if="yamlDialogVisible"
          v-model="yamlContent"
          :theme="'vs-dark'"
          language="yaml"
        />
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="yamlDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSaveYAML" :loading="saving">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- YAML 创建弹窗 -->
    <el-dialog v-model="createYamlDialogVisible" title="YAML 创建 Service" width="900px" :lock-scroll="false" class="yaml-dialog">
      <div class="yaml-editor-wrapper">
        <YamlEditor
          v-if="createYamlDialogVisible"
          v-model="createYamlContent"
          :theme="'vs-dark'"
          language="yaml"
        />
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="createYamlDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSaveCreateYAML" :loading="creating">创建</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 编辑对话框 -->
    <ServiceEditDialog
      ref="editDialogRef"
      :clusterId="clusterId"
      @success="handleEditSuccess"
    />

    <!-- Service详情对话框 -->
    <ServiceDetailDialog
      ref="detailDialogRef"
      :clusterId="clusterId"
      @terminal="handleTerminal"
      @logs="handleLogs"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Connection, Document, Edit, Delete } from '@element-plus/icons-vue'
import { load, dump } from 'js-yaml'
import { getServices, getServiceYAML, updateServiceYAML, createServiceYAML, deleteService, getNamespaces, type ServiceInfo } from '@/api/kubernetes'
import { useKubernetesStore } from '@/stores/kubernetes'
import ServiceEditDialog from './ServiceEditDialog.vue'
import ServiceDetailDialog from './ServiceDetailDialog.vue'
import YamlEditor from '@/components/YamlEditor.vue'

const props = defineProps<{
  clusterId?: number
  namespace?: string
}>()

const emit = defineEmits(['edit', 'yaml', 'refresh', 'count-update', 'terminal', 'logs'])

const loading = ref(false)
const saving = ref(false)
const serviceList = ref<ServiceInfo[]>([])
const namespaces = ref<any[]>([])
const searchName = ref('')
const filterType = ref('')
// const filterNamespace = ref('') // 移除本地 filterNamespace，使用 store
const kubernetesStore = useKubernetesStore()

const selectedNamespaces = computed({
  get: () => kubernetesStore.selectedNamespaces,
  set: (val: string[]) => kubernetesStore.setNamespaces(val)
})

const handleNamespaceChange = (val: string[]) => {
  // 处理"所有命名空间"唯一性逻辑
  if (val.includes('')) {
    // 如果最新选中的是所有(也就是最后一个是'')，或者之前有其他现在加上了'' => 清空其他只留''
    if (val[val.length - 1] === '') {
       kubernetesStore.setNamespaces([''])
    } else {
       // 如果之前是''，现在选了其他 => 去掉''
       const newVal = val.filter(v => v !== '')
       kubernetesStore.setNamespaces(newVal)
    }
  } else {
    // 没选所有
    if (val.length === 0) {
      // 全不选 => 默认为所有
       kubernetesStore.setNamespaces([''])
    } else {
       kubernetesStore.setNamespaces(val)
    }
  }
  handleSearch()
}
const currentPage = ref(1)
const pageSize = ref(10)
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedService = ref<ServiceInfo | null>(null)
const originalJsonData = ref<any>(null) // 保存原始 JSON 数据
const editDialogRef = ref<any>(null)
const detailDialogRef = ref<any>(null)

// YAML 创建相关
const createYamlDialogVisible = ref(false)
const creating = ref(false)
const createYamlContent = ref('')

const filteredServices = computed(() => {
  let result = serviceList.value
  if (searchName.value) {
    result = result.filter(s => s.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  if (filterType.value) {
    result = result.filter(s => s.type === filterType.value)
  }
  // 与 SecretList 保持一致的过滤逻辑
  if (selectedNamespaces.value.length > 0 && !selectedNamespaces.value.includes('')) {
    result = result.filter(s => selectedNamespaces.value.includes(s.namespace))
  }
  return result
})

const paginatedServices = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredServices.value.slice(start, end)
})

const loadServices = async (showSuccess = false) => {
  if (!props.clusterId) return
  loading.value = true
  try {
    // 与 SecretList 保持一致：总是获取所有数据，前端过滤
    const data = await getServices(props.clusterId)
    serviceList.value = data || []
    if (showSuccess) {
      ElMessage.success('刷新成功')
    }
  } catch (error) {
    ElMessage.error('获取服务列表失败')
  } finally {
    loading.value = false
  }
}

const loadNamespaces = async () => {
  if (!props.clusterId) return
  try {
    const data = await getNamespaces(props.clusterId)
    namespaces.value = data || []
  } catch (error) {
  }
}

const handleSearch = () => {
  currentPage.value = 1
}

const getTypeTagType = (type: string) => {
  const map: Record<string, string> = {
    ClusterIP: 'success',
    NodePort: 'warning',
    LoadBalancer: 'danger'
  }
  return map[type] || 'info'
}

const handleCreate = () => {
  editDialogRef.value?.openCreate(namespaces.value)
}

const handleCreateYAML = () => {
  const defaultNamespace = props.namespace || 'default'
  // 设置默认 YAML 模板
  createYamlContent.value = `apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: ${defaultNamespace}
spec:
  type: ClusterIP
  selector:
    app: my-app
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
`
  createYamlDialogVisible.value = true
}

const handleEdit = (service: ServiceInfo) => {
  editDialogRef.value?.openEdit(service, namespaces.value)
}

const handleEditSuccess = () => {
  emit('refresh')
  loadServices()
}

const handleEditYAML = async (service: ServiceInfo) => {
  if (!props.clusterId) return
  selectedService.value = service
  try {
    const response = await getServiceYAML(props.clusterId, service.namespace, service.name)
    // 保存原始 JSON 数据
    originalJsonData.value = response.items || response
    // 转换为 YAML 格式
    const yaml = dump(originalJsonData.value, { indent: 2, lineWidth: -1 })
    yamlContent.value = yaml
    yamlDialogVisible.value = true
  } catch (error) {
    ElMessage.error('获取 YAML 失败')
  }
}


// 使用 js-yaml 库解析 YAML
const yamlToJson = (yaml: string): any => {
  try {
    return load(yaml)
  } catch (error) {
    throw error
  }
}

const handleSaveYAML = async () => {
  if (!props.clusterId || !selectedService.value) return

  saving.value = true
  try {
    // 尝试将 YAML 转回 JSON
    let jsonData
    try {
      jsonData = yamlToJson(yamlContent.value)
      // 确保基本的元数据存在
      if (!jsonData.metadata) {
        jsonData.metadata = {}
      }
      if (!jsonData.metadata.name && selectedService.value) {
        jsonData.metadata.name = selectedService.value.name
      }
      if (!jsonData.metadata.namespace && selectedService.value) {
        jsonData.metadata.namespace = selectedService.value.namespace
      }
      if (!jsonData.apiVersion) {
        jsonData.apiVersion = 'v1'
      }
      if (!jsonData.kind) {
        jsonData.kind = 'Service'
      }
    } catch (e) {
      ElMessage.error('YAML 格式错误，请检查缩进和语法')
      saving.value = false
      return
    }

    await updateServiceYAML(
      props.clusterId,
      selectedService.value.namespace,
      selectedService.value.name,
      jsonData
    )
    ElMessage.success('保存成功')
    yamlDialogVisible.value = false
    emit('refresh')
    await loadServices()
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const handleYamlInput = () => {
  // 处理输入
}

const handleDelete = async (service: ServiceInfo) => {
  if (!props.clusterId) return
  try {
    await ElMessageBox.confirm(`确定要删除服务 ${service.name} 吗？`, '删除确认', { type: 'error' })
    await deleteService(props.clusterId, service.namespace, service.name)
    ElMessage.success('删除成功')
    emit('refresh')
    await loadServices()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

const handleSaveCreateYAML = async () => {
  if (!props.clusterId) return

  creating.value = true
  try {
    const jsonData = yamlToJson(createYamlContent.value)
    // 确保基本的元数据存在
    if (!jsonData.apiVersion) {
      jsonData.apiVersion = 'v1'
    }
    if (!jsonData.kind) {
      jsonData.kind = 'Service'
    }
    if (!jsonData.metadata) {
      jsonData.metadata = {}
    }

    // 从 YAML 中提取命名空间
    const namespace = jsonData.metadata.namespace || props.namespace || 'default'
    jsonData.metadata.namespace = namespace

    await createServiceYAML(
      props.clusterId,
      namespace,
      jsonData
    )
    ElMessage.success('创建成功')
    createYamlDialogVisible.value = false
    emit('refresh')
    await loadServices()
  } catch (error) {
    ElMessage.error('创建失败')
  } finally {
    creating.value = false
  }
}

const handleCreateYamlInput = () => {
  // 处理输入
}

watch(() => props.clusterId, () => {
  loadServices()
  loadNamespaces()
})



// 监听筛选后的数据变化，更新计数
watch(filteredServices, (newData) => {
  emit('count-update', newData.length)
})

onMounted(() => {
  loadServices()
  loadNamespaces()
})

const handleShowDetail = (service: ServiceInfo) => {
  detailDialogRef.value?.open(service.namespace, service.name)
}

const handleTerminal = (data: { namespace: string; name: string }) => {
  emit('terminal', data)
}

const handleLogs = (data: { namespace: string; name: string }) => {
  emit('logs', data)
}

// 暴露方法给父组件
defineExpose({
  loadData: () => loadServices(true)
})
</script>

<style scoped>
.service-list {
  width: 100%;
}

/* 黑色按钮样式 */


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
  width: 180px;
}

.search-icon {
  color: #909399;
}

.table-wrapper {
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.name-cell:hover {
  opacity: 0.8;
}

.name-icon {
  width: 36px;
  height: 36px;
  background: #0a466a;
  border-radius: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  font-size: 18px;
  flex-shrink: 0;
  border: none;
}

.name-text {
  font-weight: 500;
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

.port-item {
  font-size: 12px;
  color: #606266;
  line-height: 1.5;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.action-btn {
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

/* YAML 编辑弹窗 */
.yaml-editor-wrapper {
  display: flex;
  border: none;
  border-radius: 0;
  overflow: hidden;
  background-color: #000000;
}

.yaml-line-numbers {
  background-color: #0d0d0d;
  color: #666;
  padding: 16px 8px;
  text-align: right;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  user-select: none;
  overflow: hidden;
  min-width: 40px;
  border-right: 1px solid #333;
}

.line-number {
  height: 20.8px;
  line-height: 1.6;
}

.yaml-textarea {
  flex: 1;
  background-color: #000000;
  color: #ffffff;
  border: none;
  outline: none;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  resize: vertical;
  min-height: 400px;
}

.yaml-textarea::placeholder {
  color: #555;
}

.yaml-textarea:focus {
  outline: none;
}

.yaml-dialog :deep(.el-dialog__body) {
  padding: 0;
  background-color: #1a1a1a;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
