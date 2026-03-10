package skills

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// shellQuote 对文件路径进行 shell 安全引用，防止命令注入
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// isDangerousHostCommand 判断命令是否为危险的变更/破坏类命令（需要人工确认）
// 采用黑名单模式：只有匹配危险模式的命令才需要确认，其余命令默认直接执行
func isDangerousHostCommand(command string) bool {
	cmd := strings.TrimSpace(strings.ToLower(command))

	// 常见只读查询命令应直接放行，避免误判为高风险。
	if strings.HasPrefix(cmd, "fdisk -l") || strings.HasPrefix(cmd, "fdisk --list") {
		return false
	}
	if strings.HasPrefix(cmd, "parted ") && strings.Contains(cmd, " print") {
		return false
	}
	if strings.HasPrefix(cmd, "systemctl status ") {
		return false
	}
	if strings.HasPrefix(cmd, "service ") && strings.HasSuffix(cmd, " status") {
		return false
	}

	// 危险命令关键词/模式黑名单
	dangerousPatterns := []string{
		// 文件删除/移动（破坏性）
		"rm ", "rm\t", "rmdir ",
		// 权限修改
		"chmod ", "chown ", "chgrp ",
		// 进程管理（终止进程）
		"kill ", "kill\t", "killall ", "pkill ",
		// 服务管理（启停/重启）
		"systemctl start ", "systemctl stop ", "systemctl restart ",
		"systemctl reload ", "systemctl enable ", "systemctl disable ",
		"systemctl mask ", "systemctl unmask ",
		"service ", // service xxx start/stop/restart
		// 系统关机/重启
		"shutdown", "reboot", "init 0", "init 6", "poweroff", "halt",
		// 磁盘/分区操作
		"mkfs", "fdisk ", "parted ", "gdisk ",
		"mount ", "umount ", "swapon", "swapoff",
		"dd if=", "dd of=",
		"resize2fs ", "xfs_growfs ", "growpart ",
		"lvextend ", "lvreduce ", "lvcreate ", "lvremove ",
		"vgcreate ", "vgremove ", "vgextend ",
		"pvcreate ", "pvremove ", "pvresize ",
		// 网络配置修改
		"ifdown ", "ifup ",
		"nmcli con mod", "nmcli connection mod",
		"ip addr add", "ip addr del", "ip route add", "ip route del",
		"ip link set",
		// 防火墙修改
		"iptables -a", "iptables -d", "iptables -i", "iptables -f",
		"iptables -x", "iptables -p",
		"firewall-cmd --add", "firewall-cmd --remove",
		"firewall-cmd --set", "firewall-cmd --reload",
		"ufw allow", "ufw deny", "ufw delete", "ufw enable", "ufw disable",
		// 用户管理
		"useradd ", "userdel ", "usermod ", "groupadd ", "groupdel ",
		"passwd ", "chpasswd",
		// 包管理（安装/卸载）
		"yum install", "yum remove", "yum erase", "yum update", "yum upgrade",
		"dnf install", "dnf remove", "dnf erase", "dnf update", "dnf upgrade",
		"apt install", "apt remove", "apt purge", "apt upgrade",
		"apt-get install", "apt-get remove", "apt-get purge", "apt-get upgrade",
		"pip install", "pip uninstall", "pip3 install", "pip3 uninstall",
		"npm install", "npm uninstall", "npm update",
		// 文件写入/覆盖
		"sed -i", "tee ", "tee\t",
		// 定时任务修改
		"crontab -e", "crontab -r",
		// Docker 变更操作
		"docker stop", "docker rm", "docker rmi", "docker pull",
		"docker run", "docker exec", "docker restart",
		"docker start", "docker kill", "docker pause", "docker unpause",
		"docker update", "docker rename", "docker prune",
		"docker network create", "docker network rm",
		"docker volume create", "docker volume rm",
		"docker compose up", "docker compose down",
		"docker compose stop", "docker compose start",
		"docker compose restart", "docker compose pull",
		"docker compose rm", "docker compose build",
		"docker-compose up", "docker-compose down",
		"docker-compose stop", "docker-compose start",
		"docker-compose restart", "docker-compose pull",
		"docker-compose rm", "docker-compose build",
		// 危险 shell 操作
		"> /dev/", ">> /dev/",
		":(){ :|:& };:",
		// SELinux
		"setenforce", "setsebool",
		// 内核参数修改
		"sysctl -w",
		// 系统日志清理
		"truncate ", "shred ",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmd, pattern) {
			return true
		}
	}

	// 检查重定向覆盖（> 文件，但排除 2>/dev/null 等常见无害用法）
	if strings.Contains(cmd, " > ") && !strings.Contains(cmd, " 2>/dev/null") && !strings.Contains(cmd, " > /dev/null") {
		return true
	}

	return false
}

// RegisterHostSkills 注册主机管理 Skills
func RegisterHostSkills(registry *biz.ToolRegistry) {
	registry.Register(MustLoadBuiltinSkill("host.list", executeHostList))
	registry.Register(MustLoadBuiltinSkill("host.detail", executeHostDetail))
	registry.Register(MustLoadBuiltinSkill("host.analyze", executeHostAnalyze))
	registry.Register(MustLoadBuiltinSkill("host.collect", executeHostCollect))
	registry.Register(MustLoadBuiltinSkill("host.exec_command", executeHostExecCommand))
	registry.Register(MustLoadBuiltinSkill("host.file_manage", executeHostFileManage))
	registry.Register(MustLoadBuiltinSkill("host.manage", executeHostManage))
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
		Select("hosts.id, hosts.name, hosts.ip, hosts.port, hosts.os, hosts.os_type, hosts.status, hosts.cpu_cores, hosts.cpu_usage, hosts.memory_usage, hosts.disk_usage, COALESCE(asset_group.name, '') as group_name, hosts.tags, hosts.uptime").
		Joins("LEFT JOIN asset_group ON hosts.group_id = asset_group.id").
		Where("hosts.deleted_at IS NULL")

	if keyword != "" {
		query = query.Where("(hosts.name LIKE ? OR hosts.ip LIKE ? OR hosts.description LIKE ?)", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if osType != "" {
		query = query.Where("hosts.os_type = ?", osType)
	}
	if groupName != "" {
		query = query.Where("asset_group.name LIKE ?", "%"+groupName+"%")
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
		ctx.DB.Table("asset_group").Select("name").Where("id = ?", host.GroupID).Scan(&groupName)
	}

	// 构建状态文字
	statusText := "离线"
	if host.Status == 1 {
		statusText = "在线"
	}

	// 构建更丰富的详情
	result := map[string]any{
		"id":          host.ID,
		"name":        host.Name,
		"ip":          host.IP,
		"publicIp":    host.PublicIP,
		"port":        host.Port,
		"os":          host.OS,
		"osType":      host.OSType,
		"kernel":      host.Kernel,
		"arch":        host.Arch,
		"hostname":    host.Hostname,
		"status":      host.Status,
		"statusText":  statusText,
		"groupName":   groupName,
		"tags":        host.Tags,
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
			Select("hosts.name, hosts.ip, hosts."+cond.column+" as `usage`, COALESCE(asset_group.name, '') as group_name").
			Joins("LEFT JOIN asset_group ON hosts.group_id = asset_group.id").
			Where("hosts.deleted_at IS NULL AND hosts.status = 1 AND hosts."+cond.column+" > ?", threshold)

		if groupName != "" {
			q = q.Where("asset_group.name LIKE ?", "%"+groupName+"%")
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
		Select("COALESCE(asset_group.name, '未分组') as group_name, COUNT(*) as count, AVG(hosts.cpu_usage) as avg_cpu, AVG(hosts.memory_usage) as avg_mem, AVG(hosts.disk_usage) as avg_disk").
		Joins("LEFT JOIN asset_group ON hosts.group_id = asset_group.id").
		Where("hosts.deleted_at IS NULL AND hosts.status = 1").
		Group("asset_group.name").
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
		query = query.Joins("LEFT JOIN asset_group ON hosts.group_id = asset_group.id").
			Where("asset_group.name LIKE ?", "%"+groupName+"%")
	}
	query.Limit(50).Find(&hosts)

	if len(hosts) == 0 {
		return nil, fmt.Errorf("未找到匹配的主机")
	}

	// 主机信息采集仅执行固定只读命令，直接执行即可。
	type CollectResult struct {
		Host  string `json:"host"`
		IP    string `json:"ip"`
		Info  string `json:"info,omitempty"`
		Error string `json:"error,omitempty"`
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
		"status":             "success",
		"effectiveRiskLevel": "low",
		"message":            fmt.Sprintf("✅ 已完成 %d/%d 台主机的信息采集", successCount, len(hosts)),
		"results":            results,
		"successCount":       successCount,
	}, nil
}

// executeHostExecCommand 远程执行命令
func executeHostExecCommand(ctx biz.SkillContext) (any, error) {
	command, _ := ctx.Params["command"].(string)
	if command == "" {
		return nil, fmt.Errorf("请指定要执行的命令")
	}
	ip, _ := ctx.Params["ip"].(string)

	// 超时时间：默认 60 秒，最大 600 秒（安装软件等耗时操作可调大）
	timeoutSec := 60
	if t, ok := ctx.Params["timeout"].(float64); ok && t > 0 {
		timeoutSec = int(t)
		if timeoutSec > 600 {
			timeoutSec = 600
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
		var idHosts []SimpleHost
		ctx.DB.Table("hosts").Select("id, name, ip").Where("id IN ? AND deleted_at IS NULL", ids).Find(&idHosts)
		hosts = append(hosts, idHosts...)
	}

	if len(hosts) == 0 {
		return nil, fmt.Errorf("未找到目标主机，请指定主机 IP 或 ID 列表")
	}

	// 非危险命令 → 直接执行，跳过确认
	if !isDangerousHostCommand(command) {
		type ExecResult struct {
			Host      string `json:"host"`
			IP        string `json:"ip"`
			Output    string `json:"output"`
			Truncated bool   `json:"truncated,omitempty"`
			Error     string `json:"error,omitempty"`
		}
		var results []ExecResult
		successCount := 0

		for _, h := range hosts {
			client, _, err := CreateSSHClient(ctx.DB, h.ID)
			if err != nil {
				results = append(results, ExecResult{Host: h.Name, IP: h.IP, Error: err.Error()})
				continue
			}
			output, err := client.ExecuteWithTimeout(command, time.Duration(timeoutSec)*time.Second)
			client.Close()

			truncated := false
			if len(output) > 65536 {
				output = output[:65536] + "\n... [输出已截断，超过 64KB]"
				truncated = true
			}

			if err != nil {
				results = append(results, ExecResult{Host: h.Name, IP: h.IP, Output: output, Truncated: truncated, Error: err.Error()})
			} else {
				results = append(results, ExecResult{Host: h.Name, IP: h.IP, Output: output, Truncated: truncated})
				successCount++
			}
		}

		return map[string]any{
			"status":             "success",
			"effectiveRiskLevel": "low",
			"message":            fmt.Sprintf("✅ 命令已在 %d/%d 台主机上执行完成（安全命令，已跳过确认）", successCount, len(hosts)),
			"command":            command,
			"timeout":            timeoutSec,
			"results":            results,
			"successCount":       successCount,
			"totalCount":         len(hosts),
		}, nil
	}

	// 未确认 → 返回待确认信息
	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"message":   fmt.Sprintf("命令 [%s] 将在 %d 台主机上执行（超时: %ds）", command, len(hosts), timeoutSec),
			"command":   command,
			"timeout":   timeoutSec,
			"hostCount": len(hosts),
			"hosts":     hosts,
			"status":    "pending_confirmation",
			"warning":   "⚠️ 远程命令执行是高风险操作，请确认命令内容无误后再执行",
		}, nil
	}

	// 已确认 → 真正执行
	type ExecResult struct {
		Host      string `json:"host"`
		IP        string `json:"ip"`
		Output    string `json:"output"`
		Truncated bool   `json:"truncated,omitempty"`
		Error     string `json:"error,omitempty"`
	}
	var results []ExecResult
	successCount := 0

	for _, h := range hosts {
		client, _, err := CreateSSHClient(ctx.DB, h.ID)
		if err != nil {
			results = append(results, ExecResult{Host: h.Name, IP: h.IP, Error: err.Error()})
			continue
		}
		output, err := client.ExecuteWithTimeout(command, time.Duration(timeoutSec)*time.Second)
		client.Close()

		// 输出截断：超过 64KB 时截断，避免 LLM 上下文溢出
		truncated := false
		if len(output) > 65536 {
			output = output[:65536] + "\n... [输出已截断，超过 64KB]"
			truncated = true
		}

		if err != nil {
			results = append(results, ExecResult{Host: h.Name, IP: h.IP, Output: output, Truncated: truncated, Error: err.Error()})
		} else {
			results = append(results, ExecResult{Host: h.Name, IP: h.IP, Output: output, Truncated: truncated})
			successCount++
		}
	}

	return map[string]any{
		"status":       "success",
		"message":      fmt.Sprintf("✅ 命令已在 %d/%d 台主机上执行完成", successCount, len(hosts)),
		"command":      command,
		"timeout":      timeoutSec,
		"results":      results,
		"successCount": successCount,
		"totalCount":   len(hosts),
	}, nil
}

// isUnsafePath 检查是否为不安全的系统路径
func isUnsafePath(path string) bool {
	unsafePrefixes := []string{"/boot", "/dev", "/proc", "/sys", "/run"}
	pathLower := strings.ToLower(strings.TrimRight(path, "/"))
	for _, prefix := range unsafePrefixes {
		if pathLower == prefix || strings.HasPrefix(pathLower, prefix+"/") {
			return true
		}
	}
	return false
}

// executeHostFileManage 远程文件管理
func executeHostFileManage(ctx biz.SkillContext) (any, error) {
	action, _ := ctx.Params["action"].(string)
	filePath, _ := ctx.Params["path"].(string)
	ip, _ := ctx.Params["ip"].(string)

	if action == "" {
		return nil, fmt.Errorf("请指定操作类型: list / read / download / write / backup")
	}
	if filePath == "" {
		return nil, fmt.Errorf("请指定文件或目录路径")
	}

	// 路径安全检查
	if isUnsafePath(filePath) {
		return nil, fmt.Errorf("安全检查未通过：禁止操作系统关键路径 [%s]（/boot, /dev, /proc, /sys, /run）", filePath)
	}
	if strings.Contains(filePath, "..") {
		return nil, fmt.Errorf("安全检查未通过：路径不允许包含 '..'")
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
		output, err := client.Execute(fmt.Sprintf("ls -la %s", shellQuote(filePath)))
		if err != nil {
			return nil, fmt.Errorf("列出目录失败: %v", err)
		}
		return map[string]any{
			"status":             "success",
			"effectiveRiskLevel": "low",
			"host":               hostInfo.IP,
			"path":               filePath,
			"listing":            output,
		}, nil
	}

	// read 操作是低风险，直接执行（查看文件内容，限制 64KB）
	if action == "read" {
		client, hostInfo, err := CreateSSHClient(ctx.DB, hostID)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		output, err := client.Execute(fmt.Sprintf("head -c 65536 %s", shellQuote(filePath)))
		if err != nil {
			return nil, fmt.Errorf("读取文件失败: %v", err)
		}
		return map[string]any{
			"status":             "success",
			"effectiveRiskLevel": "low",
			"host":               hostInfo.IP,
			"path":               filePath,
			"content":            output,
		}, nil
	}

	// backup 操作：备份文件为 .bak.时间戳
	if action == "backup" {
		if !isConfirmed(ctx.Params) {
			return map[string]any{
				"action":             "backup",
				"path":               filePath,
				"hostID":             hostID,
				"status":             "pending_confirmation",
				"effectiveRiskLevel": "medium",
				"warning":            fmt.Sprintf("⚠️ 将备份文件: %s → %s.bak.时间戳，请确认", filePath, filePath),
			}, nil
		}
		client, hostInfo, err := CreateSSHClient(ctx.DB, hostID)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		timestamp := time.Now().Format("20060102150405")
		backupPath := fmt.Sprintf("%s.bak.%s", filePath, timestamp)
		output, err := client.Execute(fmt.Sprintf("cp -p %s %s && echo 'backup ok'", shellQuote(filePath), shellQuote(backupPath)))
		if err != nil {
			return nil, fmt.Errorf("备份文件失败: %v", err)
		}
		return map[string]any{
			"status":             "success",
			"effectiveRiskLevel": "medium",
			"message":            fmt.Sprintf("✅ 已备份文件: %s → %s", filePath, backupPath),
			"host":               hostInfo.IP,
			"sourcePath":         filePath,
			"backupPath":         backupPath,
			"output":             output,
		}, nil
	}

	// write 操作：将内容写入文件（通过 heredoc）
	if action == "write" {
		content, _ := ctx.Params["content"].(string)
		if content == "" {
			return nil, fmt.Errorf("write 操作需要指定 content 参数（文件内容）")
		}

		if !isConfirmed(ctx.Params) {
			// 展示待写入内容的前 500 字符
			preview := content
			if len(preview) > 500 {
				preview = preview[:500] + "\n... [内容已截断]"
			}
			return map[string]any{
				"action":             "write",
				"path":               filePath,
				"hostID":             hostID,
				"contentLength":      len(content),
				"preview":            preview,
				"status":             "pending_confirmation",
				"effectiveRiskLevel": "high",
				"warning":            fmt.Sprintf("⚠️ 将写入 %d 字节到文件 %s，请确认（建议先使用 backup 操作备份原文件）", len(content), filePath),
			}, nil
		}

		if len(content) > 1024*1024 {
			return nil, fmt.Errorf("写入内容过大（%d 字节），最大支持 1MB", len(content))
		}
		client, hostInfo, err := CreateSSHClient(ctx.DB, hostID)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		// 使用 base64 编码传输，避免 heredoc 注入
		encoded := base64.StdEncoding.EncodeToString([]byte(content))
		writeCmd := fmt.Sprintf("echo %s | base64 -d > %s", shellQuote(encoded), shellQuote(filePath))
		output, err := client.ExecuteWithTimeout(writeCmd, 30*time.Second)
		if err != nil {
			return nil, fmt.Errorf("写入文件失败: %v", err)
		}
		return map[string]any{
			"status":             "success",
			"effectiveRiskLevel": "high",
			"message":            fmt.Sprintf("✅ 已成功写入 %d 字节到 %s", len(content), filePath),
			"host":               hostInfo.IP,
			"path":               filePath,
			"output":             output,
		}, nil
	}

	// download 为只读读取，与 read 一样直接执行。
	if action == "download" {
		client, hostInfo, err := CreateSSHClient(ctx.DB, hostID)
		if err != nil {
			return nil, err
		}
		defer client.Close()
		output, err := client.Execute(fmt.Sprintf("head -c 65536 %s", shellQuote(filePath)))
		if err != nil {
			return nil, fmt.Errorf("读取文件失败: %v", err)
		}
		return map[string]any{
			"status":             "success",
			"effectiveRiskLevel": "low",
			"message":            fmt.Sprintf("✅ 已读取文件 %s 的内容", filePath),
			"host":               hostInfo.IP,
			"path":               filePath,
			"content":            output,
		}, nil
	}

	return nil, fmt.Errorf("不支持的操作类型: %s，支持: list / read / download / write / backup", action)
}

// executeHostManage 主机管理（创建/修改/删除/查凭证/查分组）
func executeHostManage(ctx biz.SkillContext) (any, error) {
	action, _ := ctx.Params["action"].(string)
	if action == "" {
		return nil, fmt.Errorf("请指定操作: create/update/delete/list_credentials/list_groups")
	}

	switch action {
	case "list_credentials":
		type CredInfo struct {
			ID       uint   `json:"id"`
			Name     string `json:"name"`
			Type     string `json:"type"`
			Category string `json:"category"`
			Username string `json:"username"`
		}
		var creds []CredInfo
		ctx.DB.Table("credentials").Select("id, name, type, category, username").Where("deleted_at IS NULL AND (category = 'all' OR category = 'host')").Find(&creds)
		return map[string]any{
			"credentials":        creds,
			"total":              len(creds),
			"hint":               "创建主机时可以使用以上凭证的 ID",
			"effectiveRiskLevel": "low",
		}, nil

	case "list_groups":
		type GroupInfo struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		}
		var groups []GroupInfo
		ctx.DB.Table("asset_group").Select("id, name").Where("deleted_at IS NULL").Order("name").Find(&groups)
		return map[string]any{
			"groups":             groups,
			"total":              len(groups),
			"hint":               "创建主机时可以使用以上分组的 ID",
			"effectiveRiskLevel": "low",
		}, nil

	case "create":
		name, _ := ctx.Params["name"].(string)
		ip, _ := ctx.Params["ip"].(string)
		sshUser, _ := ctx.Params["ssh_user"].(string)
		if name == "" || ip == "" || sshUser == "" {
			return nil, fmt.Errorf("创建主机需要指定 name, ip, ssh_user 参数")
		}

		// 检查 IP 是否已存在
		var existCount int64
		ctx.DB.Table("hosts").Where("ip = ? AND deleted_at IS NULL", ip).Count(&existCount)
		if existCount > 0 {
			return nil, fmt.Errorf("主机 IP %s 已存在，无需重复添加", ip)
		}

		port := 22
		if v, ok := ctx.Params["port"].(float64); ok && v > 0 {
			port = int(v)
		}
		osType := "linux"
		if v, _ := ctx.Params["os_type"].(string); v != "" {
			osType = v
		}
		tags, _ := ctx.Params["tags"].(string)
		description, _ := ctx.Params["description"].(string)
		var credentialID uint
		if v, ok := ctx.Params["credential_id"].(float64); ok && v > 0 {
			credentialID = uint(v)
		}
		var groupID uint
		if v, ok := ctx.Params["group_id"].(float64); ok && v > 0 {
			groupID = uint(v)
		}
		rdpPort := 3389
		if v, ok := ctx.Params["rdp_port"].(float64); ok && v > 0 {
			rdpPort = int(v)
		}

		if !isConfirmed(ctx.Params) {
			// 查凭证名称
			credName := ""
			if credentialID > 0 {
				ctx.DB.Table("credentials").Select("name").Where("id = ?", credentialID).Scan(&credName)
			}
			groupName := ""
			if groupID > 0 {
				ctx.DB.Table("asset_group").Select("name").Where("id = ?", groupID).Scan(&groupName)
			}
			return map[string]any{
				"action":      "create",
				"name":        name,
				"ip":          ip,
				"port":        port,
				"sshUser":     sshUser,
				"osType":      osType,
				"credential":  credName,
				"group":       groupName,
				"tags":        tags,
				"description": description,
				"status":      "pending_confirmation",
				"warning":     fmt.Sprintf("即将创建主机 [%s](%s:%d)，SSH用户: %s，系统: %s，请确认", name, ip, port, sshUser, osType),
			}, nil
		}

		// 确认后创建
		now := time.Now()
		if err := ctx.DB.Exec(
			`INSERT INTO hosts (name, ip, port, ssh_user, credential_id, group_id, os_type, type, tags, description, status, rdp_port, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, 'self', ?, ?, -1, ?, ?, ?)`,
			name, ip, port, sshUser, credentialID, groupID, osType, tags, description, rdpPort, now, now,
		).Error; err != nil {
			return nil, fmt.Errorf("创建主机失败: %v", err)
		}

		return map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("✅ 已成功创建主机 [%s](%s:%d)", name, ip, port),
		}, nil

	case "update":
		hostID, _ := ctx.Params["id"].(float64)
		hostIP, _ := ctx.Params["host_ip"].(string)

		var targetID uint
		var targetName, targetIP string
		if hostID > 0 {
			ctx.DB.Table("hosts").Select("id, name, ip").Where("id = ? AND deleted_at IS NULL", uint(hostID)).Row().Scan(&targetID, &targetName, &targetIP)
		} else if hostIP != "" {
			ctx.DB.Table("hosts").Select("id, name, ip").Where("ip = ? AND deleted_at IS NULL", hostIP).Row().Scan(&targetID, &targetName, &targetIP)
		} else {
			return nil, fmt.Errorf("请提供主机 ID 或 IP")
		}
		if targetID == 0 {
			return nil, fmt.Errorf("未找到主机")
		}

		updates := map[string]any{"updated_at": time.Now()}
		changeDesc := []string{}
		if v, _ := ctx.Params["name"].(string); v != "" {
			updates["name"] = v
			changeDesc = append(changeDesc, fmt.Sprintf("名称→%s", v))
		}
		if v, _ := ctx.Params["ip"].(string); v != "" {
			updates["ip"] = v
			changeDesc = append(changeDesc, fmt.Sprintf("IP→%s", v))
		}
		if v, ok := ctx.Params["port"].(float64); ok && v > 0 {
			updates["port"] = int(v)
			changeDesc = append(changeDesc, fmt.Sprintf("端口→%d", int(v)))
		}
		if v, _ := ctx.Params["ssh_user"].(string); v != "" {
			updates["ssh_user"] = v
			changeDesc = append(changeDesc, fmt.Sprintf("SSH用户→%s", v))
		}
		if v, ok := ctx.Params["credential_id"].(float64); ok && v > 0 {
			updates["credential_id"] = uint(v)
			changeDesc = append(changeDesc, fmt.Sprintf("凭证ID→%d", int(v)))
		}
		if v, ok := ctx.Params["group_id"].(float64); ok && v > 0 {
			updates["group_id"] = uint(v)
			changeDesc = append(changeDesc, fmt.Sprintf("分组ID→%d", int(v)))
		}
		if v, _ := ctx.Params["tags"].(string); v != "" {
			updates["tags"] = v
			changeDesc = append(changeDesc, fmt.Sprintf("标签→%s", v))
		}
		if v, _ := ctx.Params["description"].(string); v != "" {
			updates["description"] = v
			changeDesc = append(changeDesc, "备注已更新")
		}

		if len(changeDesc) == 0 {
			return nil, fmt.Errorf("请指定要修改的参数")
		}

		if !isConfirmed(ctx.Params) {
			return map[string]any{
				"action":  "update",
				"id":      targetID,
				"name":    targetName,
				"ip":      targetIP,
				"changes": changeDesc,
				"status":  "pending_confirmation",
				"warning": fmt.Sprintf("即将修改主机 [%s](%s): %v，请确认", targetName, targetIP, changeDesc),
			}, nil
		}

		if err := ctx.DB.Table("hosts").Where("id = ?", targetID).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("更新失败: %v", err)
		}

		return map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("✅ 已更新主机 [%s](%s)", targetName, targetIP),
		}, nil

	case "delete":
		hostID, _ := ctx.Params["id"].(float64)
		hostIP, _ := ctx.Params["host_ip"].(string)

		var targetID uint
		var targetName, targetIP string
		if hostID > 0 {
			ctx.DB.Table("hosts").Select("id, name, ip").Where("id = ? AND deleted_at IS NULL", uint(hostID)).Row().Scan(&targetID, &targetName, &targetIP)
		} else if hostIP != "" {
			ctx.DB.Table("hosts").Select("id, name, ip").Where("ip = ? AND deleted_at IS NULL", hostIP).Row().Scan(&targetID, &targetName, &targetIP)
		} else {
			return nil, fmt.Errorf("请提供主机 ID 或 IP")
		}
		if targetID == 0 {
			return nil, fmt.Errorf("未找到主机")
		}

		if !isConfirmed(ctx.Params) {
			return map[string]any{
				"action":  "delete",
				"id":      targetID,
				"name":    targetName,
				"ip":      targetIP,
				"status":  "pending_confirmation",
				"warning": fmt.Sprintf("⚠️ 即将删除主机 [%s](%s)，此操作不可恢复，请确认", targetName, targetIP),
			}, nil
		}

		now := time.Now()
		if err := ctx.DB.Table("hosts").Where("id = ?", targetID).Update("deleted_at", now).Error; err != nil {
			return nil, fmt.Errorf("删除失败: %v", err)
		}

		return map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("✅ 已删除主机 [%s](%s)", targetName, targetIP),
		}, nil

	default:
		return nil, fmt.Errorf("不支持的操作: %s，支持: create/update/delete/list_credentials/list_groups", action)
	}
}
