package server

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/mom/pkg/response"
	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// ListModels 获取模型配置列表
func (h *Handler) ListModels(c *gin.Context) {
	var models []biz.AIModelConfig
	if err := h.db.Order("is_default DESC, id ASC").Find(&models).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取模型列表失败")
		return
	}
	// 标记是否已设置 API Key
	for i := range models {
		models[i].APIKeySet = models[i].APIKey != ""
	}
	response.Success(c, models)
}

// CreateModel 添加模型配置
func (h *Handler) CreateModel(c *gin.Context) {
	var req struct {
		Name         string  `json:"name" binding:"required"`
		Provider     string  `json:"provider" binding:"required"`
		BaseURL      string  `json:"baseUrl"`
		APIKey       string  `json:"apiKey"`
		ModelName    string  `json:"modelName" binding:"required"`
		MaxTokens    int     `json:"maxTokens"`
		Temperature  float64 `json:"temperature"`
		MaxToolCalls int     `json:"maxToolCalls"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	model := &biz.AIModelConfig{
		Name:         req.Name,
		Provider:     req.Provider,
		BaseURL:      req.BaseURL,
		APIKey:       req.APIKey,
		ModelName:    req.ModelName,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		MaxToolCalls: req.MaxToolCalls,
		Status:       1,
	}
	if model.MaxTokens == 0 {
		model.MaxTokens = 4096
	}
	if model.Temperature == 0 {
		model.Temperature = 0.7
	}
	if model.MaxToolCalls == 0 {
		model.MaxToolCalls = 10
	}

	if err := h.db.Create(model).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "创建模型配置失败")
		return
	}
	model.APIKeySet = model.APIKey != ""
	response.Success(c, model)
}

// UpdateModel 更新模型配置
func (h *Handler) UpdateModel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "无效的 ID")
		return
	}

	var existing biz.AIModelConfig
	if err := h.db.First(&existing, id).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "模型配置不存在")
		return
	}

	var req struct {
		Name         string  `json:"name"`
		Provider     string  `json:"provider"`
		BaseURL      string  `json:"baseUrl"`
		APIKey       string  `json:"apiKey"`
		ModelName    string  `json:"modelName"`
		MaxTokens    int     `json:"maxTokens"`
		Temperature  float64 `json:"temperature"`
		MaxToolCalls *int    `json:"maxToolCalls"`
		Status       *int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Provider != "" {
		updates["provider"] = req.Provider
	}
	if req.BaseURL != "" {
		updates["base_url"] = req.BaseURL
	}
	if req.APIKey != "" {
		updates["api_key"] = req.APIKey
	}
	if req.ModelName != "" {
		updates["model_name"] = req.ModelName
	}
	if req.MaxTokens > 0 {
		updates["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		updates["temperature"] = req.Temperature
	}
	if req.MaxToolCalls != nil {
		updates["max_tool_calls"] = *req.MaxToolCalls
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if err := h.db.Model(&existing).Updates(updates).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "更新失败")
		return
	}

	response.Success(c, nil)
}

// DeleteModel 删除模型配置
func (h *Handler) DeleteModel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "无效的 ID")
		return
	}

	if err := h.db.Delete(&biz.AIModelConfig{}, id).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}
	response.Success(c, nil)
}

// TestModel 测试模型连通性
func (h *Handler) TestModel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "无效的 ID")
		return
	}

	var model biz.AIModelConfig
	if err := h.db.First(&model, id).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "模型配置不存在")
		return
	}

	adapter := biz.NewModelAdapter(&model)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := adapter.TestConnection(ctx); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "连接测试失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, "模型连接测试成功", nil)
}

// SetDefaultModel 设置默认模型
func (h *Handler) SetDefaultModel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "无效的 ID")
		return
	}

	// 取消所有默认
	h.db.Model(&biz.AIModelConfig{}).Where("is_default = ?", true).Update("is_default", false)
	// 设置新默认
	h.db.Model(&biz.AIModelConfig{}).Where("id = ?", id).Update("is_default", true)

	response.Success(c, nil)
}
