---
name: k8s.cluster_status
description: 查询所有 Kubernetes 集群的状态概览，包括集群名称、版本、节点数、Pod 数、状态等信息
category: k8s
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    cluster_name:
      type: string
      description: 集群名称筛选（可选）
---

# K8s 集群状态查询

查询平台管理的所有 Kubernetes 集群的健康状态概览。

## 使用场景

- 用户询问"集群健康状态"
- 用户想了解有多少个集群、哪些异常
- 日常巡检时检查集群状态

## 返回数据

- `clusters`: 集群列表，每条包含 id, name, alias, apiEndpoint, version, status, statusText, provider, region, nodeCount, podCount
- `total`: 集群总数
- `normal`: 正常集群数
- `abnormal`: 异常集群数

## 状态说明

- 1: 正常
- 2: 连接失败
- 3: 不可用
