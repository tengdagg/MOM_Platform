# Kubernetes 插件

iom Kubernetes 集群管理插件

## 目录结构

```
plugins/kubernetes/
├── biz/                    # 业务逻辑层
│   └── cluster.go         # 集群业务逻辑
├── data/                   # 数据层
│   ├── models/            # 数据模型
│   │   └── cluster.go     # 集群模型
│   └── repository/        # 数据访问层
│       └── cluster_repository.go  # 集群数据访问
├── service/                # 服务层
│   └── cluster_service.go # 集群服务
├── server/                 # HTTP 层
│   ├── cluster_handler.go # 集群 HTTP 处理器
│   └── router.go          # 路由注册
├── plugin.go              # 插件主入口
├── go.mod                 # Go 模块文件
└── README.md              # 说明文档
```

## 功能特性

### 1. 集群管理
- 创建/更新/删除集群
- 集群列表查询
- 集群详情查看
- 集群连接测试
- KubeConfig 加密存储

### 2. 安全特性
- KubeConfig AES-GCM 加密存储
- 软删除机制
- 连接状态监控

### 3. 云厂商支持
- 自建集群 (native)
- AWS EKS
- 阿里云 ACK
- 腾讯云 TKE

## API 接口

### 集群管理接口

- `POST /api/v1/kubernetes/clusters` - 创建集群
- `GET /api/v1/kubernetes/clusters` - 获取集群列表
- `GET /api/v1/kubernetes/clusters/:id` - 获取集群详情
- `PUT /api/v1/kubernetes/clusters/:id` - 更新集群
- `DELETE /api/v1/kubernetes/clusters/:id` - 删除集群
- `POST /api/v1/kubernetes/clusters/:id/test` - 测试集群连接

### 请求示例

#### 创建集群
```json
{
  "name": "prod-cluster",
  "alias": "生产环境集群",
  "apiEndpoint": "https://k8s-api.example.com",
  "kubeConfig": "base64 encoded kubeconfig",
  "region": "cn-beijing",
  "provider": "aliyun",
  "description": "生产环境 Kubernetes 集群"
}
```

## 数据库表

### k8s_clusters

| 字段         | 类型         | 说明           |
|------------|------------|--------------|
| id         | uint       | 主键          |
| name       | string     | 集群名称（唯一）    |
| alias      | string     | 集群别名         |
| api_endpoint | string    | API Server 地址 |
| kube_config | text       | KubeConfig（加密） |
| version    | string     | Kubernetes 版本  |
| status     | int        | 状态：1-正常 2-失败 |
| region     | string     | 区域           |
| provider   | string     | 云服务商         |
| description| string     | 描述           |
| created_by | uint       | 创建人ID        |
| created_at | time       | 创建时间         |
| updated_at | time       | 更新时间         |
| is_deleted | boolean    | 是否删除（软删除）    |

## 使用说明

### 1. 在主项目中注册插件

编辑 `internal/server/http.go`：

```go
import (
    k8splugin "github.com/ydcloud-dy/iom/plugins/kubernetes"
)

func NewHTTPServer(conf *conf.Config, svc *service.Service, db *gorm.DB) *HTTPServer {
    // ...

    // 创建插件管理器
    pluginMgr := plugin.NewManager(db)

    // 注册 Kubernetes 插件
    if err := pluginMgr.Register(k8splugin.New()); err != nil {
        appLogger.Error("注册Kubernetes插件失败", zap.Error(err))
    }

    // ...
}
```

### 2. 添加本地依赖

在主项目的 `go.mod` 中添加：

```
replace github.com/ydcloud-dy/iom/plugins/kubernetes => ./plugins/kubernetes
```

### 3. 运行迁移

插件会在初始化时自动创建数据表。

## 开发说明

### 添加新功能

1. 在 `biz/` 添加业务逻辑
2. 在 `service/` 添加服务层
3. 在 `server/` 添加 HTTP 处理器
4. 在 `server/router.go` 注册路由
5. 更新插件菜单（如果需要）

### 依赖管理

插件使用 `replace` 指令引用主项目，确保与主项目版本一致。

## 扩展性

插件设计遵循以下原则：

1. **可插拔**：完全独立，可随时启用/禁用
2. **低耦合**：通过接口与主项目交互
3. **易扩展**：清晰的分层架构，便于添加新功能
4. **安全性**：敏感数据加密存储

## License

Copyright iom Team
