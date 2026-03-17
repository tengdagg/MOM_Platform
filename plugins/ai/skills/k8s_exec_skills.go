package skills

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func detectUnsupportedK8sExecCommand(command string) string {
	cmd := strings.TrimSpace(strings.ToLower(command))
	if cmd == "" {
		return ""
	}

	exactInteractive := map[string]string{
		"top":    "交互式终端程序",
		"htop":   "交互式终端程序",
		"less":   "交互式分页程序",
		"more":   "交互式分页程序",
		"vi":     "交互式编辑器",
		"vim":    "交互式编辑器",
		"nano":   "交互式编辑器",
		"screen": "终端复用会话",
		"tmux":   "终端复用会话",
	}
	if reason, ok := exactInteractive[cmd]; ok {
		return reason
	}

	prefixReasons := []struct {
		prefix string
		reason string
	}{
		{"vi ", "交互式编辑器"},
		{"vim ", "交互式编辑器"},
		{"nano ", "交互式编辑器"},
		{"less ", "交互式分页程序"},
		{"more ", "交互式分页程序"},
		{"top ", "交互式终端程序"},
		{"htop ", "交互式终端程序"},
		{"watch ", "持续刷新型交互命令"},
		{"tail -f", "持续跟随型命令"},
		{"journalctl -f", "持续跟随型命令"},
		{"docker logs -f", "持续跟随型命令"},
		{"docker attach", "交互式容器附着"},
		{"ssh ", "嵌套远程登录"},
		{"sftp ", "交互式文件会话"},
		{"ftp ", "交互式文件会话"},
		{"telnet ", "交互式远程会话"},
		{"mysql ", "交互式数据库会话"},
		{"psql ", "交互式数据库会话"},
		{"redis-cli", "交互式数据库会话"},
		{"kubectl exec", "嵌套容器终端"},
	}
	for _, item := range prefixReasons {
		if strings.HasPrefix(cmd, item.prefix) {
			return item.reason
		}
	}
	return ""
}

func requiresK8sShellSession(command string) bool {
	cmd := strings.TrimSpace(strings.ToLower(command))
	if cmd == "" {
		return false
	}

	exactSessionCommands := map[string]bool{
		"bash": true,
		"sh":   true,
		"ash":  true,
		"dash": true,
		"zsh":  true,
		"ksh":  true,
	}
	if exactSessionCommands[cmd] {
		return true
	}

	prefixes := []string{
		"cd ",
		"export ",
		"unset ",
		"alias ",
		"unalias ",
		"source ",
		". ",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(cmd, prefix) {
			return true
		}
	}
	return false
}

func isSafeK8sReadCommand(command string) bool {
	cmd := strings.TrimSpace(strings.ToLower(command))
	if cmd == "" {
		return false
	}

	safePrefixes := []string{
		"ls", "pwd", "id", "whoami", "ps ", "cat ", "grep ", "egrep ", "rg ",
		"head ", "tail ", "env", "printenv", "df", "du", "free", "uname", "uptime",
		"mount", "findmnt", "ss ", "netstat ", "ip addr show", "ip route show",
		"hostname", "date", "echo ", "which ", "type ", "nslookup ", "dig ",
	}
	for _, prefix := range safePrefixes {
		if cmd == prefix || strings.HasPrefix(cmd, prefix+" ") || strings.HasPrefix(cmd, prefix+"\t") {
			return true
		}
		if strings.HasPrefix(cmd, prefix) && prefix == "ls" {
			return true
		}
		if strings.HasPrefix(cmd, prefix) && (prefix == "df" || prefix == "du") {
			return true
		}
	}
	return false
}

func isDangerousK8sExecCommand(command string) bool {
	cmd := strings.TrimSpace(strings.ToLower(command))
	if cmd == "" {
		return false
	}
	dangerousPatterns := []string{
		"rm ", "rm\t", "rmdir ", "chmod ", "chown ", "chgrp ",
		"kill ", "kill\t", "killall ", "pkill ",
		"reboot", "shutdown", "poweroff", "halt",
		"apt install", "apt remove", "apt purge", "apt upgrade",
		"apt-get install", "apt-get remove", "apt-get purge", "apt-get upgrade",
		"apk add", "apk del",
		"yum install", "yum remove", "yum update", "yum upgrade",
		"dnf install", "dnf remove", "dnf update", "dnf upgrade",
		"pip install", "pip uninstall", "pip3 install", "pip3 uninstall",
		"npm install", "npm uninstall", "npm update",
		"sed -i", "tee ", "tee\t", "kubectl apply", "kubectl delete",
		"supervisorctl restart", "supervisorctl stop", "supervisorctl start",
	}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmd, pattern) {
			return true
		}
	}
	if strings.Contains(cmd, " > ") && !strings.Contains(cmd, " 2>/dev/null") && !strings.Contains(cmd, " > /dev/null") {
		return true
	}
	return false
}

func executeK8sExecCommand(ctx biz.SkillContext) (any, error) {
	rawCommand, _ := ctx.Params["command"].(string)
	command, err := normalizeExecCommand(rawCommand, "请指定要在 Pod 内执行的命令，空白命令不会执行")
	if err != nil {
		return nil, err
	}
	if reason := detectUnsupportedK8sExecCommand(command); reason != "" {
		return nil, fmt.Errorf("当前命令不适合通过 AI Pod 命令执行：%s。请改为单条可完成的容器命令，或使用平台 Pod 终端进行持续交互。", reason)
	}

	timeoutSec := 60
	if t, ok := ctx.Params["timeout"].(float64); ok && t > 0 {
		timeoutSec = int(t)
		if timeoutSec > 600 {
			timeoutSec = 600
		}
	}

	restConfig, target, err := resolveK8sExecTarget(ctx)
	if err != nil {
		return nil, err
	}

	safeDirect := !isDangerousK8sExecCommand(command) && (isSafeK8sReadCommand(command) || requiresK8sShellSession(command))
	hasActiveSession := aiK8sShellSessions.HasActiveSession(ctx.SessionID, target)
	if !safeDirect && !isConfirmed(ctx.Params) {
		return map[string]any{
			"status":      "pending_confirmation",
			"warning":     "⚠️ Pod 内命令执行可能修改容器文件、进程或运行状态，请确认命令内容无误后再执行",
			"message":     fmt.Sprintf("命令 [%s] 将在 Pod %s/%s（容器 %s）中执行", command, target.Namespace, target.PodName, target.Container),
			"command":     command,
			"cluster":     target.ClusterName,
			"clusterID":   target.ClusterID,
			"namespace":   target.Namespace,
			"podName":     target.PodName,
			"container":   target.Container,
			"sessionMode": map[string]any{"required": requiresK8sShellSession(command), "activeSessionExists": hasActiveSession},
		}, nil
	}

	type ExecResult struct {
		Cluster       string `json:"cluster"`
		Namespace     string `json:"namespace"`
		PodName       string `json:"podName"`
		Container     string `json:"container"`
		Output        string `json:"output"`
		Error         string `json:"error,omitempty"`
		Truncated     bool   `json:"truncated,omitempty"`
		SessionMode   string `json:"sessionMode,omitempty"`
		SessionReused bool   `json:"sessionReused,omitempty"`
		ExecutionMode string `json:"executionMode,omitempty"`
	}

	result := ExecResult{
		Cluster:   target.ClusterName,
		Namespace: target.Namespace,
		PodName:   target.PodName,
		Container: target.Container,
	}

	if requiresK8sShellSession(command) || hasActiveSession {
		output, reused, execErr := aiK8sShellSessions.ExecuteCommand(ctx.SessionID, restConfig, target, command)
		truncated, finalOutput := truncateK8sExecOutput(output)
		result.Output = finalOutput
		result.Truncated = truncated
		result.SessionMode = "interactive"
		result.SessionReused = reused
		result.ExecutionMode = buildExecutionModeLabel("interactive", reused)
		if execErr != nil {
			result.Error = normalizeK8sExecError(execErr, true).Error()
		}
	} else {
		output, execErr := executeK8sCommandOnce(restConfig, target, command, time.Duration(timeoutSec)*time.Second)
		truncated, finalOutput := truncateK8sExecOutput(output)
		result.Output = finalOutput
		result.Truncated = truncated
		result.SessionMode = "oneshot"
		result.ExecutionMode = buildExecutionModeLabel("oneshot", false)
		if execErr != nil {
			result.Error = normalizeK8sExecError(execErr, false).Error()
		}
	}

	successCount := 1
	message := fmt.Sprintf("✅ 命令已在 Pod %s/%s（容器 %s）上执行完成", target.Namespace, target.PodName, target.Container)
	if result.Error != "" {
		successCount = 0
		message = fmt.Sprintf("命令在 Pod %s/%s（容器 %s）上执行失败", target.Namespace, target.PodName, target.Container)
	}
	response := map[string]any{
		"status":                    "success",
		"message":                   message,
		"command":                   command,
		"cluster":                   target.ClusterName,
		"clusterID":                 target.ClusterID,
		"namespace":                 target.Namespace,
		"podName":                   target.PodName,
		"container":                 target.Container,
		"results":                   []ExecResult{result},
		"sessionMode":               result.SessionMode,
		"sessionReused":             result.SessionReused,
		"executionMode":             result.ExecutionMode,
		"successCount":              successCount,
		"totalCount":                1,
		"sessionIdleTimeoutSeconds": int(aiK8sSessionIdleTimeout.Seconds()),
	}
	if safeDirect {
		response["effectiveRiskLevel"] = "low"
		if result.Error == "" {
			response["message"] = fmt.Sprintf("✅ 命令已在 Pod %s/%s（容器 %s）上执行完成（安全命令，已跳过确认）", target.Namespace, target.PodName, target.Container)
		}
	} else if isDangerousK8sExecCommand(command) {
		response["effectiveRiskLevel"] = "critical"
	}
	return response, nil
}

func executeK8sSessionStatus(ctx biz.SkillContext) (any, error) {
	if ctx.SessionID == 0 {
		return nil, fmt.Errorf("缺少 AI 会话上下文，无法查询 Pod 会话状态")
	}
	filter, err := buildK8sSessionFilter(ctx)
	if err != nil {
		return nil, err
	}
	sessions := aiK8sShellSessions.ListSessions(ctx.SessionID, filter)
	return map[string]any{
		"sessions":                  sessions,
		"activeCount":               len(sessions),
		"sessionIdleTimeoutSeconds": int(aiK8sSessionIdleTimeout.Seconds()),
		"effectiveRiskLevel":        "low",
		"message":                   fmt.Sprintf("当前对话中有 %d 个 Pod 交互会话处于活动状态", len(sessions)),
	}, nil
}

func executeK8sCloseSession(ctx biz.SkillContext) (any, error) {
	if ctx.SessionID == 0 {
		return nil, fmt.Errorf("缺少 AI 会话上下文，无法关闭 Pod 会话")
	}
	closeAll, _ := ctx.Params["all"].(bool)
	filter := aiK8sSessionFilter{}
	var err error
	if !closeAll {
		filter, err = buildK8sSessionFilter(ctx)
		if err != nil {
			return nil, err
		}
		if filter.ClusterID == 0 && strings.TrimSpace(filter.Namespace) == "" && strings.TrimSpace(filter.PodName) == "" && strings.TrimSpace(filter.Container) == "" {
			return nil, fmt.Errorf("请指定 cluster_id/cluster_name、namespace、pod_name 或 container，或传 all=true 关闭全部 Pod 会话")
		}
	}

	closedCount := aiK8sShellSessions.CloseSessions(ctx.SessionID, filter)
	targetText := "当前对话中的所有 Pod 会话"
	if !closeAll {
		targetText = "符合筛选条件的 Pod 交互会话"
	}
	return map[string]any{
		"closedCount":               closedCount,
		"effectiveRiskLevel":        "low",
		"message":                   fmt.Sprintf("已关闭 %s，共 %d 条。", targetText, closedCount),
		"sessionIdleTimeoutSeconds": int(aiK8sSessionIdleTimeout.Seconds()),
	}, nil
}

func resolveK8sExecTarget(ctx biz.SkillContext) (*rest.Config, aiK8sExecTarget, error) {
	clusterID, clusterName, err := FindClusterID(ctx.DB, ctx.Params)
	if err != nil {
		return nil, aiK8sExecTarget{}, err
	}
	namespace, _ := ctx.Params["namespace"].(string)
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		namespace = "default"
	}
	podName, _ := ctx.Params["pod_name"].(string)
	podName = strings.TrimSpace(podName)
	if podName == "" {
		return nil, aiK8sExecTarget{}, fmt.Errorf("请指定 pod_name")
	}
	container, _ := ctx.Params["container"].(string)
	container = strings.TrimSpace(container)

	restConfig, _, err := GetK8sRESTConfig(ctx.DB, clusterID)
	if err != nil {
		return nil, aiK8sExecTarget{}, err
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, aiK8sExecTarget{}, fmt.Errorf("创建 K8s 客户端失败: %v", err)
	}
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, aiK8sExecTarget{}, fmt.Errorf("获取 Pod 失败: %v", err)
	}
	if pod.Status.Phase != v1.PodRunning {
		return nil, aiK8sExecTarget{}, fmt.Errorf("Pod %s/%s 当前状态为 %s，暂不支持执行命令", namespace, podName, pod.Status.Phase)
	}

	containerNames := make([]string, 0, len(pod.Spec.Containers))
	for _, c := range pod.Spec.Containers {
		containerNames = append(containerNames, c.Name)
	}
	if container == "" {
		if len(containerNames) == 1 {
			container = containerNames[0]
		} else {
			return nil, aiK8sExecTarget{}, fmt.Errorf("Pod %s/%s 包含多个容器，请指定 container。可选值: %s", namespace, podName, strings.Join(containerNames, ", "))
		}
	}
	if !containsString(containerNames, container) {
		return nil, aiK8sExecTarget{}, fmt.Errorf("Pod %s/%s 中不存在容器 %s。可选值: %s", namespace, podName, container, strings.Join(containerNames, ", "))
	}

	return restConfig, aiK8sExecTarget{
		ClusterID:   clusterID,
		ClusterName: clusterName,
		Namespace:   namespace,
		PodName:     podName,
		Container:   container,
	}, nil
}

func executeK8sCommandOnce(restConfig *rest.Config, target aiK8sExecTarget, command string, timeout time.Duration) (string, error) {
	execURL, err := buildK8sExecURL(restConfig, target, false, []string{"/bin/sh", "-c", command})
	if err != nil {
		return "", err
	}
	executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execURL)
	if err != nil {
		return "", fmt.Errorf("创建 Pod exec executor 失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	output := strings.TrimSpace(strings.TrimSpace(stdout.String()) + "\n" + strings.TrimSpace(stderr.String()))
	output = strings.TrimSpace(output)
	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("Pod 命令执行超时")
	}
	if err != nil {
		return output, normalizeK8sExecError(err, false)
	}
	return output, nil
}

func truncateK8sExecOutput(output string) (bool, string) {
	if len(output) <= 65536 {
		return false, output
	}
	return true, output[:65536] + "\n... [输出已截断，超过 64KB]"
}

func buildK8sSessionFilter(ctx biz.SkillContext) (aiK8sSessionFilter, error) {
	clusterID, err := resolveOptionalK8sClusterID(ctx)
	if err != nil {
		return aiK8sSessionFilter{}, err
	}
	namespace, _ := ctx.Params["namespace"].(string)
	podName, _ := ctx.Params["pod_name"].(string)
	container, _ := ctx.Params["container"].(string)
	return aiK8sSessionFilter{
		ClusterID: clusterID,
		Namespace: strings.TrimSpace(namespace),
		PodName:   strings.TrimSpace(podName),
		Container: strings.TrimSpace(container),
	}, nil
}

func resolveOptionalK8sClusterID(ctx biz.SkillContext) (uint, error) {
	if v, ok := ctx.Params["cluster_id"].(float64); ok && v > 0 {
		return uint(v), nil
	}
	if v, ok := ctx.Params["cluster_name"].(string); ok && strings.TrimSpace(v) != "" {
		clusterID, _, err := FindClusterID(ctx.DB, ctx.Params)
		if err != nil {
			return 0, err
		}
		return clusterID, nil
	}
	return 0, nil
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func normalizeK8sExecError(err error, shellRequired bool) error {
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(err.Error())
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "executable file not found") ||
		strings.Contains(lower, "no such file or directory") ||
		strings.Contains(lower, `stat /bin/sh`) ||
		strings.Contains(lower, `stat /bin/bash`) {
		if shellRequired {
			return fmt.Errorf("容器内缺少可用 shell（/bin/sh 或 bash），无法建立交互式 Pod 会话。可改用单条非交互命令，或确认镜像是否为 distroless/minimal")
		}
		return fmt.Errorf("容器内缺少可用 shell（/bin/sh），无法执行当前命令。请确认镜像是否包含 shell，或改用无需 shell 的排障方式")
	}
	if strings.Contains(lower, "unable to upgrade connection") {
		return fmt.Errorf("Pod exec 连接升级失败: %s", msg)
	}
	return err
}
