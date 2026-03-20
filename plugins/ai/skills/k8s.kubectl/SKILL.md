---
name: k8s.kubectl
description: 通用 Kubernetes 资源操作，面向 Pod/Deployment/Service/Node/PVC 等资源对象本身；适用于 get/describe/logs/events/scale/restart/delete，不用于进入容器内部执行 Linux/Shell 命令
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
    action:
      type: string
      description: "操作类型: cluster_status(查询集群概览)/get(查询列表)/describe(查看详情)/scale(扩缩容)/restart(重启)/delete(删除)/cordon(节点不可调度)/uncordon(节点恢复调度)/drain(排空节点)/logs(查看日志)/events(查看事件)/top(资源使用量)"
    resource_type:
      type: string
      description: "资源类型，如: pods/deployments/services/configmaps/ingresses/nodes/pv/pvc/secrets/jobs/cronjobs/daemonsets/statefulsets/namespaces/endpoints/replicasets/hpa/networkpolicies 等"
    resource_name:
      type: string
      description: 资源名称（describe/delete/scale/restart 等操作时需要）
    namespace:
      type: string
      description: 命名空间，不传则查询所有命名空间。对于非命名空间资源（如 nodes/pv）无需指定
    labels:
      type: string
      description: 标签筛选器，如 "app=nginx,env=prod"
    field_selector:
      type: string
      description: 字段筛选器，如 "status.phase=Running"
    replicas:
      type: integer
      description: 扩缩容时的目标副本数
    container:
      type: string
      description: 查看日志时指定容器名称
    tail_lines:
      type: integer
      description: 查看日志时返回的行数，默认 100
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true（delete/scale/restart/cordon/uncordon/drain 等写操作需要确认）
  required:
    - action
    - resource_type
---

# 通用 Kubernetes 资源操作

提供类似 kubectl 的全面 K8s 资源管理能力，支持查询和操作任意类型的 Kubernetes 资源。

## 使用场景

- "查看 test 命名空间下所有 Pod 的状态"
- "查看所有 Service"
- "描述 nginx Deployment 的详细信息"
- "查看所有节点状态"
- "查看 ConfigMap 列表"
- "删除某个 Pod"
- "查看 CronJob 列表"
- "查看 Ingress 配置"
- "查看 PVC 使用情况"
- "查看某个 Namespace 下的常见资源（最常用）": `kubectl get all -n <namespace>`
  > ⚠️ 注意：`get all` 只包含 Pod, Service, Deployment, ReplicaSet, StatefulSet, DaemonSet, Job, CronJob。
  > ❗ 不会包含：ConfigMap, Secret, PVC, Ingress, ServiceAccount, CRD 资源。

## 选择规则

遇到以下需求，优先选择 `k8s.kubectl`：

- 查看 Kubernetes 资源对象列表、状态、详情、事件
- 查看 Pod 标准日志（stdout/stderr）
- 查看 Deployment / StatefulSet / Service / Node / PVC / Ingress 等资源信息
- 扩缩容、重启、删除、cordon、uncordon、drain 等集群层变更

以下场景不要优先选它，应改用 `k8s.exec_command`：

- 进入 Pod 或容器内部执行命令
- 查看容器内文件、目录、环境变量、进程、挂载点
- 需要 `cd /app` 之后继续执行多条依赖上下文的命令
- 明确要排查容器文件系统内部问题，而不是资源对象状态

快速判断：

- 如果用户问的是"这个 Pod / Deployment / Service / Node 现在怎么样"，用 `k8s.kubectl`
- 如果用户问的是"容器里面有什么文件 / 进程 / 环境变量"，用 `k8s.exec_command`
- 如果用户说"看日志"，默认理解为标准输出日志，用 `k8s.kubectl(action="logs")`
- 如果用户说"看容器里的某个日志文件"，例如 `/var/log/app.log`，才改用 `k8s.exec_command`

## 支持的操作

| 操作 | 说明 | 风险等级 |
|------|------|---------|
| get | 查询资源列表 | 低 |
| describe | 查看资源详情（含事件） | 低 |
| logs | 查看 Pod 日志 | 低 |
| events | 查看事件 | 低 |
| top | 查看资源使用量 | 低 |
| scale | 扩缩容 | 高（需确认） |
| restart | 滚动重启 | 高（需确认） |
| delete | 删除资源 | 高（需确认） |
| cordon | 节点设为不可调度 | 危险（需确认） |
| uncordon | 节点恢复调度 | 高（需确认） |
| drain | 排空节点 | 危险（需确认） |

## 注意事项

- 写操作（delete/scale/restart/cordon/drain）需要用户确认
- 查询操作（get/describe/logs/events/top）直接执行
- 实际返回会携带 `effectiveRiskLevel`，用于区分当前动作是真正的低风险查询还是高风险变更
- 支持所有标准 Kubernetes 资源类型
- 如果用户说的是"进入 Pod 看文件/进程/环境变量"，不要用本 Skill，应切换为 `k8s.exec_command`
