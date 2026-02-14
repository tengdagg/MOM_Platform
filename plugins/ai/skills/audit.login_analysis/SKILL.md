---
name: audit.login_analysis
description: 分析登录行为，包括登录次数统计、失败登录记录、异常登录 IP 检测
category: audit
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    days:
      type: integer
      description: 统计最近 N 天的登录日志，默认 7
      default: 7
---

# 登录行为分析

分析系统登录行为，检测异常登录模式。

## 使用场景

- 安全检查：是否有暴力破解尝试
- 用户询问"有没有异常登录"
- 审计登录行为是否合规

## 分析内容

1. 总登录次数和失败次数
2. 频繁失败的用户（失败 > 2 次的用户）
3. 登录 IP 分布（Top 10）

## 返回数据

- `totalLogins`: 总登录次数
- `failedLogins`: 失败登录次数
- `failedUsers`: 频繁失败的用户列表
- `topIPs`: 登录 IP Top 10
