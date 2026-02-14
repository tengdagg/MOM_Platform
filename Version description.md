### 20260214
完成的所有改动
1. AI 操作审计日志
文件: plugins/ai/biz/agent.go
在 executeTool 中新增审计日志记录逻辑：每次 Skill 执行完毕后异步写入 sys_operation_log 表
记录内容包括：操作人（用户名+AI 标记）、Skill 名称、参数、结果描述、风险等级、耗时
智能模块分类：根据 Skill 前缀自动归类（AI-Kubernetes、AI-主机管理、AI-任务中心 等）
pending_confirmation 状态的调用不记录（只记录实际执行的操作）
审计日志中的 method 为 SKILL，user_agent 为 MOM-AI-Agent/1.0，便于筛选 AI 操作
2. 内置 Skills 启用/禁用功能修复
后端 (plugins/ai/server/skill_handler.go, router.go):
新增 ToggleBuiltinSkill 接口 (PUT /toggle-builtin)：通过名称切换内置 Skill 状态
首次禁用时自动在数据库创建记录（默认 is_enabled=false）
后续 toggle 直接更新数据库
ListSkills 改进：已有 DB 记录的内置 Skill 保留数据库中的启用状态
Agent 运行时使用 GetToolDefinitionsFiltered(db) 自动排除被禁用的 Skills
新增 GetSkillStats 统计接口 (GET /stats)
前端 (web/src/views/ai/AISkills.vue, web/src/api/ai.ts):
移除了 :disabled="!skill.id" 限制，所有 Skill（包括内置的）都可以 toggle
内置 Skill 通过 toggleBuiltinSkill(name) API 调用
禁用后的 Skill 卡片自动变半透明 + 显示"已禁用"标签
3. Skills 管理页面数量统计
前端 (AISkills.vue):
新增彩色统计卡片行：
Skills 总数（紫色渐变）
内置已启用/总数（绿色渐变）
自定义已启用/总数（粉色渐变）
按分类统计
上传/删除/切换状态后自动刷新统计
4. 上传替换内置 Skills 说明
上传说明区增加了"替换内置 Skill"提示：上传同名包会自动覆盖
原有的上传逻辑已支持同名覆盖（uploadFromSKILLMD 中 Where("name = ?") 检查）
5. 增强所有内置 Skills
审计类 Skills (audit_skills.go):
修复 audit.operation_summary: 表名 sys_operation_logs → sys_operation_log，新增按模块/操作类型过滤，AI 操作统计，错误操作统计，最近操作记录
修复 audit.login_analysis: 表名修复，新增异常行为自动检测（暴力破解、IP 攻击），按时间段分布，成功/失败分离统计
修复 audit.session_summary: 表名 terminal_audit_logs → ssh_terminal_sessions，新增按主机统计，最近会话记录
修复 audit.data_changes: 表名 audit_logs → sys_operation_log，正确排除查询操作只看变更，新增按用户统计变更，AI 操作变更统计
分析类 Skills (analysis_skills.go):
analysis.infra_report: 表名修复，新增资源使用率平均值、按云厂商分组统计、SSL 即将到期统计、今日 AI 操作数、登录失败统计、用户概况、AI Skills 概况
analysis.security_audit: 表名修复，每个风险项新增 details 字段（具体列表），新增高负载主机检测、AI 高风险操作检测
analysis.capacity_plan: 细分 CPU/内存/磁盘高使用率主机列表，新增按分组统计资源使用，更详细的扩容建议
任务类 Skills (task_skills.go):
task.execute: 从"仅返回提示"改为真正执行远程命令（通过 SSH），支持按 IP/分组/ID 列表选择目标主机，安全命令黑名单扩展
task.history: 新增关键词搜索，按类型统计，成功/失败计数
task.ansible: 新增 list 操作查看可用模板，确认后创建任务执行记录
监控类 Skills (monitor_skills.go):
monitor.domain_status: 新增慢域名检测、平均响应时间统计、状态码信息
monitor.alert_summary: 新增按严重程度统计、未处理告警计数、最近告警记录
monitor.alert_config: create 操作确认后真正创建告警规则到数据库，新增 enable/disable 操作，delete 操作确认后真正删除
云账号类 Skills (cloud_skills.go):
cloud.list_accounts: 新增状态过滤、按厂商分组统计
cloud.list_instances: 新增区域/状态/关键词过滤、更丰富的字段输出
cloud.import_hosts: 支持通过账号名称查找、确认后创建导入任务记录
主机类 Skills (host_skills.go):
host.list: 新增标签搜索、描述搜索、排序支持（CPU/内存/磁盘/名称/IP）、Linux/Windows 分类计数
host.detail: 新增公网 IP、CPU 型号、云平台信息、结构化资源数据、状态文字、分组名称
host.analyze: 新增按分组统计资源使用率、整体资源统计、健康评估等级（healthy/warning/critical）