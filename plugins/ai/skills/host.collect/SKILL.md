---
name: host.collect
description: 触发主机信息采集，重新采集指定主机或分组的系统信息（CPU、内存、磁盘、网络等）
category: host
riskLevel: medium
scriptType: builtin
parameters:
  type: object
  properties:
    host_ids:
      type: array
      items:
        type: integer
      description: 要采集的主机 ID 列表
    group_name:
      type: string
      description: 按分组名称采集，采集该分组下所有主机
    ip:
      type: string
      description: 指定单台主机 IP 进行采集
---

# 主机信息采集

触发对指定主机的系统信息重新采集。

## 使用场景

- 用户说"重新采集所有主机信息"
- 用户说"采集北京分组的主机"
- 新添加主机后需要采集信息

## 注意事项

- 采集操作需要 SSH 连接目标主机
- 采集过程可能需要几秒到几分钟
- 执行的是固定系统信息采集命令，属于低风险操作，直接执行
- 返回结果用于辅助分析，不会在本函数内修改主机配置
