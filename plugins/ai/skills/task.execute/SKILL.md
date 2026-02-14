---
name: task.execute
description: 在指定主机上执行 Ad-hoc 命令。这是高风险操作，执行前需要用户确认。通过主机 IP 或名称指定目标主机
category: task
riskLevel: critical
scriptType: builtin
parameters:
  type: object
  properties:
    host_ip:
      type: string
      description: 目标主机 IP
    command:
      type: string
      description: 要执行的命令
  required:
    - host_ip
    - command
---

# 远程命令执行

在指定主机上执行 Ad-hoc 命令。

## ⚠️ 危险操作

此 Skill 风险等级为 **critical**，AI 不会直接执行命令，而是提示用户通过任务中心手动操作。

## 使用场景

- 用户要求在某台主机上执行命令
- 批量检查服务状态
- 紧急运维操作

## 安全策略

- 所有命令执行需要用户明确确认
- 操作会记录到审计日志
- 需要用户具有相应的 RBAC 权限
