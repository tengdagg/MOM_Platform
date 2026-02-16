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

package asset

import (
	"github.com/gin-gonic/gin"
	assetbiz "github.com/ydcloud-dy/mom/internal/biz/asset"
	rbacbiz "github.com/ydcloud-dy/mom/internal/biz/rbac"
	assetdata "github.com/ydcloud-dy/mom/internal/data/asset"
	rbacdata "github.com/ydcloud-dy/mom/internal/data/rbac"
	assetService "github.com/ydcloud-dy/mom/internal/service/asset"
	rbacService "github.com/ydcloud-dy/mom/internal/service/rbac"
	"gorm.io/gorm"
)

type HTTPServer struct {
	assetGroupService       *assetService.AssetGroupService
	hostService             *assetService.HostService
	networkDeviceService    *assetService.NetworkDeviceService
	terminalManager         *TerminalManager
	networkTerminalManager  *NetworkTerminalManager
	terminalAuditHandler    *TerminalAuditHandler
	systemConfigHandler     *SystemConfigHandler
	authMiddleware          *rbacService.AuthMiddleware
}

func NewHTTPServer(
	assetGroupService *assetService.AssetGroupService,
	hostService *assetService.HostService,
	networkDeviceService *assetService.NetworkDeviceService,
	terminalManager *TerminalManager,
	networkTerminalManager *NetworkTerminalManager,
	db *gorm.DB,
	authMiddleware *rbacService.AuthMiddleware,
) *HTTPServer {
	return &HTTPServer{
		assetGroupService:      assetGroupService,
		hostService:            hostService,
		networkDeviceService:   networkDeviceService,
		terminalManager:        terminalManager,
		networkTerminalManager: networkTerminalManager,
		terminalAuditHandler:   NewTerminalAuditHandler(db),
		systemConfigHandler:    NewSystemConfigHandler(db),
		authMiddleware:         authMiddleware,
	}
}

func (s *HTTPServer) RegisterRoutes(r *gin.RouterGroup) {
	// 资产分组管理
	groups := r.Group("/asset-groups")
	{
		groups.GET("/tree", s.assetGroupService.GetGroupTree)
		groups.GET("/parent-options", s.assetGroupService.GetParentOptions)
		groups.POST("", s.authMiddleware.RequireAdmin(), s.assetGroupService.CreateGroup)
		groups.GET("/:id", s.assetGroupService.GetGroup)
		groups.PUT("/:id", s.authMiddleware.RequireAdmin(), s.assetGroupService.UpdateGroup)
		groups.DELETE("/:id", s.authMiddleware.RequireAdmin(), s.assetGroupService.DeleteGroup)
	}

	// 主机管理
	hosts := r.Group("/hosts")
	{
		hosts.GET("", s.hostService.ListHosts)
		hosts.GET("/template/download", s.hostService.DownloadExcelTemplate)
		// 创建类操作仅管理员（无 :id，资产级权限无法校验）
		hosts.POST("/import", s.authMiddleware.RequireAdmin(), s.hostService.ImportFromExcel)
		hosts.POST("/batch-collect", s.authMiddleware.RequireAdmin(), s.hostService.BatchCollectHostInfo)
		hosts.POST("/batch-delete", s.authMiddleware.RequireAdmin(), s.hostService.BatchDeleteHosts)

		// 查看权限 - 查看主机详情
		hosts.GET("/:id",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionView),
			s.hostService.GetHost)

		// 创建主机 — 仅管理员
		hosts.POST("",
			s.authMiddleware.RequireAdmin(),
			s.hostService.CreateHost)
		// 编辑主机 — 资产级权限
		hosts.PUT("/:id",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionEdit),
			s.hostService.UpdateHost)

		// 删除权限 - 删除主机
		hosts.DELETE("/:id",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionDelete),
			s.hostService.DeleteHost)

		// 采集权限 - 采集主机信息
		hosts.POST("/:id/collect",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionCollect),
			s.hostService.CollectHostInfo)
		hosts.POST("/:id/test", s.hostService.TestHostConnection)

		// 文件管理权限 - 文件上传、下载、删除
		hosts.GET("/:id/files",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionFile),
			s.hostService.ListHostFiles)
		hosts.POST("/:id/files/upload",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionFile),
			s.hostService.UploadHostFile)
		hosts.GET("/:id/files/download",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionFile),
			s.hostService.DownloadHostFile)
		hosts.DELETE("/:id/files",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionFile),
			s.hostService.DeleteHostFile)
	}

	// 凭证管理（仅管理员可增删改）
	credentials := r.Group("/credentials")
	{
		credentials.GET("", s.hostService.ListCredentials)
		credentials.GET("/all", s.hostService.GetAllCredentials)
		credentials.GET("/:id", s.hostService.GetCredential)
		credentials.POST("", s.authMiddleware.RequireAdmin(), s.hostService.CreateCredential)
		credentials.PUT("/:id", s.authMiddleware.RequireAdmin(), s.hostService.UpdateCredential)
		credentials.DELETE("/:id", s.authMiddleware.RequireAdmin(), s.hostService.DeleteCredential)
	}

	// 云平台账号管理（仅管理员可增删改）
	cloudAccounts := r.Group("/cloud-accounts")
	{
		cloudAccounts.GET("", s.hostService.ListCloudAccounts)
		cloudAccounts.GET("/all", s.hostService.GetAllCloudAccounts)
		cloudAccounts.GET("/:id", s.hostService.GetCloudAccount)
		cloudAccounts.GET("/:id/regions", s.hostService.GetCloudRegions)
		cloudAccounts.GET("/:id/instances", s.hostService.GetCloudInstances)
		cloudAccounts.POST("", s.authMiddleware.RequireAdmin(), s.hostService.CreateCloudAccount)
		cloudAccounts.PUT("/:id", s.authMiddleware.RequireAdmin(), s.hostService.UpdateCloudAccount)
		cloudAccounts.DELETE("/:id", s.authMiddleware.RequireAdmin(), s.hostService.DeleteCloudAccount)
		cloudAccounts.POST("/import", s.authMiddleware.RequireAdmin(), s.hostService.ImportFromCloud)
	}

	// 网络设备管理（增删改用资产级权限校验，创建仅管理员）
	networkDevices := r.Group("/network-devices")
	{
		networkDevices.GET("", s.networkDeviceService.ListNetworkDevices)
		networkDevices.GET("/all", s.networkDeviceService.GetAllNetworkDevices)
		networkDevices.POST("", s.authMiddleware.RequireAdmin(), s.networkDeviceService.CreateNetworkDevice)
		networkDevices.GET("/:id", s.networkDeviceService.GetNetworkDevice)
		networkDevices.PUT("/:id",
			s.authMiddleware.RequireNetworkDevicePermission(rbacbiz.PermissionEdit),
			s.networkDeviceService.UpdateNetworkDevice)
		networkDevices.DELETE("/:id",
			s.authMiddleware.RequireNetworkDevicePermission(rbacbiz.PermissionDelete),
			s.networkDeviceService.DeleteNetworkDevice)
		networkDevices.POST("/:id/test", s.networkDeviceService.TestNetworkDeviceConnection)
	}

	// SSH终端 - 终端权限
	terminal := r.Group("/asset/terminal")
	{
		terminal.GET("/:id",
			s.authMiddleware.RequireHostPermission(rbacbiz.PermissionTerminal),
			s.HandleSSHConnection)
		terminal.POST("/:id/resize", s.ResizeTerminal)
	}

	// 网络设备终端 — 资产级终端权限
	networkTerminal := r.Group("/asset/network-terminal")
	{
		networkTerminal.GET("/:id",
			s.authMiddleware.RequireNetworkDevicePermission(rbacbiz.PermissionTerminal),
			s.networkTerminalManager.HandleNetworkTerminalConnection)
	}

	// 终端审计（仅管理员可回放、删除、配置保留期、手动清理）
	terminalSessions := r.Group("/terminal-sessions")
	{
		terminalSessions.GET("", s.terminalAuditHandler.ListTerminalSessions)
		terminalSessions.GET("/:id/play", s.authMiddleware.RequireAdmin(), s.terminalAuditHandler.PlayTerminalSession)
		terminalSessions.DELETE("/:id", s.authMiddleware.RequireAdmin(), s.terminalAuditHandler.DeleteTerminalSession)
		terminalSessions.GET("/retention", s.authMiddleware.RequireAdmin(), s.terminalAuditHandler.GetRetentionConfig)
		terminalSessions.PUT("/retention", s.authMiddleware.RequireAdmin(), s.terminalAuditHandler.UpdateRetentionConfig)
		terminalSessions.POST("/cleanup", s.authMiddleware.RequireAdmin(), s.terminalAuditHandler.CleanupExpiredSessions)
	}

	// 系统配置
	sysConfig := r.Group("/system-config")
	{
		sysConfig.GET("", s.systemConfigHandler.GetAllConfig)
		sysConfig.PUT("", s.systemConfigHandler.SaveAllConfig)
	}

	// 启动终端审计定时清理任务
	s.terminalAuditHandler.StartCleanupScheduler()

	// 启动审计日志定时清理任务
	s.systemConfigHandler.StartAuditLogCleanupScheduler()
}

// NewAssetServices 创建asset相关的服务
func NewAssetServices(db *gorm.DB) (
	*assetService.AssetGroupService,
	*assetService.HostService,
	*assetService.NetworkDeviceService,
	*TerminalManager,
	*NetworkTerminalManager,
) {
	// 初始化Repository
	assetGroupRepo := assetdata.NewAssetGroupRepo(db)
	hostRepo := assetdata.NewHostRepo(db)
	credentialRepo := assetdata.NewCredentialRepo(db)
	cloudAccountRepo := assetdata.NewCloudAccountRepo(db)
	assetPermissionRepo := rbacdata.NewAssetPermissionRepo(db)
	assetAuthorizationRepo := rbacdata.NewAssetAuthorizationRepo(db)
	networkDeviceRepo := assetdata.NewNetworkDeviceRepo(db)

	// 初始化UseCase
	assetGroupUseCase := assetbiz.NewAssetGroupUseCase(assetGroupRepo)
	credentialUseCase := assetbiz.NewCredentialUseCase(credentialRepo, hostRepo, networkDeviceRepo)
	cloudAccountUseCase := assetbiz.NewCloudAccountUseCase(cloudAccountRepo)
	hostUseCase := assetbiz.NewHostUseCase(hostRepo, credentialRepo, assetGroupRepo, cloudAccountRepo)
	assetPermissionUseCase := rbacbiz.NewAssetPermissionUseCase(assetPermissionRepo)
	assetAuthorizationUseCase := rbacbiz.NewAssetAuthorizationUseCase(assetAuthorizationRepo)
	networkDeviceUseCase := assetbiz.NewNetworkDeviceUseCase(networkDeviceRepo, credentialRepo, assetGroupRepo)

	// 初始化Service
	assetGroupService := assetService.NewAssetGroupService(assetGroupUseCase)
	assetGroupService.SetAssetAuthorizationRepo(assetAuthorizationRepo)
	assetGroupService.SetDB(db)
	hostService := assetService.NewHostService(hostUseCase, credentialUseCase, cloudAccountUseCase, assetPermissionUseCase)
	hostService.SetAssetAuthorizationUseCase(assetAuthorizationUseCase)
	networkDeviceService := assetService.NewNetworkDeviceService(networkDeviceUseCase)

	// 初始化TerminalManager
	terminalManager := NewTerminalManager(hostUseCase, db)
	networkTerminalManager := NewNetworkTerminalManager(networkDeviceUseCase, db)

	return assetGroupService, hostService, networkDeviceService, terminalManager, networkTerminalManager
}
