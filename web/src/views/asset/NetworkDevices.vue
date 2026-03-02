<template>
  <div class="network-devices-page">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><SetUp /></el-icon>
        </div>
        <div>
          <h2 class="page-title">网络设备</h2>
          <p class="page-subtitle">管理交换机、路由器、防火墙等网络设备，支持 SSH 和 Telnet 终端连接</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="terminal-button" @click="handleOpenTerminal">
          <el-icon style="margin-right: 6px;"><Monitor /></el-icon>
          终端
        </el-button>
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增设备
        </el-button>
      </div>
    </div>

    <!-- 主内容区域：左侧分组树 + 右侧设备列表 -->
    <div class="main-content">
      <!-- 左侧分组树 -->
      <div class="left-panel">
        <div class="panel-header">
          <div class="panel-title">
            <el-icon class="panel-icon"><Collection /></el-icon>
            <span>资产分组</span>
          </div>
          <div class="panel-actions">
            <el-tooltip content="新增分组" placement="top">
              <el-button circle size="small" @click="handleAddGroup">
                <el-icon><Plus /></el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip :content="isExpandAll ? '折叠全部' : '展开全部'" placement="top">
              <el-button circle size="small" @click="toggleExpandAll">
                <el-icon><Sort /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>
        <div class="panel-body">
          <el-input
            v-model="groupSearchKeyword"
            placeholder="搜索分组..."
            clearable
            size="small"
            class="group-search"
            @input="filterGroupTreeData"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <div class="tree-container" v-loading="groupLoading">
            <el-tree
              ref="groupTreeRef"
              :data="filteredGroupTree"
              :props="treeProps"
              :default-expand-all="false"
              :expand-on-click-node="false"
              :highlight-current="true"
              node-key="id"
              class="group-tree"
              @node-click="handleGroupClick"
            >
              <template #default="{ node, data }">
                <div class="tree-node">
                  <span class="node-icon">
                    <el-icon v-if="!data.parentId || data.parentId === 0" color="#67c23a">
                      <Folder />
                    </el-icon>
                    <el-icon v-else color="#409eff">
                      <FolderOpened />
                    </el-icon>
                  </span>
                  <span class="node-label">{{ data.name }}</span>
                  <span class="node-count">({{ data.deviceCount || 0 }})</span>
                  <span class="node-actions" @click.stop>
                    <el-dropdown trigger="click" @command="(cmd: string) => handleGroupAction(cmd, data)">
                      <el-icon class="more-icon"><MoreFilled /></el-icon>
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item command="edit">
                            <el-icon><Edit /></el-icon> 编辑
                          </el-dropdown-item>
                          <el-dropdown-item command="delete">
                            <el-icon><Delete /></el-icon> 删除
                          </el-dropdown-item>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </span>
                </div>
              </template>
            </el-tree>
            <el-empty v-if="filteredGroupTree.length === 0 && !groupLoading" description="暂无分组" :image-size="60" />
          </div>
        </div>
      </div>

      <!-- 右侧设备列表 -->
      <div class="right-panel">
        <!-- 当前分组提示 -->
        <div v-if="selectedGroup" class="group-breadcrumb">
          <span class="breadcrumb-label">当前分组：</span>
          <el-tag size="small" effect="plain" closable @close="clearGroupSelection">
            {{ selectedGroup.name }}
          </el-tag>
        </div>

        <!-- 搜索和筛选 -->
        <div class="filter-bar">
          <div class="filter-inputs">
            <el-input
              v-model="searchKeyword"
              placeholder="搜索设备名称、IP、品牌型号、SN..."
              clearable
              style="width: 280px"
              @clear="loadDevices"
              @keyup.enter="loadDevices"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
            <el-select v-model="filterDeviceType" placeholder="设备类型" clearable style="width: 130px" @change="loadDevices">
              <el-option v-for="dt in deviceTypes" :key="dt.value" :label="dt.label" :value="dt.value" />
            </el-select>
            <el-select v-model="filterProtocol" placeholder="连接协议" clearable style="width: 130px" @change="loadDevices">
              <el-option label="SSH" value="ssh" />
              <el-option label="Telnet" value="telnet" />
            </el-select>
          </div>
          <div class="filter-actions">
            <el-button
              v-if="selectedDevices.length > 0"
              type="danger"
              plain
              @click="handleBatchDelete"
            >
              <el-icon style="margin-right: 4px;"><Delete /></el-icon>
              批量删除 ({{ selectedDevices.length }})
            </el-button>
            <el-button @click="resetFilters">
              <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
              重置
            </el-button>
            <el-button @click="loadDevices">
              <el-icon style="margin-right: 4px;"><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>

        <!-- 设备表格 -->
        <div class="table-wrapper">
          <el-table
            :data="deviceList"
            v-loading="loading"
            class="modern-table"
            :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
            @selection-change="handleSelectionChange"
            @row-dblclick="handleConnect"
          >
            <el-table-column type="selection" width="30" fixed="left" />
            <el-table-column width="30" fixed="left" align="center">
              <template #default="{ row }">
                <div class="device-avatar">
                  <el-icon class="device-icon" :style="{ color: getDeviceTypeColor(row.deviceType) }">
                    <component :is="getDeviceTypeIcon(row.deviceType)" />
                  </el-icon>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="设备名称" prop="name" min-width="100" fixed="left">
              <template #default="{ row }">
                <div class="device-name-cell">
                  <div class="device-info">
                    <div class="device-name">{{ row.name }}</div>
                    <div class="device-meta">
                      <span class="ip">{{ row.ip }}</span>
                    </div>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="80" align="center">
              <template #default="{ row }">
                <div class="status-cell">
                  <span class="status-dot" :class="`status-dot-${row.status ?? -1}`"></span>
                  <span class="status-text" :class="`status-text-${row.status ?? -1}`">
                    {{ row.status === 1 ? '在线' : row.status === 0 ? '离线' : '未知' }}
                  </span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="品牌" prop="brand" width="120">
              <template #default="{ row }">
                <span>{{ row.brand || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="型号" prop="brandModel" min-width="140" show-overflow-tooltip>
              <template #default="{ row }">
                <span>{{ row.brandModel || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="设备类型" prop="deviceType" width="120" align="center">
              <template #default="{ row }">
                <el-tag :color="getDeviceTypeColor(row.deviceType)" effect="dark" size="small" style="border: none; color: #fff;">
                  {{ getDeviceTypeLabel(row.deviceType) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="协议" prop="protocol" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="row.protocol === 'ssh' ? 'success' : 'warning'" size="small" effect="plain">
                  {{ row.protocol?.toUpperCase() }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="端口" prop="port" width="70" align="center" />
            <el-table-column label="凭证" min-width="100">
              <template #default="{ row }">
                <span v-if="row.credential">{{ row.credential.name }}</span>
                <span v-else class="text-muted">未配置</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="180" fixed="right" align="center">
              <template #default="{ row }">
                <div class="action-buttons">
                  <el-tooltip content="连接" placement="top">
                    <el-button
                      link
                      class="action-btn action-connect"
                      @click="handleConnect(row)"
                    >
                      <el-icon><Connection /></el-icon>
                    </el-button>
                  </el-tooltip>
                  <el-tooltip content="测试连接" placement="top">
                    <el-button
                      link
                      class="action-btn action-refresh"
                      @click="handleTest(row)"
                    >
                      <el-icon><CircleCheck /></el-icon>
                    </el-button>
                  </el-tooltip>
                  <el-tooltip content="编辑" placement="top">
                    <el-button
                      link
                      class="action-btn action-edit"
                      @click="handleEdit(row)"
                    >
                      <el-icon><Edit /></el-icon>
                    </el-button>
                  </el-tooltip>
                  <el-tooltip content="删除" placement="top">
                    <el-button
                      link
                      class="action-btn action-delete"
                      @click="handleDelete(row)"
                    >
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </el-tooltip>
                </div>
              </template>
            </el-table-column>
          </el-table>

          <!-- 分页 -->
          <div class="pagination-bar">
            <el-pagination
              v-model:current-page="currentPage"
              v-model:page-size="pageSize"
              :total="total"
              :page-sizes="[10, 20, 50, 100]"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="loadDevices"
              @current-change="loadDevices"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- 新增/编辑弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑网络设备' : '新增网络设备'"
      width="600px"
      destroy-on-close
      class="device-dialog"
    >
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="100px">
        <el-form-item label="设备名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入设备名称" />
        </el-form-item>
        <el-form-item label="IP 地址" prop="ip">
          <el-input v-model="form.ip" placeholder="请输入设备 IP 地址" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="品牌" prop="brand">
              <el-select v-model="form.brand" placeholder="请选择品牌" filterable allow-create style="width: 100%">
                <el-option-group v-for="grp in brandOptions" :key="grp.group" :label="grp.group">
                  <el-option v-for="item in grp.items" :key="item.value" :label="item.label" :value="item.value" />
                </el-option-group>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="型号" prop="brandModel">
              <el-input v-model="form.brandModel" placeholder="如：S5720、WS-C2960" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="SN 序列号" prop="serialNumber">
          <el-input v-model="form.serialNumber" placeholder="非必填" />
        </el-form-item>
        <el-form-item label="设备类型" prop="deviceType">
          <el-select v-model="form.deviceType" placeholder="请选择" style="width: 100%">
            <el-option v-for="dt in deviceTypes" :key="dt.value" :label="dt.label" :value="dt.value" />
          </el-select>
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="连接协议" prop="protocol">
              <el-select v-model="form.protocol" style="width: 100%" @change="onProtocolChange">
                <el-option label="SSH" value="ssh" />
                <el-option label="Telnet" value="telnet" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="端口" prop="port">
              <el-input-number v-model="form.port" :min="1" :max="65535" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="凭证" prop="credentialId">
          <el-select v-model="form.credentialId" placeholder="选择凭证" clearable style="width: 100%">
            <el-option v-for="c in credentials" :key="c.id" :label="`${c.name} (${c.type === 'password' ? '密码' : '密钥'})`" :value="c.id" />
          </el-select>
          <div v-if="form.protocol === 'telnet'" style="color: #909399; font-size: 12px; margin-top: 4px;">
            Telnet 连接时在终端中交互输入账号密码，此处仅作记录
          </div>
        </el-form-item>
        <el-form-item label="分组" prop="groupId">
          <el-tree-select
            v-model="form.groupId"
            :data="groupTree"
            :props="{ value: 'id', label: 'name', children: 'children' }"
            clearable
            check-strictly
            placeholder="选择分组"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="标签" prop="tags">
          <el-input v-model="form.tags" placeholder="多个标签用逗号分隔" />
        </el-form-item>
        <el-form-item label="备注" prop="description">
          <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>

    <!-- 新增/编辑分组对话框 -->
    <el-dialog
      v-model="groupDialogVisible"
      :title="isGroupEdit ? '编辑分组' : '新增分组'"
      width="480px"
      destroy-on-close
      class="device-dialog"
    >
      <el-form ref="groupFormRef" :model="groupForm" :rules="groupRules" label-width="100px">
        <el-form-item label="上级分组">
          <el-tree-select
            v-model="groupForm.parentId"
            :data="groupTree"
            :props="{ value: 'id', label: 'name', children: 'children' }"
            clearable
            check-strictly
            placeholder="不选择则为顶级分组"
          />
        </el-form-item>
        <el-form-item label="分组名称" prop="name">
          <el-input v-model="groupForm.name" placeholder="请输入分组名称" />
        </el-form-item>
        <el-form-item label="分组编码" prop="code">
          <el-input v-model="groupForm.code" placeholder="请输入分组编码" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="groupForm.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="groupForm.sort" :min="0" />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-radio-group v-model="groupForm.status">
            <el-radio :label="1">正常</el-radio>
            <el-radio :label="0">停用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="groupDialogVisible = false">取消</el-button>
        <el-button class="black-button" @click="handleGroupSubmit" :loading="groupSubmitting">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import {
  Plus, Search, Monitor, SetUp, Edit, Delete, Connection,
  CircleCheck, Collection, Folder, FolderOpened, Sort,
  MoreFilled, Refresh, RefreshLeft
} from '@element-plus/icons-vue'
import {
  getNetworkDeviceList,
  createNetworkDevice,
  updateNetworkDevice,
  deleteNetworkDevice,
  testNetworkDeviceConnection,
  getCredentials
} from '@/api/host'
import {
  getGroupTree,
  createGroup,
  updateGroup,
  deleteGroup
} from '@/api/assetGroup'
import { useRouter } from 'vue-router'
import { getUserDevicePermissions } from '@/api/assetPermission'
import { PERMISSION, hasPermission } from '@/utils/permission'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

// 管理员判断
const isAdmin = computed(() => {
  const roles = userStore.userInfo?.roles || []
  return roles.some((r: any) => r.code === 'admin')
})

// 设备权限映射
const devicePermissions = ref<Map<number, number>>(new Map())

const hasDevicePermission = (deviceId: number, permission: number): boolean => {
  if (isAdmin.value) return true
  const perms = devicePermissions.value.get(deviceId) || 0
  return hasPermission(perms, permission)
}

// 加载所有设备的权限
const loadDevicePermissions = async () => {
  if (isAdmin.value) return
  const newMap = new Map<number, number>()
  for (const device of deviceList.value) {
    try {
      const res: any = await getUserDevicePermissions(device.id)
      if (res && res.permissions !== undefined) {
        newMap.set(device.id, res.permissions)
      }
    } catch {}
  }
  devicePermissions.value = newMap
}

// 设备类型选项
const deviceTypes = [
  { value: 'switch', label: '交换机' },
  { value: 'router', label: '路由器' },
  { value: 'firewall', label: '防火墙' },
  { value: 'ac', label: 'AC' },
  { value: 'ap', label: 'AP' },
  { value: 'other', label: '其他' }
]

// 品牌选项（按分类组织）
const brandOptions = [
  { group: '网络设备', items: [
    { value: 'Cisco', label: 'Cisco (思科)' },
    { value: 'Huawei', label: 'Huawei (华为)' },
    { value: 'H3C', label: 'H3C (新华三)' },
    { value: 'Ruijie', label: 'Ruijie (锐捷)' },
    { value: 'ZTE', label: 'ZTE (中兴)' },
    { value: 'Sundary', label: 'Sundary (信锐)' },
    { value: 'Juniper', label: 'Juniper (瞻博)' },
    { value: 'Aruba', label: 'Aruba' },
    { value: 'Dell', label: 'Dell' },
    { value: 'HPE', label: 'HPE' },
    { value: 'MikroTik', label: 'MikroTik' },
    { value: 'Ubiquiti', label: 'Ubiquiti' },
    { value: 'TP-Link', label: 'TP-Link' },
    { value: 'D-Link', label: 'D-Link' },
    { value: 'Netgear', label: 'Netgear' },
    { value: 'Extreme', label: 'Extreme Networks' },
    { value: 'Arista', label: 'Arista' },
    { value: 'Brocade', label: 'Brocade (博科)' },
  ]},
  { group: '安全设备', items: [
    { value: 'PaloAlto', label: 'Palo Alto Networks' },
    { value: 'Fortinet', label: 'Fortinet (飞塔)' },
    { value: 'CheckPoint', label: 'Check Point' },
    { value: 'Hillstone', label: 'Hillstone (山石网科)' },
    { value: 'Sangfor', label: 'Sangfor (深信服)' },
    { value: 'Venustech', label: 'Venustech (启明星辰)' },
    { value: 'NSFOCUS', label: 'NSFOCUS (绿盟)' },
    { value: 'TopSec', label: 'TopSec (天融信)' },
    { value: 'DPtech', label: 'DPtech (迪普科技)' },
    { value: 'Leadsec', label: 'Leadsec (网御星云)' },
    { value: 'Legendsec', label: 'Legendsec (网神)' },
  ]},
  { group: '负载均衡', items: [
    { value: 'F5', label: 'F5' },
    { value: 'A10', label: 'A10 Networks' },
    { value: 'Radware', label: 'Radware' },
    { value: 'Array', label: 'Array Networks' },
    { value: 'Sangfor-LB', label: 'Sangfor AD (深信服)' },
  ]},
  { group: '其他', items: [
    { value: 'other', label: '其他' },
  ]},
]

// 状态
const loading = ref(false)
const deviceList = ref<any[]>([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(10)
const searchKeyword = ref('')
const filterDeviceType = ref('')
const filterProtocol = ref('')
const credentials = ref<any[]>([])
const selectedDevices = ref<any[]>([])

// 分组树
const groupLoading = ref(false)
const groupTree = ref<any[]>([])
const filteredGroupTree = ref<any[]>([])
const groupSearchKeyword = ref('')
const selectedGroup = ref<any>(null)
const isExpandAll = ref(false)
const groupTreeRef = ref()

const treeProps = {
  children: 'children',
  label: 'name',
  value: 'id'
}

// 分组对话框
const groupDialogVisible = ref(false)
const isGroupEdit = ref(false)
const groupSubmitting = ref(false)
const groupFormRef = ref<FormInstance>()
const groupForm = reactive({
  id: 0,
  parentId: null as number | null,
  name: '',
  code: '',
  description: '',
  sort: 0,
  status: 1
})

const groupRules = {
  name: [{ required: true, message: '请输入分组名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入分组编码', trigger: 'blur' }],
  status: [{ required: true, message: '请选择状态', trigger: 'change' }]
}

// 弹窗
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const form = reactive({
  id: 0,
  name: '',
  ip: '',
  brand: '',
  brandModel: '',
  serialNumber: '',
  deviceType: 'switch',
  protocol: 'ssh',
  port: 22,
  credentialId: undefined as number | undefined,
  groupId: undefined as number | undefined,
  tags: '',
  description: ''
})

const formRules = {
  name: [{ required: true, message: '请输入设备名称', trigger: 'blur' }],
  ip: [{ required: true, message: '请输入IP地址', trigger: 'blur' }],
  brand: [{ required: true, message: '请选择品牌', trigger: 'change' }],
  deviceType: [{ required: true, message: '请选择设备类型', trigger: 'change' }],
  protocol: [{ required: true, message: '请选择连接协议', trigger: 'change' }]
}

onMounted(() => {
  loadGroupTree()
  loadDevices()
  loadCredentials()
})

// 分组树相关
async function loadGroupTree() {
  groupLoading.value = true
  try {
    const data = await getGroupTree('network')
    groupTree.value = data || []
    filteredGroupTree.value = data || []
  } catch (error) {
    console.error('获取分组树失败', error)
  } finally {
    groupLoading.value = false
  }
}

function filterGroupTreeData() {
  if (!groupSearchKeyword.value) {
    filteredGroupTree.value = groupTree.value
    return
  }
  filteredGroupTree.value = searchTreeNodes(groupTree.value, groupSearchKeyword.value)
}

function searchTreeNodes(nodes: any[], keyword: string): any[] {
  const result: any[] = []
  for (const node of nodes) {
    const matchName = node.name?.toLowerCase().includes(keyword.toLowerCase())
    let filteredChildren: any[] = []
    if (node.children && node.children.length > 0) {
      filteredChildren = searchTreeNodes(node.children, keyword)
    }
    if (matchName || filteredChildren.length > 0) {
      result.push({
        ...node,
        children: filteredChildren.length > 0 ? filteredChildren : node.children
      })
    }
  }
  return result
}

function toggleExpandAll() {
  isExpandAll.value = !isExpandAll.value
  const treeStore = groupTreeRef.value?.store
  if (!treeStore) return
  for (const key in treeStore.nodesMap) {
    const node = treeStore.nodesMap[key]
    if (node) {
      node.expanded = isExpandAll.value
    }
  }
}

function handleGroupClick(data: any) {
  selectedGroup.value = data
  currentPage.value = 1
  loadDevices()
}

function clearGroupSelection() {
  selectedGroup.value = null
  groupTreeRef.value?.setCurrentKey(null)
  currentPage.value = 1
  loadDevices()
}

function handleGroupAction(command: string, data: any) {
  if (command === 'edit') {
    handleEditGroup(data)
  } else if (command === 'delete') {
    handleDeleteGroup(data)
  }
}

function handleAddGroup() {
  isGroupEdit.value = false
  Object.assign(groupForm, { id: 0, parentId: null, name: '', code: '', description: '', sort: 0, status: 1 })
  groupDialogVisible.value = true
}

function handleEditGroup(data: any) {
  isGroupEdit.value = true
  Object.assign(groupForm, {
    id: data.id,
    parentId: data.parentId || null,
    name: data.name,
    code: data.code || '',
    description: data.description || '',
    sort: data.sort || 0,
    status: data.status ?? 1
  })
  groupDialogVisible.value = true
}

async function handleDeleteGroup(data: any) {
  try {
    await ElMessageBox.confirm(`确定删除分组「${data.name}」？`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
    await deleteGroup(data.id)
    ElMessage.success('删除成功')
    loadGroupTree()
    if (selectedGroup.value?.id === data.id) {
      clearGroupSelection()
    }
  } catch { /* cancelled */ }
}

async function handleGroupSubmit() {
  if (!groupFormRef.value) return
  await groupFormRef.value.validate()
  groupSubmitting.value = true
  try {
    const payload: any = {
      name: groupForm.name,
      code: groupForm.code,
      description: groupForm.description,
      sort: groupForm.sort,
      status: groupForm.status,
      parentId: groupForm.parentId || 0,
      category: 'network'
    }
    if (isGroupEdit.value) {
      await updateGroup(groupForm.id, payload)
      ElMessage.success('更新成功')
    } else {
      await createGroup(payload)
      ElMessage.success('创建成功')
    }
    groupDialogVisible.value = false
    loadGroupTree()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    groupSubmitting.value = false
  }
}

// 设备列表
async function loadDevices() {
  loading.value = true
  try {
    const params: any = {
      page: currentPage.value,
      pageSize: pageSize.value
    }
    if (searchKeyword.value) params.keyword = searchKeyword.value
    if (filterDeviceType.value) params.deviceType = filterDeviceType.value
    if (filterProtocol.value) params.protocol = filterProtocol.value
    if (selectedGroup.value) params.groupId = selectedGroup.value.id

    const res: any = await getNetworkDeviceList(params)
    deviceList.value = res?.list || []
    total.value = res?.total || 0
    // 加载设备级权限
    await loadDevicePermissions()
  } catch (e) {
    console.error('加载网络设备失败', e)
  } finally {
    loading.value = false
  }
}

function resetFilters() {
  searchKeyword.value = ''
  filterDeviceType.value = ''
  filterProtocol.value = ''
  clearGroupSelection()
}

async function loadCredentials() {
  try {
    const res: any = await getCredentials('network')
    credentials.value = Array.isArray(res) ? res : []
  } catch (e) { /* ignore */ }
}

function handleAdd() {
  if (!isAdmin.value) {
    ElMessage.error('无权限，请联系管理员操作')
    return
  }
  isEdit.value = false
  resetForm()
  dialogVisible.value = true
}

function handleEdit(row: any) {
  if (!isAdmin.value && !hasDevicePermission(row.id, PERMISSION.EDIT)) {
    ElMessage.error('无权限，请联系管理员操作')
    return
  }
  isEdit.value = true
  Object.assign(form, {
    id: row.id,
    name: row.name,
    ip: row.ip,
    brand: row.brand || '',
    brandModel: row.brandModel || '',
    serialNumber: row.serialNumber || '',
    deviceType: row.deviceType,
    protocol: row.protocol,
    port: row.port,
    credentialId: row.credentialId || undefined,
    groupId: row.groupId || undefined,
    tags: row.tags || '',
    description: row.description || ''
  })
  dialogVisible.value = true
}

function resetForm() {
  Object.assign(form, {
    id: 0,
    name: '',
    ip: '',
    brand: '',
    brandModel: '',
    serialNumber: '',
    deviceType: 'switch',
    protocol: 'ssh',
    port: 22,
    credentialId: undefined,
    groupId: undefined,
    tags: '',
    description: ''
  })
}

function onProtocolChange(val: string) {
  form.port = val === 'telnet' ? 23 : 22
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate()

  submitting.value = true
  try {
    if (isEdit.value) {
      await updateNetworkDevice(form.id, form)
      ElMessage.success('更新成功')
    } else {
      await createNetworkDevice(form)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadDevices()
    loadGroupTree()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

// 选中项变化
function handleSelectionChange(selection: any[]) {
  selectedDevices.value = selection
}

// 批量删除
async function handleBatchDelete() {
  if (selectedDevices.value.length === 0) {
    ElMessage.warning('请先选择要删除的设备')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedDevices.value.length} 台设备吗？`,
      '批量删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    let successCount = 0
    for (const device of selectedDevices.value) {
      if (!hasDevicePermission(device.id, PERMISSION.DELETE)) {
        continue
      }
      try {
        await deleteNetworkDevice(device.id)
        successCount++
      } catch (e) {
        console.error('删除设备失败', device.name, e)
      }
    }

    if (successCount > 0) {
      ElMessage.success(`成功删除 ${successCount} 台设备`)
      loadDevices()
    } else {
      ElMessage.warning('没有设备被删除，可能原因：权限不足或设备不存在')
    }
  } catch { /* cancelled */ }
}

async function handleDelete(row: any) {
  if (!isAdmin.value && !hasDevicePermission(row.id, PERMISSION.DELETE)) {
    ElMessage.error('无权限，请联系管理员操作')
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除设备「${row.name}」(${row.ip})？`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
    await deleteNetworkDevice(row.id)
    ElMessage.success('删除成功')
    loadDevices()
    loadGroupTree()
  } catch { /* cancelled */ }
}

async function handleTest(row: any) {
  if (!isAdmin.value && !hasDevicePermission(row.id, PERMISSION.VIEW)) {
    ElMessage.error('无权限，请联系管理员操作')
    return
  }
  ElMessage.info('正在测试连接...')
  try {
    const res: any = await testNetworkDeviceConnection(row.id)
    ElMessage.success(res?.message || '连接成功')
  } catch (e: any) {
    ElMessage.error(e.message || '连接测试失败')
  } finally {
    // 测试完成后刷新列表，更新设备状态
    loadDevices()
  }
}

function handleConnect(row: any) {
  if (!isAdmin.value && !hasDevicePermission(row.id, PERMISSION.TERMINAL)) {
    ElMessage.error('无权限，请联系管理员操作')
    return
  }
  // 在新窗口打开终端页面
  const url = router.resolve({
    path: '/terminal',
    query: {
      type: 'network-device',
      deviceId: row.id
    }
  }).href
  window.open(url, '_blank')
}

function handleOpenTerminal() {
  const url = router.resolve({
    path: '/terminal',
    query: { type: 'network-device' }
  }).href
  window.open(url, '_blank')
}

function getDeviceTypeLabel(type: string) {
  const found = deviceTypes.find(d => d.value === type)
  return found ? found.label : type
}

function getDeviceTypeColor(type: string) {
  const colors: Record<string, string> = {
    switch: '#409EFF',
    router: '#67C23A',
    firewall: '#E6A23C',
    ac: '#909399',
    ap: '#909399',
    other: '#606266'
  }
  return colors[type] || '#606266'
}

function getDeviceTypeIcon(type: string) {
  // All use SetUp icon for simplicity; could be differentiated later
  return SetUp
}
</script>

<style scoped>
.network-devices-page {
  padding: 0;
  background: #f5f7fa;
  min-height: 100%;
  display: flex;
  flex-direction: column;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 10px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 1px 8px rgba(0, 0, 0, 0.04);
  flex-shrink: 0;
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.page-title-icon {
  width: 36px;
  height: 36px;
  background: #0a466a;
  border-radius: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  font-size: 16px;
  flex-shrink: 0;
  border: none;
}

.page-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 2px 0 0 0;
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.terminal-button {
  background-color: #1a1a1a !important;
  color: #ffffff !important;
  border-color: #1a1a1a !important;
  border-radius: 0;
  padding: 10px 20px;
  font-weight: 500;
}

.terminal-button:hover {
  background-color: #0d5a87 !important;
  border-color: #0d5a87 !important;
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

/* 主内容区域 */
.main-content {
  display: flex;
  gap: 12px;
  flex: 1;
  min-height: 0;
}

/* 左侧分组面板 */
.left-panel {
  width: 200px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 1px 8px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.panel-header {
  padding: 10px 12px;
  border-bottom: 1px solid #e4e7ed;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  font-size: 13px;
  color: #303133;
}

.panel-icon {
  font-size: 14px;
  color: #ffffff;
}

.panel-actions {
  display: flex;
  gap: 6px;
}

.panel-body {
  flex: 1;
  padding: 8px 10px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.group-search {
  margin-bottom: 12px;
}

.group-search :deep(.el-input__wrapper) {
  border-radius: 0;
}

.tree-container {
  flex: 1;
  overflow-y: auto;
}

.group-tree {
  background: transparent;
}

.group-tree :deep(.el-tree-node__content) {
  border-radius: 0;
  padding: 6px 8px;
  transition: all 0.2s ease;
}

.group-tree :deep(.el-tree-node__content:hover) {
  background-color: #f5f7fa;
}

.group-tree :deep(.is-current > .el-tree-node__content) {
  background-color: #ecf5ff;
  color: #409eff;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
  width: 0;
  font-size: 13px;
}

.node-icon {
  flex-shrink: 0;
  font-size: 13px;
}

.node-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ip {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #909399;
}

/* 状态单元格 */
.status-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot-1 {
  background: #67c23a;
  box-shadow: 0 0 0 2px rgba(103, 194, 58, 0.2);
}

.status-dot-0 {
  background: #f56c6c;
  box-shadow: 0 0 0 2px rgba(245, 108, 108, 0.2);
}

.status-dot--1 {
  background: #909399;
  box-shadow: 0 0 0 2px rgba(144, 148, 153, 0.2);
}

.status-text {
  font-size: 13px;
  font-weight: 500;
}

.status-text-1 {
  color: #67c23a;
}

.status-text-0 {
  color: #f56c6c;
}

.status-text--1 {
  color: #909399;
}

/* 覆盖现代表格样式 */
.node-count {
  font-size: 11px;
  color: #909399;
  flex-shrink: 0;
}

.node-actions {
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.2s;
}

.group-tree :deep(.el-tree-node__content:hover) .node-actions {
  opacity: 1;
}

.more-icon {
  font-size: 14px;
  cursor: pointer;
  color: #909399;
}

.more-icon:hover {
  color: #409eff;
}

/* 右侧设备列表面板 */
.right-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.group-breadcrumb {
  padding: 6px 14px;
  background: #fff;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.03);
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.breadcrumb-label {
  color: #909399;
}

.filter-bar {
  padding: 10px 14px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 1px 8px rgba(0, 0, 0, 0.04);
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  flex-wrap: wrap;
  gap: 8px;
}

.filter-inputs {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
}

.filter-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

/* 设备信息单元格 */
.device-name-cell {
  display: flex;
  align-items: center;
  gap: 4px; /* Reduced gap */
}

.device-icon {
  font-size: 18px;
  /* Removed background property */
}

.device-name {
  font-weight: 600; /* Made bold */
  color: #303133;
  font-size: 14px;
}

.ip-text {
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #909399; /* Changed to light gray */
}

.text-muted {
  color: #c0c4cc;
  font-size: 12px;
}

.pagination-bar {
  display: flex;
  justify-content: flex-end;
  padding: 12px 0 0;
}

:deep(.device-dialog) {
  border-radius: 0;
}

:deep(.el-table) {
  --el-table-border-color: #ebeef5;
}

:deep(.el-table th) {
  background-color: #f5f7fa;
  font-weight: 600;
  color: #303133;
  font-size: 13px;
}

:deep(.el-table td) {
  font-size: 13px;
}

:deep(.el-tag) {
  border-radius: 0;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
}

.action-btn {
  width: 28px;
  height: 28px;
  border-radius: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn :deep(.el-icon) {
  font-size: 14px;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-connect:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-refresh:hover {
  background-color: #f0f9eb;
  color: #67c23a;
}

.action-edit:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

.table-wrapper {
  background: #fff;
  flex: 1;
}
</style>
