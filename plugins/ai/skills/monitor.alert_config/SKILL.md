---
name: monitor.alert_config
description: 配置告警规则，支持创建、查询、启用/禁用和删除域名监控的告警配置
category: monitor
riskLevel: medium
scriptType: builtin
parameters:
  type: object
  properties:
    action:
      type: string
      description: "操作: list/create/enable/disable/delete"
    name:
      type: string
      description: 规则名称（创建时可选，自动生成；启用/禁用时可用于模糊匹配规则）
    rule_id:
      type: integer
      description: 规则 ID（启用/禁用/删除时使用）
    domain:
      type: string
      description: 监控域名（创建时必填）
    alert_type:
      type: string
      description: "告警类型（创建时必填）: domain_down/high_response_time/ssl_expiring/ssl_expired"
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数（create/delete 操作需要确认）
  required:
    - action
---

# 告警规则配置

配置域名监控的告警规则，支持创建、查询、启用/禁用和删除。

## 使用场景

- 查询当前配置的告警规则
- 为域名添加监控告警规则
- 启用或禁用现有告警规则
- 删除不再需要的告警规则

## 操作类型

### list - 查询告警规则

列出所有告警规则配置。

**返回：**
- 告警规则列表，包含 ID、名称、告警类型、启用状态、阈值、域名监控ID、严重等级、通知渠道

### create - 创建告警规则

为域名创建新的告警规则。

**必需参数：**
- `domain`: 监控域名
- `alert_type`: 告警类型

**可选参数：**
- `name`: 规则名称，不提供时自动生成为 `{domain}-{alert_type}-alert`

### enable / disable - 启用或禁用告警规则

按规则 ID 或名称模糊匹配来启用/禁用告警规则。

**参数（二选一）：**
- `rule_id`: 规则 ID
- `name`: 规则名称（模糊匹配）

### delete - 删除告警规则

删除指定 ID 的告警规则。

**必需参数：**
- `rule_id`: 规则 ID

## 告警类型

| 类型 | 说明 | 触发条件 |
|------|------|---------|
| domain_down | 域名不可访问 | 域名无法访问或返回错误状态码 |
| high_response_time | 响应时间过高 | 响应时间超过设定阈值（毫秒） |
| ssl_expiring | SSL 即将到期 | 证书剩余有效期小于设定天数 |
| ssl_expired | SSL 已过期 | 证书已过期 |

## 使用示例

**创建域名不可用告警：**
- action: create
- domain: example.com
- alert_type: domain_down

**禁用某条告警规则：**
- action: disable
- rule_id: 5

**启用名称匹配的告警规则：**
- action: enable
- name: example

**删除告警规则：**
- action: delete
- rule_id: 3

## 注意事项

- 同一域名可以配置多个不同类型的告警规则
- 创建规则前需要先通过 `monitor.domain_manage` 添加域名监控
- create 和 delete 操作需要用户确认
- enable/disable 操作直接执行，无需确认
