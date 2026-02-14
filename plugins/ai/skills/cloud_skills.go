package skills

import (
	"fmt"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// RegisterCloudSkills 注册云账号 Skills
func RegisterCloudSkills(registry *biz.ToolRegistry) {
	registry.Register(MustLoadBuiltinSkill("cloud.list_accounts", executeCloudListAccounts))
	registry.Register(MustLoadBuiltinSkill("cloud.list_instances", executeCloudListInstances))
}

// executeCloudListAccounts 查询云账号列表
func executeCloudListAccounts(ctx biz.SkillContext) (any, error) {
	provider, _ := ctx.Params["provider"].(string)

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

	var accounts []AccountInfo
	if err := query.Order("id ASC").Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("查询云账号失败: %v", err)
	}

	providerMap := map[string]string{
		"aliyun": "阿里云", "tencent": "腾讯云", "aws": "AWS",
		"jdcloud": "京东云", "baidu": "百度云", "ksyun": "金山云",
	}

	type AccountVO struct {
		AccountInfo
		ProviderName string `json:"providerName"`
		StatusText   string `json:"statusText"`
	}

	var result []AccountVO
	for _, a := range accounts {
		vo := AccountVO{
			AccountInfo:  a,
			ProviderName: providerMap[a.Provider],
			StatusText:   "启用",
		}
		if a.Status == 0 {
			vo.StatusText = "禁用"
		}
		result = append(result, vo)
	}

	return map[string]any{
		"accounts": result,
		"total":    len(result),
	}, nil
}

// executeCloudListInstances 查询云主机实例
func executeCloudListInstances(ctx biz.SkillContext) (any, error) {
	provider, _ := ctx.Params["provider"].(string)
	limit := 20
	if l, ok := ctx.Params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	type InstanceInfo struct {
		ID              uint   `json:"id"`
		Name            string `json:"name"`
		IP              string `json:"ip"`
		CloudProvider   string `json:"cloudProvider"`
		CloudInstanceID string `json:"cloudInstanceId"`
		OS              string `json:"os"`
		Status          int    `json:"status"`
	}

	query := ctx.DB.Table("hosts").
		Where("deleted_at IS NULL AND type = 'cloud'")
	if provider != "" {
		query = query.Where("cloud_provider = ?", provider)
	}

	var instances []InstanceInfo
	if err := query.Order("id DESC").Limit(limit).Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("查询云主机失败: %v", err)
	}

	return map[string]any{
		"instances": instances,
		"total":     len(instances),
	}, nil
}
