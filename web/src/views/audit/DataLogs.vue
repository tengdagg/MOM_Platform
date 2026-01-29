<template>
  <div class="data-logs-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><DataLine /></el-icon>
        </div>
        <div>
          <h2 class="page-title">数据日志</h2>
          <p class="page-subtitle">记录系统数据的变更历史</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleSearch">
          <el-icon style="margin-right: 6px;"><Search /></el-icon>
          查询
        </el-button>
        <el-button class="black-button" @click="handleReset">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          重置
        </el-button>
        <el-button class="black-button danger" @click="handleBatchDelete" :disabled="selectedIds.length === 0">
          <el-icon style="margin-right: 6px;"><Delete /></el-icon>
          批量删除
        </el-button>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <el-input
        v-model="searchForm.username"
        placeholder="搜索用户名..."
        clearable
        class="filter-input"
      >
        <template #prefix>
          <el-icon class="filter-icon"><User /></el-icon>
        </template>
      </el-input>
      <el-select
        v-model="searchForm.tableName"
        placeholder="数据表"
        clearable
        class="filter-select"
      >
        <el-option label="用户表" value="sys_user" />
        <el-option label="角色表" value="sys_role" />
        <el-option label="部门表" value="sys_department" />
        <el-option label="菜单表" value="sys_menu" />
        <el-option label="岗位表" value="sys_position" />
      </el-select>
      <el-select
        v-model="searchForm.action"
        placeholder="操作类型"
        clearable
        class="filter-select"
      >
        <el-option label="创建" value="create" />
        <el-option label="更新" value="update" />
        <el-option label="删除" value="delete" />
      </el-select>
      <el-date-picker
        v-model="dateRange"
        type="daterange"
        range-separator="至"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        value-format="YYYY-MM-DD"
        class="filter-date"
      />
    </div>

    <!-- 数据表格 -->
    <div class="table-wrapper">
      <el-table
        :data="logList"
        v-loading="loading"
        @selection-change="handleSelectionChange"
        class="modern-table"
        size="default"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column label="ID" prop="id" width="80" align="center">
          <template #default="{ row }">
            <span class="id-text">#{{ row.id }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作用户" prop="username" min-width="140">
          <template #default="{ row }">
            <div class="user-cell">
              <el-icon class="user-icon"><User /></el-icon>
              <span>{{ row.realName || row.username }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="数据表" prop="tableName" width="140">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.tableName }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="记录ID" prop="recordId" width="90" align="center">
          <template #default="{ row }">
            <span class="id-text">{{ row.recordId }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" prop="action" width="80">
          <template #default="{ row }">
            <el-tag :type="getActionType(row.action)" size="small">
              {{ getActionLabel(row.action) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="变更字段" prop="diffFields" min-width="150" show-overflow-tooltip />
        <el-table-column label="数据变更" width="100" align="center">
          <template #default="{ row }">
            <el-button link class="detail-btn" @click="showDataDiff(row)">
              查看详情
            </el-button>
          </template>
        </el-table-column>
        <el-table-column label="IP地址" prop="ip" width="130" />
        <el-table-column label="操作时间" prop="createdAt" width="170" />
        <el-table-column label="操作" width="80" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
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
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next"
          @size-change="loadLogList"
          @current-change="loadLogList"
        />
      </div>
    </div>

    <!-- 数据变更详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="数据变更详情"
      width="700px"
      class="detail-dialog"
      :close-on-click-modal="false"
    >
      <div v-if="currentRow" class="detail-content">
        <div class="detail-info">
          <div class="info-item">
            <span class="info-label">操作用户:</span>
            <span class="info-value">{{ currentRow.realName || currentRow.username }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">数据表:</span>
            <el-tag size="small" type="info">{{ currentRow.tableName }}</el-tag>
          </div>
          <div class="info-item">
            <span class="info-label">记录ID:</span>
            <span class="info-value">{{ currentRow.recordId }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">操作类型:</span>
            <el-tag :type="getActionType(currentRow.action)" size="small">
              {{ getActionLabel(currentRow.action) }}
            </el-tag>
          </div>
        </div>

        <div v-if="currentRow.action === 'create'" class="data-section">
          <h4 class="section-title">新增数据</h4>
          <pre class="json-data">{{ formatJson(currentRow.newData) }}</pre>
        </div>

        <div v-else-if="currentRow.action === 'update'" class="data-section">
          <h4 class="section-title">原始数据</h4>
          <pre class="json-data">{{ formatJson(currentRow.oldData) }}</pre>
          <h4 class="section-title">新数据</h4>
          <pre class="json-data">{{ formatJson(currentRow.newData) }}</pre>
          <div v-if="currentRow.diffFields" class="diff-section">
            <h4 class="section-title">变更字段</h4>
            <el-tag type="warning" size="small">{{ currentRow.diffFields }}</el-tag>
          </div>
        </div>

        <div v-else-if="currentRow.action === 'delete'" class="data-section">
          <h4 class="section-title">删除数据</h4>
          <pre class="json-data">{{ formatJson(currentRow.oldData) }}</pre>
        </div>
      </div>

      <template #footer>
        <el-button @click="detailDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { DataLine, Delete, Search, Refresh, User } from '@element-plus/icons-vue'
import { getDataLogList, deleteDataLog, deleteDataLogsBatch } from '@/api/audit'

// 搜索表单
const searchForm = reactive({
  username: '',
  tableName: '',
  action: '',
  startTime: '',
  endTime: ''
})

// 日期范围
const dateRange = ref<[string, string]>([])

// 监听日期范围变化
watch(dateRange, (newVal) => {
  if (newVal && newVal.length === 2) {
    searchForm.startTime = newVal[0]
    searchForm.endTime = newVal[1]
  } else {
    searchForm.startTime = ''
    searchForm.endTime = ''
  }
})

// 日志列表
const logList = ref<any[]>([])
const loading = ref(false)

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 选中的ID
const selectedIds = ref<number[]>([])

// 详情对话框
const detailDialogVisible = ref(false)
const currentRow = ref<any>(null)

// 加载日志列表
const loadLogList = async () => {
  loading.value = true
  try {
    const res: any = await getDataLogList({
      page: pagination.page,
      pageSize: pagination.pageSize,
      ...searchForm
    })
    logList.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    ElMessage.error('获取日志列表失败')
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  pagination.page = 1
  loadLogList()
}

// 重置
const handleReset = () => {
  searchForm.username = ''
  searchForm.tableName = ''
  searchForm.action = ''
  searchForm.startTime = ''
  searchForm.endTime = ''
  dateRange.value = []
  pagination.page = 1
  loadLogList()
}

// 删除
const handleDelete = (row: any) => {
  ElMessageBox.confirm('确定要删除这条日志吗?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteDataLog(row.id)
      ElMessage.success('删除成功')
      loadLogList()
    } catch (error) {
      ElMessage.error('删除失败')
    }
  })
}

// 批量删除
const handleBatchDelete = () => {
  ElMessageBox.confirm(`确定要删除选中的 ${selectedIds.value.length} 条日志吗?`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteDataLogsBatch(selectedIds.value)
      ElMessage.success('删除成功')
      selectedIds.value = []
      loadLogList()
    } catch (error) {
      ElMessage.error('删除失败')
    }
  })
}

// 选择变化
const handleSelectionChange = (selection: any[]) => {
  selectedIds.value = selection.map(item => item.id)
}

// 显示数据差异
const showDataDiff = (row: any) => {
  currentRow.value = row
  detailDialogVisible.value = true
}

// 获取操作类型标签样式
const getActionType = (action: string) => {
  const map: Record<string, string> = {
    'create': 'success',
    'update': 'warning',
    'delete': 'danger'
  }
  return map[action] || 'info'
}

// 获取操作类型标签
const getActionLabel = (action: string) => {
  const map: Record<string, string> = {
    'create': '创建',
    'update': '更新',
    'delete': '删除'
  }
  return map[action] || action
}

// 格式化JSON
const formatJson = (jsonStr: string) => {
  try {
    const obj = JSON.parse(jsonStr)
    return JSON.stringify(obj, null, 2)
  } catch {
    return jsonStr
  }
}

// 实时搜索
watch([() => searchForm.username, () => searchForm.tableName, () => searchForm.action], () => {
  pagination.page = 1
  loadLogList()
})

onMounted(() => {
  loadLogList()
})
</script>

<style scoped>
.data-logs-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
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
  background: #0a466a;
  border-radius: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  font-size: 22px;
  flex-shrink: 0;
  border: none;
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

.black-button {
  background-color: #0a466a !important;
  color: #ffffff !important;
  border-color: #0a466a !important;
  border-radius: 0;
  padding: 10px 20px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #0d5a87 !important;
  border-color: #0d5a87 !important;
}

.black-button.danger {
  background-color: #f56c6c !important;
  border-color: #f56c6c !important;
}

.black-button.danger:hover {
  background-color: #f78989 !important;
}

.black-button:disabled {
  background-color: #c0c4cc !important;
  border-color: #c0c4cc !important;
}

/* 筛选栏 */
.filter-bar {
  margin-bottom: 16px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  gap: 12px;
  align-items: center;
}

.filter-input {
  width: 200px;
}

.filter-select {
  width: 140px;
}

.filter-date {
  width: 260px;
}

.filter-icon {
  color: #ffffff;
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

.id-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #909399;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-icon {
  color: #ffffff;
  font-size: 16px;
}

.detail-btn {
  color: #ffffff;
}

.detail-btn:hover {
  color: #bfa13f;
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

/* 详情对话框 */
.detail-dialog :deep(.el-dialog__header) {
  background: linear-gradient(135deg, #0a466a 0%, #0d5a87 100%);
  color: #ffffff;
  border-radius: 0;
  padding: 20px 24px;
}

.detail-dialog :deep(.el-dialog__title) {
  color: #ffffff;
  font-size: 16px;
  font-weight: 600;
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-info {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  padding: 16px;
  background: #f8fafc;
  border-radius: 0;
  border: 1px solid #e4e7ed;
}

.info-item {
  display: flex;
  gap: 10px;
  align-items: center;
}

.info-label {
  color: #606266;
  font-weight: 600;
  font-size: 14px;
  min-width: 70px;
}

.info-value {
  color: #303133;
  font-size: 14px;
  font-weight: 500;
}

.data-section {
  margin-top: 8px;
}

.section-title {
  margin: 16px 0 8px 0;
  font-size: 14px;
  color: #606266;
  font-weight: 600;
}

.diff-section {
  margin-top: 16px;
}

.json-data {
  background: #1a1a1a;
  color: #ffffff;
  padding: 16px;
  border-radius: 0;
  font-size: 13px;
  font-family: 'Monaco', 'Menlo', monospace;
  max-height: 300px;
  overflow-y: auto;
  margin: 0;
  border: none;
}
</style>
