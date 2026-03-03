package skills

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// getDeviceCredential 从数据库获取设备凭证（自动解密）
func getDeviceCredential(db *gorm.DB, deviceID uint) (username, password string, err error) {
	var credentialID uint
	db.Table("network_devices").Select("credential_id").Where("id = ?", deviceID).Scan(&credentialID)
	if credentialID == 0 {
		return "", "", fmt.Errorf("未配置凭证")
	}

	var encUsername, encPassword string
	db.Table("credentials").Select("username").Where("id = ?", credentialID).Scan(&encUsername)
	db.Table("credentials").Select("password").Where("id = ?", credentialID).Scan(&encPassword)

	// 用户名不加密，密码需要解密（复用 service_helper.go 的 decryptCredential）
	username = encUsername
	password, err = decryptCredential(encPassword)
	if err != nil {
		return "", "", fmt.Errorf("解密密码失败: %v", err)
	}
	return username, password, nil
}

// buildDeviceSSHConfig 构建网络设备 SSH 配置（兼容老设备 + keyboard-interactive）
func buildDeviceSSHConfig(username, password string) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = password
				}
				return answers, nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
		Config: ssh.Config{
			KeyExchanges: []string{
				"curve25519-sha256", "curve25519-sha256@libssh.org",
				"ecdh-sha2-nistp256", "ecdh-sha2-nistp384", "ecdh-sha2-nistp521",
				"diffie-hellman-group14-sha256", "diffie-hellman-group14-sha1",
				"diffie-hellman-group1-sha1",
			},
			Ciphers: []string{
				"aes128-gcm@openssh.com", "aes256-gcm@openssh.com",
				"chacha20-poly1305@openssh.com",
				"aes128-ctr", "aes192-ctr", "aes256-ctr",
				"aes128-cbc", "aes192-cbc", "aes256-cbc", "3des-cbc",
			},
		},
	}
}

// RegisterDeviceSkills 注册网络设备管理 Skills
func RegisterDeviceSkills(registry *biz.ToolRegistry) {
	registry.Register(MustLoadBuiltinSkill("device.list", executeDeviceList))
	registry.Register(MustLoadBuiltinSkill("device.detail", executeDeviceDetail))
	registry.Register(MustLoadBuiltinSkill("device.test_connection", executeDeviceTestConnection))
	registry.Register(MustLoadBuiltinSkill("device.exec_command", executeDeviceExecCommand))
	registry.Register(MustLoadBuiltinSkill("device.manage", executeDeviceManage))
}

// executeDeviceList 查询网络设备列表
func executeDeviceList(ctx biz.SkillContext) (any, error) {
	keyword, _ := ctx.Params["keyword"].(string)
	deviceType, _ := ctx.Params["device_type"].(string)
	protocol, _ := ctx.Params["protocol"].(string)
	brand, _ := ctx.Params["brand"].(string)
	groupName, _ := ctx.Params["group_name"].(string)
	limit := 20
	if l, ok := ctx.Params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	type DeviceResult struct {
		ID           uint   `json:"id"`
		Name         string `json:"name"`
		IP           string `json:"ip"`
		Brand        string `json:"brand"`
		BrandModel   string `json:"brandModel"`
		SerialNumber string `json:"serialNumber"`
		DeviceType   string `json:"deviceType"`
		Protocol     string `json:"protocol"`
		Port         int    `json:"port"`
		Status       int    `json:"status"`
		GroupName    string `json:"groupName"`
		Tags         string `json:"tags"`
	}

	query := ctx.DB.Table("network_devices").
		Select("network_devices.id, network_devices.name, network_devices.ip, network_devices.brand, network_devices.brand_model, network_devices.serial_number, network_devices.device_type, network_devices.protocol, network_devices.port, network_devices.status, COALESCE(asset_group.name, '') as group_name, network_devices.tags").
		Joins("LEFT JOIN asset_group ON network_devices.group_id = asset_group.id").
		Where("network_devices.deleted_at IS NULL")

	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("(network_devices.name LIKE ? OR network_devices.ip LIKE ? OR network_devices.brand_model LIKE ? OR network_devices.serial_number LIKE ?)", like, like, like, like)
	}
	if deviceType != "" {
		query = query.Where("network_devices.device_type = ?", deviceType)
	}
	if protocol != "" {
		query = query.Where("network_devices.protocol = ?", protocol)
	}
	if brand != "" {
		query = query.Where("network_devices.brand LIKE ?", "%"+brand+"%")
	}
	if groupName != "" {
		query = query.Where("asset_group.name LIKE ?", "%"+groupName+"%")
	}
	if statusVal, ok := ctx.Params["status"].(float64); ok {
		query = query.Where("network_devices.status = ?", int(statusVal))
	}

	var devices []DeviceResult
	if err := query.Order("network_devices.id DESC").Limit(limit).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("查询网络设备失败: %v", err)
	}

	// 统计
	var total int64
	ctx.DB.Table("network_devices").Where("deleted_at IS NULL").Count(&total)
	var onlineCount int64
	ctx.DB.Table("network_devices").Where("deleted_at IS NULL AND status = 1").Count(&onlineCount)
	var offlineCount int64
	ctx.DB.Table("network_devices").Where("deleted_at IS NULL AND status = 0").Count(&offlineCount)

	// 按设备类型统计
	type TypeStat struct {
		DeviceType string `json:"deviceType"`
		Count      int64  `json:"count"`
	}
	var typeStats []TypeStat
	ctx.DB.Table("network_devices").
		Select("device_type, COUNT(*) as count").
		Where("deleted_at IS NULL").
		Group("device_type").
		Find(&typeStats)

	byType := make(map[string]int64)
	for _, ts := range typeStats {
		byType[ts.DeviceType] = ts.Count
	}

	return map[string]any{
		"devices":     devices,
		"total":       total,
		"online":      onlineCount,
		"offline":     offlineCount,
		"unknown":     total - onlineCount - offlineCount,
		"byType":      byType,
		"resultCount": len(devices),
	}, nil
}

// executeDeviceDetail 查询网络设备详情
func executeDeviceDetail(ctx biz.SkillContext) (any, error) {
	ip, _ := ctx.Params["ip"].(string)
	name, _ := ctx.Params["name"].(string)
	deviceID, _ := ctx.Params["id"].(float64)

	type DeviceDetail struct {
		ID           uint   `json:"id"`
		Name         string `json:"name"`
		IP           string `json:"ip"`
		Brand        string `json:"brand"`
		BrandModel   string `json:"brandModel"`
		SerialNumber string `json:"serialNumber"`
		DeviceType   string `json:"deviceType"`
		Protocol     string `json:"protocol"`
		Port         int    `json:"port"`
		CredentialID uint   `json:"credentialId"`
		GroupID      uint   `json:"groupId"`
		Status       int    `json:"status"`
		Tags         string `json:"tags"`
		Description  string `json:"description"`
	}

	query := ctx.DB.Table("network_devices").Where("deleted_at IS NULL")
	if deviceID > 0 {
		query = query.Where("id = ?", int(deviceID))
	} else if ip != "" {
		query = query.Where("ip = ?", ip)
	} else if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	} else {
		return nil, fmt.Errorf("请提供设备 IP、名称或 ID")
	}

	var device DeviceDetail
	if err := query.First(&device).Error; err != nil {
		return nil, fmt.Errorf("网络设备不存在")
	}

	// 获取分组名称
	var groupName string
	if device.GroupID > 0 {
		ctx.DB.Table("asset_group").Select("name").Where("id = ?", device.GroupID).Scan(&groupName)
	}

	// 获取凭证名称
	var credentialName string
	if device.CredentialID > 0 {
		ctx.DB.Table("credentials").Select("name").Where("id = ?", device.CredentialID).Scan(&credentialName)
	}

	statusText := "未知"
	switch device.Status {
	case 1:
		statusText = "在线"
	case 0:
		statusText = "离线"
	}

	deviceTypeText := map[string]string{
		"switch":   "交换机",
		"router":   "路由器",
		"firewall": "防火墙",
		"ac":       "AC",
		"ap":       "AP",
		"other":    "其他",
	}

	return map[string]any{
		"id":             device.ID,
		"name":           device.Name,
		"ip":             device.IP,
		"brand":          device.Brand,
		"brandModel":     device.BrandModel,
		"serialNumber":   device.SerialNumber,
		"deviceType":     device.DeviceType,
		"deviceTypeText": deviceTypeText[device.DeviceType],
		"protocol":       strings.ToUpper(device.Protocol),
		"port":           device.Port,
		"status":         device.Status,
		"statusText":     statusText,
		"groupName":      groupName,
		"credentialName": credentialName,
		"tags":           device.Tags,
		"description":    device.Description,
	}, nil
}

// executeDeviceTestConnection 测试网络设备连接
func executeDeviceTestConnection(ctx biz.SkillContext) (any, error) {
	ip, _ := ctx.Params["ip"].(string)
	groupName, _ := ctx.Params["group_name"].(string)

	type SimpleDevice struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		IP       string `json:"ip"`
		Protocol string `json:"protocol"`
		Port     int    `json:"port"`
	}
	var devices []SimpleDevice
	query := ctx.DB.Table("network_devices").Select("id, name, ip, protocol, port").Where("deleted_at IS NULL")

	if ip != "" {
		query = query.Where("ip = ?", ip)
	}
	if groupName != "" {
		query = query.Joins("LEFT JOIN asset_group ON network_devices.group_id = asset_group.id").
			Where("asset_group.name LIKE ?", "%"+groupName+"%")
	}
	if deviceIDs, ok := ctx.Params["device_ids"].([]any); ok && len(deviceIDs) > 0 {
		ids := make([]uint, 0, len(deviceIDs))
		for _, id := range deviceIDs {
			if v, ok := id.(float64); ok {
				ids = append(ids, uint(v))
			}
		}
		query = query.Where("id IN ?", ids)
	}
	query.Limit(50).Find(&devices)

	if len(devices) == 0 {
		return nil, fmt.Errorf("未找到匹配的网络设备")
	}

	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"deviceCount": len(devices),
			"devices":     devices,
			"status":      "pending_confirmation",
			"warning":     fmt.Sprintf("⚠️ 将对 %d 台网络设备进行连接测试，请确认执行", len(devices)),
		}, nil
	}

	// 已确认 → 执行连接测试
	type TestResult struct {
		Device   string `json:"device"`
		IP       string `json:"ip"`
		Protocol string `json:"protocol"`
		Status   string `json:"status"`
		Latency  string `json:"latency,omitempty"`
		Error    string `json:"error,omitempty"`
	}
	var results []TestResult
	successCount := 0

	for _, d := range devices {
		startTime := time.Now()
		addr := fmt.Sprintf("%s:%d", d.IP, d.Port)

		if d.Protocol == "telnet" {
			// Telnet 测试：TCP 连接
			conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
			latency := time.Since(startTime).Round(time.Millisecond).String()
			if err != nil {
				results = append(results, TestResult{Device: d.Name, IP: d.IP, Protocol: "Telnet", Status: "失败", Error: err.Error()})
				ctx.DB.Table("network_devices").Where("id = ?", d.ID).Update("status", 0)
			} else {
				conn.Close()
				results = append(results, TestResult{Device: d.Name, IP: d.IP, Protocol: "Telnet", Status: "成功", Latency: latency})
				ctx.DB.Table("network_devices").Where("id = ?", d.ID).Update("status", 1)
				successCount++
			}
		} else {
			// SSH 测试：获取凭证（解密）并尝试连接
			username, password, credErr := getDeviceCredential(ctx.DB, d.ID)
			if credErr != nil {
				results = append(results, TestResult{Device: d.Name, IP: d.IP, Protocol: "SSH", Status: "失败", Error: credErr.Error()})
				continue
			}

			config := buildDeviceSSHConfig(username, password)
			client, err := ssh.Dial("tcp", addr, config)
			latency := time.Since(startTime).Round(time.Millisecond).String()
			if err != nil {
				results = append(results, TestResult{Device: d.Name, IP: d.IP, Protocol: "SSH", Status: "失败", Error: err.Error()})
				ctx.DB.Table("network_devices").Where("id = ?", d.ID).Update("status", 0)
			} else {
				client.Close()
				results = append(results, TestResult{Device: d.Name, IP: d.IP, Protocol: "SSH", Status: "成功", Latency: latency})
				ctx.DB.Table("network_devices").Where("id = ?", d.ID).Update("status", 1)
				successCount++
			}
		}
	}

	return map[string]any{
		"status":       "success",
		"message":      fmt.Sprintf("✅ 连接测试完成 %d/%d 台设备连接正常", successCount, len(devices)),
		"results":      results,
		"successCount": successCount,
		"totalCount":   len(devices),
	}, nil
}

// executeDeviceExecCommand 在网络设备上远程执行命令
func executeDeviceExecCommand(ctx biz.SkillContext) (any, error) {
	command, _ := ctx.Params["command"].(string)
	if command == "" {
		return nil, fmt.Errorf("请指定要执行的命令")
	}
	ip, _ := ctx.Params["ip"].(string)

	// 安全检查：拒绝危险的网络设备配置命令
	dangerousPatterns := []string{
		"write erase", "erase startup", "format", "delete /force",
		"reset saved-configuration", "restore factory",
		"no service password", "shutdown",
	}
	cmdLower := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmdLower, pattern) {
			return nil, fmt.Errorf("安全检查未通过：命令包含危险操作 [%s]，已被拒绝", pattern)
		}
	}

	// 查找目标设备
	type SimpleDevice struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		IP       string `json:"ip"`
		Protocol string `json:"protocol"`
		Port     int    `json:"port"`
	}
	var devices []SimpleDevice

	if ip != "" {
		ctx.DB.Table("network_devices").Select("id, name, ip, protocol, port").
			Where("ip = ? AND deleted_at IS NULL", ip).Find(&devices)
	}
	if deviceIDs, ok := ctx.Params["device_ids"].([]any); ok && len(deviceIDs) > 0 {
		ids := make([]uint, 0, len(deviceIDs))
		for _, id := range deviceIDs {
			if v, ok := id.(float64); ok {
				ids = append(ids, uint(v))
			}
		}
		var idDevices []SimpleDevice
		ctx.DB.Table("network_devices").Select("id, name, ip, protocol, port").
			Where("id IN ? AND deleted_at IS NULL", ids).Find(&idDevices)
		devices = append(devices, idDevices...)
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("未找到目标网络设备，请指定设备 IP 或 ID 列表")
	}

	// 未确认 → 返回待确认信息
	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"message":     fmt.Sprintf("命令 [%s] 将在 %d 台网络设备上执行", command, len(devices)),
			"command":     command,
			"deviceCount": len(devices),
			"devices":     devices,
			"status":      "pending_confirmation",
			"warning":     "⚠️ 网络设备远程命令执行是高风险操作，请确认命令内容无误后再执行",
		}, nil
	}

	// 已确认 → 通过 SSH 执行命令
	type ExecResult struct {
		Device string `json:"device"`
		IP     string `json:"ip"`
		Output string `json:"output"`
		Error  string `json:"error,omitempty"`
	}
	var results []ExecResult
	successCount := 0

	for _, d := range devices {
		if d.Protocol == "telnet" {
			results = append(results, ExecResult{
				Device: d.Name, IP: d.IP,
				Error: "Telnet 设备暂不支持 AI 远程命令执行，请使用终端手动操作",
			})
			continue
		}

		// SSH 执行：获取凭证（解密）
		username, password, credErr := getDeviceCredential(ctx.DB, d.ID)
		if credErr != nil {
			results = append(results, ExecResult{Device: d.Name, IP: d.IP, Error: credErr.Error()})
			continue
		}

		addr := fmt.Sprintf("%s:%d", d.IP, d.Port)
		config := buildDeviceSSHConfig(username, password)

		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			results = append(results, ExecResult{Device: d.Name, IP: d.IP, Error: fmt.Sprintf("SSH连接失败: %v", err)})
			continue
		}

		session, err := client.NewSession()
		if err != nil {
			client.Close()
			results = append(results, ExecResult{Device: d.Name, IP: d.IP, Error: fmt.Sprintf("创建会话失败: %v", err)})
			continue
		}

		output, err := session.CombinedOutput(command)
		session.Close()
		client.Close()

		if err != nil {
			results = append(results, ExecResult{Device: d.Name, IP: d.IP, Output: string(output), Error: err.Error()})
		} else {
			results = append(results, ExecResult{Device: d.Name, IP: d.IP, Output: string(output)})
			successCount++
		}
	}

	return map[string]any{
		"status":       "success",
		"message":      fmt.Sprintf("✅ 命令已在 %d/%d 台网络设备上执行完成", successCount, len(devices)),
		"command":      command,
		"results":      results,
		"successCount": successCount,
		"totalCount":   len(devices),
	}, nil
}

// executeDeviceManage 网络设备管理（创建/修改/删除/查凭证/查分组）
func executeDeviceManage(ctx biz.SkillContext) (any, error) {
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
		ctx.DB.Table("credentials").Select("id, name, type, category, username").Where("deleted_at IS NULL AND (category = 'all' OR category = 'network')").Find(&creds)
		return map[string]any{
			"credentials": creds,
			"total":       len(creds),
			"hint":        "创建设备时可以使用以上凭证的 ID",
		}, nil

	case "list_groups":
		type GroupInfo struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		}
		var groups []GroupInfo
		ctx.DB.Table("asset_group").Select("id, name").Where("deleted_at IS NULL").Order("name").Find(&groups)
		return map[string]any{
			"groups": groups,
			"total":  len(groups),
			"hint":   "创建设备时可以使用以上分组的 ID",
		}, nil

	case "create":
		name, _ := ctx.Params["name"].(string)
		ip, _ := ctx.Params["ip"].(string)
		deviceType, _ := ctx.Params["device_type"].(string)
		if name == "" || ip == "" || deviceType == "" {
			return nil, fmt.Errorf("创建网络设备需要指定 name, ip, device_type 参数")
		}

		// 校验设备类型
		validTypes := map[string]string{"switch": "交换机", "router": "路由器", "firewall": "防火墙", "ac": "AC", "ap": "AP", "other": "其他"}
		typeText, validType := validTypes[deviceType]
		if !validType {
			return nil, fmt.Errorf("不支持的设备类型: %s，支持: switch/router/firewall/ac/ap/other", deviceType)
		}

		// 检查 IP 是否已存在
		var existCount int64
		ctx.DB.Table("network_devices").Where("ip = ? AND deleted_at IS NULL", ip).Count(&existCount)
		if existCount > 0 {
			return nil, fmt.Errorf("网络设备 IP %s 已存在，无需重复添加", ip)
		}

		protocol := "ssh"
		if v, _ := ctx.Params["protocol"].(string); v != "" {
			protocol = v
		}
		port := 22
		if protocol == "telnet" {
			port = 23
		}
		if v, ok := ctx.Params["port"].(float64); ok && v > 0 {
			port = int(v)
		}
		brand, _ := ctx.Params["brand"].(string)
		brandModel, _ := ctx.Params["brand_model"].(string)
		serialNumber, _ := ctx.Params["serial_number"].(string)
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

		if !isConfirmed(ctx.Params) {
			credName := ""
			if credentialID > 0 {
				ctx.DB.Table("credentials").Select("name").Where("id = ?", credentialID).Scan(&credName)
			}
			groupName := ""
			if groupID > 0 {
				ctx.DB.Table("asset_group").Select("name").Where("id = ?", groupID).Scan(&groupName)
			}
			return map[string]any{
				"action":       "create",
				"name":         name,
				"ip":           ip,
				"deviceType":   typeText,
				"protocol":     strings.ToUpper(protocol),
				"port":         port,
				"brand":        brand,
				"brandModel":   brandModel,
				"serialNumber": serialNumber,
				"credential":   credName,
				"group":        groupName,
				"status":       "pending_confirmation",
				"warning":      fmt.Sprintf("即将创建%s [%s](%s:%d)，协议: %s，请确认", typeText, name, ip, port, strings.ToUpper(protocol)),
			}, nil
		}

		// 确认后创建
		now := time.Now()
		if err := ctx.DB.Exec(
			`INSERT INTO network_devices (name, ip, device_type, protocol, port, brand, brand_model, serial_number, credential_id, group_id, tags, description, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, -1, ?, ?)`,
			name, ip, deviceType, protocol, port, brand, brandModel, serialNumber, credentialID, groupID, tags, description, now, now,
		).Error; err != nil {
			return nil, fmt.Errorf("创建网络设备失败: %v", err)
		}

		return map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("✅ 已成功创建%s [%s](%s:%d)", typeText, name, ip, port),
		}, nil

	case "update":
		deviceID, _ := ctx.Params["id"].(float64)
		deviceIP, _ := ctx.Params["device_ip"].(string)

		var targetID uint
		var targetName, targetIP string
		if deviceID > 0 {
			ctx.DB.Table("network_devices").Select("id, name, ip").Where("id = ? AND deleted_at IS NULL", uint(deviceID)).Row().Scan(&targetID, &targetName, &targetIP)
		} else if deviceIP != "" {
			ctx.DB.Table("network_devices").Select("id, name, ip").Where("ip = ? AND deleted_at IS NULL", deviceIP).Row().Scan(&targetID, &targetName, &targetIP)
		} else {
			return nil, fmt.Errorf("请提供设备 ID 或 IP")
		}
		if targetID == 0 {
			return nil, fmt.Errorf("未找到网络设备")
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
		if v, _ := ctx.Params["protocol"].(string); v != "" {
			updates["protocol"] = v
			changeDesc = append(changeDesc, fmt.Sprintf("协议→%s", strings.ToUpper(v)))
		}
		if v, _ := ctx.Params["device_type"].(string); v != "" {
			updates["device_type"] = v
			changeDesc = append(changeDesc, fmt.Sprintf("类型→%s", v))
		}
		if v, _ := ctx.Params["brand"].(string); v != "" {
			updates["brand"] = v
			changeDesc = append(changeDesc, fmt.Sprintf("品牌→%s", v))
		}
		if v, _ := ctx.Params["brand_model"].(string); v != "" {
			updates["brand_model"] = v
			changeDesc = append(changeDesc, fmt.Sprintf("型号→%s", v))
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
				"warning": fmt.Sprintf("即将修改设备 [%s](%s): %v，请确认", targetName, targetIP, changeDesc),
			}, nil
		}

		if err := ctx.DB.Table("network_devices").Where("id = ?", targetID).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("更新失败: %v", err)
		}

		return map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("✅ 已更新设备 [%s](%s)", targetName, targetIP),
		}, nil

	case "delete":
		deviceID, _ := ctx.Params["id"].(float64)
		deviceIP, _ := ctx.Params["device_ip"].(string)

		var targetID uint
		var targetName, targetIP string
		if deviceID > 0 {
			ctx.DB.Table("network_devices").Select("id, name, ip").Where("id = ? AND deleted_at IS NULL", uint(deviceID)).Row().Scan(&targetID, &targetName, &targetIP)
		} else if deviceIP != "" {
			ctx.DB.Table("network_devices").Select("id, name, ip").Where("ip = ? AND deleted_at IS NULL", deviceIP).Row().Scan(&targetID, &targetName, &targetIP)
		} else {
			return nil, fmt.Errorf("请提供设备 ID 或 IP")
		}
		if targetID == 0 {
			return nil, fmt.Errorf("未找到网络设备")
		}

		if !isConfirmed(ctx.Params) {
			return map[string]any{
				"action":  "delete",
				"id":      targetID,
				"name":    targetName,
				"ip":      targetIP,
				"status":  "pending_confirmation",
				"warning": fmt.Sprintf("⚠️ 即将删除网络设备 [%s](%s)，此操作不可恢复，请确认", targetName, targetIP),
			}, nil
		}

		now := time.Now()
		if err := ctx.DB.Table("network_devices").Where("id = ?", targetID).Update("deleted_at", now).Error; err != nil {
			return nil, fmt.Errorf("删除失败: %v", err)
		}

		return map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("✅ 已删除网络设备 [%s](%s)", targetName, targetIP),
		}, nil

	default:
		return nil, fmt.Errorf("不支持的操作: %s，支持: create/update/delete/list_credentials/list_groups", action)
	}
}
