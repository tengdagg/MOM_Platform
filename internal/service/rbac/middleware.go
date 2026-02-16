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

package rbac

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/mom/internal/biz/rbac"
	"github.com/ydcloud-dy/mom/pkg/response"
)

const (
	UserIdKey   = "user_id"
	UsernameKey = "username"
)

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) uint {
	if userID, exists := c.Get(UserIdKey); exists {
		if id, ok := userID.(uint); ok {
			return id
		}
	}
	return 0
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) string {
	if username, exists := c.Get(UsernameKey); exists {
		if name, ok := username.(string); ok {
			return name
		}
	}
	return ""
}

// AuthMiddleware JWT认证中间件
type AuthMiddleware struct {
	authService            *AuthService
	assetPermissionRepo    rbac.AssetPermissionRepo
	assetAuthorizationRepo rbac.AssetAuthorizationRepo
	menuRepo               rbac.MenuRepo
}

func NewAuthMiddleware(authService *AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// SetAssetPermissionRepo 设置资产权限仓储（旧版，兼容）
func (m *AuthMiddleware) SetAssetPermissionRepo(repo rbac.AssetPermissionRepo) {
	m.assetPermissionRepo = repo
}

// SetAssetAuthorizationRepo 设置资产授权仓储（新版）
func (m *AuthMiddleware) SetAssetAuthorizationRepo(repo rbac.AssetAuthorizationRepo) {
	m.assetAuthorizationRepo = repo
}

// SetMenuRepo 设置菜单仓储
func (m *AuthMiddleware) SetMenuRepo(repo rbac.MenuRepo) {
	m.menuRepo = repo
}

// AuthRequired JWT认证
func (m *AuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// 优先从 Authorization header 获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		// 如果 header 中没有，尝试从 query 参数获取（用于 WebSocket 连接）
		if token == "" {
			token = c.Query("token")
		}

		// 如果都没有，返回未授权
		if token == "" {
			response.ErrorCode(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}

		claims, err := m.authService.ParseToken(token)
		if err != nil {
			response.ErrorCode(c, http.StatusUnauthorized, "token无效或已过期")
			c.Abort()
			return
		}

		c.Set(UserIdKey, claims.UserID)
		c.Set(UsernameKey, claims.Username)
		c.Next()
	}
}

// RequireAdmin 检查是否为管理员
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			response.ErrorCode(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}

		// 获取用户角色
		roles, err := m.authService.roleUseCase.GetByUserID(c.Request.Context(), userID)
		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "获取用户角色失败")
			c.Abort()
			return
		}

		// 检查是否有admin角色
		hasAdminRole := false
		for _, role := range roles {
			if role.Code == "admin" {
				hasAdminRole = true
				break
			}
		}

		if !hasAdminRole {
			response.ErrorCode(c, http.StatusForbidden, "无权限，请联系管理员操作")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) RequireMenuPermission(codes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			response.ErrorCode(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}

		// 获取用户的角色
		roles, err := m.authService.roleUseCase.GetByUserID(c.Request.Context(), userID)
		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "获取用户角色失败")
			c.Abort()
			return
		}

		// 超级管理员放行
		for _, role := range roles {
			if role.Code == "admin" {
				c.Next()
				return
			}
		}

		// 使用菜单仓储查询用户拥有的按钮权限
		if m.menuRepo == nil {
			// 如果未设置 menuRepo，暂时放行
			c.Next()
			return
		}

		btnCodes, err := m.menuRepo.GetButtonCodesByUserID(c.Request.Context(), userID)
		if err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "权限检查失败")
			c.Abort()
			return
		}

		// 转成 set 方便查找
		codeSet := make(map[string]struct{}, len(btnCodes))
		for _, code := range btnCodes {
			codeSet[code] = struct{}{}
		}

		// 检查用户是否拥有所需权限中的任意一个
		for _, requiredCode := range codes {
			if _, ok := codeSet[requiredCode]; ok {
				c.Next()
				return
			}
		}

		response.ErrorCode(c, http.StatusForbidden, "无权限，请联系管理员操作")
		c.Abort()
	}
}

// RequireNetworkDevicePermission 检查网络设备操作权限的中间件
func (m *AuthMiddleware) RequireNetworkDevicePermission(operation uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			response.ErrorCode(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}

		deviceIDStr := c.Param("id")
		if deviceIDStr == "" {
			c.Next()
			return
		}

		deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
		if err != nil {
			response.ErrorCode(c, http.StatusBadRequest, "无效的设备ID")
			c.Abort()
			return
		}

		// 优先使用新的授权仓储
		if m.assetAuthorizationRepo != nil {
			hasPermission, err := m.assetAuthorizationRepo.CheckNetworkDeviceOperationPermission(
				c.Request.Context(), userID, uint(deviceID), operation,
			)
			if err != nil {
				response.ErrorCode(c, http.StatusInternalServerError, "权限检查失败")
				c.Abort()
				return
			}
			if !hasPermission {
				response.ErrorCode(c, http.StatusForbidden, "无权限，请联系管理员操作")
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// 兼容旧版
		if m.assetPermissionRepo != nil {
			hasPermission, err := m.assetPermissionRepo.CheckNetworkDeviceOperationPermission(
				c.Request.Context(), userID, uint(deviceID), operation,
			)
			if err != nil {
				response.ErrorCode(c, http.StatusInternalServerError, "权限检查失败")
				c.Abort()
				return
			}
			if !hasPermission {
				response.ErrorCode(c, http.StatusForbidden, "无权限，请联系管理员操作")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// RequireHostPermission 检查主机操作权限的中间件
func (m *AuthMiddleware) RequireHostPermission(operation uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			response.ErrorCode(c, http.StatusUnauthorized, "未登录")
			c.Abort()
			return
		}

		hostIDStr := c.Param("id")
		if hostIDStr == "" {
			c.Next()
			return
		}

		hostID, err := strconv.ParseUint(hostIDStr, 10, 32)
		if err != nil {
			response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
			c.Abort()
			return
		}

		// 优先使用新的授权仓储
		if m.assetAuthorizationRepo != nil {
			hasPermission, err := m.assetAuthorizationRepo.CheckHostOperationPermission(
				c.Request.Context(), userID, uint(hostID), operation,
			)
			if err != nil {
				response.ErrorCode(c, http.StatusInternalServerError, "权限检查失败")
				c.Abort()
				return
			}
			if !hasPermission {
				response.ErrorCode(c, http.StatusForbidden, "无权限，请联系管理员操作")
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// 兼容旧版
		if m.assetPermissionRepo != nil {
			hasPermission, err := m.assetPermissionRepo.CheckHostOperationPermission(
				c.Request.Context(), userID, uint(hostID), operation,
			)
			if err != nil {
				response.ErrorCode(c, http.StatusInternalServerError, "权限检查失败")
				c.Abort()
				return
			}
			if !hasPermission {
				response.ErrorCode(c, http.StatusForbidden, "无权限，请联系管理员操作")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
