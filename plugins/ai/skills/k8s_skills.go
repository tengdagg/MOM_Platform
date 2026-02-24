package skills

import (
	"fmt"

	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
	"github.com/ydcloud-dy/mom/plugins/kubernetes/service"
)

// RegisterK8sSkills 注册 Kubernetes Skills
func RegisterK8sSkills(registry *biz.ToolRegistry) {
	// k8s.kubectl 是万能 K8s 操作 Skill，覆盖 get/describe/logs/scale/restart/delete/cordon/drain 等
	registry.Register(MustLoadBuiltinSkill("k8s.kubectl", executeK8sKubectl))
	// 以下是特定场景的 Skill，参数定义更精确，帮助 LLM 更准确地选择
	registry.Register(MustLoadBuiltinSkill("k8s.scale", executeK8sScale))
	registry.Register(MustLoadBuiltinSkill("k8s.restart", executeK8sRestart))
	registry.Register(MustLoadBuiltinSkill("k8s.diagnose", executeK8sDiagnose))
	registry.Register(MustLoadBuiltinSkill("k8s.node_manage", executeK8sNodeManage))
	registry.Register(MustLoadBuiltinSkill("k8s.log_query", executeK8sLogQuery))
	registry.Register(MustLoadBuiltinSkill("k8s.helm_manage", executeK8sHelmManage))
}

// isConfirmed 检查是否已确认执行
func isConfirmed(params map[string]any) bool {
	if confirmed, ok := params["confirmed"].(bool); ok && confirmed {
		return true
	}
	// 兼容字符串 "true"
	if confirmed, ok := params["confirmed"].(string); ok && confirmed == "true" {
		return true
	}
	return false
}

// extractValuesAsYAML 从参数中提取 values，支持 string（YAML）和 map（JSON 对象）两种格式
// AI 模型通常传 JSON 对象如 {"image": {"tag": "1.29"}}，需要转成 YAML 字符串
func extractValuesAsYAML(v any) string {
	if v == nil {
		return ""
	}
	// 如果是字符串，直接返回（认为是 YAML）
	if s, ok := v.(string); ok {
		return s
	}
	// 如果是 map（AI 传的 JSON 对象），转成 YAML
	if m, ok := v.(map[string]any); ok {
		data, err := yaml.Marshal(m)
		if err != nil {
			return ""
		}
		return string(data)
	}
	return ""
}

// executeK8sScale 扩缩容工作负载
func executeK8sScale(ctx biz.SkillContext) (any, error) {
	clusterID, clusterName, err := FindClusterID(ctx.DB, ctx.Params)
	if err != nil {
		return nil, err
	}
	resourceName, _ := ctx.Params["resource_name"].(string)
	if resourceName == "" {
		return nil, fmt.Errorf("请指定工作负载名称")
	}
	replicas, ok := ctx.Params["replicas"].(float64)
	if !ok {
		return nil, fmt.Errorf("请指定目标副本数")
	}
	namespace, _ := ctx.Params["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}
	resourceType, _ := ctx.Params["resource_type"].(string)
	if resourceType == "" {
		resourceType = "Deployment"
	}

	// 未确认 → 返回待确认信息
	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"action":       "scale",
			"cluster":      clusterName,
			"clusterID":    clusterID,
			"namespace":    namespace,
			"resourceType": resourceType,
			"resourceName": resourceName,
			"replicas":     int(replicas),
			"status":       "pending_confirmation",
			"warning":      fmt.Sprintf("⚠️ 将把 %s/%s 在集群 %s(%s) 中的副本数调整为 %d，请确认执行", resourceType, resourceName, clusterName, namespace, int(replicas)),
		}, nil
	}

	// 已确认 → 真正执行
	clientset, _, err := GetK8sClientset(ctx.DB, clusterID)
	if err != nil {
		return nil, fmt.Errorf("连接集群失败: %v", err)
	}

	if err := ScaleWorkload(clientset, namespace, resourceType, resourceName, int32(replicas)); err != nil {
		return nil, fmt.Errorf("扩缩容失败: %v", err)
	}

	return map[string]any{
		"status":       "success",
		"message":      fmt.Sprintf("✅ 已成功将 %s/%s 的副本数调整为 %d", resourceType, resourceName, int(replicas)),
		"cluster":      clusterName,
		"namespace":    namespace,
		"resourceType": resourceType,
		"resourceName": resourceName,
		"replicas":     int(replicas),
	}, nil
}

// executeK8sRestart 重启工作负载
func executeK8sRestart(ctx biz.SkillContext) (any, error) {
	clusterID, clusterName, err := FindClusterID(ctx.DB, ctx.Params)
	if err != nil {
		return nil, err
	}
	resourceName, _ := ctx.Params["resource_name"].(string)
	if resourceName == "" {
		return nil, fmt.Errorf("请指定工作负载名称")
	}
	namespace, _ := ctx.Params["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}
	resourceType, _ := ctx.Params["resource_type"].(string)
	if resourceType == "" {
		resourceType = "Deployment"
	}

	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"action":       "restart",
			"cluster":      clusterName,
			"clusterID":    clusterID,
			"namespace":    namespace,
			"resourceType": resourceType,
			"resourceName": resourceName,
			"status":       "pending_confirmation",
			"warning":      fmt.Sprintf("⚠️ 将滚动重启 %s/%s (集群: %s, 命名空间: %s)，请确认执行", resourceType, resourceName, clusterName, namespace),
		}, nil
	}

	clientset, _, err := GetK8sClientset(ctx.DB, clusterID)
	if err != nil {
		return nil, fmt.Errorf("连接集群失败: %v", err)
	}

	if err := RestartWorkload(clientset, namespace, resourceType, resourceName); err != nil {
		return nil, fmt.Errorf("重启失败: %v", err)
	}

	return map[string]any{
		"status":  "success",
		"message": fmt.Sprintf("✅ 已触发 %s/%s 的滚动重启", resourceType, resourceName),
		"cluster": clusterName,
	}, nil
}

// executeK8sDiagnose 诊断 Pod/节点问题（低风险，直接执行）
func executeK8sDiagnose(ctx biz.SkillContext) (any, error) {
	clusterID, clusterName, err := FindClusterID(ctx.DB, ctx.Params)
	if err != nil {
		return nil, err
	}
	podName, _ := ctx.Params["pod_name"].(string)
	nodeName, _ := ctx.Params["node_name"].(string)
	namespace, _ := ctx.Params["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	if podName == "" && nodeName == "" {
		return nil, fmt.Errorf("请指定要诊断的 Pod 名称或节点名称")
	}

	clientset, _, err := GetK8sClientset(ctx.DB, clusterID)
	if err != nil {
		return nil, fmt.Errorf("连接集群失败: %v", err)
	}

	if podName != "" {
		result, err := DiagnosePod(clientset, namespace, podName)
		if err != nil {
			return nil, err
		}
		result["cluster"] = clusterName
		return result, nil
	}

	// 节点诊断
	node, err := clientset.CoreV1().Nodes().Get(ctx.Context(), nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点 %s 失败: %v", nodeName, err)
	}

	var conditions []map[string]string
	for _, c := range node.Status.Conditions {
		conditions = append(conditions, map[string]string{
			"type":    string(c.Type),
			"status":  string(c.Status),
			"reason":  c.Reason,
			"message": c.Message,
		})
	}

	return map[string]any{
		"cluster":        clusterName,
		"nodeName":       nodeName,
		"conditions":     conditions,
		"unschedulable":  node.Spec.Unschedulable,
		"kubeletVersion": node.Status.NodeInfo.KubeletVersion,
		"osImage":        node.Status.NodeInfo.OSImage,
	}, nil
}

// executeK8sNodeManage 节点管理
func executeK8sNodeManage(ctx biz.SkillContext) (any, error) {
	clusterID, clusterName, err := FindClusterID(ctx.DB, ctx.Params)
	if err != nil {
		return nil, err
	}
	nodeName, _ := ctx.Params["node_name"].(string)
	action, _ := ctx.Params["action"].(string)
	if nodeName == "" || action == "" {
		return nil, fmt.Errorf("请指定节点名称和操作类型 (cordon/uncordon/drain)")
	}

	actionDesc := map[string]string{
		"cordon":   "设为不可调度（新 Pod 不会被调度到此节点）",
		"uncordon": "恢复可调度",
		"drain":    "排空节点（驱逐所有 Pod 并设为不可调度）",
	}
	desc := actionDesc[action]
	if desc == "" {
		return nil, fmt.Errorf("不支持的操作: %s，支持: cordon/uncordon/drain", action)
	}

	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"action":   action,
			"cluster":  clusterName,
			"nodeName": nodeName,
			"status":   "pending_confirmation",
			"warning":  fmt.Sprintf("⚠️ 危险操作: 将对节点 %s 执行 %s - %s，请确认执行", nodeName, action, desc),
		}, nil
	}

	clientset, _, err := GetK8sClientset(ctx.DB, clusterID)
	if err != nil {
		return nil, fmt.Errorf("连接集群失败: %v", err)
	}

	switch action {
	case "cordon":
		if err := CordonNode(clientset, nodeName, true); err != nil {
			return nil, fmt.Errorf("cordon 失败: %v", err)
		}
		return map[string]any{"status": "success", "message": fmt.Sprintf("✅ 节点 %s 已设为不可调度", nodeName)}, nil
	case "uncordon":
		if err := CordonNode(clientset, nodeName, false); err != nil {
			return nil, fmt.Errorf("uncordon 失败: %v", err)
		}
		return map[string]any{"status": "success", "message": fmt.Sprintf("✅ 节点 %s 已恢复可调度", nodeName)}, nil
	case "drain":
		evicted, err := DrainNode(clientset, nodeName)
		if err != nil {
			return nil, fmt.Errorf("drain 失败: %v", err)
		}
		return map[string]any{"status": "success", "message": fmt.Sprintf("✅ 节点 %s 已排空，驱逐了 %d 个 Pod", nodeName, evicted)}, nil
	}

	return nil, fmt.Errorf("未知操作: %s", action)
}

// executeK8sLogQuery 查询 Pod 日志（低风险，直接执行）
func executeK8sLogQuery(ctx biz.SkillContext) (any, error) {
	clusterID, clusterName, err := FindClusterID(ctx.DB, ctx.Params)
	if err != nil {
		return nil, err
	}
	podName, _ := ctx.Params["pod_name"].(string)
	if podName == "" {
		return nil, fmt.Errorf("请指定 Pod 名称")
	}
	namespace, _ := ctx.Params["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}
	container, _ := ctx.Params["container"].(string)
	tailLines := int64(100)
	if tl, ok := ctx.Params["tail_lines"].(float64); ok && tl > 0 {
		tailLines = int64(tl)
	}
	previous, _ := ctx.Params["previous"].(bool)

	clientset, _, err := GetK8sClientset(ctx.DB, clusterID)
	if err != nil {
		return nil, fmt.Errorf("连接集群失败: %v", err)
	}

	logs, err := GetPodLogs(clientset, namespace, podName, container, tailLines, previous)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"cluster":   clusterName,
		"namespace": namespace,
		"podName":   podName,
		"container": container,
		"tailLines": tailLines,
		"logs":      logs,
	}, nil
}

// executeK8sHelmManage Helm Release 管理
func executeK8sHelmManage(ctx biz.SkillContext) (any, error) {
	clusterID, clusterName, err := FindClusterID(ctx.DB, ctx.Params)
	if err != nil {
		return nil, err
	}
	action, _ := ctx.Params["action"].(string)
	if action == "" {
		return nil, fmt.Errorf("请指定操作: list/install/upgrade/uninstall/status")
	}
	namespace, _ := ctx.Params["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}
	releaseName, _ := ctx.Params["release_name"].(string)
	chartName, _ := ctx.Params["chart_name"].(string)
	chartVersion, _ := ctx.Params["chart_version"].(string)
	values := extractValuesAsYAML(ctx.Params["values"])

	// 初始化 Helm Service
	// 注意：这里需要引入 plugins/kubernetes/service 包，请确保已导入
	clusterService := service.NewClusterService(ctx.DB)
	helmService := service.NewHelmService(ctx.DB, clusterService)

	result := map[string]any{
		"action":    action,
		"cluster":   clusterName,
		"namespace": namespace,
	}

	switch action {
	case "list":
		releases, err := helmService.ListReleases(ctx.Context(), uint(clusterID))
		if err != nil {
			return nil, fmt.Errorf("查询 Helm Release 列表失败: %v", err)
		}
		// 过滤 namespace
		var filteredReleases []service.ReleaseInfo
		for _, r := range releases {
			if namespace == "" || r.Namespace == namespace {
				filteredReleases = append(filteredReleases, r)
			}
		}
		result["releases"] = filteredReleases
		result["total"] = len(filteredReleases)
		result["message"] = fmt.Sprintf("查询到 %d 个 Helm Release", len(filteredReleases))

	case "install":
		if chartName == "" || releaseName == "" {
			return nil, fmt.Errorf("安装 Release 需要指定 chart_name 和 release_name")
		}
		// 获取 repoId - AI 可能不知道 RepoID，这里简化处理：
		// 1. 如果有 repo_id 参数直接使用
		// 2. 如果没有，尝试从所有 Repo 中查找 chart (暂不支持，需要更复杂的逻辑)
		// 目前暂不支持 AI 直接安装，除非它知道 repoId。
		// 让 AI 返回提示信息，建议用户通过 UI 操作，或者我们后续增强支持通过 Chart 名称自动查找 Repo
		repoIDFloat, ok := ctx.Params["repo_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("安装操作需要指定 repo_id (Helm 仓库 ID)")
		}
		repoID := uint(repoIDFloat)

		if !isConfirmed(ctx.Params) {
			result["releaseName"] = releaseName
			result["chartName"] = chartName
			result["chartVersion"] = chartVersion
			result["repoId"] = repoID
			result["status"] = "pending_confirmation"
			result["warning"] = fmt.Sprintf("⚠️ 将在集群 %s 安装 Helm Release: %s (Chart: %s)，请确认执行", clusterName, releaseName, chartName)
			return result, nil
		}

		req := &service.InstallReleaseRequest{
			ClusterID:   uint(clusterID),
			RepoID:      repoID,
			ChartName:   chartName,
			Version:     chartVersion,
			ReleaseName: releaseName,
			Namespace:   namespace,
			Values:      values,
		}
		release, err := helmService.InstallRelease(ctx.Context(), req)
		if err != nil {
			return nil, fmt.Errorf("安装 Helm Release 失败: %v", err)
		}
		result["status"] = "success"
		result["message"] = fmt.Sprintf("✅ Helm Release %s 安装成功", releaseName)
		result["data"] = release

	case "upgrade":
		if releaseName == "" {
			return nil, fmt.Errorf("升级 Release 需要指定 release_name")
		}
		if !isConfirmed(ctx.Params) {
			result["releaseName"] = releaseName
			result["chartName"] = chartName
			result["chartVersion"] = chartVersion
			result["status"] = "pending_confirmation"
			result["warning"] = fmt.Sprintf("⚠️ 将升级集群 %s 的 Helm Release: %s，请确认执行", clusterName, releaseName)
			return result, nil
		}

		release, err := helmService.UpgradeRelease(ctx.Context(), uint(clusterID), namespace, releaseName, values)
		if err != nil {
			return nil, fmt.Errorf("升级 Helm Release 失败: %v", err)
		}

		result["status"] = "success"
		result["message"] = fmt.Sprintf("✅ Helm Release %s 升级成功", releaseName)
		result["data"] = release

	case "uninstall":
		if releaseName == "" {
			return nil, fmt.Errorf("卸载 Release 需要指定 release_name")
		}
		if !isConfirmed(ctx.Params) {
			result["releaseName"] = releaseName
			result["status"] = "pending_confirmation"
			result["warning"] = fmt.Sprintf("⚠️ 将卸载集群 %s 的 Helm Release: %s，请确认执行", clusterName, releaseName)
			return result, nil
		}

		if err := helmService.UninstallRelease(ctx.Context(), uint(clusterID), namespace, releaseName); err != nil {
			return nil, fmt.Errorf("卸载 Helm Release 失败: %v", err)
		}
		result["status"] = "success"
		result["message"] = fmt.Sprintf("✅ Helm Release %s 已成功卸载", releaseName)

	case "status":
		if releaseName == "" {
			return nil, fmt.Errorf("查看状态需要指定 release_name")
		}
		release, err := helmService.GetRelease(ctx.Context(), uint(clusterID), namespace, releaseName)
		if err != nil {
			return nil, fmt.Errorf("获取 Helm Release 状态失败: %v", err)
		}
		result["releaseName"] = releaseName
		result["status"] = release.Status
		result["revision"] = release.Revision
		result["updated"] = release.Updated
		result["chart"] = release.Chart
		result["appVersion"] = release.AppVersion
		// 避免返回过多内容，截断 manifest
		if len(release.Manifest) > 1000 {
			release.Manifest = release.Manifest[:1000] + "...(truncated)"
		}
		result["data"] = release
		result["message"] = fmt.Sprintf("Helm Release %s 状态: %s", releaseName, release.Status)

	default:
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}

	return result, nil
}
