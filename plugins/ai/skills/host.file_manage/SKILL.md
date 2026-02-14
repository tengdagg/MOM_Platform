---
name: host.file_manage
description: 管理远程主机文件，支持查看目录、上传文件、下载文件操作
category: host
riskLevel: high
scriptType: builtin
parameters:
  type: object
  properties:
    host_id:
      type: integer
      description: 目标主机 ID
    ip:
      type: string
      description: 目标主机 IP（与 host_id 二选一）
    action:
      type: string
      description: "操作类型: list(列出目录) / download(下载文件) / upload(上传文件)"
    path:
      type: string
      description: 远程文件或目录路径
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - action
    - path
---

# 远程文件管理

管理远程主机上的文件和目录。

## 使用场景

- 用户说"查看 192.168.1.10 的 /var/log/ 目录"
- 用户说"从服务器下载 /var/log/syslog"
- 用户说"列出主机上的配置文件"

## 注意事项

- 需要主机 SSH 连接权限
- 大文件传输可能较慢
- 下载操作会返回文件内容摘要
