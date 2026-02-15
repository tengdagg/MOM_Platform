import { usePermissionStore } from '@/stores/permission'

/**
 * 按钮级权限 composable
 * 用法：
 *   const { hasPerm, hasAnyPerm } = useButtonPermission()
 *   v-if="hasPerm('host:create')"
 */
export function useButtonPermission() {
  const permissionStore = usePermissionStore()

  const hasPerm = (code: string): boolean => {
    return permissionStore.hasPerm(code)
  }

  const hasAnyPerm = (...codes: string[]): boolean => {
    return permissionStore.hasAnyPerm(...codes)
  }

  return {
    hasPerm,
    hasAnyPerm,
  }
}
