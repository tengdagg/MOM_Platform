---
name: k8s.scale
description: 扩缩容 Kubernetes 工作负载（Deployment/StatefulSet），调整 Pod 副本数量
category: k8s
riskLevel: high
scriptType: builtin
parameters:
  type: object
  properties:
    cluster_id:
      type: integer
      description: 集群 ID
    cluster_name:
      type: string
      description: 集群名称（与 cluster_id 二选一）
    namespace:
      type: string
      description: 命名空间，默认 default
    resource_type:
      type: string
      description: "资源类型: Deployment 或 StatefulSet"
    resource_name:
      type: string
      description: 工作负载名称
    replicas:
      type: integer
      description: 目标副本数
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - resource_name
    - replicas
---

# k8s.scale - 工作负载扩缩容

## 功能描述

扩缩容 Kubernetes 工作负载（Deployment/StatefulSet），调整 Pod 副本数量。支持根据业务负载动态调整服务实例数量，实现弹性伸缩。

## 使用场景

- **扩展 Pod 副本**：当业务负载增加时，增加 Pod 副本数以提高服务处理能力
- **缩小 Pod 副本**：当业务负载降低时，减少 Pod 副本数以节省资源
- **水平扩展**：根据监控指标或计划任务自动调整副本数
- **资源优化**：在低峰期减少副本数，高峰期增加副本数

## 注意事项

⚠️ **高风险操作，需用户确认**

- 缩容操作会导致 Pod 被终止，可能影响正在处理的请求
- 扩容操作会增加资源消耗，需确保集群有足够的资源
- StatefulSet 的缩容操作会按逆序删除 Pod，需要谨慎操作
- 建议在业务低峰期进行缩容操作
- 确保有足够的节点资源支持扩容后的 Pod 调度
- 监控扩缩容后的服务状态，确保服务正常运行

## 参数说明

- `resource_name`（必需）：工作负载名称
- `replicas`（必需）：目标副本数，必须大于等于 0
- `cluster_id` 或 `cluster_name`：指定目标集群
- `namespace`：命名空间，默认为 `default`
- `resource_type`：资源类型，支持 `Deployment` 或 `StatefulSet`
