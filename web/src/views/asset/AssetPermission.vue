<template>
  <div class="permission-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Lock /></el-icon>
        </div>
        <div>
          <h2 class="page-title">权限配置</h2>
          <p class="page-subtitle">配置角色对资产分组、主机和网络设备的访问权限</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          添加权限
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.roleName"
          placeholder="搜索角色名称..."
          clearable
          class="search-input"
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-input
          v-model="searchForm.groupName"
          placeholder="搜索资产分组..."
          clearable
          class="search-input"
          @input="handleSearch"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <div class="search-actions">
        <el-button class="reset-btn" @click="handleReset">
          <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <!-- 表格和分页容器 -->
    <div class="table-wrapper">
      <el-table
        :data="filteredPermissions"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column prop="id" label="ID" width="80" align="center" />

        <el-table-column label="角色" min-width="150">
          <template #default="{ row }">
            <el-tag type="primary">{{ row.roleName }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="资产类型" width="120" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.assetType === 'network_device'" type="warning" size="small">网络设备</el-tag>
            <el-tag v-else type="primary" size="small">主机</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="资产分组" min-width="160">
          <template #default="{ row }">
            <el-tag type="success">{{ row.assetGroupName }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="资产范围" min-width="180">
          <template #default="{ row }">
            <el-tag v-if="!row.hostIds || row.hostIds.length === 0" type="info">
              {{ row.assetType === 'network_device' ? '全部设备' : '全部主机' }}
            </el-tag>
            <div v-else>
              <el-tag type="warning" size="small">
                指定 {{ row.hostIds.length }} 台{{ row.assetType === 'network_device' ? '设备' : '主机' }}
              </el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="操作权限" min-width="200">
          <template #default="{ row }">
            <div class="permission-tags">
              <el-tag v-if="(row.permissions & 1) > 0" size="small" type="success">查看</el-tag>
              <el-tag v-if="(row.permissions & 2) > 0" size="small" type="primary">编辑</el-tag>
              <el-tag v-if="(row.permissions & 4) > 0" size="small" type="danger">删除</el-tag>
              <el-tag v-if="(row.permissions & 8) > 0" size="small" type="warning">
                {{ row.assetType === 'network_device' ? '终端(SSH/Telnet)' : '终端(SSH/RDP)' }}
              </el-tag>
              <el-tag v-if="(row.permissions & 16) > 0" size="small" type="info">文件</el-tag>
              <el-tag v-if="(row.permissions & 32) > 0" size="small">采集</el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="createdAt" label="创建时间" min-width="180">
          <template #default="{ row }">
            {{ formatTime(row.createdAt) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="120" align="center" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑" placement="top">
                <el-button
                  link
                  class="action-btn action-edit"
                  @click="handleEditClick(row)"
                >
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button
                  link
                  class="action-btn action-delete"
                  @click="handleDeleteClick(row)"
                >
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

    <!-- 批量添加权限对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title="批量添加权限"
      width="80%"
      class="permission-dialog batch-dialog responsive-dialog"
      :close-on-click-modal="false"
      @close="handleDialogClose"
    >
      <!-- 角色选择 -->
      <div class="batch-role-section">
        <label class="batch-label">选择角色</label>
        <el-select
          v-model="batchRoleId"
          placeholder="请选择角色"
          style="width: 300px"
          clearable
          filterable
        >
          <el-option
            v-for="role in roleList"
            :key="role.id"
            :label="role.name"
            :value="role.id"
          />
        </el-select>
      </div>

      <!-- 规则列表 -->
      <div class="batch-rules-section">
        <div class="batch-rules-header">
          <span class="batch-label">权限规则</span>
          <el-button size="small" @click="addRuleRow" :disabled="!batchRoleId">
            <el-icon style="margin-right: 4px;"><Plus /></el-icon>
            添加规则
          </el-button>
        </div>

        <div v-if="ruleRows.length === 0" class="batch-empty">
          <span>暂无规则，请点击"添加规则"开始配置</span>
        </div>

        <div v-for="(rule, index) in ruleRows" :key="rule.id" class="rule-row">
          <div class="rule-row-header">
            <span class="rule-row-index">规则 {{ index + 1 }}</span>
            <el-button link type="danger" @click="removeRuleRow(index)" class="rule-remove-btn">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>

          <div class="rule-row-body">
            <!-- 资产类型 -->
            <div class="rule-field">
              <label class="rule-field-label">资产类型</label>
              <el-radio-group v-model="rule.assetType" size="small" @change="onRuleAssetTypeChange(rule)">
                <el-radio-button value="host">主机</el-radio-button>
                <el-radio-button value="network_device">网络设备</el-radio-button>
              </el-radio-group>
            </div>

            <!-- 资产分组 -->
            <div class="rule-field">
              <label class="rule-field-label">资产分组</label>
              <el-tree-select
                v-model="rule.assetGroupId"
                :data="rule.groupTreeData"
                check-strictly
                :render-after-expand="false"
                placeholder="选择分组"
                style="width: 100%"
                size="small"
                @change="onRuleGroupChange(rule)"
              />
            </div>

            <!-- 资产范围 -->
            <div class="rule-field">
              <label class="rule-field-label">资产范围</label>
              <el-radio-group v-model="rule.hostMode" size="small" @change="onRuleHostModeChange(rule)">
                <el-radio-button value="all">{{ rule.assetType === 'network_device' ? '全部设备' : '全部主机' }}</el-radio-button>
                <el-radio-button value="specific">{{ rule.assetType === 'network_device' ? '指定设备' : '指定主机' }}</el-radio-button>
              </el-radio-group>
            </div>

            <!-- 指定主机/设备 -->
            <div v-if="rule.hostMode === 'specific'" class="rule-field rule-field-wide">
              <label class="rule-field-label">{{ rule.assetType === 'network_device' ? '选择设备' : '选择主机' }}</label>
              <el-select
                v-model="rule.hostIds"
                multiple
                :placeholder="rule.assetType === 'network_device' ? '选择设备' : '选择主机'"
                style="width: 100%"
                size="small"
                :loading="rule.loadingHosts"
              >
                <el-option
                  v-for="item in rule.hostList"
                  :key="item.id"
                  :label="`${item.name} (${item.ip})`"
                  :value="item.id"
                />
              </el-select>
            </div>

            <!-- 操作权限 -->
            <div class="rule-field rule-field-wide">
              <label class="rule-field-label">操作权限</label>
              <el-checkbox-group v-model="rule.permissions" size="small">
                <el-checkbox :value="1">查看</el-checkbox>
                <el-checkbox :value="2">编辑</el-checkbox>
                <el-checkbox :value="4">删除</el-checkbox>
                <el-checkbox :value="8">终端</el-checkbox>
                <template v-if="rule.assetType !== 'network_device'">
                  <el-checkbox :value="16">文件</el-checkbox>
                  <el-checkbox :value="32">采集</el-checkbox>
                </template>
              </el-checkbox-group>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <span v-if="ruleRows.length > 0" class="rule-count-hint">共 {{ ruleRows.length }} 条规则</span>
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleBatchSubmit" :loading="submitting" :disabled="!batchRoleId || ruleRows.length === 0">
            提交 {{ ruleRows.length > 0 ? `(${ruleRows.length} 条)` : '' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 编辑权限对话框（单条编辑） -->
    <el-dialog
      v-model="editDialogVisible"
      title="编辑权限"
      width="50%"
      class="permission-dialog responsive-dialog"
      :close-on-click-modal="false"
      @close="handleEditDialogClose"
    >
      <el-form
        ref="editFormRef"
        :model="editFormData"
        :rules="editFormRules"
        label-width="100px"
      >
        <el-form-item label="角色" prop="roleId">
          <el-select
            v-model="editFormData.roleId"
            placeholder="请选择角色"
            style="width: 100%"
            clearable
            filterable
            disabled
          >
            <el-option
              v-for="role in roleList"
              :key="role.id"
              :label="role.name"
              :value="role.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="资产类型">
          <el-tag :type="editFormData.assetType === 'network_device' ? 'warning' : 'primary'">
            {{ editFormData.assetType === 'network_device' ? '网络设备' : '主机' }}
          </el-tag>
        </el-form-item>

        <el-form-item label="资产分组" prop="assetGroupId">
          <el-tree-select
            v-model="editFormData.assetGroupId"
            :data="editGroupTreeData"
            check-strictly
            :render-after-expand="false"
            placeholder="请选择资产分组"
            style="width: 100%"
            disabled
          />
        </el-form-item>

        <el-form-item :label="editFormData.assetType === 'network_device' ? '网络设备' : '主机'">
          <el-radio-group v-model="editHostSelectionType" @change="handleEditHostTypeChange">
            <el-radio value="all">{{ editFormData.assetType === 'network_device' ? '全部设备' : '全部主机' }}</el-radio>
            <el-radio value="specific">{{ editFormData.assetType === 'network_device' ? '指定设备' : '指定主机' }}</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="editHostSelectionType === 'specific'" :label="editFormData.assetType === 'network_device' ? '选择设备' : '选择主机'" prop="hostIds">
          <el-select
            v-model="editFormData.hostIds"
            multiple
            :placeholder="editFormData.assetType === 'network_device' ? '请选择网络设备' : '请选择主机'"
            style="width: 100%"
            :loading="editLoadingHosts"
          >
            <el-option
              v-for="item in editHostList"
              :key="item.id"
              :label="`${item.name} (${item.ip})`"
              :value="item.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="操作权限">
          <el-checkbox-group v-model="editFormData.permissions">
            <template v-if="editFormData.assetType === 'network_device'">
              <el-checkbox :value="1">查看 - 查看网络设备详情</el-checkbox>
              <el-checkbox :value="2">编辑 - 创建、修改设备配置</el-checkbox>
              <el-checkbox :value="4">删除 - 删除网络设备</el-checkbox>
              <el-checkbox :value="8">终端 - SSH/Telnet远程连接</el-checkbox>
            </template>
            <template v-else>
              <el-checkbox :value="1">查看 - 查看主机详情</el-checkbox>
              <el-checkbox :value="2">编辑 - 创建、修改主机配置</el-checkbox>
              <el-checkbox :value="4">删除 - 删除主机</el-checkbox>
              <el-checkbox :value="8">终端 - SSH/RDP远程连接</el-checkbox>
              <el-checkbox :value="16">文件 - 文件上传、下载、删除</el-checkbox>
              <el-checkbox :value="32">采集 - 采集主机系统信息</el-checkbox>
            </template>
          </el-checkbox-group>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="editDialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleEditSubmit" :loading="editSubmitting">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Delete, Search, RefreshLeft, Lock, Edit } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import {
  getAssetPermissions,
  createAssetPermission,
  deleteAssetPermission,
  getAssetPermissionDetail,
  updateAssetPermission
} from '@/api/assetPermission'
import { getAllRoles } from '@/api/role'
import { getGroupTree } from '@/api/assetGroup'
import { getHostList, getNetworkDeviceList } from '@/api/host'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const isAdmin = computed(() => {
  const roles = userStore.userInfo?.roles || []
  return roles.some((r: any) => r.code === 'admin')
})

// ==================== 列表页相关 ====================
const loading = ref(false)
const permissions = ref<any[]>([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const deletingId = ref(0)
const roleList = ref<any[]>([])

const searchForm = reactive({
  roleName: '',
  groupName: ''
})

const filteredPermissions = computed(() => {
  let result = permissions.value
  if (searchForm.roleName) {
    result = result.filter(item => item.roleName?.includes(searchForm.roleName))
  }
  if (searchForm.groupName) {
    result = result.filter(item => item.assetGroupName?.includes(searchForm.groupName))
  }
  return result
})

const loadPermissions = async () => {
  loading.value = true
  try {
    const response = await getAssetPermissions({ page: page.value, pageSize: pageSize.value })
    permissions.value = response.list || []
    total.value = response.total || 0
  } catch (error: any) {
    ElMessage.error('加载权限列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const loadRoles = async () => {
  try {
    const response = await getAllRoles()
    roleList.value = (response || []).map((item: any) => ({
      id: item.ID,
      name: item.name,
      code: item.code
    }))
  } catch (error: any) {
    ElMessage.error('加载角色列表失败: ' + (error.message || '未知错误'))
  }
}

const handleSearch = () => { page.value = 1; loadPermissions() }
const handleReset = () => { searchForm.roleName = ''; searchForm.groupName = ''; page.value = 1; loadPermissions() }
const handleSizeChange = () => { page.value = 1; loadPermissions() }
const handlePageChange = () => { loadPermissions() }

const handleDeleteClick = (row: any) => {
  ElMessageBox.confirm('确定删除此权限吗？', '提示', {
    confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning'
  }).then(async () => { await handleDelete(row.id) }).catch(() => {})
}

const handleDelete = async (id: number) => {
  if (!isAdmin.value) {
    ElMessage.error('无权限，请联系管理员操作')
    return
  }
  deletingId.value = id
  try {
    await deleteAssetPermission(id)
    ElMessage.success('删除成功')
    loadPermissions()
  } catch (error: any) {
    ElMessage.error('删除失败: ' + (error.message || '未知错误'))
  } finally {
    deletingId.value = 0
  }
}

const formatTime = (time: string) => {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN')
}

// ==================== 批量添加对话框 ====================

interface RuleRow {
  id: number               // 唯一标识
  assetType: string        // 'host' | 'network_device'
  assetGroupId: number | null
  hostMode: string         // 'all' | 'specific'
  hostIds: number[]
  permissions: number[]
  groupTreeData: any[]     // 每行独立的分组树
  hostList: any[]          // 每行独立的主机列表
  loadingHosts: boolean
}

const dialogVisible = ref(false)
const submitting = ref(false)
const batchRoleId = ref<number | null>(null)
const ruleRows = ref<RuleRow[]>([])
let ruleIdCounter = 0

// 转换树形数据格式
const convertTreeData = (nodes: any[]): any[] => {
  return nodes.map((node: any) => ({
    value: node.id,
    label: node.name,
    children: node.children ? convertTreeData(node.children) : undefined
  }))
}

// 为某一行加载分组树
const loadGroupTreeForRule = async (rule: RuleRow) => {
  try {
    const category = rule.assetType === 'network_device' ? 'network' : 'host'
    const data = await getGroupTree(category)
    rule.groupTreeData = convertTreeData(data || [])
  } catch {
    rule.groupTreeData = []
  }
}

// 为某一行加载主机/设备列表
const loadHostsForRule = async (rule: RuleRow) => {
  if (!rule.assetGroupId) return
  rule.loadingHosts = true
  try {
    if (rule.assetType === 'network_device') {
      const response = await getNetworkDeviceList({ page: 1, pageSize: 1000, groupId: rule.assetGroupId })
      rule.hostList = (response.list || []).map((item: any) => ({ id: item.id, name: item.name, ip: item.ip }))
    } else {
      const response = await getHostList({ page: 1, pageSize: 1000, groupId: rule.assetGroupId })
      rule.hostList = (response.list || []).map((item: any) => ({ id: item.id, name: item.name, ip: item.ip }))
    }
  } catch {
    rule.hostList = []
  } finally {
    rule.loadingHosts = false
  }
}

// 添加规则行
const addRuleRow = () => {
  const rule: RuleRow = {
    id: ++ruleIdCounter,
    assetType: 'host',
    assetGroupId: null,
    hostMode: 'all',
    hostIds: [],
    permissions: [1], // 默认查看
    groupTreeData: [],
    hostList: [],
    loadingHosts: false
  }
  ruleRows.value.push(rule)
  // 必须拿到 reactive proxy 引用，否则异步赋值不触发响应式
  const reactiveRule = ruleRows.value[ruleRows.value.length - 1]
  loadGroupTreeForRule(reactiveRule)
}

// 移除规则行
const removeRuleRow = (index: number) => {
  ruleRows.value.splice(index, 1)
}

// 规则行：资产类型变化
const onRuleAssetTypeChange = (rule: RuleRow) => {
  rule.assetGroupId = null
  rule.hostIds = []
  rule.hostList = []
  rule.hostMode = 'all'
  // 网络设备没有文件(16)和采集(32)权限，移除
  if (rule.assetType === 'network_device') {
    rule.permissions = rule.permissions.filter(p => p <= 8)
  }
  loadGroupTreeForRule(rule)
}

// 规则行：分组变化
const onRuleGroupChange = (rule: RuleRow) => {
  rule.hostIds = []
  rule.hostList = []
  if (rule.assetGroupId) {
    loadHostsForRule(rule)
  }
}

// 规则行：资产范围变化
const onRuleHostModeChange = (rule: RuleRow) => {
  if (rule.hostMode === 'specific' && rule.assetGroupId) {
    loadHostsForRule(rule)
  } else {
    rule.hostIds = []
  }
}

// 打开添加对话框
const handleAdd = () => {
  if (!isAdmin.value) {
    ElMessage.error('无权限，请联系管理员操作')
    return
  }
  batchRoleId.value = null
  ruleRows.value = []
  ruleIdCounter = 0
  dialogVisible.value = true
}

const handleDialogClose = () => {
  batchRoleId.value = null
  ruleRows.value = []
}

// 批量提交
const handleBatchSubmit = async () => {
  if (!batchRoleId.value) {
    ElMessage.warning('请选择角色')
    return
  }
  if (ruleRows.value.length === 0) {
    ElMessage.warning('请至少添加一条规则')
    return
  }

  // 校验每一行
  for (let i = 0; i < ruleRows.value.length; i++) {
    const rule = ruleRows.value[i]
    if (!rule.assetGroupId) {
      ElMessage.warning(`规则 ${i + 1}：请选择资产分组`)
      return
    }
    if (rule.permissions.length === 0) {
      ElMessage.warning(`规则 ${i + 1}：请至少选择一项操作权限`)
      return
    }
    if (rule.hostMode === 'specific' && rule.hostIds.length === 0) {
      ElMessage.warning(`规则 ${i + 1}：已选择"指定${rule.assetType === 'network_device' ? '设备' : '主机'}"，请选择具体的${rule.assetType === 'network_device' ? '设备' : '主机'}`)
      return
    }
  }

  submitting.value = true
  let successCount = 0
  let failCount = 0

  try {
    for (const rule of ruleRows.value) {
      const permBitmask = rule.permissions.reduce((acc, val) => acc | val, 0)
      try {
        await createAssetPermission({
          roleId: batchRoleId.value,
          assetGroupId: rule.assetGroupId!,
          assetType: rule.assetType,
          hostIds: rule.hostMode === 'all' ? [] : rule.hostIds,
          permissions: permBitmask
        })
        successCount++
      } catch {
        failCount++
      }
    }

    if (failCount === 0) {
      ElMessage.success(`成功添加 ${successCount} 条权限规则`)
    } else {
      ElMessage.warning(`添加完成：成功 ${successCount} 条，失败 ${failCount} 条`)
    }
    dialogVisible.value = false
    loadPermissions()
  } catch (error: any) {
    ElMessage.error('添加失败: ' + (error.message || '未知错误'))
  } finally {
    submitting.value = false
  }
}

// ==================== 编辑对话框（单条） ====================
const editDialogVisible = ref(false)
const editSubmitting = ref(false)
const editFormRef = ref<FormInstance>()
const editHostSelectionType = ref('all')
const editLoadingHosts = ref(false)
const editHostList = ref<any[]>([])
const editGroupTreeData = ref<any[]>([])

const editFormData = reactive({
  id: null as number | null,
  roleId: null as number | null,
  assetGroupId: null as number | null,
  assetType: 'host' as string,
  hostIds: [] as number[],
  permissions: [] as number[]
})

const editFormRules: FormRules = {
  roleId: [{ required: true, message: '请选择角色', trigger: 'change' }]
}

const handleEditClick = async (row: any) => {
  if (!isAdmin.value) {
    ElMessage.error('无权限，请联系管理员操作')
    return
  }
  try {
    // 加载分组树
    const data = await getGroupTree()
    editGroupTreeData.value = convertTreeData(data || [])

    const detail = await getAssetPermissionDetail(row.id)
    editFormData.id = detail.id
    editFormData.roleId = detail.roleId
    editFormData.assetGroupId = detail.assetGroupId
    editFormData.assetType = detail.assetType || 'host'
    editFormData.hostIds = detail.hostIds || []
    editFormData.permissions = []

    if ((detail.permissions & 1) > 0) editFormData.permissions.push(1)
    if ((detail.permissions & 2) > 0) editFormData.permissions.push(2)
    if ((detail.permissions & 4) > 0) editFormData.permissions.push(4)
    if ((detail.permissions & 8) > 0) editFormData.permissions.push(8)
    if ((detail.permissions & 16) > 0) editFormData.permissions.push(16)
    if ((detail.permissions & 32) > 0) editFormData.permissions.push(32)

    editHostSelectionType.value = (!detail.hostIds || detail.hostIds.length === 0) ? 'all' : 'specific'

    if (editHostSelectionType.value === 'specific') {
      await loadEditHosts(detail.assetGroupId)
    }

    editDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error('加载权限详情失败: ' + (error.message || '未知错误'))
  }
}

const loadEditHosts = async (groupId?: number) => {
  if (!groupId) return
  editLoadingHosts.value = true
  try {
    if (editFormData.assetType === 'network_device') {
      const response = await getNetworkDeviceList({ page: 1, pageSize: 1000, groupId })
      editHostList.value = (response.list || []).map((item: any) => ({ id: item.id, name: item.name, ip: item.ip }))
    } else {
      const response = await getHostList({ page: 1, pageSize: 1000, groupId })
      editHostList.value = (response.list || []).map((item: any) => ({ id: item.id, name: item.name, ip: item.ip }))
    }
  } catch (error: any) {
    ElMessage.error('加载列表失败: ' + (error.message || '未知错误'))
  } finally {
    editLoadingHosts.value = false
  }
}

const handleEditHostTypeChange = (value: string) => {
  if (value === 'specific' && editFormData.assetGroupId) {
    loadEditHosts(editFormData.assetGroupId)
  } else {
    editFormData.hostIds = []
  }
}

const handleEditDialogClose = () => {
  editFormData.id = null
  editFormData.roleId = null
  editFormData.assetGroupId = null
  editFormData.assetType = 'host'
  editFormData.hostIds = []
  editFormData.permissions = []
  editHostSelectionType.value = 'all'
  editHostList.value = []
  editGroupTreeData.value = []
  editFormRef.value?.clearValidate()
}

const handleEditSubmit = async () => {
  if (editFormData.id === null) return
  editSubmitting.value = true
  try {
    const permBitmask = editFormData.permissions.reduce((acc, val) => acc | val, 0)
    await updateAssetPermission(editFormData.id, {
      roleId: editFormData.roleId!,
      assetGroupId: editFormData.assetGroupId!,
      assetType: editFormData.assetType,
      hostIds: editHostSelectionType.value === 'all' ? [] : editFormData.hostIds,
      permissions: permBitmask
    })
    ElMessage.success('更新成功')
    editDialogVisible.value = false
    loadPermissions()
  } catch (error: any) {
    ElMessage.error('更新失败: ' + (error.message || '未知错误'))
  } finally {
    editSubmitting.value = false
  }
}

// ==================== 初始化 ====================
onMounted(() => {
  loadPermissions()
  loadRoles()
})
</script>

<style scoped>
.permission-container {
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

.action-edit:hover {
  background-color: #e6f7ff;
  color: #1890ff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

/* 分页 */
.pagination-container {
  padding: 12px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
  border-radius: 0;
  display: flex;
  justify-content: flex-end;
}

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 12px;
}

.rule-count-hint {
  margin-right: auto;
  font-size: 13px;
  color: #909399;
}

:deep(.permission-dialog) {
  border-radius: 0;
}

:deep(.permission-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.permission-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.permission-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

/* 批量对话框 */
:deep(.batch-dialog) {
  max-width: 1100px;
  min-width: 700px;
}

:deep(.batch-dialog .el-dialog__body) {
  padding: 20px 24px;
  max-height: 65vh;
  overflow-y: auto;
}

/* 批量角色区域 */
.batch-role-section {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
  margin-bottom: 16px;
}

.batch-label {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  white-space: nowrap;
}

/* 批量规则区域 */
.batch-rules-section {
  /* wrapper */
}

.batch-rules-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.batch-empty {
  text-align: center;
  padding: 40px 20px;
  color: #909399;
  font-size: 14px;
  border: 1px dashed #dcdfe6;
}

/* 规则行 */
.rule-row {
  border: 1px solid #e4e7ed;
  margin-bottom: 12px;
  background: #fafbfc;
  transition: box-shadow 0.2s;
}

.rule-row:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.rule-row-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  background: #f0f2f5;
  border-bottom: 1px solid #e4e7ed;
}

.rule-row-index {
  font-size: 13px;
  font-weight: 600;
  color: #606266;
}

.rule-remove-btn {
  font-size: 16px;
}

.rule-row-body {
  padding: 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-start;
}

.rule-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 180px;
  flex: 1;
}

.rule-field-wide {
  min-width: 300px;
  flex: 2;
}

.rule-field-label {
  font-size: 12px;
  font-weight: 500;
  color: #909399;
}

/* 标签样式 */
:deep(.el-tag) {
  border-radius: 0;
  padding: 4px 10px;
  font-weight: 500;
}

/* 权限标签样式 */
.permission-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

/* 响应式对话框 */
:deep(.responsive-dialog) {
  max-width: 900px;
  min-width: 500px;
}

@media (max-width: 768px) {
  :deep(.responsive-dialog .el-dialog) {
    width: 95% !important;
    max-width: none;
    min-width: auto;
  }

  :deep(.batch-dialog .el-dialog) {
    width: 95% !important;
    max-width: none;
    min-width: auto;
  }

  .search-input {
    width: auto;
    flex: 1;
    min-width: 200px;
  }

  .search-inputs {
    flex-direction: column;
  }

  .rule-row-body {
    flex-direction: column;
  }

  .rule-field, .rule-field-wide {
    min-width: auto;
    width: 100%;
  }
}
</style>
