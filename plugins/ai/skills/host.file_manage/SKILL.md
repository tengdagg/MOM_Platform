---
name: host.file_manage
description: 管理远程主机文件，支持查看目录、读取文件、备份文件、写入文件、下载文件操作
category: host
riskLevel: high
scriptType: builtin
parameters:
  type: object
  properties:
    host_id:
      type: integer
      description: 目标主机 ID
    ip:
      type: string
      description: 目标主机 IP（与 host_id 二选一）
    action:
      type: string
      description: "操作类型: list(列出目录) / read(读取文件内容) / backup(备份文件) / write(写入文件) / download(下载文件)"
    path:
      type: string
      description: 远程文件或目录路径
    content:
      type: string
      description: 文件内容（write 操作时必填）
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - action
    - path
---

# 远程文件管理

管理远程主机上的文件和目录，支持查看、读取、备份和写入操作。

## 使用场景

- 用户说"查看 192.168.1.10 的 /var/log/ 目录"
- 用户说"读取 /etc/nginx/nginx.conf 配置文件"
- 用户说"备份一下 sshd 配置文件"
- 用户说"修改 nginx 配置文件"
- 用户说"从服务器下载 /var/log/syslog"

## 操作说明

### list — 列出目录（低风险，无需确认）

列出指定目录的文件和子目录，返回 `ls -la` 格式的列表。

### read — 读取文件内容（低风险，无需确认）

读取指定文件的内容，限制 64KB，适合查看配置文件等文本文件。
- 适用于快速查看配置，如 `/etc/nginx/nginx.conf`、`/etc/hosts`

### backup — 备份文件（中风险，需确认）

将文件备份为 `.bak.时间戳` 格式，保留原始文件权限。
- 备份路径格式：`原路径.bak.20260227135000`
- **建议在修改任何配置文件前先执行 backup**

### write — 写入文件（高风险，需确认）

将指定内容写入指定文件路径，通过 SSH heredoc 方式实现。
- 需要指定 `content` 参数
- 确认时会展示待写入内容的预览（前 500 字符）
- ⚠️ **会覆盖原文件内容**，强烈建议先执行 `backup` 操作

### download — 下载文件（低风险，无需确认）

读取远程文件内容（限制 64KB），与 `read` 类似，直接执行即可。

## 配置文件修改最佳实践

修改配置文件的推荐流程：

1. **read** — 先读取当前配置：`action=read, path=/etc/nginx/nginx.conf`
2. **backup** — 备份原文件：`action=backup, path=/etc/nginx/nginx.conf`
3. **write** — 写入新内容：`action=write, path=/etc/nginx/nginx.conf, content=...`
4. 使用 `host.exec_command` 验证并重新加载服务

## 安全限制

### 禁止操作的系统路径

以下路径会被自动拒绝操作：
- `/boot` — 引导目录
- `/dev` — 设备文件
- `/proc` — 进程文件系统
- `/sys` — 系统文件系统
- `/run` — 运行时目录

### 文件大小限制

- 读取操作最大返回 64KB 内容，超出部分会被截断
- 写入操作建议控制文件内容在合理大小内

## 返回数据

### list 操作返回
- `status`: success
- `host`: 主机 IP
- `path`: 目录路径
- `listing`: 目录列表内容

### read 操作返回
- `status`: success
- `host`: 主机 IP
- `path`: 文件路径
- `content`: 文件内容

### backup 操作返回
- `status`: success
- `sourcePath`: 原文件路径
- `backupPath`: 备份文件路径

### write 操作返回
- `status`: success
- `host`: 主机 IP
- `path`: 写入的文件路径

## 注意事项

- 需要主机 SSH 连接权限
- read 和 list 操作为低风险，直接执行
- download、read、list 操作会直接执行
- backup、write 操作需要用户确认
- write 操作会覆盖原文件，操作前请务必备份
