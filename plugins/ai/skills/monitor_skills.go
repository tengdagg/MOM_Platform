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
}

// executeMonitorDomainStatus 域名监控状态
func executeMonitorDomainStatus(ctx biz.SkillContext) (any, error) {
	statusFilter, _ := ctx.Params["status"].(string)
	domainFilter, _ := ctx.Params["domain"].(string)

	type DomainInfo struct {
		ID           uint       `json:"id"`
		Domain       string     `json:"domain"`
		Status       string     `json:"status"`
		ResponseTime int        `json:"responseTime"`
		SSLValid     bool       `json:"sslValid"`
		SSLExpiry    *time.Time `json:"sslExpiry"`
		EnableAlert  bool       `json:"enableAlert"`
		LastCheck    *time.Time `json:"lastCheck"`
	}

	query := ctx.DB.Table("domain_monitors")
	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}
	if domainFilter != "" {
		query = query.Where("domain LIKE ?", "%"+domainFilter+"%")
	}

	var domains []DomainInfo
	if err := query.Order("status ASC, domain ASC").Find(&domains).Error; err != nil {
		return map[string]any{
			"domains": []any{},
			"total":   0,
			"message": "暂无域名监控数据",
		}, nil
	}

	normalCount := 0
	abnormalCount := 0
	sslExpiringSoon := 0
	for _, d := range domains {
		switch d.Status {
		case "normal":
			normalCount++
		case "abnormal":
			abnormalCount++
		}
		if d.SSLExpiry != nil && d.SSLExpiry.Before(time.Now().AddDate(0, 0, 30)) {
			sslExpiringSoon++
		}
	}

	return map[string]any{
		"domains":         domains,
		"total":           len(domains),
		"normal":          normalCount,
		"abnormal":        abnormalCount,
		"sslExpiringSoon": sslExpiringSoon,
	}, nil
}

// executeMonitorAlertSummary 告警汇总
func executeMonitorAlertSummary(ctx biz.SkillContext) (any, error) {
	days := 1
	if d, ok := ctx.Params["days"].(float64); ok && d > 0 {
		days = int(d)
	}
	since := time.Now().AddDate(0, 0, -days)

	var totalAlerts int64
	ctx.DB.Table("alert_logs").Where("created_at >= ?", since).Count(&totalAlerts)

	// 按类型统计
	type TypeStat struct {
		AlertType string `json:"alertType"`
		Count     int64  `json:"count"`
	}
	var typeStats []TypeStat
	ctx.DB.Table("alert_logs").
		Select("alert_type, COUNT(*) as count").
		Where("created_at >= ?", since).
		Group("alert_type").
		Order("count DESC").
		Find(&typeStats)

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

	return map[string]any{
		"period":      fmt.Sprintf("最近 %d 天", days),
		"totalAlerts": totalAlerts,
		"byType":      typeStats,
		"byStatus":    statusStats,
	}, nil
}
