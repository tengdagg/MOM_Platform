package skills

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// TaskHistorySkill 查询任务执行历史
type TaskHistorySkill struct{}

func (s *TaskHistorySkill) Name() string        { return "task.history" }
func (s *TaskHistorySkill) Description() string {
	return "查询任务执行历史记录，支持按状态（成功/失败）、时间范围筛选，返回最近的任务执行结果"
}
func (s *TaskHistorySkill) RiskLevel() string   { return "low" }
func (s *TaskHistorySkill) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"days": {"type": "integer", "description": "查询最近 N 天的记录，默认 7", "default": 7},
			"status": {"type": "string", "description": "状态筛选: success / failed / running"},
			"limit": {"type": "integer", "description": "返回数量，默认 20", "default": 20}
		}
	}`)
}

func (s *TaskHistorySkill) Execute(ctx biz.SkillContext) (any, error) {
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

// TaskExecuteSkill 执行任务（高风险）
type TaskExecuteSkill struct{}

func (s *TaskExecuteSkill) Name() string        { return "task.execute" }
func (s *TaskExecuteSkill) Description() string {
	return "在指定主机上执行 Ad-hoc 命令。这是高风险操作，执行前需要用户确认。通过主机 IP 或名称指定目标主机"
}
func (s *TaskExecuteSkill) RiskLevel() string   { return "critical" }
func (s *TaskExecuteSkill) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"host_ip": {"type": "string", "description": "目标主机 IP"},
			"command": {"type": "string", "description": "要执行的命令"}
		},
		"required": ["host_ip", "command"]
	}`)
}

func (s *TaskExecuteSkill) Execute(ctx biz.SkillContext) (any, error) {
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

// RegisterTaskSkills 注册任务中心 Skills
func RegisterTaskSkills(registry *biz.ToolRegistry) {
	registry.Register(&TaskHistorySkill{})
	registry.Register(&TaskExecuteSkill{})
}
