package biz

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ContextBuilder 上下文构建器
// 注入用户信息、权限范围、平台概况到系统提示词中
type ContextBuilder struct {
	db *gorm.DB
}

// NewContextBuilder 创建上下文构建器
func NewContextBuilder(db *gorm.DB) *ContextBuilder {
	return &ContextBuilder{db: db}
}

// BuildSystemContext 构建系统上下文信息
func (b *ContextBuilder) BuildSystemContext(userID uint, username string) string {
	var parts []string

	// 用户信息
	parts = append(parts, fmt.Sprintf("当前用户: %s (ID: %d)", username, userID))
	parts = append(parts, fmt.Sprintf("当前时间: %s", time.Now().Format("2006-01-02 15:04:05")))

	// 用户角色
	var roles []string
	b.db.Table("sys_role").
		Select("sys_role.name").
		Joins("JOIN sys_user_role ON sys_role.id = sys_user_role.role_id").
		Where("sys_user_role.user_id = ?", userID).
		Pluck("name", &roles)
	if len(roles) > 0 {
		parts = append(parts, fmt.Sprintf("用户角色: %s", strings.Join(roles, ", ")))
	}

	// 平台概况
	var hostCount, deviceCount, clusterCount int64
	b.db.Table("hosts").Where("deleted_at IS NULL").Count(&hostCount)
	b.db.Table("network_devices").Where("deleted_at IS NULL").Count(&deviceCount)
	b.db.Table("k8s_clusters").Count(&clusterCount)
	parts = append(parts, fmt.Sprintf("平台概况: 管理 %d 台主机, %d 台网络设备, %d 个 K8s 集群", hostCount, deviceCount, clusterCount))

	return strings.Join(parts, "\n")
}

// ConversationTemplate 对话模板
type ConversationTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Prompt      string `json:"prompt"`
	Category    string `json:"category"`
}

// GetDefaultTemplates 获取默认对话模板
func GetDefaultTemplates() []ConversationTemplate {
	return []ConversationTemplate{
		{
			ID:          "daily_inspection",
			Name:        "每日巡检",
			Description: "全面检查基础设施运行状态",
			Icon:        "📋",
			Prompt:      "请对平台进行全面的每日巡检，包括：\n1. 主机在线状态和资源使用情况\n2. 网络设备在线状态和连接情况\n3. K8s 集群健康状态\n4. 域名监控状态\n5. 安全风险检查\n6. 今日操作日志异常\n请以报告格式输出，标注需要关注的问题。",
			Category:    "运维",
		},
		{
			ID:          "weekly_report",
			Name:        "周报生成",
			Description: "生成本周运维周报",
			Icon:        "📊",
			Prompt:      "请帮我生成本周的运维周报，包含以下内容：\n1. 基础设施概况（主机/网络设备/集群/域名统计）\n2. 本周重要操作记录\n3. 告警汇总\n4. 资源使用趋势\n5. 安全态势\n6. 下周计划建议",
			Category:    "报告",
		},
		{
			ID:          "troubleshoot",
			Name:        "故障排查",
			Description: "协助排查系统问题",
			Icon:        "🔍",
			Prompt:      "我遇到了一些系统问题，请帮我排查：\n1. 先检查所有主机状态，是否有离线的\n2. 检查网络设备连接状态\n3. 检查 K8s 集群健康状态\n4. 查看最近的告警\n5. 分析最近的异常操作日志\n请根据发现的问题给出排查建议。",
			Category:    "运维",
		},
		{
			ID:          "security_check",
			Name:        "安全检查",
			Description: "全面的安全态势分析",
			Icon:        "🔒",
			Prompt:      "请进行一次全面的安全检查：\n1. 登录失败记录分析（是否有暴力破解）\n2. 异常操作行为检测\n3. SSL 证书到期检查\n4. 离线主机排查\n5. K8s 集群安全状态\n请标注风险等级和处理建议。",
			Category:    "安全",
		},
		{
			ID:          "capacity_plan",
			Name:        "容量规划",
			Description: "资源使用分析与扩容建议",
			Icon:        "📈",
			Prompt:      "请分析当前的资源使用情况并给出容量规划建议：\n1. CPU/内存/磁盘使用率分析\n2. 高负载主机清单\n3. 资源使用趋势\n4. 扩容建议和优先级\n5. 成本优化建议",
			Category:    "分析",
		},
	}
}
