---
name: device.detail
description: 查询单台网络设备的详细信息，包括品牌型号、连接协议、端口、凭证、分组等。通过 IP 地址或设备名查找
category: device
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    ip:
      type: string
      description: 设备 IP 地址
    name:
      type: string
      description: 设备名称
    id:
      type: integer
      description: 设备 ID
---

# 网络设备详情查询

查询指定网络设备的完整详细信息。

## 使用场景

- 用户询问某台网络设备的详细配置
- 用户想查看某个 IP 的设备信息
- 需要了解设备的连接方式（SSH/Telnet）和端口

## 参数说明

至少提供以下参数之一：
- `ip`: 精确匹配 IP 地址
- `name`: 模糊匹配设备名称
- `id`: 精确匹配设备 ID

## 返回数据

完整的设备信息，包括：
- 基本信息：id, name, ip, brand, brandModel, serialNumber, deviceType
- 连接信息：protocol, port, credentialName
- 状态信息：status, statusText
- 附加信息：groupName, tags, description
