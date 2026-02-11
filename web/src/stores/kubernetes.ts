import { defineStore } from 'pinia'

interface KubernetesState {
    selectedClusterId: number | null
    selectedNamespaces: string[]
    fullNamespaceAccess: boolean  // 是否拥有所有命名空间访问权限
    canWrite: boolean             // 是否拥有写入权限（安装/升级/删除等操作）
}

const STORAGE_KEY = 'kubernetes_global_state'

export const useKubernetesStore = defineStore('kubernetes', {
    state: (): KubernetesState => {
        // 从 localStorage 恢复状态
        const saved = localStorage.getItem(STORAGE_KEY)
        if (saved) {
            try {
                const parsed = JSON.parse(saved)
                return {
                    selectedClusterId: parsed.selectedClusterId || null,
                    selectedNamespaces: parsed.selectedNamespaces || [],
                    fullNamespaceAccess: parsed.fullNamespaceAccess !== false, // 默认 true
                    canWrite: parsed.canWrite !== false // 默认 true
                }
            } catch {
                // 解析失败，使用默认值
            }
        }
        return {
            selectedClusterId: null,
            selectedNamespaces: [],
            fullNamespaceAccess: true,
            canWrite: true
        }
    },

    getters: {
        // 是否选择了命名空间
        hasNamespaceFilter: (state) => state.selectedNamespaces.length > 0,

        // 获取命名空间过滤参数（用于 API 请求）
        namespaceParam: (state) => {
            if (state.selectedNamespaces.length === 0) return ''
            if (state.selectedNamespaces.length === 1) return state.selectedNamespaces[0]
            return state.selectedNamespaces.join(',')
        },

        // 是否为只读模式（没有写入权限）
        isReadOnly: (state) => !state.canWrite
    },

    actions: {
        // 设置集群
        setCluster(clusterId: number | null) {
            this.selectedClusterId = clusterId
            this.persistState()
        },

        // 设置命名空间（多选）
        setNamespaces(namespaces: string[]) {
            // 当用户没有全部命名空间访问权限时，不允许选择"所有命名空间"
            if (!this.fullNamespaceAccess) {
                namespaces = namespaces.filter(ns => ns !== '')
                // 如果过滤后为空，保持当前选择不变（防止清空到"全部"）
                if (namespaces.length === 0 && this.selectedNamespaces.length > 0) {
                    return
                }
            }
            this.selectedNamespaces = namespaces
            this.persistState()
        },

        // 设置是否拥有全部命名空间访问权限
        setFullNamespaceAccess(fullAccess: boolean) {
            this.fullNamespaceAccess = fullAccess
            this.persistState()
        },

        // 设置是否拥有写入权限
        setCanWrite(canWrite: boolean) {
            this.canWrite = canWrite
            this.persistState()
        },

        // 清空命名空间选择
        clearNamespaces() {
            this.selectedNamespaces = []
            this.persistState()
        },

        // 持久化状态到 localStorage
        persistState() {
            localStorage.setItem(STORAGE_KEY, JSON.stringify({
                selectedClusterId: this.selectedClusterId,
                selectedNamespaces: this.selectedNamespaces,
                fullNamespaceAccess: this.fullNamespaceAccess,
                canWrite: this.canWrite
            }))
        },

        // 重置状态
        reset() {
            this.selectedClusterId = null
            this.selectedNamespaces = []
            this.fullNamespaceAccess = true
            this.canWrite = true
            localStorage.removeItem(STORAGE_KEY)
        }
    }
})
