---
name: device.manage
description: 管理网络设备：创建新设备、修改设备信息、删除设备。也可列出可用凭证和分组供选择
category: device
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
      description: 设备名称（创建时必填）
    ip:
      type: string
      description: 设备IP地址（创建时必填）
    device_type:
      type: string
      description: "设备类型（创建时必填）: switch=交换机, router=路由器, firewall=防火墙, ac=无线控制器, ap=无线AP, other=其他"
    protocol:
      type: string
      description: "连接协议: ssh 或 telnet，默认ssh"
    port:
      type: integer
      description: 连接端口，默认22（SSH）或23（Telnet）
    brand:
      type: string
      description: 品牌（如 Huawei, Cisco, H3C）
    brand_model:
      type: string
      description: 设备型号
    serial_number:
      type: string
      description: SN序列号
    credential_id:
      type: integer
      description: 凭证ID
    group_id:
      type: integer
      description: 分组ID
    tags:
      type: string
      description: 标签（逗号分隔）
    description:
      type: string
      description: 备注说明
    id:
      type: integer
      description: 设备ID（修改/删除时使用）
    device_ip:
      type: string
      description: 通过IP查找设备（修改/删除时使用）
    confirmed:
      type: boolean
      description: 确认执行操作
  required:
    - action
---

# 网络设备管理

创建、修改或删除网络设备，也可以查询可用凭证和分组。

## 使用场景

- 用户要求"添加一台交换机 192.168.1.1"
- 用户要求"删除设备 xxx"
- 用户要求"修改路由器的端口"
- 用户要求"有哪些凭证可以用"、"有哪些分组"

## 注意事项

- 创建/修改/删除是写操作，需要用户确认后才执行
- 创建设备时 name、ip、device_type 为必填项
- list_credentials 和 list_groups 为辅助查询，无需确认
