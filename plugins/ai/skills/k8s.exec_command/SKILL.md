---
name: k8s.exec_command
description: 在指定 Kubernetes Pod 容器内部执行 Linux/Shell 命令；适用于查看容器内文件、进程、环境变量、网络状态和连续 shell 排障，不用于查询或修改 Kubernetes 资源对象
category: k8s
riskLevel: critical
scriptType: builtin
parameters:
  type: object
  properties:
    cluster_id:
      type: integer
      description: 集群 ID
    cluster_name:
      type: string
      description: 集群名称（与 cluster_id 二选一）
    namespace:
      type: string
      description: Pod 所在命名空间，默认 default
    pod_name:
      type: string
      description: Pod 名称
    container:
      type: string
      description: 容器名称；多容器 Pod 时必填
    command:
      type: string
      description: 要在容器内执行的命令
    timeout:
      type: integer
      description: 一次性执行超时时间（秒），默认 60，最大 600
      default: 60
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - command
    - pod_name
---

# Pod 内命令执行

在指定 Pod 容器内执行命令，适合容器排障、查看配置、检查进程、验证环境变量等场景。

## 使用场景

- "查看 default 命名空间下 nginx Pod 的环境变量"
- "进入容器后切到 /app 再继续排查"
- "在 Pod 里查看配置文件和进程状态"
- "连续执行 cd /app、ls、cat、printenv 这类上下文相关命令"

## 选择规则

遇到以下需求，优先选择 `k8s.exec_command`：

- 用户明确说"进入 Pod"、"进入容器"、"到容器里执行"
- 用户要看容器内文件，例如 `/etc/nginx/nginx.conf`、`/app/config.yaml`
- 用户要看容器内进程、环境变量、目录、挂载、网络命名空间信息
- 用户要求连续执行 `cd` / `export` / `source` / `bash` 这类依赖 shell 上下文的命令

以下场景不要优先选它，应改用 `k8s.kubectl`：

- 查看 Pod / Deployment / Service / Node / PVC 等资源对象
- 查看资源事件、资源 YAML、资源状态
- 查看标准容器日志（stdout/stderr）
- 扩缩容、重启、删除、cordon、drain 等 K8s 资源变更

快速判断：

- 如果用户说"进入 Pod / 进入容器 / 到容器里执行"，直接优先 `k8s.exec_command`
- 如果用户目标路径是容器内文件，例如 `/app`、`/etc/nginx/nginx.conf`、`/var/log/...`，优先 `k8s.exec_command`
- 如果用户问的是资源对象状态、事件、YAML、副本数、Service/Ingress 配置，不要先用本 Skill，应切到 `k8s.kubectl`
- 如果用户只是想看 Pod 标准日志，不要误用本 Skill

## 执行模型

1. 普通单条命令默认使用一次性 `pods/exec`
2. `cd`、`export`、`source`、单独 `bash/sh` 等依赖 shell 上下文的命令，会自动复用当前对话中的 Pod shell 会话
3. 如果当前对话里同一个 Pod/容器已经存在活动会话，后续命令会继续在该上下文中执行
4. 空闲约 10 分钟后会自动关闭，也可以用 `k8s.close_session` 主动结束

## 风险说明

- 只读查看类命令会直接执行，例如 `ls`、`pwd`、`ps`、`cat`、`grep`、`env`、`df`、`ip addr show`
- 依赖上下文但本身无破坏性的命令，例如 `cd`、`export`、`source`、`bash`，会直接进入或复用 Pod shell 会话
- 可能修改容器文件、进程或运行状态的命令会进入人工确认流程
- 不适合执行 `top`、`htop`、`less`、`vi`、`nano`、`watch`、`tail -f`、嵌套 `ssh/mysql/psql` 等持续交互命令

## 注意事项

- 多容器 Pod 时请显式指定 `container`
- 该 Skill 依赖容器内存在可用 shell；若容器是 distroless 等无 shell 镜像，可能无法执行
- 进入交互会话后，当前目录和环境变量会被保留到后续同一对话的命令中
- 可用 `k8s.session_status` 查看当前对话中的 Pod 会话状态
- 查看应用标准日志优先使用 `k8s.kubectl(action="logs")`，只有明确要看容器内某个日志文件时才使用本 Skill
