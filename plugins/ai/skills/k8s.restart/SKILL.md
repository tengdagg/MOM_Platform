---
name: k8s.restart
description: 重启 Kubernetes 工作负载，通过滚动重启实现 Pod 更新
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
      description: 集群名称
    namespace:
      type: string
      description: 命名空间
    resource_type:
      type: string
      description: "资源类型: Deployment/StatefulSet/DaemonSet"
    resource_name:
      type: string
      description: 工作负载名称
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - resource_name
---

# k8s.restart - 滚动重启工作负载

## 功能描述

重启 Kubernetes 工作负载，通过滚动重启实现 Pod 更新。此操作会触发工作负载的滚动更新，逐个替换 Pod，确保服务在重启过程中保持可用性。

## 使用场景

- **服务重启**：当服务出现异常或需要重新加载配置时，重启工作负载
- **配置更新**：在配置变更后，通过重启使新配置生效
- **故障恢复**：当 Pod 出现问题时，通过重启尝试恢复服务
- **版本回滚**：配合版本管理，重启到之前的版本
- **定期维护**：定期重启服务以清理内存泄漏等问题

## 注意事项

⚠️ **高风险操作，需用户确认**

- 滚动重启会逐个替换 Pod，确保服务可用性，但可能影响性能
- 重启过程中，部分 Pod 会暂时不可用
- StatefulSet 的滚动重启会按顺序进行，耗时可能较长
- 确保工作负载有足够的副本数，避免重启时服务完全不可用
- 建议在业务低峰期进行重启操作
- 监控重启后的服务状态，确保所有 Pod 正常运行
- 通过设置 annotation 触发滚动更新，不会修改工作负载的其他配置

## 参数说明

- `resource_name`（必需）：工作负载名称
- `cluster_id` 或 `cluster_name`：指定目标集群
- `namespace`：命名空间
- `resource_type`：资源类型，支持 `Deployment`、`StatefulSet` 或 `DaemonSet`
