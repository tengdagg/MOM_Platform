---
name: task.history
description: 查询任务执行历史记录，支持按状态（成功/失败）、时间范围筛选，返回最近的任务执行结果
category: task
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    days:
      type: integer
      description: 查询最近 N 天的记录，默认 7
      default: 7
    status:
      type: string
      description: "状态筛选: success / failed / running"
    limit:
      type: integer
      description: 返回数量，默认 20
      default: 20
---

# 任务执行历史

查询任务中心的执行历史记录。

## 使用场景

- 用户询问"最近失败的任务"
- 查看任务执行趋势
- 审计任务执行记录

## 返回数据

- `records`: 任务记录列表
- `total`: 总数
- `failed`: 失败数
