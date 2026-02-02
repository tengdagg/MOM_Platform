import { defineStore } from 'pinia'

interface KubernetesState {
    selectedClusterId: number | null
    selectedNamespaces: string[]
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
                    selectedNamespaces: parsed.selectedNamespaces || []
                }
            } catch {
                // 解析失败，使用默认值
            }
        }
        return {
            selectedClusterId: null,
            selectedNamespaces: []
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
        }
    },

    actions: {
        // 设置集群
        setCluster(clusterId: number | null) {
            this.selectedClusterId = clusterId
            this.persistState()
        },

        // 设置命名空间（多选）
        setNamespaces(namespaces: string[]) {
            this.selectedNamespaces = namespaces
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
                selectedNamespaces: this.selectedNamespaces
            }))
        },

        // 重置状态
        reset() {
            this.selectedClusterId = null
            this.selectedNamespaces = []
            localStorage.removeItem(STORAGE_KEY)
        }
    }
})
