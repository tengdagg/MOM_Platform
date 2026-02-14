package skills

import (
	"fmt"
	"time"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// RegisterTaskSkills 注册任务中心 Skills
func RegisterTaskSkills(registry *biz.ToolRegistry) {
	registry.Register(MustLoadBuiltinSkill("task.history", executeTaskHistory))
	registry.Register(MustLoadBuiltinSkill("task.execute", executeTaskExecute))
}

// executeTaskHistory 查询任务执行历史
func executeTaskHistory(ctx biz.SkillContext) (any, error) {
	days := 7
	if d, ok := ctx.Params["days"].(float64); ok && d > 0 {
		days = int(d)
	}
	statusFilter, _ := ctx.Params["status"].(string)
	limit := 20
	if l, ok := ctx.Params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	since := time.Now().AddDate(0, 0, -days)

	type TaskRecord struct {
		ID        uint      `json:"id"`
		Name      string    `json:"name"`
		Type      string    `json:"type"`
		Status    string    `json:"status"`
		Username  string    `json:"username"`
		CreatedAt time.Time `json:"createdAt"`
		Duration  int       `json:"duration"`
	}

	query := ctx.DB.Table("task_execution_records").
		Where("created_at >= ?", since)

	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}

	var records []TaskRecord
	if err := query.Order("created_at DESC").Limit(limit).Find(&records).Error; err != nil {
		// 表可能不存在，返回空结果
		return map[string]any{
			"period":  fmt.Sprintf("最近 %d 天", days),
			"records": []any{},
			"total":   0,
			"message": "暂无任务执行记录",
		}, nil
	}

	// 统计
	var totalCount int64
	ctx.DB.Table("task_execution_records").Where("created_at >= ?", since).Count(&totalCount)
	var failedCount int64
	ctx.DB.Table("task_execution_records").Where("created_at >= ? AND status = 'failed'", since).Count(&failedCount)

	return map[string]any{
		"period":  fmt.Sprintf("最近 %d 天", days),
		"records": records,
		"total":   totalCount,
		"failed":  failedCount,
	}, nil
}

// executeTaskExecute 执行任务（高风险）
func executeTaskExecute(ctx biz.SkillContext) (any, error) {
	hostIP, _ := ctx.Params["host_ip"].(string)
	command, _ := ctx.Params["command"].(string)

	// 高风险操作 - 当前只返回提示，实际执行需要通过确认机制
	return map[string]any{
		"status":  "pending_confirmation",
		"message": fmt.Sprintf("⚠️ 高风险操作: 将在主机 %s 上执行命令: %s\n请通过任务中心手动执行，AI 暂不支持直接执行远程命令。", hostIP, command),
		"host":    hostIP,
		"command": command,
	}, nil
}
