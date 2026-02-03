<template>
  <div class="roles-tab">
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
                <el-icon class="name-icon"><Key /></el-icon>
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

      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredData.length"
          layout="total, sizes, prev, pager, next, jumper"
        />
      </div>
    </div>

    <!-- YAML 弹窗 -->
    <el-dialog v-model="yamlDialogVisible" :title="yamlDialogTitle" width="900px" class="yaml-dialog">
      <div class="yaml-editor-wrapper">
        <div class="yaml-line-numbers">
          <div v-for="line in yamlLineCount" :key="line" class="line-number">{{ line }}</div>
        </div>
        <textarea
          v-model="yamlContent"
          class="yaml-textarea"
          spellcheck="false"
          @scroll="handleYamlScroll"
          ref="yamlTextarea"
        ></textarea>
      </div>
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
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Key, Edit, Delete, Plus, Document } from '@element-plus/icons-vue'
import { getRoles, createRoleFromYAML, updateRoleFromYAML, deleteRole, type RoleInfo } from '@/api/kubernetes'
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
const roles = ref<RoleInfo[]>([])
const kubernetesStore = useKubernetesStore()
const selectedNamespaces = computed(() => kubernetesStore.selectedNamespaces)
const currentPage = ref(1)
const pageSize = ref(10)

// YAML 编辑
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedRole = ref<RoleInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const saving = ref(false)
const isCreateMode = ref(false)

// YAML对话框标题
const yamlDialogTitle = computed(() => {
  if (isCreateMode.value) {
    return '新增 Role (YAML)'
  }
  return `Role YAML - ${selectedRole.value?.name || ''}`
})

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

const filteredData = computed(() => {
  let result = roles.value
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

const paginatedList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredData.value.slice(start, end)
})

const loadData = async () => {
  loading.value = true
  try {
    let nsParam = undefined
    if (selectedNamespaces.value.length === 1) {
      nsParam = selectedNamespaces.value[0]
    }
    const data = await getRoles(props.clusterId, nsParam)
    roles.value = data || []
  } catch (error) {
    ElMessage.error('获取 Role 列表失败')
  } finally {
    loading.value = false
  }
}

// YAML 创建
const handleCreate = () => {
  isCreateMode.value = true
  selectedRole.value = null
  // 默认 Role YAML 模板
  const ns = props.namespace || 'default'
  yamlContent.value = `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: example-role
  namespace: ${ns}
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
`
  yamlDialogVisible.value = true
}

// 编辑 YAML
const handleEdit = async (row: RoleInfo) => {
  selectedRole.value = row
  isCreateMode.value = false

  try {
    const token = localStorage.getItem('token') || ''
    const response: any = await axios.get(
      `/api/v1/plugins/kubernetes/resources/roles/${row.namespace}/${row.name}/yaml`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    // 清理不需要的字段并修复 apiVersion
    const cleanData = cleanK8sResource(response.data.data)
    // 将返回的JSON对象转换为YAML字符串
    yamlContent.value = yaml.dump(cleanData, {
      indent: 2,
      lineWidth: -1,
      noRefs: true,
      sortKeys: false
    })
    yamlDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
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

// 删除 Role
const handleDelete = async (row: RoleInfo) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除 Role ${row.name} 吗？此操作不可恢复！`,
      '删除 Role 确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error'
      }
    )
    // 先执行删除
    await deleteRole(props.clusterId, row.namespace, row.name)
    ElMessage.success('删除成功')
    // 删除成功后再刷新列表，刷新失败不影响删除结果
    try {
      await loadData()
    } catch (refreshError) {
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.message || '删除失败')
    }
  }
}

// 保存 YAML
const handleSaveYAML = async () => {
  saving.value = true
  try {
    // 从 YAML 中解析对象
    const yamlObj: any = yaml.load(yamlContent.value)
    if (!yamlObj || !yamlObj.metadata || !yamlObj.metadata.name) {
      ElMessage.error('YAML 中缺少必要的 metadata.name 字段')
      return
    }
    const name = yamlObj.metadata.name
    const namespace = yamlObj.metadata.namespace || props.namespace || 'default'

    if (isCreateMode.value) {
      // 创建模式
      await createRoleFromYAML(props.clusterId, namespace, yamlObj)
      ElMessage.success('创建成功')
    } else {
      // 编辑模式 - 调用更新 API
      await updateRoleFromYAML(props.clusterId, selectedRole.value!.namespace, selectedRole.value!.name, yamlObj)
      ElMessage.success('更新成功')
    }

    yamlDialogVisible.value = false
    await loadData()
  } catch (error: any) {
    ElMessage.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    saving.value = false
  }
}

// YAML编辑器滚动处理（同步行号滚动）
const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

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
  handleCreate,
  loadData
})
</script>

<style scoped>
.roles-tab { width: 100%; }

.table-wrapper { background: #fff; border-radius: 0; box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06); overflow: hidden; }
.modern-table { width: 100%; }
.modern-table :deep(.el-table__row:hover) { background-color: #f8fafc !important; }
.name-cell { display: flex; align-items: center; gap: 10px; }
.name-icon-wrapper { width: 32px; height: 32px; border-radius: 0; background: #0a466a; display: flex; align-items: center; justify-content: center; border: none; flex-shrink: 0; }
.name-icon { color: #ffffff; font-size: 14px; }
.name-text { font-weight: 600; color: #303133; }
.action-btn { color: #ffffff; margin: 0 4px; }
.action-btn.danger { color: #f56c6c; }
.action-btn:hover { transform: scale(1.1); }
.pagination-wrapper { display: flex; justify-content: flex-end; padding: 16px 20px; border-top: 1px solid #f0f0f0; }

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

/* 按钮样式 - 使用全局样式 .black-button */
</style>
