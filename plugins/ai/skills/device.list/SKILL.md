---
name: device.list
description: 查询网络设备列表，支持按关键词搜索（名称/IP/型号/SN），设备类型、连接协议、分组、状态筛选
category: device
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    keyword:
      type: string
      description: 搜索关键词（设备名称、IP、型号或SN序列号）
    device_type:
      type: string
      description: "设备类型: switch=交换机, router=路由器, firewall=防火墙, ac=AC, ap=AP, other=其他"
    protocol:
      type: string
      description: "连接协议: ssh 或 telnet"
    brand:
      type: string
      description: 品牌名称（模糊匹配，如 Cisco、Huawei、H3C、Ruijie）
    group_name:
      type: string
      description: 分组名称（模糊匹配）
    status:
      type: integer
      description: "状态: 1=在线 0=离线 -1=未知"
    limit:
      type: integer
      description: 返回数量限制，默认 20
---

# 网络设备列表查询

查询 MOM 平台管理的所有网络设备（交换机、路由器、防火墙等）。

## 使用场景

- 用户询问"有哪些网络设备"、"列出所有交换机"
- 用户需要按条件筛选设备（按品牌、类型、协议、分组）
- 用户想了解网络设备总数和在线/离线情况
- 用户想查找某个品牌或型号的设备

## 返回数据

- `devices`: 设备列表数组，每条记录包含 id、name、ip、brand、brandModel、serialNumber、deviceType、protocol、port、status、groupName、tags
- `total`: 设备总数
- `online`: 在线设备数
- `offline`: 离线设备数
- `unknown`: 未知状态设备数
- `byType`: 按设备类型分组的统计
- `resultCount`: 本次返回的记录数

## 注意事项

- 默认返回最多 20 条记录，如需更多请指定 limit 参数
- status 字段：1 表示在线，0 表示离线，-1 表示未知
