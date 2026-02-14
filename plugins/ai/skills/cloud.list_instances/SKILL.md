---
name: cloud.list_instances
description: 查询已导入的云主机实例列表，支持按云厂商和区域筛选
category: cloud
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    provider:
      type: string
      description: "云厂商: aliyun / tencent / aws 等"
    region:
      type: string
      description: 区域
    limit:
      type: integer
      description: 返回数量，默认 20
      default: 20
---

# 云主机实例列表

查询已导入到平台的云主机实例。

## 使用场景

- 用户询问"AWS 有哪些实例"
- 查看某个区域的云主机
- 统计云主机数量

## 返回数据

- `instances`: 实例列表（id, name, ip, cloudProvider, cloudInstanceId, os, status）
- `total`: 实例总数
