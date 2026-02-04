<template>
  <div class="config-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Key /></el-icon>
        </div>
        <div>
          <h2 class="page-title">配置管理</h2>
          <p class="page-subtitle">管理 Kubernetes ConfigMaps、Secrets 和其他配置资源</p>
        </div>
      </div>
      <div class="header-actions">
        <el-select
          v-model="selectedClusterId"
          placeholder="选择集群"
          class="cluster-select"
          @change="handleClusterChange"
        >
          <template #prefix>
            <el-icon class="search-icon"><Platform /></el-icon>
          </template>
          <el-option
            v-for="cluster in clusterList"
            :key="cluster.id"
            :label="cluster.alias || cluster.name"
            :value="cluster.id"
          />
        </el-select>
        <el-button class="black-button" @click="loadCurrentResources">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 配置类型标签 -->
    <div class="config-types-bar">
      <div
        v-for="type in configTypes"
        :key="type.value"
        :class="['type-tab', { active: activeTab === type.value }]"
        @click="handleTabChange(type.value)"
      >
        <el-icon class="type-icon">
          <component :is="type.icon" />
        </el-icon>
        <span class="type-label">{{ type.label }}</span>
        <span v-if="type.count !== undefined" class="type-count">({{ type.count }})</span>
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="content-wrapper">
      <!-- ConfigMaps -->
      <ConfigMapList
        v-show="activeTab === 'configmaps' && selectedClusterId"
        ref="configMapListRef"
        :clusterId="selectedClusterId"
        :namespace="namespaceParam"
        @edit="handleEditConfigMap"
        @yaml="handleEditConfigMapYAML"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('configmaps', count)"
      />

      <!-- Secrets -->
      <SecretList
        v-show="activeTab === 'secrets' && selectedClusterId"
        ref="secretListRef"
        :clusterId="selectedClusterId"
        :namespace="namespaceParam"
        @edit="handleEditSecret"
        @yaml="handleEditSecretYAML"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('secrets', count)"
      />

      <!-- ResourceQuotas -->
      <ResourceQuotaList
        v-show="activeTab === 'resourcequotas' && selectedClusterId"
        ref="resourceQuotaListRef"
        :clusterId="selectedClusterId"
        :namespace="namespaceParam"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('resourcequotas', count)"
      />

      <!-- LimitRanges -->
      <LimitRangeList
        v-show="activeTab === 'limitranges' && selectedClusterId"
        ref="limitRangeListRef"
        :clusterId="selectedClusterId"
        :namespace="namespaceParam"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('limitranges', count)"
      />

      <!-- HPA -->
      <HPAList
        v-show="activeTab === 'hpa' && selectedClusterId"
        ref="hpaListRef"
        :clusterId="selectedClusterId"
        :namespace="namespaceParam"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('hpa', count)"
      />

      <!-- PodDisruptionBudgets -->
      <PodDisruptionBudgetList
        v-show="activeTab === 'pdb' && selectedClusterId"
        ref="pdbListRef"
        :clusterId="selectedClusterId"
        :namespace="namespaceParam"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('pdb', count)"
      />

      <!-- PriorityClasses -->
      <PriorityClassList
        v-show="activeTab === 'priorityclasses' && selectedClusterId"
        ref="priorityClassListRef"
        :clusterId="selectedClusterId"
        @count-update="(count) => updateCount('priorityclasses', count)"
      />

      <!-- RuntimeClasses -->
      <RuntimeClassList
        v-show="activeTab === 'runtimeclasses' && selectedClusterId"
        ref="runtimeClassListRef"
        :clusterId="selectedClusterId"
        @count-update="(count) => updateCount('runtimeclasses', count)"
      />

      <!-- Leases -->
      <LeaseList
        v-show="activeTab === 'leases' && selectedClusterId"
        ref="leaseListRef"
        :clusterId="selectedClusterId"
        @count-update="(count) => updateCount('leases', count)"
      />

      <!-- MutatingWebhooks -->
      <MutatingWebhookList
        v-show="activeTab === 'mutatingwebhooks' && selectedClusterId"
        ref="mutatingWebhookListRef"
        :clusterId="selectedClusterId"
        @count-update="(count) => updateCount('mutatingwebhooks', count)"
      />

      <!-- ValidatingWebhooks -->
      <ValidatingWebhookList
        v-show="activeTab === 'validatingwebhooks' && selectedClusterId"
        ref="validatingWebhookListRef"
        :clusterId="selectedClusterId"
        @count-update="(count) => updateCount('validatingwebhooks', count)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Platform,
  Refresh,
  Key,
  Lock,
  Histogram,
  Operation,
  TrendCharts,
  FolderOpened,
  Trophy,
  Cpu,
  Timer,
  Link
} from '@element-plus/icons-vue'
import { getClusterList, getNamespaces, type Cluster } from '@/api/kubernetes'
import { useKubernetesStore } from '@/stores/kubernetes'
import axios from 'axios'
import ConfigMapList from './config-components/ConfigMapList.vue'
import SecretList from './config-components/SecretList.vue'
import ResourceQuotaList from './config-components/ResourceQuotaList.vue'
import LimitRangeList from './config-components/LimitRangeList.vue'
import HPAList from './config-components/HPAList.vue'
import PodDisruptionBudgetList from './config-components/PodDisruptionBudgetList.vue'
import PriorityClassList from './config-components/PriorityClassList.vue'
import RuntimeClassList from './config-components/RuntimeClassList.vue'
import LeaseList from './config-components/LeaseList.vue'
import MutatingWebhookList from './config-components/MutatingWebhookList.vue'
import ValidatingWebhookList from './config-components/ValidatingWebhookList.vue'

// 使用全局 Kubernetes store
const kubernetesStore = useKubernetesStore()

// 配置类型定义
interface ConfigType {
  label: string
  value: string
  icon: any
  count: number
}

const configTypes = ref<ConfigType[]>([
  { label: 'ConfigMaps', value: 'configmaps', icon: Key, count: 0 },
  { label: 'Secrets', value: 'secrets', icon: Lock, count: 0 },
  { label: 'ResourceQuotas', value: 'resourcequotas', icon: Histogram, count: 0 },
  { label: 'LimitRanges', value: 'limitranges', icon: Operation, count: 0 },
  { label: 'HPA', value: 'hpa', icon: TrendCharts, count: 0 },
  { label: 'PodDisruptionBudgets', value: 'pdb', icon: Lock, count: 0 },
  { label: 'PriorityClasses', value: 'priorityclasses', icon: Trophy, count: 0 },
  { label: 'RuntimeClasses', value: 'runtimeclasses', icon: Cpu, count: 0 },
  { label: 'Leases', value: 'leases', icon: Timer, count: 0 },
  { label: 'MutatingWebhooks', value: 'mutatingwebhooks', icon: Link, count: 0 },
  { label: 'ValidatingWebhooks', value: 'validatingwebhooks', icon: Link, count: 0 },
])

const clusterList = ref<Cluster[]>([])
const namespaceList = ref<any[]>([])
const selectedClusterId = ref<number>()
const activeTab = ref('configmaps')

// 使用 computed 双向绑定 store 的 selectedNamespaces
const selectedNamespaces = computed({
  get: () => kubernetesStore.selectedNamespaces,
  set: (val: string[]) => kubernetesStore.setNamespaces(val)
})

// 命名空间选择变化处理
const handleNamespaceChange = (namespaces: string[]) => {
  kubernetesStore.setNamespaces(namespaces)
  loadCurrentResources()
}

// 为子组件提供的命名空间参数
const namespaceParam = computed(() => {
  if (selectedNamespaces.value.length === 0) return ''
  return selectedNamespaces.value.join(',')
})

// 子组件引用
const configMapListRef = ref()
const secretListRef = ref()
const resourceQuotaListRef = ref()
const limitRangeListRef = ref()
const hpaListRef = ref()
const pdbListRef = ref()
const priorityClassListRef = ref()
const runtimeClassListRef = ref()
const leaseListRef = ref()
const mutatingWebhookListRef = ref()
const validatingWebhookListRef = ref()

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      const savedClusterId = localStorage.getItem('config_selected_cluster_id')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
    }
  } catch (error) {
    ElMessage.error('获取集群列表失败')
  }
}

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!selectedClusterId.value) return
  try {
    const data = await getNamespaces(selectedClusterId.value)
    namespaceList.value = data || []
  } catch (error) {
    console.error('获取命名空间列表失败', error)
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('config_selected_cluster_id', selectedClusterId.value.toString())
    await loadNamespaces()
  }
}

// Tab 切换
const handleTabChange = (tab: string) => {
  activeTab.value = tab
  localStorage.setItem('config_active_tab', tab)
}

// 加载当前资源
const loadCurrentResources = async () => {
  if (!selectedClusterId.value) return

  // 根据当前激活的 tab 刷新对应的子组件数据
  switch (activeTab.value) {
    case 'configmaps':
      await configMapListRef.value?.loadConfigMaps?.()
      break
    case 'secrets':
      await secretListRef.value?.loadSecrets?.()
      break
    case 'resourcequotas':
      await resourceQuotaListRef.value?.loadResourceQuotas?.()
      break
    case 'limitranges':
      await limitRangeListRef.value?.loadLimitRanges?.()
      break
    case 'hpa':
      await hpaListRef.value?.loadHPAs?.()
      break
    case 'pdb':
      await pdbListRef.value?.loadPDBs?.()
      break
    case 'priorityclasses':
      await priorityClassListRef.value?.loadPriorityClasses?.()
      break
    case 'runtimeclasses':
      await runtimeClassListRef.value?.loadRuntimeClasses?.()
      break
    case 'leases':
      await leaseListRef.value?.loadLeases?.()
      break
    case 'mutatingwebhooks':
      await mutatingWebhookListRef.value?.loadMutatingWebhooks?.()
      break
    case 'validatingwebhooks':
      await validatingWebhookListRef.value?.loadValidatingWebhooks?.()
      break
  }
}

// ConfigMap 操作
const handleEditConfigMap = (configMap: any) => {
  ElMessage.info('编辑 ConfigMap 功能开发中...')
}

const handleEditConfigMapYAML = (configMap: any) => {
  ElMessage.info('编辑 ConfigMap YAML 功能开发中...')
}

// Secret 操作
const handleEditSecret = (secret: any) => {
  ElMessage.info('编辑 Secret 功能开发中...')
}

const handleEditSecretYAML = (secret: any) => {
  ElMessage.info('编辑 Secret YAML 功能开发中...')
}

// 更新数量
const updateCount = (type: string, count: number) => {
  const configType = configTypes.value.find(t => t.value === type)
  if (configType) {
    configType.count = count
  }
}

onMounted(async () => {
  await loadClusters()
  await loadNamespaces()
  const savedTab = localStorage.getItem('config_active_tab')
  if (savedTab) {
    activeTab.value = savedTab
  }
})
</script>

<style scoped>
.config-container {
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

.cluster-select {
  width: 200px;
}

.namespace-select {
  width: 250px;
}

/* 按钮样式 - 使用全局样式 .black-button */

.search-icon {
  color: #909399;
}

/* 配置类型标签栏 - 使用全局样式 .type-tab */

/* 内容区域 */
.content-wrapper {
  background: transparent;
}

.cluster-select :deep(.el-input__wrapper) {
  border-radius: 0;
}
</style>
