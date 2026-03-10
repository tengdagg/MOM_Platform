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


### 20260216
问题1：AI 助手 SSH 认证失败
根因：凭证密码在数据库中是 AES-GCM 加密存储 的（base64 编码）。终端连接通过 GetByIDDecrypted 方法自动解密后使用，但 AI Skill 的 device.exec_command 和 device.test_connection 直接用 SELECT password FROM credentials 读取的是 加密后的密文，拿密文当密码去认证自然失败。
另外，Cisco 等设备通常使用 keyboard-interactive 认证方式而非标准 password，原来的代码只配置了 ssh.Password()。
修复 (plugins/ai/skills/device_skills.go)：
新增 decryptCredentialPassword 函数 — 复用与 credentialRepo 相同的 AES-GCM 解密逻辑和密钥
新增 getDeviceCredential 函数 — 统一查询设备凭证并自动解密密码
新增 buildDeviceSSHConfig 函数 — 统一构建 SSH 配置，同时支持 ssh.Password() 和 ssh.KeyboardInteractive() 双认证方式
executeDeviceTestConnection 和 executeDeviceExecCommand 都改用这三个新函数
问题2：选中资产提示栏图标错位
根因：el-tag 默认的 slot 内容是 inline 布局，直接放 el-icon 组件 + 文本时，el-icon 是一个 block-level SVG 容器，会导致图标和文本不在同一行。之前用 :deep() 的 CSS 可能因为选择器优先级不够而没生效。
修复：在 el-tag 内部用 <span class="hint-tag-content"> 包裹 el-icon 和文本 span，给 .hint-tag-content 直接设置 display: inline-flex; align-items: center; gap: 4px，确保图标和文字水平居中对齐。这样不依赖 :deep() 穿透，直接在 scoped 作用域内生效。


### 20260217
问题1 — 移动主机后分组数量没有更新

根本原因：后端 UpdateHost 没有清除分组树缓存（缓存有效期5分钟），只有 Create 和 Delete 才清缓存。所以移动主机（本质是 Update）后，前端重新请求分组树拿到的还是旧的缓存数据。

修复：在 UpdateHost 成功后也调用 invalidateGroupTreeCache() 清缓存。

问题2 — 网络设备分组树没有数量显示

根本原因：后端 GetTree 只统计了 hosts 表的数量（HostCount），没有统计 network_devices 表。而前端虽然有 data.deviceCount 字段但加了 v-if 条件导致为0时不显示。

修复（4个文件）：

问题1 — 折叠/展开不生效

根本原因：Hosts.vue 里有两棵 el-tree（主机列表视图的分组树 + 终端视图的分组树），它们共用了同一个 ref="groupTreeRef"。因为 v-show 不会卸载组件，两棵树同时存在于 DOM 中，Vue 的 ref 最终指向的是终端视图那棵树，而不是你在主机列表视图看到的那棵。所以点击按钮操作的是"错误的树"。

修复：

终端视图的树改为 ref="terminalTreeRef"（独立 ref）
toggleExpandAll 根据 activeView 选择正确的树引用
问题2 — 分组类型混淆

根本原因：从主机管理分组树创建分组时，payload 里没有传 category 字段，后端默认设为 "all"（通用类型），所以主机和网络设备的分组树都能看到。

修复：

Hosts.vue 的 handleGroupSubmit 添加 category: 'host'
NetworkDevices.vue 的 handleGroupSubmit 改为 category: 'network'（之前误写成了 type）
注意：后端设计中 category: 'all' 的分组（通用分组）会同时出现在主机和网络设备的树中，这是正常行为。在业务分组子菜单中创建的通用分组仍会在两边显示。
quickActions 新增了 3 个：

查看域名监控状态 — 对应新的域名监控能力
列出所有网络设备 — 对应网络设备查询
有哪些可用的凭证？ — 利用新的辅助查询功能
quickCommands 新增了 7 个：

指令	说明	对应 Skill
/添加主机	引导式添加新主机	host.manage
/网络设备	查看网络设备状态	device.list
/添加设备	引导式添加网络设备	device.manage
/域名监控	查看域名和SSL状态	monitor.domain_status
/添加域名	添加新域名监控	monitor.domain_manage
/任务历史	查看任务执行记录	task.history
同时 /巡检 的 prompt 也更新为包含网络设备的巡检。

凭证管理的「认证方式」和「适用类别」下拉筛选已修复



### 20260219
自定义 Skill 热加载修复完成，编译通过 ✅

改了 3 个文件：

文件	改动
agent.go	新增 ToolRegistry.Unregister() 方法
skill_engine.go	新增 CustomScriptSkill.SetDefinition() 方法
skill_handler.go	上传/更新后立即 Register，删除时立即 Unregister
修复后的行为：

操作	修复前	修复后
上传新 Skill	❌ 需重启	✅ 立即可用
更新已有 Skill	❌ 需重启	✅ 立即生效
删除 Skill	⚠️ Registry 残留	✅ 立即移除

### 20260219
k8s_helm_mamage 原来直接返回 "模拟成功消息" 的逻辑解决，
再检查其他 Skill 有哪些是模拟不是真正的操作处理掉


### 20260224
1、终端 Windows 虚拟键盘已改成悬浮可拖拽
2、修复一些功能异常问题



### 20260226
后端：

plugins/ai/biz/models.go — AIModelConfig 新增 MaxToolCalls 字段（默认 10）
plugins/ai/server/model_handler.go — Create/Update 支持新字段
plugins/ai/biz/agent.go — Run/RunStream 接受可配置的 maxToolCalls 参数 + context 取消支持
plugins/ai/server/chat_handler.go — 传递 maxToolCalls，新增 stop 消息处理 + context.WithCancel 管理
前端：

web/src/views/ai/AIModelConfig.vue — 弹窗新增"最大调用次数"设置 + 卡片展示
web/src/views/ai/AIChat.vue — 发送中时按钮变为红色停止按钮（脉冲动画），点击发送 {type:"stop"}


### 20260227

文件	改动
host_skills.go	exec_command: 黑名单 6→10、超时可配 max 300s、输出 64KB 截断
host_skills.go	file_manage: 新增 read/write/backup 三个操作 + 路径安全检查
task_skills.go	execute: 超时可配 + 输出截断（与 exec_command 对齐）
agent.go	systemPrompt 追加「复杂运维操作指导」6 条原则
3 个 SKILL.md	文档从 46/45/60 行扩展到 120+/110+/100 行
效果示例： 当你对 AI 说"帮我在 web-01 上安装 nginx"，AI 现在会：

rpm -q nginx（检查是否已安装）
yum install -y nginx（timeout=120 安装）
systemctl start nginx && systemctl enable nginx（启动+开机自启）
systemctl status nginx（验证状态）
当你说"帮我修改 nginx 配置"，AI 会：

file_manage(read) 查看当前配置
file_manage(backup) 备份为 .bak.时间戳
file_manage(write) 写入新内容
exec_command 执行 nginx -t && systemctl reload nginx


### 20260303

#### 1. Dashboard 模型调用次数统计改为真实数据
**问题**：原来"模型调用次数"从操作日志 `sys_operation_log` 模糊匹配关键字（ai、model、chat 等），匹配范围过广导致数据不准确（实际没调用模型却显示 53 次）。

**修复**：
- 后端新增 `GET /api/v1/plugins/ai/stats/model-calls` 接口，直接从 `ai_chat_messages` 表统计 `role='assistant'` 的消息数（每条 = 一次真实模型调用），通过 `ai_chat_sessions.model_id` 关联 `ai_model_configs.name` 获取模型名称
- 返回数据：按天+模型分组的 `daily` 数组、`todayTotal`、`total`
- 前端 Dashboard 图表改为按模型分组显示多条折线，每个模型独立颜色，支持图例切换
- 移除旧的 `getOperationLogList` 依赖和 `isModelCallLog` 模糊匹配逻辑

**涉及文件**：
- `plugins/ai/server/handler.go` — 新增 `GetModelCallStats` 方法
- `plugins/ai/server/router.go` — 注册 `/stats/model-calls` 路由
- `web/src/api/ai.ts` — 新增 `getModelCallStats` API
- `web/src/views/Dashboard.vue` — 重写模型调用图表渲染逻辑

#### 2. 任务中心模板管理按钮修复
**问题**：新建旁的 Setting、FullScreen 按钮无事件绑定点击无反应；操作列只有编辑无删除；删除函数未调用 API。

**修复**：
- 移除无功能的 Setting、FullScreen 按钮，保留 Refresh 并添加 tooltip
- 操作列改为与主机管理一致的图标按钮样式（link 类型 + action-btn/action-edit/action-delete 样式类，28x28 方形，hover 缩放变色）
- 启用删除按钮，`handleDelete` 补全 `await deleteJobTemplate(row.id)` 调用
- 新增 `deleteJobTemplate` API 导入

**涉及文件**：
- `web/src/views/task/Templates.vue` — 模板/样式/逻辑全面修复

#### 3. 用户管理本地用户标签显示
**问题**：LDAP 用户有橙色 LDAP 标签，本地用户没有标签；LDAP 标签尺寸偏大。

**修复**：
- 本地用户新增灰色 `info` 类型 "本地" 标签（`effect="plain"`）
- 统一使用 `.source-tag` 样式：高度 18px、字体 11px、内边距 `0 6px`，比默认 `size="small"` 更紧凑

**涉及文件**：
- `web/src/views/system/Users.vue` — 模板新增 `v-else` 标签 + 样式

#### 4. 用户禁用/启用状态更新失败（GORM 零值陷阱）
**问题**：编辑用户后将状态改为"禁用"（`status=0`），提示更新成功但数据库仍为启用状态。

**根因**：GORM 的 `Updates(struct)` 方法默认跳过零值字段，`int` 类型的 `0` 被视为零值，`status=0` 永远不会被写入。

**修复**：将 `Update` 方法从 `Omit("created_at").Updates(user)` 改为使用 `Select` 显式指定要更新的字段列表（`real_name, email, phone, avatar, status, department_id, bio`），确保零值也能正确写入。

**涉及文件**：
- `internal/data/rbac/user.go` — `Update` 方法重写

#### 5. LDAP 用户编辑后来源变成本地用户
**问题**：编辑 LDAP 用户（如禁用再启用）后，用户来源从 LDAP 变成本地。

**根因**：上述修复中 `Select` 列表包含了 `"source"` 字段，但前端 `userForm` 没有 `source` 字段，提交时 `Source` 为空字符串，被强制写入数据库覆盖了原有的 `"ldap"` 值。

**修复**：从 `Select` 列表中移除 `"source"`。用户来源只在创建用户或 LDAP 同步时设置，不应通过编辑表单修改。

**涉及文件**：
- `internal/data/rbac/user.go` — `Update` 方法移除 source 字段

### 20260303-2
AI Skills & Agent 代码审阅修复（安全、稳定性、代码质量）

#### 1. Shell 命令注入修复（严重）
**问题**：`host.file_manage` 技能中 `filePath` 和 `content` 参数未做 shell 转义，直接拼接到远程命令中，恶意输入可执行任意命令。

**修复**：
- 新增 `shellQuote()` 函数，对所有文件路径进行 shell 安全引用
- `list`/`read`/`download`/`backup` 操作的路径全部使用 `shellQuote()` 包裹
- `write` 操作从 heredoc 改为 base64 编码传输，杜绝内容注入
- 新增路径遍历检查（禁止 `..`）和写入大小限制（最大 1MB）

**涉及文件**：`plugins/ai/skills/host_skills.go`

#### 2. SQL 运算符优先级 Bug（严重）
**问题**：`analysis.security_audit` 中 SQL 条件 `AND action LIKE '%critical%' OR action LIKE '%high%'` 缺少括号，导致 OR 条件脱离 AND 约束，查询结果不准确。

**修复**：加括号 → `AND (action LIKE '%critical%' OR action LIKE '%high%')`

**涉及文件**：`plugins/ai/skills/analysis_skills.go`

#### 3. SkillContext.Ctx 未赋值（严重）
**问题**：`executeTool()` 创建 `SkillContext` 时未传入 `Ctx`，导致请求取消无法传播到 Skill 执行层。

**修复**：`executeTool()` 增加 `ctx context.Context` 参数，创建 SkillContext 时正确传入 `Ctx: ctx`。

**涉及文件**：`plugins/ai/biz/agent.go`

#### 4. K8s PV 容量方法错误（严重）
**问题**：`getPV()` 中使用 `StorageEphemeral()` 获取 PV 容量，该方法返回的是临时存储而非持久卷容量，结果始终为 0。

**修复**：改为 `pv.Spec.Capacity[v1.ResourceStorage]` 正确获取存储容量。

**涉及文件**：`plugins/ai/skills/k8s_kubectl_skill.go`

#### 5. CronJob Suspend 空指针 Panic（严重）
**问题**：`getCronJobs()` 中直接 `*cj.Spec.Suspend` 解引用，当 Suspend 字段为 nil 时引发 panic。

**修复**：先判空再解引用，nil 时默认为 false。

**涉及文件**：`plugins/ai/skills/k8s_kubectl_skill.go`

#### 6. 加密密钥去重
**问题**：`device_skills.go` 中重复定义了 `credEncryptionKey` 和 `decryptCredentialPassword()`，与 `service_helper.go` 功能完全一致。

**修复**：删除 `device_skills.go` 中的重复代码，统一复用 `service_helper.go` 的 `decryptCredential()` 函数。

**涉及文件**：`plugins/ai/skills/device_skills.go`

#### 7. 流式 Tool Call Index 处理
**问题**：流式响应中多个并行工具调用时，未按 API 返回的 `index` 字段分发参数片段，导致参数可能混淆。

**修复**：ToolCall 结构体增加 `Index *int` 字段，流式处理中按 index 正确分发参数到对应 tool call。

**涉及文件**：`plugins/ai/biz/agent.go`、`plugins/ai/biz/model_adapter.go`

#### 8. 重复 message_end 事件
**问题**：`Run()` 方法中 defer 和正常退出路径各发一次 `message_end`，前端可能收到重复的结束信号。

**修复**：使用 `messageSent` 标志防止重复发送。

**涉及文件**：`plugins/ai/biz/agent.go`

#### 9. host_ids / device_ids 覆盖 ip 查询结果
**问题**：`host.exec_command` 和 `device.exec_command` 中同时提供 ip 和 host_ids/device_ids 时，后者查询结果会覆盖前者。

**修复**：改为合并（append）而非覆盖。

**涉及文件**：`plugins/ai/skills/host_skills.go`、`plugins/ai/skills/device_skills.go`

#### 10. GORM baseQ 复用导致查询污染
**问题**：`monitor.alert_summary` 中共享 `baseQ` 变量，后续查询会被前一个 `Count()` 调用污染。

**修复**：每次查询独立创建 query，不复用变量。

**涉及文件**：`plugins/ai/skills/monitor_skills.go`

#### 11. 未检查的 DB 错误
**问题**：删除域名监控时关联告警配置的 `Exec` 未检查错误。

**修复**：增加错误检查和返回。

**涉及文件**：`plugins/ai/skills/monitor_skills.go`

#### 12. 禁用 Skill 未反注册
**问题**：`LoadCustomSkills()` 只注册启用的自定义 Skill，不会反注册已禁用的，禁用操作不会生效直到重启。

**修复**：加载时同时查询已禁用的自定义 Skill 并从 ToolRegistry 中移除。

**涉及文件**：`plugins/ai/biz/skill_engine.go`

#### 13. SKILL.md 参数定义对齐
**问题**：部分 SKILL.md 的参数列表与 Go 实现不一致。

**修复**：
- `monitor.alert_config/SKILL.md` — 重写 action 类型和参数列表
- `device.exec_command/SKILL.md` — 移除不存在的 timeout 参数

#### 14. analysis_skills.go 函数签名优化
**问题**：`generateCapacityRecommendations()` 使用匿名结构体参数，可读性差。

**修复**：提取为具名类型 `UsageStats`，与 `executeCapacityPlan` 复用同一类型。

---

### 20260310

#### AI Agent 确认机制重构与多轮工具调用修复（19 个文件，+1319/-284 行）

##### 1. 高风险操作确认 — 同一轮 Skill 调用不再跳过确认（严重）
**问题**：模型一次性产出多个 tool call 时（如查磁盘+执行分区+创建PV），只要第一个返回 `pending_confirmation`，代码仅打标记但不停止，后续 Skill 继续执行。导致用户看到"待确认"但实际已经跑完了后面的命令。

**修复**（`plugins/ai/biz/agent.go` — `Run` 和 `RunStream`）：
- 工具执行循环中，一旦某个 Skill 返回 `pending_confirmation`，立即 `break` 终止后续 Skill 执行
- 同时标记 `forceTextOnly=true`，下一轮 ReAct 迭代不传工具定义，强制模型只生成确认提示文本

##### 2. 确认回复直接执行原命令 — 不再让模型重新猜（严重）
**问题**：用户回复"确认"/"执行"后，系统把决定权交回模型。模型可能：(1) 不执行原命令而是换一条新命令再问确认 (2) 在失败命令和修复命令之间来回循环 (3) 直接编故事假装执行了。

**修复**（`plugins/ai/biz/agent.go` — 新增 `replayPendingAction` 机制）：
- 新增 `PendingToolAction` 结构体，保存待确认操作的完整上下文（工具名、原始参数、风险等级、警告信息）
- 新增 `getLastPendingAction(sessionID)` — 从 DB 最近助手消息的 `tool_calls` JSON 中提取最后一条 `pending_confirmation` 记录
- 新增 `replayPendingAction()` — 用户确认时，直接复用原始参数 + 注入 `confirmed=true` 执行，不再走模型
- 新增 `isAffirmativeConfirmation()` — 匹配"确认/执行/继续/好的/可以/ok/yes"等肯定词
- 新增 `isNegativeConfirmation()` — 匹配"取消/不要/不用/停止/算了/no"等否定词
- 新增 `injectConfirmedParam()` — 安全地向原始参数 JSON 注入 `confirmed=true`
- 取消操作也由后端直接处理，不再让模型生成取消文本

##### 3. 历史消息工具调用上下文还原（严重）
**问题**：`buildMessages` 构建历史上下文时只传了 `role` 和 `content`，完全丢弃了 `ToolCalls` 和 `tool` 角色消息。模型在下一轮对话中看不到之前的工具调用记录，导致：(1) 不知道之前通过工具做了什么 (2) 多步骤操作时"编故事"假装执行了后续命令 (3) 失败命令下一轮重试时不知道上次失败的具体错误。

**修复**（`plugins/ai/biz/agent.go` — `buildMessages`）：
- 当助手消息包含 `tool_calls` JSON 时，解析为标准 `ToolCall` 对象，还原为 `assistant(tool_calls=...) + tool(result=...)` 消息对
- 工具名自动通过 `SanitizeToolName` 清洗为 LLM 兼容格式
- 每个工具结果作为独立的 `tool` 角色消息，携带正确的 `tool_call_id`
- 模型现在能看到完整的工具调用历史链，多步骤操作不再丢失上下文

##### 4. 工具调用记录状态持久化修正
**问题**：`allToolCallRecords` 中的 `status` 字段原先用 `hasError` 布尔值简单映射为 "success"/"error"，`pending_confirmation` 状态被错误记录为 "success"。

**修复**：
- 统一使用 `resultStatus` 变量，优先取工具返回的 `status` 字段原始值
- `pending_confirmation` 状态正确持久化到数据库，UI 和历史回溯都能准确识别

##### 5. Skill 动作级风险推断 — `tool_call_start` 事件增强
**改进**（`plugins/ai/biz/agent.go`）：
- 新增 `inferToolRiskLevel(name, argsJSON, fallback)` — 根据工具参数动态推断风险等级（如 `lsblk` → low，`rm -rf` → critical）
- 新增 `inferToolRiskMode(name)` — 返回 `static`（固定风险）或 `dynamic`（按参数变化）
- 新增 `inferToolRiskHint(name, argsJSON, fallbackLevel)` — 生成人类可读的风险说明
- `AgentEvent` 结构体新增 `RiskMode` 和 `RiskHint` 字段
- `tool_call_start` 和 `tool_call_result` 事件均携带动态风险信息

##### 6. DeepSeek DSML 标记过滤与泄漏工具调用恢复
**改进**（`plugins/ai/biz/agent.go`）：
- 新增 `dsmlPattern` / `dsmlInvokePattern` / `dsmlParamPattern` 正则，过滤 DeepSeek 内部 `<｜DSML｜>` 标记
- 新增 `parseLeakedToolCalls()` — 恢复 `<tool_call>` 格式的泄漏调用
- 新增 `parseLeakedDSMLToolCalls()` — 恢复 DSML 格式的泄漏调用
- 新增 `parseAnyLeakedToolCalls()` — 统一入口，同时处理两种格式
- 流式输出增加 `dsmlPendingBuf` 缓冲机制，防止 `<` 字符提前泄漏到前端
- `forceTextOnly` 模式下如果模型输出纯 DSML（无有效文本），自动生成兜底确认提示
- 新增 `<think>` 标签过滤，支持 DeepSeek R1 深度思考内容的正确显示

##### 7. Skill 风险分类优化 — 低风险命令免确认
**改进**（6 个 Skill 实现文件 + 9 个 SKILL.md）：

| 文件 | 改动 |
|------|------|
| `host_skills.go` | `exec_command`: 扩展安全命令识别（`fdisk -l`、`parted print`、`docker ps/images/logs` 等）；`collect`: 移除确认，直接执行；`file_manage`: `download` 免确认；`manage`: `list_credentials`/`list_groups` 返回 `effectiveRiskLevel: low` |
| `device_skills.go` | 新增 `isSafeDeviceReadCommand` 白名单（`show`/`display`/`ping`/`traceroute` 等）；`test_connection`: 移除确认直接执行；`manage`: 列表操作免确认 |
| `task_skills.go` | `execute`: 安全命令免确认，返回动态风险等级；`ansible`: `list` 操作免确认 |
| `k8s_kubectl_skill.go` | 新增 `applyRisk` 辅助函数，所有子操作统一返回 `effectiveRiskLevel` |
| `k8s_skills.go` | `diagnose`/`log_query`: 返回 `effectiveRiskLevel: low`；`helm_manage`: `list`/`status` 免确认 |
| `monitor_skills.go` | `alert_config`: `list` 返回 `effectiveRiskLevel: low` |
| 9 个 SKILL.md | 更新风险说明、安全策略、参数定义，与后端逻辑对齐 |

##### 8. AI Chat 前端工具卡片增强
**改进**（`web/src/views/ai/AIChat.vue`）：
- 工具卡片新增"动态风险"标签（`riskMode === 'dynamic'` 时显示）
- 状态标签新增"待确认"样式（橙色闪烁边框）
- `pending_confirmation` 状态的工具卡片自动展开，显示操作详情和风险说明
- 新增 `parseToolResultStatus`、`getToolStatusType`、`getToolStatusLabel` 辅助函数

##### 9. Skill 管理 API 风险元数据
**改进**（`plugins/ai/server/skill_handler.go`）：
- 新增 `buildSkillRiskMeta` — 根据 Skill 名称和参数模式判断 `riskMode`（static/dynamic）和 `riskHint`
- `ListSkills` 接口返回值增加 `riskMode` 和 `riskHint` 字段

##### 涉及文件清单
| 文件 | 类型 |
|------|------|
| `plugins/ai/biz/agent.go` | 核心改动：确认机制重构、历史还原、风险推断、DSML 过滤 |
| `plugins/ai/server/skill_handler.go` | Skill 列表 API 风险元数据 |
| `plugins/ai/skills/host_skills.go` | 主机 Skill 风险分类 |
| `plugins/ai/skills/device_skills.go` | 网络设备 Skill 风险分类 |
| `plugins/ai/skills/task_skills.go` | 任务 Skill 风险分类 |
| `plugins/ai/skills/k8s_kubectl_skill.go` | K8s kubectl Skill 风险分类 |
| `plugins/ai/skills/k8s_skills.go` | K8s 诊断/日志/Helm Skill 风险分类 |
| `plugins/ai/skills/monitor_skills.go` | 监控告警 Skill 风险分类 |
| `plugins/ai/skills/host.exec_command/SKILL.md` | 文档对齐 |
| `plugins/ai/skills/host.collect/SKILL.md` | 文档对齐 |
| `plugins/ai/skills/host.file_manage/SKILL.md` | 文档对齐 |
| `plugins/ai/skills/device.exec_command/SKILL.md` | 文档对齐 |
| `plugins/ai/skills/device.test_connection/SKILL.md` | 文档对齐 |
| `plugins/ai/skills/k8s.kubectl/SKILL.md` | 文档对齐 |
| `plugins/ai/skills/k8s.helm_manage/SKILL.md` | 文档对齐 |
| `plugins/ai/skills/task.execute/SKILL.md` | 文档对齐 |
| `plugins/ai/skills/task.ansible/SKILL.md` | 文档对齐 |
| `web/src/views/ai/AIChat.vue` | 前端工具卡片增强 |
| `web/src/views/ai/AISkills.vue` | Skill 管理页风险元数据展示 |