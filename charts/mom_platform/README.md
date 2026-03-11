# mom Helm Chart

mom 的官方 Helm Chart，用于在 Kubernetes 上部署 mom 运维管理平台。

## 前置条件

- Kubernetes 1.24+
- Helm 3.0+
- PV provisioner（如需持久化存储）
- Ingress Controller（如需外部访问）

## 安装

### 方式一：本地安装

```bash

# 克隆项目
git clone https://github.com/ydcloud-dy/mom.git
cd mom

# 使用默认配置安装
helm install mom ./charts/mom_platform \
  --namespace mom \
  --create-namespace

# 使用自定义配置安装
helm install mom ./charts/mom_platform \
  --namespace mom \
  --create-namespace \
  -f my-values.yaml
```

### 方式二：指定参数安装

```bash
helm install mom ./charts/mom_platform \
  --namespace mom \
  --create-namespace \
  --set ingress.hosts[0].host=mom.mycompany.com \
  --set mysql.auth.rootPassword=MySecurePassword \
  --set server.jwtSecret=my-jwt-secret-key \
  --set security.credentialEncryptionKey=0123456789abcdef0123456789abcdef \
  --set security.k8sEncryptionKey=fedcba9876543210fedcba9876543210
```

## 卸载

```bash
helm uninstall mom -n mom
kubectl delete namespace mom
```

## 配置参数

### 全局配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `global.storageClass` | 全局存储类 | `""` |
| `global.imagePullPolicy` | 镜像拉取策略 | `IfNotPresent` |
| `global.imagePullSecrets` | 镜像拉取密钥 | `[]` |

### 后端配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `backend.replicaCount` | 副本数 | `2` |
| `backend.image.repository` | 镜像仓库 | `ydcloud/mom-backend` |
| `backend.image.tag` | 镜像标签 | `latest` |
| `backend.resources.requests.memory` | 内存请求 | `256Mi` |
| `backend.resources.requests.cpu` | CPU 请求 | `100m` |
| `backend.resources.limits.memory` | 内存限制 | `512Mi` |
| `backend.resources.limits.cpu` | CPU 限制 | `500m` |

### 前端配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `frontend.replicaCount` | 副本数 | `2` |
| `frontend.image.repository` | 镜像仓库 | `ydcloud/mom-frontend` |
| `frontend.image.tag` | 镜像标签 | `latest` |
| `frontend.resources.requests.memory` | 内存请求 | `64Mi` |
| `frontend.resources.requests.cpu` | CPU 请求 | `50m` |

### MySQL 配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `mysql.enabled` | 是否启用内置 MySQL | `true` |
| `mysql.auth.rootPassword` | root 密码 | `mom@2024` |
| `mysql.auth.database` | 数据库名 | `mom` |
| `mysql.persistence.enabled` | 是否启用持久化 | `true` |
| `mysql.persistence.size` | 存储大小 | `20Gi` |

### Redis 配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `redis.enabled` | 是否启用内置 Redis | `true` |
| `redis.auth.password` | 密码 | `mom@Redis` |
| `redis.persistence.enabled` | 是否启用持久化 | `false` |

### 外部数据库配置

当 `mysql.enabled=false` 时使用：

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `externalDatabase.host` | 主机地址 | `""` |
| `externalDatabase.port` | 端口 | `3306` |
| `externalDatabase.database` | 数据库名 | `mom` |
| `externalDatabase.username` | 用户名 | `root` |
| `externalDatabase.password` | 密码 | `""` |

### 外部 Redis 配置

当 `redis.enabled=false` 时使用：

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `externalRedis.host` | 主机地址 | `""` |
| `externalRedis.port` | 端口 | `6379` |
| `externalRedis.password` | 密码 | `""` |

### Guacd / RDP 配置

用于 Windows RDP 远程桌面代理。默认以 `guacd` sidecar 方式跟随 backend 一起部署，backend 通过 `127.0.0.1:4822` 访问。

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `guacd.enabled` | 是否启用内置 guacd sidecar | `true` |
| `guacd.image.repository` | guacd 镜像仓库 | `registry.cn-hangzhou.aliyuncs.com/registry_dat/guacd` |
| `guacd.image.tag` | guacd 镜像标签 | `latest` |
| `guacd.port` | sidecar guacd 监听端口 | `4822` |
| `guacd.externalHost` | 外部 guacd 地址（当 `guacd.enabled=false` 时使用） | `""` |
| `guacd.externalPort` | 外部 guacd 端口 | `4822` |
| `guacd.drive.enabled` | 是否启用共享 RDP 文件传输目录 | `true` |
| `guacd.drive.path` | backend 与 guacd 共享目录挂载路径 | `/tmp/guacd-drive` |
| `guacd.drive.sizeLimit` | 共享目录大小限制 | `512Mi` |

### 服务器配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `server.mode` | 运行模式 | `release` |
| `server.httpPort` | HTTP 端口 | `9876` |
| `server.jwtSecret` | JWT 密钥 | `mom-jwt-secret-...` |
| `server.jwtExpire` | JWT 过期时间 | `24h` |

### 安全配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `security.credentialEncryptionKey` | 资产凭据/任务模块加解密密钥，必须为 32 字节；默认使用历史兼容值 | `mom-encrypt-key-32bytes-long!!@@` |
| `security.k8sEncryptionKey` | Kubernetes kubeconfig 加解密密钥，必须为 32 字节；默认使用历史兼容值 | `mom-k8s-encrypt-key-32byte!!@@!!` |

### Ingress 配置

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `ingress.enabled` | 是否启用 Ingress | `true` |
| `ingress.className` | Ingress 类名 | `nginx` |
| `ingress.hosts[0].host` | 主机域名 | `mom.example.com` |
| `ingress.tls` | TLS 配置 | `[]` |

## 常见配置示例

### 使用外部数据库

```yaml
mysql:
  enabled: false

externalDatabase:
  host: mysql.example.com
  port: 3306
  database: mom
  username: mom
  password: your-password
```

### 启用 HTTPS

```yaml
ingress:
  enabled: true
  hosts:
    - host: mom.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: mom-tls
      hosts:
        - mom.example.com
```

### 生产环境配置

```yaml
backend:
  replicaCount: 3
  resources:
    requests:
      memory: "512Mi"
      cpu: "200m"
    limits:
      memory: "1Gi"
      cpu: "1000m"

frontend:
  replicaCount: 3

mysql:
  persistence:
    size: 100Gi

server:
  jwtSecret: "your-very-long-random-secret-key"

security:
  credentialEncryptionKey: "0123456789abcdef0123456789abcdef"
  k8sEncryptionKey: "fedcba9876543210fedcba9876543210"
```

### 使用内置 guacd 支持 RDP

默认配置即支持，无需额外设置。安装后 backend Pod 中会自动包含一个 `guacd` sidecar，并与 backend 共享 `/tmp/guacd-drive`，满足远程桌面代理和文件传输目录需求。

```yaml
guacd:
  enabled: true
  drive:
    enabled: true
    sizeLimit: 1Gi
```

### 使用外部 guacd

如果你已经有独立部署的 `guacd` 服务，可关闭 sidecar，改为连接外部地址：

```yaml
guacd:
  enabled: false
  externalHost: guacd.default.svc.cluster.local
  externalPort: 4822
```

## 升级

```bash
helm upgrade mom ./charts/mom_platform -n mom -f values.yaml
```

## 故障排查

```bash
# 查看 Pod 状态
kubectl get pods -n mom

# 查看 Pod 日志
kubectl logs -f deployment/mom-backend -n mom
kubectl logs -f deployment/mom-frontend -n mom

# 查看 Pod 详情
kubectl describe pod <pod-name> -n mom
```
