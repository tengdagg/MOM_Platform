package asset

import (
	"context"
	"fmt"
)

// NetworkDeviceUseCase 网络设备业务逻辑
type NetworkDeviceUseCase struct {
	deviceRepo     NetworkDeviceRepo
	credentialRepo CredentialRepo
	groupRepo      AssetGroupRepo
}

// NewNetworkDeviceUseCase 创建网络设备用例
func NewNetworkDeviceUseCase(deviceRepo NetworkDeviceRepo, credentialRepo CredentialRepo, groupRepo AssetGroupRepo) *NetworkDeviceUseCase {
	return &NetworkDeviceUseCase{
		deviceRepo:     deviceRepo,
		credentialRepo: credentialRepo,
		groupRepo:      groupRepo,
	}
}

// Create 创建网络设备
func (uc *NetworkDeviceUseCase) Create(ctx context.Context, req *NetworkDeviceRequest) (*NetworkDevice, error) {
	device := req.ToModel()
	if err := uc.deviceRepo.Create(ctx, device); err != nil {
		return nil, err
	}
	return device, nil
}

// Update 更新网络设备
func (uc *NetworkDeviceUseCase) Update(ctx context.Context, req *NetworkDeviceRequest) error {
	device, err := uc.deviceRepo.GetByID(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("网络设备不存在")
	}

	device.Name = req.Name
	device.IP = req.IP
	device.Brand = req.Brand
	device.BrandModel = req.BrandModel
	device.SerialNumber = req.SerialNumber
	device.DeviceType = req.DeviceType
	device.Protocol = req.Protocol
	device.CredentialID = req.CredentialID
	device.GroupID = req.GroupID
	device.Description = req.Description
	device.Tags = req.Tags

	port := req.Port
	if port == 0 {
		if req.Protocol == "telnet" {
			port = 23
		} else {
			port = 22
		}
	}
	device.Port = port

	return uc.deviceRepo.Update(ctx, device)
}

// Delete 删除网络设备
func (uc *NetworkDeviceUseCase) Delete(ctx context.Context, id uint) error {
	return uc.deviceRepo.Delete(ctx, id)
}

// GetByID 根据ID获取网络设备详情
func (uc *NetworkDeviceUseCase) GetByID(ctx context.Context, id uint) (*NetworkDevice, error) {
	device, err := uc.deviceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 加载凭证信息（不含敏感数据）
	if device.CredentialID > 0 {
		cred, err := uc.credentialRepo.GetByID(ctx, device.CredentialID)
		if err == nil && cred != nil {
			device.Credential = &Credential{
				ID:   cred.ID,
				Name: cred.Name,
				Type: cred.Type,
			}
		}
	}

	// 加载分组信息
	if device.GroupID > 0 {
		group, err := uc.groupRepo.GetByID(ctx, device.GroupID)
		if err == nil && group != nil {
			device.Group = group
		}
	}

	return device, nil
}

// List 分页查询网络设备列表
func (uc *NetworkDeviceUseCase) List(ctx context.Context, page, pageSize int, keyword, deviceType, protocol string, groupID uint) ([]*NetworkDevice, int64, error) {
	devices, total, err := uc.deviceRepo.List(ctx, page, pageSize, keyword, deviceType, protocol, groupID)
	if err != nil {
		return nil, 0, err
	}

	// 批量加载凭证和分组
	for _, device := range devices {
		if device.CredentialID > 0 {
			cred, err := uc.credentialRepo.GetByID(ctx, device.CredentialID)
			if err == nil && cred != nil {
				device.Credential = &Credential{
					ID:   cred.ID,
					Name: cred.Name,
					Type: cred.Type,
				}
			}
		}
		if device.GroupID > 0 {
			group, err := uc.groupRepo.GetByID(ctx, device.GroupID)
			if err == nil && group != nil {
				device.Group = group
			}
		}
	}

	return devices, total, nil
}

// ListFiltered 分页查询网络设备列表（支持按可访问ID过滤）
func (uc *NetworkDeviceUseCase) ListFiltered(ctx context.Context, page, pageSize int, keyword, deviceType, protocol string, groupID uint, accessibleIDs []uint) ([]*NetworkDevice, int64, error) {
	devices, total, err := uc.deviceRepo.ListFiltered(ctx, page, pageSize, keyword, deviceType, protocol, groupID, accessibleIDs)
	if err != nil {
		return nil, 0, err
	}

	// 批量加载凭证和分组
	for _, device := range devices {
		if device.CredentialID > 0 {
			cred, err := uc.credentialRepo.GetByID(ctx, device.CredentialID)
			if err == nil && cred != nil {
				device.Credential = &Credential{
					ID:   cred.ID,
					Name: cred.Name,
					Type: cred.Type,
				}
			}
		}
		if device.GroupID > 0 {
			group, err := uc.groupRepo.GetByID(ctx, device.GroupID)
			if err == nil && group != nil {
				device.Group = group
			}
		}
	}

	return devices, total, nil
}

// GetAll 获取所有网络设备
func (uc *NetworkDeviceUseCase) GetAll(ctx context.Context) ([]*NetworkDevice, error) {
	return uc.deviceRepo.GetAll(ctx)
}

// UpdateStatus 更新设备在线状态
func (uc *NetworkDeviceUseCase) UpdateStatus(ctx context.Context, id uint, status int) error {
	return uc.deviceRepo.UpdateStatus(ctx, id, status)
}

// GetByIDForConnection 获取网络设备连接信息（含解密凭证）
func (uc *NetworkDeviceUseCase) GetByIDForConnection(ctx context.Context, id uint) (*NetworkDevice, *Credential, error) {
	device, err := uc.deviceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("网络设备不存在: %w", err)
	}

	var credential *Credential
	if device.CredentialID > 0 {
		cred, err := uc.credentialRepo.GetByIDDecrypted(ctx, device.CredentialID)
		if err != nil {
			return nil, nil, fmt.Errorf("获取凭证失败: %w", err)
		}
		credential = cred
	}

	return device, credential, nil
}
