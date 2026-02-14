---
name: monitor.alert_summary
description: 告警日志汇总分析，按时间范围统计告警数量、类型分布和严重程度
category: monitor
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    days:
      type: integer
      description: 统计最近 N 天，默认 1
      default: 1
---

# 告警汇总分析

对告警日志进行汇总统计分析。

## 使用场景

- 用户询问"今天有多少告警"
- 按类型分析告警分布
- 运维报告中的告警统计

## 返回数据

- `totalAlerts`: 告警总数
- `byType`: 按告警类型统计
- `byStatus`: 按状态统计
