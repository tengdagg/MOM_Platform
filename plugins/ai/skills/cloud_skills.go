package skills

import (
	"fmt"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// RegisterCloudSkills 注册云账号 Skills
func RegisterCloudSkills(registry *biz.ToolRegistry) {
	registry.Register(MustLoadBuiltinSkill("cloud.list_accounts", executeCloudListAccounts))
	registry.Register(MustLoadBuiltinSkill("cloud.list_instances", executeCloudListInstances))
	registry.Register(MustLoadBuiltinSkill("cloud.import_hosts", executeCloudImportHosts))
}

var providerMap = map[string]string{
	"aliyun": "阿里云", "tencent": "腾讯云", "aws": "AWS",
	"huawei": "华为云", "jdcloud": "京东云", "baidu": "百度云", "ksyun": "金山云",
}

// executeCloudListAccounts 查询云账号列表
func executeCloudListAccounts(ctx biz.SkillContext) (any, error) {
	provider, _ := ctx.Params["provider"].(string)
	statusFilter, _ := ctx.Params["status"].(string)

	type AccountInfo struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		Provider    string `json:"provider"`
		Region      string `json:"region"`
		Description string `json:"description"`
		Status      int    `json:"status"`
	}

	query := ctx.DB.Table("cloud_accounts").Where("deleted_at IS NULL")
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if statusFilter == "enabled" {
		query = query.Where("status = 1")
	} else if statusFilter == "disabled" {
		query = query.Where("status = 0")
	}

	var accounts []AccountInfo
	if err := query.Order("id ASC").Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("查询云账号失败: %v", err)
	}

	type AccountVO struct {
		AccountInfo
		ProviderName string `json:"providerName"`
		StatusText   string `json:"statusText"`
	}

	var result []AccountVO
	enabledCount := 0
	for _, a := range accounts {
		vo := AccountVO{
			AccountInfo:  a,
			ProviderName: providerMap[a.Provider],
			StatusText:   "启用",
		}
		if a.Status == 0 {
			vo.StatusText = "禁用"
		} else {
			enabledCount++
		}
		result = append(result, vo)
	}

	// 按厂商分组
	providerCounts := make(map[string]int)
	for _, a := range accounts {
		name := providerMap[a.Provider]
		if name == "" {
			name = a.Provider
		}
		providerCounts[name]++
	}

	return map[string]any{
		"accounts":   result,
		"total":      len(result),
		"enabled":    enabledCount,
		"disabled":   len(result) - enabledCount,
		"byProvider": providerCounts,
	}, nil
}

// executeCloudListInstances 查询云主机实例
func executeCloudListInstances(ctx biz.SkillContext) (any, error) {
	provider, _ := ctx.Params["provider"].(string)
	region, _ := ctx.Params["region"].(string)
	statusFilter, _ := ctx.Params["status"].(string)
	keyword, _ := ctx.Params["keyword"].(string)
	limit := 50
	if l, ok := ctx.Params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	type InstanceInfo struct {
		ID              uint   `json:"id"`
		Name            string `json:"name"`
		IP              string `json:"ip"`
		PublicIP        string `json:"publicIp"`
		CloudProvider   string `json:"cloudProvider"`
		CloudInstanceID string `json:"cloudInstanceId"`
		CloudRegion     string `json:"cloudRegion"`
		OS              string `json:"os"`
		OSType          string `json:"osType"`
		Status          int    `json:"status"`
		CPUCores        int    `json:"cpuCores"`
		MemoryUsage     float64 `json:"memoryUsage"`
		GroupName       string `json:"groupName"`
	}

	query := ctx.DB.Table("hosts").
		Select("hosts.id, hosts.name, hosts.ip, hosts.public_ip, hosts.cloud_provider, hosts.cloud_instance_id, hosts.cloud_region, hosts.os, hosts.os_type, hosts.status, hosts.cpu_cores, hosts.memory_usage, COALESCE(asset_group.name, '') as group_name").
		Joins("LEFT JOIN asset_group ON hosts.group_id = asset_group.id").
		Where("hosts.deleted_at IS NULL AND hosts.cloud_provider IS NOT NULL AND hosts.cloud_provider != ''")

	if provider != "" {
		query = query.Where("hosts.cloud_provider = ?", provider)
	}
	if region != "" {
		query = query.Where("hosts.cloud_region LIKE ?", "%"+region+"%")
	}
	if statusFilter != "" {
		switch statusFilter {
		case "running":
			query = query.Where("hosts.status = 1")
		case "stopped":
			query = query.Where("hosts.status = 0")
		}
	}
	if keyword != "" {
		query = query.Where("hosts.name LIKE ? OR hosts.ip LIKE ? OR hosts.cloud_instance_id LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	var instances []InstanceInfo
	if err := query.Order("hosts.id DESC").Limit(limit).Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("查询云主机失败: %v", err)
	}

	// 统计
	var totalInstances int64
	ctx.DB.Table("hosts").Where("deleted_at IS NULL AND cloud_provider IS NOT NULL AND cloud_provider != ''").Count(&totalInstances)

	return map[string]any{
		"instances":      instances,
		"resultCount":    len(instances),
		"totalInstances": totalInstances,
	}, nil
}

// executeCloudImportHosts 从云平台导入主机
func executeCloudImportHosts(ctx biz.SkillContext) (any, error) {
	accountID, ok := ctx.Params["account_id"].(float64)
	if !ok || accountID <= 0 {
		// 如果没有指定 ID，尝试通过名称查找
		accountName, _ := ctx.Params["account_name"].(string)
		if accountName != "" {
			var id uint
			ctx.DB.Table("cloud_accounts").Select("id").Where("name LIKE ? AND deleted_at IS NULL", "%"+accountName+"%").Scan(&id)
			if id > 0 {
				accountID = float64(id)
			} else {
				return nil, fmt.Errorf("未找到名称包含 [%s] 的云账号", accountName)
			}
		} else {
			return nil, fmt.Errorf("请指定云账号 ID 或名称")
		}
	}

	// 查询云账号信息
	type AccountInfo struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Provider string `json:"provider"`
		Region   string `json:"region"`
	}
	var account AccountInfo
	if err := ctx.DB.Table("cloud_accounts").Where("id = ? AND deleted_at IS NULL", uint(accountID)).First(&account).Error; err != nil {
		return nil, fmt.Errorf("云账号 ID=%d 不存在", int(accountID))
	}

	region, _ := ctx.Params["region"].(string)
	if region == "" {
		region = account.Region
	}
	groupName, _ := ctx.Params["group_name"].(string)

	provName := providerMap[account.Provider]
	if provName == "" {
		provName = account.Provider
	}

	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"action":       "import",
			"accountID":    uint(accountID),
			"accountName":  account.Name,
			"provider":     account.Provider,
			"providerName": provName,
			"region":       region,
			"groupName":    groupName,
			"status":       "pending_confirmation",
			"warning":      fmt.Sprintf("⚠️ 将从 %s 账号 [%s] 的 %s 区域导入主机实例到分组 [%s]，请确认执行", provName, account.Name, region, groupName),
		}, nil
	}

	// 确认后：这里实际调用会需要云 SDK，目前创建导入任务记录
	ctx.DB.Exec(`INSERT INTO job_tasks (name, task_type, status, created_by, created_at, updated_at) VALUES (?, 'cloud_import', 'pending', ?, NOW(), NOW())`,
		fmt.Sprintf("导入%s-%s-%s", provName, account.Name, region),
		ctx.UserID)

	return map[string]any{
		"status":       "success",
		"message":      fmt.Sprintf("✅ 云主机导入任务已提交: 从 %s [%s] %s 区域导入", provName, account.Name, region),
		"accountName":  account.Name,
		"provider":     provName,
		"region":       region,
		"targetGroup":  groupName,
	}, nil
}
