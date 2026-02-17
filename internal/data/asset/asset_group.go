// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package asset

import (
	"context"
	"time"

	"github.com/ydcloud-dy/mom/internal/biz/asset"
	"github.com/ydcloud-dy/mom/internal/data"
	"gorm.io/gorm"
)

type assetGroupRepo struct {
	db    *gorm.DB
	cache *data.Cache
}

func NewAssetGroupRepo(db *gorm.DB) asset.AssetGroupRepo {
	return &assetGroupRepo{db: db}
}

func NewAssetGroupRepoWithCache(db *gorm.DB, cache *data.Cache) asset.AssetGroupRepo {
	return &assetGroupRepo{db: db, cache: cache}
}

func (r *assetGroupRepo) Create(ctx context.Context, group *asset.AssetGroup) error {
	err := r.db.WithContext(ctx).Create(group).Error
	if err == nil {
		r.invalidateGroupTreeCache(ctx)
	}
	return err
}

func (r *assetGroupRepo) Update(ctx context.Context, group *asset.AssetGroup) error {
	err := r.db.WithContext(ctx).Model(group).Omit("created_at").Updates(group).Error
	if err == nil {
		r.invalidateGroupTreeCache(ctx)
	}
	return err
}

func (r *assetGroupRepo) Delete(ctx context.Context, id uint) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&asset.AssetGroup{}).Unscoped().Where("parent_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return gorm.ErrRegistered
		}
		return tx.Unscoped().Delete(&asset.AssetGroup{}, id).Error
	})
	if err == nil {
		r.invalidateGroupTreeCache(ctx)
	}
	return err
}

// invalidateGroupTreeCache 失效分组树缓存
func (r *assetGroupRepo) invalidateGroupTreeCache(ctx context.Context) {
	if r.cache != nil {
		r.cache.DelByPrefix(ctx, "asset:group_tree")
	}
}

func (r *assetGroupRepo) GetByID(ctx context.Context, id uint) (*asset.AssetGroup, error) {
	var group asset.AssetGroup
	err := r.db.WithContext(ctx).First(&group, id).Error
	return &group, err
}

func (r *assetGroupRepo) GetTree(ctx context.Context) ([]*asset.AssetGroup, error) {
	cacheKey := "asset:group_tree"
	var cached []*asset.AssetGroup
	if r.cache != nil && r.cache.Get(ctx, cacheKey, &cached) {
		return cached, nil
	}

	var groups []*asset.AssetGroup
	err := r.db.WithContext(ctx).Order("sort ASC").Find(&groups).Error
	if err != nil {
		return nil, err
	}

	// 统计每个分组的主机数量（排除软删除的记录）
	hostCounts := make(map[uint]int)
	var hostResults []struct {
		GroupID uint
		Count   int64
	}
	err = r.db.WithContext(ctx).Model(&asset.Host{}).Select("group_id, COUNT(*) as count").Where("group_id > 0").Group("group_id").Scan(&hostResults).Error
	if err == nil {
		for _, result := range hostResults {
			hostCounts[result.GroupID] = int(result.Count)
		}
	}

	// 统计每个分组的网络设备数量
	deviceCounts := make(map[uint]int)
	var deviceResults []struct {
		GroupID uint
		Count   int64
	}
	err = r.db.WithContext(ctx).Model(&asset.NetworkDevice{}).Select("group_id, COUNT(*) as count").Where("group_id > 0").Group("group_id").Scan(&deviceResults).Error
	if err == nil {
		for _, result := range deviceResults {
			deviceCounts[result.GroupID] = int(result.Count)
		}
	}

	// 为每个分组设置数量
	for _, group := range groups {
		group.HostCount = hostCounts[group.ID]
		group.DeviceCount = deviceCounts[group.ID]
	}

	tree := r.buildTree(groups, 0)
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, tree, 5*time.Minute)
	}
	return tree, nil
}

func (r *assetGroupRepo) buildTree(groups []*asset.AssetGroup, parentID uint) []*asset.AssetGroup {
	var tree []*asset.AssetGroup
	for _, group := range groups {
		if group.ParentID == parentID {
			children := r.buildTree(groups, group.ID)
			if len(children) > 0 {
				group.Children = children
				// 累加子分组的数量
				for _, child := range children {
					group.HostCount += child.HostCount
					group.DeviceCount += child.DeviceCount
				}
			}
			tree = append(tree, group)
		}
	}
	return tree
}

func (r *assetGroupRepo) GetAll(ctx context.Context) ([]*asset.AssetGroup, error) {
	var groups []*asset.AssetGroup
	err := r.db.WithContext(ctx).Order("sort ASC").Find(&groups).Error
	return groups, err
}

func (r *assetGroupRepo) List(ctx context.Context, page, pageSize int, keyword string) ([]*asset.AssetGroup, int64, error) {
	var groups []*asset.AssetGroup
	var total int64

	query := r.db.WithContext(ctx).Model(&asset.AssetGroup{})
	if keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("sort ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&groups).Error
	return groups, total, err
}

// GetDescendantIDs 获取指定分组的所有子孙分组ID
func (r *assetGroupRepo) GetDescendantIDs(ctx context.Context, id uint) ([]uint, error) {
	var ids []uint

	// 递归获取子分组ID
	var getChildren func(uint) error
	getChildren = func(parentID uint) error {
		var children []uint
		err := r.db.WithContext(ctx).Model(&asset.AssetGroup{}).Where("parent_id = ?", parentID).Pluck("id", &children).Error
		if err != nil {
			return err
		}

		ids = append(ids, children...)
		for _, childID := range children {
			if err := getChildren(childID); err != nil {
				return err
			}
		}
		return nil
	}

	if err := getChildren(id); err != nil {
		return nil, err
	}

	return ids, nil
}
