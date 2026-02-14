package skills

import (
	"fmt"
	"time"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// RegisterMonitorSkills 注册监控告警 Skills
func RegisterMonitorSkills(registry *biz.ToolRegistry) {
	registry.Register(MustLoadBuiltinSkill("monitor.domain_status", executeMonitorDomainStatus))
	registry.Register(MustLoadBuiltinSkill("monitor.alert_summary", executeMonitorAlertSummary))
	registry.Register(MustLoadBuiltinSkill("monitor.alert_config", executeMonitorAlertConfig))
}

// executeMonitorDomainStatus 域名监控状态
func executeMonitorDomainStatus(ctx biz.SkillContext) (any, error) {
	statusFilter, _ := ctx.Params["status"].(string)
	domainFilter, _ := ctx.Params["domain"].(string)
	limit := 50
	if l, ok := ctx.Params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	type DomainInfo struct {
		ID           uint       `json:"id"`
		Domain       string     `json:"domain"`
		Status       string     `json:"status"`
		ResponseTime int        `json:"responseTime"`
		SSLValid     bool       `json:"sslValid"`
		SSLExpiry    *time.Time `json:"sslExpiry"`
		EnableAlert  bool       `json:"enableAlert"`
		LastCheck    *time.Time `json:"lastCheck"`
		StatusCode   int        `json:"statusCode"`
	}

	query := ctx.DB.Table("domain_monitors")
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}
	if domainFilter != "" {
		query = query.Where("domain LIKE ?", "%"+domainFilter+"%")
	}

	var domains []DomainInfo
	if err := query.Order("status ASC, domain ASC").Limit(limit).Find(&domains).Error; err != nil {
		return map[string]any{
			"domains": []any{},
			"total":   0,
			"message": "暂无域名监控数据",
		}, nil
	}

	normalCount := 0
	abnormalCount := 0
	sslExpiringSoon := 0
	slowDomains := 0
	var totalRT int
	for _, d := range domains {
		switch d.Status {
		case "normal":
			normalCount++
		case "abnormal":
			abnormalCount++
		}
		totalRT += d.ResponseTime
		if d.ResponseTime > 3000 {
			slowDomains++
		}
		if d.SSLExpiry != nil && d.SSLExpiry.Before(time.Now().AddDate(0, 0, 30)) {
			sslExpiringSoon++
		}
	}

	avgRT := 0
	if len(domains) > 0 {
		avgRT = totalRT / len(domains)
	}

	return map[string]any{
		"domains":            domains,
		"total":              len(domains),
		"normal":             normalCount,
		"abnormal":           abnormalCount,
		"sslExpiringSoon":    sslExpiringSoon,
		"slowDomains":        slowDomains,
		"avgResponseTime":    avgRT,
		"avgResponseTimeMs":  fmt.Sprintf("%dms", avgRT),
	}, nil
}

// executeMonitorAlertSummary 告警汇总
func executeMonitorAlertSummary(ctx biz.SkillContext) (any, error) {
	days := 1
	if d, ok := ctx.Params["days"].(float64); ok && d > 0 {
		days = int(d)
	}
	severity, _ := ctx.Params["severity"].(string)
	since := time.Now().AddDate(0, 0, -days)

	baseQ := ctx.DB.Table("alert_logs").Where("created_at >= ?", since)
	if severity != "" {
		baseQ = baseQ.Where("severity = ?", severity)
	}

	var totalAlerts int64
	baseQ.Count(&totalAlerts)

	// 按类型统计
	type TypeStat struct {
		AlertType string `json:"alertType"`
		Count     int64  `json:"count"`
	}
	var typeStats []TypeStat
	tq := ctx.DB.Table("alert_logs").
		Select("alert_type, COUNT(*) as count").
		Where("created_at >= ?", since)
	if severity != "" {
		tq = tq.Where("severity = ?", severity)
	}
	tq.Group("alert_type").Order("count DESC").Find(&typeStats)

	// 按状态统计
	type StatusStat struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	var statusStats []StatusStat
	ctx.DB.Table("alert_logs").
		Select("status, COUNT(*) as count").
		Where("created_at >= ?", since).
		Group("status").
		Find(&statusStats)

	// 按严重程度统计
	type SeverityStat struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}
	var severityStats []SeverityStat
	ctx.DB.Table("alert_logs").
		Select("severity, COUNT(*) as count").
		Where("created_at >= ?", since).
		Group("severity").
		Order("count DESC").
		Find(&severityStats)

	// 最近告警记录
	type RecentAlert struct {
		ID        uint      `json:"id"`
		AlertType string    `json:"alertType"`
		Severity  string    `json:"severity"`
		Status    string    `json:"status"`
		Message   string    `json:"message"`
		Target    string    `json:"target"`
		CreatedAt time.Time `json:"createdAt"`
	}
	var recentAlerts []RecentAlert
	rq := ctx.DB.Table("alert_logs").Where("created_at >= ?", since)
	if severity != "" {
		rq = rq.Where("severity = ?", severity)
	}
	rq.Order("created_at DESC").Limit(20).Find(&recentAlerts)

	// 未处理告警
	var unresolvedCount int64
	ctx.DB.Table("alert_logs").Where("created_at >= ? AND status != 'resolved'", since).Count(&unresolvedCount)

	return map[string]any{
		"period":       fmt.Sprintf("最近 %d 天", days),
		"totalAlerts":  totalAlerts,
		"unresolved":   unresolvedCount,
		"byType":       typeStats,
		"byStatus":     statusStats,
		"bySeverity":   severityStats,
		"recentAlerts": recentAlerts,
	}, nil
}

// executeMonitorAlertConfig 告警规则配置
func executeMonitorAlertConfig(ctx biz.SkillContext) (any, error) {
	action, _ := ctx.Params["action"].(string)
	if action == "" {
		return nil, fmt.Errorf("请指定操作: list/create/update/delete/enable/disable")
	}

	switch action {
	case "list":
		type AlertRuleInfo struct {
			ID              uint   `json:"id"`
			Name            string `json:"name"`
			AlertType       string `json:"alertType"`
			Enabled         bool   `json:"enabled"`
			Threshold       int    `json:"threshold"`
			DomainMonitorID uint   `json:"domainMonitorId"`
			Severity        string `json:"severity"`
			NotifyChannel   string `json:"notifyChannel"`
		}
		var rules []AlertRuleInfo
		ctx.DB.Table("alert_configs").Find(&rules)
		return map[string]any{
			"rules": rules,
			"total": len(rules),
		}, nil

	case "create":
		domain, _ := ctx.Params["domain"].(string)
		alertType, _ := ctx.Params["alert_type"].(string)
		name, _ := ctx.Params["name"].(string)
		if domain == "" || alertType == "" {
			return nil, fmt.Errorf("创建告警规则需要指定 domain 和 alert_type")
		}
		if name == "" {
			name = fmt.Sprintf("%s-%s-alert", domain, alertType)
		}

		if !isConfirmed(ctx.Params) {
			return map[string]any{
				"action":    "create",
				"name":      name,
				"domain":    domain,
				"alertType": alertType,
				"status":    "pending_confirmation",
				"warning":   fmt.Sprintf("将为域名 %s 创建 %s 类型的告警规则 [%s]，请确认", domain, alertType, name),
			}, nil
		}

		// 确认后执行创建
		// 查找域名监控 ID
		var domainID uint
		ctx.DB.Table("domain_monitors").Select("id").Where("domain = ?", domain).Scan(&domainID)
		if domainID == 0 {
			return nil, fmt.Errorf("未找到域名 %s 的监控记录，请先添加域名监控", domain)
		}

		if err := ctx.DB.Exec(`INSERT INTO alert_configs (name, alert_type, domain_monitor_id, enabled, created_at, updated_at) VALUES (?, ?, ?, 1, NOW(), NOW())`,
			name, alertType, domainID).Error; err != nil {
			return nil, fmt.Errorf("创建告警规则失败: %v", err)
		}

		return map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("✅ 已成功创建告警规则 [%s]", name),
		}, nil

	case "enable", "disable":
		ruleID, _ := ctx.Params["rule_id"].(float64)
		ruleName, _ := ctx.Params["name"].(string)
		enabled := action == "enable"

		query := ctx.DB.Table("alert_configs")
		if ruleID > 0 {
			query = query.Where("id = ?", uint(ruleID))
		} else if ruleName != "" {
			query = query.Where("name LIKE ?", "%"+ruleName+"%")
		} else {
			return nil, fmt.Errorf("请指定规则 ID (rule_id) 或规则名称 (name)")
		}

		if err := query.Update("enabled", enabled).Error; err != nil {
			return nil, fmt.Errorf("操作失败: %v", err)
		}

		statusText := "启用"
		if !enabled {
			statusText = "禁用"
		}
		return map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("✅ 告警规则已%s", statusText),
		}, nil

	case "delete":
		ruleID, _ := ctx.Params["rule_id"].(float64)
		if ruleID <= 0 {
			return nil, fmt.Errorf("删除告警规则需要指定 rule_id")
		}

		if !isConfirmed(ctx.Params) {
			return map[string]any{
				"action":  "delete",
				"ruleID":  uint(ruleID),
				"status":  "pending_confirmation",
				"warning": fmt.Sprintf("⚠️ 将删除告警规则 ID=%d，删除后将不再收到对应告警通知", int(ruleID)),
			}, nil
		}

		if err := ctx.DB.Exec("DELETE FROM alert_configs WHERE id = ?", uint(ruleID)).Error; err != nil {
			return nil, fmt.Errorf("删除失败: %v", err)
		}

		return map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("✅ 告警规则 ID=%d 已删除", int(ruleID)),
		}, nil

	default:
		return nil, fmt.Errorf("不支持的操作: %s，支持: list/create/enable/disable/delete", action)
	}
}
