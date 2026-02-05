package server

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	"github.com/ydcloud-dy/mom/plugins/kubernetes/service"
)

// CRDHandler Kubernetes CustomResourceDefinition 处理器
type CRDHandler struct {
	clusterService *service.ClusterService
	db             *gorm.DB
}

// NewCRDHandler 创建 CRD 处理器
func NewCRDHandler(clusterService *service.ClusterService, db *gorm.DB) *CRDHandler {
	return &CRDHandler{
		clusterService: clusterService,
		db:             db,
	}
}

// CRDInfo CRD 列表信息
type CRDInfo struct {
	Name              string      `json:"name"`
	Group             string      `json:"group"`
	Version           string      `json:"version"`
	Kind              string      `json:"kind"`
	Scope             string      `json:"scope"`
	CreationTimestamp metav1.Time `json:"creationTimestamp"`
}

// createDynamicClient 创建 Dynamic Client
func (h *CRDHandler) createDynamicClient(ctx context.Context, clusterID uint) (dynamic.Interface, error) {
	kubeConfigContent, err := h.clusterService.GetClusterConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfigContent))
	if err != nil {
		return nil, err
	}

	// 增加 QPS 和 Burst
	config.QPS = 100
	config.Burst = 200

	return dynamic.NewForConfig(config)
}

// removeManagedFields 移除 managedFields
func removeManagedFields(obj map[string]interface{}) {
	metadata, ok := obj["metadata"].(map[string]interface{})
	if ok {
		delete(metadata, "managedFields")
	}
}

// Helper to parse cluster ID
func (h *CRDHandler) parseClusterID(c *gin.Context) (uint, error) {
	// 尝试从 Param 获取
	idStr := c.Param("id")
	// 如果 Param 没有，尝试从 Query 获取 (比如 ListCustomResources 路由可能是 /custom?clusterId=... 或者在 header)
	// 根据 router.go，路由都在 /clusters 下或者 /resources 下
	// router.go 中的 listCRDs 是 /resources/crds，这里没有 :id 参数！
	// resource_handler.go 是如何获取 cluster id 的？
	// 它是通过 header "X-Cluster-ID" 或者 query param?
	// 检查 router.go，resourceHandler 的路由也没有 :id。
	// 大概率是通过 header 或者 query.
	// 让我们检查一下 resource_handler.go (之前 view 过，但没仔细看 helper).
	// 假设我们在 Header 中传递 ClusterID，或者 Query Parameter "clusterId".

	if idStr == "" {
		idStr = c.Query("clusterId")
	}
	if idStr == "" {
		idStr = c.GetHeader("X-Cluster-ID")
	}

	if idStr == "" {
		return 0, fmt.Errorf("missing cluster id")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid cluster id")
	}
	return uint(id), nil
}

// ListCRDs 获取 CRD 列表
func (h *CRDHandler) ListCRDs(c *gin.Context) {
	clusterID, err := h.parseClusterID(c)
	if err != nil {
		fmt.Printf("ListCRDs ParseClusterID Error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	fmt.Printf("ListCRDs ClusterID: %d\n", clusterID)

	ctx := c.Request.Context()
	dynamicClient, err := h.createDynamicClient(ctx, clusterID)
	if err != nil {
		fmt.Printf("ListCRDs CreateDynamicClient Error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	gvr := schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	crds, err := dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Printf("ListCRDs List Error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	fmt.Printf("ListCRDs Fetched Count: %d\n", len(crds.Items))

	var crdInfos []CRDInfo
	for _, item := range crds.Items {
		name, _, _ := unstructured.NestedString(item.Object, "metadata", "name")
		creationTimestampStr, _, _ := unstructured.NestedString(item.Object, "metadata", "creationTimestamp")
		creationTimestamp, _ := time.Parse(time.RFC3339, creationTimestampStr)

		group, _, _ := unstructured.NestedString(item.Object, "spec", "group")

		// versions
		versions, _, _ := unstructured.NestedSlice(item.Object, "spec", "versions")
		var version string
		// 通常我们取 stored=true 的 version
		for _, v := range versions {
			if vMap, ok := v.(map[string]interface{}); ok {
				if stored, ok := vMap["stored"].(bool); ok && stored {
					version = vMap["name"].(string)
					break
				}
			}
		}
		if version == "" && len(versions) > 0 {
			if vMap, ok := versions[0].(map[string]interface{}); ok {
				version = vMap["name"].(string)
			}
		}

		scope, _, _ := unstructured.NestedString(item.Object, "spec", "scope")
		kind, _, _ := unstructured.NestedString(item.Object, "spec", "names", "kind")

		crdInfos = append(crdInfos, CRDInfo{
			Name:              name,
			Group:             group,
			Version:           version,
			Kind:              kind,
			Scope:             scope,
			CreationTimestamp: metav1.Time{Time: creationTimestamp},
		})
	}

	sort.Slice(crdInfos, func(i, j int) bool {
		return crdInfos[i].Name < crdInfos[j].Name
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": crdInfos})
}

// GetCRD 获取单个 CRD 详情 (YAML)
func (h *CRDHandler) GetCRD(c *gin.Context) {
	clusterID, err := h.parseClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	crdName := c.Param("name")

	ctx := c.Request.Context()
	dynamicClient, err := h.createDynamicClient(ctx, clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	gvr := schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	crd, err := dynamicClient.Resource(gvr).Get(ctx, crdName, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	removeManagedFields(crd.Object)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": crd})
}

// DeleteCRD 删除 CRD
func (h *CRDHandler) DeleteCRD(c *gin.Context) {
	clusterID, err := h.parseClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	crdName := c.Param("name")

	ctx := c.Request.Context()
	dynamicClient, err := h.createDynamicClient(ctx, clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	gvr := schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	err = dynamicClient.Resource(gvr).Delete(ctx, crdName, metav1.DeleteOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ListCustomResources 获取指定 CRD 的资源列表
func (h *CRDHandler) ListCustomResources(c *gin.Context) {
	clusterID, err := h.parseClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	group := c.Query("group")
	version := c.Query("version")
	resource := c.Query("resource") // plural name
	namespace := c.Query("namespace")

	if group == "" || version == "" || resource == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "missing group, version or resource parameters"})
		return
	}

	ctx := c.Request.Context()
	dynamicClient, err := h.createDynamicClient(ctx, clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}

	var list *unstructured.UnstructuredList
	if namespace != "" {
		list, err = dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list.Items})
}

// GetCustomResource 获取单个 CR 详情
func (h *CRDHandler) GetCustomResource(c *gin.Context) {
	clusterID, err := h.parseClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	group := c.Query("group")
	version := c.Query("version")
	resource := c.Query("resource")

	namespace := c.Param("namespace")
	name := c.Param("name")

	ctx := c.Request.Context()
	dynamicClient, err := h.createDynamicClient(ctx, clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}

	var item *unstructured.Unstructured

	if namespace == "-" || namespace == "cluster" || namespace == "" || namespace == "undefined" {
		item, err = dynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	} else {
		item, err = dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	removeManagedFields(item.Object)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": item})
}

// DeleteCustomResource 删除 CR
func (h *CRDHandler) DeleteCustomResource(c *gin.Context) {
	clusterID, err := h.parseClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	group := c.Query("group")
	version := c.Query("version")
	resource := c.Query("resource")

	namespace := c.Param("namespace")
	name := c.Param("name")

	ctx := c.Request.Context()
	dynamicClient, err := h.createDynamicClient(ctx, clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}

	if namespace == "-" || namespace == "cluster" || namespace == "" || namespace == "undefined" {
		err = dynamicClient.Resource(gvr).Delete(ctx, name, metav1.DeleteOptions{})
	} else {
		err = dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// CreateCustomResourceFromYAML YAML创建 CR
func (h *CRDHandler) CreateCustomResourceFromYAML(c *gin.Context) {
	clusterID, err := h.parseClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	group := c.Query("group")
	version := c.Query("version")
	resource := c.Query("resource")

	if group == "" || version == "" || resource == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "missing group, version or resource parameters"})
		return
	}

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var objContent map[string]interface{}
	if err := yaml.Unmarshal([]byte(req.YAML), &objContent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": fmt.Sprintf("invalid yaml: %v", err)})
		return
	}

	obj := &unstructured.Unstructured{Object: objContent}

	namespace := obj.GetNamespace()

	ctx := c.Request.Context()
	dynamicClient, err := h.createDynamicClient(ctx, clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}

	var created *unstructured.Unstructured
	if namespace != "" {
		created, err = dynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
	} else {
		created, err = dynamicClient.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": created})
}

// UpdateCustomResourceFromYAML YAML更新 CR
func (h *CRDHandler) UpdateCustomResourceFromYAML(c *gin.Context) {
	clusterID, err := h.parseClusterID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	group := c.Query("group")
	version := c.Query("version")
	resource := c.Query("resource")

	if group == "" || version == "" || resource == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "missing group, version or resource parameters"})
		return
	}

	var req struct {
		YAML string `json:"yaml"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var objContent map[string]interface{}
	if err := yaml.Unmarshal([]byte(req.YAML), &objContent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": fmt.Sprintf("invalid yaml: %v", err)})
		return
	}

	obj := &unstructured.Unstructured{Object: objContent}
	name := obj.GetName()
	namespace := obj.GetNamespace()

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "missing metadata.name in yaml"})
		return
	}

	ctx := c.Request.Context()
	dynamicClient, err := h.createDynamicClient(ctx, clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}

	var current *unstructured.Unstructured
	if namespace != "" {
		current, err = dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		current, err = dynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	obj.SetResourceVersion(current.GetResourceVersion())

	var updated *unstructured.Unstructured
	if namespace != "" {
		updated, err = dynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{})
	} else {
		updated, err = dynamicClient.Resource(gvr).Update(ctx, obj, metav1.UpdateOptions{})
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": updated})
}
