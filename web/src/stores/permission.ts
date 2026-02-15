import { defineStore } from 'pinia'
import { getUserPermissions } from '@/api/menu'

interface PermissionState {
  /** 当前用户拥有的按钮权限编码列表 */
  buttonCodes: string[]
  /** 是否已加载 */
  loaded: boolean
}

export const usePermissionStore = defineStore('permission', {
  state: (): PermissionState => ({
    buttonCodes: [],
    loaded: false,
  }),

  actions: {
    /** 从后端加载当前用户的按钮权限 */
    async loadPermissions() {
      try {
        const res = await getUserPermissions()
        // res 经过 Axios 拦截器已经是 data 部分
        this.buttonCodes = Array.isArray(res) ? res : []
        this.loaded = true
      } catch (e) {
        console.warn('[PermissionStore] 加载按钮权限失败', e)
        this.buttonCodes = []
        this.loaded = true
      }
    },

    /** 检查用户是否拥有指定按钮权限 */
    hasPerm(code: string): boolean {
      return this.buttonCodes.includes(code)
    },

    /** 检查用户是否拥有指定权限中的任意一个 */
    hasAnyPerm(...codes: string[]): boolean {
      return codes.some((code) => this.buttonCodes.includes(code))
    },

    /** 重置权限（退出登录时调用） */
    reset() {
      this.buttonCodes = []
      this.loaded = false
    },
  },
})
