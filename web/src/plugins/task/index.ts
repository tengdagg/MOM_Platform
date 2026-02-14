import type { Plugin, PluginMenuConfig, PluginRouteConfig } from '../types'
import { pluginManager } from '../manager'

/**
 * Task 任务中心插件
 * 提供执行任务、模板管理和文件分发功能
 */
class TaskPlugin implements Plugin {
  name = 'task'
  description = '任务中心插件，提供执行任务、模板管理和文件分发功能'
  version = '1.0.0'
  author = 'dat'

  /**
   * 安装插件
   */
  async install() {
  }

  /**
   * 卸载插件
   */
  async uninstall() {
  }

  /**
   * 获取插件菜单配置
   */
  getMenus(): PluginMenuConfig[] {
    const parentPath = '/task'

    return [
      {
        name: '任务中心',
        path: parentPath,
        icon: 'Tickets',
        sort: 90,
        hidden: false,
        parentPath: '',
      },
      {
        name: '执行任务',
        path: '/task/execute',
        icon: 'VideoPlay',
        sort: 1,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '模板管理',
        path: '/task/templates',
        icon: 'Document',
        sort: 2,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '文件分发',
        path: '/task/file-distribution',
        icon: 'FolderOpened',
        sort: 3,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '执行记录',
        path: '/task/execution-history',
        icon: 'Notebook',
        sort: 4,
        hidden: false,
        parentPath: parentPath,
      },
    ]
  }

  /**
   * 获取插件路由配置
   */
  getRoutes(): PluginRouteConfig[] {
    return [
      {
        path: '/task',
        name: 'Task',
        component: () => import('@/views/task/Index.vue'),
        meta: { title: '任务中心' },
        children: [
          {
            path: 'execute',
            name: 'TaskExecute',
            component: () => import('@/views/task/Execute.vue'),
            meta: { title: '执行任务' },
          },
          {
            path: 'templates',
            name: 'TaskTemplates',
            component: () => import('@/views/task/Templates.vue'),
            meta: { title: '模板管理' },
          },
          {
            path: 'file-distribution',
            name: 'TaskFileDistribution',
            component: () => import('@/views/task/FileDistribution.vue'),
            meta: { title: '文件分发' },
          },
          {
            path: 'execution-history',
            name: 'TaskExecutionHistory',
            component: () => import('@/views/task/ExecutionHistory.vue'),
            meta: { title: '执行记录' },
          },
        ],
      },
    ]
  }
}

// 创建并注册插件实例
const plugin = new TaskPlugin()
pluginManager.register(plugin)

export default plugin
