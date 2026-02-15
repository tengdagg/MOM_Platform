import request from '@/utils/request'

// 终端审计API

/**
 * 获取终端会话列表
 */
export const getTerminalSessions = (params: {
  page: number
  pageSize: number
  keyword?: string
}) => {
  return request.get('/api/v1/terminal-sessions', { params })
}

/**
 * 播放终端会话录制
 */
export const playTerminalSession = (id: number) => {
  return request.get(`/api/v1/terminal-sessions/${id}/play`, {
    responseType: 'text'
  })
}

/**
 * 删除终端会话
 */
export const deleteTerminalSession = (id: number) => {
  return request.delete(`/api/v1/terminal-sessions/${id}`)
}

/**
 * 获取终端审计保留配置
 */
export const getRetentionConfig = () => {
  return request.get('/api/v1/terminal-sessions/retention')
}

/**
 * 更新终端审计保留配置
 */
export const updateRetentionConfig = (data: { retentionDays: number; autoCleanup: boolean }) => {
  return request.put('/api/v1/terminal-sessions/retention', data)
}

/**
 * 手动清理过期会话
 */
export const cleanupExpiredSessions = () => {
  return request.post('/api/v1/terminal-sessions/cleanup')
}
