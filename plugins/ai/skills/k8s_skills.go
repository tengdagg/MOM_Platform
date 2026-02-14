package skills

import (
	"fmt"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// RegisterK8sSkills 注册 Kubernetes Skills
func RegisterK8sSkills(registry *biz.ToolRegistry) {
	registry.Register(MustLoadBuiltinSkill("k8s.cluster_status", executeK8sClusterStatus))
	registry.Register(MustLoadBuiltinSkill("k8s.list_resources", executeK8sListResources))
}

// executeK8sClusterStatus 查询集群状态
func executeK8sClusterStatus(ctx biz.SkillContext) (any, error) {
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
		"clusters": result,
		"total":    len(result),
		"normal":   normalCount,
		"abnormal": len(result) - normalCount,
	}, nil
}

// executeK8sListResources 查询 K8s 资源
func executeK8sListResources(ctx biz.SkillContext) (any, error) {
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
