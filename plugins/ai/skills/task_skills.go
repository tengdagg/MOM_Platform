package skills

import (
	"fmt"
	"strings"
	"time"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
)

// RegisterTaskSkills 注册任务中心 Skills
func RegisterTaskSkills(registry *biz.ToolRegistry) {
	registry.Register(MustLoadBuiltinSkill("task.history", executeTaskHistory))
	registry.Register(MustLoadBuiltinSkill("task.execute", executeTaskExecute))
	registry.Register(MustLoadBuiltinSkill("task.ansible", executeTaskAnsible))
}

// executeTaskHistory 查询任务执行历史
func executeTaskHistory(ctx biz.SkillContext) (any, error) {
	days := 7
	if d, ok := ctx.Params["days"].(float64); ok && d > 0 {
		days = int(d)
	}
	statusFilter, _ := ctx.Params["status"].(string)
	keyword, _ := ctx.Params["keyword"].(string)
	limit := 20
	if l, ok := ctx.Params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	since := time.Now().AddDate(0, 0, -days)

	type TaskRecord struct {
		ID        uint      `json:"id"`
		Name      string    `json:"name"`
		Type      string    `json:"type"`
		Status    string    `json:"status"`
		Username  string    `json:"username"`
		CreatedAt time.Time `json:"createdAt"`
		Duration  int       `json:"duration"`
		Output    string    `json:"output,omitempty"`
	}

	query := ctx.DB.Table("task_execution_records").
		Where("created_at >= ?", since)

	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}
	if keyword != "" {
		query = query.Where("name LIKE ? OR type LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var records []TaskRecord
	if err := query.Order("created_at DESC").Limit(limit).Find(&records).Error; err != nil {
		// 表可能不存在，返回空结果
		return map[string]any{
			"period":  fmt.Sprintf("最近 %d 天", days),
			"records": []any{},
			"total":   0,
			"message": "暂无任务执行记录",
		}, nil
	}

	// 统计
	var totalCount int64
	ctx.DB.Table("task_execution_records").Where("created_at >= ?", since).Count(&totalCount)
	var failedCount int64
	ctx.DB.Table("task_execution_records").Where("created_at >= ? AND status = 'failed'", since).Count(&failedCount)
	var successCount int64
	ctx.DB.Table("task_execution_records").Where("created_at >= ? AND status = 'success'", since).Count(&successCount)

	// 按类型统计
	type TypeStat struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}
	var typeStats []TypeStat
	ctx.DB.Table("task_execution_records").
		Select("type, COUNT(*) as count").
		Where("created_at >= ?", since).
		Group("type").
		Order("count DESC").
		Find(&typeStats)

	return map[string]any{
		"period":  fmt.Sprintf("最近 %d 天", days),
		"records": records,
		"total":   totalCount,
		"success": successCount,
		"failed":  failedCount,
		"byType":  typeStats,
	}, nil
}

// executeTaskExecute 执行 Ad-hoc 任务（在指定主机上执行命令）
func executeTaskExecute(ctx biz.SkillContext) (any, error) {
	command, _ := ctx.Params["command"].(string)
	if command == "" {
		return nil, fmt.Errorf("请指定要执行的命令")
	}

	hostIP, _ := ctx.Params["host_ip"].(string)
	groupName, _ := ctx.Params["group_name"].(string)

	// 安全检查：拒绝危险命令
	dangerousPatterns := []string{"rm -rf /", "mkfs", "dd if=", ":(){ :|:& };:", "> /dev/sd", "chmod -R 777 /", "shutdown", "reboot", "init 0", "init 6"}
	cmdLower := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmdLower, pattern) {
			return nil, fmt.Errorf("安全检查未通过：命令包含危险操作 [%s]，已被拒绝", pattern)
		}
	}

	// 查找目标主机
	type SimpleHost struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		IP   string `json:"ip"`
	}
	var hosts []SimpleHost

	query := ctx.DB.Table("hosts").Select("id, name, ip").Where("deleted_at IS NULL AND status = 1")
	if hostIP != "" {
		query = query.Where("ip = ?", hostIP)
	}
	if groupName != "" {
		query = query.Joins("LEFT JOIN asset_groups ON hosts.group_id = asset_groups.id").
			Where("asset_groups.name LIKE ?", "%"+groupName+"%")
	}
	if hostIDs, ok := ctx.Params["host_ids"].([]any); ok && len(hostIDs) > 0 {
		ids := make([]uint, 0, len(hostIDs))
		for _, id := range hostIDs {
			if v, ok := id.(float64); ok {
				ids = append(ids, uint(v))
			}
		}
		query = query.Where("id IN ?", ids)
	}
	query.Limit(50).Find(&hosts)

	if len(hosts) == 0 {
		return nil, fmt.Errorf("未找到目标主机，请指定主机 IP、分组名称或 ID 列表")
	}

	// 未确认 → 返回待确认信息
	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"message":    fmt.Sprintf("命令 [%s] 将在 %d 台主机上执行", command, len(hosts)),
			"command":    command,
			"hostCount":  len(hosts),
			"hosts":      hosts,
			"status":     "pending_confirmation",
			"warning":    "⚠️ 远程命令执行是高风险操作，请确认命令内容和目标主机无误后再执行",
		}, nil
	}

	// 已确认 → 真正执行
	type ExecResult struct {
		Host   string `json:"host"`
		IP     string `json:"ip"`
		Output string `json:"output"`
		Error  string `json:"error,omitempty"`
	}
	var results []ExecResult
	successCount := 0

	for _, h := range hosts {
		client, _, err := CreateSSHClient(ctx.DB, h.ID)
		if err != nil {
			results = append(results, ExecResult{Host: h.Name, IP: h.IP, Error: err.Error()})
			continue
		}
		output, err := client.ExecuteWithTimeout(command, 30*time.Second)
		client.Close()
		if err != nil {
			results = append(results, ExecResult{Host: h.Name, IP: h.IP, Output: output, Error: err.Error()})
		} else {
			results = append(results, ExecResult{Host: h.Name, IP: h.IP, Output: output})
			successCount++
		}
	}

	return map[string]any{
		"status":       "success",
		"message":      fmt.Sprintf("✅ 命令已在 %d/%d 台主机上执行完成", successCount, len(hosts)),
		"command":      command,
		"results":      results,
		"successCount": successCount,
		"totalCount":   len(hosts),
	}, nil
}

// executeTaskAnsible 执行 Ansible Playbook
func executeTaskAnsible(ctx biz.SkillContext) (any, error) {
	playbookName, _ := ctx.Params["playbook_name"].(string)
	action, _ := ctx.Params["action"].(string)
	if action == "" {
		action = "run"
	}

	// list 操作：列出可用的 Ansible 任务模板
	if action == "list" {
		type AnsibleInfo struct {
			ID          uint      `json:"id"`
			Name        string    `json:"name"`
			Description string    `json:"description"`
			Status      string    `json:"status"`
			CreatedAt   time.Time `json:"createdAt"`
		}
		var templates []AnsibleInfo
		query := ctx.DB.Table("ansible_tasks")
		if playbookName != "" {
			query = query.Where("name LIKE ?", "%"+playbookName+"%")
		}
		query.Order("created_at DESC").Limit(20).Find(&templates)

		return map[string]any{
			"templates": templates,
			"total":     len(templates),
			"message":   fmt.Sprintf("找到 %d 个 Ansible 任务模板", len(templates)),
		}, nil
	}

	// run 操作：执行 Playbook
	if playbookName == "" {
		return nil, fmt.Errorf("请指定 Playbook 名称")
	}

	taskName, _ := ctx.Params["task_name"].(string)
	if taskName == "" {
		taskName = "AI-" + playbookName
	}
	groupName, _ := ctx.Params["group_name"].(string)
	tags, _ := ctx.Params["tags"].(string)

	// 查询可用的 Ansible 任务模板
	type AnsibleTask struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Playbook string `json:"playbook"`
	}
	var templates []AnsibleTask
	ctx.DB.Table("ansible_tasks").Select("id, name, playbook").Where("name LIKE ?", "%"+playbookName+"%").Limit(10).Find(&templates)

	if !isConfirmed(ctx.Params) {
		return map[string]any{
			"action":       "ansible_execute",
			"taskName":     taskName,
			"playbookName": playbookName,
			"groupName":    groupName,
			"tags":         tags,
			"templates":    templates,
			"status":       "pending_confirmation",
			"warning":      fmt.Sprintf("⚠️ 高风险操作: 将执行 Ansible Playbook [%s]，目标分组: %s，请确认", playbookName, groupName),
		}, nil
	}

	// 已确认 → 记录执行请求到任务队列（实际执行由任务中心调度）
	if len(templates) > 0 {
		// 创建任务执行记录
		ctx.DB.Exec(`INSERT INTO task_execution_records (name, type, status, username, created_at, updated_at) VALUES (?, 'ansible', 'pending', ?, NOW(), NOW())`,
			taskName, fmt.Sprintf("AI(%d)", ctx.UserID))

		return map[string]any{
			"status":       "success",
			"message":      fmt.Sprintf("✅ Ansible Playbook [%s] 执行任务已提交到任务队列", playbookName),
			"taskName":     taskName,
			"playbookName": playbookName,
			"matchedTemplates": templates,
		}, nil
	}

	return map[string]any{
		"status":  "warning",
		"message": fmt.Sprintf("未找到匹配的 Ansible 任务模板 [%s]，请检查名称是否正确", playbookName),
	}, nil
}
