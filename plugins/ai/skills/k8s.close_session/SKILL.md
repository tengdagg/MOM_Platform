---
name: k8s.close_session
description: 关闭当前 AI 对话中的 Kubernetes Pod 交互式 shell 会话
category: k8s
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    cluster_id:
      type: integer
      description: 目标集群 ID
    cluster_name:
      type: string
      description: 目标集群名称（与 cluster_id 二选一）
    namespace:
      type: string
      description: 目标命名空间
    pod_name:
      type: string
      description: 目标 Pod 名称
    container:
      type: string
      description: 目标容器名称
    all:
      type: boolean
      description: 是否关闭当前对话中的全部 Pod 会话
---

# 关闭 Pod 会话

主动关闭当前对话中的 Pod 交互式 shell 会话，避免继续复用旧的工作目录、环境变量或 shell 状态。

## 适用场景

- "结束当前 Pod 会话"
- "关闭这个容器的 shell session"
- "把当前对话里所有 Pod 会话都清掉"

## 注意事项

- 关闭的是 AI 对话里的 Pod shell 上下文，不会删除 Pod
- 若指定 `all=true`，会关闭当前对话中的全部 Pod 会话
- 如果不主动关闭，会话会在空闲约 10 分钟后自动回收
