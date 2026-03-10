import request from '@/utils/request'

export interface AIChannelConversationPolicy {
  sessionMaxMessages?: number
  sessionIdleHours?: number
  sessionMaxAgeDays?: number
}

export interface AIChannelExtraConfig extends AIChannelConversationPolicy {
  [key: string]: any
}

export interface AIChannelConfig {
  id?: number
  name: string
  channelType: 'feishu' | 'wecom' | 'dingtalk'
  enabled: boolean
  status?: string
  appId: string
  appSecret?: string
  appSecretSet?: boolean
  defaultModelId?: number
  executeAsUserId?: number
  config?: AIChannelExtraConfig
  lastError?: string
  lastConnectedAt?: string
  createdAt?: string
  updatedAt?: string
}

export const getChannelList = () => {
  return request.get('/api/v1/plugins/ai/channels')
}

export const getChannelDetail = (id: number) => {
  return request.get(`/api/v1/plugins/ai/channels/${id}`)
}

export const createChannel = (data: AIChannelConfig) => {
  return request.post('/api/v1/plugins/ai/channels', data)
}

export const updateChannel = (id: number, data: AIChannelConfig) => {
  return request.put(`/api/v1/plugins/ai/channels/${id}`, data)
}

export const deleteChannel = (id: number) => {
  return request.delete(`/api/v1/plugins/ai/channels/${id}`)
}

export const testChannel = (id: number) => {
  return request.post(`/api/v1/plugins/ai/channels/${id}/test`)
}

export const startChannel = (id: number) => {
  return request.post(`/api/v1/plugins/ai/channels/${id}/start`)
}

export const stopChannel = (id: number) => {
  return request.post(`/api/v1/plugins/ai/channels/${id}/stop`)
}

export const reconnectChannel = (id: number) => {
  return request.post(`/api/v1/plugins/ai/channels/${id}/reconnect`)
}
