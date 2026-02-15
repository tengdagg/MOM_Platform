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

    <!-- 搜索和筛选 -->
    <div class="filter-bar">
      <el-input
        v-model="searchKeyword"
        placeholder="搜索设备名称、IP、品牌型号、SN..."
        clearable
        style="width: 300px"
        @clear="loadDevices"
        @keyup.enter="loadDevices"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <el-select v-model="filterDeviceType" placeholder="设备类型" clearable style="width: 140px" @change="loadDevices">
        <el-option v-for="dt in deviceTypes" :key="dt.value" :label="dt.label" :value="dt.value" />
      </el-select>
      <el-select v-model="filterProtocol" placeholder="连接协议" clearable style="width: 140px" @change="loadDevices">
        <el-option label="SSH" value="ssh" />
        <el-option label="Telnet" value="telnet" />
      </el-select>
      <el-select v-model="filterGroupId" placeholder="分组" clearable style="width: 160px" @change="loadDevices">
        <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
      </el-select>
      <el-button @click="loadDevices" type="primary" plain>
        <el-icon><Search /></el-icon>
      </el-button>
    </div>

    <!-- 设备表格 -->
    <div class="table-wrapper">
      <el-table
        :data="deviceList"
        v-loading="loading"
        stripe
        style="width: 100%"
        @row-dblclick="handleConnect"
      >
        <el-table-column label="设备名称" prop="name" min-width="140">
          <template #default="{ row }">
            <div class="device-name-cell">
              <el-icon class="device-icon" :style="{ color: getDeviceTypeColor(row.deviceType) }">
                <component :is="getDeviceTypeIcon(row.deviceType)" />
              </el-icon>
              <span>{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="IP 地址" prop="ip" min-width="130">
          <template #default="{ row }">
            <span class="ip-text">{{ row.ip }}</span>
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
        <el-table-column label="SN" prop="serialNumber" min-width="140" show-overflow-tooltip />
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
        <el-table-column label="分组" min-width="100">
          <template #default="{ row }">
            <span v-if="row.group">{{ row.group.name }}</span>
            <span v-else class="text-muted">未分组</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.status === 1" type="success" size="small" effect="plain">在线</el-tag>
            <el-tag v-else-if="row.status === 0" type="danger" size="small" effect="plain">离线</el-tag>
            <el-tag v-else type="info" size="small" effect="plain">未知</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right" align="center">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="handleConnect(row)">
              <el-icon><Connection /></el-icon> 连接
            </el-button>
            <el-button type="success" link size="small" @click="handleTest(row)">
              <el-icon><CircleCheck /></el-icon> 测试
            </el-button>
            <el-button type="warning" link size="small" @click="handleEdit(row)">
              <el-icon><Edit /></el-icon> 编辑
            </el-button>
            <el-button type="danger" link size="small" @click="handleDelete(row)">
              <el-icon><Delete /></el-icon> 删除
            </el-button>
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
          <el-select v-model="form.groupId" placeholder="选择分组" clearable style="width: 100%">
            <el-option v-for="g in groups" :key="g.id" :label="g.name" :value="g.id" />
          </el-select>
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
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import {
  Plus, Search, Monitor, SetUp, Edit, Delete, Connection,
  CircleCheck
} from '@element-plus/icons-vue'
import {
  getNetworkDeviceList,
  createNetworkDevice,
  updateNetworkDevice,
  deleteNetworkDevice,
  testNetworkDeviceConnection,
  getCredentials
} from '@/api/host'
import { getGroupTree } from '@/api/assetGroup'
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
const filterGroupId = ref<number | ''>('')
const credentials = ref<any[]>([])
const groups = ref<any[]>([])

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
  loadDevices()
  loadCredentials()
  loadGroups()
})

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
    if (filterGroupId.value) params.groupId = filterGroupId.value

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

async function loadCredentials() {
  try {
    const res: any = await getCredentials('network')
    credentials.value = Array.isArray(res) ? res : []
  } catch (e) { /* ignore */ }
}

async function loadGroups() {
  try {
    const res: any = await getGroupTree('network')
    groups.value = flattenGroups(Array.isArray(res) ? res : [])
  } catch (e) { /* ignore */ }
}

function flattenGroups(tree: any[], result: any[] = []): any[] {
  for (const node of tree) {
    result.push({ id: node.id, name: node.name })
    if (node.children && node.children.length > 0) {
      flattenGroups(node.children, result)
    }
  }
  return result
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
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
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

.filter-bar {
  display: flex;
  gap: 10px;
  padding: 10px 16px;
  background: #fff;
  margin-bottom: 10px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.03);
  align-items: center;
  flex-wrap: wrap;
}

.table-wrapper {
  background: #fff;
  padding: 0 16px 16px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.03);
}

.device-name-cell {
  display: flex;
  align-items: center;
  gap: 6px;
}

.device-icon {
  font-size: 16px;
}

.ip-text {
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #409EFF;
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
</style>
