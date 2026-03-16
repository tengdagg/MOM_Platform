---
name: device.close_session
description: 关闭当前 AI 对话中的网络设备交互会话，可关闭指定设备会话或关闭当前对话的全部设备会话
category: device
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    device_ids:
      type: array
      items:
        type: integer
      description: 可选，按设备 ID 关闭交互会话
    ip:
      type: string
      description: 可选，按设备 IP 关闭交互会话
    all:
      type: boolean
      description: 设为 true 时关闭当前对话中的全部网络设备会话
---

# 关闭网络设备会话

关闭当前 AI 对话中的网络设备交互式 shell 会话。

## 使用场景

- 用户说"结束当前交换机会话"
- 用户说"关闭 172.20.8.90 的配置会话"
- 用户说"把当前对话里所有网络设备会话都断开"

## 注意事项

- 这是 AI 对话上下文中的设备 shell 会话，不会删除设备本身
- 关闭后再次执行设备命令会重新建立会话
