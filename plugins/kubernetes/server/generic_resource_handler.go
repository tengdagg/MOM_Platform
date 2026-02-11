package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// kindToGVR 常见 Kubernetes 资源 Kind 到 GVR 的映射
var kindToGVR = map[string]schema.GroupVersionResource{
	"ServiceAccount":                  {Group: "", Version: "v1", Resource: "serviceaccounts"},
	"Service":                         {Group: "", Version: "v1", Resource: "services"},
	"ConfigMap":                       {Group: "", Version: "v1", Resource: "configmaps"},
	"Secret":                          {Group: "", Version: "v1", Resource: "secrets"},
	"PersistentVolumeClaim":           {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
	"PersistentVolume":                {Group: "", Version: "v1", Resource: "persistentvolumes"},
	"Pod":                             {Group: "", Version: "v1", Resource: "pods"},
	"Namespace":                       {Group: "", Version: "v1", Resource: "namespaces"},
	"Endpoints":                       {Group: "", Version: "v1", Resource: "endpoints"},
	"Deployment":                      {Group: "apps", Version: "v1", Resource: "deployments"},
	"StatefulSet":                     {Group: "apps", Version: "v1", Resource: "statefulsets"},
	"DaemonSet":                       {Group: "apps", Version: "v1", Resource: "daemonsets"},
	"ReplicaSet":                      {Group: "apps", Version: "v1", Resource: "replicasets"},
	"Job":                             {Group: "batch", Version: "v1", Resource: "jobs"},
	"CronJob":                         {Group: "batch", Version: "v1", Resource: "cronjobs"},
	"Ingress":                         {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
	"NetworkPolicy":                   {Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},
	"ClusterRole":                     {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"},
	"ClusterRoleBinding":              {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"},
	"Role":                            {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"},
	"RoleBinding":                     {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
	"HorizontalPodAutoscaler":         {Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
	"PodDisruptionBudget":             {Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"},
	"StorageClass":                    {Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"},
	"IngressClass":                    {Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses"},
	"ResourceQuota":                   {Group: "", Version: "v1", Resource: "resourcequotas"},
	"LimitRange":                      {Group: "", Version: "v1", Resource: "limitranges"},
	"PriorityClass":                   {Group: "scheduling.k8s.io", Version: "v1", Resource: "priorityclasses"},
	"RuntimeClass":                    {Group: "node.k8s.io", Version: "v1", Resource: "runtimeclasses"},
	"MutatingWebhookConfiguration":    {Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"},
	"ValidatingWebhookConfiguration":  {Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"},
}

// clusterScopedKinds 集群级别资源
var clusterScopedKinds = map[string]bool{
	"Namespace":                       true,
	"PersistentVolume":                true,
	"ClusterRole":                     true,
	"ClusterRoleBinding":              true,
	"StorageClass":                    true,
	"IngressClass":                    true,
	"PriorityClass":                   true,
	"RuntimeClass":                    true,
	"MutatingWebhookConfiguration":    true,
	"ValidatingWebhookConfiguration":  true,
}

// ==================== 响应结构体 ====================

// GenericResourceDetail 通用资源详情
type GenericResourceDetail struct {
	Kind              string                 `json:"kind"`
	Name              string                 `json:"name"`
	Namespace         string                 `json:"namespace"`
	CreationTimestamp string                 `json:"creationTimestamp"`
	Labels            map[string]string      `json:"labels"`
	Annotations       map[string]string      `json:"annotations"`
	Spec              map[string]interface{} `json:"spec,omitempty"`
	Status            map[string]interface{} `json:"status,omitempty"`
	Extra             map[string]interface{} `json:"extra,omitempty"`
	Events            []GenericEvent         `json:"events"`
	Endpoints         []EndpointInfo         `json:"endpoints,omitempty"`
	Pods              []PodBrief             `json:"pods,omitempty"`
	ReplicaSets       []ReplicaSetBrief      `json:"replicaSets,omitempty"`
}

// GenericEvent 事件
type GenericEvent struct {
	Type           string `json:"type"`
	Reason         string `json:"reason"`
	Message        string `json:"message"`
	Source         string `json:"source"`
	Count          int32  `json:"count"`
	FirstTimestamp string `json:"firstTimestamp"`
	LastTimestamp   string `json:"lastTimestamp"`
}

// EndpointInfo Service 端点信息
type EndpointInfo struct {
	TargetName string   `json:"targetName"`
	Addresses  []string `json:"addresses"`
}

// PodBrief Pod 简要信息
type PodBrief struct {
	Name      string `json:"name"`
	Node      string `json:"node"`
	Namespace string `json:"namespace"`
	Ready     string `json:"ready"`
	Restarts  int32  `json:"restarts"`
	Status    string `json:"status"`
	CPU       string `json:"cpu"`
	Memory    string `json:"memory"`
}

// ReplicaSetBrief ReplicaSet 简要信息
type ReplicaSetBrief struct {
	Name     string `json:"name"`
	Revision string `json:"revision"`
	Desired  int32  `json:"desired"`
	Ready    int32  `json:"ready"`
	Age      string `json:"age"`
}

// ==================== Handler ====================

// GetGenericResourceDetail 获取通用资源详情
func (h *ResourceHandler) GetGenericResourceDetail(c *gin.Context) {
	clusterIDStr := c.Query("clusterId")
	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的集群ID"})
		return
	}

	kind := c.Query("kind")
	namespace := c.Query("namespace")
	name := c.Query("name")

	if kind == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少 kind 或 name 参数"})
		return
	}

	gvr, ok := kindToGVR[kind]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": fmt.Sprintf("不支持的资源类型: %s", kind)})
		return
	}

	// 创建 dynamic client
	kubeConfigContent, err := h.clusterService.GetClusterConfig(c.Request.Context(), uint(clusterID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取集群配置失败: " + err.Error()})
		return
	}
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfigContent))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析集群配置失败: " + err.Error()})
		return
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建客户端失败: " + err.Error()})
		return
	}

	ctx := c.Request.Context()

	// 获取资源
	var obj *unstructured.Unstructured
	if clusterScopedKinds[kind] || namespace == "" {
		obj, err = dynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	} else {
		obj, err = dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取资源失败: " + err.Error()})
		return
	}

	// 提取 metadata
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	detail := GenericResourceDetail{
		Kind:              kind,
		Name:              obj.GetName(),
		Namespace:         obj.GetNamespace(),
		CreationTimestamp: obj.GetCreationTimestamp().Format("2006-01-02T15:04:05Z"),
		Labels:            labels,
		Annotations:       annotations,
		Events:            []GenericEvent{},
	}

	// 提取 spec / status
	if specRaw, found, _ := unstructured.NestedMap(obj.Object, "spec"); found {
		detail.Spec = specRaw
	}
	if statusRaw, found, _ := unstructured.NestedMap(obj.Object, "status"); found {
		detail.Status = statusRaw
	}

	// 提取非标准顶层字段 (data, type, rules, roleRef, subjects, secrets 等)
	standardKeys := map[string]bool{"apiVersion": true, "kind": true, "metadata": true, "spec": true, "status": true}
	extra := map[string]interface{}{}
	for k, v := range obj.Object {
		if !standardKeys[k] {
			extra[k] = v
		}
	}
	if len(extra) > 0 {
		detail.Extra = extra
	}

	// 获取 typed clientset (用于事件和关联资源)
	clientset, err := h.clusterService.GetCachedClientset(ctx, uint(clusterID))
	if err != nil {
		// clientset 获取失败也返回已有的数据
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": detail})
		return
	}

	// 获取事件
	eventNS := namespace
	if clusterScopedKinds[kind] {
		eventNS = ""
	}
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=%s", name, kind)
	events, err := clientset.CoreV1().Events(eventNS).List(ctx, metav1.ListOptions{FieldSelector: fieldSelector})
	if err == nil {
		for _, evt := range events.Items {
			source := evt.Source.Component
			if evt.Source.Host != "" {
				if source != "" {
					source += ", "
				}
				source += evt.Source.Host
			}
			detail.Events = append(detail.Events, GenericEvent{
				Type:           evt.Type,
				Reason:         evt.Reason,
				Message:        evt.Message,
				Source:         source,
				Count:          evt.Count,
				FirstTimestamp: evt.FirstTimestamp.Format("2006-01-02 15:04:05"),
				LastTimestamp:  evt.LastTimestamp.Format("2006-01-02 15:04:05"),
			})
		}
	}

	// Kind 特定的关联资源
	switch kind {
	case "Service":
		detail.Endpoints = fetchServiceEndpoints(ctx, clientset, namespace, name)
	case "Deployment":
		matchLabels := getMatchLabels(obj)
		if len(matchLabels) > 0 {
			metrics := fetchPodMetrics(ctx, clientset, namespace)
			detail.Pods = fetchPodsByLabels(ctx, clientset, namespace, matchLabels, metrics)
			detail.ReplicaSets = fetchReplicaSets(ctx, clientset, namespace, matchLabels)
		}
	case "StatefulSet", "DaemonSet", "ReplicaSet":
		matchLabels := getMatchLabels(obj)
		if len(matchLabels) > 0 {
			metrics := fetchPodMetrics(ctx, clientset, namespace)
			detail.Pods = fetchPodsByLabels(ctx, clientset, namespace, matchLabels, metrics)
		}
	case "Job":
		metrics := fetchPodMetrics(ctx, clientset, namespace)
		detail.Pods = fetchPodsByLabels(ctx, clientset, namespace, map[string]string{"job-name": name}, metrics)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": detail})
}

// ==================== Helper Functions ====================

func getMatchLabels(obj *unstructured.Unstructured) map[string]string {
	matchLabels, found, _ := unstructured.NestedStringMap(obj.Object, "spec", "selector", "matchLabels")
	if found {
		return matchLabels
	}
	return nil
}

func buildLabelSelector(labels map[string]string) string {
	var parts []string
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}

func fetchServiceEndpoints(ctx context.Context, clientset *kubernetes.Clientset, namespace, serviceName string) []EndpointInfo {
	ep, err := clientset.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return nil
	}
	var result []EndpointInfo
	for _, subset := range ep.Subsets {
		for _, addr := range subset.Addresses {
			targetName := ""
			if addr.TargetRef != nil {
				targetName = addr.TargetRef.Name
			}
			var addrs []string
			for _, port := range subset.Ports {
				addrs = append(addrs, fmt.Sprintf("%s:%d", addr.IP, port.Port))
			}
			result = append(result, EndpointInfo{
				TargetName: targetName,
				Addresses:  addrs,
			})
		}
	}
	return result
}

func fetchPodsByLabels(ctx context.Context, clientset *kubernetes.Clientset, namespace string, labels map[string]string, metrics map[string][2]string) []PodBrief {
	selector := buildLabelSelector(labels)
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil
	}
	var result []PodBrief
	for _, pod := range pods.Items {
		ready := 0
		total := len(pod.Spec.Containers)
		var restarts int32
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
			restarts += cs.RestartCount
		}

		cpu := "-"
		mem := "-"
		if m, ok := metrics[pod.Name]; ok {
			cpu = m[0]
			mem = m[1]
		}

		result = append(result, PodBrief{
			Name:      pod.Name,
			Node:      pod.Spec.NodeName,
			Namespace: pod.Namespace,
			Ready:     fmt.Sprintf("%d/%d", ready, total),
			Restarts:  restarts,
			Status:    string(pod.Status.Phase),
			CPU:       cpu,
			Memory:    mem,
		})
	}
	return result
}

func fetchReplicaSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string, matchLabels map[string]string) []ReplicaSetBrief {
	selector := buildLabelSelector(matchLabels)
	rsList, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil
	}
	var result []ReplicaSetBrief
	for _, rs := range rsList.Items {
		revision := rs.Annotations["deployment.kubernetes.io/revision"]
		desired := int32(0)
		if rs.Spec.Replicas != nil {
			desired = *rs.Spec.Replicas
		}
		result = append(result, ReplicaSetBrief{
			Name:     rs.Name,
			Revision: revision,
			Desired:  desired,
			Ready:    rs.Status.ReadyReplicas,
			Age:      formatAge(rs.CreationTimestamp.Time),
		})
	}
	return result
}

// fetchPodMetrics 尝试获取 Pod 指标 (metrics-server)，失败则返回空 map
func fetchPodMetrics(ctx context.Context, clientset *kubernetes.Clientset, namespace string) map[string][2]string {
	result := make(map[string][2]string)

	data, err := clientset.CoreV1().RESTClient().Get().
		AbsPath(fmt.Sprintf("/apis/metrics.k8s.io/v1beta1/namespaces/%s/pods", namespace)).
		DoRaw(ctx)
	if err != nil {
		return result
	}

	var metricsResp struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Containers []struct {
				Usage struct {
					CPU    string `json:"cpu"`
					Memory string `json:"memory"`
				} `json:"usage"`
			} `json:"containers"`
		} `json:"items"`
	}

	if err := json.Unmarshal(data, &metricsResp); err != nil {
		return result
	}

	for _, item := range metricsResp.Items {
		var totalCPUMilli int64
		var totalMemBytes int64
		for _, c := range item.Containers {
			if cpuQty, err := resource.ParseQuantity(c.Usage.CPU); err == nil {
				totalCPUMilli += cpuQty.MilliValue()
			}
			if memQty, err := resource.ParseQuantity(c.Usage.Memory); err == nil {
				totalMemBytes += memQty.Value()
			}
		}
		cpuStr := fmt.Sprintf("%.3f", float64(totalCPUMilli)/1000.0)
		memStr := formatMemoryBytes(totalMemBytes)
		result[item.Metadata.Name] = [2]string{cpuStr, memStr}
	}

	return result
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", mins)
}

func formatMemoryBytes(bytes int64) string {
	const (
		KiB = 1024
		MiB = KiB * 1024
		GiB = MiB * 1024
	)
	switch {
	case bytes >= GiB:
		return fmt.Sprintf("%.1fGiB", float64(bytes)/float64(GiB))
	case bytes >= MiB:
		return fmt.Sprintf("%.1fMiB", float64(bytes)/float64(MiB))
	case bytes >= KiB:
		return fmt.Sprintf("%.1fKiB", float64(bytes)/float64(KiB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
