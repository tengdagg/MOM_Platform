---
name: k8s.log_query
description: 查询 Kubernetes Pod 日志，支持指定容器、行数限制
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
    container:
      type: string
      description: 容器名称（多容器 Pod 时指定）
    tail_lines:
      type: integer
      description: 返回最后多少行日志，默认 100
    previous:
      type: boolean
      description: 是否查看上一次（崩溃前）的日志
  required:
    - pod_name
---

# k8s.log_query - Pod 日志查询

## 功能描述

查询 Kubernetes Pod 日志，支持指定容器、行数限制。可以查看 Pod 中容器的实时日志和历史日志，帮助排查问题、监控服务状态。

## 使用场景

- **查看服务日志**：查看应用的运行日志，了解服务状态
- **排查错误**：当服务出现错误时，查看日志定位问题原因
- **监控调试**：实时监控服务日志，进行调试和性能分析
- **故障分析**：分析 Pod 崩溃前的日志，找出崩溃原因
- **多容器 Pod**：查看 Pod 中特定容器的日志
- **日志审计**：查看历史日志记录，进行审计和分析

## 注意事项

- 这是一个只读操作，不会修改任何资源，风险较低
- 日志可能包含敏感信息（如密码、密钥等），注意保护隐私
- 对于多容器 Pod，需要指定 `container` 参数以查看特定容器的日志
- `previous` 参数用于查看已崩溃容器的上一次日志，有助于诊断 CrashLoopBackOff 问题
- 大量日志查询可能影响集群性能，建议合理设置 `tail_lines` 参数
- 日志查询受集群日志保留策略限制，过旧的日志可能无法查询

## 参数说明

- `pod_name`（必需）：Pod 名称
- `cluster_id` 或 `cluster_name`：指定目标集群
- `namespace`：命名空间
- `container`：容器名称，多容器 Pod 时必须指定
- `tail_lines`：返回最后多少行日志，默认为 100
- `previous`：是否查看上一次（崩溃前）的日志，默认为 `false`

## 使用示例

- 查看 Pod 最近 100 行日志
- 查看多容器 Pod 中特定容器的日志
- 查看崩溃容器上一次的日志（用于诊断 CrashLoopBackOff）
