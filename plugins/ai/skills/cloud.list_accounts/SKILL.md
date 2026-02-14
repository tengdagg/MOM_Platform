---
name: cloud.list_accounts
description: 查询所有云平台账号列表，包括账号名称、云厂商、状态等信息
category: cloud
riskLevel: low
scriptType: builtin
parameters:
  type: object
  properties:
    provider:
      type: string
      description: "按云厂商筛选: aliyun / tencent / aws / jdcloud / baidu / ksyun"
---

# 云账号列表

查询平台配置的所有云平台账号。

## 使用场景

- 用户询问"有哪些云账号"
- 按云厂商筛选账号
- 查看账号启用状态

## 支持的云厂商

- aliyun: 阿里云
- tencent: 腾讯云
- aws: AWS
- jdcloud: 京东云
- baidu: 百度云
- ksyun: 金山云

## 返回数据

- `accounts`: 账号列表（id, name, provider, providerName, region, status, statusText）
- `total`: 账号总数
