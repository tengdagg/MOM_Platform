package skills

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// executeK8sKubectl 通用 K8s 资源操作
func executeK8sKubectl(ctx biz.SkillContext) (any, error) {
	action, _ := ctx.Params["action"].(string)
	resourceType, _ := ctx.Params["resource_type"].(string)
	if action == "" {
		return nil, fmt.Errorf("请指定 action（操作类型）")
	}

	applyRisk := func(result any) any {
		m, ok := result.(map[string]any)
		if !ok {
			return result
		}
		switch action {
		case "get", "describe", "logs", "events", "top", "cluster_status":
			m["effectiveRiskLevel"] = "low"
		case "scale", "restart":
			m["effectiveRiskLevel"] = "high"
		case "delete", "cordon", "uncordon", "drain":
			m["effectiveRiskLevel"] = "critical"
		}
		return m
	}

	// cluster_status 不需要连接 K8s API，直接查数据库
	if action == "cluster_status" {
		result, err := k8sClusterStatus(ctx)
		if err != nil {
			return nil, err
		}
		return applyRisk(result), nil
	}

	if resourceType == "" {
		return nil, fmt.Errorf("请指定 resource_type（资源类型）")
	}

	clusterID, clusterName, err := FindClusterID(ctx.DB, ctx.Params)
	if err != nil {
		return nil, err
	}

	clientset, _, err := GetK8sClientset(ctx.DB, clusterID)
	if err != nil {
		return nil, fmt.Errorf("连接集群 %s 失败: %v", clusterName, err)
	}

	namespace, _ := ctx.Params["namespace"].(string)
	resourceName, _ := ctx.Params["resource_name"].(string)
	labels, _ := ctx.Params["labels"].(string)
	fieldSelector, _ := ctx.Params["field_selector"].(string)

	// 规范化资源类型
	rt := normalizeResourceType(resourceType)

	switch action {
	case "get":
		if rt == "all" {
			result, err := k8sGetAll(clientset, clusterName, namespace, labels, fieldSelector)
			if err != nil {
				return nil, err
			}
			return applyRisk(result), nil
		}
		result, err := k8sGet(clientset, clusterName, rt, namespace, resourceName, labels, fieldSelector)
		if err != nil {
			return nil, err
		}
		return applyRisk(result), nil
	case "describe":
		if resourceName == "" {
			return nil, fmt.Errorf("describe 操作需要指定 resource_name")
		}
		result, err := k8sDescribe(clientset, clusterName, rt, namespace, resourceName)
		if err != nil {
			return nil, err
		}
		return applyRisk(result), nil
	case "logs":
		result, err := k8sLogs(ctx, clientset, clusterName, namespace, resourceName)
		if err != nil {
			return nil, err
		}
		return applyRisk(result), nil
	case "events":
		result, err := k8sEvents(clientset, clusterName, namespace, resourceName)
		if err != nil {
			return nil, err
		}
		return applyRisk(result), nil
	case "scale":
		result, err := k8sScaleGeneric(ctx, clientset, clusterName, rt, namespace, resourceName)
		if err != nil {
			return nil, err
		}
		return applyRisk(result), nil
	case "restart":
		result, err := k8sRestartGeneric(ctx, clientset, clusterName, rt, namespace, resourceName)
		if err != nil {
			return nil, err
		}
		return applyRisk(result), nil
	case "delete":
		result, err := k8sDeleteGeneric(ctx, clientset, clusterName, rt, namespace, resourceName)
		if err != nil {
			return nil, err
		}
		return applyRisk(result), nil
	case "cordon":
		result, err := k8sCordonGeneric(ctx, clientset, clusterName, resourceName, true)
		if err != nil {
			return nil, err
		}
		return applyRisk(result), nil
	case "uncordon":
		result, err := k8sCordonGeneric(ctx, clientset, clusterName, resourceName, false)
		if err != nil {
			return nil, err
		}
		return applyRisk(result), nil
	case "drain":
		result, err := k8sDrainGeneric(ctx, clientset, clusterName, resourceName)
		if err != nil {
			return nil, err
		}
		return applyRisk(result), nil
	case "top":
		result, err := k8sTop(clientset, clusterName, rt, namespace)
		if err != nil {
			return nil, err
		}
		return applyRisk(result), nil
	default:
		return nil, fmt.Errorf("不支持的操作: %s，支持: get/describe/logs/events/scale/restart/delete/cordon/uncordon/drain/top", action)
	}
}

// normalizeResourceType 规范化资源类型名称
func normalizeResourceType(rt string) string {
	rt = strings.ToLower(strings.TrimSpace(rt))
	aliases := map[string]string{
		"pod": "pods", "po": "pods",
		"deployment": "deployments", "deploy": "deployments",
		"service": "services", "svc": "services",
		"configmap": "configmaps", "cm": "configmaps",
		"secret":  "secrets",
		"ingress": "ingresses", "ing": "ingresses",
		"node": "nodes", "no": "nodes",
		"namespace": "namespaces", "ns": "namespaces",
		"persistentvolume": "pv", "persistentvolumeclaim": "pvc",
		"statefulset": "statefulsets", "sts": "statefulsets",
		"daemonset": "daemonsets", "ds": "daemonsets",
		"replicaset": "replicasets", "rs": "replicasets",
		"job":     "jobs",
		"cronjob": "cronjobs", "cj": "cronjobs",
		"endpoint": "endpoints", "ep": "endpoints",
		"serviceaccount": "serviceaccounts", "sa": "serviceaccounts",
		"horizontalpodautoscaler": "hpa",
		"networkpolicy":           "networkpolicies", "netpol": "networkpolicies",
		"storageclass": "storageclasses", "sc": "storageclasses",
		"event": "events", "ev": "events",
		"role":               "roles",
		"rolebinding":        "rolebindings",
		"clusterrole":        "clusterroles",
		"clusterrolebinding": "clusterrolebindings",
	}
	if normalized, ok := aliases[rt]; ok {
		return normalized
	}
	return rt
}

// ============ GET ============

func k8sGet(clientset *kubernetes.Clientset, cluster, rt, ns, name, labels, fieldSelector string) (any, error) {
	ctx := context.Background()
	listOpts := metav1.ListOptions{}
	if labels != "" {
		listOpts.LabelSelector = labels
	}
	if fieldSelector != "" {
		listOpts.FieldSelector = fieldSelector
	}

	switch rt {
	case "pods":
		return getPods(ctx, clientset, cluster, ns, listOpts)
	case "deployments":
		return getDeployments(ctx, clientset, cluster, ns, listOpts)
	case "services":
		return getServices(ctx, clientset, cluster, ns, listOpts)
	case "configmaps":
		return getConfigMaps(ctx, clientset, cluster, ns, listOpts)
	case "secrets":
		return getSecrets(ctx, clientset, cluster, ns, listOpts)
	case "ingresses":
		return getIngresses(ctx, clientset, cluster, ns, listOpts)
	case "nodes":
		return getNodes(ctx, clientset, cluster, listOpts)
	case "namespaces":
		return getNamespaces(ctx, clientset, cluster, listOpts)
	case "pv":
		return getPV(ctx, clientset, cluster, listOpts)
	case "pvc":
		return getPVC(ctx, clientset, cluster, ns, listOpts)
	case "statefulsets":
		return getStatefulSets(ctx, clientset, cluster, ns, listOpts)
	case "daemonsets":
		return getDaemonSets(ctx, clientset, cluster, ns, listOpts)
	case "replicasets":
		return getReplicaSets(ctx, clientset, cluster, ns, listOpts)
	case "jobs":
		return getJobs(ctx, clientset, cluster, ns, listOpts)
	case "cronjobs":
		return getCronJobs(ctx, clientset, cluster, ns, listOpts)
	case "endpoints":
		return getEndpoints(ctx, clientset, cluster, ns, listOpts)
	case "serviceaccounts":
		return getServiceAccounts(ctx, clientset, cluster, ns, listOpts)
	case "hpa":
		return getHPA(ctx, clientset, cluster, ns, listOpts)
	case "events":
		return k8sEvents(clientset, cluster, ns, "")
	default:
		return nil, fmt.Errorf("暂不支持查询资源类型: %s，支持: pods/deployments/services/configmaps/secrets/ingresses/nodes/namespaces/pv/pvc/statefulsets/daemonsets/replicasets/jobs/cronjobs/endpoints/serviceaccounts/hpa/events", rt)
	}
}

// k8sGetAll 获取常用资源列表
func k8sGetAll(c *kubernetes.Clientset, cluster, ns, labels, fieldSelector string) (any, error) {
	// 包含 kubectl get all 默认显示的资源类型
	types := []string{"pods", "services", "deployments", "replicasets", "statefulsets", "daemonsets", "jobs", "cronjobs"}

	result := map[string]any{
		"cluster":      cluster,
		"namespace":    ns,
		"resourceType": "all",
	}

	// 并发查询太复杂，这里顺序查询即可
	for _, rt := range types {
		res, err := k8sGet(c, cluster, rt, ns, "", labels, fieldSelector)
		if err == nil {
			if m, ok := res.(map[string]any); ok {
				if items, ok := m["items"]; ok {
					result[rt] = items
				}
			}
		}
	}

	// 补充 Ingress 方便查看
	if res, err := k8sGet(c, cluster, "ingresses", ns, "", labels, fieldSelector); err == nil {
		if m, ok := res.(map[string]any); ok {
			result["ingresses"] = m["items"]
		}
	}

	return result, nil
}

func getPods(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var list *v1.PodList
	var err error
	if ns != "" {
		list, err = c.CoreV1().Pods(ns).List(ctx, opts)
	} else {
		list, err = c.CoreV1().Pods("").List(ctx, opts)
	}
	if err != nil {
		return nil, fmt.Errorf("查询 Pod 失败: %v", err)
	}

	type PodInfo struct {
		Name       string `json:"name"`
		Namespace  string `json:"namespace"`
		Status     string `json:"status"`
		IP         string `json:"ip"`
		Node       string `json:"node"`
		Restarts   int32  `json:"restarts"`
		Ready      string `json:"ready"`
		Age        string `json:"age"`
		Containers int    `json:"containers"`
	}

	var pods []PodInfo
	for _, p := range list.Items {
		readyCount := 0
		totalRestarts := int32(0)
		for _, cs := range p.Status.ContainerStatuses {
			if cs.Ready {
				readyCount++
			}
			totalRestarts += cs.RestartCount
		}
		total := len(p.Spec.Containers)
		pods = append(pods, PodInfo{
			Name:       p.Name,
			Namespace:  p.Namespace,
			Status:     string(p.Status.Phase),
			IP:         p.Status.PodIP,
			Node:       p.Spec.NodeName,
			Restarts:   totalRestarts,
			Ready:      fmt.Sprintf("%d/%d", readyCount, total),
			Age:        formatAge(p.CreationTimestamp.Time),
			Containers: total,
		})
	}
	return map[string]any{"cluster": cluster, "resourceType": "pods", "total": len(pods), "items": pods}, nil
}

func getDeployments(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var list interface{ Items() int }
	var items []map[string]any

	if ns != "" {
		dList, err := c.AppsV1().Deployments(ns).List(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("查询 Deployment 失败: %v", err)
		}
		for _, d := range dList.Items {
			items = append(items, map[string]any{
				"name": d.Name, "namespace": d.Namespace,
				"replicas": d.Status.Replicas, "ready": d.Status.ReadyReplicas,
				"available": d.Status.AvailableReplicas, "updated": d.Status.UpdatedReplicas,
				"age": formatAge(d.CreationTimestamp.Time), "images": getContainerImages(d.Spec.Template.Spec.Containers),
			})
		}
	} else {
		dList, err := c.AppsV1().Deployments("").List(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("查询 Deployment 失败: %v", err)
		}
		for _, d := range dList.Items {
			items = append(items, map[string]any{
				"name": d.Name, "namespace": d.Namespace,
				"replicas": d.Status.Replicas, "ready": d.Status.ReadyReplicas,
				"available": d.Status.AvailableReplicas, "updated": d.Status.UpdatedReplicas,
				"age": formatAge(d.CreationTimestamp.Time), "images": getContainerImages(d.Spec.Template.Spec.Containers),
			})
		}
	}
	_ = list
	return map[string]any{"cluster": cluster, "resourceType": "deployments", "total": len(items), "items": items}, nil
}

func getServices(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var svcList *v1.ServiceList
	var err error
	if ns != "" {
		svcList, err = c.CoreV1().Services(ns).List(ctx, opts)
	} else {
		svcList, err = c.CoreV1().Services("").List(ctx, opts)
	}
	if err != nil {
		return nil, fmt.Errorf("查询 Service 失败: %v", err)
	}
	var items []map[string]any
	for _, s := range svcList.Items {
		var ports []string
		for _, p := range s.Spec.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
		}
		items = append(items, map[string]any{
			"name": s.Name, "namespace": s.Namespace, "type": string(s.Spec.Type),
			"clusterIP": s.Spec.ClusterIP, "ports": strings.Join(ports, ","),
			"age": formatAge(s.CreationTimestamp.Time),
		})
	}
	return map[string]any{"cluster": cluster, "resourceType": "services", "total": len(items), "items": items}, nil
}

func getConfigMaps(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var cmList *v1.ConfigMapList
	var err error
	if ns != "" {
		cmList, err = c.CoreV1().ConfigMaps(ns).List(ctx, opts)
	} else {
		cmList, err = c.CoreV1().ConfigMaps("").List(ctx, opts)
	}
	if err != nil {
		return nil, fmt.Errorf("查询 ConfigMap 失败: %v", err)
	}
	var items []map[string]any
	for _, cm := range cmList.Items {
		items = append(items, map[string]any{
			"name": cm.Name, "namespace": cm.Namespace,
			"dataKeys": len(cm.Data), "age": formatAge(cm.CreationTimestamp.Time),
		})
	}
	return map[string]any{"cluster": cluster, "resourceType": "configmaps", "total": len(items), "items": items}, nil
}

func getSecrets(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var list *v1.SecretList
	var err error
	if ns != "" {
		list, err = c.CoreV1().Secrets(ns).List(ctx, opts)
	} else {
		list, err = c.CoreV1().Secrets("").List(ctx, opts)
	}
	if err != nil {
		return nil, fmt.Errorf("查询 Secret 失败: %v", err)
	}
	var items []map[string]any
	for _, s := range list.Items {
		items = append(items, map[string]any{
			"name": s.Name, "namespace": s.Namespace, "type": string(s.Type),
			"dataKeys": len(s.Data), "age": formatAge(s.CreationTimestamp.Time),
		})
	}
	return map[string]any{"cluster": cluster, "resourceType": "secrets", "total": len(items), "items": items}, nil
}

func getIngresses(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var items []map[string]any
	if ns != "" {
		list, err := c.NetworkingV1().Ingresses(ns).List(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("查询 Ingress 失败: %v", err)
		}
		for _, ing := range list.Items {
			var hosts, paths []string
			for _, rule := range ing.Spec.Rules {
				hosts = append(hosts, rule.Host)
				if rule.HTTP != nil {
					for _, p := range rule.HTTP.Paths {
						paths = append(paths, p.Path)
					}
				}
			}
			items = append(items, map[string]any{
				"name": ing.Name, "namespace": ing.Namespace,
				"hosts": strings.Join(hosts, ","), "paths": strings.Join(paths, ","),
				"age": formatAge(ing.CreationTimestamp.Time),
			})
		}
	} else {
		list, err := c.NetworkingV1().Ingresses("").List(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("查询 Ingress 失败: %v", err)
		}
		for _, ing := range list.Items {
			var hosts []string
			for _, rule := range ing.Spec.Rules {
				hosts = append(hosts, rule.Host)
			}
			items = append(items, map[string]any{
				"name": ing.Name, "namespace": ing.Namespace,
				"hosts": strings.Join(hosts, ","), "age": formatAge(ing.CreationTimestamp.Time),
			})
		}
	}
	return map[string]any{"cluster": cluster, "resourceType": "ingresses", "total": len(items), "items": items}, nil
}

func getNodes(ctx context.Context, c *kubernetes.Clientset, cluster string, opts metav1.ListOptions) (any, error) {
	list, err := c.CoreV1().Nodes().List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("查询节点失败: %v", err)
	}
	var items []map[string]any
	for _, n := range list.Items {
		status := "NotReady"
		for _, cond := range n.Status.Conditions {
			if cond.Type == v1.NodeReady && cond.Status == v1.ConditionTrue {
				status = "Ready"
			}
		}
		var ips []string
		for _, addr := range n.Status.Addresses {
			if addr.Type == v1.NodeInternalIP {
				ips = append(ips, addr.Address)
			}
		}
		items = append(items, map[string]any{
			"name": n.Name, "status": status, "roles": getNodeRoles(n.Labels),
			"version": n.Status.NodeInfo.KubeletVersion, "os": n.Status.NodeInfo.OSImage,
			"ip": strings.Join(ips, ","), "unschedulable": n.Spec.Unschedulable,
			"age": formatAge(n.CreationTimestamp.Time),
		})
	}
	return map[string]any{"cluster": cluster, "resourceType": "nodes", "total": len(items), "items": items}, nil
}

func getNamespaces(ctx context.Context, c *kubernetes.Clientset, cluster string, opts metav1.ListOptions) (any, error) {
	list, err := c.CoreV1().Namespaces().List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("查询命名空间失败: %v", err)
	}
	var items []map[string]any
	for _, ns := range list.Items {
		items = append(items, map[string]any{
			"name": ns.Name, "status": string(ns.Status.Phase),
			"age": formatAge(ns.CreationTimestamp.Time),
		})
	}
	return map[string]any{"cluster": cluster, "resourceType": "namespaces", "total": len(items), "items": items}, nil
}

func getPV(ctx context.Context, c *kubernetes.Clientset, cluster string, opts metav1.ListOptions) (any, error) {
	list, err := c.CoreV1().PersistentVolumes().List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("查询 PV 失败: %v", err)
	}
	var items []map[string]any
	for _, pv := range list.Items {
		capacity := ""
		if storage, ok := pv.Spec.Capacity[v1.ResourceStorage]; ok {
			capacity = storage.String()
		}
		items = append(items, map[string]any{
			"name": pv.Name, "capacity": capacity,
			"accessModes": pv.Spec.AccessModes, "status": string(pv.Status.Phase),
			"storageClass": pv.Spec.StorageClassName, "age": formatAge(pv.CreationTimestamp.Time),
		})
	}
	return map[string]any{"cluster": cluster, "resourceType": "persistentvolumes", "total": len(items), "items": items}, nil
}

func getPVC(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var list *v1.PersistentVolumeClaimList
	var err error
	if ns != "" {
		list, err = c.CoreV1().PersistentVolumeClaims(ns).List(ctx, opts)
	} else {
		list, err = c.CoreV1().PersistentVolumeClaims("").List(ctx, opts)
	}
	if err != nil {
		return nil, fmt.Errorf("查询 PVC 失败: %v", err)
	}
	var items []map[string]any
	for _, pvc := range list.Items {
		sc := ""
		if pvc.Spec.StorageClassName != nil {
			sc = *pvc.Spec.StorageClassName
		}
		items = append(items, map[string]any{
			"name": pvc.Name, "namespace": pvc.Namespace, "status": string(pvc.Status.Phase),
			"volume": pvc.Spec.VolumeName, "storageClass": sc,
			"age": formatAge(pvc.CreationTimestamp.Time),
		})
	}
	return map[string]any{"cluster": cluster, "resourceType": "pvc", "total": len(items), "items": items}, nil
}

func getStatefulSets(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var items []map[string]any
	if ns != "" {
		list, err := c.AppsV1().StatefulSets(ns).List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, s := range list.Items {
			items = append(items, map[string]any{
				"name": s.Name, "namespace": s.Namespace,
				"replicas": s.Status.Replicas, "ready": s.Status.ReadyReplicas,
				"age": formatAge(s.CreationTimestamp.Time),
			})
		}
	} else {
		list, err := c.AppsV1().StatefulSets("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, s := range list.Items {
			items = append(items, map[string]any{
				"name": s.Name, "namespace": s.Namespace,
				"replicas": s.Status.Replicas, "ready": s.Status.ReadyReplicas,
				"age": formatAge(s.CreationTimestamp.Time),
			})
		}
	}
	return map[string]any{"cluster": cluster, "resourceType": "statefulsets", "total": len(items), "items": items}, nil
}

func getDaemonSets(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var items []map[string]any
	if ns != "" {
		list, err := c.AppsV1().DaemonSets(ns).List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, d := range list.Items {
			items = append(items, map[string]any{
				"name": d.Name, "namespace": d.Namespace,
				"desired": d.Status.DesiredNumberScheduled, "ready": d.Status.NumberReady,
				"age": formatAge(d.CreationTimestamp.Time),
			})
		}
	} else {
		list, err := c.AppsV1().DaemonSets("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, d := range list.Items {
			items = append(items, map[string]any{
				"name": d.Name, "namespace": d.Namespace,
				"desired": d.Status.DesiredNumberScheduled, "ready": d.Status.NumberReady,
				"age": formatAge(d.CreationTimestamp.Time),
			})
		}
	}
	return map[string]any{"cluster": cluster, "resourceType": "daemonsets", "total": len(items), "items": items}, nil
}

func getReplicaSets(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var items []map[string]any
	if ns != "" {
		list, err := c.AppsV1().ReplicaSets(ns).List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, r := range list.Items {
			items = append(items, map[string]any{
				"name": r.Name, "namespace": r.Namespace,
				"replicas": r.Status.Replicas, "ready": r.Status.ReadyReplicas,
				"age": formatAge(r.CreationTimestamp.Time),
			})
		}
	} else {
		list, err := c.AppsV1().ReplicaSets("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, r := range list.Items {
			items = append(items, map[string]any{
				"name": r.Name, "namespace": r.Namespace,
				"replicas": r.Status.Replicas, "ready": r.Status.ReadyReplicas,
				"age": formatAge(r.CreationTimestamp.Time),
			})
		}
	}
	return map[string]any{"cluster": cluster, "resourceType": "replicasets", "total": len(items), "items": items}, nil
}

func getJobs(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var items []map[string]any
	if ns != "" {
		list, err := c.BatchV1().Jobs(ns).List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, j := range list.Items {
			items = append(items, map[string]any{
				"name": j.Name, "namespace": j.Namespace,
				"succeeded": j.Status.Succeeded, "failed": j.Status.Failed,
				"active": j.Status.Active, "age": formatAge(j.CreationTimestamp.Time),
			})
		}
	} else {
		list, err := c.BatchV1().Jobs("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, j := range list.Items {
			items = append(items, map[string]any{
				"name": j.Name, "namespace": j.Namespace,
				"succeeded": j.Status.Succeeded, "failed": j.Status.Failed,
				"active": j.Status.Active, "age": formatAge(j.CreationTimestamp.Time),
			})
		}
	}
	return map[string]any{"cluster": cluster, "resourceType": "jobs", "total": len(items), "items": items}, nil
}

func getCronJobs(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var items []map[string]any
	if ns != "" {
		list, err := c.BatchV1().CronJobs(ns).List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, cj := range list.Items {
			suspend := false
			if cj.Spec.Suspend != nil {
				suspend = *cj.Spec.Suspend
			}
			items = append(items, map[string]any{
				"name": cj.Name, "namespace": cj.Namespace,
				"schedule": cj.Spec.Schedule, "suspend": suspend,
				"active": len(cj.Status.Active), "age": formatAge(cj.CreationTimestamp.Time),
			})
		}
	} else {
		list, err := c.BatchV1().CronJobs("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, cj := range list.Items {
			suspend := false
			if cj.Spec.Suspend != nil {
				suspend = *cj.Spec.Suspend
			}
			items = append(items, map[string]any{
				"name": cj.Name, "namespace": cj.Namespace,
				"schedule": cj.Spec.Schedule, "suspend": suspend,
				"active": len(cj.Status.Active), "age": formatAge(cj.CreationTimestamp.Time),
			})
		}
	}
	return map[string]any{"cluster": cluster, "resourceType": "cronjobs", "total": len(items), "items": items}, nil
}

func getEndpoints(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var list *v1.EndpointsList
	var err error
	if ns != "" {
		list, err = c.CoreV1().Endpoints(ns).List(ctx, opts)
	} else {
		list, err = c.CoreV1().Endpoints("").List(ctx, opts)
	}
	if err != nil {
		return nil, err
	}
	var items []map[string]any
	for _, ep := range list.Items {
		var addrs []string
		for _, subset := range ep.Subsets {
			for _, addr := range subset.Addresses {
				addrs = append(addrs, addr.IP)
			}
		}
		items = append(items, map[string]any{
			"name": ep.Name, "namespace": ep.Namespace,
			"addresses": strings.Join(addrs, ","), "age": formatAge(ep.CreationTimestamp.Time),
		})
	}
	return map[string]any{"cluster": cluster, "resourceType": "endpoints", "total": len(items), "items": items}, nil
}

func getServiceAccounts(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var list *v1.ServiceAccountList
	var err error
	if ns != "" {
		list, err = c.CoreV1().ServiceAccounts(ns).List(ctx, opts)
	} else {
		list, err = c.CoreV1().ServiceAccounts("").List(ctx, opts)
	}
	if err != nil {
		return nil, err
	}
	var items []map[string]any
	for _, sa := range list.Items {
		items = append(items, map[string]any{
			"name": sa.Name, "namespace": sa.Namespace,
			"secrets": len(sa.Secrets), "age": formatAge(sa.CreationTimestamp.Time),
		})
	}
	return map[string]any{"cluster": cluster, "resourceType": "serviceaccounts", "total": len(items), "items": items}, nil
}

func getHPA(ctx context.Context, c *kubernetes.Clientset, cluster, ns string, opts metav1.ListOptions) (any, error) {
	var items []map[string]any
	if ns != "" {
		list, err := c.AutoscalingV2().HorizontalPodAutoscalers(ns).List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, h := range list.Items {
			items = append(items, map[string]any{
				"name": h.Name, "namespace": h.Namespace,
				"minReplicas": h.Spec.MinReplicas, "maxReplicas": h.Spec.MaxReplicas,
				"currentReplicas": h.Status.CurrentReplicas, "desiredReplicas": h.Status.DesiredReplicas,
				"age": formatAge(h.CreationTimestamp.Time),
			})
		}
	} else {
		list, err := c.AutoscalingV2().HorizontalPodAutoscalers("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		for _, h := range list.Items {
			items = append(items, map[string]any{
				"name": h.Name, "namespace": h.Namespace,
				"minReplicas": h.Spec.MinReplicas, "maxReplicas": h.Spec.MaxReplicas,
				"currentReplicas": h.Status.CurrentReplicas, "desiredReplicas": h.Status.DesiredReplicas,
				"age": formatAge(h.CreationTimestamp.Time),
			})
		}
	}
	return map[string]any{"cluster": cluster, "resourceType": "hpa", "total": len(items), "items": items}, nil
}

// ============ DESCRIBE ============

func k8sDescribe(clientset *kubernetes.Clientset, cluster, rt, ns, name string) (any, error) {
	ctx := context.Background()
	if ns == "" {
		ns = "default"
	}

	switch rt {
	case "pods":
		pod, err := clientset.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 Pod %s 失败: %v", name, err)
		}
		result, _ := DiagnosePod(clientset, ns, name)
		result["labels"] = pod.Labels
		result["annotations"] = pod.Annotations
		result["cluster"] = cluster
		return result, nil

	case "deployments":
		d, err := clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 Deployment %s 失败: %v", name, err)
		}
		return map[string]any{
			"cluster": cluster, "name": d.Name, "namespace": d.Namespace,
			"replicas": d.Status.Replicas, "ready": d.Status.ReadyReplicas,
			"available": d.Status.AvailableReplicas, "updated": d.Status.UpdatedReplicas,
			"strategy": string(d.Spec.Strategy.Type), "labels": d.Labels,
			"selector":   d.Spec.Selector.MatchLabels,
			"images":     getContainerImages(d.Spec.Template.Spec.Containers),
			"conditions": d.Status.Conditions,
			"age":        formatAge(d.CreationTimestamp.Time),
		}, nil

	case "services":
		s, err := clientset.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取 Service %s 失败: %v", name, err)
		}
		return map[string]any{
			"cluster": cluster, "name": s.Name, "namespace": s.Namespace,
			"type": string(s.Spec.Type), "clusterIP": s.Spec.ClusterIP,
			"ports": s.Spec.Ports, "selector": s.Spec.Selector,
			"age": formatAge(s.CreationTimestamp.Time),
		}, nil

	case "nodes":
		n, err := clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("获取节点 %s 失败: %v", name, err)
		}
		var conditions []map[string]string
		for _, c := range n.Status.Conditions {
			conditions = append(conditions, map[string]string{
				"type": string(c.Type), "status": string(c.Status), "reason": c.Reason, "message": c.Message,
			})
		}
		return map[string]any{
			"cluster": cluster, "name": n.Name, "roles": getNodeRoles(n.Labels),
			"version": n.Status.NodeInfo.KubeletVersion, "os": n.Status.NodeInfo.OSImage,
			"kernel": n.Status.NodeInfo.KernelVersion, "runtime": n.Status.NodeInfo.ContainerRuntimeVersion,
			"conditions": conditions, "unschedulable": n.Spec.Unschedulable,
			"capacity": n.Status.Capacity, "allocatable": n.Status.Allocatable,
		}, nil

	default:
		return nil, fmt.Errorf("describe 暂不支持资源类型: %s", rt)
	}
}

// ============ EVENTS ============

func k8sEvents(clientset *kubernetes.Clientset, cluster, ns, name string) (any, error) {
	ctx := context.Background()
	opts := metav1.ListOptions{}
	if name != "" {
		opts.FieldSelector = "involvedObject.name=" + name
	}
	var list *v1.EventList
	var err error
	if ns != "" {
		list, err = clientset.CoreV1().Events(ns).List(ctx, opts)
	} else {
		list, err = clientset.CoreV1().Events("").List(ctx, opts)
	}
	if err != nil {
		return nil, fmt.Errorf("查询事件失败: %v", err)
	}
	var events []map[string]any
	for i := len(list.Items) - 1; i >= 0 && len(events) < 50; i-- {
		e := list.Items[i]
		events = append(events, map[string]any{
			"type": e.Type, "reason": e.Reason, "message": e.Message,
			"object":    e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name,
			"namespace": e.Namespace, "count": e.Count,
			"time": e.LastTimestamp.Format("2006-01-02 15:04:05"),
		})
	}
	return map[string]any{"cluster": cluster, "total": len(events), "events": events}, nil
}

// ============ LOGS ============

func k8sLogs(ctx biz.SkillContext, clientset *kubernetes.Clientset, cluster, ns, name string) (any, error) {
	if name == "" {
		return nil, fmt.Errorf("查看日志需要指定 resource_name (Pod 名称)")
	}
	if ns == "" {
		ns = "default"
	}
	container, _ := ctx.Params["container"].(string)
	tailLines := int64(100)
	if tl, ok := ctx.Params["tail_lines"].(float64); ok && tl > 0 {
		tailLines = int64(tl)
	}

	logs, err := GetPodLogs(clientset, ns, name, container, tailLines, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"cluster": cluster, "namespace": ns, "pod": name, "container": container,
		"tailLines": tailLines, "logs": logs,
	}, nil
}

// ============ WRITE OPERATIONS (需确认) ============

func k8sScaleGeneric(ctx biz.SkillContext, clientset *kubernetes.Clientset, cluster, rt, ns, name string) (any, error) {
	if name == "" {
		return nil, fmt.Errorf("scale 操作需要指定 resource_name")
	}
	replicas, ok := ctx.Params["replicas"].(float64)
	if !ok {
		return nil, fmt.Errorf("scale 操作需要指定 replicas（目标副本数）")
	}
	if ns == "" {
		ns = "default"
	}
	resType := "Deployment"
	if rt == "statefulsets" {
		resType = "StatefulSet"
	}

	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"cluster": cluster, "namespace": ns, "resourceType": resType,
			"resourceName": name, "replicas": int(replicas),
			"status":  "pending_confirmation",
			"warning": fmt.Sprintf("⚠️ 将把 %s/%s 在集群 %s(%s) 中的副本数调整为 %d，请确认执行", resType, name, cluster, ns, int(replicas)),
		}, nil
	}

	if err := ScaleWorkload(clientset, ns, resType, name, int32(replicas)); err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "success", "message": fmt.Sprintf("✅ 已成功将 %s/%s 的副本数调整为 %d", resType, name, int(replicas)),
		"cluster": cluster, "namespace": ns,
	}, nil
}

func k8sRestartGeneric(ctx biz.SkillContext, clientset *kubernetes.Clientset, cluster, rt, ns, name string) (any, error) {
	if name == "" {
		return nil, fmt.Errorf("restart 操作需要指定 resource_name")
	}
	if ns == "" {
		ns = "default"
	}
	resType := "Deployment"
	switch rt {
	case "statefulsets":
		resType = "StatefulSet"
	case "daemonsets":
		resType = "DaemonSet"
	}

	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"cluster": cluster, "namespace": ns, "resourceType": resType,
			"resourceName": name, "status": "pending_confirmation",
			"warning": fmt.Sprintf("⚠️ 将滚动重启 %s/%s (集群: %s, 命名空间: %s)，请确认执行", resType, name, cluster, ns),
		}, nil
	}

	if err := RestartWorkload(clientset, ns, resType, name); err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "success", "message": fmt.Sprintf("✅ 已触发 %s/%s 的滚动重启", resType, name),
		"cluster": cluster,
	}, nil
}

func k8sDeleteGeneric(ctx biz.SkillContext, clientset *kubernetes.Clientset, cluster, rt, ns, name string) (any, error) {
	if name == "" {
		return nil, fmt.Errorf("delete 操作需要指定 resource_name")
	}
	if ns == "" {
		ns = "default"
	}
	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"cluster": cluster, "namespace": ns, "resourceType": rt,
			"resourceName": name, "status": "pending_confirmation",
			"warning": fmt.Sprintf("⚠️ 将删除 %s/%s (集群: %s, 命名空间: %s)，此操作不可撤回！请确认执行", rt, name, cluster, ns),
		}, nil
	}

	kctx := context.Background()
	var err error
	switch rt {
	case "pods":
		err = clientset.CoreV1().Pods(ns).Delete(kctx, name, metav1.DeleteOptions{})
	case "deployments":
		err = clientset.AppsV1().Deployments(ns).Delete(kctx, name, metav1.DeleteOptions{})
	case "services":
		err = clientset.CoreV1().Services(ns).Delete(kctx, name, metav1.DeleteOptions{})
	case "configmaps":
		err = clientset.CoreV1().ConfigMaps(ns).Delete(kctx, name, metav1.DeleteOptions{})
	case "secrets":
		err = clientset.CoreV1().Secrets(ns).Delete(kctx, name, metav1.DeleteOptions{})
	case "statefulsets":
		err = clientset.AppsV1().StatefulSets(ns).Delete(kctx, name, metav1.DeleteOptions{})
	case "daemonsets":
		err = clientset.AppsV1().DaemonSets(ns).Delete(kctx, name, metav1.DeleteOptions{})
	case "jobs":
		err = clientset.BatchV1().Jobs(ns).Delete(kctx, name, metav1.DeleteOptions{})
	case "cronjobs":
		err = clientset.BatchV1().CronJobs(ns).Delete(kctx, name, metav1.DeleteOptions{})
	default:
		return nil, fmt.Errorf("delete 暂不支持资源类型: %s", rt)
	}
	if err != nil {
		return nil, fmt.Errorf("删除 %s/%s 失败: %v", rt, name, err)
	}
	return map[string]any{
		"status": "success", "message": fmt.Sprintf("✅ 已成功删除 %s/%s", rt, name),
		"cluster": cluster, "namespace": ns,
	}, nil
}

func k8sCordonGeneric(ctx biz.SkillContext, clientset *kubernetes.Clientset, cluster, name string, unschedulable bool) (any, error) {
	if name == "" {
		return nil, fmt.Errorf("请指定节点名称 (resource_name)")
	}
	action := "cordon"
	desc := "设为不可调度"
	if !unschedulable {
		action = "uncordon"
		desc = "恢复可调度"
	}
	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"cluster": cluster, "nodeName": name, "action": action,
			"status":  "pending_confirmation",
			"warning": fmt.Sprintf("⚠️ 将对节点 %s 执行 %s（%s），请确认执行", name, action, desc),
		}, nil
	}
	if err := CordonNode(clientset, name, unschedulable); err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "success", "message": fmt.Sprintf("✅ 节点 %s 已%s", name, desc),
	}, nil
}

func k8sDrainGeneric(ctx biz.SkillContext, clientset *kubernetes.Clientset, cluster, name string) (any, error) {
	if name == "" {
		return nil, fmt.Errorf("请指定节点名称 (resource_name)")
	}
	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"cluster": cluster, "nodeName": name, "action": "drain",
			"status":  "pending_confirmation",
			"warning": fmt.Sprintf("⚠️ 危险操作！将排空节点 %s（驱逐所有 Pod 并设为不可调度），请确认执行", name),
		}, nil
	}
	evicted, err := DrainNode(clientset, name)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "success", "message": fmt.Sprintf("✅ 节点 %s 已排空，驱逐了 %d 个 Pod", name, evicted),
	}, nil
}

func k8sTop(clientset *kubernetes.Clientset, cluster, rt, ns string) (any, error) {
	// top 需要 metrics-server，返回资源使用统计
	return map[string]any{
		"cluster": cluster, "resourceType": rt,
		"note": "资源使用量查询需要集群安装 metrics-server，请通过 Kubernetes Dashboard 查看详细资源使用数据",
	}, nil
}

// ============ HELPERS ============

func formatAge(t time.Time) string {
	d := time.Since(t)
	if d.Hours() > 24*365 {
		return fmt.Sprintf("%dy", int(d.Hours()/24/365))
	}
	if d.Hours() > 24 {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	if d.Hours() >= 1 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	if d.Minutes() >= 1 {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

func getContainerImages(containers []v1.Container) []string {
	var images []string
	for _, c := range containers {
		images = append(images, c.Image)
	}
	return images
}

func getNodeRoles(labels map[string]string) string {
	var roles []string
	for k := range labels {
		if strings.HasPrefix(k, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(k, "node-role.kubernetes.io/")
			if role == "" {
				role = "worker"
			}
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		return "<none>"
	}
	return strings.Join(roles, ",")
}

// k8sClusterStatus 查询集群概览（从数据库快速查询，不需要连接 K8s API）
func k8sClusterStatus(ctx biz.SkillContext) (any, error) {
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
