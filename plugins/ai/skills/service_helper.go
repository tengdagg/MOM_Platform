package skills

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	sshclient "github.com/ydcloud-dy/mom/pkg/ssh"
)

// 加密密钥
var credentialEncryptionKey = []byte("mom-encrypt-key-32bytes-long!!@@") // 凭证仓库
var k8sEncryptionKey = []byte("mom-k8s-encrypt-key-32byte!!@@!!")      // K8s kubeconfig

// decryptWithKey 使用指定密钥解密 AES-GCM 加密的字符串
func decryptWithKey(ciphertext string, key []byte) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("加密数据格式错误")
	}
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// decryptCredential 解密凭证字段
func decryptCredential(ciphertext string) (string, error) {
	return decryptWithKey(ciphertext, credentialEncryptionKey)
}

// decryptKubeConfig 解密 K8s kubeconfig
func decryptKubeConfig(ciphertext string) (string, error) {
	return decryptWithKey(ciphertext, k8sEncryptionKey)
}

// ---------- K8s Helper ----------

// K8sCluster 集群基本信息
type K8sCluster struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	KubeConfig string `json:"-"`
}

// GetK8sClientset 根据集群 ID 从数据库获取 kubeconfig 并创建 clientset
func GetK8sClientset(db *gorm.DB, clusterID uint) (*kubernetes.Clientset, *K8sCluster, error) {
	var cluster K8sCluster
	if err := db.Table("k8s_clusters").Select("id, name, kube_config").
		Where("id = ?", clusterID).First(&cluster).Error; err != nil {
		return nil, nil, fmt.Errorf("集群 ID=%d 不存在", clusterID)
	}

	// 解密 kubeconfig（使用 K8s 专用密钥）
	decryptedConfig, err := decryptKubeConfig(cluster.KubeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("解密集群配置失败: %v", err)
	}

	// 创建 rest.Config
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(decryptedConfig))
	if err != nil {
		return nil, nil, fmt.Errorf("解析 KubeConfig 失败: %v", err)
	}
	restConfig.Timeout = 15 * time.Second

	// 创建 clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("创建 K8s 客户端失败: %v", err)
	}

	return clientset, &cluster, nil
}

// FindClusterID 根据参数查找集群 ID
func FindClusterID(db *gorm.DB, params map[string]any) (uint, string, error) {
	if clusterID, ok := params["cluster_id"].(float64); ok && clusterID > 0 {
		var name string
		db.Table("k8s_clusters").Where("id = ?", uint(clusterID)).Pluck("name", &name)
		return uint(clusterID), name, nil
	}
	clusterName, _ := params["cluster_name"].(string)
	if clusterName == "" {
		var clusters []struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		}
		db.Table("k8s_clusters").Select("id, name").Find(&clusters)
		if len(clusters) == 1 {
			return clusters[0].ID, clusters[0].Name, nil
		}
		return 0, "", fmt.Errorf("请指定集群名称或 ID（当前有 %d 个集群）", len(clusters))
	}
	var cluster struct {
		ID   uint
		Name string
	}
	if err := db.Table("k8s_clusters").Where("name LIKE ? OR alias LIKE ?", "%"+clusterName+"%", "%"+clusterName+"%").First(&cluster).Error; err != nil {
		return 0, "", fmt.Errorf("集群 %s 不存在", clusterName)
	}
	return cluster.ID, cluster.Name, nil
}

// ScaleWorkload 扩缩容工作负载
func ScaleWorkload(clientset *kubernetes.Clientset, namespace, resourceType, name string, replicas int32) error {
	ctx := context.Background()
	switch strings.ToLower(resourceType) {
	case "deployment":
		scale, err := clientset.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("获取 Deployment %s 失败: %v", name, err)
		}
		scale.Spec.Replicas = replicas
		_, err = clientset.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
		return err
	case "statefulset":
		scale, err := clientset.AppsV1().StatefulSets(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("获取 StatefulSet %s 失败: %v", name, err)
		}
		scale.Spec.Replicas = replicas
		_, err = clientset.AppsV1().StatefulSets(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
		return err
	default:
		return fmt.Errorf("不支持的资源类型: %s，支持 Deployment/StatefulSet", resourceType)
	}
}

// RestartWorkload 滚动重启工作负载
func RestartWorkload(clientset *kubernetes.Clientset, namespace, resourceType, name string) error {
	ctx := context.Background()
	restartAnnotation := map[string]string{
		"kubectl.kubernetes.io/restartedAt": time.Now().Format(time.RFC3339),
	}

	switch strings.ToLower(resourceType) {
	case "deployment":
		deploy, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("获取 Deployment %s 失败: %v", name, err)
		}
		if deploy.Spec.Template.Annotations == nil {
			deploy.Spec.Template.Annotations = make(map[string]string)
		}
		for k, v := range restartAnnotation {
			deploy.Spec.Template.Annotations[k] = v
		}
		_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deploy, metav1.UpdateOptions{})
		return err
	case "statefulset":
		sts, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("获取 StatefulSet %s 失败: %v", name, err)
		}
		if sts.Spec.Template.Annotations == nil {
			sts.Spec.Template.Annotations = make(map[string]string)
		}
		for k, v := range restartAnnotation {
			sts.Spec.Template.Annotations[k] = v
		}
		_, err = clientset.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
		return err
	case "daemonset":
		ds, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("获取 DaemonSet %s 失败: %v", name, err)
		}
		if ds.Spec.Template.Annotations == nil {
			ds.Spec.Template.Annotations = make(map[string]string)
		}
		for k, v := range restartAnnotation {
			ds.Spec.Template.Annotations[k] = v
		}
		_, err = clientset.AppsV1().DaemonSets(namespace).Update(ctx, ds, metav1.UpdateOptions{})
		return err
	default:
		return fmt.Errorf("不支持重启的资源类型: %s", resourceType)
	}
}

// CordonNode 设置节点不可调度 / 恢复调度
func CordonNode(clientset *kubernetes.Clientset, nodeName string, unschedulable bool) error {
	ctx := context.Background()
	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("获取节点 %s 失败: %v", nodeName, err)
	}
	node.Spec.Unschedulable = unschedulable
	_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}

// DrainNode 排空节点
func DrainNode(clientset *kubernetes.Clientset, nodeName string) (int, error) {
	ctx := context.Background()
	// 先设为不可调度
	if err := CordonNode(clientset, nodeName, true); err != nil {
		return 0, err
	}
	// 列出节点上的所有 Pod
	podList, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		return 0, fmt.Errorf("列出节点 Pod 失败: %v", err)
	}
	evicted := 0
	for _, pod := range podList.Items {
		// 跳过 DaemonSet 管理的 Pod
		if isDaemonSetPod(&pod) {
			continue
		}
		// 驱逐 Pod
		err := clientset.CoreV1().Pods(pod.Namespace).EvictV1(ctx, &policyv1.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: pod.Namespace,
			},
		})
		if err != nil {
			continue // 跳过无法驱逐的 Pod
		}
		evicted++
	}
	return evicted, nil
}

func isDaemonSetPod(pod *v1.Pod) bool {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

// GetPodLogs 获取 Pod 日志
func GetPodLogs(clientset *kubernetes.Clientset, namespace, podName, container string, tailLines int64, previous bool) (string, error) {
	ctx := context.Background()
	opts := &v1.PodLogOptions{
		Timestamps: true,
		Previous:   previous,
	}
	if container != "" {
		opts.Container = container
	}
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
	logStream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("获取日志流失败: %v", err)
	}
	defer logStream.Close()

	logBytes, err := io.ReadAll(io.LimitReader(logStream, 64*1024)) // 最多 64KB
	if err != nil {
		return "", fmt.Errorf("读取日志失败: %v", err)
	}
	return string(logBytes), nil
}

// DiagnosePod 诊断 Pod 问题
func DiagnosePod(clientset *kubernetes.Clientset, namespace, podName string) (map[string]any, error) {
	ctx := context.Background()

	// 获取 Pod 详情
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Pod %s 失败: %v", podName, err)
	}

	// 获取 Pod 事件
	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + podName,
	})
	var eventList []map[string]string
	if err == nil {
		for _, e := range events.Items {
			eventList = append(eventList, map[string]string{
				"type":    e.Type,
				"reason":  e.Reason,
				"message": e.Message,
				"time":    e.LastTimestamp.Format("2006-01-02 15:04:05"),
			})
		}
	}

	// 容器状态
	var containerStatuses []map[string]any
	for _, cs := range pod.Status.ContainerStatuses {
		status := map[string]any{
			"name":         cs.Name,
			"ready":        cs.Ready,
			"restartCount": cs.RestartCount,
			"image":        cs.Image,
		}
		if cs.State.Waiting != nil {
			status["state"] = "Waiting"
			status["reason"] = cs.State.Waiting.Reason
			status["message"] = cs.State.Waiting.Message
		} else if cs.State.Running != nil {
			status["state"] = "Running"
			status["startedAt"] = cs.State.Running.StartedAt.Format("2006-01-02 15:04:05")
		} else if cs.State.Terminated != nil {
			status["state"] = "Terminated"
			status["reason"] = cs.State.Terminated.Reason
			status["exitCode"] = cs.State.Terminated.ExitCode
		}
		containerStatuses = append(containerStatuses, status)
	}

	return map[string]any{
		"podName":    podName,
		"namespace":  namespace,
		"phase":      string(pod.Status.Phase),
		"nodeName":   pod.Spec.NodeName,
		"hostIP":     pod.Status.HostIP,
		"podIP":      pod.Status.PodIP,
		"conditions": pod.Status.Conditions,
		"containers": containerStatuses,
		"events":     eventList,
	}, nil
}

// ---------- SSH Helper ----------

// SSHHost 主机 + 凭证信息
type SSHHost struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	IP           string `json:"ip"`
	Port         int    `json:"port"`
	SSHUser      string `json:"sshUser"`
	CredentialID uint   `json:"-"`
}

// CreateSSHClient 根据主机 ID 创建 SSH 客户端
func CreateSSHClient(db *gorm.DB, hostID uint) (*sshclient.Client, *SSHHost, error) {
	var host SSHHost
	if err := db.Table("hosts").Select("id, name, ip, port, ssh_user, credential_id").
		Where("id = ? AND deleted_at IS NULL", hostID).First(&host).Error; err != nil {
		return nil, nil, fmt.Errorf("主机 ID=%d 不存在", hostID)
	}

	return createSSHClientFromHost(db, &host)
}

// CreateSSHClientByIP 根据主机 IP 创建 SSH 客户端
func CreateSSHClientByIP(db *gorm.DB, ip string) (*sshclient.Client, *SSHHost, error) {
	var host SSHHost
	if err := db.Table("hosts").Select("id, name, ip, port, ssh_user, credential_id").
		Where("ip = ? AND deleted_at IS NULL", ip).First(&host).Error; err != nil {
		return nil, nil, fmt.Errorf("IP=%s 的主机不存在", ip)
	}

	return createSSHClientFromHost(db, &host)
}

func createSSHClientFromHost(db *gorm.DB, host *SSHHost) (*sshclient.Client, *SSHHost, error) {
	if host.CredentialID == 0 {
		return nil, nil, fmt.Errorf("主机 %s(%s) 未配置凭证", host.Name, host.IP)
	}

	// 获取凭证
	var cred struct {
		Password   string
		PrivateKey string
		Passphrase string
	}
	if err := db.Table("credentials").Select("password, private_key, passphrase").
		Where("id = ? AND deleted_at IS NULL", host.CredentialID).First(&cred).Error; err != nil {
		return nil, nil, fmt.Errorf("凭证不存在")
	}

	// 解密凭证（使用凭证专用密钥）
	password, _ := decryptCredential(cred.Password)
	privateKey, _ := decryptCredential(cred.PrivateKey)
	passphrase, _ := decryptCredential(cred.Passphrase)

	if host.Port == 0 {
		host.Port = 22
	}
	if host.SSHUser == "" {
		host.SSHUser = "root"
	}

	client, err := sshclient.NewClient(host.IP, host.Port, host.SSHUser, password, []byte(privateKey), passphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("SSH 连接 %s(%s) 失败: %v", host.Name, host.IP, err)
	}

	return client, host, nil
}
