---
name: analysis.infra_report
description: 生成基础设施综合报告，汇总主机、K8s 集群、云账号、监控、任务等各模块的概况数据
category: analysis
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties: {}
---

# 基础设施综合报告

一键生成平台所有模块的综合数据报告。

## 使用场景

- 每日/每周巡检报告
- 用户询问"平台整体状况"
- 管理层汇报用的概况数据

## 报告内容

1. **主机概况**: 总数、在线/离线、Linux/Windows 分布
2. **K8s 集群**: 集群总数、正常/异常
3. **云账号**: 账号总数、启用数
4. **域名监控**: 监控域名数、正常/异常
5. **今日操作**: 操作日志数量
6. **今日会话**: 终端会话数量

## 返回数据

结构化的各模块统计数据，包含 `hosts`、`kubernetes`、`cloudAccounts`、`domainMonitor`、`todayOperations`、`todaySessions`、`generatedAt` 字段。
