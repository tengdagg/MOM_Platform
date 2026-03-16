---
name: device.exec_command
description: 在指定网络设备上远程执行命令，支持 SSH 和 Telnet；同一 AI 对话内会优先复用交互式 shell 会话
category: device
riskLevel: critical
scriptType: builtin
parameters:
  type: object
  properties:
    device_ids:
      type: array
      items:
        type: integer
      description: 目标设备 ID 列表
    ip:
      type: string
      description: 目标设备 IP（与 device_ids 二选一）
    command:
      type: string
      description: 要执行的设备命令（如 show version、display interface brief）
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - command
---

# 网络设备远程命令执行

在指定网络设备上远程执行命令。

## 使用场景

- 用户说"查看交换机的 running-config"
- 用户说"在 172.20.8.90 上执行 show version"
- 用户说"检查所有路由器的接口状态"
- 用户说"查看防火墙的 ACL 规则"
- 用户说"进入配置模式后继续配置 NTP / VLAN / 路由"

## 风险说明

- 只读查询类命令会直接执行，例如 `show` / `display` / `dis` / `ping` / `traceroute` / `?` 帮助查询
- 非只读命令会进入人工确认流程
- 同一 AI 对话内，对同一设备重复调用时会优先复用交互式 shell 上下文
- 如果已经进入配置模式，后续命令默认仍在当前模式执行；需要返回上级模式时请显式执行 `end` / `exit`
- 会话空闲约 10 分钟后会自动关闭，也可以使用 `device.close_session` 主动结束
