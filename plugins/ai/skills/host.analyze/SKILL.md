---
name: host.analyze
description: 分析主机健康状态和资源使用情况，找出磁盘使用率、CPU 使用率、内存使用率超过指定阈值的主机
category: host
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    metric:
      type: string
      description: "指标: cpu / memory / disk / all"
      default: all
    threshold:
      type: number
      description: 使用率阈值百分比，默认 80
      default: 80
---

# 主机健康分析

分析所有在线主机的资源使用情况，找出超过阈值的主机。

## 使用场景

- 用户询问"哪些主机磁盘快满了"
- 用户想要了解资源使用异常的主机
- 进行容量预警检查

## 分析逻辑

1. 查询所有在线主机（status=1）
2. 按指定指标（cpu/memory/disk/all）筛选使用率超过阈值的主机
3. 按使用率降序排列
4. 每种指标最多返回 20 条告警

## 返回数据

- `threshold`: 使用的阈值
- `alertCount`: 告警总数
- `alerts`: 告警列表，每条包含 name, ip, metric, usage
