<template>
  <div class="role-detail" v-loading="loading">
    <!-- 基本信息 -->
    <div class="section">
      <h3 class="section-title">基本信息</h3>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="角色名称">
          <el-tag type="primary" size="large">{{ role.name }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="命名空间">
          <el-tag v-if="role.namespace">{{ role.namespace }}</el-tag>
          <el-tag v-else type="success">集群级别</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">
          {{ role.age }}
        </el-descriptions-item>
        <el-descriptions-item label="标签" v-if="Object.keys(role.labels || {}).length > 0">
          <el-tag
            v-for="(value, key) in role.labels"
            :key="key"
            size="small"
            style="margin-right: 4px;"
          >
            {{ key }}: {{ value }}
          </el-tag>
        </el-descriptions-item>
      </el-descriptions>
    </div>

    <!-- 权限规则 -->
    <div class="section">
      <h3 class="section-title">权限规则</h3>
      <el-tree
        :data="permissionTree"
        :props="treeProps"
        node-key="id"
        :default-expand-all="false"
        :expand-on-click-node="true"
      >
        <template #default="{ node, data }">
          <span class="tree-node">
            <el-icon v-if="data.type === 'apiGroup'" color="#409EFF"><Folder /></el-icon>
            <el-icon v-else-if="data.type === 'resource'" color="#67C23A"><Document /></el-icon>
            <el-icon v-else-if="data.type === 'verb'" color="#E6A23C"><Operation /></el-icon>
            <span class="node-label">{{ data.label }}</span>
            <el-tag v-if="data.type === 'verb'" size="small" type="info">{{ data.value }}</el-tag>
          </span>
        </template>
      </el-tree>
    </div>

    <!-- 绑定的平台用户 -->
    <div class="section">
      <h3 class="section-title">
        绑定的平台用户
        <el-button type="primary" size="small" @click="showBindDialog = true" style="margin-left: 16px;">
          <el-icon><Plus /></el-icon>
          绑定用户
        </el-button>
      </h3>
      <div class="user-bindings">
        <el-table :data="boundUsers" border stripe>
          <el-table-column prop="username" label="用户名" />
          <el-table-column prop="realName" label="姓名" />
          <el-table-column prop="boundAt" label="绑定时间">
            <template #default="{ row }">
              {{ formatDate(row.boundAt) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120">
            <template #default="{ row }">
              <el-button link type="danger" @click="handleUnbind(row)">
                <el-icon><Delete /></el-icon>
                解绑
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 绑定用户对话框 -->
    <el-dialog
      v-model="showBindDialog"
      title="绑定用户到角色"
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form :model="bindForm" label-width="100px">
        <el-form-item label="搜索用户">
          <el-input
            v-model="userSearchKeyword"
            placeholder="输入用户名/姓名/邮箱搜索"
            clearable
            @clear="searchUsers"
            @keyup.enter="searchUsers"
          >
            <template #append>
              <el-button @click="searchUsers">
                <el-icon><Search /></el-icon>
              </el-button>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="选择用户">
          <el-select
            v-model="bindForm.userId"
            placeholder="请选择用户"
            filterable
            style="width: 100%"
          >
            <el-option
              v-for="user in availableUsers"
              :key="user.id"
              :label="`${user.username} (${user.realName})`"
              :value="user.id"
            >
              <div style="display: flex; justify-content: space-between;">
                <span>{{ user.username }}</span>
                <span style="color: #8492a6; font-size: 12px;">{{ user.realName }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="showBindDialog = false">取消</el-button>
        <el-button type="primary" @click="handleBind" :loading="bindLoading">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Folder, Document, Operation, Delete, Plus, Search } from '@element-plus/icons-vue'
import { getRoleDetail, bindUserToRole, unbindUserFromRole, getAvailableUsers, type BoundUser, type AvailableUser } from '@/api/kubernetes'

interface Role {
  name: string
  namespace?: string
  labels: Record<string, string>
  age: string
  rules: any[]
}

interface Props {
  clusterId: number
  role: Role
}

const props = defineProps<Props>()

const emit = defineEmits(['close'])

const loading = ref(false)
const permissionTree = ref<any[]>([])
const boundUsers = ref<BoundUser[]>([])

// 绑定用户对话框
const showBindDialog = ref(false)
const bindLoading = ref(false)
const userSearchKeyword = ref('')
const availableUsers = ref<AvailableUser[]>([])
const bindForm = ref({
  userId: 0 as number | undefined
})

const treeProps = {
  children: 'children',
  label: 'label'
}

// 计算角色类型
const roleType = computed(() => {
  return !props.role.namespace || props.role.namespace === '' ? 'ClusterRole' : 'Role'
})

// 加载角色详情
const loadRoleDetail = async () => {
  if (!props.clusterId || !props.role) return

  try {
    loading.value = true
    const detail = await getRoleDetail(
      props.clusterId,
      props.role.namespace || '',
      props.role.name
    )

    // 构建权限树
    permissionTree.value = buildPermissionTree(detail.rules || [])

    // 加载绑定的用户列表
    await loadBoundUsers()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '加载角色详情失败')
  } finally {
    loading.value = false
  }
}

// 加载已绑定的用户列表
const loadBoundUsers = async () => {
  try {
    const users = await getRoleBoundUsers(
      props.clusterId,
      props.role.name,
      props.role.namespace || ''
    )
    boundUsers.value = users
  } catch (error: any) {
    ElMessage.error('加载绑定用户失败')
  }
}

// 搜索可用用户
const searchUsers = async () => {
  try {
    const result = await getAvailableUsers(userSearchKeyword.value, 1, 50)
    availableUsers.value = result.list
  } catch (error: any) {
    ElMessage.error('搜索用户失败')
  }
}

// 格式化日期
const formatDate = (dateStr: string) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN')
}

// 构建权限树
const buildPermissionTree = (rules: any[]) => {
  const tree: any[] = []

  rules.forEach((rule, index) => {
    // API Groups - 注意后端返回的是 apiGroups (复数)
    const apiGroups = rule.apiGroups || ['']
    apiGroups.forEach((apiGroup: string, groupIndex: number) => {
      const apiGroupNode = {
        id: `apiGroup-${index}-${groupIndex}`,
        type: 'apiGroup',
        label: apiGroup || 'core',
        children: [] as any[]
      }

      // Resources
      const resources = rule.resources || []
      resources.forEach((resource: string, resIndex: number) => {
        const resourceNode = {
          id: `resource-${index}-${groupIndex}-${resIndex}`,
          type: 'resource',
          label: resource,
          children: [] as any[]
        }

        // Verbs
        const verbs = rule.verbs || ['*']
        verbs.forEach((verb: string, vIndex: number) => {
          const verbLabel = verb === '*' ? '所有操作' : verb
          resourceNode.children.push({
            id: `verb-${index}-${groupIndex}-${resIndex}-${vIndex}`,
            type: 'verb',
            label: '操作',
            value: verbLabel
          })
        })

        apiGroupNode.children.push(resourceNode)
      })

      tree.push(apiGroupNode)
    })
  })

  return tree
}

const handleUnbind = async (user: BoundUser) => {
  try {
    await ElMessageBox.confirm(
      `确定要解绑用户 "${user.username}" 的角色权限吗？`,
      '确认解绑',
      {
        type: 'warning',
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      }
    )

    await unbindUserFromRole({
      clusterId: props.clusterId,
      userId: user.userId,
      roleName: props.role.name,
      roleNamespace: props.role.namespace || ''
    })
    ElMessage.success('解绑成功')
    await loadBoundUsers()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('解绑失败')
    }
  }
}

// 绑定用户
const handleBind = async () => {
  if (!bindForm.value.userId) {
    ElMessage.warning('请选择用户')
    return
  }

  try {
    bindLoading.value = true
    await bindUserToRole({
      clusterId: props.clusterId,
      userId: bindForm.value.userId,
      roleName: props.role.name,
      roleNamespace: props.role.namespace || '',
      roleType: roleType.value
    })

    ElMessage.success('绑定成功')
    showBindDialog.value = false
    bindForm.value.userId = undefined
    userSearchKeyword.value = ''
    availableUsers.value = []

    await loadBoundUsers()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '绑定失败')
  } finally {
    bindLoading.value = false
  }
}

// 监听对话框打开，自动加载用户列表
const handleDialogOpen = () => {
  if (showBindDialog.value && availableUsers.value.length === 0) {
    searchUsers()
  }
}

// 监听对话框显示状态
watch(() => showBindDialog.value, (newVal) => {
  if (newVal) {
    handleDialogOpen()
  }
})

onMounted(() => {
  loadRoleDetail()
})
</script>

<style scoped lang="scss">
.role-detail {
  .section {
    margin-bottom: 30px;

    .section-title {
      font-size: 16px;
      font-weight: 500;
      color: #333;
      margin-bottom: 16px;
      padding-bottom: 8px;
      border-bottom: 2px solid #0f69a6;
    }
  }

  .tree-node {
    display: flex;
    align-items: center;
    gap: 6px;

    .node-label {
      font-size: 14px;
    }
  }

  :deep(.el-tree-node__content) {
    padding: 4px 0;
  }

  :deep(.el-descriptions) {
    margin-top: 16px;
  }
}
</style>
