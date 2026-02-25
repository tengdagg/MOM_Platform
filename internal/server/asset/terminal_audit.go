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
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	assetbiz "github.com/ydcloud-dy/mom/internal/biz/asset"
	appLogger "github.com/ydcloud-dy/mom/pkg/logger"
	"github.com/ydcloud-dy/mom/pkg/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TerminalAuditHandler 终端审计处理器
type TerminalAuditHandler struct {
	db *gorm.DB
}

// NewTerminalAuditHandler 创建终端审计处理器
func NewTerminalAuditHandler(db *gorm.DB) *TerminalAuditHandler {
	return &TerminalAuditHandler{db: db}
}

// ListTerminalSessions 获取终端会话列表
// @Summary 获取终端会话列表
// @Description 分页获取终端审计会话列表，支持搜索
// @Tags 终端审计
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键字"
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/terminal-sessions [get]
func (h *TerminalAuditHandler) ListTerminalSessions(c *gin.Context) {
	// 获取分页参数
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "10")
	keyword := c.Query("keyword")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	// 构建查询
	query := h.db.Model(&assetbiz.TerminalSession{})

	// 搜索关键词
	if keyword != "" {
		query = query.Where("host_name LIKE ? OR host_ip LIKE ? OR username LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}

	// 分页查询
	var sessions []*assetbiz.TerminalSession
	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&sessions).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		return
	}

	// 转换为VO
	list := make([]*assetbiz.TerminalSessionInfo, 0, len(sessions))
	for _, session := range sessions {
		sessionType := session.SessionType
		if sessionType == "" {
			sessionType = "ssh" // 兼容旧数据
		}
		sessionTypeText := "SSH"
		switch sessionType {
		case "rdp":
			sessionTypeText = "RDP"
		case "telnet":
			sessionTypeText = "Telnet"
		case "ssh":
			sessionTypeText = "SSH"
		}

		info := &assetbiz.TerminalSessionInfo{
			ID:              session.ID,
			SessionType:     sessionType,
			SessionTypeText: sessionTypeText,
			HostID:          session.HostID,
			HostName:        session.HostName,
			HostIP:          session.HostIP,
			UserID:          session.UserID,
			Username:        session.Username,
			Duration:        session.Duration,
			DurationText:    formatDuration(session.Duration),
			FileSize:        session.FileSize,
			FileSizeText:    formatFileSize(session.FileSize),
			Status:          session.Status,
			StatusText:      getStatusText(session.Status),
			CreatedAt:       session.CreatedAt,
			CreatedAtText:   session.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		list = append(list, info)
	}

	response.Success(c, gin.H{
		"total": total,
		"list":  list,
	})
}

// PlayTerminalSession 播放终端会话录制
// @Summary 播放终端会话
// @Description 获取终端会话的录制文件内容用于回放
// @Tags 终端审计
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "会话ID"
// @Success 200 {string} string "录制文件内容"
// @Failure 404 {object} response.Response "会话不存在"
// @Router /api/v1/terminal-sessions/{id}/play [get]
func (h *TerminalAuditHandler) PlayTerminalSession(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的会话ID")
		return
	}

	// 查询会话
	var session assetbiz.TerminalSession
	if err := h.db.First(&session, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.ErrorCode(c, http.StatusNotFound, "会话不存在")
		} else {
			response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		}
		return
	}

	// RDP会话没有录制文件
	if session.SessionType == "rdp" || session.RecordingPath == "" {
		response.ErrorCode(c, http.StatusBadRequest, "RDP会话暂不支持回放")
		return
	}

	// 读取录制文件
	content, err := os.ReadFile(session.RecordingPath)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取录制文件失败")
		return
	}

	// 返回录制文件内容
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, string(content))
}

// DeleteTerminalSession 删除终端会话
// @Summary 删除终端会话
// @Description 删除指定的终端会话记录及其录制文件
// @Tags 终端审计
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "会话ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 404 {object} response.Response "会话不存在"
// @Router /api/v1/terminal-sessions/{id} [delete]
func (h *TerminalAuditHandler) DeleteTerminalSession(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的会话ID")
		return
	}

	// 查询会话
	var session assetbiz.TerminalSession
	if err := h.db.First(&session, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.ErrorCode(c, http.StatusNotFound, "会话不存在")
		} else {
			response.ErrorCode(c, http.StatusInternalServerError, "查询失败")
		}
		return
	}

	// 删除录制文件
	// 即使文件删除失败，仍然继续删除数据库记录
	_ = os.Remove(session.RecordingPath)

	// 删除数据库记录
	if err := h.db.Delete(&session).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除失败")
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// formatDuration 格式化时长
func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	} else if seconds < 3600 {
		minutes := seconds / 60
		secs := seconds % 60
		return fmt.Sprintf("%dm %ds", minutes, secs)
	} else {
		hours := seconds / 3600
		minutes := (seconds % 3600) / 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

// formatFileSize 格式化文件大小
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// getStatusText 获取状态文本
func getStatusText(status string) string {
	statusMap := map[string]string{
		"recording": "录制中",
		"completed": "已完成",
		"failed":    "失败",
	}
	if text, ok := statusMap[status]; ok {
		return text
	}
	return status
}

// GetRetentionConfig 获取终端审计保留配置
// @Summary 获取终端审计保留配置
// @Description 获取终端审计会话记录和录制文件的保留天数配置
// @Tags 终端审计
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=map[string]interface{}} "获取成功"
// @Router /api/v1/terminal-sessions/retention [get]
func (h *TerminalAuditHandler) GetRetentionConfig(c *gin.Context) {
	var config assetbiz.SystemConfig
	result := h.db.Where("config_key = ?", "terminal_audit_retention_days").First(&config)
	if result.Error != nil {
		// 默认30天
		response.Success(c, gin.H{
			"retentionDays": 30,
			"autoCleanup":   true,
		})
		return
	}

	days, _ := strconv.Atoi(config.Value)
	if days == 0 {
		days = 30
	}

	response.Success(c, gin.H{
		"retentionDays": days,
		"autoCleanup":   true,
	})
}

// UpdateRetentionConfig 更新终端审计保留配置
// @Summary 更新终端审计保留配置
// @Description 更新终端审计记录的保留天数及是否自动清理
// @Tags 终端审计
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body map[string]interface{} true "保留配置信息"
// @Success 200 {object} response.Response "保存成功"
// @Router /api/v1/terminal-sessions/retention [put]
func (h *TerminalAuditHandler) UpdateRetentionConfig(c *gin.Context) {
	var req struct {
		RetentionDays int  `json:"retentionDays" binding:"required,min=1,max=3650"`
		AutoCleanup   bool `json:"autoCleanup"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	config := assetbiz.SystemConfig{
		ConfigKey: "terminal_audit_retention_days",
		Value:     strconv.Itoa(req.RetentionDays),
		Remark:    "终端审计记录保留天数",
	}

	// Upsert
	result := h.db.Where("config_key = ?", "terminal_audit_retention_days").First(&assetbiz.SystemConfig{})
	if result.Error != nil {
		h.db.Create(&config)
	} else {
		h.db.Model(&assetbiz.SystemConfig{}).Where("config_key = ?", "terminal_audit_retention_days").
			Updates(map[string]interface{}{"value": config.Value, "remark": config.Remark})
	}

	response.SuccessWithMessage(c, "保存成功", nil)
}

// CleanupExpiredSessions 手动触发清理过期会话
// @Summary 手动清理过期终端会话
// @Description 手动触发一次清理任务，删除超出保留期限的审计记录和录制文件
// @Tags 终端审计
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response "清理成功"
// @Router /api/v1/terminal-sessions/cleanup [post]
func (h *TerminalAuditHandler) CleanupExpiredSessions(c *gin.Context) {
	deleted, err := h.doCleanup()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "清理失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, fmt.Sprintf("清理完成，共删除 %d 条过期记录", deleted), gin.H{
		"deletedCount": deleted,
	})
}

// doCleanup 执行清理逻辑
func (h *TerminalAuditHandler) doCleanup() (int, error) {
	// 获取保留天数
	retentionDays := 30
	var config assetbiz.SystemConfig
	if err := h.db.Where("config_key = ?", "terminal_audit_retention_days").First(&config).Error; err == nil {
		if d, err := strconv.Atoi(config.Value); err == nil && d > 0 {
			retentionDays = d
		}
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	// 查询过期会话
	var sessions []*assetbiz.TerminalSession
	if err := h.db.Where("created_at < ? AND status != 'recording'", cutoffTime).Find(&sessions).Error; err != nil {
		return 0, err
	}

	if len(sessions) == 0 {
		return 0, nil
	}

	// 删除录制文件和数据库记录
	deletedCount := 0
	for _, session := range sessions {
		// 删除录制文件
		if session.RecordingPath != "" {
			_ = os.Remove(session.RecordingPath)
		}
		// 删除数据库记录
		if err := h.db.Unscoped().Delete(session).Error; err == nil {
			deletedCount++
		}
	}

	return deletedCount, nil
}

// StartCleanupScheduler 启动定时清理任务
func (h *TerminalAuditHandler) StartCleanupScheduler() {
	go func() {
		// 启动后先执行一次清理
		time.Sleep(30 * time.Second)
		deleted, err := h.doCleanup()
		if err != nil {
			appLogger.Error("终端审计初始清理失败", zap.Error(err))
		} else if deleted > 0 {
			appLogger.Info("终端审计初始清理完成", zap.Int("deletedCount", deleted))
		}

		// 每24小时执行一次
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			deleted, err := h.doCleanup()
			if err != nil {
				appLogger.Error("终端审计定时清理失败", zap.Error(err))
			} else if deleted > 0 {
				appLogger.Info("终端审计定时清理完成", zap.Int("deletedCount", deleted))
			}
		}
	}()
}
