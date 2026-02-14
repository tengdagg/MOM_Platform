package skills

import (
	"fmt"
	"strings"
	"time"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// RegisterHostSkills 注册主机管理 Skills
func RegisterHostSkills(registry *biz.ToolRegistry) {
	registry.Register(MustLoadBuiltinSkill("host.list", executeHostList))
	registry.Register(MustLoadBuiltinSkill("host.detail", executeHostDetail))
	registry.Register(MustLoadBuiltinSkill("host.analyze", executeHostAnalyze))
	registry.Register(MustLoadBuiltinSkill("host.collect", executeHostCollect))
	registry.Register(MustLoadBuiltinSkill("host.exec_command", executeHostExecCommand))
	registry.Register(MustLoadBuiltinSkill("host.file_manage", executeHostFileManage))
}

// executeHostList 查询主机列表
func executeHostList(ctx biz.SkillContext) (any, error) {
	keyword, _ := ctx.Params["keyword"].(string)
	osType, _ := ctx.Params["os_type"].(string)
	groupName, _ := ctx.Params["group_name"].(string)
	tags, _ := ctx.Params["tags"].(string)
	sortBy, _ := ctx.Params["sort_by"].(string)
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
		Tags        string  `json:"tags"`
		Uptime      string  `json:"uptime"`
	}

	query := ctx.DB.Table("hosts").
		Select("hosts.id, hosts.name, hosts.ip, hosts.port, hosts.os, hosts.os_type, hosts.status, hosts.cpu_cores, hosts.cpu_usage, hosts.memory_usage, hosts.disk_usage, COALESCE(asset_groups.name, '') as group_name, hosts.tags, hosts.uptime").
		Joins("LEFT JOIN asset_groups ON hosts.group_id = asset_groups.id").
		Where("hosts.deleted_at IS NULL")

	if keyword != "" {
		query = query.Where("(hosts.name LIKE ? OR hosts.ip LIKE ? OR hosts.description LIKE ?)", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if osType != "" {
		query = query.Where("hosts.os_type = ?", osType)
	}
	if groupName != "" {
		query = query.Where("asset_groups.name LIKE ?", "%"+groupName+"%")
	}
	if tags != "" {
		query = query.Where("hosts.tags LIKE ?", "%"+tags+"%")
	}
	if statusVal, ok := ctx.Params["status"].(float64); ok {
		query = query.Where("hosts.status = ?", int(statusVal))
	}

	// 排序
	orderClause := "hosts.id DESC"
	switch sortBy {
	case "cpu":
		orderClause = "hosts.cpu_usage DESC"
	case "memory":
		orderClause = "hosts.memory_usage DESC"
	case "disk":
		orderClause = "hosts.disk_usage DESC"
	case "name":
		orderClause = "hosts.name ASC"
	case "ip":
		orderClause = "hosts.ip ASC"
	}

	var hosts []HostResult
	if err := query.Order(orderClause).Limit(limit).Find(&hosts).Error; err != nil {
		return nil, fmt.Errorf("查询主机失败: %v", err)
	}

	// 统计
	var total int64
	ctx.DB.Table("hosts").Where("deleted_at IS NULL").Count(&total)
	var onlineCount int64
	ctx.DB.Table("hosts").Where("deleted_at IS NULL AND status = 1").Count(&onlineCount)
	var linuxCount int64
	ctx.DB.Table("hosts").Where("deleted_at IS NULL AND os_type = 'linux'").Count(&linuxCount)
	var windowsCount int64
	ctx.DB.Table("hosts").Where("deleted_at IS NULL AND os_type = 'windows'").Count(&windowsCount)

	return map[string]any{
		"hosts":       hosts,
		"total":       total,
		"online":      onlineCount,
		"offline":     total - onlineCount,
		"linux":       linuxCount,
		"windows":     windowsCount,
		"resultCount": len(hosts),
	}, nil
}

// executeHostDetail 查询主机详情
func executeHostDetail(ctx biz.SkillContext) (any, error) {
	ip, _ := ctx.Params["ip"].(string)
	name, _ := ctx.Params["name"].(string)
	hostID, _ := ctx.Params["id"].(float64)

	type HostDetail struct {
		ID              uint    `json:"id"`
		Name            string  `json:"name"`
		IP              string  `json:"ip"`
		PublicIP        string  `json:"publicIp"`
		Port            int     `json:"port"`
		OS              string  `json:"os"`
		OSType          string  `json:"osType"`
		Kernel          string  `json:"kernel"`
		Arch            string  `json:"arch"`
		Hostname        string  `json:"hostname"`
		Status          int     `json:"status"`
		CPUCores        int     `json:"cpuCores"`
		CPUUsage        float64 `json:"cpuUsage"`
		CPUModel        string  `json:"cpuModel"`
		MemoryTotal     uint64  `json:"memoryTotal"`
		MemoryUsed      uint64  `json:"memoryUsed"`
		MemoryUsage     float64 `json:"memoryUsage"`
		DiskTotal       uint64  `json:"diskTotal"`
		DiskUsed        uint64  `json:"diskUsed"`
		DiskUsage       float64 `json:"diskUsage"`
		Uptime          string  `json:"uptime"`
		Tags            string  `json:"tags"`
		Description     string  `json:"description"`
		GroupID         uint    `json:"groupId"`
		CloudProvider   string  `json:"cloudProvider"`
		CloudInstanceID string  `json:"cloudInstanceId"`
		CloudRegion     string  `json:"cloudRegion"`
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

	// 获取分组名称
	var groupName string
	if host.GroupID > 0 {
		ctx.DB.Table("asset_groups").Select("name").Where("id = ?", host.GroupID).Scan(&groupName)
	}

	// 构建状态文字
	statusText := "离线"
	if host.Status == 1 {
		statusText = "在线"
	}

	// 构建更丰富的详情
	result := map[string]any{
		"id":         host.ID,
		"name":       host.Name,
		"ip":         host.IP,
		"publicIp":   host.PublicIP,
		"port":       host.Port,
		"os":         host.OS,
		"osType":     host.OSType,
		"kernel":     host.Kernel,
		"arch":       host.Arch,
		"hostname":   host.Hostname,
		"status":     host.Status,
		"statusText": statusText,
		"groupName":  groupName,
		"tags":       host.Tags,
		"description": host.Description,
		"resources": map[string]any{
			"cpuCores":    host.CPUCores,
			"cpuModel":    host.CPUModel,
			"cpuUsage":    host.CPUUsage,
			"memoryTotal": host.MemoryTotal,
			"memoryUsed":  host.MemoryUsed,
			"memoryUsage": host.MemoryUsage,
			"diskTotal":   host.DiskTotal,
			"diskUsed":    host.DiskUsed,
			"diskUsage":   host.DiskUsage,
		},
		"uptime": host.Uptime,
	}

	if host.CloudProvider != "" {
		result["cloud"] = map[string]any{
			"provider":   host.CloudProvider,
			"instanceId": host.CloudInstanceID,
			"region":     host.CloudRegion,
		}
	}

	return result, nil
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
	groupName, _ := ctx.Params["group_name"].(string)

	type AlertHost struct {
		Name      string  `json:"name"`
		IP        string  `json:"ip"`
		Metric    string  `json:"metric"`
		Usage     float64 `json:"usage"`
		GroupName string  `json:"groupName"`
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
			Name      string  `json:"name"`
			IP        string  `json:"ip"`
			Usage     float64 `json:"usage"`
			GroupName string  `json:"groupName"`
		}
		q := ctx.DB.Table("hosts").
			Select("hosts.name, hosts.ip, hosts."+cond.column+" as `usage`, COALESCE(asset_groups.name, '') as group_name").
			Joins("LEFT JOIN asset_groups ON hosts.group_id = asset_groups.id").
			Where("hosts.deleted_at IS NULL AND hosts.status = 1 AND hosts."+cond.column+" > ?", threshold)

		if groupName != "" {
			q = q.Where("asset_groups.name LIKE ?", "%"+groupName+"%")
		}

		q.Order("hosts." + cond.column + " DESC").Limit(20).Find(&hosts)

		for _, h := range hosts {
			alerts = append(alerts, AlertHost{
				Name:      h.Name,
				IP:        h.IP,
				Metric:    strings.ToUpper(cond.metric),
				Usage:     h.Usage,
				GroupName: h.GroupName,
			})
		}
	}

	// 整体资源使用统计
	type OverallStats struct {
		AvgCPU    float64 `json:"avgCpu"`
		MaxCPU    float64 `json:"maxCpu"`
		AvgMemory float64 `json:"avgMemory"`
		MaxMemory float64 `json:"maxMemory"`
		AvgDisk   float64 `json:"avgDisk"`
		MaxDisk   float64 `json:"maxDisk"`
	}
	var overall OverallStats
	ctx.DB.Table("hosts").
		Select("AVG(cpu_usage) as avg_cpu, MAX(cpu_usage) as max_cpu, AVG(memory_usage) as avg_memory, MAX(memory_usage) as max_memory, AVG(disk_usage) as avg_disk, MAX(disk_usage) as max_disk").
		Where("deleted_at IS NULL AND status = 1").
		Scan(&overall)

	// 按分组统计
	type GroupStat struct {
		GroupName string  `json:"groupName"`
		Count     int64   `json:"count"`
		AvgCPU    float64 `json:"avgCpu"`
		AvgMem    float64 `json:"avgMemory"`
		AvgDisk   float64 `json:"avgDisk"`
	}
	var groupStats []GroupStat
	ctx.DB.Table("hosts").
		Select("COALESCE(asset_groups.name, '未分组') as group_name, COUNT(*) as count, AVG(hosts.cpu_usage) as avg_cpu, AVG(hosts.memory_usage) as avg_mem, AVG(hosts.disk_usage) as avg_disk").
		Joins("LEFT JOIN asset_groups ON hosts.group_id = asset_groups.id").
		Where("hosts.deleted_at IS NULL AND hosts.status = 1").
		Group("asset_groups.name").
		Order("avg_cpu DESC").
		Find(&groupStats)

	// 健康评估
	healthLevel := "healthy"
	if len(alerts) > 5 {
		healthLevel = "warning"
	}
	if overall.MaxCPU > 95 || overall.MaxMemory > 95 || overall.MaxDisk > 95 {
		healthLevel = "critical"
	}

	return map[string]any{
		"threshold":    threshold,
		"alertCount":   len(alerts),
		"alerts":       alerts,
		"overallStats": overall,
		"groupStats":   groupStats,
		"healthLevel":  healthLevel,
	}, nil
}

// executeHostCollect 触发主机信息采集
func executeHostCollect(ctx biz.SkillContext) (any, error) {
	ip, _ := ctx.Params["ip"].(string)
	groupName, _ := ctx.Params["group_name"].(string)

	type SimpleHost struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		IP   string `json:"ip"`
	}
	var hosts []SimpleHost
	query := ctx.DB.Table("hosts").Select("id, name, ip").Where("deleted_at IS NULL")

	if ip != "" {
		query = query.Where("ip = ?", ip)
	}
	if groupName != "" {
		query = query.Where("group_name = ?", groupName)
	}
	query.Limit(50).Find(&hosts)

	if len(hosts) == 0 {
		return nil, fmt.Errorf("未找到匹配的主机")
	}

	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"hostCount": len(hosts),
			"hosts":     hosts,
			"status":    "pending_confirmation",
			"warning":   fmt.Sprintf("⚠️ 将对 %d 台主机触发信息采集，请确认执行", len(hosts)),
		}, nil
	}

	// 已确认 → 通过 SSH 采集基本信息
	type CollectResult struct {
		Host   string `json:"host"`
		IP     string `json:"ip"`
		Info   string `json:"info,omitempty"`
		Error  string `json:"error,omitempty"`
	}
	var results []CollectResult
	successCount := 0

	for _, h := range hosts {
		client, _, err := CreateSSHClient(ctx.DB, h.ID)
		if err != nil {
			results = append(results, CollectResult{Host: h.Name, IP: h.IP, Error: err.Error()})
			continue
		}
		// 采集基本系统信息
		output, err := client.Execute("uname -a && cat /etc/os-release 2>/dev/null | head -5 && free -h | head -3 && df -h / | tail -1 && nproc")
		client.Close()
		if err != nil {
			results = append(results, CollectResult{Host: h.Name, IP: h.IP, Error: err.Error()})
		} else {
			results = append(results, CollectResult{Host: h.Name, IP: h.IP, Info: output})
			successCount++
		}
	}

	return map[string]any{
		"status":       "success",
		"message":      fmt.Sprintf("✅ 已完成 %d/%d 台主机的信息采集", successCount, len(hosts)),
		"results":      results,
		"successCount": successCount,
	}, nil
}

// executeHostExecCommand 远程执行命令
func executeHostExecCommand(ctx biz.SkillContext) (any, error) {
	command, _ := ctx.Params["command"].(string)
	if command == "" {
		return nil, fmt.Errorf("请指定要执行的命令")
	}
	ip, _ := ctx.Params["ip"].(string)

	// 安全检查：拒绝危险命令
	dangerousPatterns := []string{"rm -rf /", "mkfs", "dd if=", ":(){ :|:& };:", "> /dev/sd", "chmod -R 777 /"}
	cmdLower := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmdLower, pattern) {
			return nil, fmt.Errorf("安全检查未通过：命令包含危险操作 [%s]，已被拒绝", pattern)
		}
	}

	// 查找目标主机
	type SimpleHost struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		IP   string `json:"ip"`
	}
	var hosts []SimpleHost

	if ip != "" {
		ctx.DB.Table("hosts").Select("id, name, ip").Where("ip = ? AND deleted_at IS NULL", ip).Find(&hosts)
	}
	if hostIDs, ok := ctx.Params["host_ids"].([]any); ok && len(hostIDs) > 0 {
		ids := make([]uint, 0, len(hostIDs))
		for _, id := range hostIDs {
			if v, ok := id.(float64); ok {
				ids = append(ids, uint(v))
			}
		}
		ctx.DB.Table("hosts").Select("id, name, ip").Where("id IN ? AND deleted_at IS NULL", ids).Find(&hosts)
	}

	if len(hosts) == 0 {
		return nil, fmt.Errorf("未找到目标主机，请指定主机 IP 或 ID 列表")
	}

	// 未确认 → 返回待确认信息
	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"message":    fmt.Sprintf("命令 [%s] 将在 %d 台主机上执行", command, len(hosts)),
			"command":    command,
			"hostCount":  len(hosts),
			"hosts":      hosts,
			"status":     "pending_confirmation",
			"warning":    "⚠️ 远程命令执行是高风险操作，请确认命令内容无误后再执行",
		}, nil
	}

	// 已确认 → 真正执行
	type ExecResult struct {
		Host   string `json:"host"`
		IP     string `json:"ip"`
		Output string `json:"output"`
		Error  string `json:"error,omitempty"`
	}
	var results []ExecResult
	successCount := 0

	for _, h := range hosts {
		client, _, err := CreateSSHClient(ctx.DB, h.ID)
		if err != nil {
			results = append(results, ExecResult{Host: h.Name, IP: h.IP, Error: err.Error()})
			continue
		}
		output, err := client.ExecuteWithTimeout(command, 30*time.Second)
		client.Close()
		if err != nil {
			results = append(results, ExecResult{Host: h.Name, IP: h.IP, Output: output, Error: err.Error()})
		} else {
			results = append(results, ExecResult{Host: h.Name, IP: h.IP, Output: output})
			successCount++
		}
	}

	return map[string]any{
		"status":       "success",
		"message":      fmt.Sprintf("✅ 命令已在 %d/%d 台主机上执行完成", successCount, len(hosts)),
		"command":      command,
		"results":      results,
		"successCount": successCount,
		"totalCount":   len(hosts),
	}, nil
}

// executeHostFileManage 远程文件管理
func executeHostFileManage(ctx biz.SkillContext) (any, error) {
	action, _ := ctx.Params["action"].(string)
	filePath, _ := ctx.Params["path"].(string)
	ip, _ := ctx.Params["ip"].(string)

	if action == "" {
		return nil, fmt.Errorf("请指定操作类型: list / download / upload")
	}
	if filePath == "" {
		return nil, fmt.Errorf("请指定文件或目录路径")
	}

	// 查找主机
	var hostID uint
	query := ctx.DB.Table("hosts").Select("id").Where("deleted_at IS NULL")
	if ip != "" {
		query = query.Where("ip = ?", ip)
	}
	if hid, ok := ctx.Params["host_id"].(float64); ok && hid > 0 {
		query = query.Where("id = ?", uint(hid))
	}
	var hostRow struct{ ID uint }
	if err := query.First(&hostRow).Error; err != nil {
		return nil, fmt.Errorf("未找到目标主机")
	}
	hostID = hostRow.ID

	// list 操作是低风险，直接执行
	if action == "list" {
		client, hostInfo, err := CreateSSHClient(ctx.DB, hostID)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		output, err := client.Execute(fmt.Sprintf("ls -la %s", filePath))
		if err != nil {
			return nil, fmt.Errorf("列出目录失败: %v", err)
		}
		return map[string]any{
			"status":  "success",
			"host":    hostInfo.IP,
			"path":    filePath,
			"listing": output,
		}, nil
	}

	// download / upload 需要确认
	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"action":  action,
			"path":    filePath,
			"hostID":  hostID,
			"status":  "pending_confirmation",
			"warning": fmt.Sprintf("⚠️ 将对主机执行文件 %s 操作: %s，请确认执行", action, filePath),
		}, nil
	}

	// 已确认 → 执行 download（读取文件内容）
	if action == "download" {
		client, _, err := CreateSSHClient(ctx.DB, hostID)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		// 读取文件内容（限制大小）
		output, err := client.Execute(fmt.Sprintf("head -c 65536 %s", filePath))
		if err != nil {
			return nil, fmt.Errorf("读取文件失败: %v", err)
		}
		return map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("✅ 已读取文件 %s 的内容", filePath),
			"content": output,
		}, nil
	}

	return map[string]any{
		"status":  "pending",
		"message": fmt.Sprintf("文件 %s 操作需要通过文件管理界面完成", action),
	}, nil
}
