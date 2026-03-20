---
name: host.exec_command
description: 在指定主机上执行 Shell 命令；普通命令默认单次 SSH 执行，依赖上下文的命令会复用当前对话中的交互式 shell 会话
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
      description: 执行超时时间（秒），默认 60，最大 600
      default: 60
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - command
---

# 远程命令执行

在指定主机上远程执行 Shell 命令。支持单台或多台主机批量执行。

## 风险说明

此 Skill 是**混合型 Skill**：
1. `df -h`、`lsblk`、`fdisk -l`、`parted ... print free`、`systemctl status`、`cat` 等常见查看类命令会直接执行
2. 安装软件、修改配置、启停服务、磁盘扩容、Docker 变更等高风险命令仍需要两步确认
3. 真正的破坏性命令会被安全检查直接拒绝
4. 普通单条命令默认新建一次 SSH 会话执行；当命令依赖当前目录、环境变量或 shell 状态时，会自动切换到当前对话内的交互式 shell 会话

## 使用场景

- 用户说"在所有 Web 服务器上检查 nginx 状态"
- 用户说"在 192.168.1.10 上执行 df -h"
- 用户说"查看所有主机的内存使用"
- 用户要求安装软件、查看服务状态等

## 选择规则

优先选择 `host.exec_command` 的场景：

- 用户明确指定单台主机 IP、主机 ID，或语义上就是单机执行
- 用户要在某几台明确指定的主机上执行命令，而不是"某个分组全部主机"
- 用户需要连续执行依赖 shell 上下文的命令，例如 `cd`、`source`、`export`、单独 `bash/sh`
- 用户是在做精确排障，例如查看某台机器的磁盘、进程、配置、服务状态

以下场景不要优先选它，应改用 `task.execute`：

- 用户明确说"对某个主机分组批量执行"
- 用户说"对生产环境全部主机"、"对 web 组所有机器"、"对某分组在线主机统一执行"
- 目标是任务中心式的 Ad-hoc 批量执行，而不是单机 SSH 排障

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
- `effectiveRiskLevel`: 实际动作风险等级（如 `low` / `critical`）
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

- 查看类命令会直接执行；变更类命令需要用户确认
- 普通命令默认走单次 SSH 执行，适合 `df -h`、`systemctl status`、`cat`、`grep` 等一次性命令
- 对于 `cd`、`export`、`source`、单独 `bash/sh` 等依赖 shell 上下文的命令，会自动复用当前对话中的主机交互会话
- 可用 `host.session_status` 查看当前对话的主机会话状态，必要时用 `host.close_session` 主动结束会话
- 不适合执行 `top`、`htop`、`less`、`vi`、`nano`、`tail -f`、`watch`、嵌套 `ssh/telnet/mysql/psql` 等持续交互命令
- 支持使用 `&&`、`;`、`||` 组合成一条可一次性完成的命令
- 批量执行时按顺序逐台执行，不是并行
- 输出内容超过 64KB 时会被截断
- 安装软件等操作建议增大 timeout 参数（默认 30s 可能不够）
