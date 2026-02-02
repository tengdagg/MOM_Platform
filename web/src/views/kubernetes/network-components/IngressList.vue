<template>
  <div class="ingress-list">
    <div class="search-bar">
      <div class="search-bar-left">
        <el-input v-model="searchName" placeholder="搜索 Ingress 名称..." clearable class="search-input" @input="handleSearch">
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
        <el-button class="black-button" @click="handleCreate">创建 Ingress</el-button>
        <el-button class="black-button" @click="handleCreateYAML">
          <el-icon><Document /></el-icon> YAML创建
        </el-button>
      </div>
    </div>

    <div class="table-wrapper">
      <el-table :data="filteredIngresses" v-loading="loading" class="modern-table" size="default" :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }">
        <el-table-column label="名称" prop="name" min-width="180" fixed>
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-blue"><Link /></el-icon>
              名称
            </span>
          </template>
          <template #default="{ row }">
            <div class="name-cell">
              <el-icon class="name-icon"><Link /></el-icon>
              <div class="name-text">{{ row.name }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="命名空间" prop="namespace" width="140" />
        <el-table-column label="主机名" min-width="200">
          <template #default="{ row }">
            <div v-for="host in row.hosts" :key="host" class="host-item">
              <el-icon class="host-icon"><Monitor /></el-icon>
              <span class="host-text">{{ host }}</span>
            </div>
            <div v-if="!row.hosts.length" class="empty-text">-</div>
          </template>
        </el-table-column>
        <el-table-column label="路径" min-width="250">
          <template #default="{ row }">
            <div v-for="(path, index) in row.paths" :key="`${path.path}-${index}`" class="path-item">
              <div class="path-path">{{ path.path || '/' }}</div>
              <div class="path-service">
                <el-icon class="service-icon"><Connection /></el-icon>
                <span>{{ path.service }}</span>
                <span class="path-port">:{{ path.port }}</span>
              </div>
            </div>
            <div v-if="!row.paths.length" class="empty-text">-</div>
          </template>
        </el-table-column>
        <el-table-column label="Ingress Class" prop="ingressClass" width="150">
          <template #default="{ row }">
            {{ row.ingressClass || '-' }}
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
    </div>

    <el-dialog v-model="yamlDialogVisible" :title="`Ingress YAML - ${selectedIngress?.name}`" width="900px" :lock-scroll="false" class="yaml-dialog">
      <div class="yaml-editor-wrapper">
        <div class="yaml-line-numbers">
          <div v-for="line in yamlLineCount" :key="line" class="line-number">{{ line }}</div>
        </div>
        <textarea
          v-model="yamlContent"
          class="yaml-textarea"
          spellcheck="false"
          @input="handleYamlInput"
          @scroll="handleYamlScroll"
          ref="yamlTextarea"
        ></textarea>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="yamlDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSaveYAML" :loading="saving">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- YAML 创建弹窗 -->
    <el-dialog v-model="createYamlDialogVisible" title="YAML 创建 Ingress" width="900px" :lock-scroll="false" class="yaml-dialog">
      <div class="yaml-editor-wrapper">
        <div class="yaml-line-numbers">
          <div v-for="line in createYamlLineCount" :key="line" class="line-number">{{ line }}</div>
        </div>
        <textarea
          v-model="createYamlContent"
          class="yaml-textarea"
          spellcheck="false"
          @input="handleCreateYamlInput"
          @scroll="handleCreateYamlScroll"
          ref="createYamlTextarea"
        ></textarea>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="createYamlDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSaveCreateYAML" :loading="creating">创建</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 编辑对话框 -->
    <IngressEditDialog
      ref="editDialogRef"
      :clusterId="clusterId"
      @success="handleEditSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Link, Document, Edit, Delete, Monitor, Connection } from '@element-plus/icons-vue'
import { load, dump } from 'js-yaml'
import { getIngresses, getIngressYAML, updateIngressYAML, createIngressYAML, createIngress, deleteIngress, getNamespaces, type IngressInfo } from '@/api/kubernetes'
import { useKubernetesStore } from '@/stores/kubernetes'
import IngressEditDialog from './IngressEditDialog.vue'

const props = defineProps<{
  clusterId?: number
  namespace?: string
}>()

const emit = defineEmits(['edit', 'yaml', 'refresh', 'count-update'])

const loading = ref(false)
const saving = ref(false)
const ingressList = ref<IngressInfo[]>([])
const namespaces = ref<any[]>([])
const searchName = ref('')
// const filterNamespace = ref('')
const kubernetesStore = useKubernetesStore()
const selectedNamespaces = computed({
  get: () => kubernetesStore.selectedNamespaces,
  set: (val: string[]) => kubernetesStore.setNamespaces(val)
})
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedIngress = ref<IngressInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const originalJsonData = ref<any>(null) // 保存原始 JSON 数据
const editDialogRef = ref<any>(null)

// YAML 创建相关
const createYamlDialogVisible = ref(false)
const creating = ref(false)
const createYamlContent = ref('')
const createYamlTextarea = ref<HTMLTextAreaElement | null>(null)

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

const createYamlLineCount = computed(() => {
  if (!createYamlContent.value) return 1
  return createYamlContent.value.split('\n').length
})

const filteredIngresses = computed(() => {
  let result = ingressList.value
  if (searchName.value) {
    result = result.filter(i => i.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  if (selectedNamespaces.value.length > 0) {
    if (selectedNamespaces.value.length > 1) {
      result = result.filter(i => selectedNamespaces.value.includes(i.namespace))
    }
  }
  return result
})

const loadIngresses = async (showSuccess = false) => {
  if (!props.clusterId) return
  loading.value = true
  try {
    let nsParam = undefined
    if (selectedNamespaces.value.length === 1) {
      nsParam = selectedNamespaces.value[0]
    }
    const data = await getIngresses(props.clusterId, nsParam)
    ingressList.value = data || []
    if (showSuccess) {
      ElMessage.success('刷新成功')
    }
  } catch (error) {
    ElMessage.error('获取 Ingress 列表失败')
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
  // 本地过滤
}

const handleCreate = () => {
  editDialogRef.value?.openCreate(namespaces.value)
}

const handleCreateYAML = () => {
  const defaultNamespace = props.namespace || 'default'
  // 设置默认 YAML 模板
  createYamlContent.value = `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
  namespace: ${defaultNamespace}
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
    - host: example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: my-service
                port:
                  number: 80
`
  createYamlDialogVisible.value = true
}

const handleEdit = (ingress: IngressInfo) => {
  editDialogRef.value?.openEdit(ingress, namespaces.value)
}

const handleEditSuccess = () => {
  emit('refresh')
  loadIngresses()
}

const handleEditYAML = async (ingress: IngressInfo) => {
  if (!props.clusterId) return
  selectedIngress.value = ingress
  try {
    const response = await getIngressYAML(props.clusterId, ingress.namespace, ingress.name)
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
  if (!props.clusterId || !selectedIngress.value) return

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
      if (!jsonData.metadata.name && selectedIngress.value) {
        jsonData.metadata.name = selectedIngress.value.name
      }
      if (!jsonData.metadata.namespace && selectedIngress.value) {
        jsonData.metadata.namespace = selectedIngress.value.namespace
      }
      if (!jsonData.apiVersion) {
        jsonData.apiVersion = 'networking.k8s.io/v1'
      }
      if (!jsonData.kind) {
        jsonData.kind = 'Ingress'
      }
    } catch (e) {
      ElMessage.error('YAML 格式错误，请检查缩进和语法')
      saving.value = false
      return
    }

    await updateIngressYAML(
      props.clusterId,
      selectedIngress.value.namespace,
      selectedIngress.value.name,
      jsonData
    )
    ElMessage.success('保存成功')
    yamlDialogVisible.value = false
    emit('refresh')
    await loadIngresses()
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

const handleYamlInput = () => {
  // 处理输入
}

const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

const handleDelete = async (ingress: IngressInfo) => {
  if (!props.clusterId) return
  try {
    await ElMessageBox.confirm(`确定要删除 Ingress ${ingress.name} 吗？`, '删除确认', { type: 'error' })
    await deleteIngress(props.clusterId, ingress.namespace, ingress.name)
    ElMessage.success('删除成功')
    emit('refresh')
    await loadIngresses()
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
      jsonData.apiVersion = 'networking.k8s.io/v1'
    }
    if (!jsonData.kind) {
      jsonData.kind = 'Ingress'
    }
    if (!jsonData.metadata) {
      jsonData.metadata = {}
    }

    // 从 YAML 中提取命名空间和名称
    const namespace = jsonData.metadata.namespace || props.namespace || 'default'
    const name = jsonData.metadata.name

    if (!name) {
      ElMessage.error('YAML 中缺少 metadata.name 字段')
      return
    }

    // 从 spec 中提取 rules
    const spec = jsonData.spec || {}
    const rules = (spec.rules || []).map((rule: any) => ({
      host: rule.host,
      paths: (rule.http?.paths || []).map((p: any) => ({
        path: p.path,
        pathType: p.pathType || 'Prefix',
        service: p.backend?.service?.name,
        port: p.backend?.service?.port?.number
      }))
    }))

    // 提取 TLS 配置
    const tls = (spec.tls || []).map((t: any) => ({
      hosts: t.hosts || [],
      secretName: t.secretName
    }))

    // 提取 ingressClassName
    const ingressClassName = spec.ingressClassName

    // 构建创建请求数据
    const createData = {
      name,
      namespace,
      ingressClassName,
      rules,
      tls
    }

    await createIngress(props.clusterId, namespace, createData)
    ElMessage.success('创建成功')
    createYamlDialogVisible.value = false
    emit('refresh')
    await loadIngresses()
  } catch (error) {
    ElMessage.error('创建失败: ' + (error as any).message)
  } finally {
    creating.value = false
  }
}

const handleCreateYamlInput = () => {
  // 处理输入
}

const handleCreateYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.create-yaml .yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

watch(() => props.clusterId, () => {
  loadIngresses()
  loadNamespaces()
})

// watch local namespace removed

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
  loadIngresses()
}

// 监听筛选后的数据变化，更新计数
watch(filteredIngresses, (newData) => {
  emit('count-update', newData.length)
})

onMounted(() => {
  loadIngresses()
  loadNamespaces()
})

// 暴露方法给父组件
defineExpose({
  loadData: () => loadIngresses(true)
})
</script>

<style scoped>
.ingress-list {
  width: 100%;
}

/* 黑色按钮样式 */
.black-button {
  background-color: #0a466a !important;
  color: #ffffff !important;
  border-color: #0a466a !important;
  border-radius: 0;
  font-weight: 500;
}

.black-button:hover {
  background-color: #0f69a6 !important;
  border-color: #0f69a6 !important;
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

.host-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.host-item:last-child {
  margin-bottom: 0;
}

.host-icon {
  color: #ffffff;
  font-size: 16px;
  flex-shrink: 0;
}

.host-text {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  word-break: break-all;
}

.path-item {
  padding: 10px 12px;
  margin-bottom: 8px;
  background-color: #fef9e7;
  border: none;
  border-radius: 0;
}

.path-item:last-child {
  margin-bottom: 0;
}

.path-path {
  font-size: 14px;
  font-weight: 700;
  color: #303133;
  margin-bottom: 6px;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
}

.path-service {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #606266;
}

.service-icon {
  color: #ffffff;
  font-size: 14px;
}

.path-port {
  color: #909399;
  font-size: 12px;
}

.empty-text {
  color: #909399;
  font-size: 14px;
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
