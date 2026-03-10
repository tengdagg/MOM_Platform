import type { Plugin, PluginMenuConfig, PluginRouteConfig } from '../types'
import { pluginManager } from '../manager'

/**
 * AI 助手插件
 * 提供 AI 对话、Skill 管理、模型配置功能
 */
class AIPlugin implements Plugin {
  name = 'ai'
  description = 'AI 智能助手，提供 Agent + Skills 运维管理能力'
  version = '1.0.0'
  author = 'dat'

  async install() {
    // 初始化
  }

  async uninstall() {
    // 清理
  }

  getMenus(): PluginMenuConfig[] {
    const parentPath = '/ai'

    return [
      {
        name: 'AI 助手',
        path: parentPath,
        icon: 'ChatDotRound',
        sort: 5,
        hidden: false,
        parentPath: '',
      },
      {
        name: 'MOM Claw',
        path: '/ai/chat',
        icon: 'ChatLineRound',
        sort: 1,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: 'Skill 管理',
        path: '/ai/skills',
        icon: 'MagicStick',
        sort: 2,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '模型配置',
        path: '/ai/models',
        icon: 'Setting',
        sort: 3,
        hidden: false,
        parentPath: parentPath,
      },
      {
        name: '聊天渠道',
        path: '/ai/channels',
        icon: 'Connection',
        sort: 4,
        hidden: false,
        parentPath: parentPath,
      },
    ]
  }

  getRoutes(): PluginRouteConfig[] {
    return [
      {
        path: '/ai/chat',
        name: 'AIChat',
        component: () => import('@/views/ai/AIChat.vue'),
        meta: { title: 'MOM Claw' },
      },
      {
        path: '/ai/skills',
        name: 'AISkills',
        component: () => import('@/views/ai/AISkills.vue'),
        meta: { title: 'Skill 管理' },
      },
      {
        path: '/ai/models',
        name: 'AIModelConfig',
        component: () => import('@/views/ai/AIModelConfig.vue'),
        meta: { title: '模型配置' },
      },
      {
        path: '/ai/channels',
        name: 'AIChannels',
        component: () => import('@/views/ai/channels/Index.vue'),
        meta: { title: '聊天渠道' },
      },
      {
        path: '/ai/channels/feishu',
        name: 'AIChannelsFeishuLegacy',
        component: () => import('@/views/ai/channels/Index.vue'),
        meta: { title: '聊天渠道', hidden: true, activeMenu: '/ai/channels' },
      },
    ]
  }
}

const plugin = new AIPlugin()
pluginManager.register(plugin)

export default plugin
