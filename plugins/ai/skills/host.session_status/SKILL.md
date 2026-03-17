---
name: host.session_status
description: 查询当前 AI 对话中的主机交互会话状态，包括目标主机、最近活跃时间和空闲超时
category: host
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    host_ids:
      type: array
      items:
        type: integer
      description: 可选，按主机 ID 过滤当前对话中的交互会话
    ip:
      type: string
      description: 可选，按主机 IP 过滤当前对话中的交互会话
---

# 主机会话状态

查询当前 AI 对话内已建立的主机交互式 shell 会话。

## 使用场景

- 用户说"看下当前主机会话还在不在"
- 用户说"列出我现在打开的主机会话"
- 用户说"查看 192.168.1.10 的会话状态"
