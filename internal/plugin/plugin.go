// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package plugin

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/mom/internal/biz/rbac"
	"gorm.io/gorm"
)

// Plugin 插件接口
// 所有插件必须实现该接口
type Plugin interface {
	// 插件名
	Name() string

	// 插件描述
	Description() string

	// 插件版本
	Version() string

	// 插件作者
	Author() string

	// 启用插件
	// Initialize plugin resources, database tables, etc.
	Enable(db *gorm.DB) error

	// 关闭插件
	// Clean up plugin resources (note: won't delete database tables by default)
	Disable(db *gorm.DB) error

	// 注册插件路由到系统路由
	// Plugin can register its API routes here
	RegisterRoutes(router *gin.RouterGroup, db *gorm.DB)

	// GetMenus Get plugin menu configuration
	// Return menu items to be added to the system
	GetMenus() []MenuConfig
}

// MenuConfig Menu configuration
type MenuConfig struct {
	// Menu name
	Name string `json:"name"`

	// Menu path (frontend route)
	Path string `json:"path"`

	// Icon name
	Icon string `json:"icon"`

	// Sort order (smaller number comes first)
	Sort int `json:"sort"`

	// Hidden or not
	Hidden bool `json:"hidden"`

	// Parent menu path (if this is a submenu)
	ParentPath string `json:"parentPath"`

	// Permission identifier (optional, for access control)
	Permission string `json:"permission"`
}

// PluginState 插件状态数据模型
type PluginState struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Name      string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Enabled   bool      `gorm:"default:false;not null" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (PluginState) TableName() string {
	return "plugin_states"
}

// Manager Plugin manager
type Manager struct {
	plugins map[string]Plugin
	db      *gorm.DB
}

// NewManager Create plugin manager
func NewManager(db *gorm.DB) *Manager {
	mgr := &Manager{
		plugins: make(map[string]Plugin),
		db:      db,
	}

	// 自动迁移插件状态表
	_ = db.AutoMigrate(&PluginState{})

	return mgr
}

// Register 注册插件
func (m *Manager) Register(plugin Plugin) error {
	name := plugin.Name()

	// Check if plugin already registered
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	// Register plugin
	m.plugins[name] = plugin

	// 初始化插件状态（如果不存在）
	var state PluginState
	if err := m.db.Where("name = ?", name).First(&state).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 插件状态不存在，创建新记录（默认禁用）
			state = PluginState{
				Name:    name,
				Enabled: false,
			}
			if err := m.db.Create(&state).Error; err != nil {
				return fmt.Errorf("failed to create plugin state: %w", err)
			}
		}
	}

	return nil
}

// Enable 启用插件
func (m *Manager) Enable(name string) error {
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Execute plugin Enable method
	if err := plugin.Enable(m.db); err != nil {
		return err
	}

	// 更新插件状态为已启用
	if err := m.db.Model(&PluginState{}).Where("name = ?", name).Update("enabled", true).Error; err != nil {
		return fmt.Errorf("failed to update plugin state: %w", err)
	}

	// 同步插件菜单到数据库
	if err := m.syncPluginMenus(plugin); err != nil {
		log.Printf("[plugin] 同步插件菜单失败 plugin=%s err=%v", name, err)
	}

	return nil
}

// Disable 禁用插件
func (m *Manager) Disable(name string) error {
	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Execute plugin Disable method
	if err := plugin.Disable(m.db); err != nil {
		return err
	}

	// 更新插件状态为已禁用
	if err := m.db.Model(&PluginState{}).Where("name = ?", name).Update("enabled", false).Error; err != nil {
		return fmt.Errorf("failed to update plugin state: %w", err)
	}

	// 删除插件菜单
	if err := m.removePluginMenus(name); err != nil {
		log.Printf("[plugin] 删除插件菜单失败 plugin=%s err=%v", name, err)
	}

	return nil
}

// IsEnabled 检查插件是否已启用
func (m *Manager) IsEnabled(name string) bool {
	var state PluginState
	if err := m.db.Where("name = ?", name).First(&state).Error; err != nil {
		return false
	}
	return state.Enabled
}

// GetPlugin Get plugin
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	plugin, exists := m.plugins[name]
	return plugin, exists
}

// GetAllPlugins Get all plugins
func (m *Manager) GetAllPlugins() []Plugin {
	plugins := make([]Plugin, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// RegisterAllRoutes Register all plugin routes
func (m *Manager) RegisterAllRoutes(router *gin.RouterGroup) {
	for _, plugin := range m.plugins {
		// 只有启用的插件才注册路由
		if m.IsEnabled(plugin.Name()) {
			// 直接将 router 传给插件，让插件自己决定路径前缀
			plugin.RegisterRoutes(router, m.db)
		}
	}
}

// GetAllMenus Get all plugin menu configurations
func (m *Manager) GetAllMenus() []MenuConfig {
	allMenus := make([]MenuConfig, 0)
	for _, plugin := range m.plugins {
		// 只有启用的插件才返回菜单
		if m.IsEnabled(plugin.Name()) {
			menus := plugin.GetMenus()
			allMenus = append(allMenus, menus...)
		}
	}
	return allMenus
}

// pathToCode 将路由路径转换为菜单编码（与前端 Menus.vue 中的 menu.path.replace(/\//g, '_') 逻辑一致）
func pathToCode(path string) string {
	return strings.ReplaceAll(path, "/", "_")
}

// syncPluginMenus 将插件菜单同步到 sys_menu 数据库表
func (m *Manager) syncPluginMenus(p Plugin) error {
	menus := p.GetMenus()
	if len(menus) == 0 {
		return nil
	}

	pluginName := p.Name()
	// path -> database ID 映射，用于子菜单关联父菜单
	pathToID := make(map[string]uint)

	// 第一轮：处理顶级菜单（parentPath 为空）
	for _, menu := range menus {
		if menu.ParentPath != "" {
			continue
		}
		code := pathToCode(menu.Path)
		visible := 1
		if menu.Hidden {
			visible = 0
		}

		var existing rbac.SysMenu
		err := m.db.Where("plugin_name = ? AND code = ?", pluginName, code).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			// 创建新菜单
			newMenu := &rbac.SysMenu{
				Name:       menu.Name,
				Code:       code,
				Type:       1, // 目录
				ParentID:   0,
				Path:       menu.Path,
				Icon:       menu.Icon,
				Sort:       menu.Sort,
				Visible:    visible,
				Status:     1,
				PluginName: pluginName,
			}
			if err := m.db.Create(newMenu).Error; err != nil {
				log.Printf("[plugin] 创建插件顶级菜单失败 code=%s err=%v", code, err)
				continue
			}
			pathToID[menu.Path] = newMenu.ID

			// 自动分配给 admin 角色
			m.assignMenuToAdmin(newMenu.ID)
		} else if err == nil {
			// 已存在，仅更新 name 和 path（不覆盖用户自定义的 visible/status/icon/sort）
			m.db.Model(&rbac.SysMenu{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
				"name": menu.Name,
				"path": menu.Path,
			})
			pathToID[menu.Path] = existing.ID
		}
	}

	// 第二轮：处理子菜单（parentPath 非空）
	for _, menu := range menus {
		if menu.ParentPath == "" {
			continue
		}
		code := pathToCode(menu.Path)
		visible := 1
		if menu.Hidden {
			visible = 0
		}

		// 查找父菜单 ID
		parentID, ok := pathToID[menu.ParentPath]
		if !ok {
			// 父菜单可能已经在数据库中（之前的启动创建的）
			parentCode := pathToCode(menu.ParentPath)
			var parentMenu rbac.SysMenu
			if err := m.db.Where("plugin_name = ? AND code = ?", pluginName, parentCode).First(&parentMenu).Error; err == nil {
				parentID = parentMenu.ID
				pathToID[menu.ParentPath] = parentID
			} else {
				log.Printf("[plugin] 找不到父菜单 parentPath=%s plugin=%s", menu.ParentPath, pluginName)
				continue
			}
		}

		var existing rbac.SysMenu
		err := m.db.Where("plugin_name = ? AND code = ?", pluginName, code).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			// 创建新子菜单
			newMenu := &rbac.SysMenu{
				Name:       menu.Name,
				Code:       code,
				Type:       2, // 菜单
				ParentID:   parentID,
				Path:       menu.Path,
				Icon:       menu.Icon,
				Sort:       menu.Sort,
				Visible:    visible,
				Status:     1,
				PluginName: pluginName,
			}
			if err := m.db.Create(newMenu).Error; err != nil {
				log.Printf("[plugin] 创建插件子菜单失败 code=%s err=%v", code, err)
				continue
			}
			pathToID[menu.Path] = newMenu.ID

			// 自动分配给 admin 角色
			m.assignMenuToAdmin(newMenu.ID)
		} else if err == nil {
			// 已存在，仅更新 name、path、parent_id（不覆盖用户自定义的 visible/status/icon/sort）
			m.db.Model(&rbac.SysMenu{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
				"name":      menu.Name,
				"path":      menu.Path,
				"parent_id": parentID,
			})
			pathToID[menu.Path] = existing.ID
		}
	}

	// 清理多余的菜单（例如插件更新后删除了某些菜单，或者路径变更导致旧菜单残留）
	var currentMenuIDs []uint
	for _, id := range pathToID {
		currentMenuIDs = append(currentMenuIDs, id)
	}

	if len(currentMenuIDs) > 0 {
		// 删除不在本次同步列表中的该插件菜单
		if err := m.db.Where("plugin_name = ? AND id NOT IN ?", pluginName, currentMenuIDs).Delete(&rbac.SysMenu{}).Error; err != nil {
			log.Printf("[plugin] 清理过期菜单失败 plugin=%s err=%v", pluginName, err)
		} else {
			// 同时清理角色关联
			// 注意：GORM Delete 软删除不会自动清理关联表，但这里因为是逻辑删除，关联表记录保留也无所谓，或者需要手动清理
			// 如果是硬删除（Unscoped）则需要手动清理关联
			// 这里我们保持软删除
		}
	}

	log.Printf("[plugin] 插件菜单同步完成 plugin=%s 菜单数=%d", pluginName, len(menus))
	return nil
}

// assignMenuToAdmin 将菜单分配给 admin 角色
func (m *Manager) assignMenuToAdmin(menuID uint) {
	// 查找 admin 角色
	var adminRole rbac.SysRole
	if err := m.db.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
		return
	}

	// 检查是否已存在关联
	var count int64
	m.db.Table("sys_role_menu").Where("role_id = ? AND menu_id = ?", adminRole.ID, menuID).Count(&count)
	if count > 0 {
		return
	}

	// 创建关联
	m.db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, menuID)
}

// removePluginMenus 删除指定插件的所有菜单
func (m *Manager) removePluginMenus(pluginName string) error {
	// 查找该插件的所有菜单 ID
	var menuIDs []uint
	if err := m.db.Model(&rbac.SysMenu{}).Where("plugin_name = ?", pluginName).Pluck("id", &menuIDs).Error; err != nil {
		return fmt.Errorf("查询插件菜单失败: %w", err)
	}

	if len(menuIDs) == 0 {
		return nil
	}

	// 删除角色-菜单关联
	if err := m.db.Exec("DELETE FROM sys_role_menu WHERE menu_id IN ?", menuIDs).Error; err != nil {
		log.Printf("[plugin] 删除角色菜单关联失败 plugin=%s err=%v", pluginName, err)
	}

	// 删除菜单记录（硬删除，不经过 GORM 的软删除）
	if err := m.db.Unscoped().Where("plugin_name = ?", pluginName).Delete(&rbac.SysMenu{}).Error; err != nil {
		return fmt.Errorf("删除插件菜单失败: %w", err)
	}

	log.Printf("[plugin] 插件菜单已删除 plugin=%s 删除数=%d", pluginName, len(menuIDs))
	return nil
}
