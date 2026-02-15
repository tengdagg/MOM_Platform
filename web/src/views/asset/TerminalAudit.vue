<template>
  <div class="terminal-audit-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Monitor /></el-icon>
        </div>
        <div>
          <h2 class="page-title">终端审计</h2>
          <p class="page-subtitle">查看和管理终端会话记录（SSH / RDP / Telnet）</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button @click="handleManualCleanup" :loading="cleaning" plain>
          <el-icon style="margin-right: 6px;"><Delete /></el-icon>
          清理过期记录
        </el-button>
        <el-button class="black-button" @click="retentionDialogVisible = true">
          <el-icon style="margin-right: 6px;"><Setting /></el-icon>
          保留配置
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索主机名、IP或用户名..."
          clearable
          class="search-input"
          @keyup.enter="loadSessions"
          @clear="loadSessions"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <div class="search-actions">
        <el-button class="reset-btn" @click="handleRefresh">
          <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <!-- 表格和分页容器 -->
    <div class="table-wrapper">
      <el-table
        :data="filteredSessions"
        v-loading="loading"
        class="modern-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
      >
        <el-table-column prop="id" label="ID" width="80" align="center" />

        <el-table-column label="类型" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.sessionType === 'rdp' ? 'warning' : 'primary'" size="small">
              {{ row.sessionTypeText || 'SSH' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="主机信息" min-width="220">
          <template #default="{ row }">
            <div class="host-info">
              <div class="host-name">
                <el-icon><Monitor /></el-icon>
                <span>{{ row.hostName }}</span>
              </div>
              <div class="host-ip">{{ row.hostIp }}</div>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="username" label="操作用户" min-width="150" align="center">
          <template #default="{ row }">
            <el-tooltip :content="row.username" placement="top">
              <el-tag type="info" class="username-tag">
                <el-icon><User /></el-icon>
                <span class="username-text">{{ row.username }}</span>
              </el-tag>
            </el-tooltip>
          </template>
        </el-table-column>

        <el-table-column prop="durationText" label="时长" min-width="100" align="center" />

        <el-table-column prop="fileSizeText" label="文件大小" min-width="110" align="center" />

        <el-table-column prop="statusText" label="状态" min-width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">{{ row.statusText }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="createdAtText" label="创建时间" min-width="180" align="center" />

        <el-table-column label="操作" width="160" align="center" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip :content="row.sessionType === 'rdp' ? 'RDP会话暂不支持回放' : '播放'" placement="top">
                <el-button
                  link
                  class="action-btn action-play"
                  @click="handlePlay(row)"
                  :loading="playingSession === row.id"
                  :disabled="row.sessionType === 'rdp'"
                >
                  <el-icon><VideoPlay /></el-icon>
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

    <!-- 播放对话框 -->
    <el-dialog
      v-model="playerVisible"
      :title="`终端回放 - ${currentSession?.hostName}`"
      width="80%"
      top="5vh"
      class="terminal-player-dialog responsive-dialog"
      :close-on-click-modal="false"
      @close="handlePlayerClose"
    >
      <AsciinemaPlayer
        v-if="recordingUrl && playerVisible"
        :src="recordingUrl"
        :autoplay="true"
      />
    </el-dialog>

    <!-- 保留配置对话框 -->
    <el-dialog
      v-model="retentionDialogVisible"
      title="审计记录保留配置"
      width="480px"
      class="retention-dialog"
      :close-on-click-modal="false"
    >
      <el-form :model="retentionForm" label-width="130px">
        <el-form-item label="保留天数">
          <el-input-number
            v-model="retentionForm.retentionDays"
            :min="1"
            :max="3650"
            :step="1"
            style="width: 200px;"
          />
          <span style="margin-left: 10px; color: #909399; font-size: 13px;">天</span>
        </el-form-item>
        <el-form-item label="自动清理">
          <el-switch v-model="retentionForm.autoCleanup" />
          <span style="margin-left: 10px; color: #909399; font-size: 12px;">每天自动清理超过保留天数的记录</span>
        </el-form-item>
        <el-alert
          type="info"
          :closable="false"
          style="margin-top: 8px;"
        >
          超过设定保留天数的终端会话记录和录制文件将被自动清理。正在录制中的会话不会被清理。
        </el-alert>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="retentionDialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleSaveRetention" :loading="savingRetention">保存</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Search,
  Monitor,
  User,
  VideoPlay,
  Delete,
  RefreshLeft,
  Setting
} from '@element-plus/icons-vue'
import { getTerminalSessions, playTerminalSession, deleteTerminalSession, getRetentionConfig, updateRetentionConfig, cleanupExpiredSessions } from '@/api/terminal'
import AsciinemaPlayer from '@/components/AsciinemaPlayer.vue'

interface TerminalSession {
  id: number
  sessionType: string
  sessionTypeText: string
  hostId: number
  hostName: string
  hostIp: string
  userId: number
  username: string
  duration: number
  durationText: string
  fileSize: number
  fileSizeText: string
  status: string
  statusText: string
  createdAt: string
  createdAtText: string
}

const loading = ref(false)
const sessions = ref<TerminalSession[]>([])
const searchKeyword = ref('')
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

// 播放相关
const playerVisible = ref(false)
const recordingUrl = ref('')
const currentSession = ref<TerminalSession | null>(null)
const playingSession = ref(0)

// 删除相关
const deletingSession = ref(0)

// 过滤后的会话列表
const filteredSessions = computed(() => {
  if (!searchKeyword.value) {
    return sessions.value
  }

  const keyword = searchKeyword.value.toLowerCase()
  return sessions.value.filter(item =>
    item.hostName?.toLowerCase().includes(keyword) ||
    item.hostIp?.toLowerCase().includes(keyword) ||
    item.username?.toLowerCase().includes(keyword)
  )
})

// 加载会话列表
const loadSessions = async () => {
  loading.value = true
  try {
    const response = await getTerminalSessions({
      page: page.value,
      pageSize: pageSize.value,
      keyword: searchKeyword.value
    })
    sessions.value = response.list || []
    total.value = response.total || 0
  } catch (error: any) {
    ElMessage.error('加载会话列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 播放会话
const handlePlay = async (session: TerminalSession) => {
  playingSession.value = session.id
  try {
    const response = await playTerminalSession(session.id)

    // 创建Blob URL
    const blob = new Blob([response], { type: 'application/json' })
    recordingUrl.value = URL.createObjectURL(blob)
    currentSession.value = session
    playerVisible.value = true
  } catch (error: any) {
    ElMessage.error('加载录制文件失败: ' + (error.message || '未知错误'))
  } finally {
    playingSession.value = 0
  }
}

// 删除会话
const handleDeleteClick = (row: TerminalSession) => {
  ElMessageBox.confirm('确定删除此会话录制吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    await handleDelete(row.id)
  }).catch(() => {})
}

const handleDelete = async (id: number) => {
  deletingSession.value = id
  try {
    await deleteTerminalSession(id)
    ElMessage.success('删除成功')
    loadSessions()
  } catch (error: any) {
    ElMessage.error('删除失败: ' + (error.message || '未知错误'))
  } finally {
    deletingSession.value = 0
  }
}

// 刷新
const handleRefresh = () => {
  searchKeyword.value = ''
  page.value = 1
  loadSessions()
}

// 分页变化
const handleSizeChange = () => {
  page.value = 1
  loadSessions()
}

const handlePageChange = () => {
  loadSessions()
}

// 关闭播放器
const handlePlayerClose = () => {
  if (recordingUrl.value) {
    URL.revokeObjectURL(recordingUrl.value)
    recordingUrl.value = ''
  }
  currentSession.value = null
}

// 获取状态类型
const getStatusType = (status: string): 'success' | 'info' | 'warning' | 'danger' => {
  const typeMap: Record<string, 'success' | 'info' | 'warning' | 'danger'> = {
    completed: 'success',
    recording: 'warning',
    failed: 'danger'
  }
  return typeMap[status] || 'info'
}

// 保留配置相关
const retentionDialogVisible = ref(false)
const savingRetention = ref(false)
const cleaning = ref(false)
const retentionForm = reactive({
  retentionDays: 30,
  autoCleanup: true
})

// 加载保留配置
const loadRetentionConfig = async () => {
  try {
    const data: any = await getRetentionConfig()
    if (data) {
      retentionForm.retentionDays = data.retentionDays || 30
      retentionForm.autoCleanup = data.autoCleanup !== false
    }
  } catch { /* ignore */ }
}

// 保存保留配置
const handleSaveRetention = async () => {
  savingRetention.value = true
  try {
    await updateRetentionConfig({
      retentionDays: retentionForm.retentionDays,
      autoCleanup: retentionForm.autoCleanup
    })
    ElMessage.success('保留配置已保存')
    retentionDialogVisible.value = false
  } catch (error: any) {
    ElMessage.error('保存失败: ' + (error.message || '未知错误'))
  } finally {
    savingRetention.value = false
  }
}

// 手动清理
const handleManualCleanup = async () => {
  try {
    await ElMessageBox.confirm(
      `确定清理超过 ${retentionForm.retentionDays} 天的审计记录吗？此操作不可恢复。`,
      '清理确认',
      {
        confirmButtonText: '确定清理',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  } catch {
    return
  }

  cleaning.value = true
  try {
    const res: any = await cleanupExpiredSessions()
    ElMessage.success(res?.message || '清理完成')
    loadSessions()
  } catch (error: any) {
    ElMessage.error('清理失败: ' + (error.message || '未知错误'))
  } finally {
    cleaning.value = false
  }
}

onMounted(() => {
  loadSessions()
  loadRetentionConfig()
})
</script>

<style scoped>
.terminal-audit-container {
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

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.retention-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.retention-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.retention-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
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

/* 主机信息 */
.host-info {
  .host-name {
    display: flex;
    align-items: center;
    gap: 6px;
    font-weight: 500;
    color: #303133;
    margin-bottom: 4px;

    :deep(.el-icon) {
      color: #409eff;
    }
  }

  .host-ip {
    font-size: 12px;
    color: #909399;
    font-family: 'Consolas', 'Monaco', monospace;
  }
}

/* 用户名标签 */
.username-tag {
  min-width: 120px;
  max-width: 130px;
  display: inline-flex !important;
  flex-direction: row !important;
  align-items: center !important;
  justify-content: center;
  gap: 6px;
  padding: 0 12px;

  :deep(.el-icon) {
    font-size: 14px;
    display: inline-block;
    vertical-align: middle;
    flex-shrink: 0;
  }

  .username-text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
    min-width: 0;
  }
}

:deep(.username-tag .el-tag__content) {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 6px;
  width: 100%;
  overflow: hidden;
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

.action-play:hover {
  background-color: #e8f4ff;
  color: #409eff;
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

/* 播放对话框样式 */
:deep(.terminal-player-dialog) {
  border-radius: 0;
}

:deep(.terminal-player-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.terminal-player-dialog .el-dialog__body) {
  padding: 20px;
  background: #000;
}

:deep(.terminal-player-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

/* 标签样式 */
:deep(.el-tag) {
  border-radius: 0;
  padding: 4px 10px;
  font-weight: 500;
}

/* 响应式对话框 */
:deep(.responsive-dialog) {
  max-width: 95vw;
  min-width: 500px;
}

@media (max-width: 768px) {
  :deep(.responsive-dialog .el-dialog) {
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
}
</style>
