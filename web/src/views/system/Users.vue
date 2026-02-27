<template>
  <div class="users-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><User /></el-icon>
        </div>
        <div>
          <h2 class="page-title">用户管理</h2>
          <p class="page-subtitle">管理系统用户，支持新增、编辑、角色分配与部门管理</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增用户
        </el-button>
      </div>
    </div>

    <div class="content-wrapper">
      <!-- 左侧部门树 -->
      <div class="dept-tree-panel">
        <div class="panel-header">
          <el-icon><OfficeBuilding /></el-icon>
          <span>部门组织</span>
        </div>
        <el-tree
          ref="treeRef"
          :data="departmentTree"
          :props="treeProps"
          :highlight-current="true"
          node-key="id"
          default-expand-all
          @node-click="handleNodeClick"
          class="dept-tree"
        >
          <template #default="{ node, data }">
            <span class="custom-tree-node">
              <span class="node-label">{{ node.label }}</span>
              <span class="node-count">({{ data.userCount || 0 }})</span>
            </span>
          </template>
        </el-tree>
      </div>

      <!-- 右侧用户列表 -->
      <div class="user-list-panel">
        <!-- 当前选中的部门显示 -->
        <div v-if="selectedDepartment" class="selected-dept-bar">
          <span class="dept-path-text">
            <span class="label">当前部门：</span>
            <span class="path">{{ selectedDepartmentPath }}</span>
          </span>
          <el-button link type="primary" @click="clearDepartmentSelection" v-if="selectedDepartment">
            查看全部用户
          </el-button>
        </div>

        <!-- 搜索栏 -->
        <div class="search-bar">
          <div class="search-inputs">
            <el-input
              v-model="searchForm.keyword"
              placeholder="搜索用户名/邮箱..."
              clearable
              class="search-input"
              @keyup.enter="loadUsers"
            >
              <template #prefix>
                <el-icon class="search-icon"><Search /></el-icon>
              </template>
            </el-input>

            <el-select v-model="searchForm.source" placeholder="用户来源" clearable class="search-input" style="width: 160px">
              <el-option label="全部" value="" />
              <el-option label="本地用户" value="local" />
              <el-option label="LDAP 用户" value="ldap" />
            </el-select>
          </div>

          <div class="search-actions">
            <el-button class="black-button" @click="loadUsers">
              <el-icon style="margin-right: 4px;"><Search /></el-icon>
              查询
            </el-button>
            <el-button class="reset-btn" @click="resetSearch">
              <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
              重置
            </el-button>
          </div>
        </div>

        <!-- 表格容器 -->
        <div class="table-wrapper">
          <el-table
            :data="userList"
            v-loading="loading"
            class="modern-table"
            :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
          >
            <el-table-column label="头像" width="60" align="center">
              <template #default="{ row }">
                <el-avatar v-if="row.avatar" :src="row.avatar" :size="32" />
                <el-avatar v-else :size="32">{{ row.realName?.substring(0, 1) || row.username.substring(0, 1) }}</el-avatar>
              </template>
            </el-table-column>
            <el-table-column prop="username" label="用户名" min-width="150">
              <template #default="{ row }">
                <div class="user-name-cell">
                  <span class="user-name">{{ row.username }}</span>
                  <el-tag v-if="row.source === 'ldap'" size="small" type="warning" effect="dark">LDAP</el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="realName" label="真实姓名" min-width="120" />
            <el-table-column prop="email" label="邮箱" min-width="180">
              <template #default="{ row }">
                <span class="description-text">{{ row.email || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="phone" label="手机号" min-width="130">
              <template #default="{ row }">
                <span class="description-text">{{ row.phone || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="部门" min-width="150">
              <template #default="{ row }">
                <span class="description-text">{{ row.department?.name || row.department?.deptName || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="row.status === 1 ? 'success' : 'danger'" effect="dark">
                  {{ row.status === 1 ? '启用' : '禁用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="200" fixed="right" align="center">
              <template #default="{ row }">
                <div class="action-buttons">
                  <el-tooltip content="编辑" placement="top">
                    <el-button link class="action-btn action-edit" @click="handleEdit(row)">
                      <el-icon><Edit /></el-icon>
                    </el-button>
                  </el-tooltip>
                  <el-tooltip v-if="row.source !== 'ldap'" content="重置密码" placement="top">
                    <el-button link class="action-btn action-password" @click="handleResetPassword(row)">
                      <el-icon><Key /></el-icon>
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
          <div class="pagination-container">
            <el-pagination
              v-model:current-page="pagination.page"
              v-model:page-size="pagination.pageSize"
              :total="pagination.total"
              :page-sizes="[10, 20, 50, 100]"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="loadUsers"
              @current-change="loadUsers"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="55%"
      class="user-dialog responsive-dialog"
      @close="handleDialogClose"
    >
      <el-form :model="userForm" :rules="rules" ref="formRef" label-width="80px" class="user-form">
        <!-- 基本信息 -->
        <div class="form-section-title">
          <el-icon><User /></el-icon>
          <span>基本信息</span>
        </div>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="用户名" prop="username">
              <el-input v-model="userForm.username" :disabled="isEdit" placeholder="请输入用户名">
                <template #prefix>
                  <el-icon><User /></el-icon>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="真实姓名" prop="realName">
              <el-input v-model="userForm.realName" placeholder="请输入真实姓名">
                <template #prefix>
                  <el-icon><Postcard /></el-icon>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="邮箱" prop="email">
              <el-input v-model="userForm.email" placeholder="请输入邮箱">
                <template #prefix>
                  <el-icon><Message /></el-icon>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="手机号" prop="phone">
              <el-input v-model="userForm.phone" placeholder="请输入手机号">
                <template #prefix>
                  <el-icon><Phone /></el-icon>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16" v-if="!isEdit">
          <el-col :span="12">
            <el-form-item label="密码" prop="password">
              <el-input v-model="userForm.password" type="password" show-password placeholder="请输入密码（至少6位）">
                <template #prefix>
                  <el-icon><Lock /></el-icon>
                </template>
              </el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="状态" prop="status">
              <el-radio-group v-model="userForm.status">
                <el-radio :label="1">启用</el-radio>
                <el-radio :label="0">禁用</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16" v-if="isEdit">
          <el-col :span="12">
            <el-form-item label="状态" prop="status">
              <el-radio-group v-model="userForm.status">
                <el-radio :label="1">启用</el-radio>
                <el-radio :label="0">禁用</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>

        <!-- 组织信息 -->
        <div class="form-section-title">
          <el-icon><OfficeBuilding /></el-icon>
          <span>组织信息</span>
        </div>

        <el-form-item label="部门" prop="departmentId">
          <el-tree-select
            v-model="userForm.departmentId"
            :data="departmentTreeData"
            :props="{ label: 'label', value: 'id', children: 'children' }"
            placeholder="请选择部门"
            clearable
            check-strictly
            :render-after-expand="false"
          >
            <template #default="{ data }">
              <span>{{ data.label }}</span>
              <span class="tree-node-count">({{ data.userCount || 0 }})</span>
            </template>
          </el-tree-select>
        </el-form-item>

        <el-form-item label="岗位" prop="positionIds">
          <el-select
            v-model="userForm.positionIds"
            multiple
            placeholder="请选择岗位"
            style="width: 100%"
          >
            <el-option
              v-for="pos in positionOptions"
              :key="'pos-' + (pos.ID || pos.id)"
              :label="pos.postName"
              :value="pos.ID || pos.id"
            />
          </el-select>
        </el-form-item>

        <!-- 权限信息 -->
        <div class="form-section-title">
          <el-icon><Key /></el-icon>
          <span>权限信息</span>
        </div>

        <el-form-item label="角色" prop="roleIds">
          <el-select
            v-model="userForm.roleIds"
            multiple
            placeholder="请选择角色"
            style="width: 100%"
          >
            <el-option
              v-for="role in roleOptions"
              :key="'role-' + role.ID"
              :label="role.name"
              :value="role.ID"
            >
              <span>{{ role.name }}</span>
              <span class="role-code">{{ role.code }}</span>
            </el-option>
          </el-select>
        </el-form-item>

        <!-- 其他信息 -->
        <div class="form-section-title">
          <el-icon><Document /></el-icon>
          <span>其他信息</span>
        </div>

        <el-form-item label="个人简介" prop="bio">
          <el-input
            v-model="userForm.bio"
            type="textarea"
            :rows="3"
            placeholder="请输入个人简介"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmit" :loading="submitLoading">
            <el-icon><Check /></el-icon>
            确定
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog
      v-model="resetPasswordVisible"
      title="重置密码"
      width="40%"
      class="responsive-dialog"
      @close="handleResetPasswordClose"
    >
      <el-form :model="resetPasswordForm" :rules="resetPasswordRules" ref="resetPasswordFormRef" label-width="100px">
        <el-form-item label="用户名">
          <el-input v-model="resetPasswordForm.username" disabled />
        </el-form-item>
        <el-form-item label="新密码" prop="password">
          <el-input v-model="resetPasswordForm.password" type="password" show-password placeholder="请输入新密码（至少6位）" />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input v-model="resetPasswordForm.confirmPassword" type="password" show-password placeholder="请再次输入新密码" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="resetPasswordVisible = false">取消</el-button>
        <el-button type="primary" @click="handleResetPasswordSubmit" :loading="resetPasswordLoading">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch } from 'vue'
import { ElMessage, ElMessageBox, FormInstance } from 'element-plus'
import {
  User, Postcard, Message, Phone, Lock,
  OfficeBuilding, Key, Document, Check,
  Plus, Edit, Delete, Search, RefreshLeft
} from '@element-plus/icons-vue'
import { getUserList, createUser, updateUser, deleteUser, resetUserPassword, assignUserRoles, assignUserPositions } from '@/api/user'
import { getDepartmentTree } from '@/api/department'
import { getAllRoles } from '@/api/role'
import { getPositionList } from '@/api/position'

const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const treeRef = ref()

// 部门树相关
const departmentTree = ref([])
const treeProps = {
  children: 'children',
  label: 'deptName'
}
const selectedDepartment = ref(null)
const selectedDepartmentPath = ref('')

// 角色和岗位选项
const roleOptions = ref([])
const positionOptions = ref([])

// 处理部门树数据，转换为el-tree-select需要的格式
const departmentTreeData = computed(() => {
  const convertTree = (nodes: any[]): any[] => {
    return nodes.map(node => ({
      id: node.id,
      label: node.deptName || node.name,
      userCount: node.userCount || 0,
      children: node.children ? convertTree(node.children) : []
    }))
  }
  return convertTree(departmentTree.value)
})

// 重置密码相关
const resetPasswordVisible = ref(false)
const resetPasswordLoading = ref(false)
const resetPasswordFormRef = ref<FormInstance>()
const resetPasswordForm = reactive({
  userId: 0,
  username: '',
  password: '',
  confirmPassword: ''
})

const searchForm = reactive({
  keyword: '',
  departmentId: null as number | null,
  source: ''
})

const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

const userList = ref([])

const userForm = reactive({
  id: 0,
  username: '',
  password: '',
  realName: '',
  email: '',
  phone: '',
  status: 1,
  departmentId: null as number | null,
  positionIds: [] as number[],
  roleIds: [] as number[],
  bio: ''
})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
  ],
  email: [{ required: true, message: '请输入邮箱', trigger: 'blur' }]
}

const resetPasswordRules = {
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (rule: any, value: any, callback: any) => {
        if (value !== resetPasswordForm.password) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

const loadUsers = async () => {
  loading.value = true
  try {
    const res = await getUserList({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword,
      departmentId: searchForm.departmentId,
      source: searchForm.source || undefined
    })
    userList.value = res.list || []
    pagination.total = res.total || 0
  } catch (error) {
  } finally {
    loading.value = false
  }
}

// 加载部门树
const loadDepartmentTree = async () => {
  try {
    const res = await getDepartmentTree()
    departmentTree.value = res || []
  } catch (error) {
  }
}

// 加载角色选项
const loadRoleOptions = async () => {
  try {
    roleOptions.value = []  // 先清空
    const res = await getAllRoles()
    roleOptions.value = res || []
  } catch (error) {
  }
}

// 加载岗位选项
const loadPositionOptions = async () => {
  try {
    positionOptions.value = []  // 先清空
    const res = await getPositionList({ page: 1, pageSize: 1000 })
    const list = res.list || []
    if (list.length > 0) {
    }
    positionOptions.value = list
  } catch (error) {
  }
}

// 岗位选择变化
const handlePositionChange = (value: any) => {
}

// 构建部门路径
const buildDepartmentPath = (node: any, path: string[] = []): string => {
  path.unshift(node.deptName || node.name)
  if (node.parent && departmentTree.value) {
    const findParent = (nodes: any[], id: number): any => {
      for (const n of nodes) {
        if (n.id === id) return n
        if (n.children) {
          const found = findParent(n.children, id)
          if (found) return found
        }
      }
      return null
    }
    const parent = findParent(departmentTree.value, node.parentId)
    if (parent) {
      return buildDepartmentPath(parent, path)
    }
  }
  return path.join(' / ')
}

// 处理部门节点点击
const handleNodeClick = (data: any) => {
  selectedDepartment.value = data
  selectedDepartmentPath.value = buildDepartmentPath(data)
  searchForm.departmentId = data.id
  pagination.page = 1
  loadUsers()
}

// 清除部门选择
const clearDepartmentSelection = () => {
  selectedDepartment.value = null
  selectedDepartmentPath.value = ''
  searchForm.departmentId = null
  treeRef.value?.setCurrentKey(null)
  pagination.page = 1
  loadUsers()
}

const resetSearch = () => {
  searchForm.keyword = ''
  searchForm.source = ''
  pagination.page = 1
  loadUsers()
}

const handleAdd = () => {
  isEdit.value = false
  dialogTitle.value = '新增用户'
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true
  dialogTitle.value = '编辑用户'

  // 调试：打印原始数据
  if (row.roles && Array.isArray(row.roles)) {
    row.roles.forEach((r: any, index: number) => {
    })
  }
  if (row.positions && Array.isArray(row.positions)) {
    row.positions.forEach((p: any, index: number) => {
    })
  }

  // 正确处理ID字段，兼容大小写
  userForm.id = Number(row.ID || row.id)
  userForm.username = row.username
  userForm.realName = row.realName || ''
  userForm.email = row.email || ''
  userForm.phone = row.phone || ''
  userForm.status = row.status ?? 1
  userForm.departmentId = row.departmentId ? Number(row.departmentId) : null

  // 处理岗位ID
  if (row.positionIds && Array.isArray(row.positionIds)) {
    userForm.positionIds = row.positionIds.map((id: any) => Number(id))
  } else if (row.positions && Array.isArray(row.positions) && row.positions.length > 0) {
    userForm.positionIds = row.positions.map((p: any) => Number(p.ID || p.id))
  } else {
    userForm.positionIds = []
  }

  // 处理角色ID
  if (row.roleIds && Array.isArray(row.roleIds)) {
    userForm.roleIds = row.roleIds
  } else if (row.roles && Array.isArray(row.roles) && row.roles.length > 0) {
    userForm.roleIds = row.roles.map((r: any) => r.ID || r.id)
  } else {
    userForm.roleIds = []
  }


  userForm.bio = row.bio || ''
  dialogVisible.value = true
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除该用户吗？', '提示', {
      type: 'warning'
    })
    await deleteUser(row.ID || row.id)
    ElMessage.success('删除成功')
    loadUsers()
  } catch (error) {
    if (error !== 'cancel') {
    }
  }
}

const handleResetPassword = (row: any) => {
  resetPasswordForm.userId = row.ID || row.id
  resetPasswordForm.username = row.username
  resetPasswordForm.password = ''
  resetPasswordForm.confirmPassword = ''
  resetPasswordVisible.value = true
}

const handleResetPasswordSubmit = async () => {
  if (!resetPasswordFormRef.value) return

  await resetPasswordFormRef.value.validate(async (valid) => {
    if (valid) {
      resetPasswordLoading.value = true
      try {
        await resetUserPassword(resetPasswordForm.userId, resetPasswordForm.password)
        ElMessage.success('密码重置成功')
        resetPasswordVisible.value = false
      } catch (error) {
      } finally {
        resetPasswordLoading.value = false
      }
    }
  })
}

const handleResetPasswordClose = () => {
  resetPasswordFormRef.value?.resetFields()
  Object.assign(resetPasswordForm, {
    userId: 0,
    username: '',
    password: '',
    confirmPassword: ''
  })
}

const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitLoading.value = true
      try {
        // 保存角色ID和岗位ID，过滤掉null值
        const roleIds = (userForm.roleIds || []).filter((id: any) => id != null)
        const positionIds = (userForm.positionIds || []).filter((id: any) => id != null)

        if (isEdit.value) {
          // 清理userForm中的null值，避免发送到后端
          const userData = {
            ...userForm,
            positionIds: positionIds,
            roleIds: roleIds
          }

          // 更新用户基本信息
          await updateUser(userForm.id, userData)

          // 分配角色（传空数组表示清空角色）
          await assignUserRoles(userForm.id, roleIds)

          // 分配岗位（传空数组表示清空岗位）
          await assignUserPositions(userForm.id, positionIds)

          ElMessage.success('更新成功')
        } else {
          // 创建新用户
          await createUser(userForm)

          // 分配角色
          if (roleIds.length > 0) {
            // 获取刚创建的用户ID，这里需要从响应中获取
            // 暂时先跳过，需要后端返回创建的用户ID
          }

          ElMessage.success('创建成功')
        }

        dialogVisible.value = false
        loadUsers()
      } catch (error) {
        ElMessage.error('操作失败')
      } finally {
        submitLoading.value = false
      }
    }
  })
}

const handleDialogClose = () => {
  formRef.value?.resetFields()
  Object.assign(userForm, {
    id: 0,
    username: '',
    password: '',
    realName: '',
    email: '',
    phone: '',
    status: 1,
    departmentId: null,
    positionIds: [],
    roleIds: [],
    bio: ''
  })
}

// 监控岗位选项和选择的变化
watch(positionOptions, (newVal) => {
}, { deep: true })

watch(() => userForm.positionIds, (newVal) => {
})

onMounted(() => {
  loadDepartmentTree()
  loadRoleOptions()
  loadPositionOptions()
  loadUsers()
})
</script>

<style scoped>
.users-container {
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

/* 内容区域 */
.content-wrapper {
  display: flex;
  gap: 12px;
  min-height: calc(100vh - 180px);
}

/* 左侧部门树面板 */
.dept-tree-panel {
  width: 240px;
  min-width: 240px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-header {
  padding: 12px 16px;
  font-weight: 600;
  font-size: 14px;
  color: #303133;
  border-bottom: 1px solid #f0f0f0;
  background-color: #fafbfc;
  display: flex;
  align-items: center;
  gap: 8px;
}

.panel-header .el-icon {
  color: #0a466a;
  font-size: 16px;
}

.dept-tree {
  flex: 1;
  padding: 8px;
  overflow-y: auto;
  background-color: #fff;
  font-size: 13px;
}

.dept-tree :deep(.el-tree-node__content) {
  height: 34px;
  border-radius: 0;
}

.dept-tree :deep(.el-tree-node__content:hover) {
  background-color: #f5f7fa;
}

.custom-tree-node {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding-right: 6px;
}

.node-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.node-count {
  color: #909399;
  font-size: 12px;
}

/* 右侧用户列表面板 */
.user-list-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.selected-dept-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  margin-bottom: 12px;
  background-color: #fff;
  border-left: 3px solid #409eff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  font-size: 13px;
}

.dept-path-text {
  display: flex;
  align-items: center;
  gap: 6px;
}

.dept-path-text .label {
  color: #606266;
  font-weight: 500;
}

.dept-path-text .path {
  color: #409eff;
  font-weight: 500;
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

/* 用户名单元格 */
.user-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-name {
  font-weight: 500;
}

.description-text {
  color: #606266;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
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

.action-edit:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-password:hover {
  background-color: #f0f9eb;
  color: #67C23A;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

/* 分页器 */
.pagination-container {
  padding: 12px 16px;
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid #f0f0f0;
}

/* 标签样式 */
:deep(.el-tag) {
  border-radius: 0;
  padding: 4px 10px;
  font-weight: 500;
}

/* 输入框样式 */
:deep(.el-input__wrapper),
:deep(.el-textarea__inner) {
  border-radius: 0;
}

:deep(.el-select .el-input__wrapper) {
  border-radius: 0;
}

/* 用户对话框样式 */
:deep(.user-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.user-dialog .el-dialog__body) {
  padding: 24px;
  max-height: 60vh;
  overflow-y: auto;
}

:deep(.user-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

.user-form {
  width: 100%;
}

.user-form :deep(.el-form-item) {
  margin-bottom: 14px;
}

.user-form :deep(.el-form-item__label) {
  font-size: 13px;
}

.form-section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 14px;
  padding-bottom: 8px;
  border-bottom: 1px solid #f0f0f0;
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}

.form-section-title .el-icon {
  font-size: 15px;
  color: #409eff;
}

.form-section-title + .el-form {
  margin-top: 12px;
}

/* 树形选择器节点样式 */
.tree-node-count {
  margin-left: 6px;
  color: #909399;
  font-size: 11px;
}

/* 角色选择器选项样式 */
.role-code {
  margin-left: 10px;
  color: #909399;
  font-size: 11px;
}

/* 对话框底部 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.dialog-footer .el-button {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* 输入框图标样式 */
.user-form :deep(.el-input__prefix) {
  color: #a8abb2;
}

/* 响应式对话框 */
:deep(.responsive-dialog) {
  max-width: 1200px;
  min-width: 400px;
}

@media (max-width: 768px) {
  :deep(.responsive-dialog .el-dialog) {
    width: 95% !important;
    max-width: none;
    min-width: auto;
  }
}
</style>
