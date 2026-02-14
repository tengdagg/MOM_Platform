---
name: task.execute
description: 在指定主机上执行 Ad-hoc 命令。支持按 IP、分组名称或主机 ID 列表选择目标主机。高风险操作，执行前需用户确认。自动拒绝危险命令
category: task
riskLevel: critical
scriptType: builtin
parameters:
  type: object
  properties:
    command:
      type: string
      description: 要执行的 Shell 命令
    host_ip:
      type: string
      description: 目标主机 IP
    group_name:
      type: string
      description: 目标主机分组名称（在分组内所有在线主机上执行）
    host_ids:
      type: array
      items:
        type: integer
      description: 目标主机 ID 列表
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - command
---

# 远程命令执行

在一台或多台指定主机上执行 Ad-hoc Shell 命令。

## ⚠️ 危险操作

此 Skill 风险等级为 **critical**，需要两步确认：
1. 首次调用返回待执行的主机列表和命令详情
2. 用户确认后带 `confirmed=true` 再次调用真正执行

## 使用场景

- 用户要求在某台主机上执行命令
- 批量在某个分组的所有主机上检查服务状态
- 紧急运维操作（如重启服务、查看日志等）

## 安全策略

- 所有命令执行需要用户明确确认
- 自动拒绝危险命令（rm -rf /、mkfs、dd、shutdown 等）
- 执行超时限制：30 秒
- 操作会记录到审计日志
- 最多同时在 50 台主机上执行

## 返回数据

- `results`: 每台主机的执行结果（output、error）
- `successCount`: 成功执行的主机数
- `totalCount`: 总目标主机数
