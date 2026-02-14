---
name: monitor.domain_status
description: 查询域名监控状态，列出所有被监控的域名及其当前状态（正常/异常/暂停），响应时间，SSL 证书有效期等
category: monitor
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    status:
      type: string
      description: "状态筛选: normal / abnormal / paused"
    domain:
      type: string
      description: 域名关键词搜索
---

# 域名监控状态

查询域名监控的实时状态。

## 使用场景

- 用户询问"哪些域名不可访问"
- 检查 SSL 证书到期情况
- 域名监控概况报告

## 返回数据

- `domains`: 域名列表（domain, status, responseTime, sslValid, sslExpiry, lastCheck）
- `total`: 监控域名总数
- `normal`: 正常数量
- `abnormal`: 异常数量
- `sslExpiringSoon`: 30 天内 SSL 即将到期的数量
