---
name: analysis.security_audit
description: 执行安全审计检查，分析登录异常、高权限用户、禁用但未清理的账号等安全风险
category: analysis
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    days:
      type: integer
      description: 分析最近 N 天的数据，默认 7
      default: 7
---

# 安全审计分析

对平台安全相关的数据进行综合审计分析。

## 使用场景

- 定期安全检查
- 用户询问"有没有安全风险"
- 合规审计

## 检查项目

1. **登录分析**: 失败登录次数、频繁失败用户
2. **用户分析**: 管理员用户数、禁用用户数
3. **安全建议**: 基于数据给出的安全改进建议

## 返回数据

- `loginAnalysis`: 登录统计（totalLogins, failedLogins, suspiciousUsers）
- `userAnalysis`: 用户统计（totalUsers, adminUsers, disabledUsers）
- `recommendations`: 安全建议列表
