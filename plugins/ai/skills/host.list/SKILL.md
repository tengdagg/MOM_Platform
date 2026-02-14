---
name: host.list
description: 查询主机列表，支持按关键词搜索（名称或 IP），按操作系统类型筛选（linux/windows），按状态筛选（1:在线 0:离线），按分组名称筛选
category: host
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    keyword:
      type: string
      description: 搜索关键词（主机名或 IP）
    os_type:
      type: string
      description: "操作系统类型: linux 或 windows"
    status:
      type: integer
      description: "状态: 1=在线 0=离线"
    group_name:
      type: string
      description: 分组名称
    limit:
      type: integer
      description: 返回数量限制，默认 20
---

# 主机列表查询

查询 MOM 平台管理的所有主机列表。

## 使用场景

- 用户询问"有哪些主机"、"列出所有主机"
- 用户需要按条件筛选主机（按 IP、名称、操作系统、状态、分组）
- 用户想了解主机总数和在线/离线情况

## 返回数据

- `hosts`: 主机列表数组，每条记录包含 id、name、ip、port、os、osType、status、cpuCores、cpuUsage、memoryUsage、diskUsage、groupName
- `total`: 主机总数
- `online`: 在线主机数
- `offline`: 离线主机数
- `resultCount`: 本次返回的记录数

## 注意事项

- 默认返回最多 20 条记录，如需更多请指定 limit 参数
- status 字段：1 表示在线，0 表示离线
- 使用率字段为百分比数值（0-100）
