package skills

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// AuditOperationSummarySkill 操作日志统计
type AuditOperationSummarySkill struct{}

func (s *AuditOperationSummarySkill) Name() string        { return "audit.operation_summary" }
func (s *AuditOperationSummarySkill) Description() string {
	return "统计分析操作日志，包括各模块操作次数、各用户操作次数、操作类型分布。支持按时间范围筛选"
}
func (s *AuditOperationSummarySkill) RiskLevel() string   { return "low" }
func (s *AuditOperationSummarySkill) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"days": {"type": "integer", "description": "统计最近 N 天的日志，默认 1（今天）", "default": 1},
			"username": {"type": "string", "description": "按用户名筛选"}
		}
	}`)
}

func (s *AuditOperationSummarySkill) Execute(ctx biz.SkillContext) (any, error) {
	days := 1
	if d, ok := ctx.Params["days"].(float64); ok && d > 0 {
		days = int(d)
	}
	username, _ := ctx.Params["username"].(string)

	since := time.Now().AddDate(0, 0, -days)

	baseQuery := ctx.DB.Table("sys_operation_logs").Where("created_at >= ? AND deleted_at IS NULL", since)
	if username != "" {
		baseQuery = baseQuery.Where("username = ?", username)
	}

	// 总操作次数
	var totalOps int64
	baseQuery.Count(&totalOps)

	// 按模块统计
	type ModuleStat struct {
		Module string `json:"module"`
		Count  int64  `json:"count"`
	}
	var moduleStats []ModuleStat
	ctx.DB.Table("sys_operation_logs").
		Select("module, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", since).
		Group("module").
		Order("count DESC").
		Limit(10).
		Find(&moduleStats)

	// 按用户统计
	type UserStat struct {
		Username string `json:"username"`
		Count    int64  `json:"count"`
	}
	var userStats []UserStat
	ctx.DB.Table("sys_operation_logs").
		Select("username, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", since).
		Group("username").
		Order("count DESC").
		Limit(10).
		Find(&userStats)

	// 按操作类型统计
	type ActionStat struct {
		Action string `json:"action"`
		Count  int64  `json:"count"`
	}
	var actionStats []ActionStat
	ctx.DB.Table("sys_operation_logs").
		Select("action, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", since).
		Group("action").
		Order("count DESC").
		Find(&actionStats)

	return map[string]any{
		"period":      fmt.Sprintf("最近 %d 天", days),
		"totalOps":    totalOps,
		"byModule":    moduleStats,
		"byUser":      userStats,
		"byAction":    actionStats,
	}, nil
}

// AuditLoginAnalysisSkill 登录行为分析
type AuditLoginAnalysisSkill struct{}

func (s *AuditLoginAnalysisSkill) Name() string        { return "audit.login_analysis" }
func (s *AuditLoginAnalysisSkill) Description() string {
	return "分析登录行为，包括登录次数统计、失败登录记录、异常登录 IP 检测"
}
func (s *AuditLoginAnalysisSkill) RiskLevel() string   { return "low" }
func (s *AuditLoginAnalysisSkill) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"days": {"type": "integer", "description": "统计最近 N 天的登录日志，默认 7", "default": 7}
		}
	}`)
}

func (s *AuditLoginAnalysisSkill) Execute(ctx biz.SkillContext) (any, error) {
	days := 7
	if d, ok := ctx.Params["days"].(float64); ok && d > 0 {
		days = int(d)
	}
	since := time.Now().AddDate(0, 0, -days)

	// 登录总次数
	var totalLogins int64
	ctx.DB.Table("sys_login_logs").Where("created_at >= ? AND deleted_at IS NULL", since).Count(&totalLogins)

	// 失败登录
	var failedLogins int64
	ctx.DB.Table("sys_login_logs").Where("created_at >= ? AND deleted_at IS NULL AND status = 0", since).Count(&failedLogins)

	// 按用户统计失败登录
	type FailedUser struct {
		Username string `json:"username"`
		Count    int64  `json:"count"`
	}
	var failedUsers []FailedUser
	ctx.DB.Table("sys_login_logs").
		Select("username, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL AND status = 0", since).
		Group("username").
		Having("count > 2").
		Order("count DESC").
		Find(&failedUsers)

	// 按 IP 统计
	type IPStat struct {
		IP    string `json:"ip"`
		Count int64  `json:"count"`
	}
	var ipStats []IPStat
	ctx.DB.Table("sys_login_logs").
		Select("ip, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", since).
		Group("ip").
		Order("count DESC").
		Limit(10).
		Find(&ipStats)

	return map[string]any{
		"period":       fmt.Sprintf("最近 %d 天", days),
		"totalLogins":  totalLogins,
		"failedLogins": failedLogins,
		"failedUsers":  failedUsers,
		"topIPs":       ipStats,
	}, nil
}

// AuditSessionSummarySkill 终端会话汇总
type AuditSessionSummarySkill struct{}

func (s *AuditSessionSummarySkill) Name() string        { return "audit.session_summary" }
func (s *AuditSessionSummarySkill) Description() string {
	return "统计终端会话（SSH/RDP）的数量、类型分布和连接情况"
}
func (s *AuditSessionSummarySkill) RiskLevel() string   { return "low" }
func (s *AuditSessionSummarySkill) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"days": {"type": "integer", "description": "统计最近 N 天，默认 1", "default": 1}
		}
	}`)
}

func (s *AuditSessionSummarySkill) Execute(ctx biz.SkillContext) (any, error) {
	days := 1
	if d, ok := ctx.Params["days"].(float64); ok && d > 0 {
		days = int(d)
	}
	since := time.Now().AddDate(0, 0, -days)

	var totalSessions int64
	ctx.DB.Table("terminal_audit_logs").Where("created_at >= ?", since).Count(&totalSessions)

	// 按协议类型统计
	type ProtoStat struct {
		Protocol string `json:"protocol"`
		Count    int64  `json:"count"`
	}
	var protoStats []ProtoStat
	ctx.DB.Table("terminal_audit_logs").
		Select("protocol, COUNT(*) as count").
		Where("created_at >= ?", since).
		Group("protocol").
		Find(&protoStats)

	// 按用户统计
	type UserStat struct {
		Username string `json:"username"`
		Count    int64  `json:"count"`
	}
	var userStats []UserStat
	ctx.DB.Table("terminal_audit_logs").
		Select("username, COUNT(*) as count").
		Where("created_at >= ?", since).
		Group("username").
		Order("count DESC").
		Limit(10).
		Find(&userStats)

	return map[string]any{
		"period":        fmt.Sprintf("最近 %d 天", days),
		"totalSessions": totalSessions,
		"byProtocol":    protoStats,
		"byUser":        userStats,
	}, nil
}

// RegisterAuditSkills 注册审计分析 Skills
func RegisterAuditSkills(registry *biz.ToolRegistry) {
	registry.Register(&AuditOperationSummarySkill{})
	registry.Register(&AuditLoginAnalysisSkill{})
	registry.Register(&AuditSessionSummarySkill{})
}
