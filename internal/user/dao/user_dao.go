/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package dao

import (
	"context"
	"errors"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserDAO interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByUsernames(ctx context.Context, usernames []string) ([]*model.User, error)
	GetAllUsers(ctx context.Context) ([]*model.User, error)
	GetUserByID(ctx context.Context, id int) (*model.User, error)
	GetUserByIDs(ctx context.Context, ids []int) ([]*model.User, error)
	GetPermCode(ctx context.Context, uid int) ([]string, error)
	ChangePassword(ctx context.Context, uid int, password string) error
	WriteOff(ctx context.Context, username, password string) error
	UpdateProfile(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, uid int) error
}

type userDAO struct {
	db            *gorm.DB
	l             *zap.Logger
	permissionDao dao.PermissionDAO
}

func NewUserDAO(db *gorm.DB, l *zap.Logger, permissionDao dao.PermissionDAO) UserDAO {
	return &userDAO{
		db:            db,
		l:             l,
		permissionDao: permissionDao,
	}
}

func (u *userDAO) getTime(ctx context.Context) int64 {
	return time.Now().Unix()
}

// CreateUser 创建用户
func (u *userDAO) CreateUser(ctx context.Context, user *model.User) error {
	user.CreatedAt = u.getTime(ctx)
	user.UpdatedAt = u.getTime(ctx)

	// 使用事务和一次性查询检查唯一性约束
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		query := tx.Model(&model.User{}).Where("deleted_at = ?", 0).
			Where("username = ? OR (mobile = ? AND mobile != '') OR (fei_shu_user_id = ? AND fei_shu_user_id != '')",
				user.Username, user.Mobile, user.FeiShuUserId)

		if err := query.Count(&count).Error; err != nil {
			u.l.Error("检查唯一性约束失败", zap.Error(err))
			return err
		}

		if count > 0 {
			// 进一步确定具体是哪个字段重复
			var existingUser model.User
			if err := tx.Where("username = ? AND deleted_at = ?", user.Username, 0).First(&existingUser).Error; err == nil {
				return errors.New("用户名已存在")
			}
			if user.Mobile != "" {
				if err := tx.Where("mobile = ? AND deleted_at = ?", user.Mobile, 0).First(&existingUser).Error; err == nil {
					return errors.New("手机号已存在")
				}
			}
			if user.FeiShuUserId != "" {
				if err := tx.Where("fei_shu_user_id = ? AND deleted_at = ?", user.FeiShuUserId, 0).First(&existingUser).Error; err == nil {
					return errors.New("飞书用户ID已存在")
				}
			}
		}

		// 创建用户
		if err := tx.Create(user).Error; err != nil {
			u.l.Error("创建用户失败", zap.Error(err))
			return err
		}

		return nil
	})

	return err
}

// GetUserByUsername 根据用户名获取用户信息
func (u *userDAO) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	if username == "" {
		return nil, errors.New("username 不能为空")
	}

	var user model.User
	if err := u.db.WithContext(ctx).Where("username = ? AND deleted_at = ?", username, 0).First(&user).Error; err != nil {
		u.l.Error("根据用户名获取用户失败", zap.String("username", username), zap.Error(err))
		return nil, err
	}

	return &user, nil
}

// GetAllUsers 获取所有用户
func (u *userDAO) GetAllUsers(ctx context.Context) ([]*model.User, error) {
	var users []*model.User
	if err := u.db.WithContext(ctx).
		Where("deleted_at = ?", 0).
		Preload("Roles").
		Preload("Menus").
		Preload("Apis").
		Find(&users).Error; err != nil {
		u.l.Error("获取所有用户失败", zap.Error(err))
		return nil, err
	}

	return users, nil
}

// GetUserByID 根据用户ID获取用户信息
func (u *userDAO) GetUserByID(ctx context.Context, id int) (*model.User, error) {
	if id <= 0 {
		return nil, errors.New("invalid user id")
	}

	var user model.User
	if err := u.db.WithContext(ctx).
		Where("id = ? AND deleted_at = ?", id, 0).
		Preload("Roles").
		Preload("Apis").
		First(&user).Error; err != nil {
		u.l.Error("根据ID获取用户失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	// 获取用户的菜单并构建树状结构
	menus, err := u.getUserMenus(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Menus = menus

	return &user, nil
}

// getUserMenus 获取用户菜单并构建树状结构
func (u *userDAO) getUserMenus(ctx context.Context, userID int) ([]*model.Menu, error) {
	var menus []*model.Menu
	if err := u.db.WithContext(ctx).
		Table("menus").
		Joins("LEFT JOIN user_menus ON menus.id = user_menus.menu_id").
		Where("user_menus.user_id = ? AND menus.deleted_at = ?", userID, 0).
		Order("menus.parent_id, menus.id").
		Find(&menus).Error; err != nil {
		u.l.Error("获取用户菜单失败", zap.Int("id", userID), zap.Error(err))
		return nil, err
	}

	return buildMenuTree(menus), nil
}

// buildMenuTree 构建菜单树
func buildMenuTree(menus []*model.Menu) []*model.Menu {
	menuMap := make(map[int]*model.Menu)
	var rootMenus []*model.Menu

	// 建立id->menu的映射
	for _, menu := range menus {
		menuMap[menu.ID] = menu
	}

	// 构建父子关系
	for _, menu := range menus {
		if menu.ParentID == 0 {
			rootMenus = append(rootMenus, menu)
		} else if parent, ok := menuMap[menu.ParentID]; ok {
			parent.Children = append(parent.Children, menu)
		}
	}

	return rootMenus
}

// GetPermCode 获取用户权限码
func (u *userDAO) GetPermCode(ctx context.Context, uid int) ([]string, error) {
	return nil, nil
}

// GetUserByUsernames 批量获取用户信息
func (u *userDAO) GetUserByUsernames(ctx context.Context, usernames []string) ([]*model.User, error) {
	if len(usernames) == 0 {
		return nil, errors.New("usernames cannot be empty")
	}

	var users []*model.User
	if err := u.db.WithContext(ctx).
		Where("username in (?) AND deleted_at = ?", usernames, 0).
		Find(&users).Error; err != nil {
		u.l.Error("批量获取用户失败", zap.Strings("usernames", usernames), zap.Error(err))
		return nil, err
	}

	return users, nil
}

// ChangePassword 修改密码
func (u *userDAO) ChangePassword(ctx context.Context, uid int, password string) error {
	if uid <= 0 {
		return errors.New("invalid user id")
	}
	if password == "" {
		return errors.New("password cannot be empty")
	}

	if err := u.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND deleted_at = ?", uid, 0).
		Update("password", password).Error; err != nil {
		u.l.Error("修改密码失败", zap.Int("uid", uid), zap.Error(err))
		return err
	}

	return nil
}

// UpdateProfile 更新用户信息
func (u *userDAO) UpdateProfile(ctx context.Context, user *model.User) error {
	if user == nil || user.ID <= 0 {
		return errors.New("invalid user")
	}

	// 使用事务和一次性查询检查唯一性约束
	err := u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		query := tx.Model(&model.User{}).Where("deleted_at = ?", 0).
			Where("id != ? AND ((mobile = ? AND mobile != '') OR (fei_shu_user_id = ? AND fei_shu_user_id != ''))",
				user.ID, user.Mobile, user.FeiShuUserId)

		if err := query.Count(&count).Error; err != nil {
			u.l.Error("检查唯一性约束失败", zap.Error(err))
			return err
		}

		if count > 0 {
			// 进一步确定具体是哪个字段重复
			var existingUser model.User
			if user.Mobile != "" {
				if err := tx.Where("id != ? AND mobile = ? AND deleted_at = ?", user.ID, user.Mobile, 0).First(&existingUser).Error; err == nil {
					return errors.New("手机号已存在")
				}
			}
			if user.FeiShuUserId != "" {
				if err := tx.Where("id != ? AND fei_shu_user_id = ? AND deleted_at = ?", user.ID, user.FeiShuUserId, 0).First(&existingUser).Error; err == nil {
					return errors.New("飞书用户ID已存在")
				}
			}
		}

		updates := map[string]interface{}{
			"real_name":       user.RealName,
			"desc":            user.Desc,
			"mobile":          user.Mobile,
			"fei_shu_user_id": user.FeiShuUserId,
			"account_type":    user.AccountType,
			"home_path":       user.HomePath,
			"enable":          user.Enable,
			"updated_at":      u.getTime(ctx),
		}

		if err := tx.Model(&model.User{}).
			Where("id = ? AND deleted_at = ?", user.ID, 0).
			Updates(updates).Error; err != nil {
			u.l.Error("更新用户信息失败", zap.Int("uid", user.ID), zap.Error(err))
			return err
		}

		return nil
	})

	return err
}

// WriteOff 注销用户
func (u *userDAO) WriteOff(ctx context.Context, username string, password string) error {
	if username == "" || password == "" {
		return errors.New("username and password cannot be empty")
	}

	// 软删除用户
	updates := map[string]interface{}{
		"deleted_at": u.getTime(ctx),
		"updated_at": u.getTime(ctx),
	}

	if err := u.db.WithContext(ctx).
		Model(&model.User{}).
		Where("username = ? AND deleted_at = ?", username, 0).
		Updates(updates).Error; err != nil {
		u.l.Error("注销用户失败", zap.String("username", username), zap.Error(err))
		return err
	}

	return nil
}

// DeleteUser 删除用户
func (u *userDAO) DeleteUser(ctx context.Context, uid int) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除用户角色关联
		if err := tx.Table("user_roles").Where("user_id = ?", uid).Delete(nil).Error; err != nil {
			u.l.Error("删除用户角色关联失败", zap.Int("uid", uid), zap.Error(err))
		}

		// 删除用户菜单关联
		if err := tx.Table("user_menus").Where("user_id = ?", uid).Delete(nil).Error; err != nil {
			u.l.Warn("删除用户菜单关联失败", zap.Int("uid", uid), zap.Error(err))
		}

		// 删除用户API关联
		if err := tx.Table("user_apis").Where("user_id = ?", uid).Delete(nil).Error; err != nil {
			u.l.Warn("删除用户API关联失败", zap.Int("uid", uid), zap.Error(err))
		}

		// 删除用户权限策略
		if err := u.permissionDao.RemoveUserPermissions(ctx, uid); err != nil {
			u.l.Warn("删除用户权限策略失败", zap.Int("uid", uid), zap.Error(err))
			return err
		}

		// 删除用户
		if err := tx.Where("id = ? AND deleted_at = ?", uid, 0).Delete(&model.User{}).Error; err != nil {
			u.l.Error("删除用户失败", zap.Int("uid", uid), zap.Error(err))
			return err
		}

		return nil
	})
}

// GetUserByIDs 批量获取用户信息
func (u *userDAO) GetUserByIDs(ctx context.Context, ids []int) ([]*model.User, error) {
	if len(ids) == 0 {
		return nil, errors.New("ids cannot be empty")
	}

	var users []*model.User
	if err := u.db.WithContext(ctx).
		Where("id in (?) AND deleted_at = ?", ids, 0).
		Find(&users).Error; err != nil {
		u.l.Error("批量获取用户失败", zap.Ints("ids", ids), zap.Error(err))
		return nil, err
	}

	return users, nil
}
