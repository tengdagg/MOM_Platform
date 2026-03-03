# mom 数据库迁移指南

<p align="center">
  <img src="https://img.shields.io/badge/Database-MySQL%208.0+-4479A1?style=flat&logo=mysql" alt="MySQL">
  <img src="https://img.shields.io/badge/Migrate-golang--migrate%20v4-00ADD8?style=flat&logo=go" alt="golang-migrate">
  <img src="https://img.shields.io/badge/Charset-utf8mb4-green?style=flat" alt="Charset">
</p>

---

## 概述

MOM Platform 使用 [golang-migrate](https://github.com/golang-migrate/migrate) 进行数据库版本化迁移管理。所有 SQL 迁移文件通过 Go 的 `embed.FS` 嵌入到二进制中，应用启动时**自动执行**，无需手动导入 SQL。

### 迁移文件位置

```
internal/migration/
├── migration.go                        # 迁移执行逻辑 + baseline 处理
└── migrations/
    ├── 000001_init_schema.up.sql       # 初始建表 + 种子数据
    ├── 000001_init_schema.down.sql     # 回滚：删除所有表
    ├── 000002_xxx.up.sql               # 后续增量迁移...
    └── 000002_xxx.down.sql
```

---

## 工作原理

### 启动时自动迁移

应用在 `cmd/server/server.go` 中调用 `migration.Run(dsn)`，执行流程：

1. 打开原生 `database/sql` 连接
2. **Baseline 检测**：如果检测到已有业务表（`sys_user`）但没有 `schema_migrations` 表，说明是从旧版本升级，自动标记 v1 已完成（跳过初始建表）
3. 使用 `embed.FS` 读取 SQL 迁移文件
4. 调用 `m.Up()` 执行所有未执行的迁移
5. 打印当前数据库版本

### 版本追踪

golang-migrate 通过 `schema_migrations` 表追踪迁移状态：

| 字段 | 类型 | 说明 |
|:-----|:-----|:-----|
| `version` | bigint | 迁移版本号 |
| `dirty` | boolean | 是否处于脏状态（迁移中断） |

---

## 快速开始

### 全新部署

```bash
# 1. 创建空数据库
mysql -u root -p -e "CREATE DATABASE mom CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 2. 配置 config.yaml 中的数据库连接

# 3. 启动应用（自动执行所有迁移）
go run main.go server
# 输出: [迁移] 数据库版本: 1, dirty: false
```

无需手动导入任何 SQL 文件，启动即完成建表和初始数据填充。

### 从旧版本升级

如果数据库是之前通过 `init.sql` 手动导入创建的：

1. 启动新版应用
2. 迁移模块自动检测到 `sys_user` 表存在但 `schema_migrations` 不存在
3. 自动执行 baseline：创建 `schema_migrations` 表并标记 v1 完成
4. 只执行 v1 之后的增量迁移

**无需任何手动操作**，平滑升级。

### 验证迁移状态

```sql
USE mom;

-- 查看当前迁移版本
SELECT * FROM schema_migrations;

-- 查看所有表
SHOW TABLES;
```

---

## 添加新迁移

### 命名规范

```
{序号}_{描述}.up.sql      # 正向迁移
{序号}_{描述}.down.sql    # 回滚迁移
```

序号为 6 位数字，从当前最大值递增：

```
000001_init_schema.up.sql
000001_init_schema.down.sql
000002_add_ai_tables.up.sql
000002_add_ai_tables.down.sql
000003_add_network_device_table.up.sql
000003_add_network_device_table.down.sql
```

### 编写迁移

**up.sql** — 正向迁移（创建/修改）：

```sql
-- 000002_add_ai_tables.up.sql
CREATE TABLE IF NOT EXISTS `ai_chat_sessions` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `user_id` bigint unsigned NOT NULL,
    `title` varchar(255) DEFAULT '',
    `created_at` datetime DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` datetime DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**down.sql** — 回滚迁移（撤销 up.sql 的变更）：

```sql
-- 000002_add_ai_tables.down.sql
DROP TABLE IF EXISTS `ai_chat_sessions`;
```

### 注意事项

- 每个迁移文件只包含**一条**完整的 DDL 语句，或用分号分隔的多条语句
- `up.sql` 和 `down.sql` 必须成对存在
- 迁移文件一旦发布到生产环境，**不能修改**，只能添加新的迁移
- 使用 `IF NOT EXISTS` / `IF EXISTS` 增强幂等性

---

## 表结构概览

### 总计：33+ 张表

### 系统核心表 (11 个)

| 表名 | 说明 | 主要字段 |
|:-----|:-----|:---------|
| `sys_user` | 用户表 | username, password, real_name, email, status |
| `sys_role` | 角色表 | name, code, description, status |
| `sys_department` | 部门表 | name, code, parent_id, dept_type |
| `sys_menu` | 菜单表 | name, code, type, parent_id, path, component |
| `sys_position` | 职位表 | post_name, post_code, post_status |
| `sys_user_role` | 用户-角色关联 | user_id, role_id |
| `sys_role_menu` | 角色-菜单关联 | role_id, menu_id |
| `sys_user_position` | 用户-职位关联 | user_id, position_id |
| `sys_operation_log` | 操作审计日志 | user_id, module, action, method, path |
| `sys_login_log` | 登录审计日志 | user_id, login_type, login_status, ip |
| `sys_data_log` | 数据变更日志 | user_id, table_name, action, old_data, new_data |

### 资产管理表 (6 个)

| 表名 | 说明 | 主要字段 |
|:-----|:-----|:---------|
| `asset_group` | 资产分组 | name, code, parent_id, description |
| `credentials` | 访问凭证 | name, type, username, password, private_key |
| `hosts` | 主机/服务器 | name, ip, port, ssh_user, credential_id, status |
| `cloud_accounts` | 云账户 | name, provider, access_key, secret_key |
| `sys_role_asset_permission` | 角色资产权限 | role_id, asset_group_id, host_ids, permissions |
| `ssh_terminal_sessions` | SSH 终端会话 | host_id, user_id, recording_path, duration |

### 任务管理表 (3 个)

| 表名 | 说明 | 主要字段 |
|:-----|:-----|:---------|
| `job_templates` | 任务模板 | name, code, content, category, platform, timeout |
| `job_tasks` | 任务执行记录 | name, template_id, task_type, status, result |
| `ansible_tasks` | Ansible 任务 | name, playbook_content, inventory, status |

### Kubernetes 表 (5 个)

| 表名 | 说明 | 主要字段 |
|:-----|:-----|:---------|
| `k8s_clusters` | Kubernetes 集群 | name, api_endpoint, kube_config, version, status |
| `k8s_user_kube_configs` | 用户 kubeconfig | cluster_id, user_id, service_account, namespace |
| `k8s_user_role_bindings` | 用户角色绑定 | cluster_id, user_id, role_name, role_type |
| `k8s_cluster_inspections` | 集群巡检记录 | cluster_id, status, score, report_data |
| `k8s_terminal_sessions` | K8S 终端会话 | cluster_id, pod_name, container_name, recording_path |

### 监控告警表 (6 个)

| 表名 | 说明 | 主要字段 |
|:-----|:-----|:---------|
| `domain_monitors` | 域名监控 | domain, status, ssl_valid, ssl_expiry, response_time |
| `alert_configs` | 告警配置 | name, alert_type, enabled, threshold |
| `alert_channels` | 告警通道 | name, channel_type, enabled, config |
| `alert_receivers` | 告警接收人 | name, email, phone, wechat_id, dingtalk_id |
| `alert_receiver_channels` | 接收人-通道关联 | receiver_id, channel_id, config |
| `alert_logs` | 告警日志 | alert_type, domain, status, message, sent_at |

### 插件管理表 (1 个)

| 表名 | 说明 | 主要字段 |
|:-----|:-----|:---------|
| `plugin_states` | 插件状态 | name, enabled |

---

## 初始化数据

初始迁移 `000001_init_schema` 包含以下种子数据：

### 默认账号

| 用户名 | 密码 | 角色 | 邮箱 |
|:-------|:-----|:-----|:-----|
| admin | 123456 | 管理员 | admin@mom.io |

> **重要**: 生产环境请立即修改默认密码！

### 默认角色

| ID | 名称 | 编码 |
|:---|:-----|:-----|
| 1 | 管理员 | admin |
| 2 | 普通用户 | user |

### 默认插件状态

| 插件 | 状态 |
|:-----|:-----|
| kubernetes | 启用 |
| monitor | 启用 |
| task | 启用 |

---

## 数据库配置

确保 `config/config.yaml` 中的数据库配置正确：

```yaml
database:
  driver: mysql
  host: 127.0.0.1
  port: 3306
  database: mom
  username: root
  password: "your-password"
  charset: utf8mb4
  max_idle_conns: 10
  max_open_conns: 100
```

---

## 常见操作

### 重置数据库

```bash
# 删除并重建数据库，重启应用自动迁移
mysql -u root -p -e "DROP DATABASE IF EXISTS mom; CREATE DATABASE mom CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
go run main.go server
```

### 备份数据库

```bash
# 完整备份
mysqldump -u root -p mom > mom_backup_$(date +%Y%m%d).sql

# 压缩备份
mysqldump -u root -p mom | gzip > mom_$(date +%Y%m%d).sql.gz
```

### 恢复数据库

```bash
mysql -u root -p mom < mom_backup.sql
```

### 重置管理员密码

```sql
-- 密码: 123456 (bcrypt 加密)
UPDATE sys_user
SET password='$2a$10$RLkgoedTSa0dYj3ujbXMcunSED3c6GLvfdKYsmpz0l0YFZbVrSBqW'
WHERE username='admin';
```

---

## 故障排查

### Q: 迁移报 dirty 状态？

**A:** 说明上一次迁移中断，需要手动修复：

```sql
-- 查看当前状态
SELECT * FROM schema_migrations;

-- 如果确认数据完整，强制标记为干净
UPDATE schema_migrations SET dirty = false;
```

### Q: 字符集问题导致乱码？

**A:** 确保数据库和连接都使用 `utf8mb4`：

```sql
ALTER DATABASE mom CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### Q: 从旧版本升级后迁移未生效？

**A:** 检查 baseline 是否正确执行：

```sql
-- 应该看到 version=1, dirty=false
SELECT * FROM schema_migrations;

-- 如果表不存在，可能 baseline 失败，手动执行：
CREATE TABLE IF NOT EXISTS `schema_migrations` (`version` bigint NOT NULL PRIMARY KEY, `dirty` boolean NOT NULL);
INSERT INTO `schema_migrations` (`version`, `dirty`) VALUES (1, false);
-- 然后重启应用
```

### Q: 如何查看表结构？

```sql
SHOW TABLES;
DESCRIBE sys_user;
SHOW CREATE TABLE sys_user;
```

---

## 相关文档

- [mom 主文档](../README.md)
- [数据库结构参考](DATABASE.md)
- [部署指南](../docs/deployment.md)
