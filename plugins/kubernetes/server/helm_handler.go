package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	rbacBiz "github.com/ydcloud-dy/mom/internal/biz/rbac"
	rbacData "github.com/ydcloud-dy/mom/internal/data/rbac"
	"github.com/ydcloud-dy/mom/plugins/kubernetes/model"
	"github.com/ydcloud-dy/mom/plugins/kubernetes/service"
	"gorm.io/gorm"
)

type HelmHandler struct {
	helmService *service.HelmService
	db          *gorm.DB
}

func NewHelmHandler(helmService *service.HelmService, db *gorm.DB) *HelmHandler {
	return &HelmHandler{
		helmService: helmService,
		db:          db,
	}
}

// checkNamespaceWritePermission 检查用户是否对指定命名空间有写权限
// 写权限要求: 平台管理员、cluster-owner、或对应命名空间的 namespace-owner
// namespace-viewer、cluster-viewer 等只读角色没有写权限
func (h *HelmHandler) checkNamespaceWritePermission(c *gin.Context, clusterID uint64, namespace string) bool {
	currentUserID, ok := GetCurrentUserID(c)
	if !ok {
		return false
	}

	// 1. 平台管理员直接放行
	roleRepo := rbacData.NewRoleRepo(h.db)
	roleUseCase := rbacBiz.NewRoleUseCase(roleRepo)
	roles, err := roleUseCase.GetByUserID(context.Background(), currentUserID)
	if err == nil {
		for _, role := range roles {
			if role.Code == "admin" {
				return true
			}
		}
	}

	// 2. 检查 K8s 角色绑定
	roleBindingService := service.NewRoleBindingService(h.db)
	uid := uint64(currentUserID)
	bindings, err := roleBindingService.GetUserRoleBindings(c.Request.Context(), clusterID, &uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取用户权限失败: " + err.Error(),
		})
		return false
	}

	for _, b := range bindings {
		roleName, _ := b["roleName"].(string)
		roleNamespace, _ := b["roleNamespace"].(string)

		// 集群级别的 owner 角色 → 对所有命名空间有写权限
		if roleNamespace == "" && isWriteRole(roleName) {
			return true
		}

		// 命名空间级别的 owner 角色 → 仅对该命名空间有写权限
		if roleNamespace == namespace && isWriteRole(roleName) {
			return true
		}
	}

	// 没有写权限
	c.JSON(http.StatusForbidden, gin.H{
		"code":    403,
		"message": "权限不足：您在命名空间 " + namespace + " 中没有写入权限（需要 namespace-owner 或 cluster-owner 角色），请联系管理员在「集群授权」中为您分配相应角色",
	})
	return false
}

// isWriteRole 判断角色是否具有写权限
// owner 类角色有写权限，viewer 类角色没有
func isWriteRole(roleName string) bool {
	// 明确的只读角色
	readOnlyRoles := []string{
		"namespace-viewer",
		"cluster-viewer",
		"view",
		"view-config",
		"view-secret",
	}
	for _, r := range readOnlyRoles {
		if roleName == r {
			return false
		}
	}

	// 明确的写角色
	writeRoles := []string{
		"namespace-owner",
		"cluster-owner",
		"admin",
		"cluster-admin",
		"manage-workload",
		"manage-config",
		"manage-network",
		"manage-storage",
	}
	for _, r := range writeRoles {
		if roleName == r {
			return true
		}
	}

	// 包含 "owner"、"admin"、"manage" 关键字的角色视为有写权限
	lowerName := strings.ToLower(roleName)
	if strings.Contains(lowerName, "owner") ||
		strings.Contains(lowerName, "admin") ||
		strings.Contains(lowerName, "manage") {
		return true
	}

	// 包含 "viewer"、"view"、"readonly" 关键字的角色视为只读
	if strings.Contains(lowerName, "viewer") ||
		strings.Contains(lowerName, "view") ||
		strings.Contains(lowerName, "readonly") ||
		strings.Contains(lowerName, "read-only") {
		return false
	}

	// 未知角色默认有写权限（自定义的 K8s Role/ClusterRole）
	return true
}

// ==================== 仓库管理（无需命名空间权限检查） ====================

// ListRepos 获取仓库列表
func (h *HelmHandler) ListRepos(c *gin.Context) {
	repos, err := h.helmService.ListRepos(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": repos})
}

// AddRepo 添加仓库
func (h *HelmHandler) AddRepo(c *gin.Context) {
	var req model.HelmRepo
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.helmService.AddRepo(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "添加成功", "data": req})
}

// UpdateRepo 更新仓库
func (h *HelmHandler) UpdateRepo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	var req model.HelmRepo
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uint(id)

	if err := h.helmService.UpdateRepo(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "更新成功", "data": req})
}

// DeleteRepo 删除仓库
func (h *HelmHandler) DeleteRepo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	if err := h.helmService.DeleteRepo(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "删除成功"})
}

// ListCharts 获取 Chart 列表
func (h *HelmHandler) ListCharts(c *gin.Context) {
	repoIDStr := c.Param("repoId")
	repoID, err := strconv.ParseUint(repoIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 Repo ID"})
		return
	}

	charts, err := h.helmService.ListCharts(c.Request.Context(), uint(repoID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": charts})
}

// GetChartVersions 获取 Chart 的所有版本
func (h *HelmHandler) GetChartVersions(c *gin.Context) {
	repoIDStr := c.Param("repoId")
	repoID, err := strconv.ParseUint(repoIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 Repo ID"})
		return
	}

	chartName := c.Param("chartName")
	if chartName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chart 名称不能为空"})
		return
	}

	versions, err := h.helmService.GetChartVersions(c.Request.Context(), uint(repoID), chartName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": versions})
}

// ==================== Release 管理（读操作，无需写权限检查） ====================

// ListReleases 获取 Release 列表
func (h *HelmHandler) ListReleases(c *gin.Context) {
	clusterIDStr := c.Param("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 Cluster ID"})
		return
	}

	releases, err := h.helmService.ListReleases(c.Request.Context(), uint(clusterID))
	if err != nil {
		HandleK8sError(c, err, "Helm Releases")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": releases})
}

// GetRelease 获取 Release 详情
func (h *HelmHandler) GetRelease(c *gin.Context) {
	clusterIDStr := c.Param("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 Cluster ID"})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	release, err := h.helmService.GetRelease(c.Request.Context(), uint(clusterID), namespace, name)
	if err != nil {
		HandleK8sError(c, err, "Helm Release")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": release})
}

// ==================== Release 写操作（需要 namespace-owner 权限） ====================

// InstallRelease 安装 Release
func (h *HelmHandler) InstallRelease(c *gin.Context) {
	var req service.InstallReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get clusterId from URL param
	clusterIDStr := c.Param("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 Cluster ID"})
		return
	}
	req.ClusterID = uint(clusterID)

	// RBAC 检查：用户是否对目标命名空间有写权限
	if !h.checkNamespaceWritePermission(c, clusterID, req.Namespace) {
		return
	}

	release, err := h.helmService.InstallRelease(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "error": "操作失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "安装成功", "data": release})
}

// UpgradeRelease 升级 Release
func (h *HelmHandler) UpgradeRelease(c *gin.Context) {
	clusterIDStr := c.Param("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 Cluster ID"})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	// RBAC 检查：用户是否对目标命名空间有写权限
	if !h.checkNamespaceWritePermission(c, clusterID, namespace) {
		return
	}

	var req service.UpgradeReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	release, err := h.helmService.UpgradeRelease(c.Request.Context(), uint(clusterID), namespace, name, req.Values)
	if err != nil {
		HandleK8sError(c, err, "Helm Release")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "升级成功", "data": release})
}

// UninstallRelease 卸载 Release
func (h *HelmHandler) UninstallRelease(c *gin.Context) {
	clusterIDStr := c.Param("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 Cluster ID"})
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")

	// RBAC 检查：用户是否对目标命名空间有写权限
	if !h.checkNamespaceWritePermission(c, clusterID, namespace) {
		return
	}

	if err := h.helmService.UninstallRelease(c.Request.Context(), uint(clusterID), namespace, name); err != nil {
		HandleK8sError(c, err, "Helm Release")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "卸载成功"})
}
