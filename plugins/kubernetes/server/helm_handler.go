package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/mom/plugins/kubernetes/model"
	"github.com/ydcloud-dy/mom/plugins/kubernetes/service"
)

type HelmHandler struct {
	helmService *service.HelmService
}

func NewHelmHandler(helmService *service.HelmService) *HelmHandler {
	return &HelmHandler{
		helmService: helmService,
	}
}

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

	if err := h.helmService.UninstallRelease(c.Request.Context(), uint(clusterID), namespace, name); err != nil {
		HandleK8sError(c, err, "Helm Release")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "卸载成功"})
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

	release, err := h.helmService.InstallRelease(c.Request.Context(), &req)
	if err != nil {
		HandleK8sError(c, err, "Helm Install")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "安装成功", "data": release})
}
