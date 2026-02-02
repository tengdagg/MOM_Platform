<template>
  <div class="ingressclass-list">
    <div class="search-bar">
      <div class="search-bar-left">
        <el-input v-model="searchName" placeholder="搜索 IngressClass 名称..." clearable class="search-input" @input="handleSearch">
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>
      </div>
    </div>

    <div class="table-wrapper">
      <el-table :data="filteredIngressClasses" v-loading="loading" class="modern-table" size="default" :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }">
        <el-table-column label="名称" prop="name" min-width="200" fixed>
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-blue"><Connection /></el-icon>
              名称
            </span>
          </template>
          <template #default="{ row }">
            <div class="name-cell">
              <el-icon class="name-icon"><Connection /></el-icon>
              <div>
                <div class="name-text">{{ row.name }}</div>
                <el-tag v-if="row.isDefault" size="small" type="success" class="default-tag">默认</el-tag>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="控制器" min-width="300">
          <template #default="{ row }">
            <span class="controller-text">{{ row.controller }}</span>
          </template>
        </el-table-column>
        <el-table-column label="参数" min-width="200">
          <template #default="{ row }">
            <div v-if="row.parameters">
              <el-tag size="small" type="info">{{ row.parameters.kind }}/{{ row.parameters.name }}</el-tag>
            </div>
            <span v-else class="empty-text">-</span>
          </template>
        </el-table-column>
        <el-table-column label="存活时间" prop="age" width="120" />
        <el-table-column label="操作" width="100" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="查看 YAML" placement="top">
                <el-button link class="action-btn" @click="handleViewYAML(row)">
                  <el-icon :size="18"><Document /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="yamlDialogVisible" :title="`IngressClass YAML - ${selectedIngressClass?.name}`" width="900px" :lock-scroll="false" class="yaml-dialog">
      <div class="yaml-editor-wrapper">
        <div class="yaml-line-numbers">
          <div v-for="line in yamlLineCount" :key="line" class="line-number">{{ line }}</div>
        </div>
        <textarea
          v-model="yamlContent"
          class="yaml-textarea"
          spellcheck="false"
          readonly
          ref="yamlTextarea"
        ></textarea>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="yamlDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Document, Connection } from '@element-plus/icons-vue'
import { getIngressClasses, getIngressClassYAML, type IngressClassInfo } from '@/api/kubernetes'
import { dump } from 'js-yaml'

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['refresh', 'count-update'])

const loading = ref(false)
const ingressClassList = ref<IngressClassInfo[]>([])
const searchName = ref('')
const selectedIngressClass = ref<IngressClassInfo | null>(null)

// YAML 查看相关
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 过滤后的 IngressClass 列表
const filteredIngressClasses = computed(() => {
  let result = ingressClassList.value
  if (searchName.value) {
    result = result.filter(ic => ic.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  return result
})

// 加载 IngressClass 列表
const loadIngressClasses = async (showSuccess = false) => {
  if (!props.clusterId) return

  loading.value = true
  try {
    const data = await getIngressClasses(props.clusterId)
    ingressClassList.value = data || []
    emit('count-update', ingressClassList.value.length)
    if (showSuccess) {
      ElMessage.success('刷新成功')
    }
  } catch (error) {
    ElMessage.error('获取 IngressClass 列表失败')
  } finally {
    loading.value = false
  }
}

// 搜索处理
const handleSearch = () => {
  // 本地过滤，不需要重新请求
}

// 查看 YAML
const handleViewYAML = async (ingressClass: IngressClassInfo) => {
  selectedIngressClass.value = ingressClass
  try {
    const response = await getIngressClassYAML(props.clusterId!, ingressClass.name)
    const data = response.items || response
    yamlContent.value = dump(data, { noRefs: true, sortKeys: false })
    yamlDialogVisible.value = true
  } catch (error) {
    ElMessage.error('获取 YAML 失败')
  }
}

// 暴露刷新方法给父组件
const refresh = () => {
  loadIngressClasses(true)
}

// 监听 clusterId 变化
watch(() => props.clusterId, () => {
  if (props.clusterId) {
    loadIngressClasses()
  }
}, { immediate: true })

defineExpose({
  refresh,
  loadIngressClasses
})
</script>

<style scoped>
.ingressclass-list {
  height: 100%;
}

.search-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  gap: 10px;
}

.search-bar-left {
  display: flex;
  gap: 10px;
  align-items: center;
}

.search-input {
  width: 280px;
}

.search-icon {
  color: #909399;
}

.table-wrapper {
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.header-with-icon {
  display: flex;
  align-items: center;
  gap: 6px;
}

.header-icon {
  font-size: 14px;
}

.header-icon-blue {
  color: #409eff;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.name-icon {
  font-size: 20px;
  color: #909399;
}

.name-text {
  font-weight: 500;
  color: #303133;
}

.default-tag {
  margin-top: 4px;
}

.controller-text {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  color: #606266;
}

.empty-text {
  color: #909399;
}

.action-buttons {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.action-btn {
  padding: 4px;
  color: #606266;
}

.action-btn:hover {
  color: #409eff;
}

/* YAML 弹窗样式 */
.yaml-dialog :deep(.el-dialog__body) {
  padding: 0;
}

.yaml-editor-wrapper {
  display: flex;
  height: 500px;
  background: #1e1e1e;
  border-radius: 4px;
  overflow: hidden;
}

.yaml-line-numbers {
  width: 50px;
  background: #252526;
  color: #858585;
  text-align: right;
  padding: 10px 8px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
  overflow-y: hidden;
  user-select: none;
}

.line-number {
  height: 20.8px;
}

.yaml-textarea {
  flex: 1;
  background: #1e1e1e;
  color: #d4d4d4;
  border: none;
  padding: 10px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
  resize: none;
  outline: none;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
