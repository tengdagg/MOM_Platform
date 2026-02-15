---
name: device.test_connection
description: 测试网络设备的 SSH/Telnet 连接是否正常，支持单台或批量测试
category: device
riskLevel: high
scriptType: builtin
parameters:
  type: object
  properties:
    device_ids:
      type: array
      items:
        type: integer
      description: 要测试的设备 ID 列表
    ip:
      type: string
      description: 指定单台设备 IP 进行测试
    group_name:
      type: string
      description: 按分组名称测试，测试该分组下所有设备
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
---

# 网络设备连接测试

测试指定网络设备的 SSH 或 Telnet 连接是否正常。

## 使用场景

- 用户说"测试所有交换机的连接"
- 用户说"检查 172.20.8.90 能不能连上"
- 网络故障排查时批量测试设备可达性

## 注意事项

- 测试操作会尝试建立 SSH/Telnet 连接，可能触发设备日志
- 批量测试时按顺序逐台测试，较多设备可能耗时较长
- 测试完成后会自动更新设备的在线/离线状态
