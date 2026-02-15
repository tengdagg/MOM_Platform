package asset

import (
	"context"

	"github.com/ydcloud-dy/mom/internal/biz/asset"
	"gorm.io/gorm"
)

type networkDeviceRepo struct {
	db *gorm.DB
}

// NewNetworkDeviceRepo 创建网络设备仓库
func NewNetworkDeviceRepo(db *gorm.DB) asset.NetworkDeviceRepo {
	return &networkDeviceRepo{db: db}
}

// Create 创建网络设备
func (r *networkDeviceRepo) Create(ctx context.Context, device *asset.NetworkDevice) error {
	return r.db.WithContext(ctx).Create(device).Error
}

// Update 更新网络设备
func (r *networkDeviceRepo) Update(ctx context.Context, device *asset.NetworkDevice) error {
	return r.db.WithContext(ctx).Save(device).Error
}

// Delete 删除网络设备
func (r *networkDeviceRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&asset.NetworkDevice{}, id).Error
}

// GetByID 根据ID获取网络设备
func (r *networkDeviceRepo) GetByID(ctx context.Context, id uint) (*asset.NetworkDevice, error) {
	var device asset.NetworkDevice
	if err := r.db.WithContext(ctx).First(&device, id).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

// List 分页查询网络设备
func (r *networkDeviceRepo) List(ctx context.Context, page, pageSize int, keyword, deviceType, protocol string, groupID uint) ([]*asset.NetworkDevice, int64, error) {
	var devices []*asset.NetworkDevice
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.NetworkDevice{})

	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR ip LIKE ? OR brand_model LIKE ? OR serial_number LIKE ?", like, like, like, like)
	}
	if deviceType != "" {
		query = query.Where("device_type = ?", deviceType)
	}
	if protocol != "" {
		query = query.Where("protocol = ?", protocol)
	}
	if groupID > 0 {
		query = query.Where("group_id = ?", groupID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&devices).Error; err != nil {
		return nil, 0, err
	}

	return devices, total, nil
}

// ListFiltered 带可访问ID过滤的分页查询
func (r *networkDeviceRepo) ListFiltered(ctx context.Context, page, pageSize int, keyword, deviceType, protocol string, groupID uint, accessibleIDs []uint) ([]*asset.NetworkDevice, int64, error) {
	var devices []*asset.NetworkDevice
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.NetworkDevice{})

	// 如果 accessibleIDs 不为 nil（非管理员），则只显示可访问的设备
	if accessibleIDs != nil {
		if len(accessibleIDs) == 0 {
			return devices, 0, nil // 无权限，返回空
		}
		query = query.Where("id IN ?", accessibleIDs)
	}

	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR ip LIKE ? OR brand_model LIKE ? OR serial_number LIKE ?", like, like, like, like)
	}
	if deviceType != "" {
		query = query.Where("device_type = ?", deviceType)
	}
	if protocol != "" {
		query = query.Where("protocol = ?", protocol)
	}
	if groupID > 0 {
		query = query.Where("group_id = ?", groupID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&devices).Error; err != nil {
		return nil, 0, err
	}

	return devices, total, nil
}

// GetAll 获取所有网络设备
func (r *networkDeviceRepo) GetAll(ctx context.Context) ([]*asset.NetworkDevice, error) {
	var devices []*asset.NetworkDevice
	if err := r.db.WithContext(ctx).Find(&devices).Error; err != nil {
		return nil, err
	}
	return devices, nil
}

// UpdateStatus 更新网络设备在线状态
func (r *networkDeviceRepo) UpdateStatus(ctx context.Context, id uint, status int) error {
	return r.db.WithContext(ctx).Model(&asset.NetworkDevice{}).Where("id = ?", id).Update("status", status).Error
}

// CountByCredentialID 统计使用指定凭证的网络设备数量
func (r *networkDeviceRepo) CountByCredentialID(ctx context.Context, credentialID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&asset.NetworkDevice{}).Where("credential_id = ?", credentialID).Count(&count).Error
	return count, err
}
