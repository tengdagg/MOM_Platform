package skills

import (
	"fmt"
	"time"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// RegisterAnalysisSkills 注册综合分析 Skills
func RegisterAnalysisSkills(registry *biz.ToolRegistry) {
	registry.Register(MustLoadBuiltinSkill("analysis.infra_report", executeInfraReport))
	registry.Register(MustLoadBuiltinSkill("analysis.security_audit", executeSecurityAudit))
	registry.Register(MustLoadBuiltinSkill("analysis.capacity_plan", executeCapacityPlan))
}

// executeInfraReport 基础设施综合报告
func executeInfraReport(ctx biz.SkillContext) (any, error) {
	result := make(map[string]any)

	// 主机概况
	var totalHosts, onlineHosts, linuxHosts, windowsHosts int64
	ctx.DB.Table("hosts").Where("deleted_at IS NULL").Count(&totalHosts)
	ctx.DB.Table("hosts").Where("deleted_at IS NULL AND status = 1").Count(&onlineHosts)
	ctx.DB.Table("hosts").Where("deleted_at IS NULL AND os_type = 'linux'").Count(&linuxHosts)
	ctx.DB.Table("hosts").Where("deleted_at IS NULL AND os_type = 'windows'").Count(&windowsHosts)

	result["hosts"] = map[string]any{
		"total":   totalHosts,
		"online":  onlineHosts,
		"offline": totalHosts - onlineHosts,
		"linux":   linuxHosts,
		"windows": windowsHosts,
	}

	// K8s 集群概况
	var totalClusters, normalClusters int64
	ctx.DB.Table("k8s_clusters").Count(&totalClusters)
	ctx.DB.Table("k8s_clusters").Where("status = 1").Count(&normalClusters)
	result["kubernetes"] = map[string]any{
		"totalClusters":  totalClusters,
		"normalClusters": normalClusters,
	}

	// 云账号概况
	var totalAccounts, enabledAccounts int64
	ctx.DB.Table("cloud_accounts").Where("deleted_at IS NULL").Count(&totalAccounts)
	ctx.DB.Table("cloud_accounts").Where("deleted_at IS NULL AND status = 1").Count(&enabledAccounts)
	result["cloudAccounts"] = map[string]any{
		"total":   totalAccounts,
		"enabled": enabledAccounts,
	}

	// 域名监控概况
	var totalDomains, normalDomains, abnormalDomains int64
	ctx.DB.Table("domain_monitors").Count(&totalDomains)
	ctx.DB.Table("domain_monitors").Where("status = 'normal'").Count(&normalDomains)
	ctx.DB.Table("domain_monitors").Where("status = 'abnormal'").Count(&abnormalDomains)
	result["domainMonitor"] = map[string]any{
		"total":    totalDomains,
		"normal":   normalDomains,
		"abnormal": abnormalDomains,
	}

	// 今日操作日志
	today := time.Now().Truncate(24 * time.Hour)
	var todayOps int64
	ctx.DB.Table("sys_operation_logs").Where("created_at >= ? AND deleted_at IS NULL", today).Count(&todayOps)
	result["todayOperations"] = todayOps

	// 今日终端会话
	var todaySessions int64
	ctx.DB.Table("terminal_audit_logs").Where("created_at >= ?", today).Count(&todaySessions)
	result["todaySessions"] = todaySessions

	result["generatedAt"] = time.Now().Format("2006-01-02 15:04:05")

	return result, nil
}

// executeSecurityAudit 安全态势分析
func executeSecurityAudit(ctx biz.SkillContext) (any, error) {
	days := 7
	if d, ok := ctx.Params["days"].(float64); ok && d > 0 {
		days = int(d)
	}
	since := time.Now().AddDate(0, 0, -days)

	risks := make([]map[string]any, 0)

	// 检查离线主机
	var offlineHosts int64
	ctx.DB.Table("hosts").Where("deleted_at IS NULL AND status = 0").Count(&offlineHosts)
	if offlineHosts > 0 {
		risks = append(risks, map[string]any{
			"type":        "offline_hosts",
			"severity":    "medium",
			"description": fmt.Sprintf("有 %d 台主机处于离线状态", offlineHosts),
		})
	}

	// 检查登录失败
	var failedLogins int64
	ctx.DB.Table("sys_login_logs").Where("created_at >= ? AND deleted_at IS NULL AND status = 0", since).Count(&failedLogins)
	if failedLogins > 10 {
		risks = append(risks, map[string]any{
			"type":        "failed_logins",
			"severity":    "high",
			"description": fmt.Sprintf("最近 %d 天有 %d 次登录失败记录，请排查是否有暴力破解", days, failedLogins),
		})
	}

	// 检查 SSL 证书即将到期
	threshold := time.Now().AddDate(0, 0, 30)
	var expiringSSL int64
	ctx.DB.Table("domain_monitors").Where("ssl_expiry IS NOT NULL AND ssl_expiry < ?", threshold).Count(&expiringSSL)
	if expiringSSL > 0 {
		risks = append(risks, map[string]any{
			"type":        "ssl_expiring",
			"severity":    "medium",
			"description": fmt.Sprintf("有 %d 个域名的 SSL 证书将在 30 天内到期", expiringSSL),
		})
	}

	// 检查异常域名
	var abnormalDomains int64
	ctx.DB.Table("domain_monitors").Where("status = 'abnormal'").Count(&abnormalDomains)
	if abnormalDomains > 0 {
		risks = append(risks, map[string]any{
			"type":        "abnormal_domains",
			"severity":    "high",
			"description": fmt.Sprintf("有 %d 个域名当前状态异常", abnormalDomains),
		})
	}

	// 检查 K8s 集群异常
	var abnormalClusters int64
	ctx.DB.Table("k8s_clusters").Where("status != 1").Count(&abnormalClusters)
	if abnormalClusters > 0 {
		risks = append(risks, map[string]any{
			"type":        "k8s_cluster_issue",
			"severity":    "high",
			"description": fmt.Sprintf("有 %d 个 Kubernetes 集群状态异常", abnormalClusters),
		})
	}

	riskLevel := "low"
	for _, r := range risks {
		if r["severity"] == "high" {
			riskLevel = "high"
			break
		}
		if r["severity"] == "medium" {
			riskLevel = "medium"
		}
	}

	return map[string]any{
		"overallRisk": riskLevel,
		"riskCount":   len(risks),
		"risks":       risks,
		"period":      fmt.Sprintf("最近 %d 天", days),
	}, nil
}

// executeCapacityPlan 容量规划建议
func executeCapacityPlan(ctx biz.SkillContext) (any, error) {
	type UsageStats struct {
		AvgCPU    float64 `json:"avgCpu"`
		MaxCPU    float64 `json:"maxCpu"`
		AvgMemory float64 `json:"avgMemory"`
		MaxMemory float64 `json:"maxMemory"`
		AvgDisk   float64 `json:"avgDisk"`
		MaxDisk   float64 `json:"maxDisk"`
	}

	var stats UsageStats
	ctx.DB.Table("hosts").
		Select("AVG(cpu_usage) as avg_cpu, MAX(cpu_usage) as max_cpu, AVG(memory_usage) as avg_memory, MAX(memory_usage) as max_memory, AVG(disk_usage) as avg_disk, MAX(disk_usage) as max_disk").
		Where("deleted_at IS NULL AND status = 1").
		Scan(&stats)

	// 高使用率主机
	type HighUsageHost struct {
		Name        string  `json:"name"`
		IP          string  `json:"ip"`
		CPUUsage    float64 `json:"cpuUsage"`
		MemoryUsage float64 `json:"memoryUsage"`
		DiskUsage   float64 `json:"diskUsage"`
	}
	var highUsage []HighUsageHost
	ctx.DB.Table("hosts").
		Select("name, ip, cpu_usage, memory_usage, disk_usage").
		Where("deleted_at IS NULL AND status = 1 AND (cpu_usage > 80 OR memory_usage > 80 OR disk_usage > 80)").
		Order("cpu_usage + memory_usage + disk_usage DESC").
		Limit(10).
		Find(&highUsage)

	return map[string]any{
		"overallStats":    stats,
		"highUsageHosts":  highUsage,
		"highUsageCount":  len(highUsage),
		"recommendations": generateCapacityRecommendations(stats, len(highUsage)),
	}, nil
}

func generateCapacityRecommendations(stats struct {
	AvgCPU    float64 `json:"avgCpu"`
	MaxCPU    float64 `json:"maxCpu"`
	AvgMemory float64 `json:"avgMemory"`
	MaxMemory float64 `json:"maxMemory"`
	AvgDisk   float64 `json:"avgDisk"`
	MaxDisk   float64 `json:"maxDisk"`
}, highUsageCount int) []string {
	var recs []string
	if stats.AvgCPU > 70 {
		recs = append(recs, "整体 CPU 使用率偏高，建议考虑扩展计算资源或优化高负载服务")
	}
	if stats.AvgMemory > 70 {
		recs = append(recs, "整体内存使用率偏高，建议增加内存或排查内存泄漏")
	}
	if stats.AvgDisk > 70 {
		recs = append(recs, "整体磁盘使用率偏高，建议清理日志和临时文件，或扩容磁盘")
	}
	if highUsageCount > 3 {
		recs = append(recs, fmt.Sprintf("有 %d 台主机资源使用率超过 80%%，需重点关注", highUsageCount))
	}
	if len(recs) == 0 {
		recs = append(recs, "当前资源使用情况良好，暂无需扩容")
	}
	return recs
}
