<template>
  <div class="terminal-page">
    <!-- 左侧分组和主机列表 -->
    <div class="terminal-sidebar">
      <div class="sidebar-header">
        <!-- 主机/网络设备切换 -->
        <div class="sidebar-segmented">
          <div
            class="seg-item"
            :class="{ active: sidebarMode === 'host' }"
            @click="switchSidebarMode('host')"
          >主机</div>
          <div
            class="seg-item"
            :class="{ active: sidebarMode === 'network-device' }"
            @click="switchSidebarMode('network-device')"
          >网络设备</div>
        </div>
        <div class="search-box">
          <el-input
            v-model="searchKeyword"
            :placeholder="sidebarMode === 'host' ? '搜索主机...' : '搜索网络设备...'"
            clearable
            size="small"
            class="search-input"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </div>
      </div>

      <div class="sidebar-content">
        <el-tree
          ref="treeRef"
          :data="filteredTreeData"
          :props="treeProps"
          :default-expand-all="false"
          :expand-on-click-node="false"
          :highlight-current="true"
          node-key="id"
          class="terminal-tree"
        >
          <template #default="{ node, data }">
            <div class="tree-node" @dblclick="handleNodeDblClick(data, node, $event)">
              <span class="node-icon">
                <el-icon v-if="data.type === 'group'"><Folder /></el-icon>
                <svg v-else-if="data.type === 'network-device'" class="host-svg-icon" viewBox="0 0 24 24" fill="currentColor" style="color: #e6a23c;">
                  <path d="M1 9l2 2c4.97-4.97 13.03-4.97 18 0l2-2C16.93 2.93 7.08 2.93 1 9zm8 8l3 3 3-3c-1.65-1.66-4.34-1.66-6 0zm-4-4l2 2c2.76-2.76 7.24-2.76 10 0l2-2C15.14 9.14 8.87 9.14 5 13z"/>
                </svg>
                <svg v-else class="host-svg-icon" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M20 18c1.1 0 1.99-.9 1.99-2L22 6c0-1.11-.9-2-2-2H4c-1.11 0-2 .89-2 2v10c0 1.1.89 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"/>
                </svg>
              </span>
              <span class="node-label">{{ node.label }}</span>
              <span v-if="data.type === 'group'" class="node-count">({{ data.hostCount || data.deviceCount || 0 }})</span>
              <span v-if="data.type === 'host' || data.type === 'network-device'" class="node-status" :class="getStatusClass(data.status)"></span>
              <el-tag v-if="data.type === 'network-device' && data.protocol" size="small" :type="data.protocol === 'ssh' ? 'success' : 'warning'" effect="plain" class="node-protocol-tag">
                {{ data.protocol?.toUpperCase() }}
              </el-tag>
            </div>
          </template>
        </el-tree>
      </div>
    </div>

    <!-- 右侧终端区域 -->
    <div class="terminal-main">
      <div class="header-right-absolute">
          <el-button class="back-btn" link @click="handleGoBack" title="返回">
            <el-icon :size="18"><ArrowLeft /></el-icon>
            <span style="margin-left: 4px;">返回</span>
          </el-button>
          <el-dropdown trigger="click" @command="handleUserCommand">
            <div class="terminal-user-info">
              <el-avatar :size="32" :src="avatarUrl" class="user-avatar">
                <el-icon><UserFilled /></el-icon>
              </el-avatar>
              <div class="user-details">
                <span class="user-name">{{ userStore.userInfo?.realName || userStore.userInfo?.username || 'Guest' }}</span>
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

      <div class="tabs-container">
        <el-tabs
          v-model="activeTab"
          type="card"
          class="terminal-tabs"
          @tab-remove="handleTabRemove"
        >
          <el-tab-pane
            v-for="tab in terminalTabs"
            :key="tab.id"
            :name="tab.id"
            :closable="terminalTabs.length > 1"
          >
            <template #label>
              <span class="tab-label">
                <svg class="tab-icon" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M20 18c1.1 0 1.99-.9 1.99-2L22 6c0-1.11-.9-2-2-2H4c-1.11 0-2 .89-2 2v10c0 1.1.89 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"/>
                </svg>
                <span class="tab-name">{{ tab.label }}</span>
                <div v-if="tab.connected" class="status-dot online"></div>
                <div v-else-if="tab.connecting" class="status-dot connecting"></div>
                <div v-else class="status-dot offline"></div>
              </span>
            </template>
            <template #default>
              <div class="tab-content">
                <div v-if="tab.host" class="terminal-connected">
                  <div class="terminal-body">
                    <div v-if="tab.host?.osType === 'windows'" class="terminal-toolbar">
                      <el-button-group>
                        <el-button size="small" @click="sendKeyCombination(tab.id, [0xFFE3, 0xFFE9, 0xFFFF])" title="Ctrl+Alt+Del">Ctrl+Alt+Del</el-button>
                        <el-button size="small" @click="sendKeyCombination(tab.id, [0xFFEB])" title="Windows Key">Win</el-button>
                        <el-button size="small" @click="sendKeyCombination(tab.id, [0xFFE9, 0xFF09])" title="Alt+Tab">Alt+Tab</el-button>
                        <el-button size="small" @click="sendKeyCombination(tab.id, [0xFF1B])" title="Esc">Esc</el-button>
                        <el-button size="small" @click="openFileManager(tab.id)" title="文件管理">
                          <el-icon><Folder /></el-icon> 文件管理
                        </el-button>
                        <el-button size="small" @click="toggleVirtualKeyboard(tab.id)" title="虚拟键盘">
                          <el-icon><Keyboard /></el-icon> 键盘
                        </el-button>
                      </el-button-group>
                    </div>
                    <div class="terminal-viewport">
                      <div v-show="tab.host.osType !== 'windows'" :ref="el => terminalRefs[tab.id] = el as HTMLElement" class="xterm-container"></div>
                      <div v-show="tab.host.osType === 'windows'" :ref="el => guacamoleRefs[tab.id] = el as HTMLElement" class="guacamole-container"></div>
                    </div>
                  </div>
                </div>
                <div v-else class="terminal-empty">
                  <div class="empty-icon">
                    <svg viewBox="0 0 24 24" fill="currentColor">
                      <path d="M20 18c1.1 0 1.99-.9 1.99-2L22 6c0-1.11-.9-2-2-2H4c-1.11 0-2 .89-2 2v10c0 1.1.89 2 2 2H0v2h24v-2h-4zM4 6h16v10H4V6z"/>
                    </svg>
                  </div>
                  <div class="empty-text">双击主机打开终端</div>
                  <div class="empty-hint">在左侧资产分组中选择主机</div>
                </div>
              </div>
            </template>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>
      </div>


    <!-- 文件管理抽屉 -->
    <el-drawer
      v-model="fileDrawerVisible"
      title="文件管理"
      direction="rtl"
      size="350px"
      :show-close="true"
      :destroy-on-close="true"
      class="file-drawer-dark"
    >
      <div class="file-manager-content">
        <div class="upload-section">
          <h3>上传文件至 RDP</h3>
          <el-upload
            class="upload-demo"
            drag
            action=""
            :auto-upload="false"
            :on-change="handleFileSelect"
            :show-file-list="false"
            :disabled="isUploading"
          >
            <el-icon class="el-icon--upload"><upload-filled /></el-icon>
            <div class="el-upload__text">
              拖拽文件到此处或 <em>点击选择</em>
            </div>
            <div class="el-upload__tip">选择后自动上传</div>
          </el-upload>

          <div v-if="selectedFile" class="file-preview">
            <div class="file-info">
              <el-icon><Document /></el-icon>
              <span class="file-name">{{ selectedFile.name }}</span>
              <span class="file-size">{{ formatFileSize(selectedFile.size) }}</span>
            </div>

            <div class="progress-area">
              <el-progress
                :percentage="uploadProgress"
                :status="uploadStatus"
                :stroke-width="6"
                :show-text="true"
              />
              <div class="upload-speed" v-if="uploadSpeed">{{ uploadSpeed }}</div>
            </div>
          </div>
        </div>

        <div class="download-section">
          <h3>下载提示</h3>
          <p class="hint-text">
            在 Windows RDP 中，将文件复制或拖拽到 <b>Download</b> 文件夹（位于 Guacamole Filesystem 驱动器 (G:) 下），浏览器将自动开始下载。
          </p>
        </div>
      </div>
    </el-drawer>

    <!-- 悬浮虚拟键盘 -->
    <Teleport to="body">
      <div
        v-show="virtualKeyboardVisible"
        class="floating-keyboard"
        :style="{ left: kbPos.x + 'px', top: kbPos.y + 'px' }"
      >
        <div class="kb-titlebar" @mousedown.prevent="startKbDrag">
          <span class="kb-title">⌨ 虚拟键盘</span>
          <button class="kb-close" @click="virtualKeyboardVisible = false">✕</button>
        </div>
        <div class="virtual-keyboard">
          <!-- 功能键行 -->
          <div class="kb-row">
            <button v-for="key in keyboardLayout.functionRow" :key="key.label" class="kb-key kb-fn" @mousedown.prevent="sendVKey(key.code)" :title="key.label">{{ key.label }}</button>
          </div>
          <!-- 数字行 -->
          <div class="kb-row">
            <button v-for="key in keyboardLayout.numberRow" :key="key.label" class="kb-key" :class="{ 'kb-wide': key.wide }" @mousedown.prevent="sendVKey(key.code)" :title="key.label">{{ key.label }}</button>
          </div>
          <!-- 第一字母行 -->
          <div class="kb-row">
            <button v-for="key in keyboardLayout.row1" :key="key.label" class="kb-key" :class="{ 'kb-wide': key.wide }" @mousedown.prevent="sendVKey(key.code)" :title="key.label">{{ key.label }}</button>
          </div>
          <!-- 第二字母行 -->
          <div class="kb-row">
            <button v-for="key in keyboardLayout.row2" :key="key.label" class="kb-key" :class="{ 'kb-wide': key.wide }" @mousedown.prevent="sendVKey(key.code)" :title="key.label">{{ key.label }}</button>
          </div>
          <!-- 第三字母行 -->
          <div class="kb-row">
            <button v-for="key in keyboardLayout.row3" :key="key.label" class="kb-key" :class="{ 'kb-wide': key.wide }" @mousedown.prevent="sendVKey(key.code)" :title="key.label">{{ key.label }}</button>
          </div>
          <!-- 空格行 -->
          <div class="kb-row">
            <button v-for="key in keyboardLayout.spaceRow" :key="key.label" class="kb-key" :class="{ 'kb-space': key.space, 'kb-wide': key.wide }" @mousedown.prevent="sendVKey(key.code)" :title="key.label">{{ key.label }}</button>
          </div>
        </div>
      </div>
    </Teleport>

</template>>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick, onBeforeUnmount } from 'vue'
import { Collection, Search, Monitor, Folder, Edit as Keyboard } from '@element-plus/icons-vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import Guacamole from 'guacamole-common-js'
import { getHostList, getAllNetworkDevices } from '@/api/host'
import { getGroupTree } from '@/api/assetGroup'
import { useUserStore } from '@/stores/user'
import { useRouter, useRoute } from 'vue-router'
import { UserFilled, ArrowDown, ArrowLeft, User, SwitchButton, UploadFilled, Document } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

// 头像URL
const avatarUrl = computed(() => {
  const avatar = userStore.userInfo?.avatar || ''
  if (!avatar) return ''
  if (avatar.startsWith('data:')) return avatar
  const separator = avatar.includes('?') ? '&' : '?'
  return `${avatar}${separator}t=${userStore.avatarTimestamp}`
})

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

const handleUserCommand = (command: string) => {
  if (command === 'logout') {
    userStore.logout()
    router.push('/login')
  } else if (command === 'profile') {
    router.push('/profile')
  }
}

const handleGoBack = () => {
  // 如果是从新窗口打开的（有 history），先尝试返回
  if (window.history.length > 1) {
    router.back()
  } else {
    // 新窗口打开的情况下关闭窗口，或者跳转到默认页
    window.close()
    // 如果 window.close() 被阻止，回到资产首页
    router.push('/asset/network-devices')
  }
}

const treeRef = ref()
const searchKeyword = ref('')
const activeTab = ref('1')
const sidebarMode = ref<'host' | 'network-device'>('host')
const allNetworkDevices = ref<any[]>([])
const terminalRefs = ref<Record<string, HTMLElement>>({})
const terminals = ref<Record<string, Terminal>>({})
const fitAddons = ref<Record<string, FitAddon>>({})
const wss = ref<Record<string, WebSocket>>({})
const resizeCleanups = ref<Record<string, () => void>>({})

// Guacamole related refs
const guacamoleRefs = ref<Record<string, HTMLElement>>({})
const guacamoleClients = ref<Record<string, any>>({})
const guacamoleTunnels = ref<Record<string, any>>({})
const guacamoleResizeObservers = ref<Record<string, ResizeObserver>>({})

// 终端标签页
interface TerminalTab {
  id: string
  label: string
  host: any
  connected: boolean
  connecting: boolean
}

const terminalTabs = ref<TerminalTab[]>([
  {
    id: '1',
    label: '新建终端',
    host: null,
    connected: false,
    connecting: false
  }
])

// 树形配置
const treeProps = {
  children: 'children',
  label: 'label',
  value: 'id'
}

// 加载分组树
const groupTree = ref<any[]>([])
const networkGroupTree = ref<any[]>([])
const allHosts = ref<any[]>([])

const loadGroupTree = async () => {
  try {
    const data = await getGroupTree('host')
    groupTree.value = data || []
  } catch (error) {
  }
}

const loadNetworkGroupTree = async () => {
  try {
    const data = await getGroupTree('network')
    networkGroupTree.value = data || []
  } catch (error) {
  }
}

const loadAllHosts = async () => {
  try {
    const res = await getHostList({ page: 1, pageSize: 10000 })
    allHosts.value = res.list || []
  } catch (error) {
    allHosts.value = []
  }
}

const loadAllNetworkDevices = async () => {
  try {
    const res: any = await getAllNetworkDevices()
    allNetworkDevices.value = Array.isArray(res) ? res : []
  } catch (error) {
    allNetworkDevices.value = []
  }
}

const switchSidebarMode = (mode: 'host' | 'network-device') => {
  sidebarMode.value = mode
  searchKeyword.value = ''
  if (mode === 'network-device') {
    if (allNetworkDevices.value.length === 0) loadAllNetworkDevices()
    if (networkGroupTree.value.length === 0) loadNetworkGroupTree()
  }
}

// 构建带主机的树形数据
const treeData = computed(() => {
  if (sidebarMode.value === 'network-device') {
    return buildNetworkDeviceTree()
  }
  return buildHostTree()
})

function buildHostTree() {
  if (!groupTree.value || groupTree.value.length === 0) {
    return []
  }

  const buildTree = (groups: any[]): any[] => {
    return groups.map((group: any) => {
      const node: any = {
        ...group,
        type: 'group',
        label: group.name,
        children: group.children ? buildTree(group.children) : []
      }

      // 添加该分组下的主机
      const groupHosts = allHosts.value.filter((h: any) => h.groupId === group.id)
      if (groupHosts.length > 0) {
        const hostNodes = groupHosts.map((host: any) => ({
          ...host,
          type: 'host',
          label: host.name
        }))
        node.children = [...node.children, ...hostNodes]
        node.hostCount = hostNodes.length
      } else {
        node.hostCount = 0
      }

      return node
    })
  }

  return buildTree(groupTree.value)
}

function buildNetworkDeviceTree() {
  if (!networkGroupTree.value || networkGroupTree.value.length === 0) {
    // 没有分组时，直接列出所有设备
    return allNetworkDevices.value.map((d: any) => ({
      ...d,
      type: 'network-device',
      label: `${d.name} (${d.ip})`
    }))
  }

  const buildTree = (groups: any[]): any[] => {
    return groups.map((group: any) => {
      const node: any = {
        ...group,
        type: 'group',
        label: group.name,
        children: group.children ? buildTree(group.children) : []
      }

      const groupDevices = allNetworkDevices.value.filter((d: any) => d.groupId === group.id)
      if (groupDevices.length > 0) {
        const deviceNodes = groupDevices.map((d: any) => ({
          ...d,
          type: 'network-device',
          label: `${d.name} (${d.ip})`
        }))
        node.children = [...node.children, ...deviceNodes]
        node.deviceCount = deviceNodes.length
      } else {
        node.deviceCount = 0
      }

      return node
    })
  }

  const tree = buildTree(networkGroupTree.value)

  // 添加未分组的设备
  const ungroupedDevices = allNetworkDevices.value.filter((d: any) => !d.groupId || d.groupId === 0)
  if (ungroupedDevices.length > 0) {
    tree.push({
      id: 'ungrouped',
      type: 'group',
      label: '未分组',
      deviceCount: ungroupedDevices.length,
      children: ungroupedDevices.map((d: any) => ({
        ...d,
        type: 'network-device',
        label: `${d.name} (${d.ip})`
      }))
    })
  }

  return tree
}

// 过滤后的树形数据
const filteredTreeData = computed(() => {
  if (!searchKeyword.value) {
    return treeData.value
  }

  const keyword = searchKeyword.value.toLowerCase()
  const filterTree = (nodes: any[]): any[] => {
    const result: any[] = []
    for (const node of nodes) {
      const matchName = node.label?.toLowerCase().includes(keyword)
      const matchIp = node.ip?.toLowerCase().includes(keyword)

      let filteredChildren: any[] = []
      if (node.children && node.children.length > 0) {
        filteredChildren = filterTree(node.children)
      }

      if ((matchName || matchIp) || filteredChildren.length > 0) {
        result.push({
          ...node,
          children: filteredChildren.length > 0 ? filteredChildren : node.children
        })
      }
    }
    return result
  }

  return filterTree(treeData.value)
})

// 获取状态样式类
const getStatusClass = (status: number) => {
  if (status === 1) return 'online'
  if (status === 0) return 'offline'
  return 'unknown'
}

// 双击节点
const handleNodeDblClick = (data: any, node: any, event: Event) => {
  event.preventDefault()
  event.stopPropagation()

  if (data.type === 'network-device') {
    openNetworkDeviceTerminal(data)
  } else if (data.type === 'host' || (data.ip && data.port)) {
    openTerminal(data)
  }
}

// 打开新的终端标签页
const openTerminal = async (host: any) => {

  const tabId = Date.now().toString()

  // 计算相同主机名的标签数量，用于生成唯一的标签名称
  const sameHostTabs = terminalTabs.value.filter(t => t.host && t.host.name === host.name)
  let label = host.name
  if (sameHostTabs.length > 0) {
    label = `${host.name} (${sameHostTabs.length + 1})`
  }

  const newTab: TerminalTab = {
    id: tabId,
    label: label,
    host: host,
    connected: false,
    connecting: true
  }


  // 直接添加新标签页，每次双击都打开新标签
  terminalTabs.value.push(newTab)

  // 切换到新标签页
  activeTab.value = tabId


  await nextTick()
  initTerminal(tabId, host)
}


// 初始化 Guacamole (RDP connection for Windows hosts)
const initGuacamole = async (tabId: string, host: any) => {
  await nextTick()

  // Wait for the guacamole container ref to become available (retry up to 20 times)
  let el = guacamoleRefs.value[tabId]
  let retries = 0
  while (!el && retries < 20) {
    await new Promise(resolve => setTimeout(resolve, 100))
    await nextTick()
    el = guacamoleRefs.value[tabId]
    retries++
  }
  if (!el) {
    console.error('Guacamole container not found for tab:', tabId)
    const tab = terminalTabs.value.find(t => t.id === tabId)
    if (tab) {
      tab.connecting = false
      tab.connected = false
    }
    return
  }

  // Cleanup any existing client for this tab
  if (guacamoleClients.value[tabId]) {
    guacamoleClients.value[tabId].disconnect()
    delete guacamoleClients.value[tabId]
  }
  el.innerHTML = ''

  const token = localStorage.getItem('token') || ''

  // Wait for element to have non-zero dimensions
  let sizeRetries = 0
  while ((el.clientWidth === 0 || el.clientHeight === 0) && sizeRetries < 20) {
    await new Promise(resolve => setTimeout(resolve, 100))
    sizeRetries++
  }
  const width = el.clientWidth || 1024
  const height = el.clientHeight || 768

  const isDev = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const backendHost = window.location.hostname
  const backendPort = isDev ? ':9876' : (window.location.port ? ':' + window.location.port : '')
  
  const wsUrl = `${protocol}//${backendHost}${backendPort}/api/v1/asset/terminal/${host.id}`
  const connectParams = `token=${token}&width=${width}&height=${height}`

  console.log('Guacamole connecting to:', wsUrl, 'params:', connectParams)
  
  const tunnel = new Guacamole.WebSocketTunnel(wsUrl)
  const client = new Guacamole.Client(tunnel)
  guacamoleClients.value[tabId] = client
  guacamoleTunnels.value[tabId] = tunnel

  // Display setup
  const guacDisplay = client.getDisplay()
  const display = guacDisplay.getElement()
  el.appendChild(display)

  // Auto-scale display to fit container when remote display resizes
  const fitDisplay = () => {
    const displayWidth = guacDisplay.getWidth()
    const displayHeight = guacDisplay.getHeight()
    if (displayWidth && displayHeight && el.clientWidth && el.clientHeight) {
      const scale = Math.min(
        el.clientWidth / displayWidth,
        el.clientHeight / displayHeight
      )
      guacDisplay.scale(scale)
    }
  }
  guacDisplay.onresize = fitDisplay

  // Also fit when container becomes visible / resizes
  const resizeObs = new ResizeObserver(() => fitDisplay())
  resizeObs.observe(el)
  guacamoleResizeObservers.value[tabId] = resizeObs

  // Tunnel state change handler
  tunnel.onstatechange = (state: number) => {
    const tab = terminalTabs.value.find(t => t.id === tabId)
    if (!tab) return
    // 0=CONNECTING, 1=OPEN, 2=CLOSED, 3=UNSTABLE
    if (state === 0) {
      tab.connecting = true
      tab.connected = false
    } else if (state === 1) {
      console.log('Guacamole tunnel open for tab:', tabId)
    } else if (state === 2) {
      console.log('Guacamole tunnel closed for tab:', tabId)
      tab.connecting = false
      tab.connected = false
    }
  }

  // Tunnel error handler
  tunnel.onerror = (status: any) => {
    console.error('Guacamole tunnel error:', status)
    const tab = terminalTabs.value.find(t => t.id === tabId)
    if (tab) {
      tab.connecting = false
      tab.connected = false
    }
    ElMessage.error('RDP 连接失败，请检查主机配置或联系管理员')
  }

  // Client state change handler
  client.onstatechange = (state: number) => {
    const tab = terminalTabs.value.find(t => t.id === tabId)
    if (!tab) return
    // 0=IDLE, 1=CONNECTING, 2=WAITING, 3=CONNECTED, 4=DISCONNECTING, 5=DISCONNECTED
    if (state === 3) {
      console.log('Guacamole RDP connected for tab:', tabId)
      tab.connecting = false
      tab.connected = true
    } else if (state === 5) {
      console.log('Guacamole RDP disconnected for tab:', tabId)
      tab.connecting = false
      tab.connected = false
    }
  }

  // Client error handler
  client.onerror = (error: any) => {
    console.error('Guacamole client error:', error)
    const tab = terminalTabs.value.find(t => t.id === tabId)
    if (tab) {
      tab.connecting = false
      tab.connected = false
    }
    // Display Guacamole error message if available
    if (error && error.message) {
      ElMessage.error(error.message)
    }
  }

  // Mouse input
  const mouse = new Guacamole.Mouse(display)
  mouse.onmousedown = mouse.onmouseup = mouse.onmousemove = (mouseState: any) => {
    client.sendMouseState(mouseState)
  }

  // Keyboard input (scoped to display element to avoid conflicts with other tabs)
  const keyboard = new Guacamole.Keyboard(document)
  keyboard.onkeydown = (keysym: any) => {
    client.sendKeyEvent(1, keysym)
  }
  keyboard.onkeyup = (keysym: any) => {
    client.sendKeyEvent(0, keysym)
  }

  // Connect - 查询参数通过 connect(data) 传递，guacamole-common-js 会拼接为 URL?data
  client.connect(connectParams)

  // 处理文件下载 (remote -> client)
  client.onfile = (stream: any, mimetype: string, filename: string) => {
    console.log('Guacamole file download started:', filename, mimetype)
    const reader = new Guacamole.BlobReader(stream, mimetype)

    // CRITICAL: Send initial ACK to tell guacd we are ready to receive data.
    // Without this, guacd waits forever and never sends any blob data.
    stream.sendAck('OK', Guacamole.Status.Code.SUCCESS)

    reader.onend = () => {
      const blob = reader.getBlob()
      console.log('Guacamole file download complete:', filename, 'size:', blob.size)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      // Delay revokeObjectURL slightly to ensure the download starts
      setTimeout(() => URL.revokeObjectURL(url), 1000)
      ElMessage.success(`文件 ${filename} 下载完成`)
    }

    reader.onerror = (status: any) => {
      console.error('Guacamole file download error:', filename, status)
      ElMessage.error(`文件 ${filename} 下载失败`)
    }
  }

  client.onfilesystem = (object: any, name: string) => {
    console.log('Guacamole filesystem available:', name)
    // 可以在这里处理文件系统对象，比如浏览文件
    // 目前 guacd 默认的 native drive 行为已经支持了基础的上传(通过流)和下载(通过 onfile)
  }
}

// 文件管理相关状态
const fileDrawerVisible = ref(false)
const selectedFile = ref<File | null>(null)
const isUploading = ref(false)
const uploadProgress = ref(0)
const uploadStatus = ref('') // 'success' | 'exception' | 'warning'
const uploadSpeed = ref('')
const currentFileManagerTabId = ref('')

const openFileManager = (tabId: string) => {
  currentFileManagerTabId.value = tabId
  fileDrawerVisible.value = true
  // 重置状态
  selectedFile.value = null
  isUploading.value = false
  uploadProgress.value = 0
  uploadStatus.value = ''
}

const handleFileSelect = (file: any) => {
  if (isUploading.value) return
  selectedFile.value = file.raw
  uploadProgress.value = 0
  uploadStatus.value = ''
  uploadSpeed.value = ''
  // Auto-start upload immediately after file selection
  nextTick(() => startUpload())
}

const formatFileSize = (bytes: number) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const startUpload = () => {
  if (!selectedFile.value || !currentFileManagerTabId.value) return
  
  const client = guacamoleClients.value[currentFileManagerTabId.value]
  if (!client) {
    ElMessage.error('RDP 连接未就绪')
    return
  }

  isUploading.value = true
  uploadProgress.value = 0
  uploadStatus.value = ''
  
  const file = selectedFile.value
  const stream = client.createFileStream(file.type, file.name)
  const writer = new Guacamole.ArrayBufferWriter(stream)
  
  const chunkSize = 65536 // 64KB chunks
  let offset = 0
  const reader = new FileReader()
  
  const startTime = Date.now()
  
  reader.onload = (e) => {
    const buffer = e.target?.result as ArrayBuffer
    const bytes = new Uint8Array(buffer)
    
    const uploadNextChunk = () => {
      if (offset >= bytes.length) {
        writer.sendEnd()
        isUploading.value = false
        uploadProgress.value = 100
        uploadStatus.value = 'success'
        ElMessage.success('上传完成')
        return
      }
      
      const end = Math.min(offset + chunkSize, bytes.length)
      const chunk = bytes.subarray(offset, end)
      writer.sendData(chunk)
      
      offset = end
      const progress = Math.floor((offset / bytes.length) * 100)
      uploadProgress.value = progress
      
      // Calculate speed
      const elapsed = (Date.now() - startTime) / 1000
      if (elapsed > 0.5) {
        const speed = offset / elapsed
        uploadSpeed.value = formatFileSize(speed) + '/s'
      }

      // Allow UI update
      setTimeout(uploadNextChunk, 5)
    }

    writer.onack = (status: any) => {
      if (status.isError()) {
        isUploading.value = false
        uploadStatus.value = 'exception'
        ElMessage.error('上传出错: ' + status.message)
        writer.sendEnd() // Close stream on error
      }
    }
    
    uploadNextChunk()
  }
  
  reader.onerror = () => {
    isUploading.value = false
    uploadStatus.value = 'exception'
    ElMessage.error('读取文件失败')
    writer.sendEnd()
  }
  
  reader.readAsArrayBuffer(file)
}


// 预检查终端连接权限（在建立 WebSocket 之前，通过 HTTP 请求检测 401/403）
const checkTerminalPermission = async (host: any): Promise<{ ok: boolean; message: string }> => {
  const token = localStorage.getItem('token') || ''
  const isDev = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
  const backendHost = window.location.hostname
  const backendPort = isDev ? ':9876' : (window.location.port ? ':' + window.location.port : '')
  const checkUrl = `${window.location.protocol}//${backendHost}${backendPort}/api/v1/asset/terminal/${host.id}?token=${token}`

  try {
    const resp = await fetch(checkUrl)
    if (resp.status === 401) {
      return { ok: false, message: '登录已过期，请重新登录' }
    }
    if (resp.status === 403) {
      return { ok: false, message: '没有访问权限，请联系管理员配置终端连接权限' }
    }
    // Any other status (including 400 from failed WS upgrade) means permission is OK
    return { ok: true, message: '' }
  } catch {
    // Network error - let the WebSocket try anyway
    return { ok: true, message: '' }
  }
}

// 初始化终端
const initTerminal = async (tabId: string, host: any) => {
  await nextTick()

  const el = terminalRefs.value[tabId]
  if (!el && host.osType !== 'windows') {
    return
  }

  // Pre-check permission before establishing WebSocket
  const permCheck = await checkTerminalPermission(host)
  if (!permCheck.ok) {
    const tab = terminalTabs.value.find(t => t.id === tabId)
    if (tab) {
      tab.connecting = false
      tab.connected = false
    }
    ElMessage.warning(permCheck.message)
    return
  }

  // Windows RDP connection
  if (host.osType === 'windows') {
    initGuacamole(tabId, host)
    return
  }


  // 等待容器获得正确的尺寸（不为0）
  let attempts = 0
  while ((el.clientWidth === 0 || el.clientHeight === 0) && attempts < 50) {
    await new Promise(resolve => setTimeout(resolve, 50))
    attempts++
  }

  if (el.clientWidth === 0 || el.clientHeight === 0) {
    return
  }


  // 创建新终端
  const term = new Terminal({
    cursorBlink: true,
    fontSize: 16,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#d4d4d4',
      selection: '#264f78',
      black: '#1e1e1e',
      red: '#f14c4c',
      green: '#23d18b',
      yellow: '#e5e510',
      blue: '#409eff',
      magenta: '#d16969',
      cyan: '#4ec9b0',
      white: '#d4d4d4',
      brightBlack: '#808080',
      brightRed: '#f14c4c',
      brightGreen: '#23d18b',
      brightYellow: '#dcdb77',
      brightBlue: '#409eff',
      brightMagenta: '#d16969',
      brightCyan: '#4ec9b0',
      brightWhite: '#ffffff',
    }
  })

  const fitAddon = new FitAddon()

  term.loadAddon(fitAddon)
  term.open(el)

  terminals.value[tabId] = term
  fitAddons.value[tabId] = fitAddon

  // 打印容器信息

  // 等待DOM完全渲染后再获取准确的终端尺寸
  await nextTick()
  await new Promise(resolve => setTimeout(resolve, 300))


  // 适配终端大小
  try {
    fitAddon.fit()
  } catch (e) {
  }

  // 再次等待并重新适配以确保尺寸正确
  await new Promise(resolve => setTimeout(resolve, 200))
  try {
    fitAddon.fit()
  } catch (e) {
  }

  const dims = { cols: term.cols, rows: term.rows }

  // 检查cols是否异常小，如果是则使用合理的默认值
  if (dims.cols < 80) {
    dims.cols = 120
  }
  if (dims.rows < 20) {
    dims.rows = 30
  }


  term.writeln('\x1b[1;32m正在连接...\x1b[0m')

  // 连接SSH - 直接连接到后端服务器（不通过 Vite 代理）
  const token = localStorage.getItem('token') || ''
  // 根据当前环境判断后端地址
  const isDev = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const backendHost = window.location.hostname
  const backendPort = isDev ? ':9876' : (window.location.port ? ':' + window.location.port : '')
  // 将终端尺寸作为参数传递
  const wsUrl = `${protocol}//${backendHost}${backendPort}/api/v1/asset/terminal/${host.id}?token=${token}&cols=${dims.cols}&rows=${dims.rows}`


  const ws = new WebSocket(wsUrl)

  // 处理终端输入 - 发送到 WebSocket
  term.onData(data => {
    // 直接发送到服务器
    if (ws.readyState === WebSocket.OPEN) {
      try {
        ws.send(data)
      } catch (e) {
      }
    } else {
    }
  })

  ws.onopen = () => {
    const tab = terminalTabs.value.find(t => t.id === tabId)
    if (tab) {
      tab.connecting = false
      tab.connected = true
    }

    if (term) {
      term.writeln('\x1b[1;32m✓ 连接成功\x1b[0m')
      term.writeln(`\x1b[90m已连接到: ${host.name} (${host.ip}:${host.port})\x1b[0m`)
      term.writeln('')
    }
  }

  ws.onmessage = async (event) => {
    if (term) {
      // WebSocket现在发送二进制消息，需要正确处理
      if (event.data instanceof Blob) {
        const arrayBuffer = await event.data.arrayBuffer()
        const uint8Array = new Uint8Array(arrayBuffer)
        term.write(uint8Array)
      } else if (event.data instanceof ArrayBuffer) {
        const uint8Array = new Uint8Array(event.data)
        term.write(uint8Array)
      } else {
        // 兼容文本消息
        term.write(event.data)
      }
    }
  }

  ws.onerror = () => {
    if (term) {
      term.writeln('\x1b[1;31m✗ 连接失败，请检查主机配置或联系管理员\x1b[0m')
    }
  }

  ws.onclose = () => {
    const tab = terminalTabs.value.find(t => t.id === tabId)
    if (tab) {
      tab.connected = false
    }

    if (term) {
      term.writeln('\r\n\x1b[1;33m⟳ 连接已关闭\x1b[0m')
    }
  }

  wss.value[tabId] = ws

  // 添加窗口resize事件监听
  const handleResize = () => {
    if (fitAddon && term && ws.readyState === WebSocket.OPEN) {
      try {
        fitAddon.fit()
        const newDims = { cols: term.cols, rows: term.rows }

        // 发送resize消息到后端
        const resizeMsg = JSON.stringify({
          type: 'resize',
          cols: newDims.cols,
          rows: newDims.rows
        })
        ws.send(resizeMsg)
      } catch (e) {
      }
    }
  }

  // 使用防抖处理resize事件
  let resizeTimer: ReturnType<typeof setTimeout>
  const debouncedResize = () => {
    clearTimeout(resizeTimer)
    resizeTimer = setTimeout(handleResize, 100)
  }

  window.addEventListener('resize', debouncedResize)

  // 保存cleanup函数
  const cleanup = () => {
    window.removeEventListener('resize', debouncedResize)
    clearTimeout(resizeTimer)
  }



  // 保存到resizeCleanups，以便在标签关闭时调用
  resizeCleanups.value[tabId] = cleanup

  // 在ws关闭时清理
  const originalOnClose = ws.onclose
  ws.onclose = (e) => {
    cleanup()
    delete resizeCleanups.value[tabId]
    if (originalOnClose) {
      originalOnClose.call(ws, e as CloseEvent)
    }
  }
}



// 打开网络设备终端
const openNetworkDeviceTerminal = async (device: any) => {
  const tabId = Date.now().toString()

  const sameDeviceTabs = terminalTabs.value.filter(t => t.host && t.host.id === device.id && t.host._isNetworkDevice)
  let label = device.name || device.ip
  if (sameDeviceTabs.length > 0) {
    label = `${label} (${sameDeviceTabs.length + 1})`
  }

  const newTab: TerminalTab = {
    id: tabId,
    label: `${label} [${device.protocol?.toUpperCase() || 'SSH'}]`,
    host: { ...device, _isNetworkDevice: true },
    connected: false,
    connecting: true
  }

  terminalTabs.value.push(newTab)
  activeTab.value = tabId

  await nextTick()
  initNetworkDeviceTerminal(tabId, device)
}

// 初始化网络设备终端（SSH 或 Telnet）
const initNetworkDeviceTerminal = async (tabId: string, device: any) => {
  await nextTick()

  const el = terminalRefs.value[tabId]
  if (!el) return

  // 等待容器获得正确的尺寸
  let attempts = 0
  while ((el.clientWidth === 0 || el.clientHeight === 0) && attempts < 50) {
    await new Promise(resolve => setTimeout(resolve, 50))
    attempts++
  }

  const term = new Terminal({
    cursorBlink: true,
    fontSize: 16,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#d4d4d4',
      selection: '#264f78',
      black: '#1e1e1e',
      red: '#f14c4c',
      green: '#23d18b',
      yellow: '#e5e510',
      blue: '#409eff',
      magenta: '#d16969',
      cyan: '#4ec9b0',
      white: '#d4d4d4',
      brightBlack: '#808080',
      brightRed: '#f14c4c',
      brightGreen: '#23d18b',
      brightYellow: '#dcdb77',
      brightBlue: '#409eff',
      brightMagenta: '#d16969',
      brightCyan: '#4ec9b0',
      brightWhite: '#ffffff',
    }
  })

  const fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(el)

  terminals.value[tabId] = term
  fitAddons.value[tabId] = fitAddon

  await nextTick()
  await new Promise(resolve => setTimeout(resolve, 300))
  try { fitAddon.fit() } catch {}
  await new Promise(resolve => setTimeout(resolve, 200))
  try { fitAddon.fit() } catch {}

  const dims = { cols: term.cols, rows: term.rows }
  if (dims.cols < 80) dims.cols = 120
  if (dims.rows < 20) dims.rows = 30

  term.writeln(`\x1b[1;33m正在连接 ${device.name} (${device.ip}) [${device.protocol?.toUpperCase()}]...\x1b[0m`)

  const token = localStorage.getItem('token') || ''
  const isDev = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const backendHost = window.location.hostname
  const backendPort = isDev ? ':9876' : (window.location.port ? ':' + window.location.port : '')
  const wsUrl = `${protocol}//${backendHost}${backendPort}/api/v1/asset/network-terminal/${device.id}?token=${token}&cols=${dims.cols}&rows=${dims.rows}`

  const ws = new WebSocket(wsUrl)

  term.onData(data => {
    if (ws.readyState === WebSocket.OPEN) {
      try { ws.send(data) } catch {}
    }
  })

  let deviceConnected = false

  ws.onopen = () => {
    // WebSocket 已连接到后端，SSH/Telnet 连接正在建立中...
  }

  ws.onmessage = async (event) => {
    if (event.data instanceof Blob) {
      // 二进制数据 = 来自 SSH/Telnet 的终端输出，说明设备连接成功
      if (!deviceConnected) {
        deviceConnected = true
        const tab = terminalTabs.value.find(t => t.id === tabId)
        if (tab) {
          tab.connecting = false
          tab.connected = true
        }
        term.writeln(`\x1b[1;32m✓ 连接成功\x1b[0m`)
        term.writeln(`\x1b[90m已连接到: ${device.name} (${device.ip}:${device.port}) [${device.protocol?.toUpperCase()}]\x1b[0m`)
        term.writeln('')
      }
      const arrayBuffer = await event.data.arrayBuffer()
      term.write(new Uint8Array(arrayBuffer))
    } else if (event.data instanceof ArrayBuffer) {
      if (!deviceConnected) {
        deviceConnected = true
        const tab = terminalTabs.value.find(t => t.id === tabId)
        if (tab) {
          tab.connecting = false
          tab.connected = true
        }
        term.writeln(`\x1b[1;32m✓ 连接成功\x1b[0m`)
        term.writeln(`\x1b[90m已连接到: ${device.name} (${device.ip}:${device.port}) [${device.protocol?.toUpperCase()}]\x1b[0m`)
        term.writeln('')
      }
      term.write(new Uint8Array(event.data))
    } else {
      // 文本消息 = 后端发送的状态/错误信息（如 "SSH连接失败: ..."）
      term.write(`\r\n\x1b[1;31m${event.data}\x1b[0m`)
    }
  }

  ws.onerror = () => {
    if (!deviceConnected) {
      term.writeln('\x1b[1;31m✗ WebSocket 连接失败\x1b[0m')
    }
  }

  ws.onclose = () => {
    const tab = terminalTabs.value.find(t => t.id === tabId)
    if (tab) {
      tab.connecting = false
      tab.connected = false
    }
    if (!deviceConnected) {
      term.writeln('\x1b[1;31m✗ 连接失败，请检查设备配置或凭证\x1b[0m')
    }
    term.writeln('\r\n\x1b[1;33m⟳ 连接已关闭\x1b[0m')
  }

  wss.value[tabId] = ws

  // resize 处理
  const handleResize = () => {
    if (fitAddon && term && ws.readyState === WebSocket.OPEN) {
      try {
        fitAddon.fit()
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      } catch {}
    }
  }

  let resizeTimer: ReturnType<typeof setTimeout>
  const debouncedResize = () => {
    clearTimeout(resizeTimer)
    resizeTimer = setTimeout(handleResize, 100)
  }

  window.addEventListener('resize', debouncedResize)
  resizeCleanups.value[tabId] = () => {
    window.removeEventListener('resize', debouncedResize)
    clearTimeout(resizeTimer)
  }

  const originalOnClose = ws.onclose
  ws.onclose = (e) => {
    if (resizeCleanups.value[tabId]) {
      resizeCleanups.value[tabId]()
      delete resizeCleanups.value[tabId]
    }
    if (originalOnClose) originalOnClose.call(ws, e as CloseEvent)
  }
}

// 发送按键组合
const sendKeyCombination = (tabId: string, keys: number[]) => {
  const client = guacamoleClients.value[tabId]
  if (!client) return

  // 按下按键
  keys.forEach(k => client.sendKeyEvent(1, k))
  // 释放按键 (反向)
  const reversedKeys = [...keys].reverse()
  reversedKeys.forEach(k => client.sendKeyEvent(0, k))
}

// 虚拟键盘相关
const virtualKeyboardVisible = ref(false)
const virtualKeyboardTabId = ref('')
const kbPos = ref({ x: Math.max(0, (window.innerWidth - 680) / 2), y: window.innerHeight - 340 })
const kbDragState = ref({ dragging: false, offsetX: 0, offsetY: 0 })

const toggleVirtualKeyboard = (tabId: string) => {
  virtualKeyboardTabId.value = tabId
  virtualKeyboardVisible.value = !virtualKeyboardVisible.value
}

const startKbDrag = (e: MouseEvent) => {
  kbDragState.value = {
    dragging: true,
    offsetX: e.clientX - kbPos.value.x,
    offsetY: e.clientY - kbPos.value.y
  }
  const onMouseMove = (ev: MouseEvent) => {
    if (!kbDragState.value.dragging) return
    kbPos.value = {
      x: Math.max(0, Math.min(window.innerWidth - 200, ev.clientX - kbDragState.value.offsetX)),
      y: Math.max(0, Math.min(window.innerHeight - 50, ev.clientY - kbDragState.value.offsetY))
    }
  }
  const onMouseUp = () => {
    kbDragState.value.dragging = false
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  }
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

// US 键盘布局 (X11 keysym)
const keyboardLayout = {
  functionRow: [
    { label: 'Esc', code: 0xFF1B },
    { label: 'F1', code: 0xFFBE }, { label: 'F2', code: 0xFFBF }, { label: 'F3', code: 0xFFC0 }, { label: 'F4', code: 0xFFC1 },
    { label: 'F5', code: 0xFFC2 }, { label: 'F6', code: 0xFFC3 }, { label: 'F7', code: 0xFFC4 }, { label: 'F8', code: 0xFFC5 },
    { label: 'F9', code: 0xFFC6 }, { label: 'F10', code: 0xFFC7 }, { label: 'F11', code: 0xFFC8 }, { label: 'F12', code: 0xFFC9 },
    { label: 'PrtSc', code: 0xFF61 }, { label: 'Del', code: 0xFFFF }
  ],
  numberRow: [
    { label: '`', code: 0x60 }, { label: '1', code: 0x31 }, { label: '2', code: 0x32 }, { label: '3', code: 0x33 },
    { label: '4', code: 0x34 }, { label: '5', code: 0x35 }, { label: '6', code: 0x36 }, { label: '7', code: 0x37 },
    { label: '8', code: 0x38 }, { label: '9', code: 0x39 }, { label: '0', code: 0x30 },
    { label: '-', code: 0x2D }, { label: '=', code: 0x3D },
    { label: 'Backspace', code: 0xFF08, wide: true }
  ],
  row1: [
    { label: 'Tab', code: 0xFF09, wide: true },
    { label: 'Q', code: 0x71 }, { label: 'W', code: 0x77 }, { label: 'E', code: 0x65 }, { label: 'R', code: 0x72 },
    { label: 'T', code: 0x74 }, { label: 'Y', code: 0x79 }, { label: 'U', code: 0x75 }, { label: 'I', code: 0x69 },
    { label: 'O', code: 0x6F }, { label: 'P', code: 0x70 },
    { label: '[', code: 0x5B }, { label: ']', code: 0x5D }, { label: '\\', code: 0x5C }
  ],
  row2: [
    { label: 'Caps', code: 0xFFE5, wide: true },
    { label: 'A', code: 0x61 }, { label: 'S', code: 0x73 }, { label: 'D', code: 0x64 }, { label: 'F', code: 0x66 },
    { label: 'G', code: 0x67 }, { label: 'H', code: 0x68 }, { label: 'J', code: 0x6A }, { label: 'K', code: 0x6B },
    { label: 'L', code: 0x6C }, { label: ';', code: 0x3B }, { label: "'", code: 0x27 },
    { label: 'Enter', code: 0xFF0D, wide: true }
  ],
  row3: [
    { label: 'Shift', code: 0xFFE1, wide: true },
    { label: 'Z', code: 0x7A }, { label: 'X', code: 0x78 }, { label: 'C', code: 0x63 }, { label: 'V', code: 0x76 },
    { label: 'B', code: 0x62 }, { label: 'N', code: 0x6E }, { label: 'M', code: 0x6D },
    { label: ',', code: 0x2C }, { label: '.', code: 0x2E }, { label: '/', code: 0x2F },
    { label: '↑', code: 0xFF52 },
    { label: 'Shift', code: 0xFFE2, wide: true }
  ],
  spaceRow: [
    { label: 'Ctrl', code: 0xFFE3, wide: true },
    { label: 'Win', code: 0xFFEB },
    { label: 'Alt', code: 0xFFE9 },
    { label: 'Space', code: 0x20, space: true },
    { label: 'Alt', code: 0xFFEA },
    { label: '←', code: 0xFF51 }, { label: '↓', code: 0xFF54 }, { label: '→', code: 0xFF53 }
  ]
}

// 发送虚拟键盘按键
const sendVKey = (keyCode: number) => {
  const tabId = virtualKeyboardTabId.value
  const client = guacamoleClients.value[tabId]
  if (!client) return
  client.sendKeyEvent(1, keyCode)
  client.sendKeyEvent(0, keyCode)
}

// 关闭指定标签
const closeTerminal = (tabId: string) => {
  const tab = terminalTabs.value.find(t => t.id === tabId)
  if (!tab) return

  // 清理resize监听器
  if (resizeCleanups.value[tabId]) {
    resizeCleanups.value[tabId]()
    delete resizeCleanups.value[tabId]
  }

  // 关闭WebSocket
  if (wss.value[tabId]) {
    wss.value[tabId].close()
    delete wss.value[tabId]
  }

  // 销毁终端
  if (terminals.value[tabId]) {
    terminals.value[tabId]?.dispose()
    delete terminals.value[tabId]
  }

  // Destruction Guacamole
  if (guacamoleClients.value[tabId]) {
    guacamoleClients.value[tabId].disconnect()
    delete guacamoleClients.value[tabId]
    delete guacamoleTunnels.value[tabId]
    delete guacamoleRefs.value[tabId]
  }
  if (guacamoleResizeObservers.value[tabId]) {
    guacamoleResizeObservers.value[tabId].disconnect()
    delete guacamoleResizeObservers.value[tabId]
  }

  // 删除标签
  const index = terminalTabs.value.findIndex(t => t.id === tabId)
  if (index > -1) {
    terminalTabs.value.splice(index, 1)
  }

  // 如果没有标签了，添加一个默认的空标签
  if (terminalTabs.value.length === 0) {
    terminalTabs.value.push({
      id: Date.now().toString(),
      label: '新建终端',
      host: null,
      connected: false,
      connecting: false
    })
  }

  // 切换到第一个标签
  activeTab.value = terminalTabs.value[0].id
}

// 处理标签关闭
const handleTabRemove = (tabId: string) => {
  closeTerminal(tabId)
}

// 初始化
onMounted(async () => {
  await loadGroupTree()
  await loadAllHosts()
  await loadAllNetworkDevices()
  await loadNetworkGroupTree()

  // 检查是否有来自 query 参数的网络设备连接请求
  if (route.query.type === 'network-device') {
    sidebarMode.value = 'network-device'
    if (route.query.deviceId) {
      const deviceId = Number(route.query.deviceId)
      const device = allNetworkDevices.value.find((d: any) => d.id === deviceId)
      if (device) {
        await new Promise(resolve => setTimeout(resolve, 200))
        openNetworkDeviceTerminal(device)
      }
    }
  }

  // 检查是否有从 Hosts 页面双击传来的主机列表
  const dblClickHosts = sessionStorage.getItem('dblClickHosts')
  if (dblClickHosts) {
    try {
      const hosts = JSON.parse(dblClickHosts)
      if (Array.isArray(hosts) && hosts.length > 0) {
        // 清空 sessionStorage
        sessionStorage.removeItem('dblClickHosts')

        // 为每个主机打开一个终端标签
        for (const host of hosts) {
          // 等待一下，避免同时打开多个连接
          await new Promise(resolve => setTimeout(resolve, 100))
          openTerminal(host)
        }
      }
    } catch (e) {
    }
  }
})

// 组件销毁时关闭所有连接
onBeforeUnmount(() => {
  // 清理所有resize监听器
  Object.keys(resizeCleanups.value).forEach(tabId => {
    if (resizeCleanups.value[tabId]) {
      resizeCleanups.value[tabId]()
    }
  })
  resizeCleanups.value = {}

  // 关闭所有WebSocket
  Object.keys(wss.value).forEach(tabId => {
    if (wss.value[tabId]) {
      wss.value[tabId].close()
    }
  })

  // 销毁所有终端
  Object.values(terminals.value).forEach(term => {
    term?.dispose()
  })
  
  // Cleanup Guacamole
  Object.values(guacamoleClients.value).forEach(client => {
    client?.disconnect()
  })
})
</script>

<style scoped>
.terminal-page {
  display: flex;
  height: 100vh;
  background: #1e1e1e;
}



.guacamole-container {
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  background-color: #1e1e1e;
  overflow: hidden;
  
  :deep(div) {
    cursor: none;
  }

  :deep(canvas) {
    image-rendering: auto;
  }
}

/* 左侧边栏 */
.terminal-sidebar {
  width: 280px;
  min-width: 280px;
  background: #252526;
  border-right: 1px solid #3c3c3c;
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  padding: 16px;
  border-bottom: 1px solid #3c3c3c;
}

.sidebar-segmented {
  display: flex;
  background: #1e1e1e;
  border: 1px solid #3c3c3c;
  margin-bottom: 12px;
  overflow: hidden;
}

.seg-item {
  flex: 1;
  text-align: center;
  padding: 6px 0;
  font-size: 12px;
  color: #858585;
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.seg-item:hover {
  color: #cccccc;
  background: rgba(255, 255, 255, 0.05);
}

.seg-item.active {
  background: #4ec9b0;
  color: #1e1e1e;
  font-weight: 600;
}

.sidebar-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #cccccc;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 12px;
}

.sidebar-title .el-icon {
  font-size: 18px;
  color: #4ec9b0;
}

.host-count {
  margin-left: auto;
  font-size: 12px;
  color: #858585;
  font-weight: normal;
}

.node-protocol-tag {
  margin-left: auto;
  font-size: 10px;
  transform: scale(0.85);
  border-radius: 0 !important;
}

.search-box {
  margin-top: 8px;
}

.search-input {
  width: 100%;
}

.search-input :deep(.el-input__wrapper) {
  background: #3c3c3c;
  border: 1px solid #4e4e4e;
  box-shadow: none;
  border-radius: 0;
}

.search-input :deep(.el-input__wrapper:hover) {
  border-color: #5e5e5e;
}

.search-input :deep(.el-input__wrapper.is-focus) {
  border-color: #4ec9b0;
}

.search-input :deep(.el-input__inner) {
  color: #cccccc;
}

.search-input :deep(.el-input__inner::placeholder) {
  color: #858585;
}

.search-input :deep(.el-input__prefix) {
  color: #858585;
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.sidebar-content::-webkit-scrollbar {
  width: 8px;
}

.sidebar-content::-webkit-scrollbar-track {
  background: #1e1e1e;
}

.sidebar-content::-webkit-scrollbar-thumb {
  background: #4e4e4e;
  border-radius: 0;
}

.sidebar-content::-webkit-scrollbar-thumb:hover {
  background: #5e5e5e;
}

/* 树形结构样式 */
.terminal-tree {
  background: transparent;
  color: #cccccc;
}

.terminal-tree :deep(.el-tree-node__content) {
  height: auto;
  padding: 4px 0;
  background: transparent;
  border-radius: 0;
  transition: background-color 0.2s ease;
}

.terminal-tree :deep(.el-tree-node__content:hover) {
  background: rgba(255, 255, 255, 0.05);
}

.terminal-tree :deep(.is-current > .el-tree-node__content) {
  background: rgba(78, 201, 176, 0.1);
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  padding: 4px 8px;
  border-radius: 0;
}

.node-icon {
  font-size: 16px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.node-icon .el-icon {
  color: #858585;
}

.host-svg-icon {
  width: 16px;
  height: 16px;
  color: #858585;
}

.node-label {
  flex: 1;
  font-size: 13px;
  color: #cccccc;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-count {
  font-size: 11px;
  color: #858585;
  margin-left: 4px;
}

.node-status {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.node-status.online {
  background: #4ec9b0;
  box-shadow: 0 0 4px rgba(78, 201, 176, 0.4);
}

.node-status.offline {
  background: #858585;
}

.node-status.unknown {
  background: #dcdcaa;
}

/* 右侧主区域 */
.terminal-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #1e1e1e;
  overflow: hidden;
  position: relative;
}

.tabs-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.terminal-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.terminal-tabs :deep(.el-tabs__header) {
  background: #2d2d30;
  border-bottom: 1px solid #3c3c3c;
  margin: 0;
  padding: 0 8px;
}

.terminal-tabs :deep(.el-tabs__nav-wrap) {
  padding: 8px 0 0;
}

.terminal-tabs :deep(.el-tabs__nav) {
  border: none;
}

.terminal-tabs :deep(.el-tabs__item) {
  color: #cccccc;
  border: 1px solid #3c3c3c;
  border-bottom: none;
  padding: 0 12px;
  height: 36px;
  line-height: 34px;
  margin-right: 4px;
  border-radius: 0;
  background: #3c3c3c;
  transition: all 0.2s ease;
  font-size: 12px;
}

.tab-content {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.terminal-tabs :deep(.el-tabs__item:hover) {
  background: #4e4e4e;
}

.terminal-tabs :deep(.el-tabs__item.is-active) {
  color: #cccccc;
  background: #1e1e1e;
  border-color: #3c3c3c;
  border-bottom: 1px solid #1e1e1e;
}

.terminal-tabs :deep(.el-tabs__active-bar) {
  display: none;
}

.terminal-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
  padding: 0;
  height: 100%;
}

.terminal-tabs :deep(.el-tab-pane) {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 6px;
}

.tab-icon {
  width: 14px;
  height: 14px;
}

.tab-name {
  font-size: 12px;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.status-dot.online {
  background: #4ec9b0;
  box-shadow: 0 0 4px rgba(78, 201, 176, 0.6);
}

.status-dot.connecting {
  background: #e5e510;
  animation: pulse 1s ease-in-out infinite;
}

.status-dot.offline {
  background: #858585;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.4;
  }
}


.header-right-absolute {
  position: absolute;
  top: 0;
  right: 0;
  z-index: 1000;
  height: 40px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding-right: 16px;
  background: #2d2d30;
  pointer-events: auto;
}

.back-btn {
  color: #cccccc !important;
  font-size: 13px;
  &:hover {
    color: #ffffff !important;
  }
}

.terminal-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  height: 100%;
}

.terminal-user-info {
  display: flex;
  align-items: center;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
}

.terminal-user-info:hover {
  background: #3c3c3c;
}

.user-avatar {
  background: #0a466a;
}

.user-details {
  display: flex;
  flex-direction: column;
  margin-left: 8px;
  margin-right: 8px;
}

.user-name {
  font-size: 14px;
  color: #cccccc;
  line-height: 1.2;
}

.user-role {
  font-size: 11px;
  color: #858585;
  line-height: 1.2;
}

.dropdown-arrow {
  font-size: 12px;
  color: #858585;
}


.terminal-connected {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.terminal-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  position: relative;
  background: #000;
  overflow: hidden;
}

.terminal-viewport {
  flex: 1;
  position: relative;
  overflow: hidden;
}

.terminal-toolbar {
  padding: 6px 12px;
  background: #252526;
  border-bottom: 1px solid #3c3c3c;
  display: flex;
  gap: 8px;
  align-items: center;
}

.terminal-toolbar :deep(.el-button) {
  background: #3c3c3c;
  border-color: #4e4e4e;
  color: #cccccc;
}

.terminal-toolbar :deep(.el-button:hover) {
  background: #4e4e4e;
  border-color: #5e5e5e;
  color: #ffffff;
}


.xterm-container {
  width: 100%;
  height: 100%;
  padding: 0;
  box-sizing: border-box;
}

.xterm-container :deep(.xterm) {
  height: 100%;
}

.xterm-container :deep(.xterm .xterm-viewport) {
  background-color: #1e1e1e !important;
}

.xterm-container :deep(.xterm .xterm-screen) {
  padding: 0;
}

.terminal-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: #1e1e1e;
}

.empty-icon {
  width: 64px;
  height: 64px;
  margin-bottom: 16px;
  opacity: 0.3;
}

.empty-icon svg {
  width: 100%;
  height: 100%;
  fill: #cccccc;
}

.empty-text {
  font-size: 14px;
  color: #858585;
  margin-bottom: 8px;
}

.empty-hint {
  font-size: 12px;
  color: #606060;
}

/* 关闭按钮样式 */
.terminal-tabs :deep(.el-icon-close) {
  color: #858585;
  transition: color 0.2s ease;
}

.terminal-tabs :deep(.el-icon-close:hover) {
  color: #cccccc;
}

</style>

<!-- 非 scoped 样式：el-drawer 会 teleport 到 body，scoped 样式无法生效 -->
<style>
/* 文件管理 & 虚拟键盘抽屉 - 深色主题 */
.file-drawer-dark .el-drawer {
  background: #1e1e1e !important;
  border-left: 1px solid #3c3c3c;
}
.file-drawer-dark .el-drawer__header {
  background: #2d2d30;
  border-bottom: 1px solid #3c3c3c;
  color: #cccccc;
  margin-bottom: 0;
  padding: 16px 20px;
}
.file-drawer-dark .el-drawer__title {
  color: #cccccc;
  font-weight: 600;
}
.file-drawer-dark .el-drawer__close-btn {
  color: #858585;
}
.file-drawer-dark .el-drawer__close-btn:hover {
  color: #cccccc;
}
.file-drawer-dark .el-drawer__body {
  background: #252526;
  padding: 16px;
}

/* 文件管理内容 */
.file-drawer-dark .file-manager-content {
  padding: 0;
}

.file-drawer-dark .upload-section,
.file-drawer-dark .download-section {
  margin-bottom: 24px;
}

.file-drawer-dark .upload-section h3,
.file-drawer-dark .download-section h3 {
  font-size: 14px;
  color: #cccccc;
  margin-bottom: 12px;
  font-weight: 600;
}

.file-drawer-dark .upload-demo {
  margin-bottom: 12px;
}

.file-drawer-dark .upload-demo .el-upload-dragger {
  background: #1e1e1e;
  border: 1px dashed #4e4e4e;
  border-radius: 4px;
  padding: 20px;
  transition: border-color 0.2s;
}

.file-drawer-dark .upload-demo .el-upload-dragger:hover {
  border-color: #4ec9b0;
}

.file-drawer-dark .upload-demo .el-upload-dragger.is-dragover {
  border-color: #4ec9b0;
  background: rgba(78, 201, 176, 0.05);
}

.file-drawer-dark .upload-demo .el-icon--upload {
  color: #858585;
  font-size: 32px;
  margin-bottom: 8px;
}

.file-drawer-dark .upload-demo .el-upload__text {
  color: #858585;
  font-size: 13px;
}

.file-drawer-dark .upload-demo .el-upload__text em {
  color: #4ec9b0;
}

.file-drawer-dark .upload-demo .el-upload__tip {
  color: #606060;
  font-size: 11px;
  margin-top: 4px;
  text-align: center;
}

.file-drawer-dark .file-preview {
  background: #1e1e1e;
  padding: 12px;
  border-radius: 4px;
  border: 1px solid #3c3c3c;
}

.file-drawer-dark .file-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #cccccc;
  margin-bottom: 10px;
}

.file-drawer-dark .file-info .el-icon {
  color: #4ec9b0;
  flex-shrink: 0;
}

.file-drawer-dark .file-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-drawer-dark .file-size {
  color: #858585;
  font-size: 12px;
  flex-shrink: 0;
}

.file-drawer-dark .progress-area {
  margin-bottom: 0;
}

.file-drawer-dark .progress-area .el-progress-bar__outer {
  background: #3c3c3c;
}

.file-drawer-dark .progress-area .el-progress-bar__inner {
  background: #4ec9b0;
}

.file-drawer-dark .progress-area .el-progress__text {
  color: #cccccc;
  font-size: 12px;
}

.file-drawer-dark .upload-speed {
  font-size: 11px;
  color: #858585;
  text-align: right;
  margin-top: 4px;
}

.file-drawer-dark .hint-text {
  font-size: 13px;
  color: #b0b0b0;
  line-height: 1.6;
  background: #1e1e1e;
  padding: 12px;
  border-radius: 4px;
  border-left: 3px solid #4ec9b0;
}

.file-drawer-dark .hint-text b {
  color: #4ec9b0;
}

/* 悬浮虚拟键盘 */
.floating-keyboard {
  position: fixed;
  z-index: 99999;
  min-width: 640px;
  background: rgba(30, 30, 30, 0.78);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
  overflow: hidden;
  user-select: none;
}

.kb-titlebar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.05);
  cursor: grab;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.kb-titlebar:active {
  cursor: grabbing;
}

.kb-title {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.65);
  letter-spacing: 0.5px;
}

.kb-close {
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.5);
  font-size: 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}

.kb-close:hover {
  background: #e06c75;
  color: #fff;
}

.floating-keyboard .virtual-keyboard {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 10px;
}

.floating-keyboard .kb-row {
  display: flex;
  gap: 3px;
  justify-content: center;
}

.floating-keyboard .kb-key {
  min-width: 36px;
  height: 32px;
  padding: 0 6px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 5px;
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.82);
  font-size: 11px;
  font-family: 'Segoe UI', system-ui, sans-serif;
  cursor: pointer;
  transition: all 0.1s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  user-select: none;
  white-space: nowrap;
}

.floating-keyboard .kb-key:hover {
  background: rgba(255, 255, 255, 0.16);
  border-color: rgba(255, 255, 255, 0.22);
}

.floating-keyboard .kb-key:active {
  background: #4ec9b0;
  border-color: #4ec9b0;
  color: #1e1e1e;
  transform: scale(0.94);
}

.floating-keyboard .kb-fn {
  min-width: 32px;
  font-size: 10px;
  height: 26px;
  background: rgba(255, 255, 255, 0.04);
}

.floating-keyboard .kb-wide {
  min-width: 56px;
  flex-shrink: 0;
}

.floating-keyboard .kb-space {
  flex: 1;
  max-width: 280px;
}
</style>
