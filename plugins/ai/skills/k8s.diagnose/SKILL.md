---
name: k8s.diagnose
description: 诊断 Kubernetes Pod 或节点问题，分析事件、状态、日志等信息
category: k8s
riskLevel: low
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
    namespace:
      type: string
      description: 命名空间
    pod_name:
      type: string
      description: Pod 名称
    node_name:
      type: string
      description: 节点名称（诊断节点时使用）
---

# k8s.diagnose - Pod/节点问题诊断

## 功能描述

诊断 Kubernetes Pod 或节点问题，分析事件、状态、日志等信息。通过收集 Pod 或节点的详细信息，帮助快速定位和解决故障。

## 使用场景

- **Pod 启动失败**：排查 Pod 无法启动的原因，查看事件和状态信息
- **CrashLoopBackOff**：分析 Pod 反复崩溃的原因，查看容器退出码和日志
- **Pod 状态异常**：诊断 Pod 处于 Pending、Error、Unknown 等异常状态的原因
- **节点问题**：诊断节点不可用、资源不足等问题
- **资源调度问题**：分析 Pod 无法调度到节点的原因
- **网络问题**：排查 Pod 网络连接问题
- **存储问题**：诊断 PVC 挂载失败等问题

## 注意事项

- 这是一个只读操作，不会修改任何资源，风险较低
- 诊断信息包括事件列表、Pod 状态、容器状态、资源使用情况等
- 对于节点诊断，会返回节点状态、资源使用、调度信息等
- 建议结合日志查询功能，获取更详细的错误信息
- 诊断结果可能包含敏感信息，注意保护隐私

## 参数说明

- `pod_name`：Pod 名称（诊断 Pod 时使用）
- `node_name`：节点名称（诊断节点时使用）
- `cluster_id` 或 `cluster_name`：指定目标集群
- `namespace`：命名空间（诊断 Pod 时需要）

## 返回信息

- **事件列表**：Pod 或节点的相关事件，按时间排序
- **状态信息**：Pod 或节点的当前状态和详细信息
- **资源使用**：CPU、内存等资源的使用情况
- **调度信息**：Pod 的调度状态和节点分配情况
