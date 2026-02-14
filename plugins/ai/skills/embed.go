package skills

import "embed"

// SkillFS 嵌入所有内置 Skill 的 SKILL.md 定义文件
// 每个 Skill 按照标准目录结构组织:
//
//	skill-name/
//	├── SKILL.md          (必需) YAML 前置元数据 + Markdown 指令
//	└── 打包资源           (可选)
//	    ├── scripts/      可执行代码
//	    ├── references/   上下文文档
//	    └── assets/       输出文件（模板等）
//
//go:embed host.list/SKILL.md host.detail/SKILL.md host.analyze/SKILL.md
//go:embed k8s.cluster_status/SKILL.md k8s.list_resources/SKILL.md
//go:embed audit.operation_summary/SKILL.md audit.login_analysis/SKILL.md audit.session_summary/SKILL.md
//go:embed task.history/SKILL.md task.execute/SKILL.md
//go:embed monitor.domain_status/SKILL.md monitor.alert_summary/SKILL.md
//go:embed cloud.list_accounts/SKILL.md cloud.list_instances/SKILL.md
//go:embed analysis.infra_report/SKILL.md analysis.security_audit/SKILL.md analysis.capacity_plan/SKILL.md
var SkillFS embed.FS
