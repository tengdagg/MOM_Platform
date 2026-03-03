<template>
  <div class="dashboard">
    <el-row :gutter="10" class="stats-row">
      <el-col :xl="4" :lg="4" :md="8" :sm="12" :xs="12" v-for="(stat, index) in topStats" :key="index">
        <el-card class="stat-card" shadow="hover">
          <div class="stat-content">
            <div class="stat-icon" :style="{ backgroundColor: stat.color }">
              <el-icon :size="22" color="#fff">
                <component :is="stat.icon" />
              </el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stat.value }}</div>
              <div class="stat-label">{{ stat.label }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="chart-row">
      <el-col :span="12">
        <el-card class="chart-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">主机状态分布</span>
              <span class="view-all-link" @click="$router.push('/asset/hosts')">查看全部</span>
            </div>
          </template>
          <div ref="hostStatusChart" class="chart-container"></div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="chart-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">网络设备状态分布</span>
              <span class="view-all-link" @click="$router.push('/asset/network-devices')">查看全部</span>
            </div>
          </template>
          <div ref="deviceStatusChart" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="chart-row">
      <el-col :span="12">
        <el-card class="chart-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">K8s节点数概览</span>
              <span class="view-all-link" @click="$router.push('/kubernetes/clusters')">查看全部</span>
            </div>
          </template>
          <div ref="k8sResourceChart" class="chart-container"></div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="chart-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">模型调用次数（最近7天）</span>
              <span class="view-all-link" @click="$router.push('/audit/operation-logs')">查看全部</span>
            </div>
          </template>
          <div ref="operationTrendChart" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="chart-row">
      <el-col :span="12">
        <el-card class="chart-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">告警类型分布</span>
              <span class="view-all-link" @click="$router.push('/monitor/alert-logs')">查看全部</span>
            </div>
          </template>
          <div ref="alertStatsChart" class="chart-container"></div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="chart-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">AI Skill 分类分布</span>
              <span class="view-all-link" @click="$router.push('/ai/skills')">查看全部</span>
            </div>
          </template>
          <div ref="aiSkillChart" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="quick-access-row">
      <el-col :span="24">
        <el-card class="quick-access-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">快速入口</span>
            </div>
          </template>
          <div class="quick-access-grid">
            <div class="quick-item" @click="$router.push('/asset/hosts')">
              <el-icon :size="24" color="#409EFF"><OfficeBuilding /></el-icon>
              <span>主机管理</span>
            </div>
            <div class="quick-item" @click="$router.push('/asset/network-devices')">
              <el-icon :size="24" color="#9254de"><SetUp /></el-icon>
              <span>网络设备</span>
            </div>
            <div class="quick-item" @click="$router.push('/ai/chat')">
              <el-icon :size="24" color="#13c2c2"><MagicStick /></el-icon>
              <span>AI 助手</span>
            </div>
            <div class="quick-item" @click="$router.push('/kubernetes/clusters')">
              <el-icon :size="24" color="#67C23A"><Connection /></el-icon>
              <span>K8s集群</span>
            </div>
            <div class="quick-item" @click="$router.push('/audit/operation-logs')">
              <el-icon :size="24" color="#E6A23C"><Document /></el-icon>
              <span>操作日志</span>
            </div>
            <div class="quick-item" @click="$router.push('/monitor/alert-logs')">
              <el-icon :size="24" color="#F56C6C"><Warning /></el-icon>
              <span>告警日志</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, markRaw, nextTick } from 'vue'
import type { ECharts } from 'echarts'
import * as echarts from 'echarts'
import {
  OfficeBuilding,
  SetUp,
  Connection,
  Document,
  Warning,
  MagicStick
} from '@element-plus/icons-vue'
import { getHostList, getNetworkDeviceList } from '@/api/host'
import { getClusterList } from '@/api/kubernetes'
import { getAlertLogs } from '@/api/alert-config'
import { getSkillStats, getModelCallStats } from '@/api/ai'

const topStats = ref([
  { label: '主机总数', value: '0', icon: markRaw(OfficeBuilding), color: '#409EFF' },
  { label: '网络设备', value: '0', icon: markRaw(SetUp), color: '#9254de' },
  { label: 'K8s集群', value: '0', icon: markRaw(Connection), color: '#67C23A' },
  { label: 'AI Skills启用', value: '0', icon: markRaw(MagicStick), color: '#13c2c2' },
  { label: '今日模型调用', value: '0', icon: markRaw(Document), color: '#E6A23C' },
  { label: '活跃告警', value: '0', icon: markRaw(Warning), color: '#F56C6C' }
])

const hostStatusChart = ref<HTMLElement>()
const deviceStatusChart = ref<HTMLElement>()
const k8sResourceChart = ref<HTMLElement>()
const operationTrendChart = ref<HTMLElement>()
const alertStatsChart = ref<HTMLElement>()
const aiSkillChart = ref<HTMLElement>()

const chartInstances = new Map<string, ECharts>()

const hosts = ref<any[]>([])
const devices = ref<any[]>([])
const clusters = ref<any[]>([])
const alertLogs = ref<any[]>([])
const skillStats = ref<any>(null)
const modelCallStats = ref<any>(null)

const toList = (res: any): any[] => {
  if (!res) return []
  if (Array.isArray(res)) return res
  if (Array.isArray(res.list)) return res.list
  return []
}

const toTotal = (res: any, list: any[]): number => {
  if (!res) return list.length
  if (typeof res.total === 'number') return res.total
  return list.length
}

const ensureChart = (key: string, el?: HTMLElement): ECharts | null => {
  if (!el) return null
  if (chartInstances.has(key)) return chartInstances.get(key) || null
  const ins = echarts.init(el)
  chartInstances.set(key, ins)
  return ins
}

const resizeAllCharts = () => {
  chartInstances.forEach((chart) => chart.resize())
}

const renderHostStatusChart = () => {
  const chart = ensureChart('host', hostStatusChart.value)
  if (!chart) return
  const online = hosts.value.filter((h) => h?.status === 1).length
  const offline = Math.max(hosts.value.length - online, 0)
  chart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: { bottom: 0 },
    series: [{
      name: '主机状态',
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['50%', '45%'],
      data: [
        { value: online, name: '在线', itemStyle: { color: '#67C23A' } },
        { value: offline, name: '离线', itemStyle: { color: '#909399' } }
      ]
    }]
  })
}

const renderDeviceStatusChart = () => {
  const chart = ensureChart('device', deviceStatusChart.value)
  if (!chart) return
  const online = devices.value.filter((d) => d?.status === 1).length
  const offline = devices.value.filter((d) => d?.status === 0).length
  const unknown = Math.max(devices.value.length - online - offline, 0)
  chart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: { bottom: 0 },
    series: [{
      name: '网络设备状态',
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['50%', '45%'],
      data: [
        { value: online, name: '在线', itemStyle: { color: '#67C23A' } },
        { value: offline, name: '离线', itemStyle: { color: '#F56C6C' } },
        { value: unknown, name: '未知', itemStyle: { color: '#909399' } }
      ]
    }]
  })
}

const renderK8sResourceChart = () => {
  const chart = ensureChart('k8s', k8sResourceChart.value)
  if (!chart) return
  const names = clusters.value.map((c) => c.name || '未命名')
  const nodeCounts = clusters.value.map((c) => c.nodeCount || 0)
  chart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['节点数'], top: 8 },
    grid: { left: '3%', right: '4%', bottom: '3%', top: '16%', containLabel: true },
    xAxis: {
      type: 'category',
      data: names.length ? names : ['暂无数据'],
      axisLabel: { interval: 0, rotate: names.length > 3 ? 25 : 0 }
    },
    yAxis: { type: 'value' },
    series: [
      { name: '节点数', type: 'bar', data: nodeCounts.length ? nodeCounts : [0], itemStyle: { color: '#409EFF' }, barMaxWidth: 40 }
    ]
  })
}

const modelColors = ['#E6A23C', '#409EFF', '#67C23A', '#9254de', '#F56C6C', '#13c2c2', '#ff85c0']

const renderOperationTrendChart = () => {
  const chart = ensureChart('operation', operationTrendChart.value)
  if (!chart) return
  const daily = modelCallStats.value?.daily || []

  const today = new Date()
  const dates: string[] = []
  const dateKeys: string[] = []
  for (let i = 6; i >= 0; i--) {
    const d = new Date(today)
    d.setDate(d.getDate() - i)
    dates.push(`${d.getMonth() + 1}/${d.getDate()}`)
    const y = d.getFullYear()
    const m = String(d.getMonth() + 1).padStart(2, '0')
    const dd = String(d.getDate()).padStart(2, '0')
    dateKeys.push(`${y}-${m}-${dd}`)
  }

  const modelMap = new Map<string, Map<string, number>>()
  for (const row of daily) {
    const name = row.modelName || '默认模型'
    if (!modelMap.has(name)) modelMap.set(name, new Map())
    modelMap.get(name)!.set(row.day, row.count)
  }

  const modelNames = Array.from(modelMap.keys())
  const series = modelNames.map((name, idx) => {
    const dayMap = modelMap.get(name)!
    const data = dateKeys.map((dk) => dayMap.get(dk) || 0)
    const color = modelColors[idx % modelColors.length]
    return {
      name,
      type: 'line' as const,
      smooth: true,
      data,
      areaStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: color + '4D' },
          { offset: 1, color: color + '0A' }
        ])
      },
      itemStyle: { color },
      lineStyle: { width: 2 }
    }
  })

  if (series.length === 0) {
    series.push({
      name: '模型调用',
      type: 'line' as const,
      smooth: true,
      data: dateKeys.map(() => 0),
      areaStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: 'rgba(230, 162, 60, 0.3)' },
          { offset: 1, color: 'rgba(230, 162, 60, 0.04)' }
        ])
      },
      itemStyle: { color: '#E6A23C' },
      lineStyle: { width: 2 }
    })
  }

  chart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: modelNames, top: 4 },
    grid: { left: '3%', right: '4%', bottom: '3%', top: modelNames.length > 1 ? '18%' : '10%', containLabel: true },
    xAxis: { type: 'category', boundaryGap: false, data: dates },
    yAxis: { type: 'value', minInterval: 1 },
    series
  })
}

const renderAlertStatsChart = () => {
  const chart = ensureChart('alert', alertStatsChart.value)
  if (!chart) return
  const typeMap = new Map<string, number>()
  alertLogs.value.forEach((log: any) => {
    const type = log?.alertType || '未知'
    typeMap.set(type, (typeMap.get(type) || 0) + 1)
  })
  const typeData = Array.from(typeMap.entries())
    .map(([name, value]) => ({ name, value }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 6)
  const dataForLegend = typeData.length ? typeData : [{ name: '暂无数据', value: 0 }]
  chart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: {
      bottom: 0,
      formatter: (name: string) => {
        const row = dataForLegend.find((i) => i.name === name)
        return `${name} (${row ? row.value : 0})`
      }
    },
    series: [{
      name: '告警类型',
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['50%', '45%'],
      label: {
        show: true,
        formatter: '{b}: {c}',
        fontSize: 12
      },
      data: typeData.length ? typeData : [{ name: '暂无数据', value: 0 }]
    }]
  })
}

const renderAISkillChart = () => {
  const chart = ensureChart('aiSkill', aiSkillChart.value)
  if (!chart) return
  const byCategory = skillStats.value?.byCategory || {}
  const data = Object.keys(byCategory).map((key) => ({
    name: key || 'other',
    value: Number(byCategory[key]) || 0
  }))
  chart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: { bottom: 0 },
    series: [{
      name: 'Skill分类',
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['50%', '45%'],
      data: data.length ? data : [{ name: '暂无数据', value: 1 }]
    }]
  })
}

const fetchHosts = async () => {
  try {
    const res: any = await getHostList({ page: 1, pageSize: 200 })
    const list = toList(res)
    hosts.value = list
    topStats.value[0].value = String(toTotal(res, list))
    renderHostStatusChart()
  } catch {
    hosts.value = []
    topStats.value[0].value = '0'
    renderHostStatusChart()
  }
}

const fetchNetworkDevices = async () => {
  try {
    const res: any = await getNetworkDeviceList({ page: 1, pageSize: 200 })
    const list = toList(res)
    devices.value = list
    topStats.value[1].value = String(toTotal(res, list))
    renderDeviceStatusChart()
  } catch {
    devices.value = []
    topStats.value[1].value = '0'
    renderDeviceStatusChart()
  }
}

const fetchClusters = async () => {
  try {
    const res: any = await getClusterList()
    const list = toList(res)
    clusters.value = list
    topStats.value[2].value = String(toTotal(res, list))
    renderK8sResourceChart()
  } catch {
    clusters.value = []
    topStats.value[2].value = '0'
    renderK8sResourceChart()
  }
}

const fetchSkillStats = async () => {
  try {
    const res: any = await getSkillStats()
    skillStats.value = res || {}
    const enabled = Number(res?.builtinEnabled || 0) + Number(res?.customEnabled || 0)
    topStats.value[3].value = String(enabled)
    renderAISkillChart()
  } catch {
    skillStats.value = null
    topStats.value[3].value = '0'
    renderAISkillChart()
  }
}

const fetchModelCallStats = async () => {
  try {
    const res: any = await getModelCallStats()
    modelCallStats.value = res || {}
    topStats.value[4].value = String(res?.todayTotal || 0)
    renderOperationTrendChart()
  } catch {
    modelCallStats.value = null
    topStats.value[4].value = '0'
    renderOperationTrendChart()
  }
}

const fetchAlertLogs = async () => {
  try {
    const res: any = await getAlertLogs({ page: 1, pageSize: 500 })
    const list = toList(res)
    alertLogs.value = list
    const activeCount = list.filter((log: any) => ['failed', 'error', 'firing'].includes(String(log?.status || '').toLowerCase())).length
    topStats.value[5].value = String(activeCount)
    renderAlertStatsChart()
  } catch {
    alertLogs.value = []
    topStats.value[5].value = '0'
    renderAlertStatsChart()
  }
}

onMounted(async () => {
  window.addEventListener('resize', resizeAllCharts)
  await nextTick()
  await Promise.all([
    fetchHosts(),
    fetchNetworkDevices(),
    fetchClusters(),
    fetchSkillStats(),
    fetchModelCallStats(),
    fetchAlertLogs()
  ])
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', resizeAllCharts)
  chartInstances.forEach((chart) => chart.dispose())
  chartInstances.clear()
})
</script>

<style scoped>
.dashboard {
  padding: 0;
}

.stats-row,
.chart-row,
.quick-access-row {
  margin-bottom: 16px;
}

.stat-card,
.chart-card,
.quick-access-card {
  border-radius: 12px;
  border: 1px solid #f0f2f5;
  overflow: hidden;
}

.stat-card :deep(.el-card__body) {
  padding: 10px 12px;
}

.chart-card :deep(.el-card__header),
.quick-access-card :deep(.el-card__header) {
  padding: 12px 14px;
  border-bottom: 1px solid #f0f2f5;
}

.chart-card :deep(.el-card__body),
.quick-access-card :deep(.el-card__body) {
  padding: 14px;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.stat-icon {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-value {
  font-size: 16px;
  font-weight: 700;
  color: #303133;
  line-height: 1;
}

.stat-label {
  margin-top: 3px;
  font-size: 11px;
  color: #909399;
  white-space: nowrap;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.view-all-link {
  font-size: 12px;
  color: #409eff;
  cursor: pointer;
}

.view-all-link:hover {
  color: #66b1ff;
}

.chart-container {
  width: 100%;
  height: 260px;
}

.quick-access-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 12px;
}

.quick-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 14px;
  border-radius: 10px;
  background: linear-gradient(180deg, #fafbfc 0%, #f5f7fa 100%);
  cursor: pointer;
  transition: all 0.2s;
}

.quick-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 18px rgba(0, 0, 0, 0.08);
}

.quick-item span {
  margin-top: 8px;
  font-size: 12px;
  color: #606266;
  font-weight: 500;
}

@media (max-width: 992px) {
  .chart-container {
    height: 220px;
  }
}
</style>
