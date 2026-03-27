<template>
  <div class="authorization-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Lock /></el-icon>
        </div>
        <div>
          <h2 class="page-title">资产授权</h2>
          <p class="page-subtitle">配置用户或部门对资产分组、主机和网络设备的访问权限</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          创建授权规则
        </el-button>
      </div>
    </div>

    <!-- 主体区域：左树 + 右表 -->
    <div class="main-content">
      <!-- 左侧：资产树 -->
      <div class="tree-panel">
        <div class="tree-header">
          <span class="tree-title">资产树</span>
          <el-button link size="small" @click="loadAssetTree" style="color: #909399;">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
        <div class="tree-search">
          <el-input
            v-model="treeFilter"
            placeholder="搜索..."
            clearable
            size="small"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </div>
        <div class="tree-body">
          <el-tree
            ref="treeRef"
            :data="assetTreeData"
            :props="{ label: 'label', children: 'children' }"
            node-key="id"
            highlight-current
            default-expand-all
            :expand-on-click-node="false"
            :filter-node-method="filterNode"
            @node-click="handleNodeClick"
          >
            <template #default="{ data }">
              <span class="tree-node-label">
                <el-icon v-if="data.nodeType === 'root'" class="tree-icon"><FolderOpened /></el-icon>
                <el-icon v-else-if="data.nodeType === 'group'" class="tree-icon"><Folder /></el-icon>
                <el-icon v-else-if="data.nodeType === 'host'" class="tree-icon tree-icon-host"><Monitor /></el-icon>
                <el-icon v-else-if="data.nodeType === 'device'" class="tree-icon tree-icon-device"><SetUp /></el-icon>
                <span>{{ data.label }}</span>
                <span v-if="data.ip" class="tree-node-ip">({{ data.ip }})</span>
              </span>
            </template>
          </el-tree>
        </div>
      </div>

      <!-- 右侧：授权规则列表 -->
      <div class="table-panel">
        <!-- 当前选中提示 -->
        <div class="selected-hint" v-if="selectedNode">
          <el-tag v-if="selectedNode.nodeType === 'root'" type="info">全部规则</el-tag>
          <el-tag v-else-if="selectedNode.nodeType === 'group'" type="success" class="hint-tag">
            <span class="hint-tag-content">
              <el-icon><Folder /></el-icon>
              <span>分组: {{ selectedNode.label }}</span>
            </span>
          </el-tag>
          <el-tag v-else-if="selectedNode.nodeType === 'host'" type="primary" class="hint-tag">
            <span class="hint-tag-content">
              <el-icon><Monitor /></el-icon>
              <span>主机: {{ selectedNode.label }} {{ selectedNode.ip ? `(${selectedNode.ip})` : '' }}</span>
            </span>
          </el-tag>
          <el-tag v-else-if="selectedNode.nodeType === 'device'" type="warning" class="hint-tag">
            <span class="hint-tag-content">
              <el-icon><SetUp /></el-icon>
              <span>设备: {{ selectedNode.label }} {{ selectedNode.ip ? `(${selectedNode.ip})` : '' }}</span>
            </span>
          </el-tag>
          <el-button v-if="selectedNode.nodeType !== 'root'" link size="small" @click="handleAddForSelected" class="add-for-selected-btn">
            <el-icon style="margin-right: 4px;"><Plus /></el-icon>
            为此{{ selectedNode.nodeType === 'group' ? '分组' : '资产' }}创建授权
          </el-button>
        </div>

        <!-- 搜索栏 -->
        <div class="search-bar">
          <el-input
            v-model="keyword"
            placeholder="搜索授权规则名称..."
            clearable
            class="search-input"
            @input="handleSearch"
          >
            <template #prefix>
              <el-icon class="search-icon"><Search /></el-icon>
            </template>
          </el-input>
        </div>

        <!-- 表格 -->
        <el-table
          :data="authorizationList"
          v-loading="loading"
          class="modern-table"
          :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        >
          <el-table-column prop="name" label="规则名称" min-width="130">
            <template #default="{ row }">
              <span class="rule-name">{{ row.name }}</span>
            </template>
          </el-table-column>

          <el-table-column label="授权对象" min-width="130">
            <template #default="{ row }">
              <div class="auth-targets">
                <template v-if="row.userNames && row.userNames.length > 0">
                  <el-tag v-for="name in row.userNames" :key="'u_'+name" size="small" type="primary" style="margin: 2px;">
                    {{ name }}
                  </el-tag>
                </template>
                <template v-if="row.departmentNames && row.departmentNames.length > 0">
                  <el-tag v-for="name in row.departmentNames" :key="'d_'+name" size="small" type="success" style="margin: 2px;">
                    {{ name }}
                  </el-tag>
                </template>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="资产类型" width="110" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.assetType === 'network_device'" type="warning" size="small">网络设备</el-tag>
              <el-tag v-else type="primary" size="small">主机</el-tag>
            </template>
          </el-table-column>

          <el-table-column label="资产范围" min-width="200">
            <template #default="{ row }">
              <div>
                <el-tag type="info" size="small">{{ row.assetGroupName }}</el-tag>
                <span v-if="row.isAllAssets" class="scope-text"> / 全部</span>
                <template v-else>
                  <span class="scope-text"> / 指定 {{ row.assetIds?.length || 0 }} 台</span>
                  <div v-if="row.assetNames && row.assetNames.length > 0" class="asset-name-list">
                    <el-tag v-for="name in row.assetNames.slice(0, 3)" :key="name" size="small" type="" style="margin: 2px;">{{ name }}</el-tag>
                    <span v-if="row.assetNames.length > 3" class="scope-text">...等</span>
                  </div>
                </template>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="操作权限" min-width="200">
            <template #default="{ row }">
              <div class="permission-tags">
                <el-tag v-if="(row.permissions & 1) > 0" size="small" type="success">查看</el-tag>
                <el-tag v-if="(row.permissions & 2) > 0" size="small" type="primary">编辑</el-tag>
                <el-tag v-if="(row.permissions & 4) > 0" size="small" type="danger">删除</el-tag>
                <el-tag v-if="(row.permissions & 8) > 0" size="small" type="warning">终端</el-tag>
                <el-tag v-if="(row.permissions & 16) > 0" size="small" type="info">文件</el-tag>
                <el-tag v-if="(row.permissions & 32) > 0" size="small">采集</el-tag>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="有效期" min-width="160">
            <template #default="{ row }">
              <div v-if="row.startDate || row.expireDate" class="validity-cell">
                <span>{{ row.startDate ? formatDate(row.startDate) : '-' }}</span>
                <span> ~ </span>
                <span>{{ row.expireDate ? formatDate(row.expireDate) : '永久' }}</span>
              </div>
              <span v-else class="validity-permanent">永久有效</span>
            </template>
          </el-table-column>

          <el-table-column label="状态" width="70" align="center">
            <template #default="{ row }">
              <el-tag :type="row.isActive ? 'success' : 'info'" size="small">
                {{ row.isActive ? '启用' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column label="操作" width="100" align="center" fixed="right">
            <template #default="{ row }">
              <div class="action-buttons">
                <el-tooltip content="编辑" placement="top">
                  <el-button link class="action-btn action-edit" @click="handleEdit(row)">
                    <el-icon><Edit /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top">
                  <el-button link class="action-btn action-delete" @click="handleDeleteClick(row)">
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
            v-model:current-page="page"
            v-model:page-size="pageSize"
            :page-sizes="[10, 20, 50, 100]"
            :total="total"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleSizeChange"
            @current-change="handlePageChange"
          />
        </div>
      </div>
    </div>

    <!-- 创建/编辑授权对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '创建授权规则' : '编辑授权规则'"
      width="680px"
      class="auth-dialog responsive-dialog"
      :close-on-click-modal="false"
      @close="handleDialogClose"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-width="100px"
        label-position="top"
      >
        <el-form-item label="规则名称" prop="name">
          <el-input v-model="form.name" placeholder="例如：运维组-测试环境主机查看" />
        </el-form-item>

        <el-divider content-position="left">授权对象</el-divider>

        <el-form-item label="用户">
          <el-select
            v-model="form.userIds"
            multiple
            filterable
            placeholder="选择用户（可多选）"
            style="width: 100%"
          >
            <el-option
              v-for="u in userList"
              :key="u.id"
              :label="u.realName ? `${u.realName} (${u.username})` : u.username"
              :value="u.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="部门">
          <el-tree-select
            v-model="form.departmentIds"
            :data="departmentTree"
            multiple
            check-strictly
            :render-after-expand="false"
            placeholder="选择部门（可多选）"
            style="width: 100%"
          />
        </el-form-item>

        <el-divider content-position="left">资产范围</el-divider>

        <el-form-item label="资产类型">
          <el-radio-group v-model="form.assetType" @change="onAssetTypeChange">
            <el-radio-button value="host">主机</el-radio-button>
            <el-radio-button value="network_device">网络设备</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="资产分组" prop="assetGroupId">
          <el-tree-select
            v-model="form.assetGroupId"
            :data="groupTreeData"
            check-strictly
            :render-after-expand="false"
            placeholder="选择资产分组"
            style="width: 100%"
            @change="onGroupChange"
          />
        </el-form-item>

        <el-form-item label="资产范围">
          <el-radio-group v-model="assetScope" @change="onAssetScopeChange">
            <el-radio-button value="all">{{ form.assetType === 'network_device' ? '全部设备' : '全部主机' }}</el-radio-button>
            <el-radio-button value="specific">{{ form.assetType === 'network_device' ? '指定设备' : '指定主机' }}</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="assetScope === 'specific'" :label="form.assetType === 'network_device' ? '选择设备' : '选择主机'">
          <el-select
            v-model="form.assetIds"
            multiple
            filterable
            :placeholder="form.assetType === 'network_device' ? '选择设备（可多选）' : '选择主机（可多选）'"
            style="width: 100%"
            :loading="loadingAssets"
          >
            <el-option
              v-for="item in assetList"
              :key="item.id"
              :label="`${item.name} (${item.ip})`"
              :value="item.id"
            />
          </el-select>
        </el-form-item>

        <el-divider content-position="left">操作权限</el-divider>

        <el-form-item label="权限">
          <el-checkbox-group v-model="form.permissionList">
            <el-checkbox :value="1">查看</el-checkbox>
            <el-checkbox :value="2">编辑</el-checkbox>
            <el-checkbox :value="4">删除</el-checkbox>
            <el-checkbox :value="8">终端</el-checkbox>
            <template v-if="form.assetType !== 'network_device'">
              <el-checkbox :value="16">文件</el-checkbox>
              <el-checkbox :value="32">采集</el-checkbox>
            </template>
          </el-checkbox-group>
        </el-form-item>

        <el-divider content-position="left">有效期 & 其他</el-divider>

        <div style="display: flex; gap: 16px;">
          <el-form-item label="生效时间" style="flex: 1;">
            <el-date-picker
              v-model="form.startDate"
              type="datetime"
              placeholder="不设则立即生效"
              style="width: 100%"
              value-format="YYYY-MM-DDTHH:mm:ssZ"
            />
          </el-form-item>
          <el-form-item label="失效时间" style="flex: 1;">
            <el-date-picker
              v-model="form.expireDate"
              type="datetime"
              placeholder="不设则永久有效"
              style="width: 100%"
              value-format="YYYY-MM-DDTHH:mm:ssZ"
            />
          </el-form-item>
        </div>

        <el-form-item label="是否启用">
          <el-switch v-model="form.isActive" />
        </el-form-item>

        <el-form-item label="备注">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选备注说明" />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleSubmit" :loading="submitting">
            {{ dialogMode === 'create' ? '创建' : '保存' }}
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Delete, Search, Lock, Edit, Folder, FolderOpened, Monitor, SetUp, Refresh } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import {
  getAssetTree,
  getAssetAuthorizations,
  createAssetAuthorization,
  updateAssetAuthorization,
  deleteAssetAuthorization,
  getAssetAuthorizationDetail
} from '@/api/assetPermission'
import { getGroupTree } from '@/api/assetGroup'
import { getHostList, getNetworkDeviceList } from '@/api/host'
import { getUserList } from '@/api/user'
import { getDepartmentTree } from '@/api/department'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const isAdmin = computed(() => {
  const roles = userStore.userInfo?.roles || []
  return roles.some((r: any) => r.code === 'admin')
})

// ==================== 资产树 ====================
const treeRef = ref()
const assetTreeData = ref<any[]>([])
const treeFilter = ref('')
const selectedNode = ref<any>(null)

watch(treeFilter, (val) => {
  treeRef.value?.filter(val)
})

const filterNode = (value: string, data: any) => {
  if (!value) return true
  const search = value.toLowerCase()
  return data.label?.toLowerCase().includes(search) || data.ip?.toLowerCase().includes(search)
}

const loadAssetTree = async () => {
  try {
    const data = await getAssetTree()
    assetTreeData.value = data || []
    nextTick(() => {
      if (assetTreeData.value.length > 0) {
        treeRef.value?.setCurrentKey(0)
        selectedNode.value = { id: 0, nodeType: 'root', label: '全部' }
      }
    })
  } catch {
    assetTreeData.value = []
  }
}

const handleNodeClick = (data: any) => {
  selectedNode.value = data
  page.value = 1
  loadList()
}

// ==================== 授权列表 ====================
const loading = ref(false)
const authorizationList = ref<any[]>([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const keyword = ref('')

const loadList = async () => {
  loading.value = true
  try {
    const params: any = { page: page.value, pageSize: pageSize.value }
    if (selectedNode.value && selectedNode.value.nodeType === 'group') {
      params.assetGroupId = selectedNode.value.id
    }
    if (keyword.value) {
      params.keyword = keyword.value
    }
    const response = await getAssetAuthorizations(params)
    let list = response.list || []

    // 如果选中了具体资产（host 或 device），在前端过滤出包含该资产的规则
    if (selectedNode.value && (selectedNode.value.nodeType === 'host' || selectedNode.value.nodeType === 'device')) {
      const nodeId = selectedNode.value.id as string
      const parts = nodeId.split('_')
      const assetId = parseInt(parts[1])
      const groupId = selectedNode.value.groupId
      const isDevice = selectedNode.value.nodeType === 'device'

      list = list.filter((rule: any) => {
        // 匹配资产类型
        const typeMatch = isDevice ? rule.assetType === 'network_device' : rule.assetType !== 'network_device'
        if (!typeMatch) return false
        // 匹配分组
        if (rule.assetGroupId !== groupId) return false
        // 全部资产或包含此资产
        if (rule.isAllAssets) return true
        if (rule.assetIds && rule.assetIds.includes(assetId)) return true
        return false
      })
    }

    authorizationList.value = list
    total.value = response.total || 0
  } catch (error: any) {
    ElMessage.error('加载授权列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const handleSearch = () => { page.value = 1; loadList() }
const handleSizeChange = () => { page.value = 1; loadList() }
const handlePageChange = () => { loadList() }

const formatDate = (dateStr: string) => {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString('zh-CN')
}

// ==================== 创建/编辑对话框 ====================
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const submitting = ref(false)
const formRef = ref<FormInstance>()
const editingId = ref<number | null>(null)

const form = reactive({
  name: '',
  userIds: [] as number[],
  departmentIds: [] as number[],
  assetGroupId: null as number | null,
  assetType: 'host' as string,
  assetIds: [] as number[],
  permissionList: [1] as number[],
  startDate: null as string | null,
  expireDate: null as string | null,
  isActive: true,
  description: ''
})

const assetScope = ref('all')
const loadingAssets = ref(false)
const assetList = ref<any[]>([])
const groupTreeData = ref<any[]>([])
const userList = ref<any[]>([])
const departmentTree = ref<any[]>([])

const formRules: FormRules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  assetGroupId: [{ required: true, message: '请选择资产分组', trigger: 'change' }]
}

const convertGroupTree = (nodes: any[]): any[] => {
  return nodes.map((node: any) => ({
    value: node.id,
    label: node.name,
    children: node.children ? convertGroupTree(node.children) : undefined
  }))
}

const convertDeptTree = (nodes: any[]): any[] => {
  return nodes.map((node: any) => ({
    value: node.id || node.ID,
    label: node.deptName || node.name,
    children: node.children ? convertDeptTree(node.children) : undefined
  }))
}

const loadUsers = async () => {
  try {
    const response = await getUserList({ page: 1, pageSize: 1000 })
    userList.value = (response.list || []).map((u: any) => ({
      id: u.ID || u.id,
      username: u.username,
      realName: u.realName
    }))
  } catch { userList.value = [] }
}

const loadDepartments = async () => {
  try {
    const data = await getDepartmentTree()
    departmentTree.value = convertDeptTree(data || [])
  } catch { departmentTree.value = [] }
}

const loadGroupTree = async () => {
  try {
    const category = form.assetType === 'network_device' ? 'network' : 'host'
    const data = await getGroupTree(category)
    groupTreeData.value = convertGroupTree(data || [])
  } catch { groupTreeData.value = [] }
}

const loadAssets = async () => {
  if (!form.assetGroupId) return
  loadingAssets.value = true
  try {
    if (form.assetType === 'network_device') {
      const response = await getNetworkDeviceList({ page: 1, pageSize: 1000, groupId: form.assetGroupId })
      assetList.value = (response.list || []).map((item: any) => ({ id: item.id, name: item.name, ip: item.ip }))
    } else {
      const response = await getHostList({ page: 1, pageSize: 1000, groupId: form.assetGroupId })
      assetList.value = (response.list || []).map((item: any) => ({ id: item.id, name: item.name, ip: item.ip }))
    }
  } catch { assetList.value = [] }
  finally { loadingAssets.value = false }
}

const onAssetTypeChange = () => {
  form.assetGroupId = null
  form.assetIds = []
  assetList.value = []
  assetScope.value = 'all'
  if (form.assetType === 'network_device') {
    form.permissionList = form.permissionList.filter(p => p <= 8)
  }
  loadGroupTree()
}

const onGroupChange = () => {
  form.assetIds = []
  assetList.value = []
  if (form.assetGroupId) loadAssets()
}

const onAssetScopeChange = () => {
  if (assetScope.value === 'specific' && form.assetGroupId) {
    loadAssets()
  } else {
    form.assetIds = []
  }
}

// 普通添加
const handleAdd = () => {
  if (!isAdmin.value) { ElMessage.error('无权限，请联系管理员操作'); return }
  dialogMode.value = 'create'
  editingId.value = null
  resetForm()
  loadGroupTree()
  dialogVisible.value = true
}

// 根据树选中节点添加 - 自动预填
const handleAddForSelected = async () => {
  if (!isAdmin.value) { ElMessage.error('无权限，请联系管理员操作'); return }
  dialogMode.value = 'create'
  editingId.value = null
  resetForm()

  const node = selectedNode.value
  if (!node) return

  if (node.nodeType === 'group') {
    // 选中分组 → 预填分组，范围=全部
    form.assetGroupId = node.id
    assetScope.value = 'all'
    form.name = `${node.label} - `
    await loadGroupTree()
  } else if (node.nodeType === 'host') {
    // 选中主机 → 预填分组+指定主机
    form.assetType = 'host'
    form.assetGroupId = node.groupId
    assetScope.value = 'specific'
    const realId = parseInt(String(node.id).replace('host_', ''))
    form.assetIds = [realId]
    form.name = `${node.label} - `
    await loadGroupTree()
    await loadAssets()
  } else if (node.nodeType === 'device') {
    // 选中设备 → 预填分组+指定设备
    form.assetType = 'network_device'
    form.assetGroupId = node.groupId
    assetScope.value = 'specific'
    const realId = parseInt(String(node.id).replace('device_', ''))
    form.assetIds = [realId]
    form.name = `${node.label} - `
    await loadGroupTree()
    await loadAssets()
  }

  dialogVisible.value = true
}

const handleEdit = async (row: any) => {
  if (!isAdmin.value) { ElMessage.error('无权限，请联系管理员操作'); return }
  try {
    dialogMode.value = 'edit'
    editingId.value = row.id
    const detail = await getAssetAuthorizationDetail(row.id)

    form.name = detail.name || ''
    form.userIds = detail.userIds || []
    form.departmentIds = detail.departmentIds || []
    form.assetGroupId = detail.assetGroupId || null
    form.assetType = detail.assetType || 'host'
    form.assetIds = detail.assetIds || []
    form.startDate = detail.startDate || null
    form.expireDate = detail.expireDate || null
    form.isActive = detail.isActive !== false
    form.description = detail.description || ''

    form.permissionList = []
    const p = detail.permissions || 0
    if ((p & 1) > 0) form.permissionList.push(1)
    if ((p & 2) > 0) form.permissionList.push(2)
    if ((p & 4) > 0) form.permissionList.push(4)
    if ((p & 8) > 0) form.permissionList.push(8)
    if ((p & 16) > 0) form.permissionList.push(16)
    if ((p & 32) > 0) form.permissionList.push(32)

    assetScope.value = (!detail.assetIds || detail.assetIds.length === 0) ? 'all' : 'specific'

    await loadGroupTree()
    if (assetScope.value === 'specific' && form.assetGroupId) {
      await loadAssets()
    }

    dialogVisible.value = true
  } catch (error: any) {
    ElMessage.error('加载规则详情失败: ' + (error.message || '未知错误'))
  }
}

const resetForm = () => {
  form.name = ''
  form.userIds = []
  form.departmentIds = []
  form.assetGroupId = null
  form.assetType = 'host'
  form.assetIds = []
  form.permissionList = [1]
  form.startDate = null
  form.expireDate = null
  form.isActive = true
  form.description = ''
  assetScope.value = 'all'
  assetList.value = []
  formRef.value?.clearValidate()
}

const handleDialogClose = () => { resetForm() }

const handleSubmit = async () => {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  if (form.userIds.length === 0 && form.departmentIds.length === 0) {
    ElMessage.warning('请至少选择一个用户或部门')
    return
  }
  if (form.permissionList.length === 0) {
    ElMessage.warning('请至少选择一项操作权限')
    return
  }

  submitting.value = true
  try {
    const permBitmask = form.permissionList.reduce((acc, val) => acc | val, 0)
    const payload: any = {
      name: form.name,
      userIds: form.userIds,
      departmentIds: form.departmentIds,
      assetGroupId: form.assetGroupId,
      assetType: form.assetType,
      assetIds: assetScope.value === 'all' ? [] : form.assetIds,
      permissions: permBitmask,
      startDate: form.startDate || undefined,
      expireDate: form.expireDate || undefined,
      isActive: form.isActive,
      description: form.description
    }

    if (dialogMode.value === 'create') {
      await createAssetAuthorization(payload)
      ElMessage.success('创建成功')
    } else if (editingId.value) {
      await updateAssetAuthorization(editingId.value, payload)
      ElMessage.success('更新成功')
    }

    dialogVisible.value = false
    loadList()
  } catch (error: any) {
    ElMessage.error((dialogMode.value === 'create' ? '创建' : '更新') + '失败: ' + (error.message || '未知错误'))
  } finally { submitting.value = false }
}

// ==================== 删除 ====================
const handleDeleteClick = (row: any) => {
  if (!isAdmin.value) { ElMessage.error('无权限，请联系管理员操作'); return }
  ElMessageBox.confirm(`确定删除授权规则 "${row.name}" 吗？`, '提示', {
    confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning'
  }).then(async () => {
    try {
      await deleteAssetAuthorization(row.id)
      ElMessage.success('删除成功')
      loadList()
    } catch (error: any) {
      ElMessage.error('删除失败: ' + (error.message || '未知错误'))
    }
  }).catch(() => {})
}

// ==================== 初始化 ====================
onMounted(() => {
  loadAssetTree()
  loadList()
  loadUsers()
  loadDepartments()
})
</script>

<style scoped>
.authorization-container {
  padding: 0;
  background-color: transparent;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  padding: 16px 20px;
  background: #fff;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group { display: flex; align-items: flex-start; gap: 16px; }
.page-title-icon {
  width: 48px; height: 48px; background: #0a466a;
  display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 22px; flex-shrink: 0;
}
.page-title { margin: 0; font-size: 20px;  color: #303133; line-height: 1.3; }
.page-subtitle { margin: 4px 0 0 0; font-size: 13px; color: #909399; }

/* 主体 */
.main-content { display: flex; gap: 12px; min-height: calc(100vh - 200px); }

/* 左侧树 */
.tree-panel {
  width: 280px; min-width: 240px; background: #fff;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex; flex-direction: column; flex-shrink: 0;
}
.tree-header {
  padding: 12px 16px; border-bottom: 1px solid #f0f0f0;
  display: flex; justify-content: space-between; align-items: center;
}
.tree-title { font-size: 14px;  color: #303133; }
.tree-search { padding: 8px 12px; border-bottom: 1px solid #f0f0f0; }
.tree-search :deep(.el-input__wrapper) { border-radius: 0; }
.tree-body { flex: 1; overflow-y: auto; padding: 4px 0; }

.tree-node-label { display: flex; align-items: center; font-size: 13px; gap: 2px; }
.tree-icon { margin-right: 2px; font-size: 15px; color: #909399; }
.tree-icon-host { color: #409eff; }
.tree-icon-device { color: #e6a23c; }
.tree-node-ip { color: #909399; font-size: 11px; margin-left: 2px; }

:deep(.el-tree-node__content) { height: 32px; padding: 0 8px; }
:deep(.el-tree-node.is-current > .el-tree-node__content) {
  background-color: #e6f7ff; color: #0a466a;
}

/* 右侧 */
.table-panel {
  flex: 1; background: #fff; box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex; flex-direction: column; min-width: 0;
}

.selected-hint {
  padding: 10px 20px; border-bottom: 1px solid #f0f0f0;
  display: flex; align-items: center; gap: 10px; background: #fafbfc;
}
.selected-hint :deep(.el-tag) {
  display: inline-flex; align-items: center;
}
.hint-tag :deep(.el-tag__content) {
  display: inline-flex; align-items: center;
}
.hint-tag-content {
  display: inline-flex; align-items: center; gap: 4px; line-height: 1;
}
.hint-tag-content .el-icon {
  font-size: 14px; flex-shrink: 0; display: flex; align-items: center;
}
.add-for-selected-btn { font-size: 13px; color: #0a466a; }
.add-for-selected-btn:hover { color: #1890ff; }

.search-bar { padding: 10px 20px; border-bottom: 1px solid #f0f0f0; }
.search-input { width: 300px; }
.search-bar :deep(.el-input__wrapper) { border-radius: 0; border: 1px solid #dcdfe6; box-shadow: 0 2px 4px rgba(0,0,0,0.08); }

.rule-name {  color: #303133; }
.auth-targets { display: flex; flex-wrap: wrap; gap: 2px; }
.scope-text { font-size: 12px; color: #909399; margin-left: 4px; }
.asset-name-list { margin-top: 4px; }
.validity-cell { font-size: 12px; color: #606266; }
.validity-permanent { font-size: 12px; color: #67c23a; }
.permission-tags { display: flex; flex-wrap: wrap; gap: 4px; }

.action-buttons { display: flex; gap: 6px; align-items: center; }
.action-btn { width: 30px; height: 30px; display: flex; align-items: center; justify-content: center; transition: all 0.2s; }
.action-btn :deep(.el-icon) { font-size: 16px; }
.action-edit:hover { background-color: #e6f7ff; color: #1890ff; }
.action-delete:hover { background-color: #fee; color: #f56c6c; }

.pagination-container { padding: 12px 20px; border-top: 1px solid #f0f0f0; display: flex; justify-content: flex-end; }

/* 对话框 */
.dialog-footer { display: flex; justify-content: flex-end; gap: 12px; }
:deep(.auth-dialog) { border-radius: 0; }
:deep(.auth-dialog .el-dialog__header) { padding: 20px 24px 16px; border-bottom: 1px solid #f0f0f0; }
:deep(.auth-dialog .el-dialog__body) { padding: 24px; max-height: 70vh; overflow-y: auto; }
:deep(.auth-dialog .el-dialog__footer) { padding: 16px 24px; border-top: 1px solid #f0f0f0; }
:deep(.el-divider__text) { font-size: 13px;  color: #606266; }
:deep(.el-tag) { border-radius: 0; padding: 4px 10px;  }
.modern-table { width: 100%; flex: 1; }
:deep(.modern-table .el-table__header th:first-child .cell),
:deep(.modern-table .el-table__body td:first-child .cell) { padding-left: 32px; }

@media (max-width: 1024px) {
  .main-content { flex-direction: column; }
  .tree-panel { width: 100%; min-height: 200px; }
  .search-input { width: 100%; }
}
@media (max-width: 768px) {
  :deep(.responsive-dialog) { width: 95% !important; }
}
</style>
