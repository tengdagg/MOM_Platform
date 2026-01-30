<template>
  <div class="position-info-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Briefcase /></el-icon>
        </div>
        <div>
          <h2 class="page-title">岗位信息</h2>
          <p class="page-subtitle">管理系统岗位信息，支持岗位与用户的关联管理</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增岗位
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.postCode"
          placeholder="搜索岗位编码..."
          clearable
          class="search-input"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-input
          v-model="searchForm.postName"
          placeholder="搜索岗位名称..."
          clearable
          class="search-input"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <div class="search-actions">
        <el-button class="reset-btn" @click="resetSearch">
          <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <!-- 表格容器 -->
    <div class="table-wrapper">
      <el-table
        :data="positionList"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column prop="postCode" min-width="120">
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-blue"><Key /></el-icon>
              岗位编码
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="postName" min-width="150">
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-blue"><User /></el-icon>
              岗位名称
            </span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.postStatus === 1 ? 'success' : 'danger'" effect="dark">
              {{ row.postStatus === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createTime" label="创建时间" min-width="180" />
        <el-table-column prop="remark" label="备注" min-width="200" show-overflow-tooltip />
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn action-edit" @click="handleEdit(row)">
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="分配用户" placement="top">
                <el-button link class="action-btn action-auth" @click="handleAssignUsers(row)">
                  <el-icon><UserFilled /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button link class="action-btn action-delete" @click="handleDelete(row)">
                  <el-icon><Delete /></el-icon>
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
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadPositionList"
          @current-change="loadPositionList"
        />
      </div>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="50%"
      class="position-edit-dialog responsive-dialog"
      @close="handleDialogClose"
    >
      <el-form :model="positionForm" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="岗位名称" prop="postName">
          <el-input v-model="positionForm.postName" placeholder="请输入岗位名称" />
        </el-form-item>
        <el-form-item label="岗位编码" prop="postCode">
          <el-input v-model="positionForm.postCode" placeholder="请输入岗位编码" />
        </el-form-item>
        <el-form-item label="备注" prop="remark">
          <el-input
            v-model="positionForm.remark"
            type="textarea"
            :rows="3"
            placeholder="请输入备注"
          />
        </el-form-item>
        <el-form-item label="状态" prop="postStatus">
          <el-radio-group v-model="positionForm.postStatus">
            <el-radio :label="1">启用</el-radio>
            <el-radio :label="2">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleSubmit" :loading="submitLoading">
            确定
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 分配用户对话框 -->
    <el-dialog
      v-model="assignUsersVisible"
      title="分配用户"
      width="70%"
      class="assign-users-dialog responsive-dialog"
      @close="handleAssignUsersClose"
    >
      <div class="assign-content">
        <!-- 已分配用户 -->
        <div class="panel assigned-panel">
          <div class="panel-header">
            <div class="panel-title">
              <el-icon class="panel-icon"><UserFilled /></el-icon>
              <span>已分配用户</span>
              <el-badge :value="selectedUsers.length" class="badge" />
            </div>
          </div>
          <div class="panel-body">
            <el-empty v-if="selectedUsers.length === 0" description="暂无已分配用户" :image-size="60" />
            <div v-else class="user-cards-container">
              <div
                v-for="user in selectedUsers"
                :key="user.id || user.ID"
                class="user-card selected-card"
              >
                <div class="user-avatar">
                  <el-icon class="avatar-icon"><User /></el-icon>
                </div>
                <div class="user-info">
                  <div class="user-name">{{ user.realName || user.username }}</div>
                  <div class="user-username">@{{ user.username }}</div>
                </div>
                <el-button
                  type="danger"
                  size="small"
                  circle
                  class="remove-btn"
                  @click="removeUser(user)"
                >
                  <el-icon><Close /></el-icon>
                </el-button>
              </div>
            </div>
          </div>
        </div>

        <!-- 可用用户 -->
        <div class="panel available-panel">
          <div class="panel-header">
            <div class="panel-title">
              <el-icon class="panel-icon"><User /></el-icon>
              <span>可用用户</span>
            </div>
            <el-input
              v-model="userSearchKeyword"
              placeholder="搜索用户名..."
              clearable
              size="small"
              class="search-box"
              @input="loadAvailableUsers"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
          </div>
          <div class="panel-body">
            <div v-loading="usersLoading" class="user-cards-container">
              <el-empty v-if="availableUsers.length === 0 && !usersLoading" description="暂无可分配用户" :image-size="60" />
              <div
                v-for="user in availableUsers"
                :key="user.id || user.ID"
                class="user-card available-card"
              >
                <div class="user-avatar">
                  <el-icon class="avatar-icon"><User /></el-icon>
                </div>
                <div class="user-info">
                  <div class="user-name">{{ user.realName || user.username }}</div>
                  <div class="user-username">@{{ user.username }}</div>
                </div>
                <el-button
                  type="primary"
                  size="small"
                  circle
                  class="add-btn"
                  @click="addUser(user)"
                >
                  <el-icon><Plus /></el-icon>
                </el-button>
              </div>
            </div>
            <div v-if="userPagination.total > 0" class="panel-pagination">
              <el-pagination
                v-model:current-page="userPagination.page"
                v-model:page-size="userPagination.pageSize"
                :total="userPagination.total"
                :page-sizes="[10, 20, 50]"
                layout="total, sizes, prev, pager, next"
                small
                @size-change="loadAvailableUsers"
                @current-change="loadAvailableUsers"
              />
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="assignUsersVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleAssignUsersSubmit" :loading="submitLoading">
            保存分配
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox, FormInstance } from 'element-plus'
import {
  Search,
  RefreshLeft,
  Plus,
  Edit,
  Delete,
  Key,
  User,
  UserFilled,
  Briefcase,
  Close
} from '@element-plus/icons-vue'
import {
  getPositionList,
  createPosition,
  updatePosition,
  deletePosition,
  getPositionUsers,
  assignUsersToPosition
} from '@/api/position'
import { getUserList } from '@/api/user'

const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const assignUsersVisible = ref(false)
const usersLoading = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const formRef = ref<FormInstance>()

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 搜索表单
const searchForm = reactive({
  postCode: '',
  postName: ''
})

// 岗位列表
const positionList = ref([])

// 岗位表单
const positionForm = reactive({
  id: 0,
  postName: '',
  postCode: '',
  remark: '',
  postStatus: 1
})

// 表单验证规则
const rules = {
  postName: [{ required: true, message: '请输入岗位名称', trigger: 'blur' }],
  postCode: [{ required: true, message: '请输入岗位编码', trigger: 'blur' }]
}

// 用户相关
const currentPositionId = ref(0)
const userSearchKeyword = ref('')
const selectedUsers = ref<any[]>([])
const availableUsers = ref<any[]>([])
const userPagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 实时搜索 - 监听搜索框变化
watch([() => searchForm.postCode, () => searchForm.postName], () => {
  pagination.page = 1
  loadPositionList()
})

// 加载岗位列表
const loadPositionList = async () => {
  loading.value = true
  try {
    const res: any = await getPositionList({
      page: pagination.page,
      pageSize: pagination.pageSize,
      postCode: searchForm.postCode || undefined,
      postName: searchForm.postName || undefined
    })
    // 修复：后端现在返回 {list, page, pageSize, total} 格式
    positionList.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
    // 使用模拟数据
    if (searchForm.postCode || searchForm.postName) {
      positionList.value = []
    } else {
      positionList.value = [
        { id: 1, postCode: 'AAA', postName: '研发总监', postStatus: 1, createTime: '2023-06-14 20:08:22', remark: '主管各个部门' },
        { id: 10, postCode: 'ops', postName: '运维工程师', postStatus: 1, createTime: '2025-06-28 22:46:33', remark: '运维工程师' },
        { id: 11, postCode: 'dev', postName: '研发工程师', postStatus: 1, createTime: '2025-06-28 22:50:29', remark: '研发工程师' },
        { id: 12, postCode: 'test', postName: '测试工程师', postStatus: 2, createTime: '2025-06-28 22:52:57', remark: '测试工程师' }
      ]
    }
    pagination.total = positionList.value.length
  } finally {
    loading.value = false
  }
}

// 重置搜索
const resetSearch = () => {
  searchForm.postCode = ''
  searchForm.postName = ''
  pagination.page = 1
  loadPositionList()
}

// 重置表单
const resetForm = () => {
  positionForm.id = 0
  positionForm.postName = ''
  positionForm.postCode = ''
  positionForm.remark = ''
  positionForm.postStatus = 1
  formRef.value?.clearValidate()
}

// 新增岗位
const handleAdd = () => {
  resetForm()
  dialogTitle.value = '新增岗位'
  isEdit.value = false
  dialogVisible.value = true
}

// 编辑岗位
const handleEdit = (row: any) => {
  Object.assign(positionForm, row)
  dialogTitle.value = '编辑岗位'
  isEdit.value = true
  dialogVisible.value = true
}

// 删除岗位
const handleDelete = (row: any) => {
  ElMessageBox.confirm(`确定要删除岗位"${row.postName}"吗？`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deletePosition(row.id)
      ElMessage.success('删除成功')
      loadPositionList()
    } catch (error) {
      ElMessage.error('删除失败')
    }
  }).catch(() => {})
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitLoading.value = true
    try {
      if (isEdit.value) {
        await updatePosition(positionForm.id, positionForm)
        ElMessage.success('更新成功')
      } else {
        await createPosition(positionForm)
        ElMessage.success('创建成功')
      }
      dialogVisible.value = false
      loadPositionList()
    } catch (error) {
      ElMessage.error(isEdit.value ? '更新失败' : '创建失败')
    } finally {
      submitLoading.value = false
    }
  })
}

// 对话框关闭事件
const handleDialogClose = () => {
  formRef.value?.resetFields()
  resetForm()
}

// 分配用户 - 打开对话框
const handleAssignUsers = async (row: any) => {
  currentPositionId.value = row.id
  selectedUsers.value = []
  userSearchKeyword.value = ''
  userPagination.page = 1

  // 加载已分配的用户
  try {
    const res: any = await getPositionUsers(row.id)
    // 后端返回 {list, total, page, pageSize} 格式
    selectedUsers.value = res.list || res.data || []
  } catch (error) {
    selectedUsers.value = []
  }

  // 加载可用用户列表
  await loadAvailableUsers()

  assignUsersVisible.value = true
}

// 加载可用用户列表
const loadAvailableUsers = async () => {
  usersLoading.value = true
  try {
    const selectedUserIds = selectedUsers.value.map(u => u.id || u.ID)
    const res: any = await getUserList({
      page: userPagination.page,
      pageSize: userPagination.pageSize,
      keyword: userSearchKeyword.value || undefined
    })

    // 用户列表 API 直接返回 res.list 和 res.total
    const userList = res.list || []
    // 过滤掉已选择的用户
    availableUsers.value = userList.filter((u: any) => !selectedUserIds.includes(u.id || u.ID))
    userPagination.total = (res.total || 0) - selectedUsers.value.length
  } catch (error) {
    availableUsers.value = []
  } finally {
    usersLoading.value = false
  }
}

// 添加用户到已选列表
const addUser = (user: any) => {
  const userId = user.id || user.ID
  // 检查是否已添加
  if (selectedUsers.value.some(u => (u.id || u.ID) === userId)) {
    ElMessage.warning('该用户已添加')
    return
  }
  selectedUsers.value.push(user)
  loadAvailableUsers()
}

// 从已选列表移除用户
const removeUser = (user: any) => {
  const userId = user.id || user.ID
  selectedUsers.value = selectedUsers.value.filter(u => (u.id || u.ID) !== userId)
  loadAvailableUsers()
}

// 提交分配用户
const handleAssignUsersSubmit = async () => {
  submitLoading.value = true
  try {
    const userIds = selectedUsers.value.map(u => u.id || u.ID)
    await assignUsersToPosition(currentPositionId.value, userIds)
    ElMessage.success('分配用户成功')
    assignUsersVisible.value = false
  } catch (error) {
    ElMessage.error('分配用户失败')
  } finally {
    submitLoading.value = false
  }
}

// 分配用户对话框关闭事件
const handleAssignUsersClose = () => {
  selectedUsers.value = []
  availableUsers.value = []
  userSearchKeyword.value = ''
  currentPositionId.value = 0
}

onMounted(() => {
  loadPositionList()
})
</script>

<style scoped>
.position-info-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
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

/* 搜索栏 */
.search-bar {
  margin-bottom: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.search-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-input {
  width: 280px;
}

.search-actions {
  display: flex;
  gap: 10px;
}

.reset-btn {
  background: #f5f7fa;
  border-color: #dcdfe6;
  color: #606266;
}

.reset-btn:hover {
  background: #e6e8eb;
  border-color: #c0c4cc;
}

/* 搜索框样式 */
.search-bar :deep(.el-input__wrapper) {
  border-radius: 0;
  border: 1px solid #dcdfe6;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
  transition: all 0.3s ease;
  background-color: #fff;
}

.search-bar :deep(.el-input__wrapper:hover) {
  border-color: #ffffff;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
}

.search-bar :deep(.el-input__wrapper.is-focus) {
  border-color: #ffffff;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.25);
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

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__body-wrapper) {
  border-radius: 0;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
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

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
}

.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn :deep(.el-icon) {
  font-size: 16px;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-auth:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-edit:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
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

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 分配用户对话框 - 新UI */
.assign-content {
  display: flex;
  gap: 20px;
  min-height: 400px;
}

.panel {
  flex: 1;
  border-radius: 0;
  border: 1px solid #e4e7ed;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.assigned-panel {
  background: linear-gradient(135deg, #f5f7fa 0%, #e8eef5 100%);
}

.available-panel {
  background: #fff;
}

.panel-header {
  padding: 16px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 15px;
  color: #303133;
}

.panel-icon {
  font-size: 18px;
  color: #ffffff;
}

.badge {
  margin-left: 8px;
}

.search-box {
  width: 200px;
}

.search-box :deep(.el-input__wrapper) {
  border-radius: 0;
}

.panel-body {
  flex: 1;
  padding: 16px;
  overflow-y: auto;
  max-height: 400px;
}

.user-cards-container {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}

.user-card {
  position: relative;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 0;
  padding: 12px;
  display: flex;
  align-items: center;
  gap: 12px;
  transition: all 0.3s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.04);
}

.user-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.selected-card {
  border-color: #ffffff;
  background: linear-gradient(135deg, #fffaf0 0%, #fff5e6 100%);
}

.available-card {
  background: #fff;
}

.user-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, #d4af37 0%, #c9a227 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.avatar-icon {
  font-size: 20px;
  color: #fff;
}

.user-info {
  flex: 1;
  min-width: 0;
}

.user-name {
  font-weight: 600;
  font-size: 14px;
  color: #303133;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-username {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}

.remove-btn,
.add-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 28px;
  height: 28px;
}

.remove-btn {
  background: #fef0f0;
  border-color: #fde2e2;
  color: #f56c6c;
}

.remove-btn:hover {
  background: #f56c6c;
  border-color: #f56c6c;
  color: #fff;
}

.add-btn {
  background: #ecf5ff;
  border-color: #d9ecff;
  color: #409eff;
}

.add-btn:hover {
  background: #409eff;
  border-color: #409eff;
  color: #fff;
}

.panel-pagination {
  margin-top: 16px;
  display: flex;
  justify-content: center;
}

/* 编辑对话框样式 */
:deep(.position-edit-dialog) {
  border-radius: 0;
}

:deep(.position-edit-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.position-edit-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.position-edit-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

/* 分配用户对话框样式 */
:deep(.assign-users-dialog) {
  border-radius: 0;
}

:deep(.assign-users-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.assign-users-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.assign-users-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

/* 标签样式 */
:deep(.el-tag) {
  border-radius: 0;
  padding: 4px 10px;
  font-weight: 500;
}

/* 空状态样式 */
:deep(.el-empty) {
  padding: 40px 0;
}

/* 响应式对话框 */
:deep(.responsive-dialog) {
  max-width: 1200px;
  min-width: 500px;
}

@media (max-width: 768px) {
  :deep(.responsive-dialog .el-dialog) {
    width: 95% !important;
    max-width: none;
    min-width: auto;
  }
}
</style>
