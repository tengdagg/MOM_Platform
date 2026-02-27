# 开源推荐 | MOM Platform —— 插件化云原生运维管理平台，让运维更简单

## 前言

作为运维工程师，你是否遇到过这些痛点：

- 🤯 资产分散在 Excel、CMDB、各种云控制台，没有统一管理界面
- 😫 K8s 多集群管理需要频繁切换 kubeconfig，效率低下
- 🔑 SSH 密钥散落各处，每次登录都要找半天
- 📊 老板要运营周报，手动从各系统拼凑数据
- 🤖 想用 AI 辅助运维，但市面方案要么太贵要么不落地

如果你也有这些困扰，那 **MOM Platform** 可能正是你需要的。

> **MOM Platform**（Multi-platform Operations Manager，多元运维管理平台）是一个开源的插件化运维管理平台，采用 Go + Vue 3 前后端分离架构，支持多集群 K8s 管理、资产管理、多云账号、远程连接、AI 智能运维等功能。

**项目地址：** [https://gitee.com/monkey_dat/mom_platform](https://gitee.com/monkey_dat/mom_platform)

---

## 一、核心亮点

### 🧩 插件化架构 —— 按需加载，灵活扩展

MOM Platform 最大的设计特色是**插件化架构**。Kubernetes 管理、任务中心、监控中心、AI 助手等核心功能都以「插件」形式提供，前后端联动，支持**一键安装/卸载**。

不需要 K8s 管理？直接禁用插件即可，不会增加系统负担。需要新功能？按照插件开发规范开发一个，注册即可使用。

<!-- 建议插图：插件管理页面截图 -->

### 🤖 AI 智能运维助手 —— 自然语言驱动运维

这是我认为 MOM 最有竞争力的功能。内置的 AI 助手不是简单的聊天机器人，而是基于 **Agent + Skills** 架构的智能运维引擎：

- **ReAct 推理循环**：AI 自主决策调用合适的 Skill 完成任务
- **36 个内置 Skills**：覆盖主机管理、网络设备、K8s 操作、任务执行、监控告警、审计分析、云账号、综合报告 8 大领域
- **多模型支持**：OpenAI / DeepSeek / 通义千问 / 豆包 / Google Gemini / Ollama 本地模型
- **高风险操作两步确认**：扩缩容、远程命令执行等操作必须用户确认后才执行
- **自定义 Skill 扩展**：上传 SKILL.md 规范的 zip 包即可扩展 AI 能力

**实际使用效果：**

```
用户：帮我列出所有离线的 Linux 主机
AI  ：[调用 host.list] 找到 3 台离线主机，分别是...

用户：把生产环境的 nginx Deployment 扩容到 5 个副本
AI  ：⚠️ 这是高风险操作。即将把 Deployment/nginx 的副本数调整为 5，确认执行吗？
用户：确认
AI  ：✅ 已成功将 nginx 的副本数调整为 5

用户：帮我生成本周的基础设施运营周报
AI  ：[调用 analysis.infra_report] 正在生成报告...
     📊 本周概况：在管主机 128 台，K8s 集群 3 个，告警 12 次...
```

<!-- 建议插图：AI 对话界面截图 / AI 演示视频 GIF -->

### ☸️ 多集群 Kubernetes 管理

统一管理多个 K8s 集群，功能覆盖完整的资源生命周期：

| 功能 | 支持的资源类型 |
|------|------------|
| 工作负载 | Deployment、StatefulSet、DaemonSet、Job、CronJob |
| 网络与服务 | Service、Ingress、NetworkPolicy |
| 配置与存储 | ConfigMap、Secret、PV/PVC |
| 集群管理 | 节点列表、资源监控、污点/标签、Cordon/Drain |
| 高级功能 | CRD 管理、Helm Release、Web Terminal、集群巡检 |

还支持 **Web Terminal** 终端连接，直接在浏览器里 `kubectl exec` 进容器，支持会话录制与回放。

<!-- 建议插图：K8s 集群管理页面截图 -->

### 🖥️ SSH / RDP 远程连接

- **SSH 终端**：密码 + 密钥认证，支持拖拽上传密钥文件
- **Windows RDP**：基于 Apache Guacamole，浏览器直连 Windows 远程桌面
- **文件管理**：RDP 文件上传/下载，自动清理临时文件
- **虚拟键盘**：美式键盘布局，解决特殊字符输入问题
- **全程录制**：SSH 和 RDP 会话全程录制，支持审计回放

<!-- 建议插图：SSH 终端页面截图 + RDP 远程桌面截图 -->

### ☁️ 多云账号管理

支持 **7 大主流云厂商**一键接入：

| 云厂商 | 功能 |
|-------|------|
| 阿里云 | 云主机实例查询、一键导入资产 |
| 腾讯云 | 云主机实例查询、一键导入资产 |
| 华为云 | 云主机实例查询、一键导入资产 |
| AWS | 云主机实例查询、一键导入资产 |
| 京东云 | 云主机实例查询、一键导入资产 |
| 百度云 | 云主机实例查询、一键导入资产 |
| 金山云 | 云主机实例查询、一键导入资产 |

把分散在各云控制台的主机统一导入平台管理，再也不用切换多个控制台了。

### 🔐 精细化权限控制

- **双重 RBAC**：平台级权限 + Kubernetes 级权限
- **资产级隔离**：查看、编辑、删除、SSH、RDP、文件管理 6 种权限粒度
- **友好提示**：无权限时提示「无访问权限」而不是「连接错误」

### 📋 操作审计

运维操作全程可追溯：

- 操作日志完整记录
- SSH / RDP 会话录制与回放
- 数据变更追溯
- **AI 操作审计**：AI 助手的每次 Skill 调用都记录到日志，按模块分类筛选

---

## 二、AI 助手内置技能一览

MOM 的 AI 助手内置了 **36 个 Skills**，按 8 大分类组织：

| 分类 | Skills | 能力描述 |
|------|--------|---------|
| 🖥️ 主机管理 | `host.list` `host.detail` `host.analyze` `host.exec_command` `host.file_manage` 等 | 主机查询、分析、远程命令、文件管理 |
| 🌐 网络设备 | `device.list` `device.detail` `device.exec_command` 等 | 网络设备管理、远程命令 |
| ☸️ Kubernetes | `k8s.kubectl` `k8s.scale` `k8s.restart` `k8s.diagnose` `k8s.helm_manage` 等 | 全资源操作、扩缩容、诊断、Helm |
| 📋 任务中心 | `task.execute` `task.ansible` `task.history` | Ad-hoc 任务、Ansible Playbook |
| 📡 监控告警 | `monitor.domain_status` `monitor.alert_summary` 等 | 域名监控、告警分析 |
| 🔍 审计分析 | `audit.operation_summary` `audit.login_analysis` 等 | 操作统计、登录分析 |
| ☁️ 云账号 | `cloud.list_accounts` `cloud.import_hosts` 等 | 云账号管理、主机导入 |
| 📊 综合分析 | `analysis.infra_report` `analysis.security_audit` `analysis.capacity_plan` | 运营周报、安全审计、容量规划 |

而且支持**自定义 Skill 上传**，编写一个 `SKILL.md` 文件打包成 zip 即可扩展 AI 能力，无需修改源码。

---

## 三、技术栈

| 层级 | 技术选型 |
|------|---------|
| 后端 | Go 1.21+ / Gin / GORM / client-go / WebSocket |
| 前端 | Vue 3.5+ / TypeScript / Element Plus / Vite / xterm.js |
| 数据库 | MySQL 8.0+（兼容 TiDB 分布式数据库） |
| 缓存 | Redis 6.0+ |
| 远程桌面 | Apache Guacamole 1.5+ |
| AI | OpenAI Compatible API / 多模型适配器 |

**系统架构图：**

```
                    ┌──────────────────────────────────────┐
                    │           浏览器客户端                 │
                    │    Vue 3 + Element Plus + TypeScript   │
                    └──────────────┬───────────────────────┘
                                   │ HTTP / WebSocket
                    ┌──────────────▼───────────────────────┐
                    │           Gin HTTP Server              │
                    │     JWT Auth │ RBAC │ Audit Middleware  │
                    ├────────┬─────┴─────┬─────────┬────────┤
                    │  Core  │  Plugins  │   AI    │ Asset  │
                    │ Module │  Manager  │  Agent  │ Mgr    │
                    ├────────┼───────────┼─────────┼────────┤
                    │ User   │ K8s      │ Model   │ Host   │
                    │ Role   │ Task     │ Adapter │ Cred   │
                    │ Menu   │ Monitor  │ Skills  │ Cloud  │
                    ├────────┴───────────┴─────────┴────────┤
                    │            GORM / Data Layer           │
                    └────────┬────────────────────┬─────────┘
                             │                    │
                    ┌────────▼───────┐   ┌────────▼────────┐
                    │  MySQL / TiDB  │   │  K8s API Server │
                    └────────────────┘   └─────────────────┘
```

---

## 四、快速安装

### 方式一：Docker Compose（推荐体验）

最快的方式，几分钟即可跑起来：

```bash
# 1. 克隆项目
git clone https://gitee.com/monkey_dat/mom_platform.git
cd mom_platform

# 2. 启动所有服务
docker-compose up -d

# 3. 访问系统
# 前端：http://localhost:5173
# 后端 API：http://localhost:9876
# 默认账号：admin / 123456
```

### 方式二：源码部署（适合二次开发）

**环境要求：**
- Go 1.21+
- Node.js 18+
- MySQL 8.0+
- Redis 6.0+

**步骤：**

```bash
# 1. 克隆项目
git clone https://gitee.com/monkey_dat/mom_platform.git
cd mom_platform

# 2. 初始化数据库
mysql -u root -p -e "CREATE DATABASE mom CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -u root -p mom < migrations/init.sql

# 3. 配置
cp config/config.yaml.example config/config.yaml
# 编辑 config.yaml，修改数据库连接、Redis 等配置

# 4. 启动后端
go run main.go server

# 5. 启动前端（新终端）
cd web && npm install && npm run dev

# 6. 访问 http://localhost:5173
# 默认账号：admin / 123456
```

### 方式三：Kubernetes 部署（生产环境）

项目提供了 Helm Chart，适合生产环境高可用部署：

```bash
helm install mom ./charts -n mom-system --create-namespace
```

> ⚠️ 生产环境请务必修改默认密码！

---

## 五、功能列表总览

### 基础功能

| 功能模块 | 描述 |
|---------|------|
| 用户管理 | 用户 CRUD、LDAP 集成、密码重置、状态管理 |
| 角色管理 | 角色定义、菜单权限分配 |
| 部门管理 | 组织架构管理、部门层级 |
| 岗位管理 | 岗位定义、用户绑定 |
| 菜单管理 | 动态菜单、支持插件菜单编辑 |
| 凭据管理 | SSH 密码/密钥统一管理 |
| 资产管理 | 主机分组、标签、批量导入导出 |
| 操作审计 | 操作日志、登录日志、AI 操作审计 |

### 插件功能

| 插件 | 核心能力 |
|-----|---------|
| Kubernetes 管理 | 多集群、工作负载、网络、存储、CRD、Helm、Web Terminal、集群巡检 |
| 任务中心 | 脚本执行、模板管理、文件分发、执行历史 |
| 监控中心 | 域名监控（HTTP/SSL）、告警管理、多渠道通知 |
| AI 智能助手 | 多模型、36 Skills、自定义扩展、工具可视化、操作审计 |
| RDP 远程桌面 | Windows 远程连接、文件管理、虚拟键盘、会话录制 |

---

## 六、与同类项目对比

| 特性 | MOM Platform | JumpServer | 蓝鲸 | KubeSphere |
|-----|:---:|:---:|:---:|:---:|
| 插件化架构 | ✅ | ❌ | ✅ | ✅ |
| AI 运维助手 | ✅ 36 Skills | ❌ | ❌ | ❌ |
| 多集群 K8s | ✅ | ❌ | ✅ | ✅ |
| SSH 终端 | ✅ | ✅ | ✅ | ✅ |
| RDP 远程桌面 | ✅ | ✅ | ❌ | ❌ |
| 多云账号 | ✅ 7 家 | ❌ | ✅ | ❌ |
| 会话录制回放 | ✅ | ✅ | ❌ | ❌ |
| 集群巡检 | ✅ | ❌ | ❌ | ❌ |
| LDAP 集成 | ✅ | ✅ | ✅ | ✅ |
| 开源协议 | MIT | GPL v3 | 自有 | Apache 2.0 |
| 技术栈 | Go + Vue 3 | Python + Vue | Python + Vue | Go + Vue |
| 部署难度 | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |

---

## 七、未来规划

| 功能 | 描述 |
|------|------|
| 数据库远程终端 | MySQL、Oracle、PostgreSQL 远程终端，AI 助手支持 |
| CI/CD 集成 | 对接 GitLab、Jenkins CI，ArgoCD 持续部署 |

---

## 总结

MOM Platform 是一个**功能完整、设计现代、开箱即用**的运维管理平台。它最大的特色是：

1. **插件化架构**：按需加载，不臃肿
2. **AI 智能助手**：36 个内置技能，自然语言驱动运维
3. **全栈覆盖**：从主机资产到 K8s 集群，从 SSH 终端到 RDP 桌面
4. **安全可控**：双重 RBAC + 操作审计 + AI 高风险确认
5. **开源友好**：MIT 协议，可自由使用和二次开发

如果你正在寻找一个现代化的运维管理平台，不妨试试 MOM Platform。

**项目地址：** [https://gitee.com/monkey_dat/mom_platform](https://gitee.com/monkey_dat/mom_platform)

**觉得不错的话，请给个 ⭐ Star 支持一下！**

---

> **作者简介：** 一个热爱运维自动化的工程师，专注于云原生、DevOps 和 AIOps 领域的开源实践。
