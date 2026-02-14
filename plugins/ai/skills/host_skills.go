package skills

import (
	"fmt"
	"strings"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// RegisterHostSkills 注册主机管理 Skills
func RegisterHostSkills(registry *biz.ToolRegistry) {
	registry.Register(MustLoadBuiltinSkill("host.list", executeHostList))
	registry.Register(MustLoadBuiltinSkill("host.detail", executeHostDetail))
	registry.Register(MustLoadBuiltinSkill("host.analyze", executeHostAnalyze))
}

// executeHostList 查询主机列表
func executeHostList(ctx biz.SkillContext) (any, error) {
	keyword, _ := ctx.Params["keyword"].(string)
	osType, _ := ctx.Params["os_type"].(string)
	groupName, _ := ctx.Params["group_name"].(string)
	limit := 20
	if l, ok := ctx.Params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	type HostResult struct {
		ID          uint    `json:"id"`
		Name        string  `json:"name"`
		IP          string  `json:"ip"`
		Port        int     `json:"port"`
		OS          string  `json:"os"`
		OSType      string  `json:"osType"`
		Status      int     `json:"status"`
		CPUCores    int     `json:"cpuCores"`
		CPUUsage    float64 `json:"cpuUsage"`
		MemoryUsage float64 `json:"memoryUsage"`
		DiskUsage   float64 `json:"diskUsage"`
		GroupName   string  `json:"groupName"`
	}

	query := ctx.DB.Table("hosts").
		Select("hosts.id, hosts.name, hosts.ip, hosts.port, hosts.os, hosts.os_type, hosts.status, hosts.cpu_cores, hosts.cpu_usage, hosts.memory_usage, hosts.disk_usage, COALESCE(asset_groups.name, '') as group_name").
		Joins("LEFT JOIN asset_groups ON hosts.group_id = asset_groups.id").
		Where("hosts.deleted_at IS NULL")

	if keyword != "" {
		query = query.Where("(hosts.name LIKE ? OR hosts.ip LIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
	}
	if osType != "" {
		query = query.Where("hosts.os_type = ?", osType)
	}
	if groupName != "" {
		query = query.Where("asset_groups.name LIKE ?", "%"+groupName+"%")
	}
	if statusVal, ok := ctx.Params["status"].(float64); ok {
		query = query.Where("hosts.status = ?", int(statusVal))
	}

	var hosts []HostResult
	if err := query.Order("hosts.id DESC").Limit(limit).Find(&hosts).Error; err != nil {
		return nil, fmt.Errorf("查询主机失败: %v", err)
	}

	// 统计
	var total int64
	ctx.DB.Table("hosts").Where("deleted_at IS NULL").Count(&total)
	var onlineCount int64
	ctx.DB.Table("hosts").Where("deleted_at IS NULL AND status = 1").Count(&onlineCount)

	return map[string]any{
		"hosts":       hosts,
		"total":       total,
		"online":      onlineCount,
		"offline":     total - onlineCount,
		"resultCount": len(hosts),
	}, nil
}

// executeHostDetail 查询主机详情
func executeHostDetail(ctx biz.SkillContext) (any, error) {
	ip, _ := ctx.Params["ip"].(string)
	name, _ := ctx.Params["name"].(string)
	hostID, _ := ctx.Params["id"].(float64)

	type HostDetail struct {
		ID          uint    `json:"id"`
		Name        string  `json:"name"`
		IP          string  `json:"ip"`
		Port        int     `json:"port"`
		OS          string  `json:"os"`
		OSType      string  `json:"osType"`
		Kernel      string  `json:"kernel"`
		Arch        string  `json:"arch"`
		Hostname    string  `json:"hostname"`
		Status      int     `json:"status"`
		CPUCores    int     `json:"cpuCores"`
		CPUUsage    float64 `json:"cpuUsage"`
		MemoryTotal uint64  `json:"memoryTotal"`
		MemoryUsed  uint64  `json:"memoryUsed"`
		MemoryUsage float64 `json:"memoryUsage"`
		DiskTotal   uint64  `json:"diskTotal"`
		DiskUsed    uint64  `json:"diskUsed"`
		DiskUsage   float64 `json:"diskUsage"`
		Uptime      string  `json:"uptime"`
		Tags        string  `json:"tags"`
		Description string  `json:"description"`
	}

	query := ctx.DB.Table("hosts").Where("deleted_at IS NULL")
	if hostID > 0 {
		query = query.Where("id = ?", int(hostID))
	} else if ip != "" {
		query = query.Where("ip = ?", ip)
	} else if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	} else {
		return nil, fmt.Errorf("请提供主机 IP、名称或 ID")
	}

	var host HostDetail
	if err := query.First(&host).Error; err != nil {
		return nil, fmt.Errorf("主机不存在")
	}

	return host, nil
}

// executeHostAnalyze 分析主机健康状态
func executeHostAnalyze(ctx biz.SkillContext) (any, error) {
	metric, _ := ctx.Params["metric"].(string)
	if metric == "" {
		metric = "all"
	}
	threshold := 80.0
	if t, ok := ctx.Params["threshold"].(float64); ok && t > 0 {
		threshold = t
	}

	type AlertHost struct {
		Name   string  `json:"name"`
		IP     string  `json:"ip"`
		Metric string  `json:"metric"`
		Usage  float64 `json:"usage"`
	}

	var alerts []AlertHost

	conditions := []struct {
		metric string
		column string
	}{
		{"cpu", "cpu_usage"},
		{"memory", "memory_usage"},
		{"disk", "disk_usage"},
	}

	for _, cond := range conditions {
		if metric != "all" && metric != cond.metric {
			continue
		}
		var hosts []struct {
			Name  string  `json:"name"`
			IP    string  `json:"ip"`
			Usage float64 `json:"usage"`
		}
		ctx.DB.Table("hosts").
			Select("name, ip, "+cond.column+" as `usage`").
			Where("deleted_at IS NULL AND status = 1 AND "+cond.column+" > ?", threshold).
			Order(cond.column + " DESC").
			Limit(20).
			Find(&hosts)

		for _, h := range hosts {
			alerts = append(alerts, AlertHost{
				Name:   h.Name,
				IP:     h.IP,
				Metric: strings.ToUpper(cond.metric),
				Usage:  h.Usage,
			})
		}
	}

	return map[string]any{
		"threshold":  threshold,
		"alertCount": len(alerts),
		"alerts":     alerts,
	}, nil
}
