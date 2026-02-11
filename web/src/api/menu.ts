import request from '@/utils/request'

// 获取菜单树（仅可见菜单）
export const getMenuTree = () => {
  return request.get('/api/v1/menus/tree')
}

// 获取所有菜单树（包括隐藏菜单，用于菜单管理页面）
export const getAllMenuTree = () => {
  return request.get('/api/v1/menus/tree', { params: { all: 'true' } })
}

// 获取当前用户的菜单树
export const getUserMenu = () => {
  return request.get('/api/v1/menus/user')
}

// 获取菜单详情
export const getMenu = (id: number) => {
  return request.get(`/api/v1/menus/${id}`)
}

// 创建菜单
export const createMenu = (data: any) => {
  return request.post('/api/v1/menus', data)
}

// 更新菜单
export const updateMenu = (id: number, data: any) => {
  return request.put(`/api/v1/menus/${id}`, data)
}

// 更新菜单排序（仅更新sort字段）
export const updateMenuSort = (id: number, sort: number) => {
  return request.put(`/api/v1/menus/${id}/sort`, { sort })
}

// 批量更新菜单排序
export const batchUpdateMenuSort = (items: Array<{ id: number; sort: number }>) => {
  return request.put('/api/v1/menus/sort', { items })
}

// 删除菜单
export const deleteMenu = (id: number) => {
  return request.delete(`/api/v1/menus/${id}`)
}
