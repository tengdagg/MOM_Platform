import type { Plugin, PluginMenuConfig, PluginRouteConfig } from '../types'
import { pluginManager } from '../manager'
import DomainMonitor from './components/DomainMonitor.vue'
import AlertChannels from './components/AlertChannels.vue'
import AlertReceivers from './components/AlertReceivers.vue'
import AlertLogs from './components/AlertLogs.vue'

/**
 * 监控中心插件
 * 提供域名监控、告警管理等功能
 */
class MonitorPlugin implements Plugin {
  name = 'monitor'
  prettyName = '监控中心'
  description = '监控中心插件，提供域名监控、告警管理等功能'
  version = '1.0.0'
  author = 'dat'

  /**
   * 安装插件
   */
  async install() {
    // 初始化操作
  }

  /**
   * 卸载插件
   */
  async uninstall() {
    // 清理资源
  }

  /**
   * 获取插件菜单配置
   */
  getMenus(): PluginMenuConfig[] {
    const parentPath = '/monitor'

    return [
      {
        name: '监控中心',
        path: parentPath,
        icon: 'Monitor',
        sort: 20,
        hidden: false,
        parentPath: '',
      },
      {
        name: '域名监控',
        path: '/monitor/domain',
        icon: 'Monitor',
        sort: 1,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '告警通道',
        path: '/monitor/alert-channels',
        icon: 'Bell',
        sort: 2,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '告警接收人',
        path: '/monitor/alert-receivers',
        icon: 'User',
        sort: 3,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '告警日志',
        path: '/monitor/alert-logs',
        icon: 'Document',
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
        path: '/monitor',
        name: 'Monitor',
        component: () => import('./components/DomainMonitor.vue'),
        redirect: '/monitor/domain',
        meta: { title: '监控中心' },
      },
      {
        path: '/monitor/domain',
        name: 'DomainMonitor',
        component: () => import('./components/DomainMonitor.vue'),
        meta: { title: '域名监控' },
      },
      {
        path: '/monitor/alert-channels',
        name: 'AlertChannels',
        component: () => import('./components/AlertChannels.vue'),
        meta: { title: '告警通道' },
      },
      {
        path: '/monitor/alert-receivers',
        name: 'AlertReceivers',
        component: () => import('./components/AlertReceivers.vue'),
        meta: { title: '告警接收人' },
      },
      {
        path: '/monitor/alert-logs',
        name: 'AlertLogs',
        component: () => import('./components/AlertLogs.vue'),
        meta: { title: '告警日志' },
      },
    ]
  }
}

// 创建并注册插件实例
const plugin = new MonitorPlugin()
pluginManager.register(plugin)

export default plugin
