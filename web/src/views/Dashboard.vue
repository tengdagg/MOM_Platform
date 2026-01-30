<template>
  <div class="dashboard">
    <!-- 顶部统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :span="6" v-for="(stat, index) in topStats" :key="index">
        <el-card class="stat-card" shadow="hover">
          <div class="stat-content">
            <div class="stat-icon" :style="{ backgroundColor: stat.color }">
              <el-icon :size="22" :color="'#fff'">
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

    <!-- 图表展示区域 -->
    <el-row :gutter="20" class="chart-row">
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
              <span class="card-title">K8s集群资源概览</span>
              <span class="view-all-link" @click="$router.push('/kubernetes/clusters')">查看全部</span>
            </div>
          </template>
          <div ref="k8sResourceChart" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" class="chart-row">
      <el-col :span="12">
        <el-card class="chart-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">操作趋势（最近7天）</span>
              <span class="view-all-link" @click="$router.push('/audit/operation-logs')">查看全部</span>
            </div>
          </template>
          <div ref="operationTrendChart" class="chart-container"></div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="chart-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="card-title">告警统计</span>
              <span class="view-all-link" @click="$router.push('/monitor/alert-logs')">查看全部</span>
            </div>
          </template>
          <div ref="alertStatsChart" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 快速入口 -->
    <el-row :gutter="20" class="quick-access-row">
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
            <div class="quick-item" @click="$router.push('/asset/credentials')">
              <el-icon :size="24" color="#909399"><Key /></el-icon>
              <span>凭据管理</span>
            </div>
            <div class="quick-item" @click="$router.push('/asset/cloud-accounts')">
              <el-icon :size="24" color="#606266"><Cloudy /></el-icon>
              <span>云账号</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, markRaw } from 'vue'
import {
  OfficeBuilding,
  Connection,
  Document,
  Warning,
  Key,
  Cloudy
} from '@element-plus/icons-vue'
import { getHostList } from '@/api/host'
import { getClusterList } from '@/api/kubernetes'
import { getOperationLogList } from '@/api/audit'
import { getAlertLogs } from '@/api/alert-config'
import * as echarts from 'echarts'

// 顶部统计数据
const topStats = ref([
  {
    label: '主机总数',
    value: '0',
    icon: markRaw(OfficeBuilding),
    color: '#409EFF'
  },
  {
    label: 'K8s集群',
    value: '0',
    icon: markRaw(Connection),
    color: '#67C23A'
  },
  {
    label: '今日操作',
    value: '0',
    icon: markRaw(Document),
    color: '#E6A23C'
  },
  {
    label: '活跃告警',
    value: '0',
    icon: markRaw(Warning),
    color: '#F56C6C'
  }
])

// 图表DOM引用
const hostStatusChart = ref<HTMLElement>()
const k8sResourceChart = ref<HTMLElement>()
const operationTrendChart = ref<HTMLElement>()
const alertStatsChart = ref<HTMLElement>()

// 数据存储
const hosts = ref<any[]>([])
const clusters = ref<any[]>([])
const operationLogs = ref<any[]>([])
const alertLogs = ref<any[]>([])

// 获取主机列表
const fetchHosts = async () => {
  try {
    const res: any = await getHostList({ page: 1, pageSize: 100 })
    if (res) {
      if (res.list && Array.isArray(res.list)) {
        hosts.value = res.list
        topStats.value[0].value = String(res.total || res.list.length || 0)
      } else if (Array.isArray(res)) {
        hosts.value = res
        topStats.value[0].value = String(res.length || 0)
      }
    }
    await nextTick()
    renderHostStatusChart()
  } catch (error) {
    topStats.value[0].value = '0'
  }
}

// 获取K8s集群列表
const fetchClusters = async () => {
  try {
    const res: any = await getClusterList()
    if (res) {
      if (res.list && Array.isArray(res.list)) {
        clusters.value = res.list
        topStats.value[1].value = String(res.total || res.list.length || 0)
      } else if (Array.isArray(res)) {
        clusters.value = res
        topStats.value[1].value = String(res.length || 0)
      }
    }
    await nextTick()
    renderK8sResourceChart()
  } catch (error) {
    topStats.value[1].value = '0'
  }
}

// 获取操作日志列表
const fetchOperationLogs = async () => {
  try {
    const today = new Date()
    today.setHours(0, 0, 0, 0)

    const res: any = await getOperationLogList({ page: 1, pageSize: 500 })
    if (res) {
      if (res.list && Array.isArray(res.list)) {
        operationLogs.value = res.list
        const todayCount = res.list.filter((log: any) => {
          const logDate = new Date(log.createdAt)
          return logDate >= today
        }).length
        topStats.value[2].value = String(todayCount)
      } else if (Array.isArray(res)) {
        operationLogs.value = res
        const todayCount = res.filter((log: any) => {
          const logDate = new Date(log.createdAt)
          return logDate >= today
        }).length
        topStats.value[2].value = String(todayCount)
      }
    }
    await nextTick()
    renderOperationTrendChart()
  } catch (error) {
    topStats.value[2].value = '0'
  }
}

// 获取告警日志列表
const fetchAlertLogs = async () => {
  try {
    const res: any = await getAlertLogs({ page: 1, pageSize: 100 })
    if (res) {
      if (res.list && Array.isArray(res.list)) {
        alertLogs.value = res.list
        const activeCount = res.list.filter((log: any) => log.status === 'failed').length
        topStats.value[3].value = String(activeCount)
      } else if (Array.isArray(res)) {
        alertLogs.value = res
        const activeCount = res.filter((log: any) => log.status === 'failed').length
        topStats.value[3].value = String(activeCount)
      }
    }
    await nextTick()
    renderAlertStatsChart()
  } catch (error) {
    topStats.value[3].value = '0'
  }
}

// 渲染主机状态图表
const renderHostStatusChart = () => {
  if (!hostStatusChart.value) return

  const chart = echarts.init(hostStatusChart.value)

  const onlineCount = hosts.value.filter(h => h.status === 1).length
  const offlineCount = hosts.value.filter(h => h.status !== 1).length

  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center'
    },
    series: [
      {
        name: '主机状态',
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['40%', '50%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
          borderWidth: 2
        },
        label: {
          show: false,
          position: 'center'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 20,
            fontWeight: 'bold'
          }
        },
        labelLine: {
          show: false
        },
        data: [
          { value: onlineCount, name: '在线', itemStyle: { color: '#67C23A' } },
          { value: offlineCount, name: '离线', itemStyle: { color: '#909399' } }
        ]
      }
    ]
  }

  chart.setOption(option)

  // 响应式
  window.addEventListener('resize', () => chart.resize())
}

// 渲染K8s资源图表
const renderK8sResourceChart = () => {
  if (!k8sResourceChart.value) return

  const chart = echarts.init(k8sResourceChart.value)

  const clusterNames = clusters.value.map(c => c.name || '未命名')
  const nodeCounts = clusters.value.map(c => c.nodeCount || 0)
  const podCounts = clusters.value.map(c => c.podCount || 0)

  const option = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
    },
    legend: {
      data: ['节点数', 'Pod数'],
      top: 10
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '15%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: clusterNames.length > 0 ? clusterNames : ['暂无数据'],
      axisLabel: {
        interval: 0,
        rotate: clusterNames.length > 3 ? 30 : 0
      }
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '节点数',
        type: 'bar',
        data: nodeCounts.length > 0 ? nodeCounts : [0],
        itemStyle: { color: '#409EFF' },
        barMaxWidth: 40
      },
      {
        name: 'Pod数',
        type: 'bar',
        data: podCounts.length > 0 ? podCounts : [0],
        itemStyle: { color: '#67C23A' },
        barMaxWidth: 40
      }
    ]
  }

  chart.setOption(option)
  window.addEventListener('resize', () => chart.resize())
}

// 渲染操作趋势图表
const renderOperationTrendChart = () => {
  if (!operationTrendChart.value) return

  const chart = echarts.init(operationTrendChart.value)

  // 统计最近7天的操作数
  const today = new Date()
  const dates: string[] = []
  const counts: number[] = []

  for (let i = 6; i >= 0; i--) {
    const date = new Date(today)
    date.setDate(date.getDate() - i)
    const dateStr = `${date.getMonth() + 1}/${date.getDate()}`
    dates.push(dateStr)

    const count = operationLogs.value.filter((log: any) => {
      const logDate = new Date(log.createdAt)
      return logDate.toDateString() === date.toDateString()
    }).length

    counts.push(count)
  }

  const option = {
    tooltip: {
      trigger: 'axis'
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '10%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: dates
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '操作次数',
        type: 'line',
        smooth: true,
        data: counts,
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(230, 162, 60, 0.3)' },
            { offset: 1, color: 'rgba(230, 162, 60, 0.05)' }
          ])
        },
        itemStyle: { color: '#E6A23C' },
        lineStyle: { width: 2 }
      }
    ]
  }

  chart.setOption(option)
  window.addEventListener('resize', () => chart.resize())
}

// 渲染告警统计图表
const renderAlertStatsChart = () => {
  if (!alertStatsChart.value) return

  const chart = echarts.init(alertStatsChart.value)

  const successCount = alertLogs.value.filter((log: any) => log.status === 'success').length
  const failedCount = alertLogs.value.filter((log: any) => log.status === 'failed').length

  // 按告警类型统计
  const typeMap = new Map<string, number>()
  alertLogs.value.forEach((log: any) => {
    const type = log.alertType || '未知'
    typeMap.set(type, (typeMap.get(type) || 0) + 1)
  })

  const typeData = Array.from(typeMap.entries())
    .map(([name, value]) => ({ name, value }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 5)

  const option = {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      data: typeData.map(d => d.name)
    },
    series: [
      {
        name: '告警类型',
        type: 'pie',
        radius: ['40%', '70%'],
        center: ['40%', '50%'],
        data: typeData.length > 0 ? typeData : [{ name: '暂无数据', value: 1 }],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        }
      }
    ]
  }

  chart.setOption(option)
  window.addEventListener('resize', () => chart.resize())
}

// 页面加载时获取数据
onMounted(() => {
  fetchHosts()
  fetchClusters()
  fetchOperationLogs()
  fetchAlertLogs()
})
</script>

<style scoped>
.dashboard {
  padding: 0;
  background-color: transparent;
}

.stats-row {
  margin-bottom: 16px;
}

.stat-card {
  border-radius: 0;
  overflow: hidden;
}

.stat-card :deep(.el-card__body) {
  padding: 14px;
}

.stat-content {
  display: flex;
  align-items: center;
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 12px;
}

.stat-icon :deep(.el-icon) {
  font-size: 22px !important;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 20px;
  font-weight: bold;
  color: #303133;
  line-height: 1;
  margin-bottom: 4px;
}

.stat-label {
  font-size: 12px;
  color: #909399;
}

.chart-row {
  margin-bottom: 16px;
}

.chart-card {
  border-radius: 0;
  height: 100%;
}

.chart-card :deep(.el-card__header) {
  padding: 10px 14px;
  border-bottom: 1px solid #ebeef5;
}

.chart-card :deep(.el-card__body) {
  padding: 14px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.view-all-link {
  font-size: 12px;
  color: #409EFF;
  cursor: pointer;
}

.view-all-link:hover {
  color: #66b1ff;
}

.chart-container {
  width: 100%;
  height: 240px;
}

.quick-access-row {
  margin-bottom: 16px;
}

.quick-access-card {
  border-radius: 0;
}

.quick-access-card :deep(.el-card__header) {
  padding: 10px 14px;
  border-bottom: 1px solid #ebeef5;
}

.quick-access-card :deep(.el-card__body) {
  padding: 14px;
}

.quick-access-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));
  gap: 12px;
}

.quick-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 14px;
  border-radius: 0;
  background-color: #f5f7fa;
  cursor: pointer;
  transition: all 0.3s;
}

.quick-item :deep(.el-icon) {
  font-size: 24px !important;
}

.quick-item:hover {
  background-color: #ecf5ff;
  transform: translateY(-2px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.quick-item span {
  margin-top: 8px;
  font-size: 12px;
  color: #606266;
  font-weight: 500;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .stat-value {
    font-size: 18px;
  }

  .stat-icon {
    width: 40px;
    height: 40px;
  }

  .chart-container {
    height: 200px;
  }
}

@media (max-width: 768px) {
  .quick-access-grid {
    grid-template-columns: repeat(auto-fill, minmax(90px, 1fr));
    gap: 10px;
  }

  .quick-item {
    padding: 10px;
  }
}
</style>
