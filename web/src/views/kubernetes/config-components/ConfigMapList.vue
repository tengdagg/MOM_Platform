<template>
  <div class="configmap-list">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-bar-left">
        <el-input
          v-model="searchName"
          placeholder="搜索 ConfigMap 名称..."
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
        <el-button type="primary" class="black-button" @click="handleCreateYAML">
          <el-icon style="margin-right: 4px;"><Document /></el-icon>
          YAML创建
        </el-button>

        <el-button type="primary" class="black-button" @click="handleCreateForm">
          <el-icon style="margin-right: 4px;"><Plus /></el-icon>
          表单创建
        </el-button>
      </div>
    </div>

    <!-- ConfigMap 列表 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedConfigMaps"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="名称" prop="name" min-width="200" fixed>
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-blue"><Key /></el-icon>
              名称
            </span>
          </template>
          <template #default="{ row }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <el-icon class="name-icon" :size="18"><Key /></el-icon>
              </div>
              <div class="name-content">
                <div class="name-text">{{ row.name }}</div>
                <div class="namespace-text">{{ row.namespace }}</div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="数据项" prop="dataCount" width="100" align="center">
          <template #default="{ row }">
            <el-tag type="info" size="small">{{ row.dataCount }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="存活时间" prop="age" width="140" />

        <el-table-column label="创建时间" prop="createdAt" width="180">
          <template #default="{ row }">
            {{ row.createdAt || '-' }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="160" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑 YAML" placement="top">
                <el-button link class="action-btn" @click="handleEditYAML(row)">
                  <el-icon :size="18"><Document /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn" @click="handleEditForm(row)">
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
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredConfigMaps.length"
          layout="total, sizes, prev, pager, next"
        />
      </div>
    </div>

    <!-- YAML 弹窗 -->
    <el-dialog v-model="yamlDialogVisible" :title="yamlDialogTitle" width="900px" class="yaml-dialog">
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
          <el-button @click="yamlDialogVisible = false">关闭</el-button>
          <el-button type="primary" @click="handleSaveYAML" :loading="saving" class="black-button">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 表单创建弹窗 -->
    <el-dialog v-model="formDialogVisible" :title="formDialogTitle" width="1200px" class="form-dialog">
      <el-form :model="formData" label-width="100px" class="configmap-form">
        <div class="form-row">
          <el-form-item label="名称" required>
            <el-input v-model="formData.name" placeholder="请输入 ConfigMap 名称" style="width: 100%;" />
          </el-form-item>
          <el-form-item label="命名空间" required>
            <el-select v-model="formData.namespace" placeholder="请选择命名空间" style="width: 100%;">
              <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
            </el-select>
          </el-form-item>
        </div>

        <!-- 标签页 -->
        <el-tabs v-model="activeTab" class="form-tabs">
          <!-- 数据标签页 -->
          <el-tab-pane label="数据" name="data">
            <div class="tab-content">
              <!-- Data 部分 -->
              <div class="data-section">
                <div class="section-header">
                  <span class="section-title">Data</span>
                  <el-button size="small" type="primary" @click="addDataRow">
                    <el-icon><Plus /></el-icon> 添加数据
                  </el-button>
                </div>
                <el-table :data="formData.data" border class="form-table">
                  <el-table-column label="Key" width="200">
                    <template #default="{ row }">
                      <el-input v-model="row.key" placeholder="请输入 Key" />
                    </template>
                  </el-table-column>
                  <el-table-column label="Value">
                    <template #default="{ row }">
                      <el-input v-model="row.value" type="textarea" :rows="3" placeholder="请输入 Value" />
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="80">
                    <template #default="{ $index }">
                      <el-button link type="danger" @click="removeDataRow($index)">
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>

              <!-- BinaryData 部分 -->
              <div class="binarydata-section">
                <div class="section-header">
                  <span class="section-title">BinaryData</span>
                  <el-button size="small" type="primary" @click="addBinaryDataRow">
                    <el-icon><Plus /></el-icon> 添加二进制数据
                  </el-button>
                </div>
                <el-table :data="formData.binaryData" border class="form-table">
                  <el-table-column label="Key" width="200">
                    <template #default="{ row }">
                      <el-input v-model="row.key" placeholder="请输入 Key" />
                    </template>
                  </el-table-column>
                  <el-table-column label="Value">
                    <template #default="{ row }">
                      <el-input v-model="row.value" type="textarea" :rows="3" placeholder="请输入 Value (Base64编码)" />
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="80">
                    <template #default="{ $index }">
                      <el-button link type="danger" @click="removeBinaryDataRow($index)">
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </div>
          </el-tab-pane>

          <!-- 标签/注解标签页 -->
          <el-tab-pane label="标签/注解" name="metadata">
            <div class="tab-content">
              <div class="metadata-section">
                <div class="metadata-header">
                  <span class="metadata-title">标签</span>
                  <el-button size="small" @click="addLabelRow">
                    <el-icon><Plus /></el-icon> 添加
                  </el-button>
                </div>
                <el-table :data="formData.labels" border class="form-table">
                  <el-table-column label="Key" width="200">
                    <template #default="{ row }">
                      <el-input v-model="row.key" placeholder="请输入 Key" />
                    </template>
                  </el-table-column>
                  <el-table-column label="Value">
                    <template #default="{ row }">
                      <el-input v-model="row.value" placeholder="请输入 Value" />
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="80">
                    <template #default="{ $index }">
                      <el-button link type="danger" @click="removeLabelRow($index)">
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>

              <div class="metadata-section">
                <div class="metadata-header">
                  <span class="metadata-title">注解</span>
                  <el-button size="small" @click="addAnnotationRow">
                    <el-icon><Plus /></el-icon> 添加
                  </el-button>
                </div>
                <el-table :data="formData.annotations" border class="form-table">
                  <el-table-column label="Key" width="200">
                    <template #default="{ row }">
                      <el-input v-model="row.key" placeholder="请输入 Key" />
                    </template>
                  </el-table-column>
                  <el-table-column label="Value">
                    <template #default="{ row }">
                      <el-input v-model="row.value" placeholder="请输入 Value" />
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="80">
                    <template #default="{ $index }">
                      <el-button link type="danger" @click="removeAnnotationRow($index)">
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="formDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSaveForm" :loading="saving" class="black-button">{{ isEditMode ? '保存' : '创建' }}</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Key, Document, Delete, Plus, Edit } from '@element-plus/icons-vue'
import { getNamespaces } from '@/api/kubernetes'
import { useKubernetesStore } from '@/stores/kubernetes'
import axios from 'axios'
import * as yaml from 'js-yaml'
import YamlEditor from '@/components/YamlEditor.vue'

interface ConfigMapInfo {
  name: string
  namespace: string
  dataCount: number
  age: string
  createdAt?: string
}

interface KeyValueRow {
  key: string
  value: string
}

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['edit', 'yaml', 'refresh', 'count-update'])

const loading = ref(false)
const configMapList = ref<ConfigMapInfo[]>([])
const namespaces = ref<{ name: string }[]>([])

// 搜索和筛选
const searchName = ref('')
// const filterNamespace = ref('') // 移除本地 filterNamespace
const kubernetesStore = useKubernetesStore()

const selectedNamespaces = computed({
  get: () => kubernetesStore.selectedNamespaces,
  set: (val: string[]) => kubernetesStore.setNamespaces(val)
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
  handleSearch()
  loadConfigMaps()
}

// 分页
const currentPage = ref(1)
const pageSize = ref(10)

// YAML 编辑
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedConfigMap = ref<ConfigMapInfo | null>(null)
const saving = ref(false)
const isCreateMode = ref(false)
const isEditMode = ref(false)

// YAML对话框标题
const yamlDialogTitle = computed(() => {
  if (isCreateMode.value) {
    return '新增 ConfigMap (YAML)'
  }
  return `ConfigMap YAML - ${selectedConfigMap.value?.name || ''}`
})

// 表单对话框标题
const formDialogTitle = computed(() => {
  if (isEditMode.value) {
    return `编辑 ConfigMap - ${formData.value.name}`
  }
  return '新增 ConfigMap'
})

// 表单创建
const formDialogVisible = ref(false)
const activeTab = ref('data')
const formData = ref({
  name: '',
  namespace: '',
  data: [] as KeyValueRow[],
  binaryData: [] as KeyValueRow[],
  labels: [] as KeyValueRow[],
  annotations: [] as KeyValueRow[]
})


// 过滤后的列表
const filteredConfigMaps = computed(() => {
  let result = configMapList.value

  if (searchName.value) {
    result = result.filter(cm =>
      cm.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  if (selectedNamespaces.value.length > 0) {
    if (selectedNamespaces.value.length > 1) {
      result = result.filter(cm => selectedNamespaces.value.includes(cm.namespace))
    }
  }

  return result
})

// 分页后的列表
const paginatedConfigMaps = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredConfigMaps.value.slice(start, end)
})

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!props.clusterId) return
  try {
    const data = await getNamespaces(props.clusterId)
    namespaces.value = data || []
  } catch (error) {
  }
}

// 加载 ConfigMap 列表
const loadConfigMaps = async () => {
  if (!props.clusterId) return

  loading.value = true
  try {
    const token = localStorage.getItem('token')
    
    const params: any = { clusterId: props.clusterId }
    // 单选传递 namespace，多选不传（获取所有）
    if (selectedNamespaces.value.length === 1) {
      params.namespace = selectedNamespaces.value[0]
    }
    
    const response = await axios.get(`/api/v1/plugins/kubernetes/resources/configmaps`, {
      params,
      headers: { Authorization: `Bearer ${token}` }
    })
    configMapList.value = response.data.data || []
  } catch (error) {
    configMapList.value = []
    // 不显示错误提示，避免频繁弹出
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  currentPage.value = 1
}

// YAML 创建
const handleCreateYAML = () => {
  isCreateMode.value = true
  selectedConfigMap.value = null
  // 默认 ConfigMap YAML 模板
  yamlContent.value = `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-configmap
  namespace: default
data:
  key1: value1
  key2: value2
`
  yamlDialogVisible.value = true
}

// 表单创建
const handleCreateForm = () => {
  isEditMode.value = false
  formData.value = {
    name: '',
    namespace: namespaces.value[0]?.name || '',
    data: [],
    binaryData: [],
    labels: [],
    annotations: []
  }
  formDialogVisible.value = true
}

// 编辑表单
const handleEditForm = async (row: ConfigMapInfo) => {
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/configmaps/${row.namespace}/${row.name}/yaml`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )


    // 获取ConfigMap对象，可能是items或yaml字段
    let configMap: any = response.data?.data?.items || response.data?.data?.yaml

    // 如果是yaml字符串，需要解析
    if (typeof configMap === 'string') {
      configMap = yaml.load(configMap)
    }


    if (!configMap || !configMap.metadata) {
      ElMessage.error('获取ConfigMap数据失败')
      return
    }

    // 填充表单数据
    formData.value = {
      name: configMap.metadata?.name || '',
      namespace: configMap.metadata?.namespace || '',
      data: configMap.data ? Object.entries(configMap.data).map(([key, value]) => ({ key, value: String(value) })) : [],
      binaryData: configMap.binaryData ? Object.entries(configMap.binaryData).map(([key, value]) => ({ key, value: String(value) })) : [],
      labels: configMap.metadata?.labels ? Object.entries(configMap.metadata.labels).map(([key, value]) => ({ key, value })) : [],
      annotations: configMap.metadata?.annotations ? Object.entries(configMap.metadata.annotations).map(([key, value]) => ({ key, value })) : []
    }


    isEditMode.value = true
    formDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(`获取详情失败: ${error.response?.data?.message || error.message}`)
  }
}

// 编辑 YAML
const handleEditYAML = async (row: ConfigMapInfo) => {
  selectedConfigMap.value = row
  isCreateMode.value = false

  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/configmaps/${row.namespace}/${row.name}/yaml`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    // 后端返回的是 {data: {items: ConfigMap对象}}
    const jsonData = response.data.data?.items
    if (jsonData) {
      yamlContent.value = yaml.dump(jsonData, {
        indent: 2,
        lineWidth: -1,
        noRefs: true,
        sortKeys: false
      })
      yamlDialogVisible.value = true
    }
  } catch (error: any) {
    ElMessage.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
  }
}

// 保存 YAML
const handleSaveYAML = async () => {
  saving.value = true
  try {
    const token = localStorage.getItem('token')

    // 从 YAML 中解析对象
    const yamlObj: any = yaml.load(yamlContent.value)
    if (!yamlObj || !yamlObj.metadata || !yamlObj.metadata.name) {
      ElMessage.error('YAML 中缺少必要的 metadata.name 字段')
      return
    }
    const name = yamlObj.metadata.name
    const namespace = yamlObj.metadata.namespace || 'default'

    if (isCreateMode.value) {
      // 创建模式 - 发送 JSON 对象
      await axios.post(
        `/api/v1/plugins/kubernetes/resources/configmaps/${namespace}/yaml`,
        yamlObj,
        {
          params: { clusterId: props.clusterId },
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      ElMessage.success('创建成功')
    } else {
      // 编辑模式 - 发送 JSON 对象
      if (!selectedConfigMap.value) return
      await axios.put(
        `/api/v1/plugins/kubernetes/resources/configmaps/${selectedConfigMap.value.namespace}/${selectedConfigMap.value.name}/yaml`,
        yamlObj,
        {
          params: { clusterId: props.clusterId },
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      ElMessage.success('保存成功')
    }

    yamlDialogVisible.value = false
    await loadConfigMaps()
    emit('refresh')
  } catch (error: any) {
    ElMessage.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    saving.value = false
  }
}

// 保存表单
const handleSaveForm = async () => {
  if (!formData.value.name) {
    ElMessage.error('请输入名称')
    return
  }
  if (!formData.value.namespace) {
    ElMessage.error('请选择命名空间')
    return
  }

  saving.value = true
  try {
    const token = localStorage.getItem('token')


    // 构建 ConfigMap 对象
    const configMapObj: any = {
      apiVersion: 'v1',
      kind: 'ConfigMap',
      metadata: {
        name: formData.value.name,
        namespace: formData.value.namespace
      }
    }

    // 添加 Data
    if (formData.value.data.length > 0) {
      configMapObj.data = {}
      formData.value.data.forEach(row => {
        if (row.key) {
          configMapObj.data[row.key] = row.value
        }
      })
    }

    // 添加 BinaryData
    if (formData.value.binaryData.length > 0) {
      configMapObj.binaryData = {}
      formData.value.binaryData.forEach(row => {
        if (row.key) {
          configMapObj.binaryData[row.key] = row.value
        }
      })
    }

    // 添加标签
    if (formData.value.labels.length > 0) {
      configMapObj.metadata.labels = {}
      formData.value.labels.forEach(row => {
        if (row.key) {
          configMapObj.metadata.labels[row.key] = row.value
        }
      })
    }

    // 添加注解
    if (formData.value.annotations.length > 0) {
      configMapObj.metadata.annotations = {}
      formData.value.annotations.forEach(row => {
        if (row.key) {
          configMapObj.metadata.annotations[row.key] = row.value
        }
      })
    }

    if (isEditMode.value) {
      // 编辑模式：使用 PUT 请求
      await axios.put(
        `/api/v1/plugins/kubernetes/resources/configmaps/${formData.value.namespace}/${formData.value.name}/yaml`,
        configMapObj,
        {
          params: { clusterId: props.clusterId },
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      ElMessage.success('更新成功')
    } else {
      // 创建模式：使用 POST 请求
      await axios.post(
        `/api/v1/plugins/kubernetes/resources/configmaps/${formData.value.namespace}/yaml`,
        configMapObj,
        {
          params: { clusterId: props.clusterId },
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      ElMessage.success('创建成功')
    }

    formDialogVisible.value = false
    await loadConfigMaps()
    emit('refresh')
  } catch (error: any) {
    ElMessage.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    saving.value = false
  }
}

// 删除 ConfigMap
const handleDelete = async (row: ConfigMapInfo) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除 ConfigMap ${row.name} 吗？此操作不可恢复！`,
      '删除 ConfigMap 确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error'
      }
    )

    const token = localStorage.getItem('token')
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/configmaps/${row.namespace}/${row.name}`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('删除成功')
    await loadConfigMaps()
    emit('refresh')
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`删除失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 数据行操作
const addDataRow = () => {
  formData.value.data.push({ key: '', value: '' })
}

const removeDataRow = (index: number) => {
  formData.value.data.splice(index, 1)
}

// BinaryData 行操作
const addBinaryDataRow = () => {
  formData.value.binaryData.push({ key: '', value: '' })
}

const removeBinaryDataRow = (index: number) => {
  formData.value.binaryData.splice(index, 1)
}

// 标签行操作
const addLabelRow = () => {
  formData.value.labels.push({ key: '', value: '' })
}

const removeLabelRow = (index: number) => {
  formData.value.labels.splice(index, 1)
}

// 注解行操作
const addAnnotationRow = () => {
  formData.value.annotations.push({ key: '', value: '' })
}

const removeAnnotationRow = (index: number) => {
  formData.value.annotations.splice(index, 1)
}

// YAML编辑器输入处理
const handleYamlInput = () => {
  // 可以添加输入验证
}


// 监听 clusterId 变化
watch(() => props.clusterId, (newVal) => {
  if (newVal) {
    currentPage.value = 1
    loadNamespaces()
    loadConfigMaps()
  }
})

// 监听筛选后的数据变化，更新计数
watch(filteredConfigMaps, (newData) => {
  emit('count-update', newData.length)
})

onMounted(() => {
  if (props.clusterId) {
    loadNamespaces()
    loadConfigMaps()
  }
})

// 暴露方法给父组件
defineExpose({
  loadConfigMaps
})
</script>

<style scoped>
.configmap-list {
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

.yaml-editor-wrapper {
  display: flex;
  border: none;
  border-radius: 0;
  overflow: hidden;
  background-color: #000000;
}


/* 表单弹窗 */
.form-dialog :deep(.el-dialog__body) {
  padding: 20px 24px;
  max-height: 600px;
  overflow-y: auto;
}

.configmap-form {
  max-width: 100%;
}

.form-row {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.form-row .el-form-item {
  flex: 1;
  margin-bottom: 0;
}

.form-tabs {
  margin-top: 16px;
}

.tab-content {
  padding: 16px 0;
}

.table-actions-wrapper {
  margin-bottom: 12px;
}

.data-section {
  margin-bottom: 32px;
  padding: 16px;
  background-color: #f5f7fa;
  border-radius: 0;
  border: 1px solid #e4e7ed;
}

.binarydata-section {
  margin-bottom: 32px;
  padding: 16px;
  background-color: #f5f7fa;
  border-radius: 0;
  border: 1px solid #e4e7ed;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.data-title {
  font-weight: 600;
  color: #333;
}

.metadata-section {
  margin-bottom: 24px;
}

.metadata-section:last-child {
  margin-bottom: 0;
}

.metadata-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.metadata-title {
  font-weight: 600;
  color: #333;
}

.table-header-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.table-title {
  font-weight: 600;
  color: #ffffff;
}

.form-table {
  width: 100%;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

</style>
