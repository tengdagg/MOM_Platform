import request from '@/utils/request'

// ==================== 旧版接口（保留兼容） ====================

export const getAssetPermissions = (params: {
  page: number
  pageSize: number
  roleId?: number
  assetGroupId?: number
}) => {
  return request.get('/api/v1/asset-permissions', { params })
}

export const createAssetPermission = (data: {
  roleId: number
  assetGroupId: number
  assetType?: string
  hostIds: number[]
  permissions?: number
}) => {
  return request.post('/api/v1/asset-permissions', data)
}

export const deleteAssetPermission = (id: number) => {
  return request.delete(`/api/v1/asset-permissions/${id}`)
}

export const getAssetPermissionDetail = (id: number) => {
  return request.get(`/api/v1/asset-permissions/${id}`)
}

export const updateAssetPermission = (id: number, data: {
  roleId: number
  assetGroupId: number
  assetType?: string
  hostIds: number[]
  permissions?: number
}) => {
  return request.put(`/api/v1/asset-permissions/${id}`, data)
}

export const getAssetPermissionsByRole = (roleId: number) => {
  return request.get(`/api/v1/asset-permissions/role/${roleId}`)
}

export const getAssetPermissionsByGroup = (assetGroupId: number) => {
  return request.get(`/api/v1/asset-permissions/group/${assetGroupId}`)
}

// ==================== 新版授权接口 ====================

// 获取资产树（分组+具体资产）
export const getAssetTree = () => {
  return request.get('/api/v1/asset-authorizations/asset-tree')
}

// 获取授权规则列表
export const getAssetAuthorizations = (params: {
  page: number
  pageSize: number
  assetGroupId?: number
  keyword?: string
}) => {
  return request.get('/api/v1/asset-authorizations', { params })
}

// 创建授权规则
export const createAssetAuthorization = (data: {
  name: string
  userIds?: number[]
  departmentIds?: number[]
  assetGroupId: number
  assetType?: string
  assetIds?: number[]
  permissions: number
  startDate?: string
  expireDate?: string
  isActive?: boolean
  description?: string
}) => {
  return request.post('/api/v1/asset-authorizations', data)
}

// 更新授权规则
export const updateAssetAuthorization = (id: number, data: {
  name?: string
  userIds?: number[]
  departmentIds?: number[]
  assetGroupId?: number
  assetType?: string
  assetIds?: number[]
  permissions?: number
  startDate?: string
  expireDate?: string
  isActive?: boolean
  description?: string
}) => {
  return request.put(`/api/v1/asset-authorizations/${id}`, data)
}

// 删除授权规则
export const deleteAssetAuthorization = (id: number) => {
  return request.delete(`/api/v1/asset-authorizations/${id}`)
}

// 获取授权规则详情
export const getAssetAuthorizationDetail = (id: number) => {
  return request.get(`/api/v1/asset-authorizations/${id}`)
}

// 获取当前用户对指定主机的操作权限
export const getUserHostPermissions = (hostId: number) => {
  return request.get('/api/v1/asset-authorizations/user/host', {
    params: { hostId }
  })
}

// 获取当前用户对指定网络设备的操作权限
export const getUserDevicePermissions = (deviceId: number) => {
  return request.get('/api/v1/asset-authorizations/user/device', {
    params: { deviceId }
  })
}
