package rbac

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ydcloud-dy/mom/internal/biz/rbac"
	"gorm.io/gorm"
)

type assetAuthorizationRepo struct {
	db *gorm.DB
}

// NewAssetAuthorizationRepo 创建资产授权仓储
func NewAssetAuthorizationRepo(db *gorm.DB) rbac.AssetAuthorizationRepo {
	return &assetAuthorizationRepo{db: db}
}

// Create 创建授权规则
func (r *assetAuthorizationRepo) Create(ctx context.Context, auth *rbac.SysAssetAuthorization) error {
	return r.db.WithContext(ctx).Create(auth).Error
}

// Update 更新授权规则
func (r *assetAuthorizationRepo) Update(ctx context.Context, auth *rbac.SysAssetAuthorization) error {
	return r.db.WithContext(ctx).Save(auth).Error
}

// Delete 删除授权规则
func (r *assetAuthorizationRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&rbac.SysAssetAuthorization{}, id).Error
}

// GetByID 根据 ID 获取授权规则
func (r *assetAuthorizationRepo) GetByID(ctx context.Context, id uint) (*rbac.SysAssetAuthorization, error) {
	var auth rbac.SysAssetAuthorization
	if err := r.db.WithContext(ctx).First(&auth, id).Error; err != nil {
		return nil, err
	}
	return &auth, nil
}

// GetDetailByID 获取授权规则详情（用于编辑回显）
func (r *assetAuthorizationRepo) GetDetailByID(ctx context.Context, id uint) (*rbac.AssetAuthorizationDetailVO, error) {
	var auth rbac.SysAssetAuthorization
	if err := r.db.WithContext(ctx).First(&auth, id).Error; err != nil {
		return nil, err
	}

	var groupName string
	r.db.WithContext(ctx).Table("asset_group").Select("name").Where("id = ?", auth.AssetGroupID).Scan(&groupName)

	assetType := auth.AssetType
	if assetType == "" {
		assetType = "host"
	}

	return &rbac.AssetAuthorizationDetailVO{
		ID:             auth.ID,
		Name:           auth.Name,
		UserIDs:        []uint(auth.UserIDs),
		DepartmentIDs:  []uint(auth.DepartmentIDs),
		AssetGroupID:   auth.AssetGroupID,
		AssetGroupName: groupName,
		AssetType:      assetType,
		AssetIDs:       []uint(auth.AssetIDs),
		Permissions:    auth.Permissions,
		StartDate:      auth.StartDate,
		ExpireDate:     auth.ExpireDate,
		IsActive:       auth.IsActive,
		Description:    auth.Description,
		CreatedAt:      auth.CreatedAt,
	}, nil
}

// List 分页查询授权规则列表
func (r *assetAuthorizationRepo) List(ctx context.Context, page, pageSize int, assetGroupID *uint, keyword string) ([]*rbac.AssetAuthorizationInfo, int64, error) {
	var results []rbac.SysAssetAuthorization
	var total int64

	query := r.db.WithContext(ctx).Model(&rbac.SysAssetAuthorization{}).Where("deleted_at IS NULL")

	if assetGroupID != nil && *assetGroupID > 0 {
		query = query.Where("asset_group_id = ?", *assetGroupID)
	}
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&results).Error; err != nil {
		return nil, 0, err
	}

	// 批量查询关联名称
	var infos []*rbac.AssetAuthorizationInfo
	for _, auth := range results {
		info := &rbac.AssetAuthorizationInfo{
			ID:              auth.ID,
			Name:            auth.Name,
			UserIDs:         auth.UserIDs,
			UserNames:       []string{},
			DepartmentIDs:   auth.DepartmentIDs,
			DepartmentNames: []string{},
			AssetGroupID:    auth.AssetGroupID,
			AssetType:       auth.AssetType,
			AssetIDs:        auth.AssetIDs,
			AssetNames:      []string{},
			IsAllAssets:     len(auth.AssetIDs) == 0,
			Permissions:     auth.Permissions,
			StartDate:       auth.StartDate,
			ExpireDate:      auth.ExpireDate,
			IsActive:        auth.IsActive,
			Description:     auth.Description,
			CreatedAt:       auth.CreatedAt,
		}
		if info.AssetType == "" {
			info.AssetType = "host"
		}

		// 查询分组名
		r.db.WithContext(ctx).Table("asset_group").Select("name").Where("id = ?", auth.AssetGroupID).Scan(&info.AssetGroupName)

		// 查询用户名（处理 real_name 为空字符串的情况）
		if len(auth.UserIDs) > 0 {
			r.db.WithContext(ctx).Table("sys_user").
				Where("id IN ?", []uint(auth.UserIDs)).
				Pluck("IF(real_name IS NOT NULL AND real_name != '', real_name, username)", &info.UserNames)
		}
		// 查询部门名
		if len(auth.DepartmentIDs) > 0 {
			r.db.WithContext(ctx).Table("sys_department").
				Where("id IN ?", []uint(auth.DepartmentIDs)).
				Pluck("name", &info.DepartmentNames)
		}
		// 查询资产名
		if len(auth.AssetIDs) > 0 {
			if info.AssetType == "network_device" {
				r.db.WithContext(ctx).Table("network_devices").
					Where("id IN ? AND deleted_at IS NULL", []uint(auth.AssetIDs)).
					Pluck("CONCAT(name, ' (', ip, ')')", &info.AssetNames)
			} else {
				r.db.WithContext(ctx).Table("hosts").
					Where("id IN ? AND deleted_at IS NULL", []uint(auth.AssetIDs)).
					Pluck("CONCAT(name, ' (', ip, ')')", &info.AssetNames)
			}
		}

		infos = append(infos, info)
	}

	return infos, total, nil
}

// ---- 权限检查：通用辅助方法 ----

// isAdmin 检查用户是否为管理员
func (r *assetAuthorizationRepo) isAdmin(ctx context.Context, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("sys_user_role AS ur").
		Joins("JOIN sys_role AS r ON ur.role_id = r.id").
		Where("ur.user_id = ? AND r.code = ?", userID, "admin").
		Count(&count).Error
	return count > 0, err
}

// getUserDepartmentID 获取用户的部门ID
func (r *assetAuthorizationRepo) getUserDepartmentID(ctx context.Context, userID uint) (uint, error) {
	var deptID uint
	err := r.db.WithContext(ctx).
		Table("sys_user").
		Select("department_id").
		Where("id = ?", userID).
		Scan(&deptID).Error
	return deptID, err
}

// userMatchCondition 返回匹配用户或部门的 SQL 条件片段
// 匹配逻辑：user_ids 包含该用户 OR department_ids 包含该用户的部门
func (r *assetAuthorizationRepo) userMatchCondition() string {
	return `(JSON_CONTAINS(a.user_ids, CAST(? AS JSON)) OR JSON_CONTAINS(a.department_ids, CAST(? AS JSON)))`
}

// activeCondition 返回授权规则有效性检查的 SQL 条件
func (r *assetAuthorizationRepo) activeCondition() string {
	return `a.deleted_at IS NULL AND a.is_active = true
		AND (a.start_date IS NULL OR a.start_date <= NOW())
		AND (a.expire_date IS NULL OR a.expire_date > NOW())`
}

// ---- 主机权限检查 ----

// CheckHostPermission 检查用户是否有访问指定主机的权限
func (r *assetAuthorizationRepo) CheckHostPermission(ctx context.Context, userID, hostID uint) (bool, error) {
	admin, err := r.isAdmin(ctx, userID)
	if err != nil {
		return false, err
	}
	if admin {
		return true, nil
	}

	deptID, _ := r.getUserDepartmentID(ctx, userID)

	var groupID uint
	r.db.WithContext(ctx).Table("hosts").Select("group_id").Where("id = ?", hostID).Scan(&groupID)

	var count int64
	sql := fmt.Sprintf(`
		SELECT COUNT(*) FROM sys_asset_authorization a
		WHERE %s
		AND %s
		AND a.asset_group_id = ?
		AND COALESCE(a.asset_type, 'host') = 'host'
		AND (JSON_LENGTH(COALESCE(a.asset_ids, JSON_ARRAY())) = 0 OR JSON_CONTAINS(a.asset_ids, CAST(? AS JSON)))
	`, r.activeCondition(), r.userMatchCondition())

	err = r.db.WithContext(ctx).Raw(sql, userID, deptID, groupID, hostID).Scan(&count).Error
	return count > 0, err
}

// CheckHostOperationPermission 检查用户是否有对指定主机的特定操作权限
func (r *assetAuthorizationRepo) CheckHostOperationPermission(ctx context.Context, userID, hostID uint, operation uint) (bool, error) {
	admin, err := r.isAdmin(ctx, userID)
	if err != nil {
		return false, err
	}
	if admin {
		return true, nil
	}

	deptID, _ := r.getUserDepartmentID(ctx, userID)

	var groupID uint
	r.db.WithContext(ctx).Table("hosts").Select("group_id").Where("id = ?", hostID).Scan(&groupID)

	var count int64
	sql := fmt.Sprintf(`
		SELECT COUNT(*) FROM sys_asset_authorization a
		WHERE %s
		AND %s
		AND a.asset_group_id = ?
		AND COALESCE(a.asset_type, 'host') = 'host'
		AND (JSON_LENGTH(COALESCE(a.asset_ids, JSON_ARRAY())) = 0 OR JSON_CONTAINS(a.asset_ids, CAST(? AS JSON)))
		AND (a.permissions & ?) > 0
	`, r.activeCondition(), r.userMatchCondition())

	err = r.db.WithContext(ctx).Raw(sql, userID, deptID, groupID, hostID, operation).Scan(&count).Error
	return count > 0, err
}

// GetUserHostPermissions 获取用户对指定主机的所有操作权限（bitmask OR 合并）
func (r *assetAuthorizationRepo) GetUserHostPermissions(ctx context.Context, userID, hostID uint) (uint, error) {
	admin, err := r.isAdmin(ctx, userID)
	if err != nil {
		return 0, err
	}
	if admin {
		return rbac.PermissionAll, nil
	}

	deptID, _ := r.getUserDepartmentID(ctx, userID)

	var groupID uint
	r.db.WithContext(ctx).Table("hosts").Select("group_id").Where("id = ?", hostID).Scan(&groupID)

	var permissions uint
	sql := fmt.Sprintf(`
		SELECT COALESCE(BIT_OR(a.permissions), 0) FROM sys_asset_authorization a
		WHERE %s
		AND %s
		AND a.asset_group_id = ?
		AND COALESCE(a.asset_type, 'host') = 'host'
		AND (JSON_LENGTH(COALESCE(a.asset_ids, JSON_ARRAY())) = 0 OR JSON_CONTAINS(a.asset_ids, CAST(? AS JSON)))
	`, r.activeCondition(), r.userMatchCondition())

	err = r.db.WithContext(ctx).Raw(sql, userID, deptID, groupID, hostID).Scan(&permissions).Error
	return permissions, err
}

// GetUserAccessibleHostIDs 获取用户有权限访问的所有主机 ID 列表
func (r *assetAuthorizationRepo) GetUserAccessibleHostIDs(ctx context.Context, userID uint) ([]uint, error) {
	admin, err := r.isAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}
	if admin {
		var allHostIDs []uint
		err = r.db.WithContext(ctx).Table("hosts").Where("deleted_at IS NULL").Pluck("id", &allHostIDs).Error
		return allHostIDs, err
	}

	deptID, _ := r.getUserDepartmentID(ctx, userID)

	var hostIDs []uint
	sql := fmt.Sprintf(`
		SELECT DISTINCT h.id
		FROM hosts AS h
		JOIN sys_asset_authorization AS a ON a.asset_group_id = h.group_id
		WHERE h.deleted_at IS NULL
		AND %s
		AND %s
		AND COALESCE(a.asset_type, 'host') = 'host'
		AND (
			JSON_LENGTH(COALESCE(a.asset_ids, JSON_ARRAY())) = 0
			OR JSON_CONTAINS(a.asset_ids, CAST(h.id AS JSON))
		)
	`, r.activeCondition(), r.userMatchCondition())

	err = r.db.WithContext(ctx).Raw(sql, userID, deptID).Scan(&hostIDs).Error
	return hostIDs, err
}

// ---- 网络设备权限检查 ----

// CheckNetworkDeviceOperationPermission 检查用户是否有对指定网络设备的特定操作权限
func (r *assetAuthorizationRepo) CheckNetworkDeviceOperationPermission(ctx context.Context, userID, deviceID uint, operation uint) (bool, error) {
	admin, err := r.isAdmin(ctx, userID)
	if err != nil {
		return false, err
	}
	if admin {
		return true, nil
	}

	deptID, _ := r.getUserDepartmentID(ctx, userID)

	var groupID uint
	r.db.WithContext(ctx).Table("network_devices").Select("group_id").Where("id = ?", deviceID).Scan(&groupID)

	var count int64
	sql := fmt.Sprintf(`
		SELECT COUNT(*) FROM sys_asset_authorization a
		WHERE %s
		AND %s
		AND a.asset_group_id = ?
		AND a.asset_type = 'network_device'
		AND (JSON_LENGTH(COALESCE(a.asset_ids, JSON_ARRAY())) = 0 OR JSON_CONTAINS(a.asset_ids, CAST(? AS JSON)))
		AND (a.permissions & ?) > 0
	`, r.activeCondition(), r.userMatchCondition())

	err = r.db.WithContext(ctx).Raw(sql, userID, deptID, groupID, deviceID, operation).Scan(&count).Error
	return count > 0, err
}

// GetUserNetworkDevicePermissions 获取用户对指定网络设备的所有操作权限
func (r *assetAuthorizationRepo) GetUserNetworkDevicePermissions(ctx context.Context, userID, deviceID uint) (uint, error) {
	admin, err := r.isAdmin(ctx, userID)
	if err != nil {
		return 0, err
	}
	if admin {
		return rbac.PermissionAll, nil
	}

	deptID, _ := r.getUserDepartmentID(ctx, userID)

	var groupID uint
	r.db.WithContext(ctx).Table("network_devices").Select("group_id").Where("id = ?", deviceID).Scan(&groupID)

	var permissions uint
	sql := fmt.Sprintf(`
		SELECT COALESCE(BIT_OR(a.permissions), 0) FROM sys_asset_authorization a
		WHERE %s
		AND %s
		AND a.asset_group_id = ?
		AND a.asset_type = 'network_device'
		AND (JSON_LENGTH(COALESCE(a.asset_ids, JSON_ARRAY())) = 0 OR JSON_CONTAINS(a.asset_ids, CAST(? AS JSON)))
	`, r.activeCondition(), r.userMatchCondition())

	err = r.db.WithContext(ctx).Raw(sql, userID, deptID, groupID, deviceID).Scan(&permissions).Error
	return permissions, err
}

// GetUserAccessibleNetworkDeviceIDs 获取用户有权限访问的所有网络设备 ID 列表
func (r *assetAuthorizationRepo) GetUserAccessibleNetworkDeviceIDs(ctx context.Context, userID uint) ([]uint, error) {
	admin, err := r.isAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}
	if admin {
		return nil, nil // nil 表示不需要过滤（管理员可访问所有）
	}

	deptID, _ := r.getUserDepartmentID(ctx, userID)

	var deviceIDs []uint
	sql := fmt.Sprintf(`
		SELECT DISTINCT d.id
		FROM network_devices AS d
		JOIN sys_asset_authorization AS a ON a.asset_group_id = d.group_id
		WHERE d.deleted_at IS NULL
		AND %s
		AND %s
		AND a.asset_type = 'network_device'
		AND (
			JSON_LENGTH(COALESCE(a.asset_ids, JSON_ARRAY())) = 0
			OR JSON_CONTAINS(a.asset_ids, CAST(d.id AS JSON))
		)
	`, r.activeCondition(), r.userMatchCondition())

	err = r.db.WithContext(ctx).Raw(sql, userID, deptID).Scan(&deviceIDs).Error
	return deviceIDs, err
}

// ---- 数据迁移辅助 ----

// MigrateFromOldPermissions 将旧的 sys_role_asset_permission 数据迁移到新的 sys_asset_authorization
func MigrateFromOldPermissions(db *gorm.DB) error {
	type OldPermission struct {
		ID           uint            `gorm:"primarykey"`
		RoleID       uint
		AssetGroupID uint
		AssetType    string
		HostIDs      json.RawMessage `gorm:"type:json"`
		Permissions  uint
	}

	var oldPerms []OldPermission
	if err := db.Table("sys_role_asset_permission").Where("deleted_at IS NULL").Find(&oldPerms).Error; err != nil {
		// 旧表可能不存在了，跳过
		return nil
	}

	if len(oldPerms) == 0 {
		return nil
	}

	for _, old := range oldPerms {
		// 查询该角色下的所有用户
		var userIDs []uint
		db.Table("sys_user_role").Where("role_id = ?", old.RoleID).Pluck("user_id", &userIDs)
		if len(userIDs) == 0 {
			continue
		}

		// 解析旧的 host_ids
		var assetIDs []uint
		if len(old.HostIDs) > 0 {
			_ = json.Unmarshal(old.HostIDs, &assetIDs)
		}

		assetType := old.AssetType
		if assetType == "" {
			assetType = "host"
		}

		// 获取角色名
		var roleName string
		db.Table("sys_role").Select("name").Where("id = ?", old.RoleID).Scan(&roleName)
		// 获取分组名
		var groupName string
		db.Table("asset_group").Select("name").Where("id = ?", old.AssetGroupID).Scan(&groupName)

		// 构建名称
		typeLabel := "主机"
		if assetType == "network_device" {
			typeLabel = "网络设备"
		}
		name := fmt.Sprintf("[迁移] %s - %s %s权限", roleName, groupName, typeLabel)

		// 检查是否已迁移过
		var existCount int64
		db.Table("sys_asset_authorization").Where("name = ? AND deleted_at IS NULL", name).Count(&existCount)
		if existCount > 0 {
			continue
		}

		userIDsJSON, _ := json.Marshal(userIDs)
		assetIDsJSON, _ := json.Marshal(assetIDs)
		if assetIDs == nil {
			assetIDsJSON = []byte("[]")
		}

		var permNames []string
		perms := old.Permissions
		if (perms & rbac.PermissionView) > 0 {
			permNames = append(permNames, "查看")
		}
		if (perms & rbac.PermissionEdit) > 0 {
			permNames = append(permNames, "编辑")
		}
		if (perms & rbac.PermissionTerminal) > 0 {
			permNames = append(permNames, "终端")
		}
		desc := fmt.Sprintf("从旧权限系统迁移: 角色[%s] -> 分组[%s], 权限: %s", roleName, groupName, strings.Join(permNames, "/"))

		db.Exec(`INSERT INTO sys_asset_authorization 
			(name, user_ids, department_ids, asset_group_id, asset_type, asset_ids, permissions, is_active, description, created_at, updated_at)
			VALUES (?, ?, '[]', ?, ?, ?, ?, true, ?, NOW(), NOW())`,
			name, string(userIDsJSON), old.AssetGroupID, assetType, string(assetIDsJSON), old.Permissions, desc)
	}

	return nil
}
