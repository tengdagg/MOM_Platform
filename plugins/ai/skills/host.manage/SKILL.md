---
name: host.manage
description: 管理主机：创建新主机、修改主机信息、删除主机。也可列出可用凭证和分组供选择
category: host
riskLevel: high
scriptType: builtin
parameters:
  type: object
  properties:
    action:
      type: string
      description: "操作类型: create=创建, update=修改, delete=删除, list_credentials=列出凭证, list_groups=列出分组"
    name:
      type: string
      description: 主机名称（创建时必填）
    ip:
      type: string
      description: 主机IP地址（创建时必填）
    port:
      type: integer
      description: SSH端口，默认22
    ssh_user:
      type: string
      description: SSH用户名（创建时必填）
    credential_id:
      type: integer
      description: 凭证ID
    group_id:
      type: integer
      description: 分组ID
    os_type:
      type: string
      description: "操作系统类型: linux 或 windows，默认linux"
    tags:
      type: string
      description: 标签（逗号分隔）
    description:
      type: string
      description: 备注说明
    id:
      type: integer
      description: 主机ID（修改/删除时使用）
    host_ip:
      type: string
      description: 通过IP查找主机（修改/删除时使用）
    rdp_port:
      type: integer
      description: RDP端口，默认3389（Windows主机）
    confirmed:
      type: boolean
      description: 确认执行操作
  required:
    - action
---

# 主机管理

创建、修改或删除主机，也可以查询可用凭证和分组。

## 使用场景

- 用户要求"添加一台主机 192.168.1.100"
- 用户要求"删除主机 xxx"
- 用户要求"修改主机的SSH端口"
- 用户要求"有哪些凭证可以用"、"有哪些分组"

## 注意事项

- 创建/修改/删除是写操作，需要用户确认后才执行
- 创建主机时 name、ip、ssh_user 为必填项
- list_credentials 和 list_groups 为辅助查询，无需确认
