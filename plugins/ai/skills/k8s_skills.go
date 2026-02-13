package skills

import (
	"encoding/json"
	"fmt"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// K8sClusterStatusSkill 查询集群状态
type K8sClusterStatusSkill struct{}

func (s *K8sClusterStatusSkill) Name() string        { return "k8s.cluster_status" }
func (s *K8sClusterStatusSkill) Description() string {
	return "查询所有 Kubernetes 集群的状态概览，包括集群名称、版本、节点数、Pod 数、状态等信息"
}
func (s *K8sClusterStatusSkill) RiskLevel() string   { return "low" }
func (s *K8sClusterStatusSkill) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"cluster_name": {"type": "string", "description": "集群名称筛选（可选）"}
		}
	}`)
}

func (s *K8sClusterStatusSkill) Execute(ctx biz.SkillContext) (any, error) {
	clusterName, _ := ctx.Params["cluster_name"].(string)

	type ClusterInfo struct {
		ID          uint   `json:"id"`
		Name        string `json:"name"`
		Alias       string `json:"alias"`
		APIEndpoint string `json:"apiEndpoint"`
		Version     string `json:"version"`
		Status      int    `json:"status"`
		Provider    string `json:"provider"`
		Region      string `json:"region"`
		NodeCount   int    `json:"nodeCount"`
		PodCount    int    `json:"podCount"`
	}

	query := ctx.DB.Table("k8s_clusters")
	if clusterName != "" {
		query = query.Where("name LIKE ? OR alias LIKE ?", "%"+clusterName+"%", "%"+clusterName+"%")
	}

	var clusters []ClusterInfo
	if err := query.Find(&clusters).Error; err != nil {
		return nil, fmt.Errorf("查询集群信息失败: %v", err)
	}

	statusMap := map[int]string{1: "正常", 2: "连接失败", 3: "不可用"}
	type ClusterVO struct {
		ClusterInfo
		StatusText string `json:"statusText"`
	}

	var result []ClusterVO
	normalCount := 0
	for _, c := range clusters {
		vo := ClusterVO{ClusterInfo: c, StatusText: statusMap[c.Status]}
		if c.Status == 1 {
			normalCount++
		}
		result = append(result, vo)
	}

	return map[string]any{
		"clusters":    result,
		"total":       len(result),
		"normal":      normalCount,
		"abnormal":    len(result) - normalCount,
	}, nil
}

// K8sListResourcesSkill 查询 K8s 资源
type K8sListResourcesSkill struct{}

func (s *K8sListResourcesSkill) Name() string        { return "k8s.list_resources" }
func (s *K8sListResourcesSkill) Description() string {
	return "查询 Kubernetes 集群中的资源信息概览（从数据库缓存中获取）。返回集群的节点数和 Pod 数统计"
}
func (s *K8sListResourcesSkill) RiskLevel() string   { return "low" }
func (s *K8sListResourcesSkill) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"cluster_name": {"type": "string", "description": "集群名称（必填）"},
			"resource_type": {"type": "string", "description": "资源类型: nodes / pods / deployments / services", "default": "pods"}
		},
		"required": ["cluster_name"]
	}`)
}

func (s *K8sListResourcesSkill) Execute(ctx biz.SkillContext) (any, error) {
	clusterName, _ := ctx.Params["cluster_name"].(string)
	if clusterName == "" {
		return nil, fmt.Errorf("请提供集群名称")
	}

	var cluster struct {
		ID        uint   `json:"id"`
		Name      string `json:"name"`
		NodeCount int    `json:"nodeCount"`
		PodCount  int    `json:"podCount"`
		Version   string `json:"version"`
		Status    int    `json:"status"`
	}

	if err := ctx.DB.Table("k8s_clusters").Where("name LIKE ?", "%"+clusterName+"%").First(&cluster).Error; err != nil {
		return nil, fmt.Errorf("集群 %s 不存在", clusterName)
	}

	return map[string]any{
		"cluster":   cluster.Name,
		"nodeCount": cluster.NodeCount,
		"podCount":  cluster.PodCount,
		"version":   cluster.Version,
		"status":    cluster.Status,
		"message":   "注意: 详细的实时资源信息需要通过 K8s API 直接查询，当前返回的是数据库缓存的概览数据",
	}, nil
}

// RegisterK8sSkills 注册 Kubernetes Skills
func RegisterK8sSkills(registry *biz.ToolRegistry) {
	registry.Register(&K8sClusterStatusSkill{})
	registry.Register(&K8sListResourcesSkill{})
}
