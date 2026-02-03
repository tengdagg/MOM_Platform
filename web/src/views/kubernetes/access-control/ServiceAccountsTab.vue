<template>
  <div class="service-accounts-tab">
    <!-- 表格 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedList"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column label="名称" min-width="200" fixed="left">
          <template #default="{ row }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <el-icon class="name-icon"><User /></el-icon>
              </div>
              <span class="name-text">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="命名空间" prop="namespace" width="180">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.namespace }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="存活时间" prop="age" width="140" />

        <el-table-column label="操作" width="100" fixed="right" align="center">
          <template #default="{ row }">
            <el-button link @click="handleEdit(row)" class="action-btn">
              <el-icon :size="18"><Edit /></el-icon>
            </el-button>
            <el-button link @click="handleDelete(row)" class="action-btn danger">
              <el-icon :size="18"><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
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

    <!-- 标签弹窗 -->
    <el-dialog
      v-model="labelDialogVisible"
      title="标签"
      width="600px"
    >
      <el-table :data="labelList" max-height="400">
        <el-table-column prop="key" label="Key" min-width="150" />
        <el-table-column prop="value" label="Value" min-width="150" />
      </el-table>
    </el-dialog>

    <!-- YAML 编辑弹窗 -->
    <el-dialog
      v-model="yamlDialogVisible"
      :title="yamlDialogTitle"
      width="900px"
      :close-on-click-modal="false"
      class="yaml-dialog"
    >
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
          <el-button @click="yamlDialogVisible = false">关闭</el-button>
          <el-button type="primary" @click="handleSaveYaml" :loading="yamlSaving" class="black-button">保存</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, User, PriceTag, Edit, Delete, Plus } from '@element-plus/icons-vue'
import { getServiceAccounts, type ServiceAccountInfo } from '@/api/kubernetes'
import { useKubernetesStore } from '@/stores/kubernetes'
import axios from 'axios'
import * as yaml from 'js-yaml'

interface Props {
  clusterId: number
  namespace?: string
  searchName?: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'count-update': [count: number]
}>()
const loading = ref(false)
const serviceAccounts = ref<ServiceAccountInfo[]>([])
const kubernetesStore = useKubernetesStore()
const selectedNamespaces = computed(() => kubernetesStore.selectedNamespaces)
const currentPage = ref(1)
const pageSize = ref(10)
const labelDialogVisible = ref(false)
const labelList = ref<{ key: string; value: string }[]>([])
const yamlDialogVisible = ref(false)
const yamlDialogTitle = ref('')
const yamlContent = ref('')
const yamlSaving = ref(false)
const editingItem = ref<ServiceAccountInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 默认 ServiceAccount YAML 模板
const defaultServiceAccountYaml = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-serviceaccount
  namespace: default
`.trim()

// 过滤后的数据
const filteredData = computed(() => {
  let result = serviceAccounts.value
  if (props.searchName) {
    result = result.filter(item =>
      item.name.toLowerCase().includes(props.searchName!.toLowerCase())
    )
  }
  if (selectedNamespaces.value.length > 0) {
    if (selectedNamespaces.value.length > 1) {
      result = result.filter(item => selectedNamespaces.value.includes(item.namespace))
    }
  }
  return result
})

// 分页后的数据
const paginatedList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredData.value.slice(start, end)
})

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    let nsParam = undefined
    if (selectedNamespaces.value.length === 1) {
      nsParam = selectedNamespaces.value[0]
    }
    const data = await getServiceAccounts(props.clusterId, nsParam)
    serviceAccounts.value = data || []
  } catch (error) {
    ElMessage.error('获取 ServiceAccount 列表失败')
  } finally {
    loading.value = false
  }
}

// 分页
const handlePageChange = () => {}
const handleSizeChange = () => {}

// 显示标签
const showLabels = (row: ServiceAccountInfo) => {
  const labels = row.labels || {}
  labelList.value = Object.keys(labels).map(key => ({
    key,
    value: labels[key]
  }))
  labelDialogVisible.value = true
}

// 新增
const handleCreate = () => {
  yamlDialogTitle.value = '新增 ServiceAccount'
  yamlContent.value = defaultServiceAccountYaml
  editingItem.value = null
  yamlDialogVisible.value = true
}

// 编辑
const handleEdit = async (row: ServiceAccountInfo) => {
  try {
    const token = localStorage.getItem('token') || ''
    const response: any = await axios.get(
      `/api/v1/plugins/kubernetes/resources/serviceaccounts/${row.namespace}/${row.name}/yaml`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    // 清理不需要的字段并修复 apiVersion
    const cleanData = cleanK8sResource(response.data.data || response.data)
    // 将返回的JSON对象转换为YAML字符串
    yamlContent.value = yaml.dump(cleanData, {
      indent: 2,
      lineWidth: -1,
      noRefs: true,
      sortKeys: false
    })
    yamlDialogTitle.value = `ServiceAccount YAML - ${row.name}`
    editingItem.value = row
    yamlDialogVisible.value = true
  } catch (error) {
    ElMessage.error('获取 ServiceAccount YAML 失败')
  }
}

// 清理 K8s 资源对象，移除不需要的字段
const cleanK8sResource = (data: any): any => {
  if (!data || typeof data !== 'object') return data

  // 深拷贝避免修改原数据
  const cleaned = JSON.parse(JSON.stringify(data))

  // 移除 metadata 中的不需要字段
  if (cleaned.metadata) {
    delete cleaned.metadata.managedFields
    delete cleaned.metadata.creationTimestamp
    delete cleaned.metadata.resourceVersion
    delete cleaned.metadata.selfLink
    delete cleaned.metadata.uid
    delete cleaned.metadata.generation
  }

  // 修复 apiVersion 空串问题
  if (!cleaned.apiVersion || cleaned.apiVersion === '') {
    // 根据 kind 设置默认 apiVersion
    const kindToApiVersion: Record<string, string> = {
      ServiceAccount: 'v1',
      Role: 'rbac.authorization.k8s.io/v1',
      RoleBinding: 'rbac.authorization.k8s.io/v1',
      ClusterRole: 'rbac.authorization.k8s.io/v1',
      ClusterRoleBinding: 'rbac.authorization.k8s.io/v1',
      PodSecurityPolicy: 'policy/v1beta1'
    }
    cleaned.apiVersion = kindToApiVersion[cleaned.kind] || 'v1'
  }

  return cleaned
}

// 删除
const handleDelete = async (row: ServiceAccountInfo) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除 ServiceAccount "${row.name}" 吗？此操作不可撤销。`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    const token = localStorage.getItem('token') || ''
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/serviceaccounts/${row.namespace}/${row.name}`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('删除成功')
    await loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + (error.response?.data?.message || error.message))
    }
  }
}

// 保存 YAML
const handleSaveYaml = async () => {
  yamlSaving.value = true
  try {
    const token = localStorage.getItem('token') || ''
    const namespace = props.namespace || 'default'

    if (editingItem.value) {
      // 编辑模式 - 使用 YAML 更新 API
      await axios.put(
        `/api/v1/plugins/kubernetes/resources/serviceaccounts/${editingItem.value.namespace}/${editingItem.value.name}/yaml`,
        { clusterId: props.clusterId, yaml: yamlContent.value },
        { headers: { Authorization: `Bearer ${token}` } }
      )
      ElMessage.success('更新成功')
    } else {
      // 新增模式
      await axios.post(
        `/api/v1/plugins/kubernetes/resources/serviceaccounts/${namespace}/yaml`,
        yamlContent.value,
        {
          params: { clusterId: props.clusterId },
          headers: {
            'Content-Type': 'application/yaml',
            Authorization: `Bearer ${token}`
          }
        }
      )
      ElMessage.success('创建成功')
    }
    yamlDialogVisible.value = false
    await loadData()
  } catch (error: any) {
    ElMessage.error('保存失败: ' + (error.response?.data?.message || error.message))
  } finally {
    yamlSaving.value = false
  }
}

// YAML编辑器输入处理
const handleYamlInput = () => {
  // 可以添加输入验证
}

// YAML编辑器滚动处理（同步行号滚动）
const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

// 简单的 JSON 转 YAML (临时方案)
const JSONToYAML = (obj: any): string => {
  const yaml: string[] = []
  const convert = (o: any, indent = 0) => {
    const spaces = '  '.repeat(indent)
    if (Array.isArray(o)) {
      o.forEach(item => {
        if (typeof item === 'object' && item !== null) {
          yaml.push(spaces + '- ')
          const firstKey = Object.keys(item)[0]
          if (firstKey) {
            yaml.push(spaces + firstKey + ':')
            convert(item[firstKey], indent + 1)
          }
        } else {
          yaml.push(spaces + '- ' + item)
        }
      })
    } else if (typeof o === 'object' && o !== null) {
      Object.keys(o).forEach(key => {
        const value = o[key]
        if (value === null) {
          yaml.push(spaces + key + ': null')
        } else if (Array.isArray(value)) {
          yaml.push(spaces + key + ':')
          convert(value, indent + 1)
        } else if (typeof value === 'object') {
          yaml.push(spaces + key + ':')
          convert(value, indent + 1)
        } else {
          yaml.push(spaces + key + ': ' + value)
        }
      })
    }
  }
  convert(obj)
  return yaml.join('\n')
}

// 监听 props 变化
watch(() => [props.clusterId, props.namespace], () => {
  if (props.clusterId) {
    loadData()
  }
}, { immediate: true })

// 监听筛选后的数据变化，更新计数
watch(filteredData, (newData) => {
  emit('count-update', newData?.length || 0)
})

// 暴露方法给父组件
defineExpose({
  handleCreate
})
</script>

<style scoped>
.service-accounts-tab {
  width: 100%;
}

.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.search-section {
  flex: 1;
}

.search-input {
  width: 300px;
}

.search-icon {
  color: #909399;
}

.action-buttons {
  display: flex;
  gap: 12px;
}

/* 按钮样式 - 使用全局样式 .black-button */

.table-wrapper {
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.name-icon-wrapper {
  width: 32px;
  height: 32px;
  border-radius: 0;
  background: #0a466a;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  flex-shrink: 0;
}

.name-icon {
  color: #ffffff;
  font-size: 14px;
}

.name-text {
  font-weight: 600;
  color: #303133;
}

.secrets-count {
  color: #606266;
  font-weight: 500;
}

.label-cell {
  display: flex;
  justify-content: center;
  align-items: center;
  cursor: pointer;
  padding: 5px 0;
}

.label-badge-wrapper {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.label-icon {
  color: #ffffff;
  font-size: 18px;
}

.label-count {
  position: absolute;
  top: -6px;
  right: -6px;
  background-color: #ffffff;
  color: #000;
  font-size: 10px;
  font-weight: 600;
  min-width: 16px;
  height: 16px;
  line-height: 16px;
  padding: 0 4px;
  border-radius: 0;
  text-align: center;
  border: none;
  z-index: 1;
}

.label-cell:hover .label-icon {
  color: #bfa13f;
  transform: scale(1.1);
}

.label-cell:hover .label-count {
  background-color: #bfa13f;
}

.action-btn {
  color: #ffffff;
  margin: 0 4px;
}

.action-btn.danger {
  color: #f56c6c;
}

.action-btn:hover {
  transform: scale(1.1);
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

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  border-top: 1px solid #f0f0f0;
}
</style>
