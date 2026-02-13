<template>
  <div class="cloud-accounts-page">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Cloudy /></el-icon>
        </div>
        <div>
          <h2 class="page-title">云账号管理</h2>
          <p class="page-subtitle">管理云平台账号，用于导入云主机</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button @click="handleAdd" class="black-button">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增账号
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.keyword"
          placeholder="搜索账号名称..."
          clearable
          class="search-input"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="searchForm.provider"
          placeholder="云厂商"
          clearable
          class="search-input"
        >
          <el-option label="全部" value="" />
          <el-option label="阿里云" value="aliyun" />
          <el-option label="腾讯云" value="tencent" />
          <el-option label="AWS" value="aws" />
          <el-option label="京东云" value="jdcloud" />
          <el-option label="百度云" value="baidu" />
          <el-option label="金山云" value="ksyun" />
        </el-select>

        <el-select
          v-model="searchForm.status"
          placeholder="状态"
          clearable
          class="search-input"
        >
          <el-option label="全部" value="" />
          <el-option label="启用" :value="1" />
          <el-option label="禁用" :value="0" />
        </el-select>
      </div>

      <div class="search-actions">
        <el-button class="reset-btn" @click="handleReset">
          <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <!-- 账号列表 -->
    <div class="table-wrapper">
      <el-table :data="filteredAccountList" v-loading="loading" stripe class="modern-table" :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }">
        <el-table-column label="账号名称" prop="name" min-width="150" />
        <el-table-column label="云厂商" align="center" width="120">
          <template #default="{ row }">
            <el-tag :type="getProviderType(row.provider)">
              {{ row.providerText }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="区域" prop="region" min-width="120">
          <template #default="{ row }">
            <span v-if="row.region">{{ row.region }}</span>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="Access Key" min-width="200">
          <template #default="{ row }">
            <span class="access-key">************</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" align="center" width="80">
          <template #default="{ row }">
            <el-switch
              v-model="row.status"
              :active-value="1"
              :inactive-value="0"
              @change="handleStatusChange(row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createTime" width="180" />
        <el-table-column label="操作" width="120" align="center" fixed="right">
          <template #default="{ row }">
            <el-button
              link
              :type="row.status === 1 ? 'primary' : 'info'"
              :disabled="row.status === 0"
              @click="handleImportHost(row)"
              title="导入主机"
            >
              <el-icon><Upload /></el-icon>
            </el-button>
            <el-button link type="primary" @click="handleEdit(row)" title="编辑">
              <el-icon><Edit /></el-icon>
            </el-button>
            <el-button link type="danger" @click="handleDelete(row)" title="删除">
              <el-icon><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑云账号' : '新增云账号'"
      width="70%"
      :style="{ maxWidth: '800px' }"
      @close="handleDialogClose"
      class="account-dialog"
    >
      <div class="dialog-content">
        <el-form :model="form" :rules="rules" ref="formRef" label-width="120px" class="account-form">
          <!-- 云厂商选择 -->
          <el-form-item label="云厂商" prop="provider" required>
            <div class="provider-options-inline">
              <div
                v-for="provider in providers"
                :key="provider.value"
                :class="['provider-chip', `provider-chip--${provider.value}`, { active: form.provider === provider.value }]"
                @click="form.provider = provider.value"
              >
                <svg v-if="provider.value === 'aliyun'" class="provider-svg" viewBox="0 0 24 24"><path d="M3.996 6.34C1.794 7.846.552 10.394.552 12.001c0 1.606 1.242 4.154 3.444 5.66h3.09l-.876-1.822c-1.542-1.002-2.598-2.592-2.598-3.838s1.056-2.836 2.598-3.838l.876-1.822h-3.09zm16.008 0h-3.09l.876 1.822c1.542 1.002 2.598 2.592 2.598 3.838s-1.056 2.836-2.598 3.838l-.876 1.822h3.09c2.202-1.506 3.444-4.054 3.444-5.66 0-1.607-1.242-4.155-3.444-5.66zm-5.466 2.188H9.462L8.442 12l1.02 3.472h5.076L15.558 12l-1.02-3.472z" fill="currentColor"/></svg>
                <svg v-else-if="provider.value === 'tencent'" class="provider-svg" viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 3c1.66 0 3 1.34 3 3s-1.34 3-3 3-3-1.34-3-3 1.34-3 3-3zm0 14.2c-2.5 0-4.71-1.28-6-3.22.03-1.99 4-3.08 6-3.08 1.99 0 5.97 1.09 6 3.08-1.29 1.94-3.5 3.22-6 3.22z" fill="currentColor"/></svg>
                <svg v-else-if="provider.value === 'aws'" class="provider-svg" viewBox="0 0 24 24"><path d="M7.164 11.61c0 .343.036.621.1.822.072.2.164.422.286.658a.39.39 0 01.064.207c0 .093-.057.186-.179.279l-.593.393a.442.442 0 01-.243.086c-.093 0-.186-.043-.279-.122a2.879 2.879 0 01-.336-.436 7.23 7.23 0 01-.286-.55c-.721.85-1.629 1.279-2.721 1.279-.779 0-1.4-.222-1.857-.665-.457-.443-.69-1.036-.69-1.779 0-.786.279-1.422.843-1.9.564-.479 1.314-.714 2.264-.714.314 0 .636.022.972.064.336.043.679.107 1.036.186v-.672c0-.707-.15-1.2-.443-1.486-.3-.286-.807-.422-1.529-.422-.329 0-.664.036-1.007.114a7.41 7.41 0 00-1.007.307 2.68 2.68 0 01-.329.114.574.574 0 01-.15.029c-.129 0-.193-.093-.193-.286V6.33c0-.15.021-.257.071-.322a.758.758 0 01.279-.157c.329-.164.721-.3 1.179-.407A5.7 5.7 0 014.85 5.3c1.079 0 1.864.243 2.364.729.493.486.743 1.222.743 2.214v2.914l-.793.454zm-3.757 1.4c.3 0 .614-.057.95-.164.336-.107.636-.307.886-.586a1.42 1.42 0 00.307-.557c.057-.214.1-.471.1-.771v-.372a7.38 7.38 0 00-.829-.143 6.74 6.74 0 00-.843-.057c-.621 0-1.079.121-1.386.372-.307.25-.45.6-.45 1.057 0 .422.107.736.329.95.214.221.529.329.936.329v.007-.065zm7.436 1.007c-.171 0-.286-.029-.357-.093-.071-.057-.136-.186-.186-.364l-2.079-6.843a1.661 1.661 0 01-.079-.379c0-.15.079-.236.236-.236h.921c.179 0 .3.029.364.093.071.057.129.186.179.364l1.486 5.857 1.379-5.857c.043-.186.1-.307.171-.364a.618.618 0 01.372-.093h.75c.179 0 .3.029.372.093.071.057.136.186.171.364l1.393 5.929 1.529-5.929c.05-.186.114-.307.179-.364a.588.588 0 01.364-.093h.871c.157 0 .243.079.243.236 0 .043-.007.093-.021.15a1.335 1.335 0 01-.064.236l-2.136 6.843c-.05.186-.114.307-.186.364-.071.057-.2.093-.35.093h-.807c-.179 0-.3-.029-.372-.093-.071-.064-.136-.186-.171-.372l-1.371-5.707-1.364 5.7c-.043.186-.1.307-.171.372-.071.064-.2.093-.379.093h-.807l.003-.005zM21.9 14.12c-.486 0-.971-.057-1.443-.171-.471-.114-.843-.236-1.1-.371-.157-.086-.264-.179-.307-.264a.666.666 0 01-.064-.279v-.457c0-.193.071-.286.207-.286a.517.517 0 01.164.029c.057.021.143.057.236.093a5.15 5.15 0 001.043.329c.379.071.743.107 1.114.107.593 0 1.05-.1 1.364-.307.314-.207.479-.507.479-.886 0-.264-.086-.479-.257-.657-.171-.179-.493-.336-.957-.486l-1.371-.429c-.693-.214-1.207-.536-1.529-.964-.321-.422-.486-.893-.486-1.4 0-.407.086-.764.264-1.079.179-.314.414-.586.707-.807.293-.229.629-.393 1.014-.514A4.354 4.354 0 0121.1 5.3c.243 0 .493.014.75.05.25.036.486.079.707.136.207.064.407.129.593.2.186.079.329.157.429.236.143.093.243.186.293.286.05.093.079.214.079.364v.422c0 .193-.071.293-.207.293-.071 0-.186-.036-.336-.107a4.574 4.574 0 00-1.929-.386c-.536 0-.964.079-1.264.25-.3.171-.45.429-.45.793 0 .264.093.486.279.664.186.179.529.357 1.021.514l1.343.429c.686.214 1.186.514 1.493.9.307.386.457.829.457 1.321 0 .414-.086.793-.257 1.121a2.62 2.62 0 01-.714.836 3.179 3.179 0 01-1.079.529 4.453 4.453 0 01-1.379.2z" fill="currentColor"/></svg>
                <svg v-else-if="provider.value === 'jdcloud'" class="provider-svg" viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 15H8V7h3v10zm6 0h-3V9h3v8z" fill="currentColor"/></svg>
                <svg v-else-if="provider.value === 'baidu'" class="provider-svg" viewBox="0 0 24 24"><path d="M5.927 12.497c2.063-.443 1.782-2.909 1.72-3.308-.082-.53-.453-1.967-1.908-1.835-1.752.158-1.678 2.396-1.678 2.396-.063 1.076.92 2.955 1.866 2.747zm2.182 4.786c-.083.32-.048.687.245.958.49.453 1.396.296 1.396.296h1.21v-2.21H9.28c-.652.012-1.108.544-1.172.956zm2.184-8.397c1.218 0 2.205-1.17 2.205-2.613C12.498 4.898 11.51 3 10.293 3 9.074 3 8.087 4.898 8.087 6.273c0 1.444.988 2.613 2.206 2.613zm5.003-2.2c1.455.132 1.826-1.305 1.907-1.834.062-.399.344-2.866-1.72-3.309-.946-.208-1.928 1.672-1.865 2.748 0 0-.075 2.238 1.678 2.396zm3.392 5.399c-1.497-1.38-2.97-.498-2.97-.498l-1.675.93c-.576.317-1.236.445-1.9.366-1.002-.122-1.94-.682-1.94-.682-1.47-.825-2.506-.054-2.506-.054-.51.327-.878.816-1.027 1.394-.37 1.444.534 2.962.534 2.962.384.655.863 1.239 1.426 1.733 1.07.951 2.315 1.12 2.315 1.12 1.51.331 2.413-.085 2.413-.085 1.092-.5 1.504-1.09 1.504-1.09 1.553-1.795.558-3.568.558-3.568.392-.63 1.27-.395 1.27-.395.803.256 1.158.13 1.158.13.633-.246.84-1.163.84-1.163.09-.714-.007-1.1-.007-1.1zm-5.065-2.252c.644-.24 1.477.453 1.477.453 1.09.78 1.884.55 1.884.55 1.283-.168 1.272-1.81 1.272-1.81.017-1.166-.583-1.72-.583-1.72-.838-.844-1.52-.387-1.52-.387-.68.37-1.38-.064-1.38-.064-1.58-.94-2.574.146-2.574.146-1.22 1.4.218 2.21.218 2.21.384.495.72.63 1.205.622h.501z" fill="currentColor"/></svg>
                <svg v-else-if="provider.value === 'ksyun'" class="provider-svg" viewBox="0 0 24 24"><path d="M12 2L2 7v10l10 5 10-5V7L12 2zm0 2.18l7.12 3.56L12 11.31 4.88 7.74 12 4.18zM4 8.96l7 3.5v6.58l-7-3.5V8.96zm9 10.08V12.46l7-3.5v6.58l-7 3.5z" fill="currentColor"/></svg>
                <span class="provider-chip-label">{{ provider.label }}</span>
              </div>
            </div>
          </el-form-item>

          <el-form-item label="账号名称" prop="name">
            <el-input v-model="form.name" placeholder="如：生产环境阿里云账号" />
          </el-form-item>

          <template v-if="!isEdit">
            <el-form-item label="Access Key" prop="accessKey">
              <el-input v-model="form.accessKey" placeholder="请输入 Access Key ID" />
            </el-form-item>

            <el-form-item label="Secret Key" prop="secretKey">
              <el-input v-model="form.secretKey" type="password" show-password placeholder="请输入 Access Key Secret" />
            </el-form-item>
          </template>

          <el-alert v-else type="info" :closable="false" style="margin-bottom: 20px;">
            <template #title>
              <span style="font-size: 13px;">如需修改 Access Key 或 Secret Key，请删除后重新创建账号</span>
            </template>
          </el-alert>

          <el-form-item label="默认区域">
            <el-select v-model="form.region" placeholder="选择默认区域" filterable style="width: 100%">
              <el-option v-for="region in currentRegions" :key="region.value" :label="region.label" :value="region.value" />
            </el-select>
          </el-form-item>

          <el-form-item label="备注">
            <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选，填写备注信息" />
          </el-form-item>

          <el-form-item label="状态">
            <el-radio-group v-model="form.status">
              <el-radio :value="1">启用</el-radio>
              <el-radio :value="0">禁用</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmit" :loading="submitting">
            {{ isEdit ? '保存修改' : '创建账号' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 导入云主机对话框 -->
    <el-dialog
      v-model="importDialogVisible"
      title="导入云主机"
      width="70%"
      :style="{ maxWidth: '1200px' }"
      @close="handleImportDialogClose"
      class="import-dialog"
    >
      <el-form :model="importForm" label-width="100px" class="import-form">
        <div class="form-row">
          <el-form-item label="云账号" class="form-item-full">
            <el-select v-model="importForm.accountId" placeholder="请选择云账号" style="width: 100%" @change="handleAccountChange">
              <el-option v-for="acc in enabledAccountList" :key="acc.id" :label="acc.name" :value="acc.id">
                <span>{{ acc.name }}</span>
                <el-tag :type="getProviderType(acc.provider)" size="small" style="margin-left: 8px;">
                  {{ acc.providerText }}
                </el-tag>
              </el-option>
            </el-select>
          </el-form-item>
        </div>
        <div class="form-row">
          <el-form-item label="区域" class="form-item-half">
            <el-select v-model="importForm.region" placeholder="请选择区域" style="width: 100%" filterable>
              <el-option v-for="region in regions" :key="region.value" :label="region.label" :value="region.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="所属分组" class="form-item-half">
            <el-tree-select
              v-model="importForm.groupId"
              :data="groupTreeOptions"
              :props="{ value: 'id', label: 'name', children: 'children' }"
              clearable
              check-strictly
              placeholder="请选择分组"
              style="width: 100%"
            />
          </el-form-item>
        </div>
      </el-form>

      <div v-loading="loadingInstances" class="instances-container">
        <el-alert v-if="!selectedAccount" title="请先选择云账号" type="info" :closable="false" />
        <el-alert v-else-if="!importForm.region" title="请选择区域" type="info" :closable="false" />
        <div v-else-if="cloudHosts.length === 0" class="empty-instances">
          <el-empty description="该区域下没有可导入的云主机" />
        </div>
        <div v-else class="instances-list">
          <div class="instances-header">
            <div class="instances-info">
              <span class="instances-count">找到 <strong>{{ cloudHosts.length }}</strong> 台云主机</span>
              <span class="instances-region">当前区域: {{ importForm.region }}</span>
            </div>
            <el-checkbox v-model="selectAll" @change="handleSelectAll" size="large">
              <span class="select-all-text">全选</span>
            </el-checkbox>
          </div>
          <el-table
            ref="cloudHostsTableRef"
            :data="cloudHosts"
            @selection-change="handleSelectionChange"
            :max-height="400"
            class="cloud-hosts-table"
            :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
            stripe
          >
            <el-table-column type="selection" width="50" align="center" />
            <el-table-column label="实例名称" prop="name" min-width="160" show-overflow-tooltip>
              <template #default="{ row }">
                <div class="instance-name">
                  <el-icon class="instance-icon"><Monitor /></el-icon>
                  <span>{{ row.name }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="实例ID" prop="instanceId" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">
                <span class="instance-id">{{ row.instanceId }}</span>
              </template>
            </el-table-column>
            <el-table-column label="IP地址" min-width="150">
              <template #default="{ row }">
                <div class="ip-list">
                  <div v-if="row.publicIp" class="ip-item public-ip">
                    <el-tag size="small" type="success">公</el-tag>
                    <span>{{ row.publicIp }}</span>
                  </div>
                  <div v-if="row.privateIp" class="ip-item private-ip">
                    <el-tag size="small" type="info">私</el-tag>
                    <span>{{ row.privateIp }}</span>
                  </div>
                  <span v-if="!row.publicIp && !row.privateIp" class="text-muted">-</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="操作系统" prop="os" min-width="120" show-overflow-tooltip />
            <el-table-column label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="getStatusType(row.status)" size="small">
                  {{ getStatusText(row.status) }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <template #footer>
        <el-button @click="importDialogVisible = false" size="large">取消</el-button>
        <el-button type="primary" @click="handleConfirmImport" :loading="importing" :disabled="selectedInstances.length === 0" size="large">
          <el-icon v-if="!importing"><Upload /></el-icon>
          <span>导入 {{ selectedInstances.length > 0 ? `(${selectedInstances.length})` : '' }}</span>
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  Edit,
  Delete,
  Upload,
  Cloudy,
  Search,
  RefreshLeft,
  Monitor
} from '@element-plus/icons-vue'
import {
  getCloudAccounts,
  createCloudAccount,
  updateCloudAccount,
  deleteCloudAccount,
  importFromCloud,
  getCloudInstances,
  getCloudRegions
} from '@/api/host'
import { getGroupTree } from '@/api/assetGroup'

const loading = ref(false)
const accountList = ref<any[]>([])

// 搜索表单
const searchForm = reactive({
  keyword: '',
  provider: '',
  status: ''
})

// 过滤后的账号列表
const filteredAccountList = computed(() => {
  return accountList.value.filter((account: any) => {
    const matchKeyword = !searchForm.keyword || account.name?.toLowerCase().includes(searchForm.keyword.toLowerCase())
    const matchProvider = !searchForm.provider || account.provider === searchForm.provider
    const matchStatus = searchForm.status === '' || account.status === parseInt(searchForm.status)
    return matchKeyword && matchProvider && matchStatus
  })
})

// 重置搜索
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.provider = ''
  searchForm.status = ''
}

// 对话框相关
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref()

// 云厂商选项
const providers = [
  { value: 'aliyun', label: '阿里云', short: '阿里' },
  { value: 'tencent', label: '腾讯云', short: '腾讯' },
  { value: 'aws', label: 'AWS', short: 'AWS' },
  { value: 'jdcloud', label: '京东云', short: '京东' },
  { value: 'baidu', label: '百度云', short: '百度' },
  { value: 'ksyun', label: '金山云', short: '金山' }
]

// 当前厂商的区域列表（新增/编辑对话框用）
const currentRegions = ref<any[]>([])

const form = reactive({
  id: 0,
  name: '',
  provider: 'aliyun',
  accessKey: '',
  secretKey: '',
  region: '',
  description: '',
  status: 1
})

// 动态验证规则
const rules = computed(() => {
  const baseRules: any = {
    name: [{ required: true, message: '请输入账号名称', trigger: 'blur' }],
    provider: [{ required: true, message: '请选择云厂商', trigger: 'change' }]
  }

  // 只有新增时才验证 Access Key 和 Secret Key
  if (!isEdit.value) {
    baseRules.accessKey = [{ required: true, message: '请输入Access Key', trigger: 'blur' }]
    baseRules.secretKey = [{ required: true, message: '请输入Secret Key', trigger: 'blur' }]
  }

  return baseRules
})

// 获取启用的云账号列表
const enabledAccountList = computed(() => {
  return accountList.value.filter((a: any) => a.status === 1)
})

// 导入相关
const importDialogVisible = ref(false)
const importing = ref(false)
const loadingInstances = ref(false)
const selectedAccount = ref<any>(null)
const cloudHosts = ref<any[]>([])
const selectedInstances = ref<string[]>([])
const selectAll = ref(false)
const groupTreeOptions = ref<any[]>([])
const cloudHostsTableRef = ref()

const importForm = reactive({
  accountId: null as number | null,
  region: '',
  groupId: null as number | null
})

const regions = ref<any[]>([])

// 获取云厂商类型
const getProviderType = (provider: string) => {
  const typeMap: Record<string, string> = {
    aliyun: 'warning',
    tencent: 'info',
    jdcloud: 'danger',
    aws: '',
    baidu: 'success',
    ksyun: 'info'
  }
  return typeMap[provider] || ''
}

// 掩码Access Key
const maskAccessKey = (key: string) => {
  if (!key || key.length <= 8) return key
  return key.substring(0, 4) + '****' + key.substring(key.length - 4)
}

// 加载账号列表
const loadAccountList = async () => {
  loading.value = true
  try {
    const res = await getCloudAccounts()
    // getCloudAccounts 返回的是数组，不是 { list: [] }
    accountList.value = Array.isArray(res) ? res : []
  } catch (error) {
  } finally {
    loading.value = false
  }
}

// 加载分组树
const loadGroupTree = async () => {
  try {
    const res = await getGroupTree()
    groupTreeOptions.value = res || []
  } catch (error) {
  }
}

// 新增
const handleAdd = () => {
  Object.assign(form, {
    id: 0,
    name: '',
    provider: 'aliyun',
    accessKey: '',
    secretKey: '',
    region: '',
    description: '',
    status: 1
  })
  isEdit.value = false
  // 初始化区域列表
  currentRegions.value = getLocalRegions('aliyun')
  // 清除之前的验证
  nextTick(() => {
    formRef.value?.clearValidate()
  })
  dialogVisible.value = true
}

// 编辑
const handleEdit = async (row: any) => {
  isEdit.value = true // 先设置编辑状态

  Object.assign(form, {
    id: row.id,
    name: row.name,
    provider: row.provider,
    accessKey: '',
    secretKey: '',
    region: row.region || '',
    description: row.description || '',
    status: row.status
  })

  // 加载该账号的区域列表
  try {
    const res = await getCloudRegions(row.id)
    currentRegions.value = Array.isArray(res) ? res : []
  } catch (error) {
    currentRegions.value = getLocalRegions(row.provider)
  }

  // 清除之前的验证
  nextTick(() => {
    formRef.value?.clearValidate()
  })
  dialogVisible.value = true
}

// 删除
const handleDelete = (row: any) => {
  ElMessageBox.confirm(`确定要删除云账号"${row.name}"吗？`, '提示', {
    type: 'warning'
  }).then(async () => {
    try {
      await deleteCloudAccount(row.id)
      ElMessage.success('删除成功')
      loadAccountList()
    } catch (error: any) {
      ElMessage.error(error.message || '删除失败')
    }
  })
}

// 状态切换
const handleStatusChange = async (row: any) => {
  try {
    await updateCloudAccount(row.id, {
      id: row.id,
      name: row.name,
      provider: row.provider,
      region: row.region || '',
      description: row.description || '',
      status: row.status
    })
    ElMessage.success('状态更新成功')
  } catch (error: any) {
    // 恢复原状态
    row.status = row.status === 1 ? 0 : 1
    ElMessage.error(error.message || '状态更新失败')
  }
}

// 提交表单
const handleSubmit = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (isEdit.value) {
      await updateCloudAccount(form.id, form)
      ElMessage.success('更新成功')
    } else {
      await createCloudAccount(form)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadAccountList()
  } catch (error: any) {
    ElMessage.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

// 对话框关闭
const handleDialogClose = () => {
  formRef.value?.resetFields()
}

// 导入主机
const handleImportHost = (row: any) => {
  if (row.status === 0) {
    ElMessage.warning('该账号已禁用，无法导入主机')
    return
  }
  selectedAccount.value = row
  importForm.accountId = row.id
  importForm.region = row.region || ''
  importForm.groupId = null
  handleAccountChange()
  importDialogVisible.value = true
}

// 账号变化
const handleAccountChange = async () => {
  const account = accountList.value.find((a: any) => a.id === importForm.accountId)
  if (!account) return

  selectedAccount.value = account

  // 清空主机列表和区域
  cloudHosts.value = []
  selectedInstances.value = []
  selectAll.value = false
  regions.value = []
  importForm.region = ''

  // 从云API获取区域列表
  try {
    const res = await getCloudRegions(account.id)
    regions.value = Array.isArray(res) ? res : []

    // 设置默认区域（如果账号有默认区域且该区域在列表中）
    if (account.region) {
      const hasDefaultRegion = regions.value.some((r: any) => r.value === account.region)
      if (hasDefaultRegion) {
        importForm.region = account.region
      }
    }
  } catch (error: any) {
    ElMessage.error(error.message || '加载区域列表失败')
  }
}

// 加载云主机实例列表
const loadCloudInstances = async () => {
  if (!importForm.accountId || !importForm.region) return

  loadingInstances.value = true
  try {
    const res = await getCloudInstances(importForm.accountId, importForm.region)
    cloudHosts.value = Array.isArray(res) ? res : []
  } catch (error: any) {
    ElMessage.error(error.message || '加载云主机列表失败')
    cloudHosts.value = []
  } finally {
    loadingInstances.value = false
  }
}

// 监听区域变化，自动加载实例列表
watch(() => importForm.region, () => {
  if (importForm.region) {
    loadCloudInstances()
  }
})

// 监听表单中的云厂商变化，更新区域列表（新增/编辑对话框用）
watch(() => form.provider, async (newProvider) => {
  if (!newProvider) return
  form.region = '' // 清空已选择的区域

  // 如果是编辑模式且有账号ID，调用云API获取区域
  if (isEdit.value && form.id > 0) {
    try {
      const res = await getCloudRegions(form.id)
      currentRegions.value = Array.isArray(res) ? res : []
    } catch (error) {
      currentRegions.value = getLocalRegions(newProvider)
    }
  } else {
    // 新增模式：使用本地常用区域列表
    currentRegions.value = getLocalRegions(newProvider)
  }
})

// 本地常用区域列表（新增账号时使用）
const getLocalRegions = (provider: string): any[] => {
  const localMap: Record<string, any[]> = {
    aliyun: [
      { value: 'cn-hangzhou', label: '华东1 (杭州)' },
      { value: 'cn-shanghai', label: '华东2 (上海)' },
      { value: 'cn-beijing', label: '华北2 (北京)' },
      { value: 'cn-shenzhen', label: '华南1 (深圳)' },
      { value: 'cn-guangzhou', label: '华南2 (广州)' },
      { value: 'cn-chengdu', label: '西南1 (成都)' }
    ],
    tencent: [
      { value: 'ap-guangzhou', label: '华南地区 (广州)' },
      { value: 'ap-shanghai', label: '华东地区 (上海)' },
      { value: 'ap-beijing', label: '华北地区 (北京)' },
      { value: 'ap-chengdu', label: '西南地区 (成都)' },
      { value: 'ap-chongqing', label: '西南地区 (重庆)' }
    ],
    aws: [
      { value: 'us-east-1', label: '美国东部 (弗吉尼亚)' },
      { value: 'us-west-2', label: '美国西部 (俄勒冈)' },
      { value: 'ap-southeast-1', label: '亚太 (新加坡)' },
      { value: 'ap-northeast-1', label: '亚太 (东京)' },
      { value: 'eu-west-1', label: '欧洲 (爱尔兰)' },
      { value: 'ap-east-1', label: '亚太 (香港)' },
      { value: 'cn-north-1', label: '中国 (北京)' },
      { value: 'cn-northwest-1', label: '中国 (宁夏)' }
    ],
    jdcloud: [
      { value: 'cn-north-1', label: '华北-北京' },
      { value: 'cn-south-1', label: '华南-广州' },
      { value: 'cn-east-1', label: '华东-宿迁' },
      { value: 'cn-east-2', label: '华东-上海' }
    ],
    baidu: [
      { value: 'bj', label: '华北-北京' },
      { value: 'gz', label: '华南-广州' },
      { value: 'su', label: '华东-苏州' },
      { value: 'hkg', label: '中国香港' },
      { value: 'bd', label: '华北-保定' },
      { value: 'fwh', label: '中南-武汉' }
    ],
    ksyun: [
      { value: 'cn-beijing-6', label: '华北1 (北京)' },
      { value: 'cn-shanghai-2', label: '华东1 (上海)' },
      { value: 'cn-guangzhou-1', label: '华南1 (广州)' },
      { value: 'cn-hongkong-2', label: '中国香港' }
    ]
  }
  return localMap[provider] || []
}

// 全选
const handleSelectAll = (checked: boolean) => {
  if (cloudHostsTableRef.value) {
    cloudHosts.value.forEach((row: any) => {
      cloudHostsTableRef.value.toggleRowSelection(row, checked)
    })
  }
}

// 获取状态类型
const getStatusType = (status: string) => {
  // 统一转为小写进行比较，兼容不同云厂商返回的状态格式
  const statusLower = status.toLowerCase()
  const typeMap: Record<string, string> = {
    'running': 'success',
    'starting': 'warning',
    'stopping': 'warning',
    'stopped': 'info',
    'deleted': 'danger'
  }
  return typeMap[statusLower] || 'info'
}

// 获取状态文本
const getStatusText = (status: string) => {
  // 统一转为小写进行比较，兼容不同云厂商返回的状态格式
  const statusLower = status.toLowerCase()
  const textMap: Record<string, string> = {
    'running': '运行中',
    'starting': '启动中',
    'stopping': '停止中',
    'stopped': '已停止',
    'deleted': '已删除'
  }
  return textMap[statusLower] || status
}

// 选择变化
const handleSelectionChange = (selection: any[]) => {
  selectedInstances.value = selection.map((s: any) => s.instanceId)
}

// 确认导入
const handleConfirmImport = async () => {
  if (!importForm.groupId) {
    ElMessage.warning('请选择所属分组')
    return
  }

  importing.value = true
  try {
    await importFromCloud({
      accountId: importForm.accountId,
      region: importForm.region,
      groupId: importForm.groupId,
      instanceIds: selectedInstances.value
    })
    ElMessage.success('导入成功')
    importDialogVisible.value = false
  } catch (error: any) {
    ElMessage.error(error.message || '导入失败')
  } finally {
    importing.value = false
  }
}

// 导入对话框关闭
const handleImportDialogClose = () => {
  cloudHosts.value = []
  selectedInstances.value = []
  selectAll.value = false
}

onMounted(() => {
  loadAccountList()
  loadGroupTree()
})
</script>

<style scoped>
.cloud-accounts-page {
  padding: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
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
  flex: 1;
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

.access-key {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #606266;
}

.text-muted {
  color: #c0c4cc;
}

/* 按钮样式 - 使用全局样式 .black-button */

/* 对话框样式 */
:deep(.account-dialog) {
  border-radius: 0;
}

:deep(.account-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.account-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.account-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

.dialog-content {
  padding: 0;
}

/* 云厂商选择器 - 紧凑芯片样式 */
.provider-options-inline {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.provider-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 6px 12px;
  border: 1.5px solid #dcdfe6;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.25s ease;
  background: #fafbfc;
  user-select: none;
  font-size: 0;
}

.provider-chip:hover {
  border-color: #b0b4bb;
  background: #f5f7fa;
}

.provider-chip.active {
  border-color: var(--chip-color, #409eff);
  background: var(--chip-bg, #ecf5ff);
}

.provider-svg {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  color: #909399;
  transition: color 0.25s ease;
}

.provider-chip.active .provider-svg {
  color: var(--chip-color, #409eff);
}

.provider-chip-label {
  font-size: 13px;
  color: #606266;
  font-weight: 500;
  line-height: 1;
}

.provider-chip.active .provider-chip-label {
  color: var(--chip-color, #409eff);
  font-weight: 600;
}

/* 各厂商颜色 */
.provider-chip--aliyun { --chip-color: #ff6a00; --chip-bg: #fff7f0; }
.provider-chip--tencent { --chip-color: #00a4ff; --chip-bg: #f0f9ff; }
.provider-chip--aws { --chip-color: #ff9900; --chip-bg: #fffaf0; }
.provider-chip--jdcloud { --chip-color: #e1251b; --chip-bg: #fff0f0; }
.provider-chip--baidu { --chip-color: #306cff; --chip-bg: #f0f4ff; }
.provider-chip--ksyun { --chip-color: #1ba784; --chip-bg: #f0faf6; }

/* 表单样式 */
.account-form :deep(.el-form-item) {
  margin-bottom: 20px;
}

.account-form :deep(.el-form-item__label) {
  font-weight: 500;
  color: #606266;
  width: 120px !important;
  white-space: nowrap;
}

.account-form :deep(.el-form-item__content) {
  flex: 1;
}

.account-form :deep(.el-input),
.account-form :deep(.el-select),
.account-form :deep(.el-textarea) {
  width: 100%;
}

.account-form :deep(.el-input__wrapper),
.account-form :deep(.el-textarea__inner) {
  border-radius: 0;
  border: 1px solid #dcdfe6;
  transition: all 0.3s ease;
}

.account-form :deep(.el-input__wrapper:hover),
.account-form :deep(.el-textarea__inner:hover) {
  border-color: #ffffff;
}

.account-form :deep(.el-input__wrapper.is-focus),
.account-form :deep(.el-textarea__inner:focus) {
  border-color: #ffffff;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.2);
}

/* 对话框底部 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.empty-instances {
  padding: 40px 0;
}

/* 导入对话框样式 */
:deep(.import-dialog) {
  border-radius: 0;
}

:deep(.import-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.import-dialog .el-dialog__body) {
  padding: 24px;
  max-height: 70vh;
  overflow-y: auto;
}

:deep(.import-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

.import-form {
  margin-bottom: 16px;
}

.form-row {
  display: flex;
  gap: 16px;
}

.form-item-full {
  flex: 1;
}

.form-item-half {
  width: 50%;
}

.instances-container {
  min-height: 200px;
}

.instances-list {
  background: #fafbfc;
  border-radius: 0;
  padding: 16px;
}

.instances-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e4e7ed;
}

.instances-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.instances-count {
  font-size: 15px;
  color: #303133;
}

.instances-count strong {
  color: #ffffff;
  font-size: 18px;
}

.instances-region {
  font-size: 12px;
  color: #909399;
}

.select-all-text {
  font-size: 14px;
  font-weight: 500;
}

.cloud-hosts-table {
  background: #fff;
  border-radius: 0;
  overflow: hidden;
}

.cloud-hosts-table :deep(.el-table__header-wrapper) {
  border-radius: 0;
}

.cloud-hosts-table :deep(.el-table__body-wrapper) {
  border-radius: 0;
}

.cloud-hosts-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
}

.cloud-hosts-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.instance-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.instance-icon {
  color: #ffffff;
  font-size: 16px;
}

.instance-id {
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 12px;
  color: #606266;
  background: #f5f7fa;
  padding: 2px 8px;
  border-radius: 0;
}

.ip-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ip-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}

.ip-item span {
  font-family: 'Monaco', 'Menlo', monospace;
}

.public-ip span {
  color: #67c23a;
}

.private-ip span {
  color: #909399;
}
</style>
