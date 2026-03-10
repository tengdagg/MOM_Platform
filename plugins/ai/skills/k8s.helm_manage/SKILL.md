---
name: k8s.helm_manage
description: Helm Release 管理，支持查询、安装、升级、卸载 Helm Release
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
    action:
      type: string
      description: "操作: list/install/upgrade/uninstall/status"
    release_name:
      type: string
      description: Release 名称
    chart_name:
      type: string
      description: Chart 名称（安装/升级时需要）
    chart_version:
      type: string
      description: Chart 版本
    values:
      type: object
      description: 自定义 values（JSON 格式）
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - action
---

# k8s.helm_manage - Helm Release 管理

## 功能描述

Helm Release 管理，支持查询、安装、升级、卸载 Helm Release。Helm 是 Kubernetes 的包管理工具，通过 Chart 简化应用的部署和管理。

## 使用场景

- **部署应用**：使用 Helm Chart 快速部署应用及其依赖
- **升级应用**：升级已部署的应用到新版本
- **回滚应用**：回滚到之前的版本
- **查询 Release**：查看集群中的 Helm Release 列表和状态
- **卸载应用**：卸载不再需要的应用
- **配置管理**：通过 values 文件自定义应用配置

## 操作说明

### list（列表查询）
- 列出指定命名空间或所有命名空间的 Helm Release
- 显示 Release 名称、命名空间、版本、状态等信息
- 低风险操作，仅查询信息

### install（安装）
- 安装新的 Helm Release
- 需要指定 Chart 名称和 Release 名称
- **高风险操作**：会创建新的资源，可能影响现有服务

### upgrade（升级）
- 升级已存在的 Helm Release
- 可以升级 Chart 版本或更新配置
- **高风险操作**：会修改现有资源，可能导致服务中断

### uninstall（卸载）
- 卸载 Helm Release，删除相关资源
- **高风险操作**：会删除资源，可能导致服务不可用

### status（状态查询）
- 查询指定 Release 的详细状态
- 显示 Release 的配置、资源状态等信息
- 低风险操作，仅查询信息

## 注意事项

⚠️ **高风险操作，需用户确认**

- `list` 和 `status` 属于低风险查询，直接执行
- **安装和升级是高风险操作**，会创建或修改 Kubernetes 资源
- 安装前确保 Chart 来源可靠，避免安装恶意 Chart
- 升级操作可能导致服务短暂中断，建议在业务低峰期进行
- 卸载操作会删除所有相关资源，包括数据（如果 Chart 配置了数据删除）
- 建议在升级前备份重要数据
- 使用 `values` 参数自定义配置时，确保配置格式正确
- 某些 Chart 可能需要特定的权限或先决条件，请查看 Chart 文档
- 监控安装/升级后的服务状态，确保应用正常运行

## 参数说明

- `action`（必需）：操作类型，支持 `list`、`install`、`upgrade`、`uninstall`、`status`
- `release_name`：Release 名称（install/upgrade/uninstall/status 时需要）
- `chart_name`：Chart 名称（install/upgrade 时需要）
- `chart_version`：Chart 版本（可选，不指定则使用最新版本）
- `values`：自定义 values，JSON 格式（install/upgrade 时使用）
- `cluster_id` 或 `cluster_name`：指定目标集群
- `namespace`：命名空间
