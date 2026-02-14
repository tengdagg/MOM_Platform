---
name: analysis.capacity_plan
description: 容量规划分析，识别资源不足（CPU/内存/磁盘使用率过高）的主机，给出扩容建议
category: analysis
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    threshold:
      type: number
      description: 告警阈值百分比，默认 80
      default: 80
---

# 容量规划分析

分析所有在线主机的资源使用情况，识别瓶颈并提供扩容建议。

## 使用场景

- 用户询问"需不需要扩容"
- 月度容量规划报告
- 预算编制时的资源评估

## 分析维度

1. **CPU**: 超阈值主机数及列表
2. **内存**: 超阈值主机数及列表
3. **磁盘**: 超阈值主机数及列表
4. **综合评估**: 最紧急需要扩容的资源类型

## 返回数据

- `totalHosts`: 在线主机总数
- `threshold`: 使用的阈值
- `cpuOverload`: CPU 超载主机列表
- `memoryOverload`: 内存超载主机列表
- `diskOverload`: 磁盘超载主机列表
- `recommendations`: 容量规划建议
