# iom 数据库结构参考

## 表统计

| 类别 | 表数 | 说明 |
|-----|------|------|
| RBAC系统 | 11 | 用户、角色、部门、菜单、职位及关联表 |
| 审计日志 | 3 | 操作日志、登录日志、数据变更日志 |
| 资产管理 | 5 | 资产组、主机、凭证、云账户、权限 |
| 任务管理 | 3 | 任务模板、任务执行、Ansible任务 |
| Kubernetes | 5 | 集群、kubeconfig、角色绑定、巡检、终端 |
| 监控告警 | 6 | 域名监控、告警配置、渠道、接收人、日志 |
| **总计** | **33** | **所有应用表** |

## 数据模型关系图

```
┌─────────────────────────────────────────┐
│          RBAC 权限管理系统               │
├─────────────────────────────────────────┤
│  sys_user (用户)                        │
│  ├─→ sys_department (部门)             │
│  ├─→ sys_user_role (M2M)              │
│  │   └─→ sys_role (角色)               │
│  │       └─→ sys_role_menu (M2M)      │
│  │           └─→ sys_menu (菜单)       │
│  └─→ sys_user_position (M2M)          │
│      └─→ sys_position (职位)           │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│          审计日志系统                    │
├─────────────────────────────────────────┤
│  sys_operation_log (操作日志)            │
│  sys_login_log (登录日志)                │
│  sys_data_log (数据变更日志)             │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│          资产管理系统                    │
├─────────────────────────────────────────┤
│  asset_group (资产组)                   │
│  ├─→ hosts (主机)                      │
│  │   ├─→ credentials (凭证)             │
│  │   └─→ cloud_accounts (云账户)       │
│  └─→ sys_role_asset_permission        │
│      └─→ sys_role (角色)               │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│          Task 任务管理系统               │
├─────────────────────────────────────────┤
│  job_templates (任务模板)               │
│  ├─→ job_tasks (执行任务)               │
│  └─→ ansible_tasks (Ansible任务)      │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│          Kubernetes 管理系统             │
├─────────────────────────────────────────┤
│  k8s_clusters (集群)                    │
│  ├─→ k8s_user_kube_configs (用户配置)  │
│  ├─→ k8s_user_role_bindings (角色绑定) │
│  ├─→ k8s_cluster_inspections (巡检)    │
│  └─→ k8s_terminal_sessions (终端)      │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│          Monitor 监控告警系统            │
├─────────────────────────────────────────┤
│  domain_monitors (域名监控)             │
│  ├─→ alert_configs (告警配置)          │
│  ├─→ alert_channels (告警渠道)         │
│  │   └─→ alert_receiver_channels       │
│  ├─→ alert_receivers (告警接收人)      │
│  └─→ alert_logs (告警日志)             │
└─────────────────────────────────────────┘
```

## 字段类型对照表

### 常用字段类型

| 字段类型 | 大小 | 用途 | 示例 |
|---------|------|------|------|
| `bigint unsigned` | 8字节 | 主键、ID字段 | user_id, id |
| `int` | 4字节 | 计数、阈值、排序 | count, sort |
| `tinyint` | 1字节 | 布尔值、状态 | status, enabled |
| `varchar(n)` | n+2字节 | 短文本 | username, email |
| `text` | 65K | 中文本 | description |
| `longtext` | 4GB | 大文本 | JSON数据、日志 |
| `json` | 变长 | JSON对象 | variables, config |
| `datetime` | 8字节 | 时间戳 | created_at |

### 常见字段含义

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `id` | bigint unsigned | 主键，自增 |
| `created_at` | datetime | 创建时间，默认当前时间 |
| `updated_at` | datetime | 更新时间，自动更新 |
| `deleted_at` | datetime | 软删除标记 |
| `status` | tinyint | 状态：1启用 0禁用 |
| `sort` | int | 排序序号，升序 |
| `enabled` | tinyint | 启用标志：1启用 0禁用 |

## 性能优化策略

### 索引策略

1. **唯一索引** (UNIQUE KEY)
   - 用于唯一字段（username、code等）
   - 包含soft delete字段确保逻辑隔离

2. **普通索引** (KEY)
   - 外键字段（FK）
   - 过滤条件字段（status、type等）
   - 排序字段（sort、created_at等）
   - 时间范围查询字段

3. **复合索引**
   - 多条件查询（role_id, asset_group_id）
   - (user_id, deleted_at)

### 查询优化建议

```sql
-- ✅ 好的做法
SELECT * FROM sys_user WHERE status = 1 AND deleted_at IS NULL;
SELECT * FROM sys_operation_log WHERE created_at > NOW() - INTERVAL 7 DAY;

-- ❌ 避免的做法
SELECT * FROM sys_user WHERE YEAR(created_at) = 2024;  -- 避免函数
SELECT * FROM sys_user WHERE username LIKE '%admin%';   -- 避免前缀通配符
```

### 表分区建议

对于增长较快的日志表，建议按日期分区：

```sql
ALTER TABLE sys_operation_log PARTITION BY RANGE (YEAR(created_at) * 100 + MONTH(created_at)) (
    PARTITION p202401 VALUES LESS THAN (202402),
    PARTITION p202402 VALUES LESS THAN (202403),
    PARTITION pmax VALUES LESS THAN MAXVALUE
);
```

## 数据清理建议

### 定期清理旧日志

```sql
-- 保留最近30天的操作日志
DELETE FROM sys_operation_log WHERE created_at < DATE_SUB(NOW(), INTERVAL 30 DAY);

-- 保留最近90天的登录日志
DELETE FROM sys_login_log WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY);

-- 保留最近1年的告警日志
DELETE FROM alert_logs WHERE created_at < DATE_SUB(NOW(), INTERVAL 365 DAY);
```

### 创建定期清理任务

```sql
-- 创建事件定期清理（每周执行）
CREATE EVENT cleanup_old_logs ON SCHEDULE EVERY 1 WEEK DO
BEGIN
    DELETE FROM sys_operation_log WHERE created_at < DATE_SUB(NOW(), INTERVAL 30 DAY);
    DELETE FROM sys_login_log WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY);
    DELETE FROM alert_logs WHERE created_at < DATE_SUB(NOW(), INTERVAL 365 DAY);
END;

-- 启用事件
ALTER EVENT cleanup_old_logs ENABLE;
```

## 权限配置

### 创建应用用户

```sql
-- 创建应用用户（不使用root）
CREATE USER 'iom'@'localhost' IDENTIFIED BY 'strong_password';
GRANT ALL PRIVILEGES ON iom.* TO 'iom'@'localhost';
FLUSH PRIVILEGES;

-- 用于生产环境的只读副本用户
CREATE USER 'iom_readonly'@'localhost' IDENTIFIED BY 'readonly_password';
GRANT SELECT ON iom.* TO 'iom_readonly'@'localhost';
FLUSH PRIVILEGES;
```

## 常见SQL查询

### 查询用户及其角色

```sql
SELECT u.id, u.username, u.real_name, GROUP_CONCAT(r.name) AS roles
FROM sys_user u
LEFT JOIN sys_user_role ur ON u.id = ur.user_id
LEFT JOIN sys_role r ON ur.role_id = r.id
WHERE u.deleted_at IS NULL
GROUP BY u.id;
```

### 查询角色及其菜单权限

```sql
SELECT r.id, r.name, GROUP_CONCAT(m.name) AS menus
FROM sys_role r
LEFT JOIN sys_role_menu rm ON r.id = rm.role_id
LEFT JOIN sys_menu m ON rm.menu_id = m.id
WHERE r.deleted_at IS NULL
GROUP BY r.id;
```

### 统计用户操作日志

```sql
SELECT username, action, COUNT(*) as count
FROM sys_operation_log
WHERE created_at > DATE_SUB(NOW(), INTERVAL 7 DAY)
  AND deleted_at IS NULL
GROUP BY username, action
ORDER BY count DESC;
```

### 查询主机分组统计

```sql
SELECT ag.name, COUNT(h.id) as host_count
FROM asset_group ag
LEFT JOIN hosts h ON ag.id = h.group_id AND h.deleted_at IS NULL
WHERE ag.deleted_at IS NULL
GROUP BY ag.id, ag.name;
```

## 数据备份

### 使用 mysqldump

```bash
# 备份所有数据
mysqldump -u root -p iom > iom_full.sql

# 仅备份表结构
mysqldump -u root -p -d iom > iom_schema.sql

# 压缩备份
mysqldump -u root -p iom | gzip > iom_$(date +%Y%m%d).sql.gz

# 恢复备份
mysql -u root -p iom < iom_full.sql
```

### 自动备份脚本

```bash
#!/bin/bash
BACKUP_DIR="/path/to/backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/iom_$DATE.sql.gz"

mysqldump -u iom -p'password' iom | gzip > "$BACKUP_FILE"

# 保留最近30天的备份
find "$BACKUP_DIR" -name "iom_*.sql.gz" -mtime +30 -delete
```

## 故障排查

### 检查表状态

```sql
-- 检查表的完整性
CHECK TABLE sys_user;

-- 修复损坏的表
REPAIR TABLE sys_user;

-- 优化表空间
OPTIMIZE TABLE sys_user;
```

### 查看表大小

```sql
-- 查看单个表大小
SELECT
    table_name,
    ROUND(((data_length + index_length) / 1024 / 1024), 2) AS size_mb
FROM information_schema.TABLES
WHERE table_schema = 'iom'
ORDER BY size_mb DESC;
```

### 查看慢查询

```sql
-- 启用慢查询日志
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 2;

-- 查看慢查询
SELECT * FROM mysql.slow_log;
```

## 相关文档

- [数据库初始化指南](README.md)
- [iom 项目主文档](../../README.md)
- [MySQL 官方文档](https://dev.mysql.com/doc/)
