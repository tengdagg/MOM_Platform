---
name: task.ansible
description: 执行 Ansible Playbook 任务，支持指定主机清单、额外变量和标签
category: task
riskLevel: high
scriptType: builtin
parameters:
  type: object
  properties:
    task_name:
      type: string
      description: 任务名称
    playbook_name:
      type: string
      description: Playbook 名称或路径
    host_ids:
      type: array
      items:
        type: integer
      description: 目标主机 ID 列表
    group_name:
      type: string
      description: 目标主机分组名称
    extra_vars:
      type: object
      description: 额外变量（JSON 格式）
    tags:
      type: string
      description: 只执行指定 tag 的任务
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - playbook_name
---

# Ansible Playbook 执行

执行 Ansible Playbook 任务，支持批量部署和配置管理。

## ⚠️ 高风险操作

此 Skill 风险等级为 **high**，执行 Playbook 可能会修改系统配置、部署应用或进行批量操作。AI 在执行前需要用户明确确认。

## 使用场景

- 批量部署应用或服务
- 配置管理（如修改配置文件、安装软件包）
- 系统初始化（如服务器初始化配置）
- 执行特定标签的任务（如只执行部署任务，跳过配置任务）

## 参数说明

### 必需参数

- `playbook_name`: Playbook 名称或路径，指定要执行的 Playbook 文件

### 可选参数

- `task_name`: 任务名称，用于标识和记录此次执行
- `host_ids`: 目标主机 ID 列表，指定要执行的主机
- `group_name`: 目标主机分组名称，可以按分组批量执行
- `extra_vars`: 额外变量（JSON 格式），用于传递动态参数给 Playbook
- `tags`: 只执行指定 tag 的任务，支持选择性执行 Playbook 中的部分任务

## 执行流程

1. 验证 Playbook 文件是否存在
2. 解析主机清单（通过 host_ids 或 group_name）
3. 合并额外变量（extra_vars）
4. 如果指定了 tags，则只执行匹配的任务
5. 执行 Playbook 并返回执行结果

## 安全策略

- 所有 Playbook 执行需要用户明确确认
- 操作会记录到审计日志
- 需要用户具有相应的 RBAC 权限
- 支持通过 tags 限制执行范围，降低误操作风险

## 注意事项

- 确保 Playbook 文件路径正确
- 批量操作前建议先在测试环境验证
- 使用 tags 可以更精确地控制执行范围
- extra_vars 中的变量会覆盖 Playbook 中的默认变量
