---
name: device.session_status
description: 查询当前 AI 对话中的网络设备交互会话状态，包括目标设备、协议、最近活跃时间和空闲超时
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
      description: 可选，按设备 ID 过滤当前对话中的交互会话
    ip:
      type: string
      description: 可选，按设备 IP 过滤当前对话中的交互会话
---

# 网络设备会话状态

查询当前 AI 对话内已建立的网络设备交互式 shell 会话。

## 使用场景

- 用户说"看下当前交换机会话还在不在"
- 用户说"当前这个设备还在配置模式里吗"
- 用户说"列出我现在打开的网络设备会话"

## 返回内容

- 当前活动会话数量
- 设备名称 / IP / 协议
- 最近活跃时间
- 空闲超时秒数
