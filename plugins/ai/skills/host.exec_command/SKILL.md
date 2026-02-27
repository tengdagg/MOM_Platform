---
name: host.exec_command
description: 在指定主机上远程执行 Shell 命令，支持单台或多台主机批量执行
category: host
riskLevel: critical
scriptType: builtin
parameters:
  type: object
  properties:
    host_ids:
      type: array
      items:
        type: integer
      description: 目标主机 ID 列表
    ip:
      type: string
      description: 目标主机 IP（与 host_ids 二选一）
    command:
      type: string
      description: 要执行的 Shell 命令
    timeout:
      type: integer
      description: 执行超时时间（秒），默认 30，最大 300
      default: 30
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - command
---

# 远程命令执行

在指定主机上远程执行 Shell 命令。支持单台或多台主机批量执行。

## ⚠️ 危险操作

此 Skill 风险等级为 **critical**，需要两步确认：
1. 首次调用返回待执行的主机列表和命令详情
2. 用户确认后带 `confirmed=true` 再次调用真正执行

## 使用场景

- 用户说"在所有 Web 服务器上检查 nginx 状态"
- 用户说"在 192.168.1.10 上执行 df -h"
- 用户说"查看所有主机的内存使用"
- 用户要求安装软件、查看服务状态等

## 常见运维操作指南

### 安装软件
分步执行，先检查再安装：
1. 检查是否已安装：`rpm -q nginx` 或 `dpkg -l nginx`
2. 安装软件：`yum install -y nginx` 或 `apt install -y nginx`（需要较长超时，建议 timeout=120）
3. 启动服务：`systemctl start nginx`
4. 验证状态：`systemctl status nginx`

### 磁盘扩容
分步执行，先查看再操作：
1. 查看磁盘分区：`lsblk`
2. 查看磁盘使用率：`df -h`
3. 扩展分区（需根据实际情况选择命令）：
   - `growpart /dev/vda 1`（扩展分区）
   - `resize2fs /dev/vda1`（ext4 文件系统）
   - `xfs_growfs /`（xfs 文件系统）
4. 验证结果：`df -h`

### 修改配置
分步执行，先备份再修改：
1. 查看当前配置：`cat /etc/nginx/nginx.conf`
2. 备份配置文件：`cp /etc/nginx/nginx.conf /etc/nginx/nginx.conf.bak`
3. 使用 sed 修改（注意：不要使用 vi/nano 等交互式编辑器）：
   - `sed -i 's/old_value/new_value/g' /etc/nginx/nginx.conf`
4. 验证修改：`cat /etc/nginx/nginx.conf`
5. 重新加载服务：`systemctl reload nginx`

### 服务管理
- 查看状态：`systemctl status nginx`
- 启动服务：`systemctl start nginx`
- 停止服务：`systemctl stop nginx`
- 重启服务：`systemctl restart nginx`
- 设置开机自启：`systemctl enable nginx`

## 危险命令黑名单

以下命令模式会被自动拦截，无法执行：
- `rm -rf /` — 递归删除根目录
- `mkfs` — 格式化磁盘
- `dd if=` — 磁盘级写入
- `:(){ :|:& };:` — Fork 炸弹
- `> /dev/sd` — 覆盖磁盘设备
- `chmod -R 777 /` — 对根目录设置不安全权限
- `shutdown` — 关机
- `reboot` — 重启
- `init 0` — 关机
- `init 6` — 重启

## 参数说明

- `command`（必需）：要执行的 Shell 命令
- `host_ids`：目标主机 ID 列表，批量执行时使用
- `ip`：目标主机 IP（与 host_ids 二选一）
- `timeout`：超时时间（秒），默认 30 秒，最大 300 秒。安装软件等耗时操作建议设为 120-300

## 返回数据

- `status`: 执行状态（pending_confirmation / success）
- `command`: 执行的命令
- `results`: 每台主机的执行结果数组
  - `host`: 主机名称
  - `ip`: 主机 IP
  - `output`: 命令输出内容
  - `error`: 错误信息（如有）
- `successCount`: 成功执行的主机数
- `totalCount`: 总目标主机数

## 多步操作最佳实践

对于安装软件、修改配置等复杂操作，应遵循以下原则：
1. **分步执行**：将复杂操作拆分为多个安全的小步骤，每步执行一个命令
2. **先查后改**：修改前先查看当前状态
3. **备份优先**：修改配置文件前先备份
4. **验证结果**：每个关键步骤后验证执行结果
5. **合理超时**：安装软件等耗时操作设置较长的 timeout

## 注意事项

- 每次调用只执行一条命令，不支持管道或多命令组合（除非用 && 连接）
- 批量执行时按顺序逐台执行，不是并行
- 输出内容超过 64KB 时会被截断
- 安装软件等操作建议增大 timeout 参数（默认 30s 可能不够）
