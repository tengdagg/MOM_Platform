package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// dsmlPattern 匹配 DeepSeek 模型内部标记（<｜DSML｜...> 或 <|DSML|...> 等）
// DeepSeek R1/V3 有时将 function_calls 以文本形式输出，需要过滤
var dsmlPattern = regexp.MustCompile(`(?s)<[｜|]DSML[｜|].*`)

// thinkPattern 匹配模型输出中的 <think>...</think> 深度思考标记
var thinkPattern = regexp.MustCompile(`(?s)<think>.*?</think>\s*`)

// toolCallTextPattern 匹配模型以文本形式输出的 <tool_call>...</tool_call> 标记
// DeepSeek 等模型在多轮对话后可能不走 function calling API，而是直接输出此格式
var toolCallTextPattern = regexp.MustCompile(`(?s)<tool_call>.*?</tool_call>`)

// toolCallTextUnclosedPattern 匹配未闭合的 <tool_call>（流式输出未结束时）
var toolCallTextUnclosedPattern = regexp.MustCompile(`(?s)<tool_call>.*`)

// dsmlInvokePattern / dsmlParamPattern 用于恢复 DeepSeek 以文本泄漏的函数调用
var dsmlInvokePattern = regexp.MustCompile(`<[｜|]DSML[｜|]invoke name="([^"]+)">`)
var dsmlParamPattern = regexp.MustCompile(`<[｜|]DSML[｜|]parameter name="([^"]+)"(?: [^>]*)?>(.*)`)

// stripModelArtifacts 过滤模型输出中的内部标记（如 DeepSeek DSML、<think>、<tool_call> 标签等）
func stripModelArtifacts(content string) string {
	content = thinkPattern.ReplaceAllString(content, "")
	content = dsmlPattern.ReplaceAllString(content, "")
	content = toolCallTextPattern.ReplaceAllString(content, "")
	content = toolCallTextUnclosedPattern.ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}

// parseLeakedToolCalls 解析模型以文本形式泄漏的 <tool_call> 标记，转换为标准 ToolCall 切片
// 支持的格式:
//
//	<tool_call>tool-name<arg_key>k1</arg_key><arg_value>v1</arg_value>...</tool_call>
func parseLeakedToolCalls(content string) []ToolCall {
	// 提取所有 <tool_call>...</tool_call> 块
	re := regexp.MustCompile(`(?s)<tool_call>(.*?)</tool_call>`)
	matches := re.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		// 尝试匹配未闭合的 <tool_call>
		reUnclosed := regexp.MustCompile(`(?s)<tool_call>(.*)`)
		matches = reUnclosed.FindAllStringSubmatch(content, -1)
	}
	if len(matches) == 0 {
		return nil
	}

	var result []ToolCall
	argKeyRe := regexp.MustCompile(`(?s)<arg_key>(.*?)</arg_key>\s*<arg_value>(.*?)</arg_value>`)

	for idx, m := range matches {
		body := strings.TrimSpace(m[1])
		if body == "" {
			continue
		}

		// 工具名：<tool_call> 和第一个 <arg_key> 之间的文本
		toolName := body
		if pos := strings.Index(body, "<arg_key>"); pos > 0 {
			toolName = strings.TrimSpace(body[:pos])
		}
		toolName = strings.TrimSpace(toolName)
		if toolName == "" {
			continue
		}

		// 解析参数键值对
		params := make(map[string]any)
		argMatches := argKeyRe.FindAllStringSubmatch(body, -1)
		for _, am := range argMatches {
			key := strings.TrimSpace(am[1])
			value := strings.TrimSpace(am[2])
			if key != "" {
				// 尝试将数值字符串转为数字
				params[key] = value
			}
		}

		argsJSON, _ := json.Marshal(params)

		result = append(result, ToolCall{
			ID:   fmt.Sprintf("leaked_call_%d", idx),
			Type: "function",
			Function: FunctionCall{
				Name:      toolName,
				Arguments: string(argsJSON),
			},
		})

		log.Printf("[agent] 检测到泄漏的 <tool_call> 标记，已解析: tool=%s args=%s", toolName, string(argsJSON))
	}

	return result
}

// parseLeakedDSMLToolCalls 解析 DeepSeek 以 DSML 文本泄漏的函数调用，转换为标准 ToolCall 切片。
// 支持的格式:
//
//	<｜DSML｜invoke name="host-exec_command">
//	<｜DSML｜parameter name="ip" string="true">172.20.200.237
//	<｜DSML｜parameter name="command" string="true">lsblk -o NAME,SIZE
func parseLeakedDSMLToolCalls(content string) []ToolCall {
	var result []ToolCall
	currentName := ""
	params := make(map[string]any)

	flushCurrent := func() {
		if strings.TrimSpace(currentName) == "" {
			return
		}
		argsJSON, _ := json.Marshal(params)
		result = append(result, ToolCall{
			ID:   fmt.Sprintf("leaked_dsml_call_%d", len(result)),
			Type: "function",
			Function: FunctionCall{
				Name:      strings.TrimSpace(currentName),
				Arguments: string(argsJSON),
			},
		})
		log.Printf("[agent] 检测到泄漏的 DSML 函数调用，已解析: tool=%s args=%s", currentName, string(argsJSON))
		currentName = ""
		params = make(map[string]any)
	}

	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		if m := dsmlInvokePattern.FindStringSubmatch(line); len(m) > 1 {
			flushCurrent()
			currentName = strings.TrimSpace(m[1])
			continue
		}

		if currentName == "" {
			continue
		}

		if m := dsmlParamPattern.FindStringSubmatch(line); len(m) > 2 {
			key := strings.TrimSpace(m[1])
			value := strings.TrimSpace(m[2])
			if key != "" {
				params[key] = value
			}
		}
	}

	flushCurrent()
	return result
}

// parseAnyLeakedToolCalls 同时尝试恢复 <tool_call> 和 DSML 文本形式的函数调用。
func parseAnyLeakedToolCalls(content string) []ToolCall {
	var result []ToolCall
	if strings.Contains(content, "<tool_call>") {
		result = append(result, parseLeakedToolCalls(content)...)
	}
	if strings.Contains(content, "<｜DSML｜") || strings.Contains(content, "<|DSML|") {
		result = append(result, parseLeakedDSMLToolCalls(content)...)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// escapeHTMLForFrontend 将模型输出中的 HTML 特殊字符转义，防止前端 v-html 误解析
func escapeHTMLForFrontend(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// inferToolRiskLevel 根据工具参数推断动作级风险，避免混合 Skill 一律显示为高风险。
func (a *Agent) inferToolRiskLevel(name string, argsJSON string, fallback string) string {
	displayName := a.registry.ResolveName(name)
	var params map[string]any
	if argsJSON != "" {
		_ = json.Unmarshal([]byte(argsJSON), &params)
	}

	getAction := func() string {
		if params == nil {
			return ""
		}
		if action, ok := params["action"].(string); ok {
			return strings.ToLower(strings.TrimSpace(action))
		}
		return ""
	}
	getCommand := func() string {
		if params == nil {
			return ""
		}
		if command, ok := params["command"].(string); ok {
			return strings.ToLower(strings.TrimSpace(command))
		}
		return ""
	}
	isLikelySafeHostCommand := func(command string) bool {
		if command == "" {
			return false
		}
		safePrefixes := []string{
			"ls", "lsblk", "df", "du", "free", "uname", "uptime", "who", "w", "id", "pwd",
			"cat ", "grep ", "egrep ", "rg ", "ps ", "top ", "vmstat", "iostat", "ss ", "netstat ",
			"journalctl ", "tail ", "head ", "env", "printenv", "mount", "findmnt",
			"fdisk -l", "fdisk --list", "parted ",
			"pvs", "vgs", "lvs", "pvdisplay", "vgdisplay", "lvdisplay", "blkid",
			"ip addr show", "ip route show", "ip link show",
			"systemctl status ", "service ",
			"docker ps", "docker images", "docker inspect", "docker logs", "docker stats",
		}
		for _, prefix := range safePrefixes {
			if strings.HasPrefix(command, prefix) {
				if prefix == "parted " && !strings.Contains(command, " print") {
					continue
				}
				if prefix == "service " && !strings.HasSuffix(command, " status") {
					continue
				}
				return true
			}
		}
		return false
	}
	isLikelySafeDeviceCommand := func(command string) bool {
		if command == "" {
			return false
		}
		safePrefixes := []string{
			"show ", "display ", "dis ", "ping ", "traceroute ", "tracert ",
			"dir ", "more ", "terminal length ", "screen-length ",
		}
		for _, prefix := range safePrefixes {
			if strings.HasPrefix(command, prefix) {
				return true
			}
		}
		return command == "show" || command == "display" || command == "dis"
	}

	switch displayName {
	case "host.exec_command", "task.execute":
		if isLikelySafeHostCommand(getCommand()) {
			return "low"
		}
		return "critical"
	case "device.exec_command":
		if isLikelySafeDeviceCommand(getCommand()) {
			return "low"
		}
		return "critical"
	case "host.collect", "device.test_connection":
		return "low"
	case "task.ansible":
		if getAction() == "list" {
			return "low"
		}
		return "high"
	case "host.file_manage":
		switch getAction() {
		case "list", "read", "download":
			return "low"
		case "backup":
			return "medium"
		case "write":
			return "high"
		}
	case "host.manage", "device.manage":
		switch getAction() {
		case "list_credentials", "list_groups":
			return "low"
		case "create", "update":
			return "high"
		case "delete":
			return "critical"
		}
	case "monitor.alert_config":
		switch getAction() {
		case "list":
			return "low"
		case "enable", "disable", "create", "delete":
			return "medium"
		}
	case "k8s.kubectl":
		switch getAction() {
		case "get", "describe", "logs", "events", "top", "cluster_status":
			return "low"
		case "scale", "restart":
			return "high"
		case "delete", "cordon", "uncordon", "drain":
			return "critical"
		}
	case "k8s.helm_manage":
		switch getAction() {
		case "list", "status":
			return "low"
		case "install", "upgrade", "uninstall":
			return "high"
		}
	}

	return fallback
}

func (a *Agent) inferToolRiskMode(name string) string {
	switch a.registry.ResolveName(name) {
	case "host.exec_command", "task.execute", "device.exec_command",
		"host.collect", "device.test_connection",
		"task.ansible", "host.file_manage", "host.manage", "device.manage",
		"monitor.alert_config", "k8s.kubectl", "k8s.helm_manage":
		return "dynamic"
	default:
		return "static"
	}
}

func (a *Agent) inferToolRiskHint(name string, argsJSON string, fallbackLevel string) string {
	displayName := a.registry.ResolveName(name)
	level := a.inferToolRiskLevel(name, argsJSON, fallbackLevel)
	switch displayName {
	case "host.exec_command":
		if level == "low" {
			return "查看类主机命令，直接执行。"
		}
		return "主机变更类命令，需要人工确认后执行。"
	case "task.execute":
		if level == "low" {
			return "分组批量查看类命令，直接执行。"
		}
		return "分组批量变更命令，需要人工确认后执行。"
	case "device.exec_command":
		if level == "low" {
			return "网络设备只读查询命令，直接执行。"
		}
		return "网络设备未知或变更类命令，需要人工确认后执行。"
	case "host.collect":
		return "主机固定信息采集，属于低风险只读操作。"
	case "device.test_connection":
		return "网络设备连通性测试，属于探测/刷新类操作。"
	case "task.ansible":
		if level == "low" {
			return "仅查询 Ansible 模板，直接执行。"
		}
		return "提交 Ansible Playbook 执行任务，需要人工确认。"
	case "host.file_manage":
		switch level {
		case "low":
			return "文件查看类操作，直接执行。"
		case "medium":
			return "文件备份操作，需要人工确认。"
		default:
			return "文件写入或覆盖操作，需要人工确认。"
		}
	case "host.manage", "device.manage":
		if level == "low" {
			return "凭证/分组查询操作，直接执行。"
		}
		return "资产创建、修改或删除操作，需要人工确认。"
	case "monitor.alert_config":
		if level == "low" {
			return "告警规则查询操作，直接执行。"
		}
		return "告警规则变更操作，会修改规则状态或配置。"
	case "k8s.kubectl":
		if level == "low" {
			return "Kubernetes 查询类操作，直接执行。"
		}
		return "Kubernetes 变更或节点维护操作，需要人工确认。"
	case "k8s.helm_manage":
		if level == "low" {
			return "Helm 查询类操作，直接执行。"
		}
		return "Helm 安装、升级或卸载操作，需要人工确认。"
	default:
		return fmt.Sprintf("当前动作风险等级为 %s。", level)
	}
}

type PendingToolAction struct {
	ToolName   string
	ParamsJSON string
	RiskLevel  string
	Warning    string
}

func buildPendingConfirmationReply(result map[string]any) string {
	message, _ := result["message"].(string)
	warning, _ := result["warning"].(string)
	command, _ := result["command"].(string)
	message = strings.TrimSpace(message)
	warning = strings.TrimSpace(warning)
	command = strings.TrimSpace(command)

	parts := make([]string, 0, 6)
	parts = append(parts, "### 待确认操作")

	targetSummary := buildPendingTargetSummary(result)
	if targetSummary == "" && message != "" && command == "" {
		targetSummary = sanitizePendingMessage(message)
	}
	if targetSummary != "" {
		parts = append(parts, "**目标主机**\n"+targetSummary)
	}

	timeoutSummary := buildPendingTimeoutSummary(result)
	if timeoutSummary != "" {
		parts = append(parts, "**超时设置**\n- "+timeoutSummary)
	}

	riskSummary := buildPendingRiskSummary(result, warning)
	if riskSummary != "" {
		parts = append(parts, "**风险说明**\n"+riskSummary)
	}

	if command != "" {
		parts = append(parts, "**执行命令**\n```bash\n"+sanitizeCommandForMarkdown(command)+"\n```")
	} else if message != "" {
		parts = append(parts, "**操作说明**\n"+sanitizePendingMessage(message))
	}

	reply := strings.TrimSpace(strings.Join(parts, "\n\n"))
	if reply == "" {
		reply = "### 待确认操作\n检测到该操作需要人工确认后才能继续执行。"
	}
	if !containsExplicitConfirmationInstruction(reply) {
		reply += "\n\n请回复“确认”继续执行，或回复“取消”终止本次操作。"
	}
	return reply
}

func readCountField(result map[string]any, key string) int {
	if v, ok := result[key].(float64); ok && v > 0 {
		return int(v)
	}
	if v, ok := result[key].(int); ok && v > 0 {
		return v
	}
	return 0
}

func sanitizePendingMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	if strings.HasPrefix(message, "命令 [") && strings.Contains(message, "] 将在 ") {
		if idx := strings.LastIndex(message, "] 将在 "); idx > 0 {
			return "即将" + message[idx+2:]
		}
	}
	return message
}

func sanitizeCommandForMarkdown(command string) string {
	command = strings.TrimSpace(command)
	command = strings.ReplaceAll(command, "```", "'''")
	command = strings.ReplaceAll(command, " && ", " && \\\n")
	command = strings.ReplaceAll(command, " || ", " || \\\n")
	return command
}

func buildPendingTargetSummary(result map[string]any) string {
	if count := readCountField(result, "hostCount"); count > 0 {
		lines := []string{fmt.Sprintf("- 目标类型：主机（共 %d 台）", count)}
		lines = append(lines, buildTargetLinesFromSlice(result["hosts"], count)...)
		return strings.Join(lines, "\n")
	}
	if count := readCountField(result, "deviceCount"); count > 0 {
		lines := []string{fmt.Sprintf("- 目标类型：网络设备（共 %d 台）", count)}
		lines = append(lines, buildTargetLinesFromSlice(result["devices"], count)...)
		return strings.Join(lines, "\n")
	}
	if host, _ := result["host"].(string); strings.TrimSpace(host) != "" {
		return "- " + strings.TrimSpace(host)
	}
	return ""
}

func buildTargetLinesFromSlice(items any, total int) []string {
	v := reflect.ValueOf(items)
	if !v.IsValid() || v.Kind() != reflect.Slice || v.Len() == 0 {
		return nil
	}
	limit := v.Len()
	if limit > 3 {
		limit = 3
	}
	lines := make([]string, 0, limit+1)
	for i := 0; i < limit; i++ {
		item := v.Index(i)
		if item.Kind() == reflect.Pointer && !item.IsNil() {
			item = item.Elem()
		}
		name := extractFieldString(item, "Name")
		ip := extractFieldString(item, "IP")
		switch {
		case name != "" && ip != "":
			lines = append(lines, fmt.Sprintf("- %s (%s)", name, ip))
		case ip != "":
			lines = append(lines, "- "+ip)
		case name != "":
			lines = append(lines, "- "+name)
		}
	}
	if total > limit {
		lines = append(lines, fmt.Sprintf("- 其余 %d 台已省略", total-limit))
	}
	return lines
}

func extractFieldString(v reflect.Value, fieldName string) string {
	if !v.IsValid() {
		return ""
	}
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Map {
		mv := v.MapIndex(reflect.ValueOf(strings.ToLower(fieldName)))
		if value := reflectValueToString(mv); value != "" {
			return value
		}
		mv = v.MapIndex(reflect.ValueOf(fieldName))
		if value := reflectValueToString(mv); value != "" {
			return value
		}
		return ""
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	f := v.FieldByName(fieldName)
	if !f.IsValid() || f.Kind() != reflect.String {
		return ""
	}
	return strings.TrimSpace(f.String())
}

func reflectValueToString(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.String {
		return strings.TrimSpace(v.String())
	}
	return ""
}

func buildPendingTimeoutSummary(result map[string]any) string {
	if timeoutVal, ok := result["timeout"].(float64); ok && timeoutVal > 0 {
		return fmt.Sprintf("%ds", int(timeoutVal))
	}
	if timeoutVal, ok := result["timeout"].(int); ok && timeoutVal > 0 {
		return fmt.Sprintf("%ds", timeoutVal)
	}
	return ""
}

func buildPendingRiskSummary(result map[string]any, warning string) string {
	lines := make([]string, 0, 2)
	if riskLevel, _ := result["effectiveRiskLevel"].(string); strings.TrimSpace(riskLevel) != "" {
		lines = append(lines, "- 风险等级："+formatPendingRiskLevel(riskLevel))
	}
	warning = strings.TrimSpace(warning)
	if warning != "" {
		warning = strings.TrimPrefix(warning, "⚠️ ")
		lines = append(lines, "- "+warning)
	}
	return strings.Join(lines, "\n")
}

func formatPendingRiskLevel(risk string) string {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "critical":
		return "危险"
	case "high":
		return "高风险"
	case "medium":
		return "中风险"
	case "low":
		return "低风险"
	default:
		return risk
	}
}

func containsExplicitConfirmationInstruction(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	return strings.Contains(text, "请回复“确认”继续执行") ||
		strings.Contains(text, `请回复"确认"继续执行`) ||
		strings.Contains(text, "请回复“确认”或“取消”") ||
		strings.Contains(text, "请回复确认或取消") ||
		strings.Contains(text, "确认执行吗")
}

func mergePendingConfirmationReply(existing string, result map[string]any) string {
	existing = strings.TrimSpace(existing)
	pendingReply := buildPendingConfirmationReply(result)
	if existing == "" {
		return pendingReply
	}
	if containsExplicitConfirmationInstruction(existing) {
		return existing
	}
	if strings.Contains(existing, pendingReply) {
		return existing
	}
	return strings.TrimSpace(existing + "\n\n" + pendingReply)
}

func appendTimelineText(timeline *[]map[string]any, content string) int {
	content = strings.TrimSpace(content)
	if content == "" {
		return -1
	}
	if len(*timeline) > 0 {
		last := (*timeline)[len(*timeline)-1]
		if blockType, _ := last["type"].(string); blockType == "text" {
			lastContent, _ := last["content"].(string)
			last["content"] = strings.TrimSpace(lastContent + "\n\n" + content)
			(*timeline)[len(*timeline)-1] = last
			return len(*timeline) - 1
		}
	}
	*timeline = append(*timeline, map[string]any{
		"type":    "text",
		"content": content,
	})
	return len(*timeline) - 1
}

func updateTimelineText(timeline *[]map[string]any, idx int, content string) {
	content = strings.TrimSpace(content)
	if idx < 0 || idx >= len(*timeline) || content == "" {
		appendTimelineText(timeline, content)
		return
	}
	block := (*timeline)[idx]
	if blockType, _ := block["type"].(string); blockType != "text" {
		appendTimelineText(timeline, content)
		return
	}
	block["content"] = content
	(*timeline)[idx] = block
}

func appendTimelineTool(timeline *[]map[string]any, toolName string, params string, result string, riskLevel string, riskMode string, riskHint string, status string) {
	if strings.TrimSpace(toolName) == "" {
		return
	}
	*timeline = append(*timeline, map[string]any{
		"type":      "tool",
		"toolName":  toolName,
		"params":    params,
		"result":    result,
		"riskLevel": riskLevel,
		"riskMode":  riskMode,
		"riskHint":  riskHint,
		"status":    status,
	})
}

func isAffirmativeConfirmation(msg string) bool {
	s := strings.ToLower(strings.TrimSpace(msg))
	if s == "" {
		return false
	}
	affirmatives := []string{
		"确认", "确认执行", "执行", "继续", "继续执行", "好的", "可以", "ok", "okay", "yes", "y",
		"请执行", "开始执行", "同意", "确认一下",
	}
	for _, candidate := range affirmatives {
		if s == candidate {
			return true
		}
	}
	return false
}

func isNegativeConfirmation(msg string) bool {
	s := strings.ToLower(strings.TrimSpace(msg))
	if s == "" {
		return false
	}
	negatives := []string{
		"取消", "不要", "不用", "停止", "算了", "先不要", "不执行", "取消执行", "no", "n",
	}
	for _, candidate := range negatives {
		if s == candidate {
			return true
		}
	}
	return false
}

func injectConfirmedParam(argsJSON string) string {
	params := make(map[string]any)
	if strings.TrimSpace(argsJSON) != "" {
		if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
			return argsJSON
		}
	}
	params["confirmed"] = true
	updated, err := json.Marshal(params)
	if err != nil {
		return argsJSON
	}
	return string(updated)
}

// SkillContext Skill 执行上下文
type SkillContext struct {
	Ctx      context.Context
	SessionID uint
	UserID   uint
	Username string
	Params   map[string]any
	DB       *gorm.DB
}

// Context 返回上下文（如果未设置则返回 Background）
func (sc SkillContext) Context() context.Context {
	if sc.Ctx != nil {
		return sc.Ctx
	}
	return context.Background()
}

// Skill 技能接口
type Skill interface {
	Name() string
	Description() string
	Parameters() json.RawMessage // JSON Schema
	Execute(ctx SkillContext) (any, error)
	RiskLevel() string // low / medium / high / critical
}

// SanitizeToolName 将 Skill 名称转换为 LLM 兼容格式
// LLM API 要求名称匹配 ^[a-zA-Z0-9_-]+$ (不允许包含点号)
// 例如: host.list -> host-list, k8s.cluster_status -> k8s-cluster_status
func SanitizeToolName(name string) string {
	return strings.ReplaceAll(name, ".", "-")
}

// ToolRegistry 工具注册中心
type ToolRegistry struct {
	mu       sync.RWMutex
	skills   map[string]Skill
	aliasMap map[string]string // LLM 清洗后的名称 -> 原始名称
}

// NewToolRegistry 创建工具注册中心
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		skills:   make(map[string]Skill),
		aliasMap: make(map[string]string),
	}
}

// Register 注册 Skill
func (r *ToolRegistry) Register(skill Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()
	originalName := skill.Name()
	r.skills[originalName] = skill
	// 建立 LLM 清洗名称到原始名称的映射
	sanitized := SanitizeToolName(originalName)
	if sanitized != originalName {
		r.aliasMap[sanitized] = originalName
	}
}

// Unregister 移除 Skill（用于删除自定义 Skill 时热卸载）
func (r *ToolRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.skills, name)
	sanitized := SanitizeToolName(name)
	delete(r.aliasMap, sanitized)
}

// Get 获取 Skill（支持原始名称和 LLM 清洗后的名称）
func (r *ToolRegistry) Get(name string) (Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// 先用原始名称查找
	if s, ok := r.skills[name]; ok {
		return s, ok
	}
	// 再用别名查找（LLM 返回的是清洗后的名称）
	if original, ok := r.aliasMap[name]; ok {
		if s, ok := r.skills[original]; ok {
			return s, ok
		}
	}
	return nil, false
}

// ResolveName 将 LLM 返回的清洗名称解析为原始 Skill 名称
func (r *ToolRegistry) ResolveName(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.skills[name]; ok {
		return name
	}
	if original, ok := r.aliasMap[name]; ok {
		return original
	}
	return name
}

// GetAll 获取所有 Skill
func (r *ToolRegistry) GetAll() []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Skill, 0, len(r.skills))
	for _, s := range r.skills {
		result = append(result, s)
	}
	return result
}

// GetToolDefinitions 获取所有工具定义（发送给 LLM）
// 名称会被清洗为 LLM 兼容格式（点号替换为连字符）
func (r *ToolRegistry) GetToolDefinitions() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDefinition, 0, len(r.skills))
	for _, s := range r.skills {
		defs = append(defs, ToolDefinition{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        SanitizeToolName(s.Name()),
				Description: s.Description(),
				Parameters:  s.Parameters(),
			},
		})
	}
	return defs
}

// GetToolDefinitionsFiltered 获取工具定义（排除被禁用的 Skills）
func (r *ToolRegistry) GetToolDefinitionsFiltered(db *gorm.DB) []ToolDefinition {
	// 查询被禁用的 Skills
	disabledSet := make(map[string]bool)
	var disabledSkills []SkillDefinition
	if db != nil {
		db.Select("name").Where("is_enabled = ?", false).Find(&disabledSkills)
		for _, s := range disabledSkills {
			disabledSet[s.Name] = true
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDefinition, 0, len(r.skills))
	for _, s := range r.skills {
		if disabledSet[s.Name()] {
			continue // 跳过禁用的 Skill
		}
		defs = append(defs, ToolDefinition{
			Type: "function",
			Function: ToolFunctionDef{
				Name:        SanitizeToolName(s.Name()),
				Description: s.Description(),
				Parameters:  s.Parameters(),
			},
		})
	}
	return defs
}

// AgentEvent Agent 事件（发送给前端）
type AgentEvent struct {
	Type         string `json:"type"` // text_delta / tool_call_start / tool_call_result / action_confirm / message_end / error
	Content      string `json:"content,omitempty"`
	ToolName     string `json:"toolName,omitempty"`
	ToolParams   string `json:"toolParams,omitempty"`
	ToolResult   string `json:"toolResult,omitempty"`
	ActionID     string `json:"actionId,omitempty"`
	Description  string `json:"description,omitempty"`
	RiskLevel    string `json:"riskLevel,omitempty"`
	RiskMode     string `json:"riskMode,omitempty"`
	RiskHint     string `json:"riskHint,omitempty"`
	FinishReason string `json:"finishReason,omitempty"`
	Error        string `json:"error,omitempty"`
	Usage        *Usage `json:"usage,omitempty"`
}

// Usage Token 使用量
type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
}

// Agent AI Agent 核心引擎
type Agent struct {
	db             *gorm.DB
	registry       *ToolRegistry
	conversation   *ConversationManager
	contextBuilder *ContextBuilder
}

// NewAgent 创建 Agent
func NewAgent(db *gorm.DB, registry *ToolRegistry) *Agent {
	return &Agent{
		db:             db,
		registry:       registry,
		conversation:   NewConversationManager(db),
		contextBuilder: NewContextBuilder(db),
	}
}

// systemPrompt 系统提示词
const systemPrompt = `你是 MOM 运维管理平台的 AI 助手。你可以帮助用户管理和分析平台中的各类运维资源，包括主机管理、网络设备管理、Kubernetes 集群、任务执行、监控告警、审计日志等。

你的能力：
1. 查询和分析平台中的资源信息
2. 执行运维操作（高风险操作需要用户确认后才执行）
3. 生成运维报告和分析建议
4. 回答运维相关的技术问题

工作原则：
- 优先使用可用的工具来获取准确信息，不要编造数据
- 回答要清晰、结构化，善用表格和列表
- 如果工具返回错误，如实告知用户并给出建议

⚠️ 高风险操作确认机制（非常重要）：
对于高风险操作（扩缩容、重启、远程命令执行、节点管理等），执行流程如下：
1. 第一次调用工具时，不传 confirmed 参数。工具会返回 status="pending_confirmation" 和操作详情。
2. 你必须把操作详情清晰地展示给用户，询问用户是否确认执行。
3. 如果用户明确表示"确认"、"执行"、"好的"、"可以"等肯定回复，则第二次调用同一个工具，带上 confirmed=true 参数，这样工具才会真正执行操作。
4. 如果用户说"取消"或"不要"，则不再调用工具，告知用户操作已取消。

示例流程：
- 用户: "把 order-service 扩展到 5 个副本"
- 你: 调用 k8s-scale(resource_name="order-service", replicas=5, namespace="default") → 返回 pending_confirmation
- 你: "即将把 Deployment/order-service 的副本数从当前调整为 5，确认执行吗？"
- 用户: "确认"
- 你: 调用 k8s-scale(resource_name="order-service", replicas=5, namespace="default", confirmed=true) → 返回 success
- 你: "✅ 已成功将 order-service 的副本数调整为 5"

🔧 复杂运维操作指导：
当用户需要执行复杂运维操作（如安装软件、修改配置、磁盘扩容等）时，请遵循以下原则：
1. 分步执行：将复杂操作拆分为多个安全的小步骤，每步执行一个命令，不要一次性组合大量命令
2. 先查后改：修改配置前先用 host-file_manage(action="read") 查看当前内容；安装前先检查是否已安装
3. 备份优先：修改配置文件前先用 host-file_manage(action="backup") 备份原文件
4. 合理超时：安装软件等耗时操作使用 timeout=120 或更大值（默认 30 秒可能不够）
5. 验证结果：操作完成后执行验证命令确认操作成功（如 systemctl status、cat 查看配置等）
6. 配置修改推荐流程：read（查看）→ backup（备份）→ write（写入新内容）→ 验证 → 重载服务

🔗 跨工具协作流程（务必遵循"先查后做"原则）：

📌 主机运维：
- 查主机信息：host-list（列表筛选）→ host-detail（单台详情）
- 远程操作：host-detail（确认目标）→ host-exec_command（执行命令）→ host-exec_command（验证结果）
- 配置修改：host-file_manage(read) → host-file_manage(backup) → host-file_manage(write) → host-exec_command（验证/重载）
- 批量操作：host-list（按条件筛选得到 host_id 列表）→ host-exec_command(host_ids=[...])（批量执行）
- 新主机接入：host-manage(list_credentials)（查凭证 ID）→ host-manage(list_groups)（查分组 ID）→ host-manage(create)（创建）→ host-collect（采集信息）
- 健康巡检：host-analyze（找出异常主机）→ host-detail（查看异常主机详情）→ host-exec_command（深入排查）
- 磁盘清理：host-exec_command(df -h) → host-exec_command(du -sh) 找大目录 → host-file_manage(backup) → 清理 → 验证

📌 网络设备管理：
- 查设备信息：device-list（列表筛选）→ device-detail（单台详情）
- 批量操作前必须先查：device-list（获取设备 ID 列表）→ device-exec_command(device_ids=[...])（批量执行命令）
- 设备巡检：device-list → device-test_connection（批量测试连通性）→ 对离线设备 device-detail 排查
- 新设备接入：device-manage(list_credentials) → device-manage(list_groups) → device-manage(create) → device-test_connection（验证连通性）
- 配置查看：device-list/detail（查设备 ID）→ device-exec_command(command="show running-config" 或 "display current-configuration")
- 远程命令：只执行 show/display 等查看类命令；配置变更类命令应提醒用户高风险
- 连续配置：同一对话内对同一网络设备重复调用 device-exec_command 时，会尽量复用交互式 shell 上下文；如果已进入配置模式，后续命令默认仍在当前模式中执行，必要时请显式执行 end/exit 返回上级模式
- 会话管理：可用 device-session_status 查看当前对话中的网络设备会话状态，必要时用 device-close_session 主动结束当前设备会话或全部设备会话

📌 Kubernetes 集群：
- 资源查询：k8s-kubectl(action="get", resource="deployments/pods/services/..." ) 查看资源列表或详情
- 故障排查：k8s-diagnose（诊断 Pod/Node）→ k8s-log_query（查看日志）→ k8s-kubectl(action="describe")（查事件）→ k8s-restart（重启修复）
- 扩缩容：k8s-kubectl(action="get", resource="deployments")（查看当前副本数）→ k8s-scale（调整副本数）
- 节点维护：k8s-node_manage(action="cordon")（标记不可调度）→ k8s-node_manage(action="drain")（排空）→ 维护 → k8s-node_manage(action="uncordon")（恢复）
- Helm 应用：k8s-helm_manage(action="list")（查看已安装 Release）→ k8s-helm_manage(action="install/upgrade/uninstall")
- 创建资源：k8s-kubectl(action="create/apply", yaml="...")

📌 任务执行：
- Ad-hoc 命令：task-execute（指定 IP/分组/主机 ID 列表执行命令）
- Ansible Playbook：task-ansible（执行 Playbook）
- 查看历史：task-history（查询之前的执行结果和状态）

📌 监控告警：
- 域名监控：monitor-domain_status（查看状态）→ monitor-domain_manage(action="create")（添加监控）
- 告警管理：monitor-alert_summary（汇总告警）→ monitor-alert_config(action="list")（查看规则）→ monitor-alert_config(action="create/update")（配置规则）
- SSL 到期排查：monitor-domain_status → 筛选证书即将到期的域名 → 提醒用户续期

📌 审计安全：
- 登录安全：audit-login_analysis（分析异常登录和暴力破解）→ analysis-security_audit（综合安全评估）
- 操作追踪：audit-operation_summary（统计操作分布）→ audit-data_changes（追踪具体变更）
- 会话审计：audit-session_summary（统计终端会话情况）

📌 综合报告与分析：
- 每日巡检：host-analyze + device-list + monitor-domain_status + monitor-alert_summary + audit-login_analysis → 汇总输出
- 周报/月报：analysis-infra_report（基础设施概况）+ audit-operation_summary（操作统计）+ monitor-alert_summary（告警统计）
- 容量规划：host-analyze（资源使用分析）→ analysis-capacity_plan（生成扩容建议）
- 安全审计：audit-login_analysis + audit-data_changes + analysis-security_audit → 综合安全报告

📌 云资源管理：
- 查看云账号：cloud-list_accounts → cloud-list_instances（查看已导入实例）
- 导入云主机：cloud-list_accounts（获取账号 ID）→ cloud-list_instances（查看可导入实例）→ cloud-import_hosts（导入到 MOM 平台）→ host-collect（采集信息）`

// buildMessages 构建发送给 LLM 的消息列表（公共逻辑）
// 1. 系统提示词 + 动态上下文
// 2. 如果有摘要，注入为 system 消息
// 3. 最近 20 条历史消息
// 4. 当前用户消息
func (a *Agent) buildMessages(sessionID uint, userID uint, username string, userMessage string) []ChatCompletionMessage {
	// 获取历史消息
	historyMsgs, _ := a.conversation.GetRecentMessages(sessionID, 20)

	// 构建系统上下文
	userContext := a.contextBuilder.BuildSystemContext(userID, username)
	fullSystemPrompt := systemPrompt + "\n\n--- 当前上下文 ---\n" + userContext

	// 构建消息列表
	messages := []ChatCompletionMessage{
		{Role: "system", Content: fullSystemPrompt},
	}

	// 注入历史摘要（如果有）
	session, err := a.conversation.GetSessionByID(sessionID)
	if err == nil && session.Summary != "" {
		messages = append(messages, ChatCompletionMessage{
			Role:    "system",
			Content: "以下是之前对话的摘要，帮助你了解对话的完整背景：\n" + session.Summary,
		})
	}

	// 加载最近的历史消息（还原工具调用上下文）
	for _, msg := range historyMsgs {
		if msg.Role == "assistant" && msg.ToolCalls != "" {
			var toolRecords []map[string]any
			if err := json.Unmarshal([]byte(msg.ToolCalls), &toolRecords); err == nil && len(toolRecords) > 0 {
				var tcs []ToolCall
				for i, rec := range toolRecords {
					toolName, _ := rec["toolName"].(string)
					paramsJSON, _ := rec["params"].(string)
					if toolName == "" {
						continue
					}
					tcs = append(tcs, ToolCall{
						ID:   fmt.Sprintf("hist_%d_%d", msg.ID, i),
						Type: "function",
						Function: FunctionCall{
							Name:      SanitizeToolName(toolName),
							Arguments: paramsJSON,
						},
					})
				}
				if len(tcs) > 0 {
					messages = append(messages, ChatCompletionMessage{
						Role:      "assistant",
						Content:   msg.Content,
						ToolCalls: tcs,
					})
					for i, rec := range toolRecords {
						resultStr, _ := rec["result"].(string)
						if resultStr == "" {
							resultStr = `{"status":"unknown"}`
						}
						messages = append(messages, ChatCompletionMessage{
							Role:       "tool",
							Content:    resultStr,
							ToolCallID: fmt.Sprintf("hist_%d_%d", msg.ID, i),
						})
					}
					continue
				}
			}
		}
		messages = append(messages, ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	messages = append(messages, ChatCompletionMessage{
		Role:    "user",
		Content: userMessage,
	})

	return messages
}

// summaryPrompt 摘要生成提示词
const summaryPrompt = `请将以下对话内容压缩为一段简洁的摘要。要求：
1. 保留关键信息：用户的核心需求、重要操作结果、关键结论
2. 保留具体数据：IP地址、主机名、集群名、操作名等关键标识
3. 用第三人称描述，例如"用户查询了..."、"系统执行了..."
4. 摘要长度控制在 300-500 字以内
5. 如果已有之前的摘要，请合并新旧内容，去除重复

直接输出摘要内容，不要加任何前缀或解释。`

// triggerSummaryIfNeeded 异步检查并生成摘要
func (a *Agent) triggerSummaryIfNeeded(sessionID uint) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[agent] 摘要生成异常: %v", r)
			}
		}()

		// 消息总数 < 30 不触发
		totalMsgs := a.conversation.CountMessages(sessionID)
		if totalMsgs < 30 {
			return
		}

		// 获取当前 session 的摘要状态
		session, err := a.conversation.GetSessionByID(sessionID)
		if err != nil {
			return
		}

		// 获取需要被摘要的旧消息（排除最近 20 条，排除已摘要过的）
		oldMsgs, err := a.conversation.GetOldMessages(sessionID, session.SummaryUpToID, 20)
		if err != nil || len(oldMsgs) < 10 {
			// 新消息不足 10 条，不值得重新摘要
			return
		}

		// 构建待摘要的文本
		var textBuilder strings.Builder
		if session.Summary != "" {
			textBuilder.WriteString("【已有摘要】\n")
			textBuilder.WriteString(session.Summary)
			textBuilder.WriteString("\n\n【新增对话】\n")
		}
		for _, msg := range oldMsgs {
			if msg.Role == "user" || msg.Role == "assistant" {
				roleLabel := "用户"
				if msg.Role == "assistant" {
					roleLabel = "助手"
				}
				content := msg.Content
				// 截断过长的单条消息
				if len([]rune(content)) > 500 {
					content = string([]rune(content)[:500]) + "..."
				}
				textBuilder.WriteString(fmt.Sprintf("%s: %s\n", roleLabel, content))
			}
		}

		// 用默认模型调用 LLM 生成摘要
		var model AIModelConfig
		if err := a.db.Where("is_default = ? AND status = 1", true).First(&model).Error; err != nil {
			if err := a.db.Where("status = 1").First(&model).Error; err != nil {
				log.Printf("[agent] 摘要生成失败: 无可用模型")
				return
			}
		}

		adapter := NewModelAdapter(&model)
		summaryMessages := []ChatCompletionMessage{
			{Role: "system", Content: summaryPrompt},
			{Role: "user", Content: textBuilder.String()},
		}

		resp, err := adapter.ChatCompletion(context.Background(), summaryMessages, nil)
		if err != nil {
			log.Printf("[agent] 摘要生成失败: %v", err)
			return
		}

		if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
			newSummary := resp.Choices[0].Message.Content
			lastMsgID := oldMsgs[len(oldMsgs)-1].ID
			if err := a.conversation.UpdateSummary(sessionID, newSummary, lastMsgID); err != nil {
				log.Printf("[agent] 保存摘要失败: %v", err)
			} else {
				log.Printf("[agent] 会话 %d 摘要已更新（覆盖到消息 #%d，总消息 %d 条）", sessionID, lastMsgID, totalMsgs)
			}
		}
	}()
}

// CompactSession 手动压缩指定会话的旧上下文为摘要，保留最近消息继续参与后续对话。
func (a *Agent) CompactSession(sessionID uint) (bool, error) {
	totalMsgs := a.conversation.CountMessages(sessionID)
	if totalMsgs < 12 {
		return false, nil
	}

	session, err := a.conversation.GetSessionByID(sessionID)
	if err != nil {
		return false, err
	}

	oldMsgs, err := a.conversation.GetOldMessages(sessionID, session.SummaryUpToID, 8)
	if err != nil {
		return false, err
	}
	if len(oldMsgs) == 0 {
		return false, nil
	}

	var textBuilder strings.Builder
	if session.Summary != "" {
		textBuilder.WriteString("【已有摘要】\n")
		textBuilder.WriteString(session.Summary)
		textBuilder.WriteString("\n\n【新增对话】\n")
	}
	for _, msg := range oldMsgs {
		if msg.Role != "user" && msg.Role != "assistant" {
			continue
		}
		roleLabel := "用户"
		if msg.Role == "assistant" {
			roleLabel = "助手"
		}
		content := msg.Content
		if len([]rune(content)) > 500 {
			content = string([]rune(content)[:500]) + "..."
		}
		textBuilder.WriteString(fmt.Sprintf("%s: %s\n", roleLabel, content))
	}

	var model AIModelConfig
	if err := a.db.Where("is_default = ? AND status = 1", true).First(&model).Error; err != nil {
		if err := a.db.Where("status = 1").First(&model).Error; err != nil {
			return false, fmt.Errorf("无可用模型用于压缩上下文")
		}
	}

	adapter := NewModelAdapter(&model)
	summaryMessages := []ChatCompletionMessage{
		{Role: "system", Content: summaryPrompt},
		{Role: "user", Content: textBuilder.String()},
	}
	resp, err := adapter.ChatCompletion(context.Background(), summaryMessages, nil)
	if err != nil {
		return false, err
	}
	if len(resp.Choices) == 0 || strings.TrimSpace(resp.Choices[0].Message.Content) == "" {
		return false, fmt.Errorf("模型未返回有效摘要")
	}

	newSummary := strings.TrimSpace(resp.Choices[0].Message.Content)
	lastMsgID := oldMsgs[len(oldMsgs)-1].ID
	if err := a.conversation.UpdateSummary(sessionID, newSummary, lastMsgID); err != nil {
		return false, err
	}
	return true, nil
}

// Run 执行 Agent（非流式，简单的 ReAct 循环）
func (a *Agent) Run(ctx context.Context, adapter *ModelAdapter, sessionID uint, userMessage string, userID uint, username string, maxToolCalls int, eventCh chan<- AgentEvent) {
	messageSent := false
	defer func() {
		if r := recover(); r != nil {
			eventCh <- AgentEvent{Type: "error", Error: fmt.Sprintf("Agent 异常: %v", r)}
		}
		if !messageSent {
			eventCh <- AgentEvent{Type: "message_end"}
		}
	}()

	// 构建消息列表（含摘要注入）
	messages := a.buildMessages(sessionID, userID, username, userMessage)

	// 保存用户消息
	a.conversation.AddMessage(sessionID, "user", userMessage)
	a.conversation.AutoTitleFromFirstMessage(sessionID, userMessage)

	// 获取工具定义
	tools := a.registry.GetToolDefinitionsFiltered(a.db)

	// pendingTools：从 DB 加载上一轮 pending_confirmation 的工具（跨调用确认）
	pendingTools := a.getSessionPendingTools(sessionID)
	// forceTextOnly：当工具返回 pending_confirmation 后，下一轮迭代不传 tools，
	// 迫使 LLM 只能生成文本（确认提示），循环自然中断，等待用户真正确认。
	forceTextOnly := false

	// 收集所有工具调用记录（用于持久化）
	var allToolCallRecords []map[string]any
	var timelineRecords []map[string]any
	var contentSoFar string

	if handled, stop, replayPending := a.replayPendingAction(ctx, sessionID, userID, username, a.getLastPendingAction(sessionID), pendingTools, &messages, &allToolCallRecords, &timelineRecords, eventCh); handled {
		if replayPending {
			forceTextOnly = true
		}
		if stop {
			eventCh <- AgentEvent{Type: "message_end"}
			messageSent = true
			return
		}
	}

	// ReAct 循环
	maxIterations := maxToolCalls
	if maxIterations <= 0 {
		maxIterations = 10
	}
	for i := 0; i < maxIterations; i++ {
		// 检查是否已被取消
		select {
		case <-ctx.Done():
			a.saveAssistantMessage(sessionID, contentSoFar+"\n\n[已停止]", allToolCallRecords, timelineRecords)
			eventCh <- AgentEvent{Type: "text_delta", Content: "\n\n[已停止]"}
			return
		default:
		}

		// 如果上一轮有工具返回 pending_confirmation，本轮不传 tools，强制 LLM 只生成文本
		currentTools := tools
		if forceTextOnly {
			currentTools = nil
			forceTextOnly = false
		}

		// 调用 LLM
		resp, err := adapter.ChatCompletion(ctx, messages, currentTools)
		if err != nil {
			if ctx.Err() != nil {
				a.saveAssistantMessage(sessionID, contentSoFar+"\n\n[已停止]", allToolCallRecords, timelineRecords)
				eventCh <- AgentEvent{Type: "text_delta", Content: "\n\n[已停止]"}
				return
			}
			eventCh <- AgentEvent{Type: "error", Error: fmt.Sprintf("调用模型失败: %v", err)}
			return
		}

		if len(resp.Choices) == 0 {
			eventCh <- AgentEvent{Type: "error", Error: "模型未返回任何响应"}
			return
		}

		choice := resp.Choices[0]
		assistantMsg := choice.Message

		// 如果有文本内容，过滤模型内部标记后发送给前端
		if assistantMsg.Content != "" {
			cleaned := stripModelArtifacts(assistantMsg.Content)
			if cleaned != "" {
				contentSoFar += cleaned
				eventCh <- AgentEvent{Type: "text_delta", Content: cleaned}
			}
		}

		// 非流式也检测泄漏的文本工具调用：模型没有通过 API 调用工具，而是直接输出了调用指令
		if len(assistantMsg.ToolCalls) == 0 {
			if parsed := parseAnyLeakedToolCalls(assistantMsg.Content); len(parsed) > 0 {
				assistantMsg.ToolCalls = parsed
				log.Printf("[agent] Run: 从文本中恢复了 %d 个泄漏的工具调用", len(parsed))
			}
		}

		// 如果没有工具调用，结束循环
		if len(assistantMsg.ToolCalls) == 0 {
			finalContent := stripModelArtifacts(assistantMsg.Content)
			appendTimelineText(&timelineRecords, finalContent)
			a.saveAssistantMessage(sessionID, finalContent, allToolCallRecords, timelineRecords)
			eventCh <- AgentEvent{
				Type: "message_end",
				Usage: &Usage{
					PromptTokens:     resp.Usage.PromptTokens,
					CompletionTokens: resp.Usage.CompletionTokens,
				},
			}
			messageSent = true
			return
		}

		// 处理工具调用
		messages = append(messages, ChatCompletionMessage{
			Role:             "assistant",
			Content:          assistantMsg.Content,
			ReasoningContent: assistantMsg.ReasoningContent, // DeepSeek R1 推理内容回传
			ToolCalls:        assistantMsg.ToolCalls,
		})
		textBlockIdx := appendTimelineText(&timelineRecords, stripModelArtifacts(assistantMsg.Content))

		for _, tc := range assistantMsg.ToolCalls {
			llmName := tc.Function.Name
			toolArgs := tc.Function.Arguments
			displayName := a.registry.ResolveName(llmName)

			riskLevel := ""
			if skill, ok := a.registry.Get(llmName); ok {
				riskLevel = skill.RiskLevel()
			}
			riskLevel = a.inferToolRiskLevel(llmName, toolArgs, riskLevel)
			riskMode := a.inferToolRiskMode(llmName)
			riskHint := a.inferToolRiskHint(llmName, toolArgs, riskLevel)

			eventCh <- AgentEvent{
				Type:       "tool_call_start",
				ToolName:   displayName,
				ToolParams: toolArgs,
				RiskLevel:  riskLevel,
				RiskMode:   riskMode,
				RiskHint:   riskHint,
			}

			result := a.executeTool(ctx, sessionID, llmName, toolArgs, userID, username, pendingTools)
			resultJSON, _ := json.Marshal(result)

			// 如果工具返回了动态风险等级（如安全命令跳过确认时降为 low），覆盖静态风险等级
			if effectiveRL, ok := result["effectiveRiskLevel"].(string); ok && effectiveRL != "" {
				riskLevel = effectiveRL
			}

			resultStatus := "success"
			if status, _ := result["status"].(string); status != "" {
				resultStatus = status
			}
			if errMsg, ok := result["error"]; ok && errMsg != nil {
				resultStatus = "error"
			}

			// 管理 pending 状态
			if resultStatus == "pending_confirmation" {
				// 记录 pending 状态并标记下一轮为纯文本模式
				pendingTools[llmName] = true
				pendingTools[displayName] = true
				pendingTools[SanitizeToolName(displayName)] = true
				forceTextOnly = true
			} else {
				// 非 pending 结果：消费掉 pending 状态（单次确认）
				delete(pendingTools, llmName)
				delete(pendingTools, displayName)
				delete(pendingTools, SanitizeToolName(displayName))
			}

			// 记录工具调用
			allToolCallRecords = append(allToolCallRecords, map[string]any{
				"toolName":  displayName,
				"params":    toolArgs,
				"result":    string(resultJSON),
				"riskLevel": riskLevel,
				"status":    resultStatus,
			})
			appendTimelineTool(&timelineRecords, displayName, toolArgs, string(resultJSON), riskLevel, riskMode, riskHint, resultStatus)

			eventCh <- AgentEvent{
				Type:       "tool_call_result",
				ToolName:   displayName,
				ToolResult: string(resultJSON),
				RiskLevel:  riskLevel,
				RiskMode:   riskMode,
				RiskHint:   riskHint,
			}

			messages = append(messages, ChatCompletionMessage{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
			})

			if resultStatus == "pending_confirmation" {
				pendingReply := buildPendingConfirmationReply(result)
				replyText := mergePendingConfirmationReply(contentSoFar, result)
				if strings.TrimSpace(replyText) != strings.TrimSpace(contentSoFar) {
					delta := pendingReply
					if strings.TrimSpace(contentSoFar) != "" {
						delta = "\n\n" + pendingReply
					}
					eventCh <- AgentEvent{Type: "text_delta", Content: delta}
				}
				updateTimelineText(&timelineRecords, textBlockIdx, replyText)
				a.saveAssistantMessage(sessionID, replyText, allToolCallRecords, timelineRecords)
				eventCh <- AgentEvent{Type: "message_end"}
				messageSent = true
				return
			}
		}
	}

	// 超过最大迭代次数
	appendTimelineText(&timelineRecords, "[已达到最大工具调用次数，结束处理]")
	a.saveAssistantMessage(sessionID, "\n\n[已达到最大工具调用次数，结束处理]", allToolCallRecords, timelineRecords)
	eventCh <- AgentEvent{Type: "text_delta", Content: "\n\n[已达到最大工具调用次数，结束处理]"}
}

// RunStream 流式执行 Agent
func (a *Agent) RunStream(ctx context.Context, adapter *ModelAdapter, sessionID uint, userMessage string, userID uint, username string, maxToolCalls int, eventCh chan<- AgentEvent) {
	messageSent := false
	defer func() {
		if r := recover(); r != nil {
			eventCh <- AgentEvent{Type: "error", Error: fmt.Sprintf("Agent 异常: %v", r)}
		}
		if !messageSent {
			eventCh <- AgentEvent{Type: "message_end"}
		}
		close(eventCh)
	}()

	// 构建消息列表（含摘要注入）
	messages := a.buildMessages(sessionID, userID, username, userMessage)

	// 保存用户消息
	a.conversation.AddMessage(sessionID, "user", userMessage)
	a.conversation.AutoTitleFromFirstMessage(sessionID, userMessage)

	// 获取工具定义（排除被禁用的 Skills）
	tools := a.registry.GetToolDefinitionsFiltered(a.db)

	// pendingTools：从 DB 加载上一轮 pending_confirmation 的工具（跨调用确认）
	pendingTools := a.getSessionPendingTools(sessionID)
	// forceTextOnly：当工具返回 pending_confirmation 后，下一轮迭代不传 tools，
	// 迫使 LLM 只能生成文本（确认提示），循环自然中断，等待用户真正确认。
	forceTextOnly := false

	// 收集所有工具调用记录（用于持久化）
	var allToolCallRecords []map[string]any
	var timelineRecords []map[string]any
	var contentSoFar string

	if handled, stop, replayPending := a.replayPendingAction(ctx, sessionID, userID, username, a.getLastPendingAction(sessionID), pendingTools, &messages, &allToolCallRecords, &timelineRecords, eventCh); handled {
		if replayPending {
			forceTextOnly = true
		}
		if stop {
			eventCh <- AgentEvent{Type: "message_end"}
			messageSent = true
			return
		}
	}

	// ReAct 循环
	maxIterations := maxToolCalls
	if maxIterations <= 0 {
		maxIterations = 10
	}
	for i := 0; i < maxIterations; i++ {
		// 检查是否已被取消
		select {
		case <-ctx.Done():
			a.saveAssistantMessage(sessionID, contentSoFar+"\n\n[已停止]", allToolCallRecords, timelineRecords)
			eventCh <- AgentEvent{Type: "text_delta", Content: "\n\n[已停止]"}
			eventCh <- AgentEvent{Type: "message_end"}
			messageSent = true
			return
		default:
		}

		// 如果上一轮有工具返回 pending_confirmation，本轮不传 tools，强制 LLM 只生成文本
		currentTools := tools
		wasForceTextOnly := false
		if forceTextOnly {
			currentTools = nil
			forceTextOnly = false
			wasForceTextOnly = true
		}

		// 流式调用 LLM
		streamCh, err := adapter.ChatCompletionStream(ctx, messages, currentTools)
		if err != nil {
			if ctx.Err() != nil {
				a.saveAssistantMessage(sessionID, contentSoFar+"\n\n[已停止]", allToolCallRecords, timelineRecords)
				eventCh <- AgentEvent{Type: "text_delta", Content: "\n\n[已停止]"}
				eventCh <- AgentEvent{Type: "message_end"}
				messageSent = true
				return
			}
			eventCh <- AgentEvent{Type: "error", Error: fmt.Sprintf("调用模型失败: %v", err)}
			return
		}

		// 收集完整的响应
		var contentBuilder strings.Builder
		var reasoningBuilder strings.Builder // DeepSeek R1 推理内容
		var toolCalls []ToolCall
		toolCallArgsBuilders := make(map[int]*strings.Builder)
		var finishReason string
		dsmlDetected := false     // DeepSeek 模型内部标记检测
		var dsmlPendingBuf string // 缓冲尾部 '<'，防止 DSML 标记首字符泄漏到前端
		reasoningStarted := false
		reasoningEnded := false
		// text_delta 中 <think> 标签过滤状态
		inThinkBlock := false // 是否在 <think>...</think> 块内（抑制输出）

		for event := range streamCh {
			switch event.Type {
			case "reasoning_delta":
				// DeepSeek / M37 推理内容
				if !reasoningStarted {
					reasoningStarted = true
					// 开始输出思考过程
					eventCh <- AgentEvent{Type: "text_delta", Content: "<think>\n"}
					contentSoFar += "<think>\n"
				}
				reasoningBuilder.WriteString(event.Content)
				contentSoFar += event.Content
				eventCh <- AgentEvent{Type: "text_delta", Content: event.Content}

			case "text_delta":
				if reasoningStarted && !reasoningEnded {
					reasoningEnded = true
					eventCh <- AgentEvent{Type: "text_delta", Content: "\n</think>\n\n"}
					contentSoFar += "\n</think>\n\n"
				}

				contentBuilder.WriteString(event.Content)

				// DSML / <tool_call> 检测：一旦出现模型内部标记，立即抑制所有后续输出
				if !dsmlDetected {
					fullContent := contentBuilder.String()
					if strings.Contains(fullContent, "<｜DSML｜") || strings.Contains(fullContent, "<|DSML|") ||
						strings.Contains(fullContent, "<tool_call>") {
						dsmlDetected = true
						dsmlPendingBuf = ""
						break
					}
				}
				if dsmlDetected {
					break
				}

				// <think> 标签过滤：收集可输出的文本片段
				delta := event.Content
				var textToSend string
				if !inThinkBlock {
					if idx := strings.Index(delta, "<think>"); idx >= 0 {
						textToSend = delta[:idx]
						inThinkBlock = true
						after := delta[idx+len("<think>"):]
						if ci := strings.Index(after, "</think>"); ci >= 0 {
							inThinkBlock = false
							textToSend += after[ci+len("</think>"):]
						}
					} else {
						textToSend = delta
					}
				} else {
					if idx := strings.Index(delta, "</think>"); idx >= 0 {
						inThinkBlock = false
						textToSend = delta[idx+len("</think>"):]
					}
				}

				// DSML 前缀缓冲：将 '<' 暂存，防止 DSML 标记首字符泄漏到前端
				combined := dsmlPendingBuf + textToSend
				dsmlPendingBuf = ""
				if strings.HasSuffix(combined, "<") {
					dsmlPendingBuf = "<"
					combined = combined[:len(combined)-1]
				}
				if combined != "" {
					contentSoFar += combined
					eventCh <- AgentEvent{Type: "text_delta", Content: escapeHTMLForFrontend(combined)}
				}

			case "tool_call_delta":
				for _, tc := range event.ToolCalls {
					idx := len(toolCalls) - 1
					if tc.Index != nil {
						idx = *tc.Index
					}
					if tc.ID != "" {
						for len(toolCalls) <= idx {
							toolCalls = append(toolCalls, ToolCall{Type: "function"})
						}
						toolCalls[idx].ID = tc.ID
						toolCalls[idx].Type = "function"
						toolCalls[idx].Function.Name = tc.Function.Name
						if toolCallArgsBuilders[idx] == nil {
							toolCallArgsBuilders[idx] = &strings.Builder{}
						}
					}
					if idx >= 0 && idx < len(toolCalls) {
						if toolCallArgsBuilders[idx] == nil {
							toolCallArgsBuilders[idx] = &strings.Builder{}
						}
						toolCallArgsBuilders[idx].WriteString(tc.Function.Arguments)
					}
				}

			case "finish":
				finishReason = event.FinishReason

			case "error":
				eventCh <- AgentEvent{Type: "error", Error: event.Error}
				return

			case "done":
				// 流结束
			}
		}

		// 流式结束后：刷出 DSML 缓冲区中未匹配为 DSML 的残留文本
		if dsmlPendingBuf != "" && !dsmlDetected {
			contentSoFar += dsmlPendingBuf
			eventCh <- AgentEvent{Type: "text_delta", Content: escapeHTMLForFrontend(dsmlPendingBuf)}
			dsmlPendingBuf = ""
		}

		// 组装完整的工具调用参数
		for idx, builder := range toolCallArgsBuilders {
			if idx < len(toolCalls) {
				toolCalls[idx].Function.Arguments = builder.String()
			}
		}

		fullRawContent := contentBuilder.String()
		content := stripModelArtifacts(fullRawContent)

		// forceTextOnly 下 LLM 输出了纯 DSML（无有效文本）→ 生成兜底确认提示
		if wasForceTextOnly && dsmlDetected && strings.TrimSpace(content) == "" {
			fallback := "请确认是否执行以上操作？回复 **确认** 继续执行，或回复 **取消** 终止操作。"
			content = fallback
			contentSoFar += fallback
			eventCh <- AgentEvent{Type: "text_delta", Content: fallback}
		}

		// 如果模型没有通过 API 调用工具，但文本中泄漏了函数调用，尝试恢复为真正的工具调用
		if len(toolCalls) == 0 {
			if parsed := parseAnyLeakedToolCalls(fullRawContent); len(parsed) > 0 {
				toolCalls = parsed
				finishReason = "" // 清除 stop，允许进入工具执行流程
				log.Printf("[agent] RunStream: 从文本中恢复了 %d 个泄漏的工具调用", len(parsed))
			}
		}

		// 如果没有工具调用，结束循环
		if len(toolCalls) == 0 || finishReason == "stop" {
			appendTimelineText(&timelineRecords, content)
			a.saveAssistantMessage(sessionID, content, allToolCallRecords, timelineRecords)
			eventCh <- AgentEvent{Type: "message_end"}
			messageSent = true
			return
		}

		// 处理工具调用 —— 保留 reasoning_content（DeepSeek R1）
		assistantMessage := ChatCompletionMessage{
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCalls,
		}
		if reasoningBuilder.Len() > 0 {
			rc := reasoningBuilder.String()
			assistantMessage.ReasoningContent = &rc
		}
		messages = append(messages, assistantMessage)
		textBlockIdx := appendTimelineText(&timelineRecords, content)

		for _, tc := range toolCalls {
			// 工具执行前检查是否已被取消
			if ctx.Err() != nil {
				a.saveAssistantMessage(sessionID, contentSoFar+"\n\n[已停止]", allToolCallRecords, timelineRecords)
				eventCh <- AgentEvent{Type: "text_delta", Content: "\n\n[已停止]"}
				eventCh <- AgentEvent{Type: "message_end"}
				messageSent = true
				return
			}

			llmName := tc.Function.Name
			toolArgs := tc.Function.Arguments
			displayName := a.registry.ResolveName(llmName)

			riskLevel := ""
			if skill, ok := a.registry.Get(llmName); ok {
				riskLevel = skill.RiskLevel()
			}
			riskLevel = a.inferToolRiskLevel(llmName, toolArgs, riskLevel)
			riskMode := a.inferToolRiskMode(llmName)
			riskHint := a.inferToolRiskHint(llmName, toolArgs, riskLevel)

			eventCh <- AgentEvent{
				Type:       "tool_call_start",
				ToolName:   displayName,
				ToolParams: toolArgs,
				RiskLevel:  riskLevel,
				RiskMode:   riskMode,
				RiskHint:   riskHint,
			}

			result := a.executeTool(ctx, sessionID, llmName, toolArgs, userID, username, pendingTools)
			resultJSON, _ := json.Marshal(result)

			// 如果工具返回了动态风险等级（如安全命令跳过确认时降为 low），覆盖静态风险等级
			if effectiveRL, ok := result["effectiveRiskLevel"].(string); ok && effectiveRL != "" {
				riskLevel = effectiveRL
			}

			resultStatus := "success"
			if status, _ := result["status"].(string); status != "" {
				resultStatus = status
			}
			if errMsg, ok := result["error"]; ok && errMsg != nil {
				resultStatus = "error"
			}

			// 管理 pending 状态
			if resultStatus == "pending_confirmation" {
				pendingTools[llmName] = true
				pendingTools[displayName] = true
				pendingTools[SanitizeToolName(displayName)] = true
				forceTextOnly = true
			} else {
				delete(pendingTools, llmName)
				delete(pendingTools, displayName)
				delete(pendingTools, SanitizeToolName(displayName))
			}

			allToolCallRecords = append(allToolCallRecords, map[string]any{
				"toolName":  displayName,
				"params":    toolArgs,
				"result":    string(resultJSON),
				"riskLevel": riskLevel,
				"status":    resultStatus,
			})
			appendTimelineTool(&timelineRecords, displayName, toolArgs, string(resultJSON), riskLevel, riskMode, riskHint, resultStatus)

			eventCh <- AgentEvent{
				Type:       "tool_call_result",
				ToolName:   displayName,
				ToolResult: string(resultJSON),
				RiskLevel:  riskLevel,
				RiskMode:   riskMode,
				RiskHint:   riskHint,
			}

			messages = append(messages, ChatCompletionMessage{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
			})

			if resultStatus == "pending_confirmation" {
				pendingReply := buildPendingConfirmationReply(result)
				replyText := mergePendingConfirmationReply(content, result)
				if strings.TrimSpace(replyText) != strings.TrimSpace(content) {
					additional := pendingReply
					if strings.TrimSpace(content) != "" {
						additional = "\n\n" + pendingReply
					}
					contentSoFar += additional
					eventCh <- AgentEvent{Type: "text_delta", Content: additional}
				}
				updateTimelineText(&timelineRecords, textBlockIdx, replyText)
				a.saveAssistantMessage(sessionID, replyText, allToolCallRecords, timelineRecords)
				eventCh <- AgentEvent{Type: "message_end"}
				messageSent = true
				return
			}
		}
	}

	appendTimelineText(&timelineRecords, "[已达到最大工具调用次数]")
	a.saveAssistantMessage(sessionID, "\n\n[已达到最大工具调用次数]", allToolCallRecords, timelineRecords)
	eventCh <- AgentEvent{Type: "text_delta", Content: "\n\n[已达到最大工具调用次数]"}
	eventCh <- AgentEvent{Type: "message_end"}
	messageSent = true
}

// saveAssistantMessage 保存助手消息（附带工具调用记录）并异步触发摘要
func (a *Agent) saveAssistantMessage(sessionID uint, content string, toolCallRecords []map[string]any, timelineRecords []map[string]any) {
	if len(toolCallRecords) > 0 {
		toolCallsJSON, _ := json.Marshal(toolCallRecords)
		timelineJSON, _ := json.Marshal(timelineRecords)
		a.conversation.AddMessageWithTools(sessionID, "assistant", content, string(toolCallsJSON), string(timelineJSON))
	} else {
		a.conversation.AddMessage(sessionID, "assistant", content)
	}
	// 异步检查是否需要生成摘要
	a.triggerSummaryIfNeeded(sessionID)
}

// executeTool 执行工具
// pendingTools 记录已返回 pending_confirmation 的工具名：
//   - 启动时从 DB 会话历史加载（跨 RunStream 的合法确认）
//   - 当前循环中工具返回 pending_confirmation 后由调用方追加
//
// 安全拦截：如果 LLM 对未经 pending_confirmation 的工具直接传 confirmed=true，
// 则强制移除 confirmed 参数，确保工具会返回 pending_confirmation。
func (a *Agent) executeTool(ctx context.Context, sessionID uint, name string, argsJSON string, userID uint, username string, pendingTools map[string]bool) map[string]any {
	skill, ok := a.registry.Get(name)
	if !ok {
		return map[string]any{"error": fmt.Sprintf("工具 %s 不存在", name)}
	}

	// 解析参数
	var params map[string]any
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
			return map[string]any{"error": fmt.Sprintf("参数解析失败: %v", err)}
		}
	}

	// 安全拦截：如果 LLM 传了 confirmed=true，但该工具从未返回过 pending_confirmation，
	// 则强制移除 confirmed 参数，防止 LLM 跳过确认步骤直接执行高风险操作。
	riskLevel := skill.RiskLevel()
	if riskLevel != "" && riskLevel != "low" && isParamConfirmed(params) {
		if !pendingTools[name] {
			log.Printf("[agent] 安全拦截: 用户 %d 对工具 %s 直接传 confirmed=true 但无前置 pending_confirmation，已移除 confirmed 参数", userID, name)
			delete(params, "confirmed")
		}
	}

	startTime := time.Now()

	// 执行
	result, err := skill.Execute(SkillContext{
		Ctx:      ctx,
		SessionID: sessionID,
		UserID:   userID,
		Username: username,
		Params:   params,
		DB:       a.db,
	})

	costTime := time.Since(startTime).Milliseconds()
	displayName := a.registry.ResolveName(name)

	if err != nil {
		log.Printf("[agent] Skill %s 执行失败: %v", name, err)
		// 记录失败的操作审计
		a.writeAuditLog(userID, username, displayName, argsJSON, err.Error(), skill.RiskLevel(), costTime, false)
		return map[string]any{"error": err.Error()}
	}

	// 将结果转为 map
	var resultMap map[string]any
	switch v := result.(type) {
	case map[string]any:
		resultMap = v
	default:
		resultMap = map[string]any{"result": v}
	}

	auditRiskLevel := skill.RiskLevel()
	if effectiveRL, ok := resultMap["effectiveRiskLevel"].(string); ok && effectiveRL != "" {
		auditRiskLevel = effectiveRL
	}

	// 记录操作审计（只记录非 pending_confirmation 的实际执行，以及查询操作）
	status, _ := resultMap["status"].(string)
	if status != "pending_confirmation" {
		resultBrief := a.buildAuditDescription(displayName, params, resultMap)
		a.writeAuditLog(userID, username, displayName, argsJSON, resultBrief, auditRiskLevel, costTime, true)
	}

	return resultMap
}

// isParamConfirmed 检查参数中是否包含 confirmed=true
func isParamConfirmed(params map[string]any) bool {
	if params == nil {
		return false
	}
	if v, ok := params["confirmed"].(bool); ok && v {
		return true
	}
	if v, ok := params["confirmed"].(string); ok && v == "true" {
		return true
	}
	return false
}

func (a *Agent) getLastPendingAction(sessionID uint) *PendingToolAction {
	msgs, err := a.conversation.GetRecentMessages(sessionID, 5)
	if err != nil || len(msgs) == 0 {
		return nil
	}

	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]
		if msg.Role != "assistant" || msg.ToolCalls == "" {
			continue
		}

		var toolRecords []map[string]any
		if err := json.Unmarshal([]byte(msg.ToolCalls), &toolRecords); err != nil {
			break
		}

		for j := len(toolRecords) - 1; j >= 0; j-- {
			record := toolRecords[j]
			toolName, _ := record["toolName"].(string)
			paramsJSON, _ := record["params"].(string)
			riskLevel, _ := record["riskLevel"].(string)
			resultStr, _ := record["result"].(string)
			if toolName == "" || paramsJSON == "" || resultStr == "" {
				continue
			}

			var resultMap map[string]any
			if err := json.Unmarshal([]byte(resultStr), &resultMap); err != nil {
				continue
			}
			if status, _ := resultMap["status"].(string); status != "pending_confirmation" {
				continue
			}

			warning, _ := resultMap["warning"].(string)
			return &PendingToolAction{
				ToolName:   toolName,
				ParamsJSON: paramsJSON,
				RiskLevel:  riskLevel,
				Warning:    warning,
			}
		}
		break
	}

	return nil
}

func (a *Agent) replayPendingAction(
	ctx context.Context,
	sessionID uint,
	userID uint,
	username string,
	pendingAction *PendingToolAction,
	pendingTools map[string]bool,
	messages *[]ChatCompletionMessage,
	allToolCallRecords *[]map[string]any,
	timelineRecords *[]map[string]any,
	eventCh chan<- AgentEvent,
) (bool, bool, bool) {
	if pendingAction == nil {
		return false, false, false
	}

	if isNegativeConfirmation((*messages)[len(*messages)-1].Content) {
		cancelText := "已取消上一步待确认操作。"
		if pendingAction.Warning != "" {
			cancelText = "已取消上一步待确认操作：" + pendingAction.Warning
		}
		appendTimelineText(timelineRecords, cancelText)
		a.saveAssistantMessage(sessionID, cancelText, *allToolCallRecords, *timelineRecords)
		eventCh <- AgentEvent{Type: "text_delta", Content: cancelText}
		return true, true, false
	}

	if !isAffirmativeConfirmation((*messages)[len(*messages)-1].Content) {
		return false, false, false
	}

	llmName := pendingAction.ToolName
	displayName := a.registry.ResolveName(llmName)
	confirmedArgs := injectConfirmedParam(pendingAction.ParamsJSON)
	riskLevel := pendingAction.RiskLevel
	if riskLevel == "" {
		if skill, ok := a.registry.Get(llmName); ok {
			riskLevel = skill.RiskLevel()
		}
	}
	riskLevel = a.inferToolRiskLevel(llmName, confirmedArgs, riskLevel)
	riskMode := a.inferToolRiskMode(llmName)
	riskHint := a.inferToolRiskHint(llmName, confirmedArgs, riskLevel)

	eventCh <- AgentEvent{
		Type:       "tool_call_start",
		ToolName:   displayName,
		ToolParams: confirmedArgs,
		RiskLevel:  riskLevel,
		RiskMode:   riskMode,
		RiskHint:   riskHint,
	}

	result := a.executeTool(ctx, sessionID, llmName, confirmedArgs, userID, username, pendingTools)
	resultJSON, _ := json.Marshal(result)
	if effectiveRL, ok := result["effectiveRiskLevel"].(string); ok && effectiveRL != "" {
		riskLevel = effectiveRL
	}

	resultStatus := "success"
	if status, _ := result["status"].(string); status != "" {
		resultStatus = status
	}
	if errMsg, ok := result["error"]; ok && errMsg != nil {
		resultStatus = "error"
	}

	*allToolCallRecords = append(*allToolCallRecords, map[string]any{
		"toolName":  displayName,
		"params":    confirmedArgs,
		"result":    string(resultJSON),
		"riskLevel": riskLevel,
		"status":    resultStatus,
	})
	appendTimelineTool(timelineRecords, displayName, confirmedArgs, string(resultJSON), riskLevel, riskMode, riskHint, resultStatus)

	eventCh <- AgentEvent{
		Type:       "tool_call_result",
		ToolName:   displayName,
		ToolResult: string(resultJSON),
		RiskLevel:  riskLevel,
		RiskMode:   riskMode,
		RiskHint:   riskHint,
	}

	if resultStatus == "pending_confirmation" {
		pendingTools[llmName] = true
		pendingTools[displayName] = true
		pendingTools[SanitizeToolName(displayName)] = true
	} else {
		delete(pendingTools, llmName)
		delete(pendingTools, displayName)
		delete(pendingTools, SanitizeToolName(displayName))
	}

	toolCallID := fmt.Sprintf("confirmed_pending_%d", time.Now().UnixNano())
	*messages = append(*messages, ChatCompletionMessage{
		Role: "assistant",
		ToolCalls: []ToolCall{
			{
				ID:   toolCallID,
				Type: "function",
				Function: FunctionCall{
					Name:      SanitizeToolName(displayName),
					Arguments: confirmedArgs,
				},
			},
		},
	})
	*messages = append(*messages, ChatCompletionMessage{
		Role:       "tool",
		Content:    string(resultJSON),
		ToolCallID: toolCallID,
	})

	return true, false, resultStatus == "pending_confirmation"
}

// getSessionPendingTools 从会话历史的最近一条助手消息中提取 pending_confirmation 状态的工具名称。
// 返回的 map 同时包含原始名称和 LLM 清洗后名称，用于跨 RunStream 调用的合法确认识别。
func (a *Agent) getSessionPendingTools(sessionID uint) map[string]bool {
	pendingTools := make(map[string]bool)

	msgs, err := a.conversation.GetRecentMessages(sessionID, 5)
	if err != nil || len(msgs) == 0 {
		return pendingTools
	}

	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]
		if msg.Role != "assistant" || msg.ToolCalls == "" {
			continue
		}

		var toolRecords []map[string]any
		if err := json.Unmarshal([]byte(msg.ToolCalls), &toolRecords); err != nil {
			break
		}

		// 仅当该工具的最后一条记录是 pending_confirmation 时才视为待确认
		lastStatus := make(map[string]string)
		for _, record := range toolRecords {
			toolName, _ := record["toolName"].(string)
			resultStr, _ := record["result"].(string)
			if toolName == "" || resultStr == "" {
				continue
			}
			var resultMap map[string]any
			if err := json.Unmarshal([]byte(resultStr), &resultMap); err != nil {
				continue
			}
			if status, _ := resultMap["status"].(string); status != "" {
				lastStatus[toolName] = status
			}
		}

		for toolName, status := range lastStatus {
			if status == "pending_confirmation" {
				pendingTools[toolName] = true
				pendingTools[SanitizeToolName(toolName)] = true
			}
		}
		break
	}

	return pendingTools
}

// aiOperationLog AI 操作审计日志模型（与 sys_operation_log 表对应）
type aiOperationLog struct {
	UserID      uint      `gorm:"column:user_id"`
	Username    string    `gorm:"column:username"`
	RealName    string    `gorm:"column:real_name"`
	Module      string    `gorm:"column:module"`
	Action      string    `gorm:"column:action"`
	Description string    `gorm:"column:description"`
	Method      string    `gorm:"column:method"`
	Path        string    `gorm:"column:path"`
	Params      string    `gorm:"column:params"`
	Status      int       `gorm:"column:status"`
	ErrorMsg    string    `gorm:"column:error_msg"`
	CostTime    int64     `gorm:"column:cost_time"`
	IP          string    `gorm:"column:ip"`
	UserAgent   string    `gorm:"column:user_agent"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (aiOperationLog) TableName() string { return "sys_operation_log" }

// writeAuditLog 写入 AI 操作审计日志
func (a *Agent) writeAuditLog(userID uint, username, skillName, params, description, riskLevel string, costTime int64, success bool) {
	status := 200
	errMsg := ""
	if !success {
		status = 500
		errMsg = description
		description = "执行失败: " + description
	}

	// 模块名称根据 Skill 前缀确定
	module := "AI 助手"
	if strings.HasPrefix(skillName, "k8s.") || strings.HasPrefix(skillName, "k8s-") {
		module = "AI-Kubernetes"
	} else if strings.HasPrefix(skillName, "host.") || strings.HasPrefix(skillName, "host-") {
		module = "AI-主机管理"
	} else if strings.HasPrefix(skillName, "device.") || strings.HasPrefix(skillName, "device-") {
		module = "AI-网络设备"
	} else if strings.HasPrefix(skillName, "task.") || strings.HasPrefix(skillName, "task-") {
		module = "AI-任务中心"
	} else if strings.HasPrefix(skillName, "monitor.") || strings.HasPrefix(skillName, "monitor-") {
		module = "AI-监控告警"
	} else if strings.HasPrefix(skillName, "cloud.") || strings.HasPrefix(skillName, "cloud-") {
		module = "AI-云账号"
	} else if strings.HasPrefix(skillName, "audit.") || strings.HasPrefix(skillName, "audit-") {
		module = "AI-审计分析"
	} else if strings.HasPrefix(skillName, "analysis.") || strings.HasPrefix(skillName, "analysis-") {
		module = "AI-综合分析"
	}

	// 截断过长的参数
	if len(params) > 500 {
		params = params[:500] + "..."
	}
	if len(description) > 200 {
		description = description[:200] + "..."
	}

	// 构建操作描述
	action := skillName
	if riskLevel != "" && riskLevel != "low" {
		action = fmt.Sprintf("%s [%s]", skillName, riskLevel)
	}

	now := time.Now()
	logEntry := &aiOperationLog{
		UserID:      userID,
		Username:    username + "(AI)",
		RealName:    "AI 助手",
		Module:      module,
		Action:      action,
		Description: description,
		Method:      "SKILL",
		Path:        "/ai/skill/" + skillName,
		Params:      params,
		Status:      status,
		ErrorMsg:    errMsg,
		CostTime:    costTime,
		IP:          "AI-Agent",
		UserAgent:   "MOM-AI-Agent/1.0",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 异步写入审计日志（使用 GORM Create，有错误日志）
	go func() {
		if err := a.db.Create(logEntry).Error; err != nil {
			log.Printf("[agent] 写入 AI 审计日志失败: %v", err)
		}
	}()
}

// buildAuditDescription 构建审计描述
func (a *Agent) buildAuditDescription(skillName string, params map[string]any, result map[string]any) string {
	// 优先使用结果中的 message 字段
	if msg, ok := result["message"].(string); ok && msg != "" {
		return msg
	}
	// 使用结果中的 status 字段
	if status, ok := result["status"].(string); ok {
		return fmt.Sprintf("Skill %s 执行: %s", skillName, status)
	}
	return fmt.Sprintf("Skill %s 执行完成", skillName)
}
