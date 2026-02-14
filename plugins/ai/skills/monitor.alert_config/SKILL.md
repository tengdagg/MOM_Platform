---
name: monitor.alert_config
description: 配置告警规则，支持创建、查询和管理域名监控的告警配置
category: monitor
riskLevel: medium
scriptType: builtin
parameters:
  type: object
  properties:
    action:
      type: string
      description: "操作: list/create/update/delete"
    domain:
      type: string
      description: 监控域名
    alert_type:
      type: string
      description: "告警类型: domain_down/high_response_time/ssl_expiring/ssl_expired"
    threshold:
      type: number
      description: 告警阈值（如响应时间毫秒、SSL 过期天数）
    enable_email:
      type: boolean
      description: 是否启用邮件通知
    enable_webhook:
      type: boolean
      description: 是否启用 Webhook 通知
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - action
---

# 告警规则配置

配置域名监控的告警规则，支持多种告警类型和通知渠道。

## 使用场景

- 为域名添加监控告警规则
- 调整现有告警的阈值
- 启用或禁用告警通知渠道
- 查询当前配置的告警规则
- 删除不再需要的告警规则

## 操作类型

### list - 查询告警规则

列出指定域名的所有告警规则配置。

**参数：**
- `domain`: 可选，指定域名，不提供则列出所有告警规则

**返回：**
- 告警规则列表，包含告警类型、阈值、通知渠道等配置信息

### create - 创建告警规则

为域名创建新的告警规则。

**必需参数：**
- `domain`: 监控域名
- `alert_type`: 告警类型

**可选参数：**
- `threshold`: 告警阈值
- `enable_email`: 是否启用邮件通知（默认 false）
- `enable_webhook`: 是否启用 Webhook 通知（默认 false）

### update - 更新告警规则

修改现有告警规则的配置。

**必需参数：**
- `domain`: 监控域名
- `alert_type`: 告警类型

**可选参数：**
- `threshold`: 新的告警阈值
- `enable_email`: 更新邮件通知状态
- `enable_webhook`: 更新 Webhook 通知状态

### delete - 删除告警规则

删除指定域名的告警规则。

**必需参数：**
- `domain`: 监控域名
- `alert_type`: 告警类型

## 告警类型

### domain_down

域名不可访问告警。

- **阈值**: 不适用（基于连接状态）
- **触发条件**: 域名无法访问或返回错误状态码

### high_response_time

响应时间过高告警。

- **阈值**: 响应时间（毫秒），例如 5000 表示超过 5 秒触发告警
- **触发条件**: 域名响应时间超过设定阈值

### ssl_expiring

SSL 证书即将到期告警。

- **阈值**: 天数，例如 30 表示证书在 30 天内到期时触发告警
- **触发条件**: SSL 证书剩余有效期小于设定天数

### ssl_expired

SSL 证书已过期告警。

- **阈值**: 不适用（基于证书有效期）
- **触发条件**: SSL 证书已过期

## 通知渠道

### 邮件通知

启用后，告警触发时会发送邮件通知。

- 需要配置邮件服务器
- 支持多个收件人

### Webhook 通知

启用后，告警触发时会调用配置的 Webhook URL。

- 支持自定义 Webhook 地址
- 可以集成到第三方系统（如钉钉、企业微信、Slack 等）

## 使用示例

**创建响应时间告警：**
- action: create
- domain: example.com
- alert_type: high_response_time
- threshold: 3000
- enable_email: true

**更新 SSL 到期告警：**
- action: update
- domain: example.com
- alert_type: ssl_expiring
- threshold: 15
- enable_webhook: true

## 注意事项

- 同一域名可以配置多个不同类型的告警规则
- 阈值设置需要根据实际业务需求调整
- 建议为关键域名同时启用邮件和 Webhook 通知
- 删除告警规则前请确认是否还需要监控
