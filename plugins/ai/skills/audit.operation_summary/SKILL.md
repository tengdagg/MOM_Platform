---
name: audit.operation_summary
description: 统计分析操作日志，包括各模块操作次数、各用户操作次数、操作类型分布。支持按时间范围筛选
category: audit
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    days:
      type: integer
      description: 统计最近 N 天的日志，默认 1（今天）
      default: 1
    username:
      type: string
      description: 按用户名筛选
---

# 操作日志统计

对系统操作日志进行多维度统计分析。

## 使用场景

- 用户询问"今天做了多少操作"
- 用户想了解各模块的操作频率
- 审计某个用户的操作记录
- 日常运维报告中的操作统计

## 返回数据

- `period`: 统计时间范围描述
- `totalOps`: 总操作次数
- `byModule`: 按模块统计（模块名 + 次数）
- `byUser`: 按用户统计（用户名 + 次数，Top 10）
- `byAction`: 按操作类型统计（操作类型 + 次数）
