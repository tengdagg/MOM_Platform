---
name: monitor.domain_manage
description: 管理域名监控：创建新的域名监控、修改监控参数、删除域名监控。支持配置检查间隔、SSL检查、告警阈值等参数
category: monitor
riskLevel: medium
scriptType: builtin
parameters:
  type: object
  properties:
    action:
      type: string
      description: "操作类型: create=创建, update=修改, delete=删除"
    domain:
      type: string
      description: 要监控的域名（创建/删除时必填）
    id:
      type: integer
      description: 域名监控ID（修改/删除时使用）
    check_interval:
      type: integer
      description: 检查间隔（秒），默认300
    enable_ssl:
      type: boolean
      description: 是否启用SSL检查，默认true
    enable_alert:
      type: boolean
      description: 是否启用告警，默认false
    response_threshold:
      type: integer
      description: 响应时间告警阈值（ms），默认1000
    ssl_expiry_days:
      type: integer
      description: SSL证书过期提前告警天数，默认30
    confirmed:
      type: boolean
      description: 确认执行操作
  required:
    - action
---

# 域名监控管理

创建、修改或删除域名监控。

## 使用场景

- 用户要求"添加 xxx.com 域名监控"
- 用户要求"删除某个域名的监控"
- 用户要求"修改某个域名的监控间隔"

## 注意事项

- 创建和删除是写操作，需要用户确认后才执行
- 创建时会自动设置默认检查间隔300秒
- 域名不能重复添加
