---
name: k8s.list_resources
description: 查询 Kubernetes 集群中的资源信息概览（从数据库缓存中获取）。返回集群的节点数和 Pod 数统计
category: k8s
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    cluster_name:
      type: string
      description: 集群名称（必填）
    resource_type:
      type: string
      description: "资源类型: nodes / pods / deployments / services"
      default: pods
  required:
    - cluster_name
---

# K8s 资源列表

查询指定 Kubernetes 集群的资源概览信息。

## 使用场景

- 用户询问某个集群有多少节点、Pod
- 需要了解集群资源概况

## 注意事项

- 当前返回的是数据库缓存的概览数据，非实时 K8s API 查询
- 详细的 Pod 列表、Deployment 状态等需要通过 K8s 管理页面查看
- cluster_name 为必填参数
