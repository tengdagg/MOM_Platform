---
name: k8s.node_manage
description: Kubernetes 节点管理，支持 Cordon（设为不可调度）、Uncordon（恢复调度）、Drain（排空节点）操作
category: k8s
riskLevel: critical
scriptType: builtin
parameters:
  type: object
  properties:
    cluster_id:
      type: integer
      description: 集群 ID
    cluster_name:
      type: string
      description: 集群名称
    node_name:
      type: string
      description: 节点名称
    action:
      type: string
      description: "操作类型: cordon/uncordon/drain"
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - node_name
    - action
---

# k8s.node_manage - 节点维护管理

## 功能描述

Kubernetes 节点管理，支持 Cordon（设为不可调度）、Uncordon（恢复调度）、Drain（排空节点）操作。用于节点维护、升级或故障处理。

## 使用场景

- **节点维护**：在节点需要维护时，先 Cordon 节点，然后进行维护操作
- **节点升级**：升级节点前，Drain 节点以安全地迁移 Pod
- **故障处理**：当节点出现故障时，Drain 节点以将 Pod 迁移到其他节点
- **资源回收**：在节点下线前，排空节点上的所有 Pod
- **恢复节点**：维护完成后，Uncordon 节点以恢复调度

## 操作说明

### Cordon（设为不可调度）
- 将节点标记为不可调度，阻止新的 Pod 调度到该节点
- 不会影响节点上已有的 Pod
- 用于准备节点维护或升级

### Uncordon（恢复调度）
- 恢复节点的调度能力，允许新的 Pod 调度到该节点
- 用于节点维护完成后的恢复操作

### Drain（排空节点）
- 将节点标记为不可调度
- 驱逐节点上的所有 Pod（DaemonSet 管理的 Pod 除外）
- Pod 会被重新调度到其他可用节点
- **危险操作**：会中断节点上所有 Pod 的运行

## 注意事项

⚠️⚠️⚠️ **极高风险操作，必须用户确认**

- **Drain 操作会驱逐所有 Pod**，可能导致服务中断
- 确保集群中有足够的节点资源，以容纳被驱逐的 Pod
- 对于有状态服务（StatefulSet），Drain 操作可能导致数据不一致
- 建议在业务低峰期进行节点维护操作
- 执行 Drain 前，确保应用有足够的副本数以维持服务可用性
- 监控被驱逐 Pod 的重新调度状态，确保服务恢复正常
- 某些 Pod 可能无法被驱逐（如使用 local storage），需要手动处理

## 参数说明

- `node_name`（必需）：目标节点名称
- `action`（必需）：操作类型，支持 `cordon`、`uncordon`、`drain`
- `cluster_id` 或 `cluster_name`：指定目标集群
