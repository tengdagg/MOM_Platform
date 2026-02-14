---
name: cloud.import_hosts
description: 从云平台导入主机实例到 MOM 平台，支持阿里云、腾讯云、AWS 等
category: cloud
riskLevel: medium
scriptType: builtin
parameters:
  type: object
  properties:
    account_id:
      type: integer
      description: 云账号 ID
    provider:
      type: string
      description: "云厂商: aliyun/tencent/aws/huawei"
    region:
      type: string
      description: 区域 ID
    group_name:
      type: string
      description: 导入到的主机分组名称
    instance_ids:
      type: array
      items:
        type: string
      description: 指定实例 ID 列表（不指定则导入该区域所有实例）
    confirmed:
      type: boolean
      description: 用户确认执行时设为 true，首次调用不传此参数
  required:
    - account_id
---

# 云主机导入

从云平台批量导入主机实例到 MOM 平台进行统一管理。

## 使用场景

- 批量导入云主机到平台进行统一管理
- 将现有云资源纳入运维监控体系
- 快速同步云平台的主机信息
- 按区域或实例选择性导入主机

## 支持的云厂商

### aliyun

阿里云（Alibaba Cloud）。

- 支持 ECS 实例导入
- 自动获取实例的公网 IP、私网 IP、规格等信息
- 支持按地域（Region）筛选

### tencent

腾讯云（Tencent Cloud）。

- 支持 CVM 实例导入
- 自动获取实例的网络配置、安全组等信息
- 支持按地域筛选

### aws

亚马逊云（Amazon Web Services）。

- 支持 EC2 实例导入
- 自动获取实例的标签、安全组等信息
- 支持按区域（Region）筛选

### huawei

华为云（Huawei Cloud）。

- 支持 ECS 实例导入
- 自动获取实例的配置信息
- 支持按区域筛选

## 参数说明

### 必需参数

- `account_id`: 云账号 ID，指定要使用的云平台账号配置

### 可选参数

- `provider`: 云厂商类型，如果不提供则从 account_id 对应的账号配置中获取
- `region`: 区域 ID，指定要导入的区域，不提供则导入所有区域
- `group_name`: 主机分组名称，导入的主机会自动添加到指定分组
- `instance_ids`: 实例 ID 列表，指定要导入的实例，不提供则导入该区域所有实例

## 导入流程

1. 验证云账号配置和权限
2. 连接云平台 API 获取实例列表
3. 根据 region 和 instance_ids 筛选目标实例
4. 获取实例详细信息（IP、规格、状态等）
5. 将实例信息导入到 MOM 平台
6. 如果指定了 group_name，将主机添加到对应分组
7. 返回导入结果统计

## 导入信息

导入的主机会自动获取以下信息：

- **实例 ID**: 云平台的实例标识
- **实例名称**: 实例的显示名称
- **公网 IP**: 公网访问 IP 地址
- **私网 IP**: 内网 IP 地址
- **实例规格**: CPU、内存等配置信息
- **操作系统**: 操作系统类型和版本
- **实例状态**: 运行中、已停止等状态
- **创建时间**: 实例创建时间
- **区域信息**: 所属区域和可用区

## 使用示例

**导入指定区域的所有实例：**
- account_id: 1
- provider: aliyun
- region: cn-hangzhou
- group_name: production

**导入指定实例列表：**
- account_id: 1
- instance_ids: ["i-1234567890abcdef0", "i-0987654321fedcba0"]
- group_name: staging

**导入所有区域的实例：**
- account_id: 1
- provider: tencent
- group_name: all-cloud-hosts

## 注意事项

- 确保云账号配置正确且具有查询实例的权限
- 导入前建议先查询实例列表确认要导入的实例
- 已存在的实例（相同 instance_id）不会重复导入
- 导入的主机默认状态为"未初始化"，需要后续配置
- 批量导入大量实例时可能需要较长时间
- 建议按区域分批导入，避免一次性导入过多实例
- 导入的主机需要配置凭据后才能执行操作

## 安全策略

- 需要用户具有云账号访问权限
- 导入操作会记录到审计日志
- 支持按实例 ID 精确控制导入范围
- 导入的主机信息来自云平台，确保数据准确性
