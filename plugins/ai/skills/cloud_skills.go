package skills

import (
	"encoding/json"
	"fmt"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// CloudListAccountsSkill 查询云账号列表
type CloudListAccountsSkill struct{}

func (s *CloudListAccountsSkill) Name() string        { return "cloud.list_accounts" }
func (s *CloudListAccountsSkill) Description() string {
	return "查询所有云平台账号列表，包括账号名称、云厂商、状态等信息"
}
func (s *CloudListAccountsSkill) RiskLevel() string   { return "low" }
func (s *CloudListAccountsSkill) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"provider": {"type": "string", "description": "按云厂商筛选: aliyun / tencent / aws / jdcloud / baidu / ksyun"}
		}
	}`)
}

func (s *CloudListAccountsSkill) Execute(ctx biz.SkillContext) (any, error) {
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

// CloudListInstancesSkill 查询云主机实例
type CloudListInstancesSkill struct{}

func (s *CloudListInstancesSkill) Name() string        { return "cloud.list_instances" }
func (s *CloudListInstancesSkill) Description() string {
	return "查询已导入的云主机实例列表，支持按云厂商和区域筛选"
}
func (s *CloudListInstancesSkill) RiskLevel() string   { return "low" }
func (s *CloudListInstancesSkill) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"provider": {"type": "string", "description": "云厂商: aliyun / tencent / aws 等"},
			"region": {"type": "string", "description": "区域"},
			"limit": {"type": "integer", "description": "返回数量，默认 20", "default": 20}
		}
	}`)
}

func (s *CloudListInstancesSkill) Execute(ctx biz.SkillContext) (any, error) {
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

// RegisterCloudSkills 注册云账号 Skills
func RegisterCloudSkills(registry *biz.ToolRegistry) {
	registry.Register(&CloudListAccountsSkill{})
	registry.Register(&CloudListInstancesSkill{})
}
