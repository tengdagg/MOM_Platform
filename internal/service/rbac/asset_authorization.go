package rbac

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/mom/internal/biz/rbac"
	"github.com/ydcloud-dy/mom/internal/data"
	"github.com/ydcloud-dy/mom/pkg/response"
	"gorm.io/gorm"
)

// AssetAuthorizationService 资产授权服务
type AssetAuthorizationService struct {
	useCase *rbac.AssetAuthorizationUseCase
	db      *gorm.DB
	cache   *data.Cache
}

func NewAssetAuthorizationService(useCase *rbac.AssetAuthorizationUseCase) *AssetAuthorizationService {
	return &AssetAuthorizationService{useCase: useCase}
}

// SetDB 设置数据库连接（用于资产树查询）
func (s *AssetAuthorizationService) SetDB(db *gorm.DB) {
	s.db = db
}

// SetCache 注入缓存
func (s *AssetAuthorizationService) SetCache(cache *data.Cache) {
	s.cache = cache
}

// invalidateAuthorizationCache 授权规则变更时，失效所有用户的权限缓存
func (s *AssetAuthorizationService) invalidateAuthorizationCache(ctx context.Context) {
	if s.cache == nil {
		return
	}
	s.cache.DelByPrefix(ctx, "user:")
}

// AssetTreeNode 资产树节点
type AssetTreeNode struct {
	ID       interface{}      `json:"id"`       // 分组用 uint, 资产用 "host_1" / "device_1"
	Label    string           `json:"label"`
	NodeType string           `json:"nodeType"` // root / group / host / device
	IP       string           `json:"ip,omitempty"`
	GroupID  uint             `json:"groupId,omitempty"`
	Children []*AssetTreeNode `json:"children,omitempty"`
}

// GetAssetTree 获取资产树（分组+具体资产）
func (s *AssetAuthorizationService) GetAssetTree(c *gin.Context) {
	if s.db == nil {
		response.ErrorCode(c, http.StatusInternalServerError, "服务未初始化")
		return
	}

	// 查询所有分组（host + network 两类）
	type GroupRow struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		ParentID uint   `json:"parentId"`
		Category string `json:"category"`
	}
	var groups []GroupRow
	s.db.Table("asset_group").
		Select("id, name, parent_id, COALESCE(category, 'all') as category").
		Where("deleted_at IS NULL").
		Order("parent_id, id").
		Find(&groups)

	// 查询所有主机
	type AssetRow struct {
		ID      uint   `json:"id"`
		Name    string `json:"name"`
		IP      string `json:"ip"`
		GroupID uint   `json:"groupId"`
	}
	var hosts []AssetRow
	s.db.Table("hosts").
		Select("id, name, ip, group_id").
		Where("deleted_at IS NULL").
		Find(&hosts)

	// 查询所有网络设备
	var devices []AssetRow
	s.db.Table("network_devices").
		Select("id, name, ip, group_id").
		Where("deleted_at IS NULL").
		Find(&devices)

	// 构建分组映射
	groupMap := make(map[uint]*AssetTreeNode)
	var rootChildren []*AssetTreeNode

	for _, g := range groups {
		node := &AssetTreeNode{
			ID:       g.ID,
			Label:    g.Name,
			NodeType: "group",
			GroupID:  g.ParentID,
			Children: []*AssetTreeNode{},
		}
		groupMap[g.ID] = node
	}

	// 构建层级
	for _, g := range groups {
		node := groupMap[g.ID]
		if g.ParentID > 0 {
			if parent, ok := groupMap[g.ParentID]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		rootChildren = append(rootChildren, node)
	}

	// 把主机挂到对应分组下
	for _, h := range hosts {
		assetNode := &AssetTreeNode{
			ID:       "host_" + strconv.FormatUint(uint64(h.ID), 10),
			Label:    h.Name,
			NodeType: "host",
			IP:       h.IP,
			GroupID:  h.GroupID,
		}
		if parent, ok := groupMap[h.GroupID]; ok {
			parent.Children = append(parent.Children, assetNode)
		} else {
			rootChildren = append(rootChildren, assetNode)
		}
	}

	// 把网络设备挂到对应分组下
	for _, d := range devices {
		assetNode := &AssetTreeNode{
			ID:       "device_" + strconv.FormatUint(uint64(d.ID), 10),
			Label:    d.Name,
			NodeType: "device",
			IP:       d.IP,
			GroupID:  d.GroupID,
		}
		if parent, ok := groupMap[d.GroupID]; ok {
			parent.Children = append(parent.Children, assetNode)
		} else {
			rootChildren = append(rootChildren, assetNode)
		}
	}

	// 根节点
	root := &AssetTreeNode{
		ID:       0,
		Label:    "全部",
		NodeType: "root",
		Children: rootChildren,
	}

	response.Success(c, []*AssetTreeNode{root})
}

// CreateAssetAuthorization 创建授权规则
func (s *AssetAuthorizationService) CreateAssetAuthorization(c *gin.Context) {
	var req rbac.AssetAuthorizationCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if len(req.UserIDs) == 0 && len(req.DepartmentIDs) == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "请至少选择一个用户或部门")
		return
	}

	if req.Permissions == 0 {
		req.Permissions = rbac.PermissionView
	}
	assetType := req.AssetType
	if assetType == "" {
		assetType = "host"
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	auth := &rbac.SysAssetAuthorization{
		Name:          req.Name,
		UserIDs:       req.UserIDs,
		DepartmentIDs: req.DepartmentIDs,
		AssetGroupID:  req.AssetGroupID,
		AssetType:     assetType,
		AssetIDs:      req.AssetIDs,
		Permissions:   req.Permissions,
		StartDate:     req.StartDate,
		ExpireDate:    req.ExpireDate,
		IsActive:      isActive,
		Description:   req.Description,
	}

	if err := s.useCase.Create(c.Request.Context(), auth); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	s.invalidateAuthorizationCache(c.Request.Context())
	response.SuccessWithMessage(c, "创建成功", auth)
}

// UpdateAssetAuthorization 更新授权规则
func (s *AssetAuthorizationService) UpdateAssetAuthorization(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	// 先获取旧记录
	existing, err := s.useCase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "授权规则不存在")
		return
	}

	var req rbac.AssetAuthorizationUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 更新字段
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.UserIDs != nil {
		existing.UserIDs = req.UserIDs
	}
	if req.DepartmentIDs != nil {
		existing.DepartmentIDs = req.DepartmentIDs
	}
	if req.AssetGroupID > 0 {
		existing.AssetGroupID = req.AssetGroupID
	}
	if req.AssetType != "" {
		existing.AssetType = req.AssetType
	}
	if req.AssetIDs != nil {
		existing.AssetIDs = req.AssetIDs
	}
	if req.Permissions > 0 {
		existing.Permissions = req.Permissions
	}
	if req.StartDate != nil {
		existing.StartDate = req.StartDate
	}
	if req.ExpireDate != nil {
		existing.ExpireDate = req.ExpireDate
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if req.Description != "" {
		existing.Description = req.Description
	}

	if err := s.useCase.Update(c.Request.Context(), existing); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	s.invalidateAuthorizationCache(c.Request.Context())
	response.SuccessWithMessage(c, "更新成功", nil)
}

// DeleteAssetAuthorization 删除授权规则
func (s *AssetAuthorizationService) DeleteAssetAuthorization(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := s.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	s.invalidateAuthorizationCache(c.Request.Context())
	response.SuccessWithMessage(c, "删除成功", nil)
}

// GetAssetAuthorizationDetail 获取授权规则详情
func (s *AssetAuthorizationService) GetAssetAuthorizationDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的ID")
		return
	}

	detail, err := s.useCase.GetDetailByID(c.Request.Context(), uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, detail)
}

// ListAssetAuthorizations 授权规则列表
func (s *AssetAuthorizationService) ListAssetAuthorizations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")

	var assetGroupID *uint
	if gidStr := c.Query("assetGroupId"); gidStr != "" {
		gid, err := strconv.ParseUint(gidStr, 10, 32)
		if err == nil {
			gidVal := uint(gid)
			assetGroupID = &gidVal
		}
	}

	list, total, err := s.useCase.List(c.Request.Context(), page, pageSize, assetGroupID, keyword)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"total": total,
		"list":  list,
	})
}

// GetUserHostPermissions 获取当前用户对指定主机的操作权限
func (s *AssetAuthorizationService) GetUserHostPermissions(c *gin.Context) {
	hostIDStr := c.Query("hostId")
	if hostIDStr == "" {
		response.ErrorCode(c, http.StatusBadRequest, "主机ID不能为空")
		return
	}
	hostID, err := strconv.ParseUint(hostIDStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的主机ID")
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		response.ErrorCode(c, http.StatusUnauthorized, "未授权")
		return
	}

	permissions, err := s.useCase.GetUserHostPermissions(c.Request.Context(), userID, uint(hostID))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"permissions": permissions,
	})
}

// GetUserDevicePermissions 获取当前用户对指定网络设备的操作权限
func (s *AssetAuthorizationService) GetUserDevicePermissions(c *gin.Context) {
	deviceIDStr := c.Query("deviceId")
	if deviceIDStr == "" {
		response.ErrorCode(c, http.StatusBadRequest, "设备ID不能为空")
		return
	}
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的设备ID")
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		response.ErrorCode(c, http.StatusUnauthorized, "未授权")
		return
	}

	permissions, err := s.useCase.GetUserNetworkDevicePermissions(c.Request.Context(), userID, uint(deviceID))
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"permissions": permissions,
	})
}
