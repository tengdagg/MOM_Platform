package asset

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	assetbiz "github.com/ydcloud-dy/mom/internal/biz/asset"
	auditbiz "github.com/ydcloud-dy/mom/internal/biz/audit"
	appLogger "github.com/ydcloud-dy/mom/pkg/logger"
	"github.com/ydcloud-dy/mom/pkg/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SystemConfigHandler 系统配置处理器
type SystemConfigHandler struct {
	db *gorm.DB
}

// NewSystemConfigHandler 创建系统配置处理器
func NewSystemConfigHandler(db *gorm.DB) *SystemConfigHandler {
	return &SystemConfigHandler{db: db}
}

// GetAllConfig 获取所有系统配置
func (h *SystemConfigHandler) GetAllConfig(c *gin.Context) {
	var configs []assetbiz.SystemConfig
	h.db.Find(&configs)

	result := make(map[string]string)
	for _, cfg := range configs {
		result[cfg.ConfigKey] = cfg.Value
	}

	response.Success(c, result)
}

// SaveAllConfig 批量保存系统配置
func (h *SystemConfigHandler) SaveAllConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	for key, value := range req {
		var strVal string
		switch v := value.(type) {
		case string:
			strVal = v
		case float64:
			if v == float64(int(v)) {
				strVal = strconv.Itoa(int(v))
			} else {
				strVal = strconv.FormatFloat(v, 'f', -1, 64)
			}
		case bool:
			strVal = strconv.FormatBool(v)
		default:
			b, _ := json.Marshal(v)
			strVal = string(b)
		}

		var existing assetbiz.SystemConfig
		result := h.db.Where("config_key = ?", key).First(&existing)
		if result.Error != nil {
			// 不存在，创建
			h.db.Create(&assetbiz.SystemConfig{
				ConfigKey: key,
				Value:     strVal,
			})
		} else {
			// 存在，更新
			h.db.Model(&assetbiz.SystemConfig{}).Where("config_key = ?", key).Update("value", strVal)
		}
	}

	response.SuccessWithMessage(c, "保存成功", nil)
}

// StartAuditLogCleanupScheduler 启动审计日志定时清理任务
func (h *SystemConfigHandler) StartAuditLogCleanupScheduler() {
	go func() {
		// 启动后 60 秒执行一次
		time.Sleep(60 * time.Second)
		h.doAuditLogCleanup()

		// 每 24 小时执行一次
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			h.doAuditLogCleanup()
		}
	}()
}

// doAuditLogCleanup 执行审计日志清理
func (h *SystemConfigHandler) doAuditLogCleanup() {
	// 读取保留天数
	retentionDays := 30
	var cfg assetbiz.SystemConfig
	if err := h.db.Where("config_key = ?", "logRetentionDays").First(&cfg).Error; err == nil {
		if d, err := strconv.Atoi(cfg.Value); err == nil && d > 0 {
			retentionDays = d
		}
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	totalDeleted := 0

	// 清理操作日志
	result := h.db.Unscoped().Where("created_at < ?", cutoff).Delete(&auditbiz.SysOperationLog{})
	if result.Error == nil && result.RowsAffected > 0 {
		totalDeleted += int(result.RowsAffected)
		appLogger.Info("清理过期操作日志", zap.Int64("count", result.RowsAffected), zap.Int("retentionDays", retentionDays))
	}

	// 清理登录日志
	result = h.db.Unscoped().Where("created_at < ?", cutoff).Delete(&auditbiz.SysLoginLog{})
	if result.Error == nil && result.RowsAffected > 0 {
		totalDeleted += int(result.RowsAffected)
		appLogger.Info("清理过期登录日志", zap.Int64("count", result.RowsAffected), zap.Int("retentionDays", retentionDays))
	}

	// 清理数据日志
	result = h.db.Unscoped().Where("created_at < ?", cutoff).Delete(&auditbiz.SysDataLog{})
	if result.Error == nil && result.RowsAffected > 0 {
		totalDeleted += int(result.RowsAffected)
		appLogger.Info("清理过期数据日志", zap.Int64("count", result.RowsAffected), zap.Int("retentionDays", retentionDays))
	}

	// 清理孤立的终端录制文件（可选）
	h.cleanOrphanedRecordings()

	if totalDeleted > 0 {
		appLogger.Info("审计日志定时清理完成",
			zap.Int("totalDeleted", totalDeleted),
			zap.Int("retentionDays", retentionDays))
	}
}

// cleanOrphanedRecordings 清理没有数据库记录的孤立录制文件
func (h *SystemConfigHandler) cleanOrphanedRecordings() {
	recordingDir := "./data/terminal-recordings"
	entries, err := os.ReadDir(recordingDir)
	if err != nil {
		return
	}

	// 获取所有有记录的文件路径
	var sessions []struct {
		RecordingPath string
	}
	h.db.Model(&assetbiz.TerminalSession{}).Select("recording_path").Where("recording_path != ''").Find(&sessions)

	validPaths := make(map[string]bool)
	for _, s := range sessions {
		validPaths[s.RecordingPath] = true
	}

	cleaned := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fullPath := recordingDir + "/" + entry.Name()
		if !validPaths[fullPath] {
			// 文件在目录中但不在数据库中，属于孤立文件
			info, err := entry.Info()
			if err != nil {
				continue
			}
			// 只清理超过 7 天的孤立文件（避免清理正在使用的）
			if time.Since(info.ModTime()) > 7*24*time.Hour {
				_ = os.Remove(fullPath)
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		appLogger.Info("清理孤立录制文件", zap.Int("count", cleaned))
	}
}
