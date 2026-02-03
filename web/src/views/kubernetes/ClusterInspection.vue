<template>
  <div class="inspection-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><DocumentChecked /></el-icon>
        </div>
        <div>
          <h2 class="page-title">集群巡检</h2>
          <p class="page-subtitle">对 Kubernetes 集群进行健康检查，生成巡检报告</p>
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
        <el-button class="black-button" @click="handleStartInspection" :loading="inspecting" :disabled="!selectedClusterId">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          {{ inspecting ? '巡检中...' : '开始巡检' }}
        </el-button>
        <el-button @click="historyDialogVisible = true">
          <el-icon style="margin-right: 6px;"><Clock /></el-icon>
          历史记录
        </el-button>
      </div>
    </div>

    <!-- 巡检进度 -->
    <div class="progress-section" v-if="inspecting">
      <div class="progress-card">
        <div class="progress-header">
          <div class="progress-animation">
            <div class="progress-spinner"></div>
          </div>
          <div class="progress-info">
            <span class="progress-title">正在巡检集群...</span>
            <span class="progress-step">{{ progressInfo.currentStep }}</span>
          </div>
        </div>
        <el-progress
          :percentage="progressInfo.progress"
          :stroke-width="16"
          :color="progressColors"
          striped
          striped-flow
        >
          <template #default="{ percentage }">
            <span class="progress-text">{{ percentage }}%</span>
          </template>
        </el-progress>
      </div>
    </div>

    <!-- 空状态 - 未执行巡检 -->
    <div class="empty-state" v-if="!inspecting && !inspectionResult">
      <div class="empty-content">
        <div class="empty-icon">
          <el-icon><DocumentChecked /></el-icon>
        </div>
        <h3 class="empty-title">开始集群健康巡检</h3>
        <p class="empty-desc">选择一个 Kubernetes 集群，执行全面的健康检查，包括节点状态、工作负载、网络、存储、安全配置等多个维度</p>
        <div class="empty-features">
          <div class="feature-item">
            <el-icon><Monitor /></el-icon>
            <span>节点健康</span>
          </div>
          <div class="feature-item">
            <el-icon><Box /></el-icon>
            <span>工作负载</span>
          </div>
          <div class="feature-item">
            <el-icon><Connection /></el-icon>
            <span>网络状态</span>
          </div>
          <div class="feature-item">
            <el-icon><Files /></el-icon>
            <span>存储管理</span>
          </div>
          <div class="feature-item">
            <el-icon><Lock /></el-icon>
            <span>安全配置</span>
          </div>
          <div class="feature-item">
            <el-icon><DataAnalysis /></el-icon>
            <span>容量规划</span>
          </div>
        </div>
        <el-button class="black-button start-btn" size="large" @click="handleStartInspection" :disabled="!selectedClusterId">
          <el-icon style="margin-right: 8px;"><Refresh /></el-icon>
          立即开始巡检
        </el-button>
      </div>
    </div>

    <!-- 巡检结果详情 -->
    <div class="result-section" v-if="inspectionResult && !inspecting">
      <!-- 统计卡片 -->
      <div class="stats-grid">
        <div class="stat-card score-card">
          <div class="score-circle" :class="getScoreClass(inspectionResult.score)">
            <svg viewBox="0 0 100 100">
              <circle class="score-bg" cx="50" cy="50" r="45" />
              <circle
                class="score-progress"
                cx="50" cy="50" r="45"
                :stroke-dasharray="scoreCircle"
                :stroke-dashoffset="scoreOffset"
              />
            </svg>
            <div class="score-value">{{ inspectionResult.score }}</div>
            <div class="score-label">健康评分</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon stat-icon-blue">
            <el-icon><DocumentChecked /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-label">检查项目</div>
            <div class="stat-value">{{ inspectionResult.summary.totalChecks }}</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon stat-icon-green">
            <el-icon><CircleCheckFilled /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-label">通过项</div>
            <div class="stat-value success-text">{{ inspectionResult.summary.passedChecks }}</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon" :class="getIssueIconClass()">
            <el-icon><WarnTriangleFilled /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-label">发现问题</div>
            <div class="stat-value" :class="getIssueClass()">
              {{ inspectionResult.summary.warningChecks + inspectionResult.summary.failedChecks }}
            </div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon stat-icon-blue">
            <el-icon><Timer /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-label">巡检耗时</div>
            <div class="stat-value">{{ currentInspection?.duration || 0 }}s</div>
          </div>
        </div>
      </div>
      <el-tabs v-model="activeTab" class="inspection-tabs">
        <el-tab-pane label="概览" name="overview">
          <div class="overview-content">
            <!-- 结果统计 -->
            <div class="result-summary">
              <div class="summary-item success">
                <el-icon><CircleCheckFilled /></el-icon>
                <span class="summary-label">正常</span>
                <span class="summary-value">{{ inspectionResult.summary.passedChecks }}</span>
              </div>
              <div class="summary-item warning">
                <el-icon><WarningFilled /></el-icon>
                <span class="summary-label">警告</span>
                <span class="summary-value">{{ inspectionResult.summary.warningChecks }}</span>
              </div>
              <div class="summary-item error">
                <el-icon><CircleCloseFilled /></el-icon>
                <span class="summary-label">异常</span>
                <span class="summary-value">{{ inspectionResult.summary.failedChecks }}</span>
              </div>
            </div>

            <!-- 检查项列表 -->
            <div class="check-items-table">
              <el-table :data="allCheckItems" style="width: 100%" :header-cell-style="tableHeaderStyle">
                <el-table-column prop="category" label="类别" width="120" />
                <el-table-column prop="name" label="检查项" width="200" />
                <el-table-column label="状态" width="100">
                  <template #default="{ row }">
                    <el-tag :type="getStatusTagType(row.status)" effect="dark" size="small">
                      {{ getStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="value" label="检查值" width="150" show-overflow-tooltip />
                <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
                <el-table-column prop="suggestion" label="建议" min-width="200" show-overflow-tooltip />
              </el-table>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="集群信息" name="cluster">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">Kubernetes 版本</span>
                <span class="info-value">{{ inspectionResult.clusterInfo.version }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">平台</span>
                <span class="info-value">{{ inspectionResult.clusterInfo.platform }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">Go 版本</span>
                <span class="info-value">{{ inspectionResult.clusterInfo.goVersion }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">连接延迟</span>
                <span class="info-value">{{ inspectionResult.clusterInfo.connectionDelay }}ms</span>
              </div>
            </div>
            <div class="check-items-table">
              <el-table :data="inspectionResult.clusterInfo.items" style="width: 100%" :header-cell-style="tableHeaderStyle">
                <el-table-column prop="category" label="类别" width="120" />
                <el-table-column prop="name" label="检查项" width="200" />
                <el-table-column label="状态" width="100">
                  <template #default="{ row }">
                    <el-tag :type="getStatusTagType(row.status)" effect="dark" size="small">
                      {{ getStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="value" label="检查值" width="150" />
                <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
              </el-table>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="节点健康" name="nodes">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">节点总数</span>
                <span class="info-value">{{ inspectionResult.nodeHealth.totalNodes }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">Ready 节点</span>
                <span class="info-value success-text">{{ inspectionResult.nodeHealth.readyNodes }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">NotReady 节点</span>
                <span class="info-value" :class="{ 'error-text': inspectionResult.nodeHealth.notReadyNodes > 0 }">
                  {{ inspectionResult.nodeHealth.notReadyNodes }}
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">有资源压力</span>
                <span class="info-value" :class="{ 'warning-text': inspectionResult.nodeHealth.pressureNodes > 0 }">
                  {{ inspectionResult.nodeHealth.pressureNodes }}
                </span>
              </div>
            </div>
            <!-- 节点资源利用率表格 -->
            <div class="check-items-table" v-if="inspectionResult.nodeHealth.nodeUtilization.length">
              <h4 class="table-title">节点资源利用率</h4>
              <el-table :data="inspectionResult.nodeHealth.nodeUtilization" style="width: 100%" :header-cell-style="tableHeaderStyle">
                <el-table-column prop="name" label="节点名称" min-width="200" />
                <el-table-column label="状态" width="100">
                  <template #default="{ row }">
                    <el-tag :type="row.status === 'Ready' ? 'success' : 'danger'" effect="dark" size="small">
                      {{ row.status }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="cpuCapacity" label="CPU容量" width="120" />
                <el-table-column prop="memoryCapacity" label="内存容量" width="120" />
                <el-table-column label="Pod数量" width="120">
                  <template #default="{ row }">
                    {{ row.podCount }}/{{ row.podCapacity }}
                  </template>
                </el-table-column>
              </el-table>
            </div>
            <!-- 检查项 -->
            <div class="check-items-table">
              <h4 class="table-title">检查项</h4>
              <el-table :data="inspectionResult.nodeHealth.items" style="width: 100%" :header-cell-style="tableHeaderStyle">
                <el-table-column prop="category" label="类别" width="120" />
                <el-table-column prop="name" label="检查项" width="200" />
                <el-table-column label="状态" width="100">
                  <template #default="{ row }">
                    <el-tag :type="getStatusTagType(row.status)" effect="dark" size="small">
                      {{ getStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="value" label="检查值" width="150" />
                <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
                <el-table-column prop="suggestion" label="建议" min-width="200" show-overflow-tooltip />
              </el-table>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="工作负载" name="workloads">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">Deployment</span>
                <span class="info-value">{{ inspectionResult.workloads.healthyDeployments }}/{{ inspectionResult.workloads.totalDeployments }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">DaemonSet</span>
                <span class="info-value">{{ inspectionResult.workloads.healthyDaemonSets }}/{{ inspectionResult.workloads.totalDaemonSets }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">StatefulSet</span>
                <span class="info-value">{{ inspectionResult.workloads.healthyStatefulSets }}/{{ inspectionResult.workloads.totalStatefulSets }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">Pod总数</span>
                <span class="info-value">{{ inspectionResult.workloads.runningPods }}/{{ inspectionResult.workloads.totalPods }}</span>
              </div>
            </div>
            <!-- 异常工作负载 -->
            <div class="check-items-table" v-if="inspectionResult.workloads.unhealthyWorkloads.length">
              <h4 class="table-title">异常工作负载</h4>
              <el-table :data="inspectionResult.workloads.unhealthyWorkloads" style="width: 100%" :header-cell-style="tableHeaderStyle">
                <el-table-column prop="kind" label="类型" width="120" />
                <el-table-column prop="namespace" label="命名空间" width="150" />
                <el-table-column prop="name" label="名称" min-width="200" />
                <el-table-column prop="ready" label="就绪状态" width="120" />
                <el-table-column prop="reason" label="原因" min-width="200" />
              </el-table>
            </div>
            <!-- 检查项 -->
            <div class="check-items-table">
              <h4 class="table-title">检查项</h4>
              <el-table :data="inspectionResult.workloads.items" style="width: 100%" :header-cell-style="tableHeaderStyle">
                <el-table-column prop="category" label="类别" width="120" />
                <el-table-column prop="name" label="检查项" width="200" />
                <el-table-column label="状态" width="100">
                  <template #default="{ row }">
                    <el-tag :type="getStatusTagType(row.status)" effect="dark" size="small">
                      {{ getStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="value" label="检查值" width="150" />
                <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
                <el-table-column prop="suggestion" label="建议" min-width="200" show-overflow-tooltip />
              </el-table>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="网络" name="network">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">Service总数</span>
                <span class="info-value">{{ inspectionResult.network.totalServices }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">Ingress总数</span>
                <span class="info-value">{{ inspectionResult.network.totalIngresses }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">NetworkPolicy</span>
                <span class="info-value">{{ inspectionResult.network.networkPolicies }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">无Endpoint的Service</span>
                <span class="info-value" :class="{ 'warning-text': inspectionResult.network.noEndpointServices > 0 }">
                  {{ inspectionResult.network.noEndpointServices }}
                </span>
              </div>
            </div>
            <div class="check-items-table">
              <el-table :data="inspectionResult.network.items" style="width: 100%" :header-cell-style="tableHeaderStyle">
                <el-table-column prop="category" label="类别" width="120" />
                <el-table-column prop="name" label="检查项" width="200" />
                <el-table-column label="状态" width="100">
                  <template #default="{ row }">
                    <el-tag :type="getStatusTagType(row.status)" effect="dark" size="small">
                      {{ getStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="value" label="检查值" width="150" />
                <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
              </el-table>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="存储" name="storage">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">PV总数</span>
                <span class="info-value">{{ inspectionResult.storage.totalPVs }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">PVC总数</span>
                <span class="info-value">{{ inspectionResult.storage.totalPVCs }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">StorageClass</span>
                <span class="info-value">{{ inspectionResult.storage.storageClasses }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">Pending PVC</span>
                <span class="info-value" :class="{ 'warning-text': inspectionResult.storage.pendingPVCs > 0 }">
                  {{ inspectionResult.storage.pendingPVCs }}
                </span>
              </div>
            </div>
            <div class="check-items-table">
              <el-table :data="inspectionResult.storage.items" style="width: 100%" :header-cell-style="tableHeaderStyle">
                <el-table-column prop="category" label="类别" width="120" />
                <el-table-column prop="name" label="检查项" width="200" />
                <el-table-column label="状态" width="100">
                  <template #default="{ row }">
                    <el-tag :type="getStatusTagType(row.status)" effect="dark" size="small">
                      {{ getStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="value" label="检查值" width="150" />
                <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
                <el-table-column prop="suggestion" label="建议" min-width="200" show-overflow-tooltip />
              </el-table>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="安全" name="security">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">ServiceAccount</span>
                <span class="info-value">{{ inspectionResult.security.serviceAccounts }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">特权Pod</span>
                <span class="info-value" :class="{ 'warning-text': inspectionResult.security.privilegedPods > 0 }">
                  {{ inspectionResult.security.privilegedPods }}
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">hostNetwork Pod</span>
                <span class="info-value" :class="{ 'warning-text': inspectionResult.security.hostNetworkPods > 0 }">
                  {{ inspectionResult.security.hostNetworkPods }}
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">cluster-admin绑定</span>
                <span class="info-value">{{ inspectionResult.security.clusterAdminBindings }}</span>
              </div>
            </div>
            <div class="check-items-table">
              <el-table :data="inspectionResult.security.items" style="width: 100%" :header-cell-style="tableHeaderStyle">
                <el-table-column prop="category" label="类别" width="120" />
                <el-table-column prop="name" label="检查项" width="200" />
                <el-table-column label="状态" width="100">
                  <template #default="{ row }">
                    <el-tag :type="getStatusTagType(row.status)" effect="dark" size="small">
                      {{ getStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="value" label="检查值" width="150" />
                <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
                <el-table-column prop="suggestion" label="建议" min-width="200" show-overflow-tooltip />
              </el-table>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="容量" name="capacity">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">CPU分配率</span>
                <span class="info-value" :class="getCapacityClass(inspectionResult.capacity.cpuAllocatePercent)">
                  {{ inspectionResult.capacity.cpuAllocatePercent.toFixed(1) }}%
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">内存分配率</span>
                <span class="info-value" :class="getCapacityClass(inspectionResult.capacity.memoryAllocatePercent)">
                  {{ inspectionResult.capacity.memoryAllocatePercent.toFixed(1) }}%
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">Pod密度</span>
                <span class="info-value" :class="getCapacityClass(inspectionResult.capacity.podDensityPercent)">
                  {{ inspectionResult.capacity.podDensityPercent.toFixed(1) }}%
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">Pod数量</span>
                <span class="info-value">{{ inspectionResult.capacity.currentPodCount }}/{{ inspectionResult.capacity.totalPodCapacity }}</span>
              </div>
            </div>
            <div class="check-items-table">
              <el-table :data="inspectionResult.capacity.items" style="width: 100%" :header-cell-style="tableHeaderStyle">
                <el-table-column prop="category" label="类别" width="120" />
                <el-table-column prop="name" label="检查项" width="200" />
                <el-table-column label="状态" width="100">
                  <template #default="{ row }">
                    <el-tag :type="getStatusTagType(row.status)" effect="dark" size="small">
                      {{ getStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="value" label="检查值" width="150" />
                <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
                <el-table-column prop="suggestion" label="建议" min-width="200" show-overflow-tooltip />
              </el-table>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="事件" name="events">
          <div class="tab-content">
            <div class="info-grid">
              <div class="info-item">
                <span class="info-label">Warning事件</span>
                <span class="info-value" :class="{ 'warning-text': inspectionResult.events.warningEvents > 10 }">
                  {{ inspectionResult.events.warningEvents }}
                </span>
              </div>
              <div class="info-item">
                <span class="info-label">Error事件</span>
                <span class="info-value" :class="{ 'error-text': inspectionResult.events.errorEvents > 0 }">
                  {{ inspectionResult.events.errorEvents }}
                </span>
              </div>
            </div>
            <!-- 最近事件 -->
            <div class="check-items-table" v-if="inspectionResult.events.recentEvents.length">
              <h4 class="table-title">最近告警事件</h4>
              <el-table :data="inspectionResult.events.recentEvents" style="width: 100%" :header-cell-style="tableHeaderStyle">
                <el-table-column prop="type" label="类型" width="100" />
                <el-table-column prop="reason" label="原因" width="150" />
                <el-table-column prop="object" label="对象" width="200" show-overflow-tooltip />
                <el-table-column prop="namespace" label="命名空间" width="120" />
                <el-table-column prop="count" label="次数" width="80" />
                <el-table-column prop="lastSeen" label="最后发生" width="180" />
                <el-table-column prop="message" label="消息" min-width="300" show-overflow-tooltip />
              </el-table>
            </div>
            <!-- 检查项 -->
            <div class="check-items-table">
              <h4 class="table-title">检查项</h4>
              <el-table :data="inspectionResult.events.items" style="width: 100%" :header-cell-style="tableHeaderStyle">
                <el-table-column prop="category" label="类别" width="120" />
                <el-table-column prop="name" label="检查项" width="200" />
                <el-table-column label="状态" width="100">
                  <template #default="{ row }">
                    <el-tag :type="getStatusTagType(row.status)" effect="dark" size="small">
                      {{ getStatusText(row.status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="value" label="检查值" width="150" />
                <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
                <el-table-column prop="suggestion" label="建议" min-width="200" show-overflow-tooltip />
              </el-table>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>

      <!-- 导出按钮 -->
      <div class="export-actions">
        <el-button class="black-button" @click="handleExportExcel" :loading="exporting">
          <el-icon style="margin-right: 6px;"><Download /></el-icon>
          导出Excel报告
        </el-button>
        <el-button @click="handleStartInspection" :disabled="inspecting">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          重新巡检
        </el-button>
      </div>
    </div>

    <!-- 历史记录对话框 -->
    <el-dialog v-model="historyDialogVisible" title="巡检历史记录" width="900px">
      <template v-if="historyList.length === 0 && !historyLoading">
        <div class="history-empty">
          <el-icon class="history-empty-icon"><Clock /></el-icon>
          <p class="history-empty-text">暂无巡检记录</p>
          <p class="history-empty-hint">执行集群巡检后，历史记录将会显示在这里</p>
        </div>
      </template>
      <template v-else>
        <el-table :data="historyList" style="width: 100%" :header-cell-style="tableHeaderStyle" v-loading="historyLoading">
          <el-table-column prop="clusterName" label="集群" width="150" />
          <el-table-column label="评分" width="100">
            <template #default="{ row }">
              <span :class="getScoreClass(row.score)">{{ row.score }}/100</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'completed' ? 'success' : row.status === 'running' ? 'warning' : 'danger'" size="small">
                {{ row.status === 'completed' ? '完成' : row.status === 'running' ? '进行中' : '失败' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="checkCount" label="检查项" width="80" />
          <el-table-column prop="passCount" label="通过" width="80" />
          <el-table-column prop="warningCount" label="警告" width="80" />
          <el-table-column prop="failCount" label="失败" width="80" />
          <el-table-column prop="duration" label="耗时(s)" width="80" />
          <el-table-column prop="createdAt" label="时间" width="180" />
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link size="small" @click="handleViewHistory(row)" :disabled="row.status !== 'completed'">
                查看
              </el-button>
              <el-button type="danger" link size="small" @click="handleDeleteHistory(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="pagination-wrapper" v-if="historyTotal > 0">
          <el-pagination
            v-model:current-page="historyPage"
            v-model:page-size="historyPageSize"
            :total="historyTotal"
            :page-sizes="[10, 20, 50]"
            layout="total, sizes, prev, pager, next"
            @size-change="loadHistory"
            @current-change="loadHistory"
          />
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  DocumentChecked,
  Platform,
  Refresh,
  Clock,
  Timer,
  WarnTriangleFilled,
  CircleCheckFilled,
  WarningFilled,
  CircleCloseFilled,
  Download,
  Monitor,
  Box,
  Connection,
  Files,
  Lock,
  DataAnalysis
} from '@element-plus/icons-vue'
import { getClusterList, type Cluster } from '@/api/kubernetes'
import {
  startInspection,
  getInspectionProgress,
  getInspectionResult,
  getInspectionHistory,
  deleteInspection,
  exportInspection,
  type InspectionResult,
  type InspectionProgress,
  type InspectionHistoryItem,
  type ClusterInspection,
  type CheckItem
} from '@/api/inspection'

// 状态
const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()
const inspecting = ref(false)
const currentInspectionId = ref<number>()
const currentInspection = ref<ClusterInspection | null>(null)
const inspectionResult = ref<InspectionResult | null>(null)
const progressInfo = ref<InspectionProgress>({
  inspectionId: 0,
  status: '',
  progress: 0,
  currentStep: '',
  completedClusters: 0,
  totalClusters: 0
})
const activeTab = ref('overview')
const exporting = ref(false)

// 历史记录
const historyDialogVisible = ref(false)
const historyList = ref<InspectionHistoryItem[]>([])
const historyLoading = ref(false)
const historyPage = ref(1)
const historyPageSize = ref(10)
const historyTotal = ref(0)

// 进度条颜色
const progressColors = [
  { color: '#909399', percentage: 30 },
  { color: '#e6a23c', percentage: 60 },
  { color: '#67c23a', percentage: 100 }
]

// 表格头样式
const tableHeaderStyle = {
  background: '#000',
  color: '#fff',
  fontSize: '14px',
  fontWeight: '600'
}

// 计算所有检查项
const allCheckItems = computed(() => {
  if (!inspectionResult.value) return []
  const items: CheckItem[] = []
  items.push(...(inspectionResult.value.clusterInfo?.items || []))
  items.push(...(inspectionResult.value.nodeHealth?.items || []))
  items.push(...(inspectionResult.value.components?.items || []))
  items.push(...(inspectionResult.value.workloads?.items || []))
  items.push(...(inspectionResult.value.network?.items || []))
  items.push(...(inspectionResult.value.storage?.items || []))
  items.push(...(inspectionResult.value.security?.items || []))
  items.push(...(inspectionResult.value.config?.items || []))
  items.push(...(inspectionResult.value.capacity?.items || []))
  items.push(...(inspectionResult.value.events?.items || []))
  return items
})

// 圆环进度条计算
const scoreCircle = computed(() => {
  const circumference = 2 * Math.PI * 45
  return `${circumference} ${circumference}`
})

const scoreOffset = computed(() => {
  if (!inspectionResult.value) return 283
  const circumference = 2 * Math.PI * 45
  const progress = inspectionResult.value.score / 100
  return circumference * (1 - progress)
})

// 监听历史对话框打开
watch(historyDialogVisible, (visible) => {
  if (visible) {
    loadHistory()
  }
})

// 获取评分样式类
const getScoreClass = (score: number) => {
  if (score >= 80) return 'score-excellent'
  if (score >= 60) return 'score-good'
  return 'score-poor'
}

// 获取问题图标样式
const getIssueIconClass = () => {
  if (!inspectionResult.value) return 'stat-icon-blue'
  const issues = inspectionResult.value.summary.warningChecks + inspectionResult.value.summary.failedChecks
  if (issues === 0) return 'stat-icon-green'
  if (inspectionResult.value.summary.failedChecks > 0) return 'stat-icon-red'
  return 'stat-icon-orange'
}

// 获取问题数量样式
const getIssueClass = () => {
  if (!inspectionResult.value) return ''
  if (inspectionResult.value.summary.failedChecks > 0) return 'error-text'
  if (inspectionResult.value.summary.warningChecks > 0) return 'warning-text'
  return 'success-text'
}

// 获取状态标签类型
const getStatusTagType = (status: string) => {
  switch (status) {
    case 'success': return 'success'
    case 'warning': return 'warning'
    case 'error': return 'danger'
    default: return 'info'
  }
}

// 获取状态文本
const getStatusText = (status: string) => {
  switch (status) {
    case 'success': return '正常'
    case 'warning': return '警告'
    case 'error': return '异常'
    default: return '未知'
  }
}

// 获取容量样式
const getCapacityClass = (percent: number) => {
  if (percent > 85) return 'error-text'
  if (percent > 70) return 'warning-text'
  return 'success-text'
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0 && !selectedClusterId.value) {
      selectedClusterId.value = clusterList.value[0].id
    }
  } catch (error: any) {
    ElMessage.error('获取集群列表失败: ' + (error.message || '未知错误'))
  }
}

// 处理集群变更
const handleClusterChange = () => {
  inspectionResult.value = null
  currentInspection.value = null
}

// 开始巡检
const handleStartInspection = async () => {
  if (!selectedClusterId.value) {
    ElMessage.warning('请先选择集群')
    return
  }

  inspecting.value = true
  inspectionResult.value = null
  progressInfo.value = {
    inspectionId: 0,
    status: 'running',
    progress: 0,
    currentStep: '初始化巡检...',
    completedClusters: 0,
    totalClusters: 1
  }

  try {
    const data = await startInspection({
      clusterIds: [selectedClusterId.value]
    })
    currentInspectionId.value = data.inspectionId
    pollProgress()
  } catch (error: any) {
    inspecting.value = false
    ElMessage.error('启动巡检失败: ' + (error.message || '未知错误'))
  }
}

// 轮询进度
const pollProgress = async () => {
  if (!currentInspectionId.value) return

  try {
    const data = await getInspectionProgress(currentInspectionId.value)
    progressInfo.value = data

    if (data.status === 'completed') {
      inspecting.value = false
      await loadInspectionResult()
      ElMessage.success('巡检完成')
    } else if (data.status === 'failed') {
      inspecting.value = false
      ElMessage.error('巡检失败')
    } else {
      setTimeout(pollProgress, 1000)
    }
  } catch (error) {
    setTimeout(pollProgress, 2000)
  }
}

// 加载巡检结果
const loadInspectionResult = async () => {
  if (!currentInspectionId.value) return

  try {
    const data = await getInspectionResult(currentInspectionId.value)
    currentInspection.value = data.inspection
    inspectionResult.value = data.result
  } catch (error: any) {
    ElMessage.error('获取巡检结果失败: ' + (error.message || '未知错误'))
  }
}

// 导出Excel
const handleExportExcel = async () => {
  if (!currentInspectionId.value) return

  exporting.value = true
  try {
    const res = await exportInspection(currentInspectionId.value, 'excel')
    // 创建下载链接
    const blob = new Blob([res as any], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `cluster-inspection-${currentInspection.value?.clusterName || 'report'}-${new Date().toISOString().slice(0, 10)}.xlsx`
    link.click()
    window.URL.revokeObjectURL(url)
    ElMessage.success('导出成功')
  } catch (error: any) {
    ElMessage.error('导出失败: ' + (error.message || '未知错误'))
  } finally {
    exporting.value = false
  }
}

// 加载历史记录
const loadHistory = async () => {
  historyLoading.value = true
  try {
    const data = await getInspectionHistory({
      clusterId: selectedClusterId.value,
      page: historyPage.value,
      pageSize: historyPageSize.value
    })
    historyList.value = data.list || []
    historyTotal.value = data.total
  } catch (error: any) {
    ElMessage.error('获取历史记录失败: ' + (error.message || '未知错误'))
  } finally {
    historyLoading.value = false
  }
}

// 查看历史记录
const handleViewHistory = async (item: InspectionHistoryItem) => {
  currentInspectionId.value = item.id
  await loadInspectionResult()
  historyDialogVisible.value = false
}

// 删除历史记录
const handleDeleteHistory = async (item: InspectionHistoryItem) => {
  try {
    await ElMessageBox.confirm('确定要删除这条巡检记录吗?', '提示', {
      type: 'warning'
    })
    await deleteInspection(item.id)
    ElMessage.success('删除成功')
    loadHistory()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + (error.message || '未知错误'))
    }
  }
}

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.inspection-container {
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
  width: 280px;
}

/* 按钮样式 - 使用全局样式 .black-button */

/* 进度条 */
.progress-section {
  margin-bottom: 16px;
}

.progress-card {
  padding: 24px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.progress-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
}

.progress-animation {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.progress-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid #f0f0f0;
  border-top-color: #ffffff;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.progress-info {
  flex: 1;
}

.progress-title {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.progress-step {
  font-size: 14px;
  color: #909399;
}

.progress-text {
  font-size: 14px;
  font-weight: 600;
  color: #ffffff;
}

/* 空状态 */
.empty-state {
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  padding: 60px 40px;
}

.empty-content {
  max-width: 600px;
  margin: 0 auto;
  text-align: center;
}

.empty-icon {
  width: 100px;
  height: 100px;
  margin: 0 auto 24px;
  background: #0a466a;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  font-size: 48px;
  border: 2px solid #0f69a6;
}

.empty-title {
  margin: 0 0 12px;
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.empty-desc {
  margin: 0 0 32px;
  font-size: 14px;
  color: #909399;
  line-height: 1.6;
}

.empty-features {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 16px;
  margin-bottom: 32px;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: #f5f7fa;
  border-radius: 0;
  font-size: 13px;
  color: #606266;
}

.feature-item .el-icon {
  font-size: 16px;
  color: #ffffff;
}

.start-btn {
  height: 44px;
  padding: 0 32px;
  font-size: 15px;
}

/* 结果部分 */
.result-section {
  background: #fff;
  border-radius: 0;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  padding: 20px;
}

/* 统计卡片 */
.stats-grid {
  display: grid;
  grid-template-columns: 140px repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 0;
  transition: all 0.3s ease;
}

.stat-card:not(.score-card):hover {
  background: #f0f2f5;
}

/* 评分圆环卡片 */
.score-card {
  grid-row: span 1;
  justify-content: center;
  padding: 20px;
  background: #0a466a;
}

.score-circle {
  position: relative;
  width: 100px;
  height: 100px;
}

.score-circle svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.score-circle .score-bg {
  fill: none;
  stroke: #333;
  stroke-width: 8;
}

.score-circle .score-progress {
  fill: none;
  stroke: #0f69a6;
  stroke-width: 8;
  stroke-linecap: round;
  transition: stroke-dashoffset 0.8s ease;
}

.score-circle.score-excellent .score-progress {
  stroke: #52c41a;
}

.score-circle.score-good .score-progress {
  stroke: #faad14;
}

.score-circle.score-poor .score-progress {
  stroke: #ff4d4f;
}

.score-value {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -60%);
  font-size: 28px;
  font-weight: 700;
  color: #fff;
}

.score-label {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, 50%);
  font-size: 12px;
  color: #909399;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  flex-shrink: 0;
}

.stat-icon .el-icon {
  font-size: 24px;
  color: inherit;
}

.stat-icon-blue {
  background: linear-gradient(135deg, #0a466a 0%, #0d5a87 100%);
  color: #ffffff;
  border: none;
}

.stat-icon-green {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #52c41a;
  border: 1px solid #52c41a;
}

.stat-icon-orange {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #faad14;
  border: 1px solid #faad14;
}

.stat-icon-red {
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  color: #ff4d4f;
  border: 1px solid #ff4d4f;
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 13px;
  color: #909399;
  margin-bottom: 4px;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: #303133;
  line-height: 1;
}

.stat-value.score-excellent {
  color: #52c41a;
}

.stat-value.score-good {
  color: #faad14;
}

.stat-value.score-poor {
  color: #ff4d4f;
}

.inspection-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: 20px;
  }

  :deep(.el-tabs__item) {
    font-size: 14px;
    font-weight: 500;
  }

  :deep(.el-tabs__item.is-active) {
    color: #ffffff;
  }

  :deep(.el-tabs__active-bar) {
    background-color: #ffffff;
  }
}

/* 概览内容 */
.overview-content {
  padding: 0;
}

.result-summary {
  display: flex;
  gap: 40px;
  margin-bottom: 24px;
  padding: 20px;
  background: #f5f7fa;
  border-radius: 0;
}

.summary-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.summary-item .el-icon {
  font-size: 24px;
}

.summary-item.success .el-icon {
  color: #52c41a;
}

.summary-item.warning .el-icon {
  color: #faad14;
}

.summary-item.error .el-icon {
  color: #ff4d4f;
}

.summary-label {
  font-size: 14px;
  color: #909399;
}

.summary-value {
  font-size: 24px;
  font-weight: 700;
  color: #303133;
}

/* Tab内容 */
.tab-content {
  padding: 0;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 0;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-label {
  font-size: 13px;
  color: #909399;
}

.info-value {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.success-text {
  color: #52c41a;
}

.warning-text {
  color: #faad14;
}

.error-text {
  color: #ff4d4f;
}

/* 表格 */
.check-items-table {
  margin-top: 16px;
}

.table-title {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

/* 导出按钮 */
.export-actions {
  display: flex;
  justify-content: center;
  gap: 16px;
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid #ebeef5;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

/* 响应式 */
@media (max-width: 1200px) {
  .stats-grid {
    grid-template-columns: repeat(3, 1fr);
  }

  .score-card {
    grid-column: span 1;
  }

  .info-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }

  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .header-actions {
    flex-wrap: wrap;
  }

  .cluster-select {
    width: 100%;
  }

  .empty-features {
    gap: 8px;
  }

  .feature-item {
    padding: 6px 12px;
    font-size: 12px;
  }
}

/* 历史记录空状态 */
.history-empty {
  padding: 60px 20px;
  text-align: center;
}

.history-empty-icon {
  font-size: 64px;
  color: #dcdfe6;
  margin-bottom: 16px;
}

.history-empty-text {
  margin: 0 0 8px;
  font-size: 16px;
  color: #606266;
}

.history-empty-hint {
  margin: 0;
  font-size: 14px;
  color: #909399;
}
</style>
