<template>
  <div class="menus-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Menu /></el-icon>
        </div>
        <div>
          <h2 class="page-title">菜单管理</h2>
          <p class="page-subtitle">管理系统菜单和插件菜单，支持目录、菜单、按钮三级层级结构</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="handleAdd">
          <el-icon style="margin-right: 6px;"><Plus /></el-icon>
          新增菜单
        </el-button>
        <el-button @click="toggleExpandAll">
          <el-icon style="margin-right: 6px;"><Sort /></el-icon>
          {{ expandAll ? '折叠全部' : '展开全部' }}
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchForm.name"
          placeholder="搜索菜单名称..."
          clearable
          class="search-input"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

        <el-select
          v-model="searchForm.status"
          placeholder="菜单状态"
          clearable
          class="search-input"
        >
          <el-option label="启用" :value="1" />
          <el-option label="禁用" :value="0" />
        </el-select>
      </div>

      <div class="search-actions">
        <el-button class="reset-btn" @click="handleReset">
          <el-icon style="margin-right: 4px;"><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <!-- 表格容器 -->
    <div class="table-wrapper">
      <el-table
        ref="tableRef"
        :data="filteredMenuList"
        v-loading="loading"
        :row-key="getRowKey"
        :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
        :default-expand-all="false"
        :indent="30"
        class="modern-table menu-tree-table"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        :key="tableKey"
      >
        <el-table-column label="菜单名称" prop="name" min-width="280">
          <template #default="{ row }">
            <span style="display: inline-flex; align-items: center;">
              <el-icon v-if="row.icon" :size="16" style="margin-right: 8px;">
                <component :is="getIconComponent(row.icon)" />
              </el-icon>
              {{ row.name }}
            </span>
          </template>
        </el-table-column>

        <el-table-column prop="code" min-width="150">
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-gold"><Key /></el-icon>
              菜单编码
            </span>
          </template>
        </el-table-column>

        <el-table-column label="类型" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.type === 1" class="menu-type-tag directory-tag">目录</el-tag>
            <el-tag v-else-if="row.type === 2" class="menu-type-tag menu-tag">菜单</el-tag>
            <el-tag v-else class="menu-type-tag button-tag">按钮</el-tag>
            <el-tag v-if="row.isPlugin" type="warning" size="small" effect="plain" style="margin-left: 5px;">插件</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="路由路径" prop="path" min-width="180" />

        <el-table-column label="图标" width="80" align="center">
          <template #default="{ row }">
            <el-icon v-if="row.icon" :size="18">
              <component :is="getIconComponent(row.icon)" />
            </el-icon>
            <span v-else style="color: #909399;">-</span>
          </template>
        </el-table-column>

        <el-table-column label="排序" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.sort || 0 }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" effect="dark">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="150" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip v-if="row.isPlugin" content="调整排序" placement="top">
                <el-button link class="action-btn action-sort" @click="handleEditPluginSort(row)">
                  <el-icon><Sort /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="!row.isPlugin" content="编辑" placement="top">
                <el-button link class="action-btn action-edit" @click="handleEdit(row)">
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip v-if="!row.isPlugin" content="删除" placement="top">
                <el-button link class="action-btn action-delete" @click="handleDelete(row)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="50%"
      class="menu-edit-dialog responsive-dialog"
      :close-on-click-modal="false"
      @close="handleDialogClose"
    >
      <el-alert
        v-if="editingPluginMenu"
        title="插件菜单编辑"
        type="info"
        :closable="false"
        style="margin-bottom: 15px;"
      >
        <template #default>
          <div>您正在编辑插件菜单，只能修改排序字段</div>
          <div style="font-size: 12px; color: #666; margin-top: 5px;">
            菜单名称: {{ menuForm.name }} | 路径: {{ menuForm.path }}
          </div>
        </template>
      </el-alert>

      <el-form :model="menuForm" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="菜单名称" prop="name">
          <el-input v-model="menuForm.name" :disabled="editingPluginMenu" placeholder="请输入菜单名称" />
        </el-form-item>
        <el-form-item label="菜单编码" prop="code">
          <el-input v-model="menuForm.code" :disabled="editingPluginMenu" placeholder="请输入菜单编码" />
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-radio-group v-model="menuForm.type" :disabled="editingPluginMenu">
            <el-radio :label="1">目录</el-radio>
            <el-radio :label="2">菜单</el-radio>
            <el-radio :label="3">按钮</el-radio>
          </el-radio-group>
          <div class="form-tip">
            目录：用于组织菜单的容器，不对应具体页面<br>
            菜单：对应具体页面的导航项<br>
            按钮：页面内的功能按钮，用于权限控制
          </div>
        </el-form-item>
        <el-form-item label="上级菜单" prop="parentId" v-if="!editingPluginMenu">
          <el-cascader
            v-model="menuForm.parentId"
            :options="menuTreeOptions"
            :props="{ checkStrictly: true, value: 'ID', label: 'name' }"
            clearable
            placeholder="请选择上级菜单"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="路由路径" prop="path" v-if="menuForm.type !== 3">
          <el-input v-model="menuForm.path" :disabled="editingPluginMenu" placeholder="请输入路由路径" />
        </el-form-item>
        <el-form-item label="组件路径" prop="component" v-if="menuForm.type === 2 && !editingPluginMenu">
          <el-input v-model="menuForm.component" placeholder="请输入组件路径" />
        </el-form-item>
        <el-form-item label="图标" prop="icon">
          <el-input v-model="menuForm.icon" :disabled="editingPluginMenu" placeholder="请输入图标名称" />
        </el-form-item>
        <el-form-item label="排序" prop="sort">
          <el-input-number v-model="menuForm.sort" :min="0" style="width: 100%;" />
          <div class="form-tip">数值越小越靠前</div>
        </el-form-item>
        <el-form-item label="显示状态" prop="visible" v-if="!editingPluginMenu">
          <el-radio-group v-model="menuForm.visible">
            <el-radio :label="1">显示</el-radio>
            <el-radio :label="0">隐藏</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="状态" prop="status" v-if="!editingPluginMenu">
          <el-radio-group v-model="menuForm.status">
            <el-radio :label="1">启用</el-radio>
            <el-radio :label="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button class="black-button" @click="handleSubmit" :loading="submitting">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox, FormInstance, FormRules, ElTable } from 'element-plus'
import {
  Plus,
  Search,
  RefreshLeft,
  Edit,
  Delete,
  Sort,
  Fold,
  Expand,
  Key,
  Menu as MenuIcon
} from '@element-plus/icons-vue'
import {
  HomeFilled,
  User,
  UserFilled,
  OfficeBuilding,
  Menu,
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
  Odometer
} from '@element-plus/icons-vue'
import { getMenuTree, createMenu, updateMenu, deleteMenu } from '@/api/menu'
import { pluginManager } from '@/plugins/manager'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const editingPluginMenu = ref(false)
const formRef = ref<FormInstance>()
const tableRef = ref<InstanceType<typeof ElTable>>()
const expandAll = ref(false)
const tableKey = ref(0) // 强制重新渲染表格

// 获取行的唯一标识
const getRowKey = (row: any) => {
  const key = row.ID || row.id || row.path
  return key
}

// 搜索表单
const searchForm = reactive({
  name: '',
  status: undefined as number | undefined
})

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
  'Odometer': Odometer
}

// 获取图标组件
const getIconComponent = (iconName: string) => {
  return iconMap[iconName] || Menu
}

// 插件菜单排序存储 key
const PLUGIN_MENU_SORT_KEY = 'iom_plugin_menu_sort'

const menuList = ref<any[]>([])
const menuTreeOptions = ref<any[]>([])

// 过滤后的菜单列表
const filteredMenuList = computed(() => {
  if (!searchForm.name && searchForm.status === undefined) {
    return menuList.value
  }
  return filterTree(menuList.value)
})

// 递归过滤树节点
const filterTree = (nodes: any[]): any[] => {
  const result: any[] = []

  for (const node of nodes) {
    const matchName = !searchForm.name || node.name?.toLowerCase().includes(searchForm.name.toLowerCase())
    const matchStatus = searchForm.status === undefined || node.status === searchForm.status

    let filteredChildren: any[] | undefined = undefined
    if (node.children && node.children.length > 0) {
      const children = filterTree(node.children)
      // 只有当有匹配的子节点时才设置 children
      if (children.length > 0) {
        filteredChildren = children
      }
    }

    // 如果当前节点匹配或有匹配的子节点，则保留
    if ((matchName && matchStatus) || filteredChildren) {
      result.push({
        ...node,
        // 只有当有子节点时才设置 children 属性
        children: filteredChildren,
        hasChildren: !!filteredChildren
      })
    }
  }

  return result
}

// 监听搜索条件变化，自动展开
watch([() => searchForm.name, () => searchForm.status], () => {
  if (searchForm.name || searchForm.status !== undefined) {
    expandAll.value = true
    nextTick(() => {
      toggleExpandAllRows(true)
    })
  }
})

// 重置搜索
const handleReset = () => {
  searchForm.name = ''
  searchForm.status = undefined
  expandAll.value = false
  nextTick(() => {
    toggleExpandAllRows(false)
  })
}

// 切换全部展开/折叠
const toggleExpandAll = () => {
  expandAll.value = !expandAll.value
  nextTick(() => {
    toggleExpandAllRows(expandAll.value)
  })
}

// 展开/折叠所有行
const toggleExpandAllRows = (expand: boolean) => {
  const table = tableRef.value
  if (!table) return

  // 先收起所有行
  const collapseAllRows = (rows: any[]) => {
    rows.forEach(row => {
      if (row.children && row.children.length > 0) {
        table.toggleRowExpansion(row, false)
        collapseAllRows(row.children)
      }
    })
  }

  // 如果要展开，则逐个展开
  const expandAllRows = (rows: any[]) => {
    rows.forEach(row => {
      if (row.children && row.children.length > 0) {
        table.toggleRowExpansion(row, true)
        expandAllRows(row.children)
      }
    })
  }

  if (expand) {
    expandAllRows(filteredMenuList.value)
  } else {
    collapseAllRows(filteredMenuList.value)
  }
}

const menuForm = reactive({
  id: 0,
  name: '',
  code: '',
  type: 2,
  parentId: 0,
  path: '',
  component: '',
  icon: '',
  sort: 0,
  visible: 1,
  status: 1
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入菜单名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入菜单编码', trigger: 'blur' }],
  type: [{ required: true, message: '请选择类型', trigger: 'change' }]
}

// 加载插件菜单的自定义排序
const loadPluginMenuSort = (): Map<string, number> => {
  try {
    const stored = localStorage.getItem(PLUGIN_MENU_SORT_KEY)
    if (stored) {
      const sortMap = JSON.parse(stored)
      return new Map(Object.entries(sortMap))
    }
  } catch (error) {
  }
  return new Map()
}

// 保存插件菜单的自定义排序
const savePluginMenuSort = (menuPath: string, sort: number) => {
  try {
    const sortMap = loadPluginMenuSort()
    sortMap.set(menuPath, sort)
    const sortObj = Object.fromEntries(sortMap)
    localStorage.setItem(PLUGIN_MENU_SORT_KEY, JSON.stringify(sortObj))
  } catch (error) {
  }
}

// 构建插件菜单列表
const buildPluginMenuList = () => {
  const pluginMenus: any[] = []
  const installedPlugins = pluginManager.getInstalled()

  // 加载自定义排序
  const customSort = loadPluginMenuSort()

  installedPlugins.forEach(plugin => {
    if (plugin.getMenus) {
      const menus = plugin.getMenus()
      menus.forEach(menu => {
        const parentId = (menu.parentPath && menu.parentPath !== '') ? menu.parentPath : null
        const sort = customSort.get(menu.path) ?? menu.sort

        pluginMenus.push({
          ID: menu.path,
          id: menu.path,
          name: menu.name,
          code: menu.path.replace(/\//g, '_'),
          type: menu.parentPath && menu.parentPath !== '' ? 2 : 1,
          parentId: parentId,
          path: menu.path,
          component: '',
          icon: menu.icon,
          sort: sort,
          visible: menu.hidden ? 0 : 1,
          status: 1,
          isPlugin: true,
          pluginName: plugin.name
          // 不设置 children 属性，让 buildMenuTree 根据原始数据判断
        })
      })
    }
  })

  return pluginMenus
}

// 构建菜单树 - 关键：从一开始就正确处理children
const buildMenuTree = (menus: any[]) => {
  const menuMap = new Map()


  // 第一遍循环: 创建所有菜单的副本并放入 Map
  menus.forEach(menu => {
    const id = menu.ID || menu.id
    if (!id) {
      return
    }

    // 检查原始数据是否有children
    const hasOriginalChildren = menu.children && menu.children.length > 0

    // 创建菜单副本，保留原始children状态
    const menuCopy = {
      ...menu,
      children: hasOriginalChildren ? [...menu.children] : undefined,
      hasChildren: hasOriginalChildren
    }

    menuMap.set(id, menuCopy)
  })

  const tree: any[] = []

  // 第二遍循环: 只处理顶级菜单
  menus.forEach(menu => {
    const id = menu.ID || menu.id

    // 统一处理 parentId
    let parentId = menu.parentId

    // 对于系统菜单，如果 parentId 是 0，视为顶级菜单
    if (!menu.isPlugin && (!parentId || parentId === 0)) {
      parentId = null
    }

    // 只有顶级菜单才添加到tree
    if (!parentId) {
      const menuItem = menuMap.get(id)
      if (menuItem && !tree.includes(menuItem)) {
        tree.push(menuItem)
      }
    }
  })


  return tree
}

const loadMenus = async () => {
  loading.value = true
  try {
    // 1. 获取系统菜单
    let systemMenus: any[] = []
    try {
      systemMenus = await getMenuTree() || []
    } catch (error) {
    }

    // 2. 获取插件菜单
    const pluginMenus = buildPluginMenuList()

    // 3. 清理系统菜单树中的空 children 数组
    const cleanEmptyChildren = (nodes: any[]) => {
      nodes.forEach(node => {
        if (node.children && Array.isArray(node.children)) {
          if (node.children.length === 0) {
            // 删除空的 children 数组
            delete node.children
            node.hasChildren = false
          } else {
            // 递归处理子节点
            cleanEmptyChildren(node.children)
            node.hasChildren = true
          }
        } else if (!node.children) {
          // 确保 hasChildren 设置为 false
          node.hasChildren = false
        }
      })
    }

    // 4. 清理系统菜单
    if (systemMenus && systemMenus.length > 0) {
      // 深拷贝以避免修改原始数据
      systemMenus = JSON.parse(JSON.stringify(systemMenus))
      cleanEmptyChildren(systemMenus)
    }

    // 5. 直接使用清理后的系统菜单树
    menuList.value = systemMenus || []

    // 6. 将插件菜单添加到树中
    if (pluginMenus.length > 0) {
      // 插件菜单需要根据 parentId 插入到正确的位置
      insertPluginMenus(menuList.value, pluginMenus)
    }

    // 7. 构建菜单树选项（仅包含系统菜单）
    menuTreeOptions.value = JSON.parse(JSON.stringify(systemMenus || []))
    menuTreeOptions.value.unshift({ ID: 0, name: '顶级菜单' })

    // 强制重新渲染表格
    tableKey.value++
  } finally {
    loading.value = false
  }
}

// 检查菜单code是否已存在于树中
const menuCodeExists = (tree: any[], code: string): boolean => {
  for (const menu of tree) {
    if (menu.code === code) {
      return true
    }
    if (menu.children && menu.children.length > 0) {
      if (menuCodeExists(menu.children, code)) {
        return true
      }
    }
  }
  return false
}

// 将插件菜单插入到系统菜单树中（去重：不添加已存在的菜单）
const insertPluginMenus = (tree: any[], pluginMenus: any[]) => {
  pluginMenus.forEach(pluginMenu => {
    // 检查该菜单是否已存在（通过code判断）
    if (menuCodeExists(tree, pluginMenu.code)) {
      return // 跳过已存在的菜单
    }

    const parentId = pluginMenu.parentId

    if (!parentId) {
      // 顶级插件菜单，直接添加到树根
      tree.push(pluginMenu)
    } else {
      // 查找父菜单并添加
      const parent = findMenuInTree(tree, parentId)
      if (parent) {
        if (!parent.children) {
          parent.children = []
        }
        parent.children.push(pluginMenu)
        // 设置 hasChildren 标记
        parent.hasChildren = true
      } else {
      }
    }
  })
}

// 在树中查找菜单
const findMenuInTree = (tree: any[], menuId: string | number): any => {
  for (const menu of tree) {
    const id = menu.ID || menu.id
    if (id === menuId) {
      return menu
    }
    if (menu.children && menu.children.length > 0) {
      const found = findMenuInTree(menu.children, menuId)
      if (found) return found
    }
  }
  return null
}

const handleAdd = () => {
  isEdit.value = false
  editingPluginMenu.value = false
  dialogTitle.value = '新增菜单'
  resetForm()
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true
  editingPluginMenu.value = false
  dialogTitle.value = '编辑菜单'
  menuForm.id = row.ID || row.id
  menuForm.name = row.name
  menuForm.code = row.code
  menuForm.type = row.type
  menuForm.parentId = row.parentId === 0 ? 0 : (row.parentId || 0)
  menuForm.path = row.path
  menuForm.component = row.component
  menuForm.icon = row.icon
  menuForm.sort = row.sort
  menuForm.visible = row.visible
  menuForm.status = row.status
  dialogVisible.value = true
}

// 处理插件菜单排序编辑
const handleEditPluginSort = (row: any) => {
  isEdit.value = true
  editingPluginMenu.value = true
  dialogTitle.value = '调整插件菜单排序'

  // 填充表单数据
  menuForm.id = row.ID || row.id
  menuForm.name = row.name
  menuForm.code = row.code
  menuForm.type = row.type
  menuForm.parentId = row.parentId || 0
  menuForm.path = row.path
  menuForm.component = row.component || ''
  menuForm.icon = row.icon
  menuForm.sort = row.sort
  menuForm.visible = row.visible
  menuForm.status = row.status

  dialogVisible.value = true
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm('确定要删除该菜单吗？', '提示', { type: 'warning' })
    await deleteMenu(row.ID || row.id)
    ElMessage.success('删除成功')
    loadMenus()
  } catch (error) {
    // 取消操作或错误已在 catch 块中处理
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        // 如果是编辑插件菜单，只保存排序到 localStorage
        if (editingPluginMenu.value) {
          const menuPath = menuForm.path
          const sort = menuForm.sort

          // 保存排序到 localStorage
          savePluginMenuSort(menuPath, sort)

          ElMessage.success(`插件菜单 "${menuForm.name}" 排序已更新`)
          dialogVisible.value = false
          resetForm()

          // 重新加载菜单以应用新的排序
          loadMenus()

          // 通知 Layout 刷新菜单
          window.dispatchEvent(new CustomEvent('plugins-changed'))
          return
        }

        // 系统菜单的正常处理流程
        const data = { ...menuForm }
        // 处理 parentId
        if (Array.isArray(data.parentId)) {
          const lastValue = data.parentId[data.parentId.length - 1]
          data.parentId = (lastValue !== null && lastValue !== undefined) ? lastValue : 0
        }
        // 确保parentId是数字类型
        data.parentId = Number(data.parentId)

        if (isEdit.value) {
          await updateMenu(menuForm.id, data)
          ElMessage.success('更新成功')
        } else {
          await createMenu(data)
          ElMessage.success('创建成功')
        }
        dialogVisible.value = false
        resetForm()
        loadMenus()
      } catch (error) {
        ElMessage.error('操作失败')
      } finally {
        submitting.value = false
      }
    }
  })
}

const resetForm = () => {
  editingPluginMenu.value = false
  Object.assign(menuForm, {
    id: 0,
    name: '',
    code: '',
    type: 2,
    parentId: 0,
    path: '',
    component: '',
    icon: '',
    sort: 0,
    visible: 1,
    status: 1
  })
  formRef.value?.clearValidate()
}

const handleDialogClose = () => {
  formRef.value?.resetFields()
  resetForm()
}

onMounted(() => {
  loadMenus()
})
</script>

<style scoped>
.menus-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #d4af37;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

/* 搜索栏 */
.search-bar {
  margin-bottom: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.search-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-input {
  width: 280px;
}

.search-actions {
  display: flex;
  gap: 10px;
}

.reset-btn {
  background: #f5f7fa;
  border-color: #dcdfe6;
  color: #606266;
}

.reset-btn:hover {
  background: #e6e8eb;
  border-color: #c0c4cc;
}

/* 搜索框样式 */
.search-bar :deep(.el-input__wrapper) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
  transition: all 0.3s ease;
  background-color: #fff;
}

.search-bar :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
}

.search-bar :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.25);
}

.search-icon {
  color: #d4af37;
}

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__body-wrapper) {
  border-radius: 0 0 12px 12px;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

/* 树形表格特定样式 */
.menu-tree-table :deep(.el-table__expand-icon) {
  color: #606266 !important;
  font-size: 16px !important;
  padding: 0 !important;
}

.menu-tree-table :deep(.el-table__expand-icon:hover) {
  color: #d4af37 !important;
}

.menu-tree-table :deep(.el-table__expand-icon--expanded) {
  transform: rotate(90deg);
}

/* 缩进元素 */
.menu-tree-table :deep(.el-table__indent) {
  display: inline-block !important;
  width: 30px !important;
}

/* 展开图标容器 */
.menu-tree-table :deep(.el-table__cell .el-table__expand-icon) {
  display: inline-block !important;
  margin-right: 4px !important;
}

/* 表头图标 */
.header-with-icon {
  display: flex;
  align-items: center;
  gap: 6px;
}

.header-icon {
  font-size: 16px;
}

.header-icon-gold {
  color: #d4af37;
}

/* 菜单类型标签 */
.menu-type-tag {
  border-radius: 6px;
  font-size: 12px;
  padding: 4px 10px;
  font-weight: 500;
}

.directory-tag {
  background-color: #ecf5ff;
  color: #409eff;
  border-color: #b3d8ff;
}

.menu-tag {
  background-color: #e8f5e9;
  color: #4caf50;
  border-color: #a5d6a7;
}

.button-tag {
  background-color: #fef0f0;
  color: #f56c6c;
  border-color: #fbc4c4;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
}

.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn :deep(.el-icon) {
  font-size: 16px;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-edit:hover {
  background-color: #e8f4ff;
  color: #409eff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

.action-sort:hover {
  background-color: #fff8e1;
  color: #e6a23c;
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #333333 !important;
}

/* 表单提示 */
.form-tip {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
  line-height: 1.5;
}

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.menu-edit-dialog) {
  border-radius: 12px;
}

:deep(.menu-edit-dialog .el-dialog__header) {
  padding: 20px 24px 16px;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.menu-edit-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.menu-edit-dialog .el-dialog__footer) {
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

/* 标签样式 */
:deep(.el-tag) {
  border-radius: 6px;
  padding: 4px 10px;
  font-weight: 500;
}

/* 输入框样式 */
:deep(.el-input__wrapper) {
  border-radius: 6px;
}

:deep(.el-select .el-input__wrapper) {
  border-radius: 6px;
}

:deep(.el-input-number) {
  width: 100%;
}

/* 响应式对话框 */
:deep(.responsive-dialog) {
  max-width: 1000px;
  min-width: 500px;
}

@media (max-width: 768px) {
  :deep(.responsive-dialog .el-dialog) {
    width: 95% !important;
    max-width: none;
    min-width: auto;
  }
}
</style>
