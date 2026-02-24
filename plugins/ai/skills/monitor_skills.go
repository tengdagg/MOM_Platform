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
	registry.Register(MustLoadBuiltinSkill("monitor.domain_manage", executeMonitorDomainManage))
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
		"domains":           domains,
		"total":             len(domains),
		"normal":            normalCount,
		"abnormal":          abnormalCount,
		"sslExpiringSoon":   sslExpiringSoon,
		"slowDomains":       slowDomains,
		"avgResponseTime":   avgRT,
		"avgResponseTimeMs": fmt.Sprintf("%dms", avgRT),
	}, nil
}

// executeMonitorAlertSummary 告警汇总
func executeMonitorAlertSummary(ctx biz.SkillContext) (any, error) {
	days := 1
	if d, ok := ctx.Params["days"].(float64); ok && d > 0 {
		days = int(d)
	}

	since := time.Now().AddDate(0, 0, -days)

	baseQ := ctx.DB.Table("alert_logs").Where("created_at >= ?", since)

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

	// 按渠道类型统计
	type ChannelStat struct {
		ChannelType string `json:"channelType"`
		Count       int64  `json:"count"`
	}
	var channelStats []ChannelStat
	ctx.DB.Table("alert_logs").
		Select("channel_type, COUNT(*) as count").
		Where("created_at >= ?", since).
		Group("channel_type").
		Order("count DESC").
		Find(&channelStats)

	// 最近告警记录
	type RecentAlert struct {
		ID          uint      `json:"id"`
		AlertType   string    `json:"alertType"`
		Domain      string    `json:"domain"`
		Status      string    `json:"status"`
		Message     string    `json:"message"`
		ChannelType string    `json:"channelType"`
		CreatedAt   time.Time `json:"createdAt"`
	}
	var recentAlerts []RecentAlert
	ctx.DB.Table("alert_logs").Where("created_at >= ?", since).
		Order("created_at DESC").Limit(20).Find(&recentAlerts)

	// 未处理告警
	var unresolvedCount int64
	ctx.DB.Table("alert_logs").Where("created_at >= ? AND status != 'resolved'", since).Count(&unresolvedCount)

	return map[string]any{
		"period":       fmt.Sprintf("最近 %d 天", days),
		"totalAlerts":  totalAlerts,
		"unresolved":   unresolvedCount,
		"byType":       typeStats,
		"byStatus":     statusStats,
		"byChannel":    channelStats,
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

// executeMonitorDomainManage 域名监控管理（创建/修改/删除）
func executeMonitorDomainManage(ctx biz.SkillContext) (any, error) {
	action, _ := ctx.Params["action"].(string)
	if action == "" {
		return nil, fmt.Errorf("请指定操作: create/update/delete")
	}

	switch action {
	case "create":
		domain, _ := ctx.Params["domain"].(string)
		if domain == "" {
			return nil, fmt.Errorf("创建域名监控需要指定 domain 参数")
		}

		// 检查是否已存在
		var existCount int64
		ctx.DB.Table("domain_monitors").Where("domain = ?", domain).Count(&existCount)
		if existCount > 0 {
			return nil, fmt.Errorf("域名 %s 已在监控中，无需重复添加", domain)
		}

		// 默认参数
		checkInterval := 300
		if v, ok := ctx.Params["check_interval"].(float64); ok && v > 0 {
			checkInterval = int(v)
		}
		enableSSL := true
		if v, ok := ctx.Params["enable_ssl"].(bool); ok {
			enableSSL = v
		}
		enableAlert := false
		if v, ok := ctx.Params["enable_alert"].(bool); ok {
			enableAlert = v
		}
		responseThreshold := 1000
		if v, ok := ctx.Params["response_threshold"].(float64); ok && v > 0 {
			responseThreshold = int(v)
		}
		sslExpiryDays := 30
		if v, ok := ctx.Params["ssl_expiry_days"].(float64); ok && v > 0 {
			sslExpiryDays = int(v)
		}

		if !isConfirmed(ctx.Params) {
			return map[string]any{
				"action":            "create",
				"domain":            domain,
				"checkInterval":     checkInterval,
				"enableSSL":         enableSSL,
				"enableAlert":       enableAlert,
				"responseThreshold": responseThreshold,
				"sslExpiryDays":     sslExpiryDays,
				"status":            "pending_confirmation",
				"warning":           fmt.Sprintf("即将创建域名监控 [%s]，检查间隔 %d 秒，SSL检查: %v，告警: %v，请确认", domain, checkInterval, enableSSL, enableAlert),
			}, nil
		}

		// 确认后创建
		now := time.Now()
		nextCheck := now.Add(time.Duration(checkInterval) * time.Second)
		if err := ctx.DB.Exec(
			`INSERT INTO domain_monitors (domain, status, check_interval, enable_ssl, enable_alert, response_threshold, ssl_expiry_days, next_check, created_at, updated_at) VALUES (?, 'unknown', ?, ?, ?, ?, ?, ?, ?, ?)`,
			domain, checkInterval, enableSSL, enableAlert, responseThreshold, sslExpiryDays, nextCheck, now, now,
		).Error; err != nil {
			return nil, fmt.Errorf("创建域名监控失败: %v", err)
		}

		return map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("✅ 已成功创建域名监控 [%s]，系统将在 %d 秒后开始首次检查", domain, checkInterval),
		}, nil

	case "update":
		monitorID, _ := ctx.Params["id"].(float64)
		domain, _ := ctx.Params["domain"].(string)

		// 查找目标
		var targetID uint
		var targetDomain string
		if monitorID > 0 {
			ctx.DB.Table("domain_monitors").Select("id, domain").Where("id = ?", uint(monitorID)).Row().Scan(&targetID, &targetDomain)
		} else if domain != "" {
			ctx.DB.Table("domain_monitors").Select("id, domain").Where("domain = ?", domain).Row().Scan(&targetID, &targetDomain)
		} else {
			return nil, fmt.Errorf("请提供域名监控 ID 或域名")
		}
		if targetID == 0 {
			return nil, fmt.Errorf("未找到域名监控记录")
		}

		// 构建更新字段
		updates := map[string]any{"updated_at": time.Now()}
		changeDesc := []string{}
		if v, ok := ctx.Params["check_interval"].(float64); ok && v > 0 {
			updates["check_interval"] = int(v)
			changeDesc = append(changeDesc, fmt.Sprintf("检查间隔→%d秒", int(v)))
		}
		if v, ok := ctx.Params["enable_ssl"].(bool); ok {
			updates["enable_ssl"] = v
			changeDesc = append(changeDesc, fmt.Sprintf("SSL检查→%v", v))
		}
		if v, ok := ctx.Params["enable_alert"].(bool); ok {
			updates["enable_alert"] = v
			changeDesc = append(changeDesc, fmt.Sprintf("告警→%v", v))
		}
		if v, ok := ctx.Params["response_threshold"].(float64); ok && v > 0 {
			updates["response_threshold"] = int(v)
			changeDesc = append(changeDesc, fmt.Sprintf("响应阈值→%dms", int(v)))
		}
		if v, ok := ctx.Params["ssl_expiry_days"].(float64); ok && v > 0 {
			updates["ssl_expiry_days"] = int(v)
			changeDesc = append(changeDesc, fmt.Sprintf("SSL过期提醒→%d天", int(v)))
		}

		if len(changeDesc) == 0 {
			return nil, fmt.Errorf("请指定要修改的参数")
		}

		if !isConfirmed(ctx.Params) {
			return map[string]any{
				"action":  "update",
				"id":      targetID,
				"domain":  targetDomain,
				"changes": changeDesc,
				"status":  "pending_confirmation",
				"warning": fmt.Sprintf("即将修改域名 [%s] 的监控配置: %v，请确认", targetDomain, changeDesc),
			}, nil
		}

		if err := ctx.DB.Table("domain_monitors").Where("id = ?", targetID).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("更新失败: %v", err)
		}

		return map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("✅ 已更新域名 [%s] 的监控配置", targetDomain),
		}, nil

	case "delete":
		monitorID, _ := ctx.Params["id"].(float64)
		domain, _ := ctx.Params["domain"].(string)

		var targetID uint
		var targetDomain string
		if monitorID > 0 {
			ctx.DB.Table("domain_monitors").Select("id, domain").Where("id = ?", uint(monitorID)).Row().Scan(&targetID, &targetDomain)
		} else if domain != "" {
			ctx.DB.Table("domain_monitors").Select("id, domain").Where("domain = ?", domain).Row().Scan(&targetID, &targetDomain)
		} else {
			return nil, fmt.Errorf("请提供域名监控 ID 或域名")
		}
		if targetID == 0 {
			return nil, fmt.Errorf("未找到域名监控记录")
		}

		if !isConfirmed(ctx.Params) {
			return map[string]any{
				"action":  "delete",
				"id":      targetID,
				"domain":  targetDomain,
				"status":  "pending_confirmation",
				"warning": fmt.Sprintf("⚠️ 即将删除域名 [%s] 的监控，关联的告警配置也将失效，此操作不可恢复，请确认", targetDomain),
			}, nil
		}

		// 删除关联的告警配置
		ctx.DB.Exec("DELETE FROM alert_configs WHERE domain_monitor_id = ?", targetID)
		// 删除域名监控
		if err := ctx.DB.Exec("DELETE FROM domain_monitors WHERE id = ?", targetID).Error; err != nil {
			return nil, fmt.Errorf("删除失败: %v", err)
		}

		return map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("✅ 已删除域名 [%s] 的监控及关联告警配置", targetDomain),
		}, nil

	default:
		return nil, fmt.Errorf("不支持的操作: %s，支持: create/update/delete", action)
	}
}
