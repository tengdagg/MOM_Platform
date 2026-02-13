package skills

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// MonitorDomainStatusSkill 域名监控状态
type MonitorDomainStatusSkill struct{}

func (s *MonitorDomainStatusSkill) Name() string        { return "monitor.domain_status" }
func (s *MonitorDomainStatusSkill) Description() string {
	return "查询域名监控状态，列出所有被监控的域名及其当前状态（正常/异常/暂停），响应时间，SSL 证书有效期等"
}
func (s *MonitorDomainStatusSkill) RiskLevel() string   { return "low" }
func (s *MonitorDomainStatusSkill) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"status": {"type": "string", "description": "状态筛选: normal / abnormal / paused"},
			"domain": {"type": "string", "description": "域名关键词搜索"}
		}
	}`)
}

func (s *MonitorDomainStatusSkill) Execute(ctx biz.SkillContext) (any, error) {
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

// MonitorAlertSummarySkill 告警汇总
type MonitorAlertSummarySkill struct{}

func (s *MonitorAlertSummarySkill) Name() string        { return "monitor.alert_summary" }
func (s *MonitorAlertSummarySkill) Description() string {
	return "告警日志汇总分析，按时间范围统计告警数量、类型分布和严重程度"
}
func (s *MonitorAlertSummarySkill) RiskLevel() string   { return "low" }
func (s *MonitorAlertSummarySkill) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"days": {"type": "integer", "description": "统计最近 N 天，默认 1", "default": 1}
		}
	}`)
}

func (s *MonitorAlertSummarySkill) Execute(ctx biz.SkillContext) (any, error) {
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

// RegisterMonitorSkills 注册监控告警 Skills
func RegisterMonitorSkills(registry *biz.ToolRegistry) {
	registry.Register(&MonitorDomainStatusSkill{})
	registry.Register(&MonitorAlertSummarySkill{})
}
