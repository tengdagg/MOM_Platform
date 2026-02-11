<template>
  <el-drawer
    :model-value="modelValue"
    :title="`${detail?.kind || kind}: ${detail?.name || name}`"
    size="55%"
    destroy-on-close
    append-to-body
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-loading="loading" class="rd-content">
      <template v-if="detail">
        <!-- ==================== Properties ==================== -->
        <h3 class="rd-title">Properties</h3>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="Created">{{ formatTime(detail.creationTimestamp) }}</el-descriptions-item>
          <el-descriptions-item label="Name">{{ detail.name }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.namespace" label="Namespace">
            <el-link type="primary" :underline="false">{{ detail.namespace }}</el-link>
          </el-descriptions-item>

          <!-- Labels -->
          <el-descriptions-item label="Labels">
            <div class="rd-expandable" @click="labelsOpen = !labelsOpen">
              <el-icon :class="{ rotated: labelsOpen }" class="rd-arrow"><ArrowRight /></el-icon>
              <span>{{ Object.keys(detail.labels).length }} Labels</span>
            </div>
            <div v-if="labelsOpen" class="rd-tags">
              <el-tag v-for="(v, k) in detail.labels" :key="k" size="small" class="rd-tag">{{ k }}: {{ v }}</el-tag>
              <span v-if="!Object.keys(detail.labels).length" class="rd-muted">No Labels</span>
            </div>
          </el-descriptions-item>

          <!-- Annotations -->
          <el-descriptions-item label="Annotations">
            <div class="rd-expandable" @click="annotationsOpen = !annotationsOpen">
              <el-icon :class="{ rotated: annotationsOpen }" class="rd-arrow"><ArrowRight /></el-icon>
              <span>{{ Object.keys(detail.annotations).length }} Annotations</span>
            </div>
            <div v-if="annotationsOpen" class="rd-tags">
              <template v-if="Object.keys(detail.annotations).length">
                <div v-for="(v, k) in detail.annotations" :key="k" class="rd-anno-item">
                  <div class="rd-anno-key">{{ k }}</div>
                  <div class="rd-anno-val">{{ v }}</div>
                </div>
              </template>
              <span v-else class="rd-muted">No Annotations</span>
            </div>
          </el-descriptions-item>

          <!-- ==================== Service ==================== -->
          <template v-if="detail.kind === 'Service'">
            <el-descriptions-item label="Selector">
              <el-tag v-for="(v, k) in (detail.spec?.selector || {})" :key="k" size="small" class="rd-tag">{{ k }}={{ v }}</el-tag>
              <span v-if="!Object.keys(detail.spec?.selector || {}).length" class="rd-muted">-</span>
            </el-descriptions-item>
            <el-descriptions-item label="Type">{{ detail.spec?.type || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Session Affinity">{{ detail.spec?.sessionAffinity || 'None' }}</el-descriptions-item>
          </template>

          <!-- ==================== Deployment / StatefulSet / DaemonSet ==================== -->
          <template v-if="isWorkload">
            <!-- Replicas -->
            <el-descriptions-item v-if="detail.kind !== 'DaemonSet'" label="Replicas">{{ replicasText }}</el-descriptions-item>
            <el-descriptions-item v-if="detail.kind === 'DaemonSet'" label="Nodes">
              {{ detail.status?.desiredNumberScheduled || 0 }} desired, {{ detail.status?.currentNumberScheduled || 0 }} current, {{ detail.status?.numberReady || 0 }} ready
            </el-descriptions-item>
            <el-descriptions-item label="Selector">
              <el-tag v-for="(v, k) in (detail.spec?.selector?.matchLabels || {})" :key="k" size="small" class="rd-tag">{{ k }}={{ v }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item v-if="detail.kind === 'Deployment'" label="Strategy Type">{{ detail.spec?.strategy?.type || '-' }}</el-descriptions-item>
            <el-descriptions-item v-if="detail.kind === 'StatefulSet'" label="Service Name">{{ detail.spec?.serviceName || '-' }}</el-descriptions-item>
            <el-descriptions-item v-if="detail.kind === 'StatefulSet'" label="Pod Management">{{ detail.spec?.podManagementPolicy || 'OrderedReady' }}</el-descriptions-item>
            <el-descriptions-item v-if="detail.kind === 'DaemonSet'" label="Update Strategy">{{ detail.spec?.updateStrategy?.type || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Status">
              <el-tag :type="workloadRunning ? 'success' : 'warning'" size="small">{{ workloadRunning ? 'Running' : 'Updating' }}</el-tag>
            </el-descriptions-item>
          </template>

          <!-- ==================== Job ==================== -->
          <template v-if="detail.kind === 'Job'">
            <el-descriptions-item label="Completions">{{ detail.spec?.completions ?? 1 }}</el-descriptions-item>
            <el-descriptions-item label="Parallelism">{{ detail.spec?.parallelism ?? 1 }}</el-descriptions-item>
            <el-descriptions-item label="Status">
              <span>{{ detail.status?.succeeded || 0 }} succeeded, {{ detail.status?.failed || 0 }} failed</span>
            </el-descriptions-item>
          </template>

          <!-- ==================== CronJob ==================== -->
          <template v-if="detail.kind === 'CronJob'">
            <el-descriptions-item label="Schedule">{{ detail.spec?.schedule || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Suspend">{{ detail.spec?.suspend ? 'Yes' : 'No' }}</el-descriptions-item>
            <el-descriptions-item label="Active Jobs">{{ (detail.status?.active || []).length }}</el-descriptions-item>
            <el-descriptions-item label="Last Schedule">{{ detail.status?.lastScheduleTime || '-' }}</el-descriptions-item>
          </template>

          <!-- ==================== ConfigMap ==================== -->
          <template v-if="detail.kind === 'ConfigMap'">
            <el-descriptions-item label="Data">{{ Object.keys(detail.extra?.data || {}).length }} keys</el-descriptions-item>
          </template>

          <!-- ==================== Secret ==================== -->
          <template v-if="detail.kind === 'Secret'">
            <el-descriptions-item label="Type">{{ detail.extra?.type || 'Opaque' }}</el-descriptions-item>
            <el-descriptions-item label="Data">{{ Object.keys(detail.extra?.data || {}).length }} keys</el-descriptions-item>
          </template>

          <!-- ==================== ServiceAccount ==================== -->
          <template v-if="detail.kind === 'ServiceAccount'">
            <el-descriptions-item label="Tokens">
              <span v-if="(detail.extra?.secrets || []).length">{{ (detail.extra?.secrets || []).length }} secrets</span>
              <span v-else>—</span>
            </el-descriptions-item>
          </template>

          <!-- ==================== Ingress ==================== -->
          <template v-if="detail.kind === 'Ingress'">
            <el-descriptions-item label="Ingress Class">{{ detail.spec?.ingressClassName || '-' }}</el-descriptions-item>
          </template>

          <!-- ==================== PVC ==================== -->
          <template v-if="detail.kind === 'PersistentVolumeClaim'">
            <el-descriptions-item label="Status">
              <el-tag :type="detail.status?.phase === 'Bound' ? 'success' : 'warning'" size="small">{{ detail.status?.phase || '-' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Volume">{{ detail.spec?.volumeName || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Capacity">{{ detail.status?.capacity?.storage || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Access Modes">{{ (detail.spec?.accessModes || []).join(', ') || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Storage Class">{{ detail.spec?.storageClassName || '-' }}</el-descriptions-item>
          </template>

          <!-- ==================== PV ==================== -->
          <template v-if="detail.kind === 'PersistentVolume'">
            <el-descriptions-item label="Status">
              <el-tag :type="detail.status?.phase === 'Bound' ? 'success' : 'info'" size="small">{{ detail.status?.phase || '-' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Capacity">{{ detail.spec?.capacity?.storage || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Access Modes">{{ (detail.spec?.accessModes || []).join(', ') || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Reclaim Policy">{{ detail.spec?.persistentVolumeReclaimPolicy || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Storage Class">{{ detail.spec?.storageClassName || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Claim">{{ detail.spec?.claimRef ? `${detail.spec.claimRef.namespace}/${detail.spec.claimRef.name}` : '-' }}</el-descriptions-item>
          </template>

          <!-- ==================== HPA ==================== -->
          <template v-if="detail.kind === 'HorizontalPodAutoscaler'">
            <el-descriptions-item label="Reference">{{ detail.spec?.scaleTargetRef?.kind }}/{{ detail.spec?.scaleTargetRef?.name }}</el-descriptions-item>
            <el-descriptions-item label="Min Replicas">{{ detail.spec?.minReplicas ?? 1 }}</el-descriptions-item>
            <el-descriptions-item label="Max Replicas">{{ detail.spec?.maxReplicas || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Current Replicas">{{ detail.status?.currentReplicas || 0 }}</el-descriptions-item>
          </template>

          <!-- ==================== PDB ==================== -->
          <template v-if="detail.kind === 'PodDisruptionBudget'">
            <el-descriptions-item label="Min Available">{{ detail.spec?.minAvailable ?? '-' }}</el-descriptions-item>
            <el-descriptions-item label="Max Unavailable">{{ detail.spec?.maxUnavailable ?? '-' }}</el-descriptions-item>
            <el-descriptions-item label="Current Healthy">{{ detail.status?.currentHealthy ?? '-' }}</el-descriptions-item>
            <el-descriptions-item label="Desired Healthy">{{ detail.status?.desiredHealthy ?? '-' }}</el-descriptions-item>
          </template>

          <!-- ==================== NetworkPolicy ==================== -->
          <template v-if="detail.kind === 'NetworkPolicy'">
            <el-descriptions-item label="Pod Selector">
              <el-tag v-for="(v, k) in (detail.spec?.podSelector?.matchLabels || {})" :key="k" size="small" class="rd-tag">{{ k }}={{ v }}</el-tag>
              <span v-if="!Object.keys(detail.spec?.podSelector?.matchLabels || {}).length" class="rd-muted">All Pods</span>
            </el-descriptions-item>
            <el-descriptions-item label="Policy Types">{{ (detail.spec?.policyTypes || []).join(', ') || '-' }}</el-descriptions-item>
          </template>
        </el-descriptions>

        <!-- ==================== Service: Connection ==================== -->
        <template v-if="detail.kind === 'Service'">
          <h3 class="rd-title">Connection</h3>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="Cluster IP">{{ detail.spec?.clusterIP || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Cluster IPs">{{ (detail.spec?.clusterIPs || []).join(', ') || '-' }}</el-descriptions-item>
            <el-descriptions-item label="IP Families">{{ (detail.spec?.ipFamilies || []).join(', ') || '-' }}</el-descriptions-item>
            <el-descriptions-item label="IP Family Policy">{{ detail.spec?.ipFamilyPolicy || '-' }}</el-descriptions-item>
          </el-descriptions>

          <h3 class="rd-title">Ports</h3>
          <div v-if="(detail.spec?.ports || []).length" class="rd-tags-block">
            <el-tag v-for="(p, i) in detail.spec?.ports" :key="i" class="rd-tag">
              {{ p.port }}{{ p.name ? ':' + p.name : '' }}/{{ p.protocol || 'TCP' }}
            </el-tag>
          </div>
          <div v-else class="rd-muted">No ports</div>

          <h3 class="rd-title">Endpoints</h3>
          <template v-if="detail.endpoints && detail.endpoints.length">
            <div v-for="(ep, i) in detail.endpoints" :key="i" class="rd-endpoint">
              <el-link type="primary" :underline="false" class="rd-endpoint-name">{{ ep.targetName || '-' }}</el-link>
              <span class="rd-endpoint-addr">{{ ep.addresses.join(', ') }}</span>
            </div>
          </template>
          <div v-else class="rd-muted">No endpoints</div>
        </template>

        <!-- ==================== Workload: Conditions ==================== -->
        <template v-if="isWorkload && (detail.status?.conditions || []).length">
          <h3 class="rd-title">Conditions</h3>
          <div class="rd-tags-block">
            <el-tag
              v-for="c in detail.status?.conditions"
              :key="c.type"
              :type="c.status === 'True' ? 'success' : 'danger'"
              size="small"
              class="rd-tag"
            >{{ c.type }}</el-tag>
          </div>
        </template>

        <!-- ==================== Deployment: ReplicaSets ==================== -->
        <template v-if="detail.kind === 'Deployment' && detail.replicaSets && detail.replicaSets.length">
          <h3 class="rd-title">Deploy Revisions</h3>
          <el-table :data="detail.replicaSets" size="small" border :header-cell-style="tableHeaderStyle">
            <el-table-column label="Revision" prop="revision" width="80" align="center" />
            <el-table-column label="Ready" width="100" align="center">
              <template #default="{ row }">{{ row.ready }}/{{ row.desired }}</template>
            </el-table-column>
            <el-table-column label="Age" prop="age" width="100" />
            <el-table-column label="Name" prop="name" min-width="200" show-overflow-tooltip />
          </el-table>
        </template>

        <!-- ==================== Workload / Job: Pods ==================== -->
        <template v-if="hasPods">
          <h3 class="rd-title">Pods</h3>
          <el-table :data="detail.pods" size="small" border :header-cell-style="tableHeaderStyle">
            <el-table-column label="Name" prop="name" min-width="220" show-overflow-tooltip />
            <el-table-column label="Node" prop="node" width="130" show-overflow-tooltip />
            <el-table-column label="Namespace" prop="namespace" width="100" />
            <el-table-column label="Ready" prop="ready" width="70" align="center" />
            <el-table-column label="CPU" prop="cpu" width="80" align="center" />
            <el-table-column label="Memory" prop="memory" width="90" align="center" />
            <el-table-column label="Status" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="row.status === 'Running' ? 'success' : row.status === 'Succeeded' ? 'info' : 'warning'" size="small">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </template>

        <!-- ==================== Ingress: Rules ==================== -->
        <template v-if="detail.kind === 'Ingress' && (detail.spec?.rules || []).length">
          <h3 class="rd-title">Rules</h3>
          <div v-for="(rule, ri) in detail.spec?.rules" :key="ri" class="rd-ingress-rule">
            <div class="rd-rule-host">{{ rule.host || '*' }}</div>
            <el-table :data="rule.http?.paths || []" size="small" border :header-cell-style="tableHeaderStyle">
              <el-table-column label="Path" prop="path" min-width="150" />
              <el-table-column label="Path Type" prop="pathType" width="120" />
              <el-table-column label="Backend" width="200">
                <template #default="{ row }">{{ row.backend?.service?.name }}:{{ row.backend?.service?.port?.number || row.backend?.service?.port?.name }}</template>
              </el-table-column>
            </el-table>
          </div>
        </template>

        <!-- ==================== Ingress: TLS ==================== -->
        <template v-if="detail.kind === 'Ingress' && (detail.spec?.tls || []).length">
          <h3 class="rd-title">TLS</h3>
          <el-table :data="detail.spec?.tls" size="small" border :header-cell-style="tableHeaderStyle">
            <el-table-column label="Hosts" min-width="200">
              <template #default="{ row }">{{ (row.hosts || []).join(', ') }}</template>
            </el-table-column>
            <el-table-column label="Secret" prop="secretName" width="200" />
          </el-table>
        </template>

        <!-- ==================== Role/ClusterRole: Rules ==================== -->
        <template v-if="(detail.kind === 'Role' || detail.kind === 'ClusterRole') && (detail.extra?.rules || []).length">
          <h3 class="rd-title">Rules</h3>
          <el-table :data="detail.extra?.rules" size="small" border :header-cell-style="tableHeaderStyle">
            <el-table-column label="API Groups" min-width="150">
              <template #default="{ row }">{{ (row.apiGroups || ['*']).join(', ') || '*' }}</template>
            </el-table-column>
            <el-table-column label="Resources" min-width="180">
              <template #default="{ row }">{{ (row.resources || []).join(', ') }}</template>
            </el-table-column>
            <el-table-column label="Verbs" min-width="200">
              <template #default="{ row }">{{ (row.verbs || []).join(', ') }}</template>
            </el-table-column>
          </el-table>
        </template>

        <!-- ==================== RoleBinding/ClusterRoleBinding ==================== -->
        <template v-if="detail.kind === 'RoleBinding' || detail.kind === 'ClusterRoleBinding'">
          <h3 class="rd-title">Role Reference</h3>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="Kind">{{ detail.extra?.roleRef?.kind || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Name">{{ detail.extra?.roleRef?.name || '-' }}</el-descriptions-item>
          </el-descriptions>

          <h3 v-if="(detail.extra?.subjects || []).length" class="rd-title">Subjects</h3>
          <el-table v-if="(detail.extra?.subjects || []).length" :data="detail.extra?.subjects" size="small" border :header-cell-style="tableHeaderStyle">
            <el-table-column label="Kind" prop="kind" width="150" />
            <el-table-column label="Name" prop="name" min-width="200" />
            <el-table-column label="Namespace" prop="namespace" width="150" />
          </el-table>
        </template>

        <!-- ==================== ConfigMap: Data Keys ==================== -->
        <template v-if="detail.kind === 'ConfigMap' && detail.extra?.data">
          <h3 class="rd-title">Data</h3>
          <div class="rd-tags-block">
            <el-tag v-for="(_, k) in detail.extra.data" :key="k" size="small" class="rd-tag">{{ k }}</el-tag>
          </div>
        </template>

        <!-- ==================== Secret: Data Keys ==================== -->
        <template v-if="detail.kind === 'Secret' && detail.extra?.data">
          <h3 class="rd-title">Data Keys</h3>
          <div class="rd-tags-block">
            <el-tag v-for="(_, k) in detail.extra.data" :key="k" size="small" class="rd-tag">{{ k }}</el-tag>
          </div>
        </template>

        <!-- ==================== Events ==================== -->
        <h3 class="rd-title">Events</h3>
        <el-table
          v-if="detail.events && detail.events.length"
          :data="detail.events"
          size="small"
          border
          :header-cell-style="tableHeaderStyle"
        >
          <el-table-column label="类型" prop="type" width="80">
            <template #default="{ row }">
              <el-tag :type="row.type === 'Warning' ? 'danger' : 'success'" size="small">{{ row.type }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="原因" prop="reason" width="140" />
          <el-table-column label="消息" prop="message" min-width="250" show-overflow-tooltip />
          <el-table-column label="次数" prop="count" width="60" align="center" />
          <el-table-column label="最后发生" prop="lastTimestamp" width="150" />
        </el-table>
        <el-empty v-else description="暂无事件" :image-size="60" />
      </template>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { ArrowRight } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getResourceDetail } from '@/api/kubernetes'
import type { GenericResourceDetail } from '@/api/kubernetes'

const props = defineProps<{
  modelValue: boolean
  clusterId: number | undefined
  kind: string
  name: string
  namespace: string
}>()

defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const loading = ref(false)
const detail = ref<GenericResourceDetail | null>(null)
const labelsOpen = ref(false)
const annotationsOpen = ref(false)

const tableHeaderStyle = { background: '#fafbfc', color: '#606266', fontWeight: '600' }

// ==================== Computed ====================

const isWorkload = computed(() => ['Deployment', 'StatefulSet', 'DaemonSet'].includes(detail.value?.kind || ''))

const hasPods = computed(() => {
  const k = detail.value?.kind || ''
  return ['Deployment', 'StatefulSet', 'DaemonSet', 'ReplicaSet', 'Job'].includes(k) && detail.value?.pods?.length
})

const replicasText = computed(() => {
  if (!detail.value) return '-'
  const spec = detail.value.spec || {}
  const status = detail.value.status || {}
  const desired = spec.replicas ?? 0
  const updated = status.updatedReplicas ?? 0
  const total = status.replicas ?? 0
  const available = status.availableReplicas ?? 0
  const unavailable = status.unavailableReplicas ?? 0
  return `${desired} desired, ${updated} updated, ${total} total, ${available} available, ${unavailable} unavailable`
})

const workloadRunning = computed(() => {
  if (!detail.value?.status?.conditions) return false
  const conditions = detail.value.status.conditions as any[]
  return conditions.some((c: any) => c.type === 'Available' && c.status === 'True')
})

// ==================== Fetch ====================

watch(() => props.modelValue, (visible) => {
  if (visible && props.clusterId && props.kind && props.name) {
    fetchDetail()
  }
  if (!visible) {
    detail.value = null
    labelsOpen.value = false
    annotationsOpen.value = false
  }
})

const fetchDetail = async () => {
  loading.value = true
  try {
    const res = await getResourceDetail(props.clusterId!, props.kind, props.namespace, props.name) as any
    detail.value = res
  } catch (error: any) {
    ElMessage.error('获取资源详情失败: ' + (error?.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// ==================== Helpers ====================

const formatTime = (isoStr: string) => {
  if (!isoStr) return '-'
  const date = new Date(isoStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffSecs = Math.floor(diffMs / 1000)
  const diffMins = Math.floor(diffSecs / 60)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  let ago = ''
  if (diffDays > 0) ago = `${diffDays}d ${diffHours % 24}h ago`
  else if (diffHours > 0) ago = `${diffHours}h ${diffMins % 60}m ago`
  else if (diffMins > 0) ago = `${diffMins}m ${diffSecs % 60}s ago`
  else ago = `${diffSecs}s ago`

  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  const timeStr = date.toLocaleTimeString('en-US', { hour12: false })

  return `${ago} (${y}年${m}月${d}日 GMT+8 ${timeStr})`
}
</script>

<style scoped>
.rd-content {
  min-height: 200px;
  padding: 0 4px;
}

.rd-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin: 20px 0 12px 0;
}

.rd-title:first-child {
  margin-top: 0;
}

/* Expandable Labels/Annotations */
.rd-expandable {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  user-select: none;
  color: #606266;
}

.rd-expandable:hover {
  color: #409eff;
}

.rd-arrow {
  transition: transform 0.2s;
  font-size: 12px;
}

.rd-arrow.rotated {
  transform: rotate(90deg);
}

.rd-tags {
  margin-top: 8px;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.rd-tag {
  margin-right: 4px;
  margin-bottom: 2px;
}

.rd-tags-block {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 4px 0;
}

.rd-muted {
  color: #909399;
  font-size: 13px;
}

/* Annotations */
.rd-anno-item {
  width: 100%;
  margin-bottom: 6px;
  padding: 6px 8px;
  background: #f5f7fa;
  border-radius: 4px;
}

.rd-anno-key {
  font-size: 12px;
  color: #909399;
  word-break: break-all;
  font-weight: 500;
}

.rd-anno-val {
  font-size: 12px;
  color: #303133;
  word-break: break-all;
  margin-top: 2px;
}

/* Endpoint */
.rd-endpoint {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  border-bottom: 1px solid #ebeef5;
}

.rd-endpoint:last-child {
  border-bottom: none;
}

.rd-endpoint-name {
  font-weight: 500;
  min-width: 120px;
}

.rd-endpoint-addr {
  color: #606266;
  font-size: 13px;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
}

/* Ingress Rules */
.rd-ingress-rule {
  margin-bottom: 12px;
}

.rd-rule-host {
  font-weight: 600;
  color: #303133;
  padding: 6px 0;
  font-size: 13px;
}
</style>
