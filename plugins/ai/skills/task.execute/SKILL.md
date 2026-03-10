---
name: task.execute
description: 【仅用于按分组批量执行】在指定主机分组的所有在线主机上批量执行 Ad-hoc 命令。必须通过 group_name 指定目标分组。如果用户只指定了 IP 或单台主机，请改用 host.exec_command 而不是此 Skill。查看类命令可直接执行，批量变更类命令需用户确认，并自动拒绝危险命令
category: task
riskLevel: critical
scriptType: builtin
parameters:
  type: object
  properties:
    command:
      type: string
      description: 要执行的 Shell 命令
    host_ip:
      type: string
      description: 目标主机 IP
    group_name:
      type: string
      description: 目标主机分组名称（在分组内所有在线主机上执行）
    host_ids:
      type: array
      items:
        type: integer
      description: 目标主机 ID 列表
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

# 远程命令执行（任务中心）

在一台或多台指定主机上执行 Ad-hoc Shell 命令。

## 风险说明

此 Skill 是**混合型 Skill**：
1. `df -h`、`systemctl status`、`cat`、`lsblk` 等查看类命令会直接执行
2. 安装软件、修改配置、启停服务等批量变更命令需要两步确认
3. 真正的破坏性命令会被安全检查直接拒绝

## 与 `host.exec_command` 的区别

| 特性 | `task.execute` | `host.exec_command` |
|------|---------------|-------------------|
| 适用场景 | 按分组批量执行 | 按 IP/ID 精准执行 |
| 主机选择 | 支持 group_name | 不支持分组 |
| 主机状态 | 只在**在线**主机上执行 | 不限制状态 |
| 并发限制 | 最多 50 台 | 无限制 |

## 使用场景

- 用户要求在某台主机上执行命令
- 批量在某个分组的所有主机上检查服务状态
- 紧急运维操作（如重启服务、查看日志等）

## 常见运维场景

### 批量安装软件
```
command: "yum install -y nginx"
group_name: "Web 服务器"
timeout: 120
```

### 批量检查服务状态
```
command: "systemctl status nginx"
group_name: "生产环境"
```

### 批量查看磁盘使用率
```
command: "df -h"
group_name: "所有主机"
```

## 安全策略

- 查看类命令直接执行；批量变更类命令需要用户明确确认
- 自动拒绝危险命令：`rm -rf /`、`mkfs`、`dd if=`、`shutdown`、`reboot`、`init 0`、`init 6`、Fork 炸弹等
- 执行超时限制：默认 60 秒，最大 600 秒
- 操作会记录到审计日志
- 最多同时在 50 台主机上执行
- 输出超过 64KB 会被截断

## 返回数据

- `status`: 执行状态（pending_confirmation / success）
- `effectiveRiskLevel`: 实际动作风险等级（如 `low` / `critical`）
- `command`: 执行的命令
- `timeout`: 使用的超时时间
- `results`: 每台主机的执行结果数组
  - `host`: 主机名称
  - `ip`: 主机 IP
  - `output`: 命令输出内容
  - `truncated`: 是否被截断
  - `error`: 错误信息（如有）
- `successCount`: 成功执行的主机数
- `totalCount`: 总目标主机数

## 注意事项

- 按分组执行时，只选择**在线**（status=1）的主机
- 批量执行时按顺序逐台执行，不是并行
- 安装软件等耗时操作建议增大 timeout 参数
- 输出内容超过 64KB 时会被截断
