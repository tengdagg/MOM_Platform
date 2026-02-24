package skills

import (
	"fmt"
	"time"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// RegisterAuditSkills 注册审计分析 Skills
func RegisterAuditSkills(registry *biz.ToolRegistry) {
	registry.Register(MustLoadBuiltinSkill("audit.operation_summary", executeAuditOperationSummary))
	registry.Register(MustLoadBuiltinSkill("audit.login_analysis", executeAuditLoginAnalysis))
	registry.Register(MustLoadBuiltinSkill("audit.session_summary", executeAuditSessionSummary))
	registry.Register(MustLoadBuiltinSkill("audit.data_changes", executeAuditDataChanges))
}

// executeAuditOperationSummary 操作日志统计
func executeAuditOperationSummary(ctx biz.SkillContext) (any, error) {
	days := 1
	if d, ok := ctx.Params["days"].(float64); ok && d > 0 {
		days = int(d)
	}
	username, _ := ctx.Params["username"].(string)
	module, _ := ctx.Params["module"].(string)
	action, _ := ctx.Params["action"].(string)

	since := time.Now().AddDate(0, 0, -days)

	baseQuery := ctx.DB.Table("sys_operation_log").Where("created_at >= ? AND deleted_at IS NULL", since)
	if username != "" {
		baseQuery = baseQuery.Where("username LIKE ?", "%"+username+"%")
	}
	if module != "" {
		baseQuery = baseQuery.Where("module LIKE ?", "%"+module+"%")
	}
	if action != "" {
		baseQuery = baseQuery.Where("action LIKE ?", "%"+action+"%")
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
	moduleQ := ctx.DB.Table("sys_operation_log").
		Select("module, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", since)
	if username != "" {
		moduleQ = moduleQ.Where("username LIKE ?", "%"+username+"%")
	}
	moduleQ.Group("module").Order("count DESC").Limit(15).Find(&moduleStats)

	// 按用户统计
	type UserStat struct {
		Username string `json:"username"`
		RealName string `json:"realName"`
		Count    int64  `json:"count"`
	}
	var userStats []UserStat
	ctx.DB.Table("sys_operation_log").
		Select("username, real_name, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", since).
		Group("username, real_name").
		Order("count DESC").
		Limit(10).
		Find(&userStats)

	// 按操作类型统计
	type ActionStat struct {
		Action string `json:"action"`
		Count  int64  `json:"count"`
	}
	var actionStats []ActionStat
	ctx.DB.Table("sys_operation_log").
		Select("action, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", since).
		Group("action").
		Order("count DESC").
		Limit(15).
		Find(&actionStats)

	// 错误操作统计
	var errorOps int64
	ctx.DB.Table("sys_operation_log").Where("created_at >= ? AND deleted_at IS NULL AND status >= 400", since).Count(&errorOps)

	// AI 操作统计
	var aiOps int64
	ctx.DB.Table("sys_operation_log").Where("created_at >= ? AND deleted_at IS NULL AND method = 'SKILL'", since).Count(&aiOps)

	// 最近操作记录（前 20 条）
	type RecentOp struct {
		Username    string    `json:"username"`
		Module      string    `json:"module"`
		Action      string    `json:"action"`
		Description string    `json:"description"`
		IP          string    `json:"ip"`
		Status      int       `json:"status"`
		CostTime    int64     `json:"costTime"`
		CreatedAt   time.Time `json:"createdAt"`
	}
	var recentOps []RecentOp
	recentQ := ctx.DB.Table("sys_operation_log").
		Where("created_at >= ? AND deleted_at IS NULL", since)
	if username != "" {
		recentQ = recentQ.Where("username LIKE ?", "%"+username+"%")
	}
	recentQ.Order("created_at DESC").Limit(20).Find(&recentOps)

	return map[string]any{
		"period":    fmt.Sprintf("最近 %d 天", days),
		"totalOps":  totalOps,
		"errorOps":  errorOps,
		"aiOps":     aiOps,
		"byModule":  moduleStats,
		"byUser":    userStats,
		"byAction":  actionStats,
		"recentOps": recentOps,
	}, nil
}

// executeAuditLoginAnalysis 登录行为分析
func executeAuditLoginAnalysis(ctx biz.SkillContext) (any, error) {
	days := 7
	if d, ok := ctx.Params["days"].(float64); ok && d > 0 {
		days = int(d)
	}
	username, _ := ctx.Params["username"].(string)
	since := time.Now().AddDate(0, 0, -days)

	// 登录总次数
	var totalLogins int64
	q := ctx.DB.Table("sys_login_log").Where("created_at >= ? AND deleted_at IS NULL", since)
	if username != "" {
		q = q.Where("username = ?", username)
	}
	q.Count(&totalLogins)

	// 成功登录
	var successLogins int64
	sq := ctx.DB.Table("sys_login_log").Where("created_at >= ? AND deleted_at IS NULL AND login_status = 'success'", since)
	if username != "" {
		sq = sq.Where("username = ?", username)
	}
	sq.Count(&successLogins)

	// 失败登录
	failedLogins := totalLogins - successLogins

	// 按用户统计失败登录
	type FailedUser struct {
		Username string `json:"username"`
		Count    int64  `json:"count"`
		LastIP   string `json:"lastIp"`
	}
	var failedUsers []FailedUser
	ctx.DB.Table("sys_login_log").
		Select("username, COUNT(*) as count, MAX(ip) as last_ip").
		Where("created_at >= ? AND deleted_at IS NULL AND login_status = 'failed'", since).
		Group("username").
		Having("count > 2").
		Order("count DESC").
		Find(&failedUsers)

	// 按 IP 统计
	type IPStat struct {
		IP           string `json:"ip"`
		Count        int64  `json:"count"`
		FailedCount  int64  `json:"failedCount"`
		SuccessCount int64  `json:"successCount"`
	}
	var ipStats []IPStat
	ctx.DB.Table("sys_login_log").
		Select("ip, COUNT(*) as count, SUM(CASE WHEN login_status = 'failed' THEN 1 ELSE 0 END) as failed_count, SUM(CASE WHEN login_status = 'success' THEN 1 ELSE 0 END) as success_count").
		Where("created_at >= ? AND deleted_at IS NULL", since).
		Group("ip").
		Order("count DESC").
		Limit(15).
		Find(&ipStats)

	// 异常行为检测
	anomalies := make([]map[string]any, 0)

	// 检测暴力破解（同一用户连续失败 5 次以上）
	for _, u := range failedUsers {
		if u.Count >= 5 {
			anomalies = append(anomalies, map[string]any{
				"type":        "brute_force",
				"severity":    "high",
				"description": fmt.Sprintf("用户 %s 在最近 %d 天内登录失败 %d 次，最后尝试 IP: %s", u.Username, days, u.Count, u.LastIP),
			})
		}
	}

	// 检测同一 IP 大量失败
	for _, ip := range ipStats {
		if ip.FailedCount > 10 {
			anomalies = append(anomalies, map[string]any{
				"type":        "ip_attack",
				"severity":    "high",
				"description": fmt.Sprintf("IP %s 在最近 %d 天内登录失败 %d 次，可能存在暴力破解", ip.IP, days, ip.FailedCount),
			})
		}
	}

	// 按时间段统计（每小时分布）
	type HourlyStat struct {
		Hour  int   `json:"hour"`
		Count int64 `json:"count"`
	}
	var hourlyStats []HourlyStat
	ctx.DB.Table("sys_login_log").
		Select("HOUR(created_at) as hour, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", since).
		Group("HOUR(created_at)").
		Order("hour ASC").
		Find(&hourlyStats)

	return map[string]any{
		"period":             fmt.Sprintf("最近 %d 天", days),
		"totalLogins":        totalLogins,
		"successLogins":      successLogins,
		"failedLogins":       failedLogins,
		"failedUsers":        failedUsers,
		"topIPs":             ipStats,
		"anomalies":          anomalies,
		"hourlyDistribution": hourlyStats,
	}, nil
}

// executeAuditSessionSummary 终端会话汇总
func executeAuditSessionSummary(ctx biz.SkillContext) (any, error) {
	days := 1
	if d, ok := ctx.Params["days"].(float64); ok && d > 0 {
		days = int(d)
	}
	username, _ := ctx.Params["username"].(string)
	since := time.Now().AddDate(0, 0, -days)

	// SSH 会话统计
	var sshSessions int64
	sshQ := ctx.DB.Table("ssh_terminal_sessions").Where("created_at >= ?", since)
	if username != "" {
		sshQ = sshQ.Where("username = ?", username)
	}
	sshQ.Count(&sshSessions)

	// 按用户统计 SSH 会话
	type UserStat struct {
		Username string `json:"username"`
		Count    int64  `json:"count"`
	}
	var sshUserStats []UserStat
	sshUQ := ctx.DB.Table("ssh_terminal_sessions").
		Select("username, COUNT(*) as count").
		Where("created_at >= ?", since)
	if username != "" {
		sshUQ = sshUQ.Where("username = ?", username)
	}
	sshUQ.Group("username").Order("count DESC").Limit(10).Find(&sshUserStats)

	// 按主机统计
	type HostStat struct {
		HostIP string `json:"hostIp"`
		Count  int64  `json:"count"`
	}
	var hostStats []HostStat
	ctx.DB.Table("ssh_terminal_sessions").
		Select("host_ip, COUNT(*) as count").
		Where("created_at >= ?", since).
		Group("host_ip").
		Order("count DESC").
		Limit(10).
		Find(&hostStats)

	// 最近会话记录
	type SessionRecord struct {
		ID        uint      `json:"id"`
		Username  string    `json:"username"`
		HostIP    string    `json:"hostIp"`
		Protocol  string    `json:"protocol"`
		CreatedAt time.Time `json:"createdAt"`
	}
	var recentSessions []SessionRecord
	ctx.DB.Table("ssh_terminal_sessions").
		Where("created_at >= ?", since).
		Order("created_at DESC").
		Limit(20).
		Find(&recentSessions)

	return map[string]any{
		"period":         fmt.Sprintf("最近 %d 天", days),
		"sshSessions":    sshSessions,
		"byUser":         sshUserStats,
		"byHost":         hostStats,
		"recentSessions": recentSessions,
	}, nil
}

// executeAuditDataChanges 数据变更追踪
func executeAuditDataChanges(ctx biz.SkillContext) (any, error) {
	days := 7
	if d, ok := ctx.Params["days"].(float64); ok && d > 0 {
		days = int(d)
	}
	since := time.Now().AddDate(0, 0, -days)

	username, _ := ctx.Params["username"].(string)
	keyword, _ := ctx.Params["keyword"].(string)
	module, _ := ctx.Params["module"].(string)

	// 使用 sys_operation_log（包含所有操作记录）而非不存在的 audit_logs 表
	type ChangeLog struct {
		ID          uint      `json:"id"`
		Username    string    `json:"username"`
		RealName    string    `json:"realName"`
		Module      string    `json:"module"`
		Action      string    `json:"action"`
		Description string    `json:"description"`
		Method      string    `json:"method"`
		Path        string    `json:"path"`
		Status      int       `json:"status"`
		IP          string    `json:"ip"`
		CostTime    int64     `json:"costTime"`
		CreatedAt   time.Time `json:"createdAt"`
	}

	query := ctx.DB.Table("sys_operation_log").
		Where("created_at >= ? AND deleted_at IS NULL", since).
		Where("action NOT IN ?", []string{"查询", "GET"}) // 排除查询操作，只看变更

	if module != "" {
		query = query.Where("module LIKE ?", "%"+module+"%")
	}
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if keyword != "" {
		query = query.Where("description LIKE ? OR path LIKE ? OR module LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	var changes []ChangeLog
	query.Order("created_at DESC").Limit(50).Find(&changes)

	// 按模块统计变更
	type ModuleStat struct {
		Module string `json:"module"`
		Count  int64  `json:"count"`
	}
	var moduleStats []ModuleStat
	statQ := ctx.DB.Table("sys_operation_log").
		Select("module, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", since).
		Where("action NOT IN ?", []string{"查询", "GET"})
	statQ.Group("module").Order("count DESC").Find(&moduleStats)

	// 按用户统计变更
	type UserStat struct {
		Username string `json:"username"`
		Count    int64  `json:"count"`
	}
	var userStats []UserStat
	ctx.DB.Table("sys_operation_log").
		Select("username, COUNT(*) as count").
		Where("created_at >= ? AND deleted_at IS NULL", since).
		Where("action NOT IN ?", []string{"查询", "GET"}).
		Group("username").
		Order("count DESC").
		Limit(10).
		Find(&userStats)

	// AI 操作变更
	var aiChanges int64
	ctx.DB.Table("sys_operation_log").
		Where("created_at >= ? AND deleted_at IS NULL AND method = 'SKILL'", since).
		Count(&aiChanges)

	return map[string]any{
		"period":    fmt.Sprintf("最近 %d 天", days),
		"total":     len(changes),
		"changes":   changes,
		"byModule":  moduleStats,
		"byUser":    userStats,
		"aiChanges": aiChanges,
	}, nil
}
