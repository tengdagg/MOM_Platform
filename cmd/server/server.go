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

package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ydcloud-dy/mom/cmd/root"
	"github.com/ydcloud-dy/mom/internal/biz"
	assetmodel "github.com/ydcloud-dy/mom/internal/biz/asset"
	auditmodel "github.com/ydcloud-dy/mom/internal/biz/audit"
	rbacmodel "github.com/ydcloud-dy/mom/internal/biz/rbac"
	"github.com/ydcloud-dy/mom/internal/conf"
	dataPkg "github.com/ydcloud-dy/mom/internal/data"
	rbacdata "github.com/ydcloud-dy/mom/internal/data/rbac"
	"github.com/ydcloud-dy/mom/internal/server"
	"github.com/ydcloud-dy/mom/internal/service"
	rbacservice "github.com/ydcloud-dy/mom/internal/service/rbac"
	appLogger "github.com/ydcloud-dy/mom/pkg/logger"
	"github.com/ydcloud-dy/mom/plugins/kubernetes/data/models"
	k8smodel "github.com/ydcloud-dy/mom/plugins/kubernetes/model"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 全局变量，用于在服务器生命周期内保持连接
var (
	globalData       *dataPkg.Data
	globalRedis      *dataPkg.Redis
	globalHTTPServer *server.HTTPServer
)

var Cmd = &cobra.Command{
	Use:   "server",
	Short: "启动服务",
	Long:  `启动 mom HTTP 服务器`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// 从命令行参数覆盖配置
		if mode := viper.GetString("mode"); mode != "" {
			viper.Set("server.mode", mode)
		}
		if logLevel := viper.GetString("log-level"); logLevel != "" {
			viper.Set("log.level", logLevel)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// 加载配置
		cfg, err := runServer()
		if err != nil {
			fmt.Printf("启动服务失败: %v\n", err)
			os.Exit(1)
		}

		// 等待中断信号
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		fmt.Println("\n正在关闭服务...")
		ctx := context.Background()
		if err := stopServer(ctx, cfg); err != nil {
			fmt.Printf("关闭服务失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("服务已关闭")
	},
}

func init() {
	root.Cmd.AddCommand(Cmd)
}

func runServer() (*conf.Config, error) {
	// 加载配置
	cfg, err := conf.Load(root.GetConfigFile())
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 初始化日志
	logCfg := &appLogger.Config{
		Level:      cfg.Log.Level,
		Filename:   cfg.Log.Filename,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
		Console:    cfg.Log.Console,
	}
	if err := appLogger.Init(logCfg); err != nil {
		return nil, fmt.Errorf("初始化日志失败: %w", err)
	}
	defer appLogger.Sync()

	appLogger.Info("服务启动中...",
		zap.String("version", "1.0.0"),
		zap.String("mode", cfg.Server.Mode),
	)

	// 初始化数据层
	data, err := dataPkg.NewData(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化数据层失败: %w", err)
	}
	globalData = data // 保存到全局变量，防止被垃圾回收

	// 初始化Redis
	redis, err := dataPkg.NewRedis(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化Redis失败: %w", err)
	}
	globalRedis = redis // 保存到全局变量

	// 初始化验证码存储（使用 Redis）
	rbacservice.InitCaptchaStore(redis.Get())

	// 初始化业务层
	biz := biz.NewBiz(data, redis)

	// 初始化服务层
	svc := service.NewService(biz)

	// 自动迁移数据库表
	if err := autoMigrate(data.DB()); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 初始化默认数据
	if err := initDefaultData(data.DB()); err != nil {
		return nil, fmt.Errorf("初始化默认数据失败: %w", err)
	}

	// 修复：确保个人信息菜单始终隐藏（解决历史数据问题）
	data.DB().Exec("UPDATE sys_menu SET visible = 0 WHERE code = 'profile'")

	// 创建缓存工具
	cache := dataPkg.NewCache(redis.Get())

	// 初始化HTTP服务器
	httpServer := server.NewHTTPServer(cfg, svc, data.DB(), cache)
	globalHTTPServer = httpServer // 保存到全局变量

	// 启动服务器
	go func() {
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("HTTP服务器启动失败", zap.Error(err))
		}
	}()

	// 打印启动信息
	printStartupInfo(cfg)

	return cfg, nil
}

// autoMigrate 自动迁移数据库表
func autoMigrate(db *gorm.DB) error {
	// 自动迁移表结构
	if err := db.AutoMigrate(
		&rbacmodel.SysUser{},
		&rbacmodel.SysRole{},
		&rbacmodel.SysDepartment{},
		&rbacmodel.SysMenu{},
		&rbacmodel.SysUserRole{},
		&rbacmodel.SysRoleMenu{},
		&rbacmodel.SysPosition{},
		&rbacmodel.SysUserPosition{},
		&rbacmodel.SysRoleAssetPermission{},
		&rbacmodel.SysAssetAuthorization{},
		&rbacmodel.SysLDAPConfig{},
		// Kubernetes 集群相关表
		&models.Cluster{},
		&k8smodel.UserKubeConfig{},
		&k8smodel.K8sUserRoleBinding{},
		// 审计日志相关表
		&auditmodel.SysOperationLog{},
		&auditmodel.SysLoginLog{},
		&auditmodel.SysDataLog{},
		// 资产管理相关表
		&assetmodel.AssetGroup{},
		&assetmodel.Host{},
		&assetmodel.Credential{},
		&assetmodel.CloudAccount{},
		&assetmodel.TerminalSession{},
		&assetmodel.NetworkDevice{},
		&assetmodel.SystemConfig{},
	); err != nil {
		return err
	}

	// 为用户表创建虚拟列和唯一索引
	// 问题：MySQL 唯一索引中多个 NULL 值被认为是不同的，无法正确约束
	// 解决：使用虚拟列 is_deleted (0=未删除, 1=已删除) 来创建唯一索引
	migrateUserUniqueIndex(db)

	// 修复 sys_menu 表的唯一索引问题
	// 移除 GORM 自动生成的单列唯一索引 (idx_sys_menu_code)，该索引会导致
	// 编辑菜单排序时出现 "菜单编码已存在" 的错误（因为与软删除记录冲突）
	dropIndexIfExists(db, "sys_menu", "idx_sys_menu_code")
	dropIndexIfExists(db, "sys_menu", "uk_code")

	// 扩展 sys_operation_log 列宽度以支持 AI 操作审计
	db.Exec("ALTER TABLE sys_operation_log MODIFY COLUMN `action` varchar(100) COMMENT '操作类型'")
	db.Exec("ALTER TABLE sys_operation_log MODIFY COLUMN `method` varchar(20) COMMENT '请求方法'")

	// 为 asset_group 和 credentials 表添加 category 列（用于区分主机/网络设备分组）
	if !columnExists(db, "asset_group", "category") {
		db.Exec("ALTER TABLE asset_group ADD COLUMN `category` varchar(20) DEFAULT 'all' COMMENT '分组类别 all:通用 host:主机 network:网络设备'")
	}
	if !columnExists(db, "credentials", "category") {
		db.Exec("ALTER TABLE credentials ADD COLUMN `category` varchar(20) DEFAULT 'all' COMMENT '凭证类别 all:通用 host:主机 network:网络设备'")
	}

	// 为 network_devices 表添加 brand 列（品牌和型号拆分）
	if !columnExists(db, "network_devices", "brand") {
		db.Exec("ALTER TABLE network_devices ADD COLUMN `brand` varchar(50) DEFAULT '' COMMENT '品牌' AFTER `ip`")
	}

	// 为 sys_role_asset_permission 表添加 asset_type 列（区分主机/网络设备权限）
	if !columnExists(db, "sys_role_asset_permission", "asset_type") {
		db.Exec("ALTER TABLE sys_role_asset_permission ADD COLUMN `asset_type` varchar(20) DEFAULT 'host' COMMENT '资产类型 host:主机 network_device:网络设备'")
	}

	// 自动添加网络设备菜单（如果不存在）
	var networkDeviceMenuCount int64
	db.Raw("SELECT COUNT(*) FROM sys_menu WHERE code = 'network-devices' AND deleted_at IS NULL").Scan(&networkDeviceMenuCount)
	if networkDeviceMenuCount == 0 {
		var assetMenuID uint
		db.Raw("SELECT id FROM sys_menu WHERE code = 'asset-management' AND deleted_at IS NULL LIMIT 1").Scan(&assetMenuID)
		if assetMenuID > 0 {
			db.Exec("INSERT INTO sys_menu (name, code, type, parent_id, path, component, icon, sort, visible, status, created_at, updated_at) VALUES ('网络设备', 'network-devices', 2, ?, '/asset/network-devices', 'asset/NetworkDevices', 'SetUp', 5, 1, 1, NOW(), NOW())", assetMenuID)
			// 给管理员角色授权
			var adminRoleID uint
			db.Raw("SELECT id FROM sys_role WHERE code = 'admin' AND deleted_at IS NULL LIMIT 1").Scan(&adminRoleID)
			var newMenuID uint
			db.Raw("SELECT id FROM sys_menu WHERE code = 'network-devices' AND deleted_at IS NULL LIMIT 1").Scan(&newMenuID)
			if adminRoleID > 0 && newMenuID > 0 {
				db.Exec("INSERT IGNORE INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRoleID, newMenuID)
			}
		}
	}

	// 清理已废弃的按钮级菜单（type=3），权限已改为资产级权限 + isAdmin 判断
	cleanupButtonMenus(db)

	// 更新菜单名称：权限配置 -> 资产授权
	db.Exec(`UPDATE sys_menu SET name = '资产授权' WHERE code = 'asset_permission' AND name = '权限配置' AND deleted_at IS NULL`)

	// 迁移旧版角色资产权限到新版用户/部门授权模型
	migrateAssetAuthorizations(db)

	return nil
}

// cleanupButtonMenus 清理废弃的按钮级菜单节点（type=3）
// 权限系统改为：资产级权限规则 + RequireAdmin 中间件，不再需要按钮级菜单
func cleanupButtonMenus(db *gorm.DB) {
	// 先删除 sys_role_menu 中关联的记录
	db.Exec(`DELETE FROM sys_role_menu WHERE menu_id IN (SELECT id FROM sys_menu WHERE type = 3 AND deleted_at IS NULL)`)
	// 再硬删除所有 type=3 的菜单记录
	db.Exec(`DELETE FROM sys_menu WHERE type = 3`)
}

// migrateAssetAuthorizations 将旧版角色资产权限迁移到新版用户/部门授权模型
func migrateAssetAuthorizations(db *gorm.DB) {
	// 检查新表是否已有数据，如果有则跳过迁移
	var count int64
	db.Table("sys_asset_authorization").Where("deleted_at IS NULL").Count(&count)
	if count > 0 {
		return
	}
	// 检查旧表是否有数据
	var oldCount int64
	db.Table("sys_role_asset_permission").Where("deleted_at IS NULL").Count(&oldCount)
	if oldCount == 0 {
		return
	}
	rbacdata.MigrateFromOldPermissions(db)
}

// columnExists 检查表中是否存在指定列
func columnExists(db *gorm.DB, table, column string) bool {
	var count int64
	db.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?", table, column).Scan(&count)
	return count > 0
}

// indexExists 检查表中是否存在指定索引
func indexExists(db *gorm.DB, table, index string) bool {
	var count int64
	db.Raw("SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?", table, index).Scan(&count)
	return count > 0
}

// dropIndexIfExists 安全地删除索引（存在才删除）
func dropIndexIfExists(db *gorm.DB, table, index string) {
	if indexExists(db, table, index) {
		if err := db.Exec("DROP INDEX " + index + " ON " + table).Error; err != nil {
			appLogger.Warn("删除索引失败", zap.String("table", table), zap.String("index", index), zap.Error(err))
		}
	}
}

// migrateUserUniqueIndex 迁移用户表虚拟列和唯一索引
func migrateUserUniqueIndex(db *gorm.DB) {
	// 1. 检查并添加虚拟列 is_deleted
	if !columnExists(db, "sys_user", "is_deleted") {
		// 使用 VIRTUAL 而非 STORED：MySQL 5.7+ 不支持通过 ALTER TABLE 添加 STORED 生成列
		// VIRTUAL 列在查询时动态计算，不占用磁盘空间，且支持二级索引
		if err := db.Exec("ALTER TABLE sys_user ADD COLUMN is_deleted TINYINT(1) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN 0 ELSE 1 END) VIRTUAL").Error; err != nil {
			appLogger.Warn("添加虚拟列失败", zap.Error(err))
			return // 虚拟列创建失败，后续索引也无法创建
		}
		appLogger.Info("成功添加虚拟列 is_deleted")
	}

	// 2. 删除旧的索引（安全检查后再删除）
	dropIndexIfExists(db, "sys_user", "idx_username_deleted_at")
	dropIndexIfExists(db, "sys_user", "idx_email_deleted_at")
	dropIndexIfExists(db, "sys_user", "idx_username_email_deleted_at")

	// 3. 创建新的唯一索引：用户名 + 邮箱 + is_deleted
	// 未删除的记录 (is_deleted=0) 中，username + email 组合必须唯一
	// 已删除的记录 (is_deleted=1) 不会阻止新记录创建
	if !indexExists(db, "sys_user", "idx_username_email_is_deleted") {
		if err := db.Exec("CREATE UNIQUE INDEX idx_username_email_is_deleted ON sys_user(username, email, is_deleted)").Error; err != nil {
			appLogger.Warn("创建用户名邮箱唯一索引失败", zap.Error(err))
		} else {
			appLogger.Info("成功创建用户名邮箱联合唯一索引")
		}
	}
}

// initDefaultData 初始化默认数据
func initDefaultData(db *gorm.DB) error {
	// 检查是否已有管理员用户
	var count int64
	db.Model(&rbacmodel.SysUser{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		return nil // 已存在管理员，无需初始化
	}

	appLogger.Info("开始初始化默认数据...")

	// 创建默认部门
	dept := &rbacmodel.SysDepartment{
		Name:     "总公司",
		Code:     "HQ",
		ParentID: 0,
		Sort:     0,
		Status:   1,
	}
	if err := db.Create(dept).Error; err != nil {
		return fmt.Errorf("创建默认部门失败: %w", err)
	}

	// 创建管理员角色
	adminRole := &rbacmodel.SysRole{
		Name:        "超级管理员",
		Code:        "admin",
		Description: "系统超级管理员，拥有所有权限",
		Sort:        0,
		Status:      1,
	}
	if err := db.Create(adminRole).Error; err != nil {
		return fmt.Errorf("创建管理员角色失败: %w", err)
	}

	// 创建管理员用户（密码：123456）
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("加密密码失败: %w", err)
	}

	adminUser := &rbacmodel.SysUser{
		Username:     "admin",
		Password:     string(hashedPassword),
		RealName:     "系统管理员",
		Email:        "admin@mom.com",
		Status:       1,
		DepartmentID: dept.ID,
	}
	if err := db.Create(adminUser).Error; err != nil {
		return fmt.Errorf("创建管理员用户失败: %w", err)
	}

	// 为管理员分配角色
	if err := db.Exec("INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)",
		adminUser.ID, adminRole.ID).Error; err != nil {
		return fmt.Errorf("分配管理员角色失败: %w", err)
	}

	// 创建默认菜单 - 先创建顶级菜单
	dashboardMenu := &rbacmodel.SysMenu{Name: "仪表盘", Code: "dashboard", Type: 2, ParentID: 0, Path: "/dashboard", Component: "Dashboard", Icon: "HomeFilled", Sort: 0, Visible: 1, Status: 1}
	if err := db.Create(dashboardMenu).Error; err != nil {
		return fmt.Errorf("创建首页菜单失败: %w", err)
	}
	db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, dashboardMenu.ID)

	// 创建系统管理顶级菜单（Path为空，作为目录使用）
	systemMenu := &rbacmodel.SysMenu{Name: "系统管理", Code: "system", Type: 1, ParentID: 0, Path: "", Icon: "Setting", Sort: 100, Visible: 1, Status: 1}
	if err := db.Create(systemMenu).Error; err != nil {
		return fmt.Errorf("创建系统管理菜单失败: %w", err)
	}
	db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, systemMenu.ID)

	// 创建系统管理子菜单（路径与前端路由一致）
	systemSubMenus := []*rbacmodel.SysMenu{
		{Name: "用户管理", Code: "users", Type: 2, ParentID: systemMenu.ID, Path: "/users", Component: "system/Users", Icon: "User", Sort: 1, Visible: 1, Status: 1},
		{Name: "角色管理", Code: "roles", Type: 2, ParentID: systemMenu.ID, Path: "/roles", Component: "system/Roles", Icon: "UserFilled", Sort: 2, Visible: 1, Status: 1},
		{Name: "菜单管理", Code: "menus", Type: 2, ParentID: systemMenu.ID, Path: "/menus", Component: "system/Menus", Icon: "Menu", Sort: 3, Visible: 1, Status: 1},
		{Name: "部门信息", Code: "dept-info", Type: 2, ParentID: systemMenu.ID, Path: "/dept-info", Component: "system/DeptInfo", Icon: "OfficeBuilding", Sort: 4, Visible: 1, Status: 1},
		{Name: "岗位信息", Code: "position-info", Type: 2, ParentID: systemMenu.ID, Path: "/position-info", Component: "system/PositionInfo", Icon: "Avatar", Sort: 5, Visible: 1, Status: 1},
		{Name: "系统配置", Code: "system-config", Type: 2, ParentID: systemMenu.ID, Path: "/system-config", Component: "system/SystemConfig", Icon: "Tools", Sort: 6, Visible: 1, Status: 1},
	}

	for _, menu := range systemSubMenus {
		if err := db.Create(menu).Error; err != nil {
			return fmt.Errorf("创建系统管理子菜单失败: %w", err)
		}
		db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, menu.ID)
	}

	// 创建操作审计顶级菜单
	auditMenu := &rbacmodel.SysMenu{Name: "操作审计", Code: "audit", Type: 1, ParentID: 0, Path: "/audit", Icon: "Document", Sort: 50, Visible: 1, Status: 1}
	if err := db.Create(auditMenu).Error; err != nil {
		return fmt.Errorf("创建操作审计菜单失败: %w", err)
	}
	db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, auditMenu.ID)

	// 创建操作审计子菜单（不包含数据日志）
	auditSubMenus := []*rbacmodel.SysMenu{
		{Name: "操作日志", Code: "operation-logs", Type: 2, ParentID: auditMenu.ID, Path: "/audit/operation-logs", Component: "audit/OperationLogs", Icon: "Document", Sort: 1, Visible: 1, Status: 1},
		{Name: "登录日志", Code: "login-logs", Type: 2, ParentID: auditMenu.ID, Path: "/audit/login-logs", Component: "audit/LoginLogs", Icon: "CircleCheck", Sort: 2, Visible: 1, Status: 1},
	}

	for _, menu := range auditSubMenus {
		if err := db.Create(menu).Error; err != nil {
			return fmt.Errorf("创建操作审计子菜单失败: %w", err)
		}
		db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, menu.ID)
	}

	// 创建资产管理顶级菜单
	assetMenu := &rbacmodel.SysMenu{Name: "资产管理", Code: "asset-management", Type: 1, ParentID: 0, Path: "/asset", Icon: "Coin", Sort: 1, Visible: 1, Status: 1}
	if err := db.Create(assetMenu).Error; err != nil {
		return fmt.Errorf("创建资产管理菜单失败: %w", err)
	}
	db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, assetMenu.ID)

	// 创建资产管理子菜单
	assetSubMenus := []*rbacmodel.SysMenu{
		{Name: "主机管理", Code: "host-management", Type: 2, ParentID: assetMenu.ID, Path: "/asset/hosts", Component: "asset/Hosts", Icon: "Monitor", Sort: 1, Visible: 1, Status: 1},
		{Name: "凭据管理", Code: "asset:credentials", Type: 2, ParentID: assetMenu.ID, Path: "/asset/credentials", Component: "asset/Credentials", Icon: "Lock", Sort: 2, Visible: 1, Status: 1},
		{Name: "业务分组", Code: "business-group", Type: 2, ParentID: assetMenu.ID, Path: "/asset/groups", Component: "asset/Groups", Icon: "Collection", Sort: 3, Visible: 1, Status: 1},
		{Name: "云账号管理", Code: "cloud-accounts", Type: 2, ParentID: assetMenu.ID, Path: "/asset/cloud-accounts", Component: "asset/CloudAccounts", Icon: "Cloudy", Sort: 4, Visible: 1, Status: 1},
		{Name: "网络设备", Code: "network-devices", Type: 2, ParentID: assetMenu.ID, Path: "/asset/network-devices", Component: "asset/NetworkDevices", Icon: "SetUp", Sort: 5, Visible: 1, Status: 1},
		{Name: "终端审计", Code: "asset_terminal_audit", Type: 2, ParentID: assetMenu.ID, Path: "/asset/terminal-audit", Component: "asset/TerminalAudit", Icon: "View", Sort: 6, Visible: 1, Status: 1},
		{Name: "资产授权", Code: "asset_permission", Type: 2, ParentID: assetMenu.ID, Path: "/asset/permissions", Component: "asset/AssetPermission", Icon: "Lock", Sort: 6, Visible: 1, Status: 1},
	}

	for _, menu := range assetSubMenus {
		if err := db.Create(menu).Error; err != nil {
			return fmt.Errorf("创建资产管理子菜单失败: %w", err)
		}
		db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, menu.ID)
	}

	// 创建插件管理顶级菜单
	pluginMenu := &rbacmodel.SysMenu{Name: "插件管理", Code: "plugin", Type: 1, ParentID: 0, Path: "/plugin", Icon: "Grid", Sort: 80, Visible: 1, Status: 1}
	if err := db.Create(pluginMenu).Error; err != nil {
		return fmt.Errorf("创建插件管理菜单失败: %w", err)
	}
	db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, pluginMenu.ID)

	// 创建插件管理子菜单
	pluginSubMenus := []*rbacmodel.SysMenu{
		{Name: "插件列表", Code: "plugin-list", Type: 2, ParentID: pluginMenu.ID, Path: "/plugin/list", Component: "plugin/PluginList", Icon: "Grid", Sort: 1, Visible: 1, Status: 1},
		{Name: "插件安装", Code: "plugin-install", Type: 2, ParentID: pluginMenu.ID, Path: "/plugin/install", Component: "plugin/PluginInstall", Icon: "Upload", Sort: 2, Visible: 1, Status: 1},
	}

	for _, menu := range pluginSubMenus {
		if err := db.Create(menu).Error; err != nil {
			return fmt.Errorf("创建插件管理子菜单失败: %w", err)
		}
		db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, menu.ID)
	}

	// 创建个人信息菜单（隐藏菜单）
	profileMenu := &rbacmodel.SysMenu{Name: "个人信息", Code: "profile", Type: 2, ParentID: 0, Path: "/profile", Component: "Profile", Icon: "UserFilled", Sort: 100, Visible: 0, Status: 1}
	if err := db.Create(profileMenu).Error; err != nil {
		return fmt.Errorf("创建个人信息菜单失败: %w", err)
	}
	db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", adminRole.ID, profileMenu.ID)

	appLogger.Info("默认数据初始化完成")
	appLogger.Info("默认管理员账号: admin")
	appLogger.Info("默认管理员密码: 123456")

	// 初始化系统配置
	var logConfigCount int64
	db.Model(&assetmodel.SystemConfig{}).Where("config_key = ?", "logRetentionDays").Count(&logConfigCount)
	if logConfigCount == 0 {
		if err := db.Create(&assetmodel.SystemConfig{
			ConfigKey: "logRetentionDays",
			Value:     "30",
			Remark:    "日志保留天数",
		}).Error; err != nil {
			appLogger.Warn("初始化日志保留天数配置失败", zap.Error(err))
		} else {
			appLogger.Info("初始化日志保留天数配置成功")
		}
	}

	return nil
}

func stopServer(ctx context.Context, cfg *conf.Config) error {
	appLogger.Info("服务正在关闭...")

	// 停止HTTP服务器
	if globalHTTPServer != nil {
		if err := globalHTTPServer.Stop(ctx); err != nil {
			appLogger.Error("停止HTTP服务器失败", zap.Error(err))
		}
	}

	// 关闭数据库连接
	if globalData != nil {
		if err := globalData.Close(); err != nil {
			appLogger.Error("关闭数据库连接失败", zap.Error(err))
		}
	}

	// 关闭Redis连接
	if globalRedis != nil {
		if err := globalRedis.Close(); err != nil {
			appLogger.Error("关闭Redis连接失败", zap.Error(err))
		}
	}

	return nil
}

func printStartupInfo(cfg *conf.Config) {
	listenAddr := fmt.Sprintf("%s:%d", "0.0.0.0", cfg.Server.HttpPort)
	displayAddr := fmt.Sprintf("%s:%d", "127.0.0.1", cfg.Server.HttpPort)

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("       mom 运维管理平台启动成功")
	fmt.Println("========================================")
	fmt.Printf("版本:     1.0.0\n")
	fmt.Printf("模式:     %s\n", cfg.Server.Mode)
	fmt.Printf("监听地址: http://%s\n", listenAddr)
	fmt.Printf("健康检查: http://%s/health\n", displayAddr)
	fmt.Printf("API文档:  http://%s/swagger/index.html\n", displayAddr)
	fmt.Println("========================================")
	fmt.Println()
}
