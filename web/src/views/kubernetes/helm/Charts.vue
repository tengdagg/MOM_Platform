<template>
  <div class="charts-list">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-bar-left">
        <el-input
          v-model="searchQuery"
          placeholder="搜索 Chart 名称..."
          clearable
          class="search-input"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select v-model="selectedRepo" placeholder="选择仓库" clearable @change="fetchCharts" class="filter-select">
          <el-option v-for="repo in repos" :key="repo.ID" :label="repo.name" :value="repo.ID" />
        </el-select>
      </div>

      <div class="search-bar-right">
        <el-button class="black-button" @click="showAddRepoDialog">
          <el-icon style="margin-right: 4px;"><Plus /></el-icon>
          添加仓库
        </el-button>
        <el-button :icon="Setting" @click="repoDrawerVisible = true" title="管理仓库" />
      </div>
    </div>

    <!-- Charts 网格 -->
    <div class="table-wrapper">
      <div :class="['charts-grid', { 'is-empty': !loading && filteredCharts.length === 0 }]" v-loading="loading">
        <el-empty v-if="!loading && filteredCharts.length === 0" description="暂无 Chart 或未选择仓库" />
        <el-card v-for="chart in filteredCharts" :key="chart.name" class="chart-card" shadow="hover" @click="openChartDetail(chart)">
          <div class="chart-icon">
            <img :src="chart.icon || defaultIcon" @error="handleImageError" alt="icon" />
          </div>
          <div class="chart-info">
            <h3 class="chart-name">{{ chart.name }}</h3>
            <p class="chart-desc" :title="chart.description">{{ chart.description }}</p>
            <div class="chart-meta">
              <el-tag size="small">{{ chart.version }}</el-tag>
              <span class="app-version">App: {{ chart.appVersion }}</span>
            </div>
          </div>
        </el-card>
      </div>
    </div>

    <!-- Add/Edit Repo Dialog -->
    <el-dialog v-model="dialogVisible" :title="isEditMode ? '编辑 Helm 仓库' : '添加 Helm 仓库'" width="500px" @closed="resetForm">
      <el-form :model="repoForm" label-width="100px">
        <el-form-item label="名称" required>
          <el-input v-model="repoForm.name" placeholder="仓库名称" :disabled="isEditMode" />
        </el-form-item>
        <el-form-item label="URL" required>
          <el-input v-model="repoForm.url" placeholder="仓库地址" />
        </el-form-item>
        
        <el-collapse>
          <el-collapse-item title="高级设置" name="1">
            <el-form-item label="安全">
              <el-checkbox v-model="repoForm.insecureSkipTLSVerify">跳过 TLS 验证</el-checkbox>
            </el-form-item>
            <div class="credentials-section">
              <h4>仓库认证</h4>
              <el-form-item label="用户名">
                <el-input v-model="repoForm.username" />
              </el-form-item>
              <el-form-item label="密码">
                <el-input v-model="repoForm.password" type="password" show-password />
              </el-form-item>
              <el-form-item label="CertFile">
                <el-input v-model="repoForm.certFile" type="textarea" :rows="3" placeholder="客户端证书内容" />
              </el-form-item>
              <el-form-item label="KeyFile">
                <el-input v-model="repoForm.keyFile" type="textarea" :rows="3" placeholder="客户端密钥内容" />
              </el-form-item>
              <el-form-item label="CAFile">
                <el-input v-model="repoForm.caFile" type="textarea" :rows="3" placeholder="CA 证书内容" />
              </el-form-item>
            </div>
          </el-collapse-item>
        </el-collapse>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitRepo" :loading="submitting" class="black-button">确定</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- Repo Management Drawer -->
    <el-drawer v-model="repoDrawerVisible" title="仓库管理" size="600px">
      <div class="repo-list">
        <el-table :data="repos" style="width: 100%" class="modern-table">
          <el-table-column prop="name" label="名称" width="120" />
          <el-table-column prop="url" label="URL" min-width="200" show-overflow-tooltip />
          <el-table-column label="操作" width="120" align="center">
            <template #default="{ row }">
              <el-button link type="primary" @click="editRepo(row)">
                <el-icon><Edit /></el-icon>
              </el-button>
              <el-button link type="danger" @click="deleteRepoById(row.ID)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-drawer>

    <!-- Chart Detail Drawer -->
    <el-drawer v-model="chartDrawerVisible" :title="selectedChart?.name || 'Chart 详情'" size="500px">
      <div v-if="selectedChart" class="chart-detail">
        <div class="chart-header">
          <img :src="selectedChart.icon || defaultIcon" @error="handleImageError" class="chart-detail-icon" />
          <div class="chart-header-info">
            <h2>{{ selectedChart.name }}</h2>
            <p>{{ selectedChart.description }}</p>
          </div>
        </div>

        <el-divider />

        <el-form label-width="100px" class="install-form">
          <el-form-item label="Release 名称" required>
            <el-input v-model="installForm.releaseName" placeholder="my-release" />
          </el-form-item>
          
          <el-form-item label="版本" required>
            <el-select v-model="installForm.version" placeholder="选择版本" style="width: 100%;" v-loading="versionsLoading">
              <el-option v-for="v in chartVersions" :key="v.version" :label="`${v.version} (App: ${v.appVersion})`" :value="v.version" />
            </el-select>
          </el-form-item>

          <el-form-item label="命名空间" required>
            <el-select v-model="installForm.namespace" placeholder="选择命名空间" style="width: 100%;" filterable allow-create v-loading="namespacesLoading">
              <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
            </el-select>
          </el-form-item>

          <el-form-item label="Values (YAML)">
            <el-input v-model="installForm.values" type="textarea" :rows="10" placeholder="# 自定义 values (可选)" style="font-family: monospace;" />
          </el-form-item>

          <el-form-item>
            <el-button type="primary" @click="installChart" :loading="installing" :disabled="!props.clusterId" class="black-button">
              安装
            </el-button>
            <el-text v-if="!props.clusterId" type="warning" style="margin-left: 10px;">请先选择集群</el-text>
          </el-form-item>
        </el-form>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Plus, Delete, Setting, Search, Edit } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import request from '@/utils/request'
import { getNamespaces } from '@/api/kubernetes'

const props = defineProps<{
  clusterId: number | undefined
}>()

const defaultIcon = 'https://helm.sh/img/helm.svg'
const handleImageError = (e: Event) => {
  (e.target as HTMLImageElement).src = defaultIcon
}

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const repoDrawerVisible = ref(false)
const repos = ref<any[]>([])
const selectedRepo = ref<number | undefined>(undefined)
const charts = ref<any[]>([])
const searchQuery = ref('')
const isEditMode = ref(false)
const editingRepoId = ref<number | null>(null)

// 过滤后的 Charts
const filteredCharts = computed(() => {
  if (!searchQuery.value) return charts.value
  return charts.value.filter(chart => 
    chart.name.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

// Chart Detail
const chartDrawerVisible = ref(false)
const selectedChart = ref<any>(null)
const chartVersions = ref<any[]>([])
const versionsLoading = ref(false)
const namespaces = ref<string[]>([])
const namespacesLoading = ref(false)
const installing = ref(false)

const installForm = ref({
  releaseName: '',
  version: '',
  namespace: 'default',
  values: ''
})

const repoForm = ref({
  name: '',
  url: '',
  username: '',
  password: '',
  certFile: '',
  keyFile: '',
  caFile: '',
  insecureSkipTLSVerify: false,
})

const fetchRepos = async () => {
  try {
    const res = await request.get('/api/v1/plugins/kubernetes/helm/repos')
    repos.value = res || []
    if (repos.value.length > 0 && !selectedRepo.value) {
      selectedRepo.value = repos.value[0].ID
      fetchCharts()
    }
  } catch (error) {
    console.error(error)
  }
}

const fetchCharts = async () => {
  if (!selectedRepo.value) {
    charts.value = []
    return
  }
  loading.value = true
  try {
    const res = await request.get(`/api/v1/plugins/kubernetes/helm/repos/${selectedRepo.value}/charts`)
    charts.value = res || []
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 Charts 失败')
  } finally {
    loading.value = false
  }
}

const showAddRepoDialog = () => {
  isEditMode.value = false
  editingRepoId.value = null
  resetForm()
  dialogVisible.value = true
}

const resetForm = () => {
  repoForm.value = {
    name: '',
    url: '',
    username: '',
    password: '',
    certFile: '',
    keyFile: '',
    caFile: '',
    insecureSkipTLSVerify: false,
  }
}

const editRepo = (repo: any) => {
  isEditMode.value = true
  editingRepoId.value = repo.ID
  repoForm.value = { ...repo }
  repoForm.value.password = '' // Don't show password
  dialogVisible.value = true
}

const submitRepo = async () => {
  if (!repoForm.value.name || !repoForm.value.url) {
    ElMessage.warning('名称和 URL 不能为空')
    return
  }
  submitting.value = true
  try {
    if (isEditMode.value && editingRepoId.value) {
      await request.put(`/api/v1/plugins/kubernetes/helm/repos/${editingRepoId.value}`, repoForm.value)
      ElMessage.success('仓库更新成功')
    } else {
      await request.post('/api/v1/plugins/kubernetes/helm/repos', repoForm.value)
      ElMessage.success('仓库添加成功')
    }
    dialogVisible.value = false
    fetchRepos()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || (isEditMode.value ? '更新失败' : '添加失败'))
  } finally {
    submitting.value = false
  }
}

const deleteRepoById = async (id: number) => {
  try {
    await ElMessageBox.confirm('确定删除该仓库?', '警告', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    
    await request.delete(`/api/v1/plugins/kubernetes/helm/repos/${id}`)
    ElMessage.success('仓库已删除')
    if (selectedRepo.value === id) {
      selectedRepo.value = undefined
      charts.value = []
    }
    fetchRepos()
  } catch (error) {
    if (error !== 'cancel') {
        console.error(error)
        ElMessage.error('删除失败')
    }
  }
}

const openChartDetail = async (chart: any) => {
  selectedChart.value = chart
  installForm.value = {
    releaseName: chart.name,
    version: chart.version,
    namespace: 'default',
    values: ''
  }
  chartDrawerVisible.value = true

  // Fetch versions
  versionsLoading.value = true
  try {
    const res = await request.get(`/api/v1/plugins/kubernetes/helm/repos/${selectedRepo.value}/charts/${chart.name}/versions`)
    chartVersions.value = res || []
  } catch (error) {
    console.error(error)
    chartVersions.value = [{ version: chart.version, appVersion: chart.appVersion }]
  } finally {
    versionsLoading.value = false
  }

  // Fetch namespaces
  if (props.clusterId) {
    namespacesLoading.value = true
    try {
      const nsData = await getNamespaces(props.clusterId)
      namespaces.value = nsData.map((ns: any) => ns.name)
    } catch (error) {
      console.error(error)
      namespaces.value = ['default', 'kube-system']
    } finally {
      namespacesLoading.value = false
    }
  }
}

const installChart = async () => {
  if (!props.clusterId) {
    ElMessage.warning('请先选择集群')
    return
  }
  if (!installForm.value.releaseName || !installForm.value.version || !installForm.value.namespace) {
    ElMessage.warning('请填写必填项')
    return
  }

  installing.value = true
  try {
    await request.post(`/api/v1/plugins/kubernetes/helm/clusters/${props.clusterId}/releases`, {
      repoId: selectedRepo.value,
      chartName: selectedChart.value.name,
      version: installForm.value.version,
      releaseName: installForm.value.releaseName,
      namespace: installForm.value.namespace,
      values: installForm.value.values
    })
    ElMessage.success('安装成功')
    chartDrawerVisible.value = false
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '安装失败')
  } finally {
    installing.value = false
  }
}

onMounted(() => {
  fetchRepos()
})

defineExpose({
  fetchRepos
})
</script>

<style scoped>
.charts-list {
  height: 100%;
  display: flex;
  flex-direction: column;
}

/* 搜索栏样式 - 与 ConfigMapList 保持一致 */
.search-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.search-bar-left {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-bar-right {
  display: flex;
  gap: 12px;
}

.search-input {
  width: 280px;
}

.filter-select {
  width: 200px;
}

.search-icon {
  color: #909399;
}

/* 表格容器 */
.table-wrapper {
  flex: 1;
  overflow: auto;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.charts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
  padding: 16px;
}

.charts-grid.is-empty {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 300px;
}

.chart-card {
  display: flex;
  flex-direction: row;
  cursor: pointer;
  transition: all 0.3s;
}

.chart-card:hover {
  transform: translateY(-5px);
}

.chart-icon {
  width: 80px;
  height: 80px;
  padding: 10px;
  flex-shrink: 0;
}

.chart-icon img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.chart-info {
  flex: 1;
  padding: 10px;
  overflow: hidden;
}

.chart-name {
  margin: 0 0 5px 0;
  font-size: 16px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.chart-desc {
  margin: 0 0 10px 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  height: 36px;
}

.chart-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.credentials-section {
  margin-top: 10px;
  padding: 10px;
  background-color: var(--el-fill-color-light);
  border-radius: 4px;
}

.credentials-section h4 {
  margin-top: 0;
  margin-bottom: 10px;
  font-size: 14px;
}

/* Repo Drawer */
.repo-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.repo-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
}

.repo-info {
  flex: 1;
  overflow: hidden;
}

.repo-name {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 4px;
}

.repo-url {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Chart Detail */
.chart-detail {
  padding: 0 10px;
}

.chart-header {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}

.chart-detail-icon {
  width: 80px;
  height: 80px;
  object-fit: contain;
  flex-shrink: 0;
}

.chart-header-info h2 {
  margin: 0 0 8px 0;
  font-size: 20px;
}

.chart-header-info p {
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 14px;
}

.install-form {
  margin-top: 20px;
}
</style>
