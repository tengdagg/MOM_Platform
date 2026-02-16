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


### 20260214

AI 操作审计日志不显示
根因：之前 writeAuditLog 使用 go func() + 原始 SQL 执行，错误被静默吞掉（没有 error 检查和日志），导致即使写入失败也无从得知。
修复内容（4 个文件）：
plugins/ai/biz/agent.go — 重写 writeAuditLog：
定义了 aiOperationLog 结构体，精确映射到 sys_operation_log 表的每一列
改用 GORM 的 Create() 代替原始 SQL，确保列映射正确
加入错误日志：log.Printf("[agent] 写入 AI 审计日志失败: %v", err)
填充 RealName: "AI 助手"，让操作用户列显示更清晰
Action 字段改为 skill 名称 + 风险等级，例如 k8s.kubectl [critical]
internal/biz/audit/model.go — 扩展字段宽度：
Action 从 varchar(50) 改为 varchar(100)
Method 从 varchar(10) 改为 varchar(20) 以容纳 SKILL
cmd/server/server.go — 添加 ALTER TABLE 迁移确保现有数据库列宽也被更新
internal/data/audit/operation_log.go — 优化查询：
选择 "AI 助手" 模块时自动包含所有 AI-* 子模块
action 改为 LIKE 模糊匹配
web/src/views/audit/OperationLogs.vue — 前端增强：
模块筛选下拉增加 AI 助手分组（AI 助手、AI-Kubernetes、AI-主机管理、AI-任务中心、AI-监控告警、AI-云账号、AI-审计分析、AI-综合分析）
AI 模块标签以 warning（橙色）区分，便于一眼辨识
操作类型和请求方法也新增对 AI 操作的样式支持
现在 AI 操作审计日志的效果是：
操作用户	模块	操作	操作描述	请求方法	路径
AI 助手	AI-Kubernetes	k8s.kubectl [critical]	查询到 3 个节点...	SKILL

### 20260215
1. Skills 卡片宽度对齐 page header
统计卡片（stats-row）改为 flex-wrap: nowrap，每个 stat-card 使用 flex: 1 均匀铺满整行，与 page header 完全对齐
Skill 卡片网格的最小列宽从 320px 调整到 380px，更好利用页面宽度
2. AI 对话页面 - 历史会话保留时长设置
前端：
在左侧"新建对话"按钮下方新增一个 历史保留 XX 天 的点击条
点击弹出设置对话框，支持：
输入框直接设定天数（1-365）
快捷按钮选择 7/14/30/90/180/365 天
保存：保存设置
立即清理：按设定天数清理过期会话（二次确认后执行）
后端：
新增 ai_user_settings 表，自动建表
新增 3 个 API：
GET /chat/settings/retention — 获取保留天数（默认 30 天）
PUT /chat/settings/retention — 设置保留天数
POST /chat/cleanup — 手动清理过期会话
CleanupOldSessions 方法根据 updated_at 判断过期会话，同时删除关联消息
3. 云厂商 SVG 图标迁移至 CustomIcons
将阿里云、腾讯云、AWS、京东云、百度云、金山云 6 个 SVG 图标全部迁移到 web/src/components/icons/CustomIcons.vue
CloudAccounts.vue 中改为使用 <CustomIcons :name="..." /> 组件引用
云厂商选择按钮的 border-radius 已从 4px 改为 0（移除圆角）
以后替换图标只需修改 CustomIcons.vue 一个文件即可

### 20260215
LDAP 认证功能已完整实现，以下是完整总结：
整体架构
登录流程:用户输入账号密码 → 先验证本地密码 → 失败 → 检查 LDAP 是否启用 → 是 → LDAP 认证                                       ↓                                      ↓                                    成功则通过                        成功 → 自动创建/更新本地用户 → 颁发 JWT
1. 数据库改动
sys_user 表 新增 source 字段（varchar(20)，默认值 local），区分 local 本地用户和 ldap LDAP 用户
新增 sys_ldap_config 表：存储 LDAP 配置（服务器、端口、SSL、绑定 DN/密码、搜索 Base DN、用户过滤器、属性映射、默认角色等）
通过 GORM AutoMigrate 自动创建/迁移
2. 后端 - LDAP 认证服务 (internal/service/rbac/ldap.go)
功能	说明
Authenticate()	连接 LDAP → 管理员绑定 → 搜索用户 → 用户绑定验证密码 → 返回用户属性
TestConnection()	测试 LDAP 连接和绑定
SyncLDAPUsers()	手动批量同步 LDAP 用户到本地
支持 SSL/LDAPS	UseSSL 开关
3. 后端 - 登录流程改造 (internal/service/rbac/user.go)
登录时先尝试本地密码验证
本地验证失败且 LDAP 启用时，自动 fallback 到 LDAP 认证
LDAP 认证成功后：
用户不存在 → 自动创建本地用户（标记 source=ldap），自动分配默认角色
用户已存在 → 同步 LDAP 属性（姓名、邮箱、手机号）
登录日志 loginType 区分 web 和 ldap
4. 后端 API 路由
接口	方法	说明	认证
/api/v1/public/ldap/status	GET	获取 LDAP 启用状态	公开
/api/v1/ldap/config	GET	获取 LDAP 配置	需登录
/api/v1/ldap/config	PUT	保存 LDAP 配置	需登录
/api/v1/ldap/test	POST	测试 LDAP 连接	需登录
/api/v1/ldap/sync	POST	同步 LDAP 用户	需登录
/api/v1/ldap/users	GET	查询 LDAP 用户列表	需登录
5. 前端 - 系统配置页面 (SystemConfig.vue)
新增 LDAP 认证 Tab 页，包含：
基本设置：启用开关、服务器地址、端口、SSL
绑定设置：Bind DN、密码、Base DN
搜索过滤器：自定义用户搜索 filter（支持 AD 和 OpenLDAP）
属性映射：用户名、姓名、邮箱、手机号属性
权限设置：LDAP 用户默认角色
操作按钮：测试连接、保存配置、同步用户
LDAP 用户统计卡片
6. 前端 - 登录页面 (Login.vue)
LDAP 启用时显示提示信息"支持 LDAP / AD 域账号登录"
无需额外操作，用户直接用 LDAP 账号密码登录即可
7. 前端 - 用户管理页面 (Users.vue)
用户名旁显示 LDAP 标签（橙色 el-tag）
新增 来源筛选下拉框：全部 / 本地用户 / LDAP 用户
LDAP 用户和本地用户可以正常进行角色分配、权限管理，互不影响

### 20260215

All 9 tasks from the plan have been implemented. Here's a summary of all changes:
Backend (Go)
1. Model & Migration
internal/biz/asset/host.go — Added NetworkDevice struct with all fields (name, IP, brandModel, serialNumber, deviceType, protocol, port, credentialId, groupId, etc.), NetworkDeviceRequest, and ToModel() method
internal/biz/asset/repository.go — Added NetworkDeviceRepo interface with CRUD + List + GetAll methods
cmd/server/server.go — Added &assetmodel.NetworkDevice{} to AutoMigrate; added migration logic to auto-insert "网络设备" menu for existing databases
2. Data Layer
internal/data/asset/network_device.go (new) — Repository implementation with Create, Update, Delete, GetByID, List (with keyword/deviceType/protocol/groupID filtering), GetAll
3. UseCase Layer
internal/biz/asset/network_device_usecase.go (new) — Business logic including CRUD, list with credential/group enrichment, GetByIDForConnection for terminal connections
4. Service Layer (HTTP Handlers)
internal/service/asset/network_device.go (new) — REST handlers: ListNetworkDevices, CreateNetworkDevice, UpdateNetworkDevice, DeleteNetworkDevice, GetNetworkDevice, GetAllNetworkDevices, TestNetworkDeviceConnection (SSH and Telnet testing)
5. Telnet WebSocket Handler
internal/server/asset/telnet.go (new) — Full Telnet IAC protocol implementation:
NetworkTerminalManager for managing network device terminal sessions
handleTelnetSession — Raw TCP connection with IAC negotiation (NAWS, TTYP, ECHO, SGA)
handleSSHSession — SSH connection for network devices via SSH protocol
processTelnetData — Parses IAC commands, responds to negotiations, returns cleaned data
Session recording using existing AsciinemaRecorder
Window resize support via NAWS sub-negotiation
6. Routes & Wiring
internal/server/asset/http.go — Added NetworkDeviceService and NetworkTerminalManager to HTTPServer; registered routes:
GET/POST /network-devices + GET/PUT/DELETE /network-devices/:id + POST /network-devices/:id/test
GET /asset/network-terminal/:id (WebSocket for terminal)
internal/server/http.go — Updated NewAssetServices call to receive and pass network device services
Frontend (Vue.js)
7. API Layer
web/src/api/host.ts — Added: getNetworkDeviceList, getAllNetworkDevices, getNetworkDevice, createNetworkDevice, updateNetworkDevice, deleteNetworkDevice, testNetworkDeviceConnection
8. Management Page
web/src/views/asset/NetworkDevices.vue (new) — Full CRUD page:
Page header matching Hosts.vue style
Filter bar (keyword, device type, protocol, group)
Table with device type icons/colors, protocol tags, status badges, credential/group display
Create/Edit dialog with all fields, protocol-based auto-port switching
Connection test, connect-to-terminal, and delete actions
Double-click row to open terminal
9. Router & Menu
web/src/router/index.ts — Added route /asset/network-devices → NetworkDevices.vue
cmd/server/server.go — Menu seed data includes "网络设备" under "资产管理"; auto-migration inserts menu for existing databases
10. Terminal Integration
web/src/views/asset/Terminal.vue — Enhanced sidebar:
Added segmented control ("主机" / "网络设备") at top of sidebar
Network device tree with distinct icon (WiFi-style) and protocol tag (SSH/Telnet)
openNetworkDeviceTerminal + initNetworkDeviceTerminal functions connecting to /api/v1/asset/network-terminal/:id
Query parameter support (?type=network-device&deviceId=X) for direct device connection from NetworkDevices.vue



### 20260215 
后端 (新增 internal/server/asset/system_config.go)
系统配置 API：
GET /api/v1/system-config — 从 system_config 表读取所有配置项，返回 key-value 格式
PUT /api/v1/system-config — 批量保存配置，自动处理新增/更新
审计日志定时清理调度器 (StartAuditLogCleanupScheduler)：
服务启动 60 秒后执行一次，之后每 24 小时自动执行
从 system_config 表读取 logRetentionDays 配置（默认 30 天）
自动清理超过保留期的三类日志：
操作日志 (sys_operation_log)
登录日志 (sys_login_log)
数据变更日志 (sys_data_log)
同时清理超过 7 天的孤立终端录制文件
前端 (SystemConfig.vue)
loadConfig 改为调用 GET /api/v1/system-config，从后端数据库加载配置（不再使用 localStorage）
handleSave 改为调用 PUT /api/v1/system-config，将配置持久化到数据库
"日志保留天数" 输入框旁增加了提示文字："适用于操作日志、登录日志、数据变更日志，超过天数自动清理"