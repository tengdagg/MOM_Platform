---
name: k8s.session_status
description: 查看当前 AI 对话中的 Kubernetes Pod 交互式 shell 会话状态
category: k8s
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    cluster_id:
      type: integer
      description: 按集群 ID 过滤
    cluster_name:
      type: string
      description: 按集群名称过滤（与 cluster_id 二选一）
    namespace:
      type: string
      description: 按命名空间过滤
    pod_name:
      type: string
      description: 按 Pod 名称过滤
    container:
      type: string
      description: 按容器名称过滤
---

# 查询 Pod 会话状态

查看当前对话中已经建立的 Pod shell 会话，确认是否仍在活动中，以及空闲多久会自动关闭。

## 适用场景

- "看看当前还有哪些 Pod 会话没关"
- "这个 Pod shell 会话还在不在"
- "排查为什么后续命令会复用之前的目录上下文"

## 返回信息

- 集群
- 命名空间
- Pod 名称
- 容器名称
- 最后活跃时间
- 空闲秒数
- 自动超时秒数
