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

package rbac

import (
	"context"
	"fmt"
	"time"

	"github.com/ydcloud-dy/mom/internal/biz/rbac"
	"github.com/ydcloud-dy/mom/internal/data"
	"gorm.io/gorm"
)

type menuRepo struct {
	db    *gorm.DB
	cache *data.Cache
}

func NewMenuRepo(db *gorm.DB) rbac.MenuRepo {
	return &menuRepo{db: db}
}

func NewMenuRepoWithCache(db *gorm.DB, cache *data.Cache) rbac.MenuRepo {
	return &menuRepo{db: db, cache: cache}
}

func (r *menuRepo) Create(ctx context.Context, menu *rbac.SysMenu) error {
	return r.db.WithContext(ctx).Create(menu).Error
}

func (r *menuRepo) Update(ctx context.Context, menu *rbac.SysMenu) error {
	// 先检查 code 是否与其他记录冲突
	if menu.Code != "" {
		exists, err := r.CheckCodeExists(ctx, menu.Code, menu.ID)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Duplicate entry '%s' for key 'code'", menu.Code)
		}
	}

	err := r.db.WithContext(ctx).Model(&rbac.SysMenu{}).Where("id = ?", menu.ID).Updates(map[string]interface{}{
		"name":      menu.Name,
		"code":      menu.Code,
		"type":      menu.Type,
		"parent_id": menu.ParentID,
		"path":      menu.Path,
		"component": menu.Component,
		"icon":      menu.Icon,
		"sort":      menu.Sort,
		"visible":   menu.Visible,
		"status":    menu.Status,
	}).Error

	if err == nil && r.cache != nil {
		// 清除所有用户的菜单树缓存，因为菜单结构或属性变更会影响所有有权限的用户
		// 使用 scan 模式匹配所有相关key
		r.cache.DelByPrefix(ctx, "user:*:menu_tree")
	}

	return err
}

func (r *menuRepo) UpdateSort(ctx context.Context, id uint, sort int) error {
	return r.db.WithContext(ctx).Model(&rbac.SysMenu{}).Where("id = ?", id).Update("sort", sort).Error
}

func (r *menuRepo) BatchUpdateSort(ctx context.Context, sorts []rbac.MenuSortItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range sorts {
			if err := tx.Model(&rbac.SysMenu{}).Where("id = ?", item.ID).Update("sort", item.Sort).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *menuRepo) CheckCodeExists(ctx context.Context, code string, excludeID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&rbac.SysMenu{}).
		Where("code = ? AND id != ?", code, excludeID).
		Count(&count).Error
	return count > 0, err
}

func (r *menuRepo) Delete(ctx context.Context, id uint) error {
	// new code:
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除角色菜单关联
		if err := tx.Where("menu_id = ?", id).Delete(&rbac.SysRoleMenu{}).Error; err != nil {
			return err
		}

		// 检查是否有子菜单
		var count int64
		if err := tx.Model(&rbac.SysMenu{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return gorm.ErrRegistered // 存在子菜单，不能删除
		}

		// 删除菜单
		return tx.Delete(&rbac.SysMenu{}, id).Error
	})

	if err == nil && r.cache != nil {
		r.cache.DelByPrefix(ctx, "user:*:menu_tree")
	}

	return err
}

func (r *menuRepo) GetByID(ctx context.Context, id uint) (*rbac.SysMenu, error) {
	var menu rbac.SysMenu
	err := r.db.WithContext(ctx).First(&menu, id).Error
	return &menu, err
}

func (r *menuRepo) GetTree(ctx context.Context) ([]*rbac.SysMenu, error) {
	var menus []*rbac.SysMenu
	err := r.db.WithContext(ctx).
		Where("visible = ?", 1).
		Order("sort ASC, id ASC").
		Find(&menus).Error
	if err != nil {
		return nil, err
	}
	return r.buildTree(menus, 0), nil
}

func (r *menuRepo) GetAllTree(ctx context.Context) ([]*rbac.SysMenu, error) {
	var menus []*rbac.SysMenu
	err := r.db.WithContext(ctx).
		Order("sort ASC, id ASC").
		Find(&menus).Error
	if err != nil {
		return nil, err
	}
	return r.buildTree(menus, 0), nil
}

func (r *menuRepo) buildTree(menus []*rbac.SysMenu, parentID uint) []*rbac.SysMenu {
	var tree []*rbac.SysMenu
	for _, menu := range menus {
		if menu.ParentID == parentID {
			children := r.buildTree(menus, menu.ID)
			if len(children) > 0 {
				menu.Children = children
			}
			tree = append(tree, menu)
		}
	}
	return tree
}

func (r *menuRepo) GetByUserID(ctx context.Context, userID uint) ([]*rbac.SysMenu, error) {
	cacheKey := fmt.Sprintf("user:%d:menu_tree", userID)
	var cached []*rbac.SysMenu
	if r.cache != nil && r.cache.Get(ctx, cacheKey, &cached) {
		return cached, nil
	}

	var menus []*rbac.SysMenu
	err := r.db.WithContext(ctx).
		Joins("JOIN sys_role_menu ON sys_role_menu.menu_id = sys_menu.id").
		Joins("JOIN sys_user_role ON sys_user_role.role_id = sys_role_menu.role_id").
		Where("sys_user_role.user_id = ? AND sys_menu.status = 1 AND sys_menu.visible = 1", userID).
		Distinct().
		Order("sys_menu.sort ASC, sys_menu.id ASC").
		Find(&menus).Error

	if err != nil {
		return nil, err
	}

	tree := r.buildTree(menus, 0)
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, tree, 30*time.Minute)
	}
	return tree, nil
}

func (r *menuRepo) GetByRoleID(ctx context.Context, roleID uint) ([]*rbac.SysMenu, error) {
	var menus []*rbac.SysMenu
	err := r.db.WithContext(ctx).
		Joins("JOIN sys_role_menu ON sys_role_menu.menu_id = sys_menu.id").
		Where("sys_role_menu.role_id = ?", roleID).
		Find(&menus).Error
	return menus, err
}

func (r *menuRepo) GetButtonCodesByUserID(ctx context.Context, userID uint) ([]string, error) {
	cacheKey := fmt.Sprintf("user:%d:button_codes", userID)
	var cached []string
	if r.cache != nil && r.cache.Get(ctx, cacheKey, &cached) {
		return cached, nil
	}

	var codes []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT m.code
		FROM sys_menu m
		JOIN sys_role_menu rm ON rm.menu_id = m.id
		JOIN sys_user_role ur ON ur.role_id = rm.role_id
		WHERE ur.user_id = ?
		  AND m.status = 1
		  AND m.type = 3
		  AND m.code != ''
		  AND m.deleted_at IS NULL
	`, userID).Scan(&codes).Error

	if err == nil && r.cache != nil {
		r.cache.Set(ctx, cacheKey, codes, 30*time.Minute)
	}
	return codes, err
}
