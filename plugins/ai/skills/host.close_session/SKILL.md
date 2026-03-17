---
name: host.close_session
description: 关闭当前 AI 对话中的主机交互会话，可关闭指定主机会话或关闭当前对话的全部主机会话
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
      description: 可选，按主机 ID 关闭交互会话
    ip:
      type: string
      description: 可选，按主机 IP 关闭交互会话
    all:
      type: boolean
      description: 设为 true 时关闭当前对话中的全部主机会话
---

# 关闭主机会话

关闭当前 AI 对话中的主机交互式 shell 会话。

## 使用场景

- 用户说"结束当前主机会话"
- 用户说"关闭 192.168.1.10 的交互会话"
- 用户说"把当前对话里所有主机会话都断开"
