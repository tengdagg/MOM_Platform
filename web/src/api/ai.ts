import request from '@/utils/request'

// ==================== 模型配置 ====================

export const getModelList = () => {
  return request.get('/api/v1/plugins/ai/models')
}

export const createModel = (data: any) => {
  return request.post('/api/v1/plugins/ai/models', data)
}

export const updateModel = (id: number, data: any) => {
  return request.put(`/api/v1/plugins/ai/models/${id}`, data)
}

export const deleteModel = (id: number) => {
  return request.delete(`/api/v1/plugins/ai/models/${id}`)
}

export const testModel = (id: number) => {
  return request.post(`/api/v1/plugins/ai/models/${id}/test`)
}

export const setDefaultModel = (id: number) => {
  return request.put(`/api/v1/plugins/ai/models/${id}/default`)
}

// ==================== 对话管理 ====================

export const createSession = (data?: { title?: string; modelId?: number }) => {
  return request.post('/api/v1/plugins/ai/chat/sessions', data || {})
}

export const getSessions = () => {
  return request.get('/api/v1/plugins/ai/chat/sessions')
}

export const deleteSession = (id: number) => {
  return request.delete(`/api/v1/plugins/ai/chat/sessions/${id}`)
}

export const getSessionMessages = (id: number) => {
  return request.get(`/api/v1/plugins/ai/chat/sessions/${id}/messages`)
}

export const sendMessage = (data: { sessionId: number; content: string; modelId?: number }) => {
  return request.post('/api/v1/plugins/ai/chat/send', data)
}

// ==================== Skill 管理 ====================

export const getSkillList = (category?: string) => {
  return request.get('/api/v1/plugins/ai/skills', { params: { category } })
}

export const toggleSkill = (id: number) => {
  return request.put(`/api/v1/plugins/ai/skills/${id}/toggle`)
}

export const uploadSkill = (file: File) => {
  const formData = new FormData()
  formData.append('file', file)
  return request.post('/api/v1/plugins/ai/skills/upload', formData)
}

export const deleteSkill = (id: number) => {
  return request.delete(`/api/v1/plugins/ai/skills/${id}`)
}

// ==================== 对话模板 ====================

export const getTemplates = () => {
  return request.get('/api/v1/plugins/ai/templates')
}

// ==================== WebSocket ====================

export function createChatWebSocket(): WebSocket {
  const token = localStorage.getItem('token') || ''
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  return new WebSocket(`${protocol}//${host}/api/v1/plugins/ai/chat/ws?token=${token}`)
}
