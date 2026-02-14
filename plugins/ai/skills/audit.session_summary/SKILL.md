---
name: audit.session_summary
description: 统计终端会话（SSH/RDP）的数量、类型分布和连接情况
category: audit
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

# 终端会话汇总

统计 SSH/RDP 终端会话的使用情况。

## 使用场景

- 用户询问"今天有多少终端会话"
- 审计谁在使用远程连接
- 按协议类型（SSH/RDP）统计

## 返回数据

- `totalSessions`: 会话总数
- `byProtocol`: 按协议统计（SSH/RDP）
- `byUser`: 按用户统计 Top 10
