package asset

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/mom/internal/biz/asset"
	rbac "github.com/ydcloud-dy/mom/internal/biz/rbac"
	"github.com/ydcloud-dy/mom/internal/data"
	"github.com/ydcloud-dy/mom/pkg/response"
	"golang.org/x/crypto/ssh"
)

// NetworkDeviceService 网络设备服务
type NetworkDeviceService struct {
	deviceUseCase          *asset.NetworkDeviceUseCase
	assetPermissionRepo    rbac.AssetPermissionRepo
	assetAuthorizationRepo rbac.AssetAuthorizationRepo
	cache                  *data.Cache
}

// NewNetworkDeviceService 创建网络设备服务
func NewNetworkDeviceService(deviceUseCase *asset.NetworkDeviceUseCase) *NetworkDeviceService {
	return &NetworkDeviceService{
		deviceUseCase: deviceUseCase,
	}
}

// SetAssetPermissionRepo 设置资产权限仓库（旧版）
func (s *NetworkDeviceService) SetAssetPermissionRepo(repo rbac.AssetPermissionRepo) {
	s.assetPermissionRepo = repo
}

// SetAssetAuthorizationRepo 设置资产授权仓库（新版）
func (s *NetworkDeviceService) SetAssetAuthorizationRepo(repo rbac.AssetAuthorizationRepo) {
	s.assetAuthorizationRepo = repo
}

// SetCache 注入缓存
func (s *NetworkDeviceService) SetCache(cache *data.Cache) {
	s.cache = cache
}

// invalidateGroupTreeCache 网络设备增删改时失效分组树缓存
func (s *NetworkDeviceService) invalidateGroupTreeCache(ctx context.Context) {
	if s.cache != nil {
		s.cache.DelByPrefix(ctx, "asset:group_tree")
	}
}

// ListNetworkDevices 网络设备列表（根据用户权限过滤）
// @Summary 获取网络设备列表
// @Description 分页获取网络设备列表，根据用户权限过滤
// @Tags 资产管理-网络设备
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键字(名称/IP)"
// @Param deviceType query string false "设备类型"
// @Param protocol query string false "连接协议"
// @Param groupId query int false "资产分组ID"
// @Success 200 {object} response.Response{data=map[string]interface{}} "获取成功"
// @Router /api/v1/network-devices [get]
func (s *NetworkDeviceService) ListNetworkDevices(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")
	deviceType := c.Query("deviceType")
	protocol := c.Query("protocol")

	var groupID uint
	if gidStr := c.Query("groupId"); gidStr != "" {
		id, err := strconv.ParseUint(gidStr, 10, 32)
		if err == nil {
			groupID = uint(id)
		}
	}

	// 获取用户可访问的设备ID列表（非管理员权限过滤）
	var accessibleIDs []uint
	userID := c.GetUint("user_id")
	if userID > 0 {
		if s.assetAuthorizationRepo != nil {
			ids, err := s.assetAuthorizationRepo.GetUserAccessibleNetworkDeviceIDs(c.Request.Context(), userID)
			if err == nil {
				accessibleIDs = ids
			}
		} else if s.assetPermissionRepo != nil {
			ids, err := s.assetPermissionRepo.GetUserAccessibleNetworkDeviceIDs(c.Request.Context(), userID)
			if err == nil {
				accessibleIDs = ids
			}
		}
	}

	devices, total, err := s.deviceUseCase.ListFiltered(c.Request.Context(), page, pageSize, keyword, deviceType, protocol, groupID, accessibleIDs)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":     devices,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// CreateNetworkDevice 创建网络设备
// @Summary 创建网络设备
// @Description 创建一个新的网络设备
// @Tags 资产管理-网络设备
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body asset.NetworkDeviceRequest true "网络设备信息"
// @Success 200 {object} response.Response{data=asset.NetworkDevice} "创建成功"
// @Router /api/v1/network-devices [post]
func (s *NetworkDeviceService) CreateNetworkDevice(c *gin.Context) {
	var req asset.NetworkDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	device, err := s.deviceUseCase.Create(c.Request.Context(), &req)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	s.invalidateGroupTreeCache(c.Request.Context())
	response.Success(c, device)
}

// UpdateNetworkDevice 更新网络设备
// @Summary 更新网络设备
// @Description 更新指定的网络设备信息
// @Tags 资产管理-网络设备
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "设备ID"
// @Param body body asset.NetworkDeviceRequest true "网络设备信息"
// @Success 200 {object} response.Response "更新成功"
// @Router /api/v1/network-devices/{id} [put]
func (s *NetworkDeviceService) UpdateNetworkDevice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的设备ID")
		return
	}

	var req asset.NetworkDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	req.ID = uint(id)
	if err := s.deviceUseCase.Update(c.Request.Context(), &req); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	s.invalidateGroupTreeCache(c.Request.Context())
	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteNetworkDevice 删除网络设备
// @Summary 删除网络设备
// @Description 删除指定的网络设备
// @Tags 资产管理-网络设备
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "设备ID"
// @Success 200 {object} response.Response "删除成功"
// @Router /api/v1/network-devices/{id} [delete]
func (s *NetworkDeviceService) DeleteNetworkDevice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的设备ID")
		return
	}

	if err := s.deviceUseCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	s.invalidateGroupTreeCache(c.Request.Context())
	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetNetworkDevice 获取网络设备详情
// @Summary 获取网络设备详情
// @Description 根据ID获取单个网络设备详情
// @Tags 资产管理-网络设备
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "设备ID"
// @Success 200 {object} response.Response{data=asset.NetworkDevice} "获取成功"
// @Router /api/v1/network-devices/{id} [get]
func (s *NetworkDeviceService) GetNetworkDevice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的设备ID")
		return
	}

	device, err := s.deviceUseCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, device)
}

// GetAllNetworkDevices 获取所有网络设备（不分页，根据用户权限过滤）
// @Summary 获取所有网络设备
// @Description 获取用户有权限访问的所有网络设备列表，不分页
// @Tags 资产管理-网络设备
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=[]asset.NetworkDevice} "获取成功"
// @Router /api/v1/network-devices/all [get]
func (s *NetworkDeviceService) GetAllNetworkDevices(c *gin.Context) {
	userID := c.GetUint("user_id")

	// 获取用户可访问的设备ID列表
	var accessibleIDs []uint
	if userID > 0 {
		if s.assetAuthorizationRepo != nil {
			ids, err := s.assetAuthorizationRepo.GetUserAccessibleNetworkDeviceIDs(c.Request.Context(), userID)
			if err == nil {
				accessibleIDs = ids
			}
		} else if s.assetPermissionRepo != nil {
			ids, err := s.assetPermissionRepo.GetUserAccessibleNetworkDeviceIDs(c.Request.Context(), userID)
			if err == nil {
				accessibleIDs = ids
			}
		}
	}

	devices, err := s.deviceUseCase.GetAll(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	// 如果 accessibleIDs 不为 nil（非管理员），过滤设备
	if accessibleIDs != nil {
		idSet := make(map[uint]bool, len(accessibleIDs))
		for _, id := range accessibleIDs {
			idSet[id] = true
		}
		var filtered []*asset.NetworkDevice
		for _, d := range devices {
			if idSet[d.ID] {
				filtered = append(filtered, d)
			}
		}
		devices = filtered
	}

	response.Success(c, devices)
}

// TestNetworkDeviceConnection 测试网络设备连接
// @Summary 测试网络设备连接
// @Description 用指定的协议（SSH或Telnet）测试网络设备是否能成功连接
// @Tags 资产管理-网络设备
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "设备ID"
// @Success 200 {object} response.Response "连接成功"
// @Failure 500 {object} response.Response "连接失败"
// @Router /api/v1/network-devices/{id}/test [post]
func (s *NetworkDeviceService) TestNetworkDeviceConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的设备ID")
		return
	}

	device, credential, err := s.deviceUseCase.GetByIDForConnection(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, err.Error())
		return
	}

	addr := fmt.Sprintf("%s:%d", device.IP, device.Port)

	switch device.Protocol {
	case "telnet":
		// 测试Telnet连接
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			// 连接失败，更新状态为离线
			_ = s.deviceUseCase.UpdateStatus(c.Request.Context(), uint(id), 0)
			response.ErrorCode(c, http.StatusInternalServerError, "Telnet连接失败: "+err.Error())
			return
		}
		conn.Close()
		// 连接成功，更新状态为在线
		_ = s.deviceUseCase.UpdateStatus(c.Request.Context(), uint(id), 1)
		response.SuccessWithMessage(c, "Telnet连接成功", nil)

	case "ssh":
		if credential == nil {
			response.ErrorCode(c, http.StatusBadRequest, "未配置凭证")
			return
		}
		// 测试SSH连接
		var authMethods []ssh.AuthMethod
		switch credential.Type {
		case "password":
			authMethods = append(authMethods, ssh.Password(credential.Password))
		case "key", "private_key":
			var signer ssh.Signer
			if credential.Passphrase != "" {
				signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(credential.PrivateKey), []byte(credential.Passphrase))
			} else {
				signer, err = ssh.ParsePrivateKey([]byte(credential.PrivateKey))
			}
			if err != nil {
				response.ErrorCode(c, http.StatusInternalServerError, "解析私钥失败: "+err.Error())
				return
			}
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		default:
			response.ErrorCode(c, http.StatusBadRequest, "不支持的凭证类型: "+credential.Type)
			return
		}

		config := &ssh.ClientConfig{
			Config: ssh.Config{
				// 兼容旧设备的密钥交换算法
				KeyExchanges: []string{
					"curve25519-sha256",
					"curve25519-sha256@libssh.org",
					"ecdh-sha2-nistp256",
					"ecdh-sha2-nistp384",
					"ecdh-sha2-nistp521",
					"diffie-hellman-group14-sha256",
					"diffie-hellman-group14-sha1",
					"diffie-hellman-group1-sha1",
				},
				Ciphers: []string{
					"aes128-gcm@openssh.com",
					"aes256-gcm@openssh.com",
					"chacha20-poly1305@openssh.com",
					"aes128-ctr",
					"aes192-ctr",
					"aes256-ctr",
					"aes128-cbc",
					"aes192-cbc",
					"aes256-cbc",
					"3des-cbc",
				},
			},
			User:            credential.Username,
			Auth:            authMethods,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         10 * time.Second,
		}

		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			// 连接失败，更新状态为离线
			_ = s.deviceUseCase.UpdateStatus(c.Request.Context(), uint(id), 0)
			response.ErrorCode(c, http.StatusInternalServerError, "SSH连接失败: "+err.Error())
			return
		}
		client.Close()
		// 连接成功，更新状态为在线
		_ = s.deviceUseCase.UpdateStatus(c.Request.Context(), uint(id), 1)
		response.SuccessWithMessage(c, "SSH连接成功", nil)

	default:
		response.ErrorCode(c, http.StatusBadRequest, "不支持的协议: "+device.Protocol)
	}
}
