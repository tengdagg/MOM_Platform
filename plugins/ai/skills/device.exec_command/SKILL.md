---
name: device.exec_command
description: 在指定网络设备上远程执行命令（如 show running-config、display version 等），支持 SSH 和 Telnet
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

## 风险说明

- `show` / `display` / `dis` / `ping` / `traceroute` 等常见查看命令会直接执行
- 未识别为只读查询的命令会进入确认流程
- `write erase`、`format`、`delete /force` 等破坏性命令会被直接拒绝
- 建议优先使用 `show` / `display` 类查看命令
