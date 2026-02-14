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
      description: 执行超时时间（秒），默认 30
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - command
---

# 远程命令执行

在指定主机上远程执行 Shell 命令。

## 使用场景

- 用户说"在所有 Web 服务器上检查 nginx 状态"
- 用户说"在 192.168.1.10 上执行 df -h"
- 用户说"查看所有主机的内存使用"

## 安全警告

- 这是 **危险操作**，执行前必须确认命令内容
- 禁止执行 rm -rf、格式化磁盘等破坏性命令
- 会对命令进行安全检查，拒绝高危命令
