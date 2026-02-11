<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside :width="isCollapse ? '64px' : '200px'" v-if="!hideSidebar" class="sidebar">
      <div class="logo" @click="router.push('/')">
        <img v-if="!isCollapse" :src="logoImage" alt="Logo" class="logo-image" />
        <img v-else :src="logoIcon" alt="Logo" class="logo-image-mini" />
      </div>

      <el-menu
        :default-active="activeMenu"
        class="el-menu-vertical"
        :collapse="isCollapse"
        :collapse-transition="false"
        router
        :unique-opened="true"
        background-color="#0a466a"
        text-color="#ffffff"
        active-text-color="#ffffff"
      >
        <template v-for="menu in menuList" :key="menu.ID || menu.id || menu.path">
          <!-- 有子菜单的情况 -->
          <el-sub-menu 
            v-if="menu.children && menu.children.length > 0" 
            :index="String(menu.ID || menu.id || menu.path)" 
            :class="{ 'menu-disabled': menu.status === 0 }"
          >
            <template #title>
              <el-icon><component :is="getIcon(menu.icon)" :name="menu.icon" /></el-icon>
              <span>{{ menu.name }}</span>
            </template>
            <template v-for="subMenu in menu.children" :key="subMenu.ID || subMenu.id || subMenu.path">
              <!-- support 3rd level menu -->
              <el-sub-menu
                v-if="subMenu.children && subMenu.children.length > 0"
                :index="String(subMenu.ID || subMenu.id || subMenu.path)"
                :class="{ 'menu-disabled': subMenu.status === 0 }"
              >
                <template #title>
                  <el-icon><component :is="getIcon(subMenu.icon)" :name="subMenu.icon" /></el-icon>
                  <span>{{ subMenu.name }}</span>
                </template>
                <el-menu-item
                  v-for="child in subMenu.children"
                  :key="child.ID || child.id || child.path"
                  :index="child.status === 0 ? undefined : child.path"
                  :class="{ 'menu-disabled': child.status === 0 }"
                >
                  <el-icon><component :is="getIcon(child.icon)" :name="child.icon" /></el-icon>
                  <span>{{ child.name }}</span>
                </el-menu-item>
              </el-sub-menu>
              <el-menu-item
                v-else
                :index="subMenu.status === 0 ? undefined : subMenu.path"
                :class="{ 'menu-disabled': subMenu.status === 0 }"
              >
                <el-icon><component :is="getIcon(subMenu.icon)" :name="subMenu.icon" /></el-icon>
                <span>{{ subMenu.name }}</span>
              </el-menu-item>
            </template>
          </el-sub-menu>

          <!-- 没有子菜单的情况 -->
          <el-menu-item
            v-else
            :index="menu.status === 0 ? undefined : menu.path"
            :class="{ 'menu-disabled': menu.status === 0 }"
          >
            <el-icon><component :is="getIcon(menu.icon)" :name="menu.icon" /></el-icon>
            <span>{{ menu.name }}</span>
          </el-menu-item>
        </template>
      </el-menu>

      <!-- 收缩按钮 -->
      <div class="collapse-btn" @click="toggleCollapse">
        <el-icon :size="18">
          <Fold v-if="!isCollapse" />
          <Expand v-else />
        </el-icon>
      </div>
    </el-aside>

    <el-container>
      <!-- Header -->
      <el-header class="main-header">
        <div class="header-left">
          <el-breadcrumb separator="/" class="breadcrumb">
            <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
            <el-breadcrumb-item v-if="currentRoute.meta.title">
              {{ currentRoute.meta.title }}
            </el-breadcrumb-item>
          </el-breadcrumb>
        </div>

        <!-- 用户信息区域 - 右侧 -->
        <div class="header-right">
          <el-dropdown trigger="click" @command="handleUserCommand">
            <div class="user-info">
              <el-avatar :size="32" :src="avatarUrl" class="user-avatar">
                <el-icon><UserFilled /></el-icon>
              </el-avatar>
              <div class="user-details">
                <span class="user-name">{{ userStore.userInfo?.realName || userStore.userInfo?.username }}</span>
                <span class="user-role">{{ userRoleDisplay }}</span>
              </div>
              <el-icon class="dropdown-arrow"><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">
                  <el-icon><User /></el-icon>
                  <span>个人信息</span>
                </el-dropdown-item>
                <el-dropdown-item command="logout" divided>
                  <el-icon><SwitchButton /></el-icon>
                  <span>退出登录</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main>
        <NoPermission v-if="hasNoPermission" />
        <router-view v-else />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'
import NoPermission from '@/views/NoPermission.vue'
import {
  HomeFilled,
  User,
  UserFilled,
  OfficeBuilding,
  Menu,
  SwitchButton,
  ArrowDown,
  Platform,
  Setting,
  Document,
  Tools,
  Monitor,
  FolderOpened,
  Connection,
  Files,
  Lock,
  View,
  Odometer,
  Tickets,
  List,
  Grid,
  Cloudy,
  Grape,
  House,
  Fold,
  Expand
} from '@element-plus/icons-vue'
import CustomIcons from '@/components/icons/CustomIcons.vue'
import { getUserMenu } from '@/api/menu'
import { pluginManager } from '@/plugins/manager'

// Logo 和 Header 图片路径
const logoImage = '/logo.png'
const logoIcon = '/logo_icon.png'  // 收起时显示的图标
const headerImage = '/header.png'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

// 菜单收缩状态
const isCollapse = ref(false)

// 切换收缩状态
const toggleCollapse = () => {
  isCollapse.value = !isCollapse.value
}

const activeMenu = computed(() => {
  if (route.meta?.activeMenu) {
    return route.meta.activeMenu as string
  }
  return route.path
})

// 是否隐藏侧边栏
const hideSidebar = computed(() => {
  return route.meta?.hideSidebar === true || false
})

// 头像URL
const avatarUrl = computed(() => {
  const avatar = userStore.userInfo?.avatar || ''
  if (!avatar) return ''
  if (avatar.startsWith('data:')) return avatar
  const separator = avatar.includes('?') ? '&' : '?'
  return `${avatar}${separator}t=${userStore.avatarTimestamp}`
})

const currentRoute = computed(() => route)

// 获取用户角色显示名称
const userRoleDisplay = computed(() => {
  const roles = userStore.userInfo?.roles || []
  if (roles.length === 0) return '普通用户'
  const adminRole = roles.find((r: any) => r.code === 'admin')
  if (adminRole) {
    return adminRole.name || '管理员'
  }
  return roles[0]?.name || '普通用户'
})

const menuList = ref<any[]>([])
const hasNoPermission = ref(false)

// 图标映射
const iconMap: Record<string, any> = {
  'HomeFilled': HomeFilled,
  'User': User,
  'UserFilled': UserFilled,
  'OfficeBuilding': OfficeBuilding,
  'Menu': Menu,
  'Platform': Platform,
  'Setting': Setting,
  'Document': Document,
  'Tools': Tools,
  'Monitor': Monitor,
  'FolderOpened': FolderOpened,
  'Connection': Connection,
  'Files': Files,
  'Lock': Lock,
  'View': View,
  'Odometer': Odometer,
  'Tickets': Tickets,
  'List': List,
  'Grid': Grid,
  'Cloudy': Cloudy,
  'Grape': Grape,
  'House': House,
  'Helm': CustomIcons,
  'Kubernetes': CustomIcons
}

const getIcon = (iconName: string) => {
  return iconMap[iconName] || Menu
}

// 从插件管理器构建菜单
const buildPluginMenus = async (authorizedPaths: Set<string>) => {
  const pluginMenus: any[] = []
  const allPlugins = pluginManager.getAll()
  const roles = userStore.userInfo?.roles || []
  const isSuperAdmin = roles.some((r: any) => r.code === 'admin')

  let enabledPluginNames: Set<string> = new Set()
  try {
    const { listPlugins } = await import('@/api/plugin')
    const backendPlugins = await listPlugins()
    enabledPluginNames = new Set(
      backendPlugins
        .filter((p: any) => p.enabled)
        .map((p: any) => p.name)
    )
  } catch (error) {
    const installedPlugins = pluginManager.getInstalled()
    enabledPluginNames = new Set(installedPlugins.map(p => p.name))
  }

  const PLUGIN_MENU_SORT_KEY = 'mom_plugin_menu_sort'
  const customSort: Map<string, number> = (() => {
    try {
      const stored = localStorage.getItem(PLUGIN_MENU_SORT_KEY)
      if (stored) {
        const sortMap = JSON.parse(stored)
        return new Map(Object.entries(sortMap))
      }
    } catch (error) {}
    return new Map()
  })()

  allPlugins.forEach(plugin => {
    if (!enabledPluginNames.has(plugin.name)) return
    if (plugin.getMenus) {
      const menus = plugin.getMenus()
      menus.forEach(menu => {
        if (!isSuperAdmin && !authorizedPaths.has(menu.path)) return
        const sort = customSort.get(menu.path) ?? menu.sort
        pluginMenus.push({
          ID: menu.path,
          name: menu.name,
          path: menu.path,
          icon: menu.icon,
          sort: sort,
          hidden: menu.hidden,
          parentPath: menu.parentPath
        })
      })
    }
  })
  return pluginMenus
}

// 构建菜单树
const buildMenuTree = (menus: any[]) => {
  const filteredMenus = menus.filter(menu => {
    return menu.visible === undefined || menu.visible === 1
  })

  const uniqueMenus: any[] = []
  const seenSignatures = new Set<string>()

  for (const menu of filteredMenus) {
    let parentKey = 'root'
    if (menu.parentId !== undefined && menu.parentId !== 0) {
      parentKey = `parent_${menu.parentId}`
    } else if (menu.parentPath !== undefined && menu.parentPath !== '' && menu.parentPath !== '/') {
      parentKey = menu.parentPath
    }
    const signature = `${menu.name}_${parentKey}`
    if (seenSignatures.has(signature)) continue
    seenSignatures.add(signature)
    uniqueMenus.push(menu)
  }

  const menuMap = new Map()
  const pathToMenuMap = new Map()

  uniqueMenus.forEach(menu => {
    const menuId = menu.ID || menu.id || menu.path
    if (!menuId) return
    const { children, ...menuWithoutChildren } = menu
    menuMap.set(menuId, menuWithoutChildren)
    if (menu.path && menu.path.startsWith('/')) {
      pathToMenuMap.set(menu.path, menuWithoutChildren)
    }
  })

  const tree: any[] = []

  filteredMenus.forEach(menu => {
    const menuId = menu.ID || menu.id || menu.path
    const menuItem = menuMap.get(menuId)
    if (!menuItem) return

    let parentId = null
    if (menu.parentPath !== undefined) {
      parentId = menu.parentPath || null
    } else if (menu.parentId !== undefined) {
      parentId = menu.parentId === 0 ? null : menu.parentId
    }

    if (parentId && menuMap.has(parentId)) {
      const parent = menuMap.get(parentId)
      if (!parent.children) parent.children = []
      parent.children.push(menuItem)
    } else if (parentId && pathToMenuMap.has(parentId)) {
      const parent = pathToMenuMap.get(parentId)
      if (!parent.children) parent.children = []
      parent.children.push(menuItem)
    } else if (parentId) {
      tree.push(menuItem)
    } else {
      tree.push(menuItem)
    }
  })

  const sortMenus = (menus: any[]) => {
    menus.sort((a, b) => (a.sort || 0) - (b.sort || 0))
    menus.forEach(menu => {
      if (menu.children && menu.children.length > 0) {
        sortMenus(menu.children)
      }
    })
  }
  sortMenus(tree)

  const cleanEmptyChildren = (nodes: any[]) => {
    for (const node of nodes) {
      if (Array.isArray(node.children) && node.children.length === 0) {
        delete node.children
        node.hasChildren = false
      } else if (Array.isArray(node.children) && node.children.length > 0) {
        node.hasChildren = true
        cleanEmptyChildren(node.children)
      } else if (node.children) {
        delete node.children
        node.hasChildren = false
      } else {
        node.hasChildren = false
      }
    }
  }
  cleanEmptyChildren(tree)

  return tree
}

// 加载菜单
const loadMenu = async () => {
  try {
    menuList.value = []
    const systemMenus = await getUserMenu() || []

    const extractPaths = (menus: any[]): Set<string> => {
      const paths = new Set<string>()
      const traverse = (items: any[]) => {
        items.forEach(item => {
          if (item.path) paths.add(item.path)
          if (item.children && item.children.length > 0) traverse(item.children)
        })
      }
      traverse(menus)
      return paths
    }

    const allAuthorizedPaths = extractPaths(systemMenus)
    const pluginMenus = await buildPluginMenus(allAuthorizedPaths)

    const pluginProvidedMenuCodes = new Set([
      'kubernetes_application_diagnosis', 'kubernetes_cluster_inspection',
      'monitor_domain', 'monitor_alert_channels', 'monitor_alert_receivers', 'monitor_alert_logs',
      'task_templates', 'task_execute', 'task_file_distribution',
      'kubernetes_clusters', 'kubernetes_nodes', 'kubernetes_namespaces',
      'kubernetes_workloads', 'kubernetes_network', 'kubernetes_config',
      'kubernetes_storage', 'kubernetes_access', 'kubernetes_audit'
    ])

    const flattenMenus = (menus: any[], result: any[] = []) => {
      menus.forEach(menu => {
        if (menu.code && pluginProvidedMenuCodes.has(menu.code)) return
        const { children, ...menuWithoutChildren } = menu
        result.push(menuWithoutChildren)
        if (children && children.length > 0) flattenMenus(children, result)
      })
      return result
    }

    const flatSystemMenus = flattenMenus(systemMenus)
    const allMenus = [...flatSystemMenus, ...pluginMenus]
    menuList.value = buildMenuTree(allMenus)

    const roles = userStore.userInfo?.roles || []
    const isSuperAdmin = roles.some((r: any) => r.code === 'admin')
    hasNoPermission.value = !isSuperAdmin && menuList.value.length === 0
  } catch (error) {
    ElMessage.error('加载菜单失败')
  }
}

const handleUserCommand = (command: string) => {
  if (command === 'logout') {
    userStore.logout()
    router.push('/login')
  } else if (command === 'profile') {
    router.push('/profile')
  }
}

onMounted(async () => {
  if (!userStore.userInfo) {
    try {
      await userStore.getProfile()
    } catch (error) {}
  }
  await new Promise(resolve => setTimeout(resolve, 100))
  loadMenu()

  const handlePluginChange = () => loadMenu()
  window.removeEventListener('plugins-changed', handlePluginChange)
  window.addEventListener('plugins-changed', handlePluginChange)
  onUnmounted(() => {
    window.removeEventListener('plugins-changed', handlePluginChange)
  })
})
</script>

<style scoped>
.layout-container {
  height: 100vh;
}

/* 侧边栏 */
.sidebar {
  background-color: #0a466a;
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  transition: width 0.3s ease;
  position: relative;
}

/* Logo */
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0a466a;
  border-bottom: 1px solid rgba(255, 255, 255, 0.15);
  cursor: pointer;
  flex-shrink: 0;
  padding: 0 12px;
}

.logo-image {
  max-height: 40px;
  max-width: 200px;
  width: auto;
  height: auto;
  object-fit: contain;
}

.logo-image-mini {
  max-height: 32px;
  max-width: 40px;
  width: auto;
  height: auto;
  object-fit: contain;
}

/* 菜单 */
.el-menu-vertical {
  border-right: none !important;
  background-color: #0a466a !important;
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.el-menu-vertical:not(.el-menu--collapse) {
  width: 200px;
}

/* 自定义滚动条 */
.el-menu-vertical::-webkit-scrollbar {
  width: 4px;
}

.el-menu-vertical::-webkit-scrollbar-track {
  background: transparent;
}

.el-menu-vertical::-webkit-scrollbar-thumb {
  background-color: rgba(255, 255, 255, 0.2);
  border-radius: 0;
}

.el-menu-vertical::-webkit-scrollbar-thumb:hover {
  background-color: rgba(255, 255, 255, 0.3);
}

/* ================================
   Zabbix 风格菜单样式
   ================================ */

/* 顶级菜单项（无子菜单，如仪表盘） */
:deep(.el-menu-vertical > .el-menu-item) {
  height: 40px;
  line-height: 40px;
  font-size: 14px !important;
  font-weight: 500; /* 增加字重 */
  margin: 0;
  padding-left: 16px !important;
  border-radius: 0;
  border-left: 3px solid transparent;
  transition: all 0.15s ease;
  color: #ffffff;
}

/* 通用菜单项样式 */
:deep(.el-menu-item) {
  height: 40px;
  line-height: 40px;
  font-size: 14px;
  font-weight: 500; /* 增加字重 */
  margin: 0;
  padding-left: 16px !important;
  border-radius: 0;
  border-left: 3px solid transparent;
  transition: all 0.15s ease;
  color: #ffffff;
}

:deep(.el-menu-item:hover) {
  background-color: rgba(0, 0, 0, 0.15) !important;
  color: #ffffff !important;
}

:deep(.el-menu-item.is-active) {
  background-color: rgba(0, 0, 0, 0.2) !important;
  border-left: 3px solid #4fc3f7 !important;
  color: #ffffff !important;
  font-weight: 600; /* 选中项更粗 */
}

:deep(.el-menu-item.is-active .el-icon) {
  color: #4fc3f7 !important;
}

:deep(.el-menu-item .el-icon) {
  font-size: 15px;
  margin-right: 10px;
  width: 18px;
  color: rgba(255, 255, 255, 0.7);
}

:deep(.el-menu-item:hover .el-icon) {
  color: rgba(255, 255, 255, 0.9);
}

/* 子菜单标题（父级菜单） */
:deep(.el-sub-menu__title) {
  height: 40px;
  line-height: 40px;
  font-size: 14px;
  font-weight: 500; /* 增加字重 */
  margin: 0;
  padding-left: 16px !important;
  border-radius: 0;
  border-left: 3px solid transparent;
  transition: all 0.15s ease;
  color: #ffffff;
}

:deep(.el-sub-menu__title:hover) {
  background-color: rgba(0, 0, 0, 0.15) !important;
  color: #ffffff !important;
}

:deep(.el-sub-menu.is-opened > .el-sub-menu__title) {
  background-color: rgba(0, 0, 0, 0.1) !important;
  color: #ffffff !important;
}

:deep(.el-sub-menu__title .el-icon) {
  font-size: 15px;
  margin-right: 10px;
  width: 18px;
  color: rgba(255, 255, 255, 0.7);
}

:deep(.el-sub-menu__title:hover .el-icon) {
  color: rgba(255, 255, 255, 0.9);
}

/* 子菜单箭头 */
:deep(.el-sub-menu__icon-arrow) {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
  transition: transform 0.2s ease;
  right: 16px;
}

:deep(.el-sub-menu.is-opened > .el-sub-menu__title .el-sub-menu__icon-arrow) {
  transform: rotate(180deg);
  color: rgba(255, 255, 255, 0.7);
}

/* 子菜单展开区域 - Zabbix 风格 */
:deep(.el-menu--inline) {
  background-color: rgba(0, 0, 0, 0.25) !important;
  border-left: 20px solid rgb(0 30 43 / 90%);
  margin-left: 0;
}

/* 子菜单内容项 */
:deep(.el-menu--inline .el-menu-item) {
  height: 38px;
  line-height: 38px;
  padding-left: 28px !important;
  margin: 0;
  font-size: 14px;
  color: #ffffff;
  border-left: 3px solid transparent;
  background-color: transparent;
  transition: all 0.15s ease;
}

:deep(.el-menu--inline .el-menu-item .el-icon) {
  font-size: 14px;
  margin-right: 8px;
  color: rgba(255, 255, 255, 0.65);
}

:deep(.el-menu--inline .el-menu-item:hover) {
  background-color: rgba(79, 195, 247, 0.15) !important;
  border-left: 3px solid rgba(255, 255, 255, 0.5) !important;
  color: #ffffff !important;
}

:deep(.el-menu--inline .el-menu-item:hover .el-icon) {
  color: rgba(255, 255, 255, 0.9);
}

:deep(.el-menu--inline .el-menu-item.is-active) {
  background-color: rgba(79, 195, 247, 0.25) !important;
  border-left: 3px solid #4fc3f7 !important;
  color: #4fc3f7 !important;
}

:deep(.el-menu--inline .el-menu-item.is-active .el-icon) {
  color: #4fc3f7;
}

/* 收缩状态样式 */
:deep(.el-menu--collapse .el-menu-item) {
  margin: 0;
  padding: 0 !important;
  justify-content: center;
  border-left: 4px solid transparent;
  transition: all 0.2s ease;
}

:deep(.el-menu--collapse .el-menu-item .el-icon) {
  margin-right: 0;
  font-size: 18px;
}

:deep(.el-menu--collapse .el-menu-item:hover) {
  background-color: rgba(255, 255, 255, 0.1) !important;
}

:deep(.el-menu--collapse .el-menu-item:hover .el-icon) {
  color: #ffffff !important;
}

:deep(.el-menu--collapse .el-menu-item.is-active) {
  background-color: rgba(79, 195, 247, 0.25) !important;
  border-left: 4px solid #4fc3f7 !important;
}

:deep(.el-menu--collapse .el-menu-item.is-active .el-icon) {
  color: #4fc3f7 !important;
  font-size: 20px;
}

:deep(.el-menu--collapse .el-sub-menu__title) {
  margin: 0;
  padding: 0 !important;
  justify-content: center;
  border-left: 4px solid transparent;
  transition: all 0.2s ease;
}

:deep(.el-menu--collapse .el-sub-menu__title .el-icon) {
  margin-right: 0;
  font-size: 18px;
}

:deep(.el-menu--collapse .el-sub-menu__title:hover) {
  background-color: rgba(255, 255, 255, 0.1) !important;
}

:deep(.el-menu--collapse .el-sub-menu.is-active > .el-sub-menu__title) {
  background-color: rgba(79, 195, 247, 0.15) !important;
  border-left: 4px solid #4fc3f7 !important;
}

:deep(.el-menu--collapse .el-sub-menu.is-active > .el-sub-menu__title .el-icon) {
  color: #4fc3f7 !important;
}

:deep(.el-menu--collapse .el-sub-menu__icon-arrow) {
  display: none;
}

/* 弹出菜单样式 - Zabbix 风格 */
:deep(.el-menu--popup) {
  min-width: 160px;
  background-color: #0a466a !important;
  border: none;
  border-radius: 0;
  padding: 0;
  box-shadow: 2px 2px 8px rgba(0, 0, 0, 0.3);
}

:deep(.el-menu--popup .el-menu-item) {
  height: 36px;
  line-height: 36px;
  margin: 0;
  padding: 0 16px !important;
  border-left: 4px solid transparent;
  color: rgba(255, 255, 255, 0.85) !important;
  border-radius: 0;
  transition: all 0.15s ease;
}

:deep(.el-menu--popup .el-menu-item .el-icon) {
  color: rgba(255, 255, 255, 0.7);
  margin-right: 8px;
}

:deep(.el-menu--popup .el-menu-item:hover) {
  background-color: rgba(255, 255, 255, 0.1) !important;
  color: #ffffff !important;
}

:deep(.el-menu--popup .el-menu-item:hover .el-icon) {
  color: #ffffff !important;
}

:deep(.el-menu--popup .el-menu-item.is-active) {
  background-color: rgba(79, 195, 247, 0.3) !important;
  border-left: 4px solid #4fc3f7 !important;
  color: #ffffff !important;
  font-weight: 600;
}

:deep(.el-menu--popup .el-menu-item.is-active .el-icon) {
  color: #4fc3f7 !important;
}

/* 收缩按钮 */
.collapse-btn {
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: rgba(255, 255, 255, 0.6);
  border-top: 1px solid rgba(0, 0, 0, 0.2);
  background-color: rgba(0, 0, 0, 0.1);
  transition: all 0.15s ease;
  flex-shrink: 0;
}

.collapse-btn:hover {
  color: #4fc3f7;
  background-color: rgba(0, 0, 0, 0.2);
}

/* Header */
.main-header {
  height: 56px !important;
  background-color: #fff;
  border-bottom: 1px solid #e6e6e6;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 20px;
}

.header-logo {
  display: flex;
  align-items: center;
}

.header-image {
  max-height: 32px;
  width: auto;
  object-fit: contain;
}

.breadcrumb {
  font-size: 13px;
}

.header-right {
  display: flex;
  align-items: center;
}

/* 用户信息 */
.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 12px;
  border-radius: 0;
  cursor: pointer;
  transition: all 0.2s ease;
}

.user-info:hover {
  background-color: #f5f7fa;
}

.user-avatar {
  background-color: #0a466a;
  flex-shrink: 0;
}

.user-avatar :deep(.el-icon) {
  font-size: 16px;
  color: #fff;
}

.user-details {
  display: flex;
  flex-direction: column;
  line-height: 1.3;
}

.user-name {
  font-size: 13px;
  font-weight: 500;
  color: #303133;
}

.user-role {
  font-size: 11px;
  color: #909399;
}

.dropdown-arrow {
  font-size: 12px;
  color: #909399;
  transition: transform 0.3s ease;
}

.user-info:hover .dropdown-arrow {
  color: #606266;
}

/* 下拉菜单 */
:deep(.el-dropdown-menu) {
  padding: 4px 0;
}

:deep(.el-dropdown-menu__item) {
  font-size: 13px;
  padding: 8px 16px;
  line-height: 1.5;
}

:deep(.el-dropdown-menu__item .el-icon) {
  margin-right: 8px;
  font-size: 14px;
}

/* 主内容区 */
.el-main {
  background-color: #f0f2f5;
  padding: 16px;
  overflow: auto;
}

/* 禁用菜单样式 */
:deep(.menu-disabled) {
  opacity: 0.4 !important;
  cursor: not-allowed !important;
  pointer-events: none !important;
}
</style>

<!-- 全局样式 - 用于弹出菜单（teleport到body的元素） -->
<style>
/* 收缩状态下的弹出子菜单样式 - Zabbix 风格 */
.el-menu--vertical .el-menu--popup-container .el-menu--popup {
  min-width: 160px !important;
  background-color: #0a466a !important;
  border: none !important;
  border-radius: 0 !important;
  padding: 0 !important;
  box-shadow: 2px 2px 8px rgba(0, 0, 0, 0.3) !important;
}

.el-menu--vertical .el-menu--popup-container .el-menu--popup .el-menu-item {
  height: 34px !important;
  line-height: 34px !important;
  margin: 0 !important;
  padding: 0 16px !important;
  border-radius: 0 !important;
  border-left: 3px solid transparent !important;
  color: rgba(255, 255, 255, 0.85) !important;
  background-color: transparent !important;
}

.el-menu--vertical .el-menu--popup-container .el-menu--popup .el-menu-item:hover {
  background-color: rgba(0, 0, 0, 0.15) !important;
  color: #ffffff !important;
}

.el-menu--vertical .el-menu--popup-container .el-menu--popup .el-menu-item.is-active {
  background-color: rgba(0, 0, 0, 0.2) !important;
  border-left: 3px solid #4fc3f7 !important;
  color: #4fc3f7 !important;
}

.el-menu--vertical .el-menu--popup-container .el-menu--popup .el-menu-item .el-icon {
  display: none !important;
}
</style>
