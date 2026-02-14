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
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
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
- 会更新主机的 CPU、内存、磁盘、网络等信息
