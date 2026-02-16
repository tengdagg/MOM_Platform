package rbac

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// SysAssetAuthorization 资产授权规则
// 参考 JumpServer 三维度授权模型：用户/部门 -> 资产分组/具体资产 -> 操作权限
type SysAssetAuthorization struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Name          string         `gorm:"type:varchar(100);not null;comment:授权名称" json:"name"`
	// --- 用户维度 ---
	UserIDs       UintArray      `gorm:"type:json;comment:授权用户ID列表" json:"userIds"`
	DepartmentIDs UintArray      `gorm:"type:json;comment:授权部门ID列表" json:"departmentIds"`
	// --- 资产维度 ---
	AssetGroupID  uint           `gorm:"not null;index;comment:资产分组/节点ID" json:"assetGroupId"`
	AssetType     string         `gorm:"type:varchar(20);default:'host';comment:资产类型 host/network_device" json:"assetType"`
	AssetIDs      UintArray      `gorm:"type:json;comment:具体资产ID列表(空=整个分组)" json:"assetIds"`
	// --- 动作维度 ---
	Permissions   uint           `gorm:"type:int unsigned;default:1;comment:权限bitmask" json:"permissions"`
	// --- 有效期 ---
	StartDate     *time.Time     `gorm:"comment:生效时间" json:"startDate"`
	ExpireDate    *time.Time     `gorm:"comment:失效时间" json:"expireDate"`
	IsActive      bool           `gorm:"default:true;comment:是否启用" json:"isActive"`
	Description   string         `gorm:"type:varchar(500);comment:备注" json:"description"`
}

// TableName 指定表名
func (SysAssetAuthorization) TableName() string {
	return "sys_asset_authorization"
}

// ---- DTO / VO ----

// AssetAuthorizationInfo 授权规则信息（用于列表展示）
type AssetAuthorizationInfo struct {
	ID              uint       `json:"id"`
	Name            string     `json:"name"`
	UserIDs         UintArray  `json:"userIds"`
	UserNames       []string   `json:"userNames"`
	DepartmentIDs   UintArray  `json:"departmentIds"`
	DepartmentNames []string   `json:"departmentNames"`
	AssetGroupID    uint       `json:"assetGroupId"`
	AssetGroupName  string     `json:"assetGroupName"`
	AssetType       string     `json:"assetType"`
	AssetIDs        UintArray  `json:"assetIds"`
	AssetNames      []string   `json:"assetNames"`
	IsAllAssets     bool       `json:"isAllAssets"`
	Permissions     uint       `json:"permissions"`
	StartDate       *time.Time `json:"startDate"`
	ExpireDate      *time.Time `json:"expireDate"`
	IsActive        bool       `json:"isActive"`
	Description     string     `json:"description"`
	CreatedAt       time.Time  `json:"createdAt"`
}

// AssetAuthorizationCreateReq 创建授权请求
type AssetAuthorizationCreateReq struct {
	Name          string     `json:"name" binding:"required"`
	UserIDs       []uint     `json:"userIds"`
	DepartmentIDs []uint     `json:"departmentIds"`
	AssetGroupID  uint       `json:"assetGroupId" binding:"required"`
	AssetType     string     `json:"assetType"`
	AssetIDs      []uint     `json:"assetIds"`
	Permissions   uint       `json:"permissions"`
	StartDate     *time.Time `json:"startDate"`
	ExpireDate    *time.Time `json:"expireDate"`
	IsActive      *bool      `json:"isActive"`
	Description   string     `json:"description"`
}

// AssetAuthorizationUpdateReq 更新授权请求
type AssetAuthorizationUpdateReq struct {
	Name          string     `json:"name"`
	UserIDs       []uint     `json:"userIds"`
	DepartmentIDs []uint     `json:"departmentIds"`
	AssetGroupID  uint       `json:"assetGroupId"`
	AssetType     string     `json:"assetType"`
	AssetIDs      []uint     `json:"assetIds"`
	Permissions   uint       `json:"permissions"`
	StartDate     *time.Time `json:"startDate"`
	ExpireDate    *time.Time `json:"expireDate"`
	IsActive      *bool      `json:"isActive"`
	Description   string     `json:"description"`
}

// AssetAuthorizationDetailVO 授权详情（用于编辑回显）
type AssetAuthorizationDetailVO struct {
	ID              uint       `json:"id"`
	Name            string     `json:"name"`
	UserIDs         []uint     `json:"userIds"`
	DepartmentIDs   []uint     `json:"departmentIds"`
	AssetGroupID    uint       `json:"assetGroupId"`
	AssetGroupName  string     `json:"assetGroupName"`
	AssetType       string     `json:"assetType"`
	AssetIDs        []uint     `json:"assetIds"`
	Permissions     uint       `json:"permissions"`
	StartDate       *time.Time `json:"startDate"`
	ExpireDate      *time.Time `json:"expireDate"`
	IsActive        bool       `json:"isActive"`
	Description     string     `json:"description"`
	CreatedAt       time.Time  `json:"createdAt"`
}

// ---- Repository Interface ----

// AssetAuthorizationRepo 资产授权仓储接口
type AssetAuthorizationRepo interface {
	// CRUD
	Create(ctx context.Context, auth *SysAssetAuthorization) error
	Update(ctx context.Context, auth *SysAssetAuthorization) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*SysAssetAuthorization, error)
	GetDetailByID(ctx context.Context, id uint) (*AssetAuthorizationDetailVO, error)
	// 列表查询
	List(ctx context.Context, page, pageSize int, assetGroupID *uint, keyword string) ([]*AssetAuthorizationInfo, int64, error)
	// 权限检查（按用户+部门匹配）
	CheckHostPermission(ctx context.Context, userID, hostID uint) (bool, error)
	CheckHostOperationPermission(ctx context.Context, userID, hostID uint, operation uint) (bool, error)
	GetUserHostPermissions(ctx context.Context, userID, hostID uint) (uint, error)
	GetUserAccessibleHostIDs(ctx context.Context, userID uint) ([]uint, error)
	CheckNetworkDeviceOperationPermission(ctx context.Context, userID, deviceID uint, operation uint) (bool, error)
	GetUserNetworkDevicePermissions(ctx context.Context, userID, deviceID uint) (uint, error)
	GetUserAccessibleNetworkDeviceIDs(ctx context.Context, userID uint) ([]uint, error)
}

// ---- UseCase ----

// AssetAuthorizationUseCase 资产授权业务逻辑
type AssetAuthorizationUseCase struct {
	repo AssetAuthorizationRepo
}

func NewAssetAuthorizationUseCase(repo AssetAuthorizationRepo) *AssetAuthorizationUseCase {
	return &AssetAuthorizationUseCase{repo: repo}
}

func (uc *AssetAuthorizationUseCase) Create(ctx context.Context, auth *SysAssetAuthorization) error {
	return uc.repo.Create(ctx, auth)
}

func (uc *AssetAuthorizationUseCase) Update(ctx context.Context, auth *SysAssetAuthorization) error {
	return uc.repo.Update(ctx, auth)
}

func (uc *AssetAuthorizationUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *AssetAuthorizationUseCase) GetByID(ctx context.Context, id uint) (*SysAssetAuthorization, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *AssetAuthorizationUseCase) GetDetailByID(ctx context.Context, id uint) (*AssetAuthorizationDetailVO, error) {
	return uc.repo.GetDetailByID(ctx, id)
}

func (uc *AssetAuthorizationUseCase) List(ctx context.Context, page, pageSize int, assetGroupID *uint, keyword string) ([]*AssetAuthorizationInfo, int64, error) {
	return uc.repo.List(ctx, page, pageSize, assetGroupID, keyword)
}

func (uc *AssetAuthorizationUseCase) CheckHostPermission(ctx context.Context, userID, hostID uint) (bool, error) {
	return uc.repo.CheckHostPermission(ctx, userID, hostID)
}

func (uc *AssetAuthorizationUseCase) CheckHostOperationPermission(ctx context.Context, userID, hostID uint, operation uint) (bool, error) {
	return uc.repo.CheckHostOperationPermission(ctx, userID, hostID, operation)
}

func (uc *AssetAuthorizationUseCase) GetUserHostPermissions(ctx context.Context, userID, hostID uint) (uint, error) {
	return uc.repo.GetUserHostPermissions(ctx, userID, hostID)
}

func (uc *AssetAuthorizationUseCase) GetUserAccessibleHostIDs(ctx context.Context, userID uint) ([]uint, error) {
	return uc.repo.GetUserAccessibleHostIDs(ctx, userID)
}

func (uc *AssetAuthorizationUseCase) CheckNetworkDeviceOperationPermission(ctx context.Context, userID, deviceID uint, operation uint) (bool, error) {
	return uc.repo.CheckNetworkDeviceOperationPermission(ctx, userID, deviceID, operation)
}

func (uc *AssetAuthorizationUseCase) GetUserNetworkDevicePermissions(ctx context.Context, userID, deviceID uint) (uint, error) {
	return uc.repo.GetUserNetworkDevicePermissions(ctx, userID, deviceID)
}

func (uc *AssetAuthorizationUseCase) GetUserAccessibleNetworkDeviceIDs(ctx context.Context, userID uint) ([]uint, error) {
	return uc.repo.GetUserAccessibleNetworkDeviceIDs(ctx, userID)
}
