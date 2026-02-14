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

	// 主机资源使用平均值
	type ResourceAvg struct {
		AvgCPU    float64 `json:"avgCpu"`
		AvgMemory float64 `json:"avgMemory"`
		AvgDisk   float64 `json:"avgDisk"`
	}
	var resAvg ResourceAvg
	ctx.DB.Table("hosts").
		Select("AVG(cpu_usage) as avg_cpu, AVG(memory_usage) as avg_memory, AVG(disk_usage) as avg_disk").
		Where("deleted_at IS NULL AND status = 1").
		Scan(&resAvg)

	result["hosts"] = map[string]any{
		"total":         totalHosts,
		"online":        onlineHosts,
		"offline":       totalHosts - onlineHosts,
		"linux":         linuxHosts,
		"windows":       windowsHosts,
		"avgCpuUsage":   fmt.Sprintf("%.1f%%", resAvg.AvgCPU),
		"avgMemUsage":   fmt.Sprintf("%.1f%%", resAvg.AvgMemory),
		"avgDiskUsage":  fmt.Sprintf("%.1f%%", resAvg.AvgDisk),
	}

	// K8s 集群概况
	var totalClusters, normalClusters int64
	ctx.DB.Table("k8s_clusters").Count(&totalClusters)
	ctx.DB.Table("k8s_clusters").Where("status = 1").Count(&normalClusters)
	result["kubernetes"] = map[string]any{
		"totalClusters":   totalClusters,
		"normalClusters":  normalClusters,
		"abnormalClusters": totalClusters - normalClusters,
	}

	// 云账号概况
	var totalAccounts, enabledAccounts int64
	ctx.DB.Table("cloud_accounts").Where("deleted_at IS NULL").Count(&totalAccounts)
	ctx.DB.Table("cloud_accounts").Where("deleted_at IS NULL AND status = 1").Count(&enabledAccounts)

	// 按云厂商统计
	type ProviderStat struct {
		Provider string `json:"provider"`
		Count    int64  `json:"count"`
	}
	var providerStats []ProviderStat
	ctx.DB.Table("cloud_accounts").
		Select("provider, COUNT(*) as count").
		Where("deleted_at IS NULL").
		Group("provider").
		Find(&providerStats)

	result["cloudAccounts"] = map[string]any{
		"total":      totalAccounts,
		"enabled":    enabledAccounts,
		"byProvider": providerStats,
	}

	// 域名监控概况
	var totalDomains, normalDomains, abnormalDomains int64
	ctx.DB.Table("domain_monitors").Count(&totalDomains)
	ctx.DB.Table("domain_monitors").Where("status = 'normal'").Count(&normalDomains)
	ctx.DB.Table("domain_monitors").Where("status = 'abnormal'").Count(&abnormalDomains)

	// SSL 即将到期
	threshold := time.Now().AddDate(0, 0, 30)
	var sslExpiring int64
	ctx.DB.Table("domain_monitors").Where("ssl_expiry IS NOT NULL AND ssl_expiry < ?", threshold).Count(&sslExpiring)

	result["domainMonitor"] = map[string]any{
		"total":       totalDomains,
		"normal":      normalDomains,
		"abnormal":    abnormalDomains,
		"sslExpiring": sslExpiring,
	}

	// 今日操作日志（使用正确的表名）
	today := time.Now().Truncate(24 * time.Hour)
	var todayOps int64
	ctx.DB.Table("sys_operation_log").Where("created_at >= ? AND deleted_at IS NULL", today).Count(&todayOps)

	// AI 操作数量
	var todayAIOps int64
	ctx.DB.Table("sys_operation_log").Where("created_at >= ? AND deleted_at IS NULL AND method = 'SKILL'", today).Count(&todayAIOps)

	result["todayOperations"] = todayOps
	result["todayAIOperations"] = todayAIOps

	// 今日终端会话（使用正确的表名）
	var todaySessions int64
	ctx.DB.Table("ssh_terminal_sessions").Where("created_at >= ?", today).Count(&todaySessions)
	result["todaySessions"] = todaySessions

	// 今日登录
	var todayLogins int64
	ctx.DB.Table("sys_login_log").Where("created_at >= ? AND deleted_at IS NULL", today).Count(&todayLogins)
	var todayFailedLogins int64
	ctx.DB.Table("sys_login_log").Where("created_at >= ? AND deleted_at IS NULL AND status = 0", today).Count(&todayFailedLogins)
	result["todayLogins"] = todayLogins
	result["todayFailedLogins"] = todayFailedLogins

	// 用户概况
	var totalUsers, activeUsers int64
	ctx.DB.Table("sys_user").Where("deleted_at IS NULL").Count(&totalUsers)
	ctx.DB.Table("sys_user").Where("deleted_at IS NULL AND status = 1").Count(&activeUsers)
	result["users"] = map[string]any{
		"total":  totalUsers,
		"active": activeUsers,
	}

	// AI Skills 概况
	var totalSkills, enabledSkills int64
	ctx.DB.Table("skill_definitions").Count(&totalSkills)
	ctx.DB.Table("skill_definitions").Where("is_enabled = 1").Count(&enabledSkills)
	result["aiSkills"] = map[string]any{
		"total":   totalSkills,
		"enabled": enabledSkills,
	}

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

	// 1. 检查离线主机
	var offlineHosts int64
	ctx.DB.Table("hosts").Where("deleted_at IS NULL AND status = 0").Count(&offlineHosts)
	if offlineHosts > 0 {
		// 获取离线主机列表
		type OfflineHost struct {
			Name string `json:"name"`
			IP   string `json:"ip"`
		}
		var offList []OfflineHost
		ctx.DB.Table("hosts").Select("name, ip").Where("deleted_at IS NULL AND status = 0").Limit(10).Find(&offList)
		risks = append(risks, map[string]any{
			"type":        "offline_hosts",
			"severity":    "medium",
			"description": fmt.Sprintf("有 %d 台主机处于离线状态", offlineHosts),
			"details":     offList,
		})
	}

	// 2. 检查登录失败（使用正确的表名）
	var failedLogins int64
	ctx.DB.Table("sys_login_log").Where("created_at >= ? AND deleted_at IS NULL AND status = 0", since).Count(&failedLogins)
	if failedLogins > 10 {
		// 获取失败最多的用户
		type FailUser struct {
			Username string `json:"username"`
			Count    int64  `json:"count"`
		}
		var failUsers []FailUser
		ctx.DB.Table("sys_login_log").
			Select("username, COUNT(*) as count").
			Where("created_at >= ? AND deleted_at IS NULL AND status = 0", since).
			Group("username").Having("count > 3").Order("count DESC").Limit(5).Find(&failUsers)

		risks = append(risks, map[string]any{
			"type":        "failed_logins",
			"severity":    "high",
			"description": fmt.Sprintf("最近 %d 天有 %d 次登录失败记录，请排查是否有暴力破解", days, failedLogins),
			"details":     failUsers,
		})
	}

	// 3. 检查 SSL 证书即将到期
	threshold := time.Now().AddDate(0, 0, 30)
	var expiringSSL int64
	ctx.DB.Table("domain_monitors").Where("ssl_expiry IS NOT NULL AND ssl_expiry < ?", threshold).Count(&expiringSSL)
	if expiringSSL > 0 {
		type SSLDomain struct {
			Domain    string     `json:"domain"`
			SSLExpiry *time.Time `json:"sslExpiry"`
		}
		var sslDomains []SSLDomain
		ctx.DB.Table("domain_monitors").Select("domain, ssl_expiry").
			Where("ssl_expiry IS NOT NULL AND ssl_expiry < ?", threshold).
			Order("ssl_expiry ASC").Limit(10).Find(&sslDomains)

		risks = append(risks, map[string]any{
			"type":        "ssl_expiring",
			"severity":    "medium",
			"description": fmt.Sprintf("有 %d 个域名的 SSL 证书将在 30 天内到期", expiringSSL),
			"details":     sslDomains,
		})
	}

	// 4. 检查异常域名
	var abnormalDomains int64
	ctx.DB.Table("domain_monitors").Where("status = 'abnormal'").Count(&abnormalDomains)
	if abnormalDomains > 0 {
		type AbnormalDomain struct {
			Domain string `json:"domain"`
			Status string `json:"status"`
		}
		var abDomains []AbnormalDomain
		ctx.DB.Table("domain_monitors").Select("domain, status").Where("status = 'abnormal'").Limit(10).Find(&abDomains)

		risks = append(risks, map[string]any{
			"type":        "abnormal_domains",
			"severity":    "high",
			"description": fmt.Sprintf("有 %d 个域名当前状态异常", abnormalDomains),
			"details":     abDomains,
		})
	}

	// 5. 检查 K8s 集群异常
	var abnormalClusters int64
	ctx.DB.Table("k8s_clusters").Where("status != 1").Count(&abnormalClusters)
	if abnormalClusters > 0 {
		type AbnCluster struct {
			Name   string `json:"name"`
			Status int    `json:"status"`
		}
		var abClusters []AbnCluster
		ctx.DB.Table("k8s_clusters").Select("name, status").Where("status != 1").Find(&abClusters)

		risks = append(risks, map[string]any{
			"type":        "k8s_cluster_issue",
			"severity":    "high",
			"description": fmt.Sprintf("有 %d 个 Kubernetes 集群状态异常", abnormalClusters),
			"details":     abClusters,
		})
	}

	// 6. 检查高负载主机
	var highLoadHosts int64
	ctx.DB.Table("hosts").Where("deleted_at IS NULL AND status = 1 AND (cpu_usage > 90 OR memory_usage > 90 OR disk_usage > 95)").Count(&highLoadHosts)
	if highLoadHosts > 0 {
		risks = append(risks, map[string]any{
			"type":        "high_load_hosts",
			"severity":    "medium",
			"description": fmt.Sprintf("有 %d 台主机资源使用率过高（CPU>90%% / 内存>90%% / 磁盘>95%%）", highLoadHosts),
		})
	}

	// 7. 检查高风险 AI 操作
	var highRiskAIOps int64
	ctx.DB.Table("sys_operation_log").
		Where("created_at >= ? AND deleted_at IS NULL AND method = 'SKILL' AND action LIKE '%critical%' OR action LIKE '%high%'", since).
		Count(&highRiskAIOps)
	if highRiskAIOps > 0 {
		risks = append(risks, map[string]any{
			"type":        "high_risk_ai_ops",
			"severity":    "medium",
			"description": fmt.Sprintf("最近 %d 天通过 AI 执行了 %d 次高风险操作", days, highRiskAIOps),
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
		"analyzedAt":  time.Now().Format("2006-01-02 15:04:05"),
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

	// 高使用率主机（细分维度）
	type HighUsageHost struct {
		Name        string  `json:"name"`
		IP          string  `json:"ip"`
		CPUUsage    float64 `json:"cpuUsage"`
		MemoryUsage float64 `json:"memoryUsage"`
		DiskUsage   float64 `json:"diskUsage"`
		CPUCores    int     `json:"cpuCores"`
		OS          string  `json:"os"`
	}
	var highCPU []HighUsageHost
	ctx.DB.Table("hosts").
		Select("name, ip, cpu_usage, memory_usage, disk_usage, cpu_cores, os").
		Where("deleted_at IS NULL AND status = 1 AND cpu_usage > 80").
		Order("cpu_usage DESC").
		Limit(10).
		Find(&highCPU)

	var highMemory []HighUsageHost
	ctx.DB.Table("hosts").
		Select("name, ip, cpu_usage, memory_usage, disk_usage, cpu_cores, os").
		Where("deleted_at IS NULL AND status = 1 AND memory_usage > 80").
		Order("memory_usage DESC").
		Limit(10).
		Find(&highMemory)

	var highDisk []HighUsageHost
	ctx.DB.Table("hosts").
		Select("name, ip, cpu_usage, memory_usage, disk_usage, cpu_cores, os").
		Where("deleted_at IS NULL AND status = 1 AND disk_usage > 80").
		Order("disk_usage DESC").
		Limit(10).
		Find(&highDisk)

	// 统计分组资源情况
	type GroupStats struct {
		GroupName string  `json:"groupName"`
		Count     int64   `json:"count"`
		AvgCPU    float64 `json:"avgCpu"`
		AvgMemory float64 `json:"avgMemory"`
		AvgDisk   float64 `json:"avgDisk"`
	}
	var groupStats []GroupStats
	ctx.DB.Table("hosts").
		Select("COALESCE(asset_groups.name, '未分组') as group_name, COUNT(*) as count, AVG(hosts.cpu_usage) as avg_cpu, AVG(hosts.memory_usage) as avg_memory, AVG(hosts.disk_usage) as avg_disk").
		Joins("LEFT JOIN asset_groups ON hosts.group_id = asset_groups.id").
		Where("hosts.deleted_at IS NULL AND hosts.status = 1").
		Group("asset_groups.name").
		Order("avg_cpu DESC").
		Find(&groupStats)

	recs := generateCapacityRecommendations(stats, len(highCPU)+len(highMemory)+len(highDisk))

	return map[string]any{
		"overallStats":     stats,
		"highCPUHosts":     highCPU,
		"highMemoryHosts":  highMemory,
		"highDiskHosts":    highDisk,
		"highCPUCount":     len(highCPU),
		"highMemoryCount":  len(highMemory),
		"highDiskCount":    len(highDisk),
		"groupStats":       groupStats,
		"recommendations":  recs,
		"analyzedAt":       time.Now().Format("2006-01-02 15:04:05"),
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
		recs = append(recs, fmt.Sprintf("整体 CPU 平均使用率 %.1f%% 偏高，建议考虑扩展计算资源或优化高负载服务", stats.AvgCPU))
	}
	if stats.MaxCPU > 95 {
		recs = append(recs, fmt.Sprintf("存在主机 CPU 使用率高达 %.1f%%，可能影响服务稳定性，建议紧急处理", stats.MaxCPU))
	}
	if stats.AvgMemory > 70 {
		recs = append(recs, fmt.Sprintf("整体内存平均使用率 %.1f%% 偏高，建议增加内存或排查内存泄漏", stats.AvgMemory))
	}
	if stats.AvgDisk > 70 {
		recs = append(recs, fmt.Sprintf("整体磁盘平均使用率 %.1f%% 偏高，建议清理日志和临时文件，或扩容磁盘", stats.AvgDisk))
	}
	if stats.MaxDisk > 90 {
		recs = append(recs, fmt.Sprintf("存在主机磁盘使用率高达 %.1f%%，磁盘空间即将耗尽，建议立即清理或扩容", stats.MaxDisk))
	}
	if highUsageCount > 3 {
		recs = append(recs, fmt.Sprintf("有 %d 台主机资源使用率超过 80%%，需重点关注并制定扩容计划", highUsageCount))
	}
	if len(recs) == 0 {
		recs = append(recs, "当前资源使用情况良好，暂无需扩容")
	}
	return recs
}
