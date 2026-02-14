---
name: host.detail
description: 查询单台主机的详细信息，包括 CPU、内存、磁盘使用情况，操作系统信息等。通过 IP 地址或主机名查找
category: host
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    ip:
      type: string
      description: 主机 IP 地址
    name:
      type: string
      description: 主机名称
    id:
      type: integer
      description: 主机 ID
---

# 主机详情查询

查询指定主机的完整详细信息。

## 使用场景

- 用户询问某台主机的详细配置
- 用户想查看某个 IP 的主机信息
- 需要诊断某台主机的资源使用情况

## 参数说明

至少提供以下参数之一：
- `ip`: 精确匹配 IP 地址
- `name`: 模糊匹配主机名称
- `id`: 精确匹配主机 ID

## 返回数据

完整的主机信息，包括：
- 基本信息：id, name, ip, port, os, osType, kernel, arch, hostname
- 状态信息：status, uptime
- 资源使用：cpuCores, cpuUsage, memoryTotal, memoryUsed, memoryUsage, diskTotal, diskUsed, diskUsage
- 附加信息：tags, description
