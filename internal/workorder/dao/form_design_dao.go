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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrFormDesignNotFound      = fmt.Errorf("表单设计不存在")
	ErrFormDesignNameExists    = fmt.Errorf("表单设计名称已存在")
	ErrFormDesignCannotPublish = fmt.Errorf("表单设计状态不是草稿，无法发布")
)

type FormDesignDAO interface {
	CreateFormDesign(ctx context.Context, formDesign *model.FormDesign) error
	UpdateFormDesign(ctx context.Context, formDesign *model.FormDesign) error
	DeleteFormDesign(ctx context.Context, id int) error
	PublishFormDesign(ctx context.Context, id int) error
	ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) (*model.ListResp[model.FormDesignItem], error)
	GetFormDesign(ctx context.Context, id int) (*model.FormDesignResp, error)
	CloneFormDesign(ctx context.Context, id int, name string, creatorID int) (*model.FormDesignResp, error)
	GetFormDesignsByIDs(ctx context.Context, ids []int) ([]model.FormDesignResp, error)
	UpdateFormDesignStatus(ctx context.Context, id int, status int8) error
	CheckFormDesignNameExists(ctx context.Context, name string, excludeID ...int) (bool, error)
}

type formDesignDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewFormDesignDAO(db *gorm.DB, logger *zap.Logger) FormDesignDAO {
	return &formDesignDAO{
		db:     db,
		logger: logger,
	}
}

// CreateFormDesign 创建表单设计
func (f *formDesignDAO) CreateFormDesign(ctx context.Context, formDesign *model.FormDesign) error {
	// 序列化Schema
	if err := f.serializeSchema(formDesign); err != nil {
		f.logger.Error("序列化表单schema失败", zap.Error(err), zap.Int("formDesignID", formDesign.ID))
		return fmt.Errorf("序列化表单schema失败: %w", err)
	}

	if err := f.db.WithContext(ctx).Create(formDesign).Error; err != nil {
		if f.isDuplicateKeyError(err) {
			f.logger.Warn("表单设计名称已存在", zap.String("name", formDesign.Name))
			return ErrFormDesignNameExists
		}
		f.logger.Error("创建表单设计失败", zap.Error(err), zap.String("name", formDesign.Name))
		return fmt.Errorf("创建表单设计失败: %w", err)
	}

	f.logger.Info("创建表单设计成功", zap.Int("id", formDesign.ID), zap.String("name", formDesign.Name))
	return nil
}

// UpdateFormDesign 更新表单设计
func (f *formDesignDAO) UpdateFormDesign(ctx context.Context, formDesign *model.FormDesign) error {
	// 序列化Schema
	if err := f.serializeSchema(formDesign); err != nil {
		f.logger.Error("序列化表单schema失败", zap.Error(err), zap.Int("formDesignID", formDesign.ID))
		return fmt.Errorf("序列化表单schema失败: %w", err)
	}

	updateData := map[string]interface{}{
		"name":        formDesign.Name,
		"description": formDesign.Description,
		"schema":      formDesign.Schema,
		"version":     formDesign.Version,
		"status":      formDesign.Status,
		"category_id": formDesign.CategoryID,
		"updated_at":  time.Now(),
	}

	result := f.db.WithContext(ctx).
		Model(&model.FormDesign{}).
		Where("id = ?", formDesign.ID).
		Updates(updateData)

	if result.Error != nil {
		if f.isDuplicateKeyError(result.Error) {
			f.logger.Warn("表单设计名称已存在", zap.String("name", formDesign.Name), zap.Int("id", formDesign.ID))
			return ErrFormDesignNameExists
		}
		f.logger.Error("更新表单设计失败", zap.Error(result.Error), zap.Int("id", formDesign.ID))
		return fmt.Errorf("更新表单设计失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		f.logger.Warn("表单设计不存在", zap.Int("id", formDesign.ID))
		return ErrFormDesignNotFound
	}

	f.logger.Info("更新表单设计成功", zap.Int("id", formDesign.ID), zap.String("name", formDesign.Name))
	return nil
}

// DeleteFormDesign 删除表单设计
func (f *formDesignDAO) DeleteFormDesign(ctx context.Context, id int) error {
	result := f.db.WithContext(ctx).Delete(&model.FormDesign{}, id)
	if result.Error != nil {
		f.logger.Error("删除表单设计失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("删除表单设计失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		f.logger.Warn("表单设计不存在", zap.Int("id", id))
		return ErrFormDesignNotFound
	}

	f.logger.Info("删除表单设计成功", zap.Int("id", id))
	return nil
}

// PublishFormDesign 发布表单设计
func (f *formDesignDAO) PublishFormDesign(ctx context.Context, id int) error {
	result := f.db.WithContext(ctx).
		Model(&model.FormDesign{}).
		Where("id = ? AND status = ?", id, 0). // 0为草稿状态，其实在service中已经检查了，这里只是防御编程一下下
		Updates(map[string]interface{}{
			"status": 1, // 1为已发布状态
		})

	if result.Error != nil {
		f.logger.Error("发布表单设计失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("发布表单设计失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		f.logger.Warn("表单设计不存在或状态不是草稿", zap.Int("id", id))
		return ErrFormDesignCannotPublish
	}

	f.logger.Info("发布表单设计成功", zap.Int("id", id))
	return nil
}

// GetFormDesign 获取表单设计
func (f *formDesignDAO) GetFormDesign(ctx context.Context, id int) (*model.FormDesignResp, error) {
	var formDesign model.FormDesign

	err := f.db.WithContext(ctx).
		Preload("Category").
		First(&formDesign, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			f.logger.Warn("表单设计不存在", zap.Int("id", id))
			return nil, ErrFormDesignNotFound
		}
		f.logger.Error("获取表单设计失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取表单设计失败: %w", err)
	}

	// 转换为响应对象
	resp, err := f.convertToFormDesignResp(&formDesign)
	if err != nil {
		f.logger.Error("转换表单设计响应失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("转换表单设计响应失败: %w", err)
	}

	return resp, nil
}

// CloneFormDesign 克隆表单设计
func (f *formDesignDAO) CloneFormDesign(ctx context.Context, id int, name string, creatorID int) (*model.FormDesignResp, error) {
	// 使用事务确保数据一致性
	var clonedFormDesign *model.FormDesign
	err := f.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取原始表单设计
		var originalFormDesign model.FormDesign
		if err := tx.Where("id = ?", id).First(&originalFormDesign).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrFormDesignNotFound
			}
			return fmt.Errorf("获取原始表单设计失败: %w", err)
		}

		// 创建克隆对象
		clonedFormDesign = &model.FormDesign{
			Name:        name,
			Description: originalFormDesign.Description,
			Schema:      originalFormDesign.Schema,
			Version:     1, // 重置版本号
			Status:      0, // 草稿状态
			CategoryID:  originalFormDesign.CategoryID,
			CreatorID:   creatorID,
		}

		// 创建克隆记录
		if err := tx.Create(clonedFormDesign).Error; err != nil {
			if f.isDuplicateKeyError(err) {
				return ErrFormDesignNameExists
			}
			return fmt.Errorf("创建克隆表单设计失败: %w", err)
		}

		return nil
	})

	if err != nil {
		f.logger.Error("克隆表单设计失败", zap.Error(err), zap.Int("originalID", id), zap.String("newName", name))
		return nil, err
	}

	// 重新获取完整的表单设计数据（包括关联数据）
	formDesign, err := f.GetFormDesign(ctx, clonedFormDesign.ID)
	if err != nil {
		f.logger.Error("获取克隆后的表单设计失败", zap.Error(err), zap.Int("id", clonedFormDesign.ID))
		return nil, fmt.Errorf("获取克隆后的表单设计失败: %w", err)
	}

	f.logger.Info("克隆表单设计成功",
		zap.Int("originalID", id),
		zap.Int("newID", clonedFormDesign.ID),
		zap.String("newName", name))

	return formDesign, nil
}

// ListFormDesign 获取表单设计列表
func (f *formDesignDAO) ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) (*model.ListResp[model.FormDesignItem], error) {
	var formDesigns []model.FormDesign
	var total int64

	db := f.db.WithContext(ctx).Model(&model.FormDesign{})

	// 构建查询条件
	db = f.buildListQuery(db, req)

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		f.logger.Error("获取表单设计总数失败", zap.Error(err))
		return nil, fmt.Errorf("获取表单设计总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err := db.Preload("Category").
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&formDesigns).Error

	if err != nil {
		f.logger.Error("获取表单设计列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取表单设计列表失败: %w", err)
	}

	// 转换为FormDesignItem
	items := make([]model.FormDesignItem, 0, len(formDesigns))
	for _, design := range formDesigns {
		items = append(items, model.FormDesignItem{
			ID:          design.ID,
			Name:        design.Name,
			Description: design.Description,
			Version:     design.Version,
			Status:      design.Status,
			CategoryID:  design.CategoryID,
			Category:    design.Category,
			CreatorID:   design.CreatorID,
			CreatorName: design.CreatorName,
			CreatedAt:   design.CreatedAt,
			UpdatedAt:   design.UpdatedAt,
		})
	}

	result := &model.ListResp[model.FormDesignItem]{
		Items: items,
		Total: total,
	}

	f.logger.Info("获取表单设计列表成功",
		zap.Int("count", len(items)),
		zap.Int64("total", total),
		zap.Int("page", req.Page),
		zap.Int("size", req.Size))

	return result, nil
}

// GetFormDesignsByIDs 批量获取表单设计
func (f *formDesignDAO) GetFormDesignsByIDs(ctx context.Context, ids []int) ([]model.FormDesignResp, error) {
	if len(ids) == 0 {
		return []model.FormDesignResp{}, nil
	}

	var formDesigns []model.FormDesign
	err := f.db.WithContext(ctx).
		Preload("Category").
		Where("id IN ?", ids).
		Find(&formDesigns).Error

	if err != nil {
		f.logger.Error("批量获取表单设计失败", zap.Error(err), zap.Ints("ids", ids))
		return nil, fmt.Errorf("批量获取表单设计失败: %w", err)
	}

	// 转换为响应对象
	resps := make([]model.FormDesignResp, 0, len(formDesigns))
	for i := range formDesigns {
		resp, err := f.convertToFormDesignResp(&formDesigns[i])
		if err != nil {
			f.logger.Warn("转换表单设计响应失败", zap.Error(err), zap.Int("id", formDesigns[i].ID))
			continue
		}
		resps = append(resps, *resp)
	}

	return resps, nil
}

// UpdateFormDesignStatus 更新表单设计状态
func (f *formDesignDAO) UpdateFormDesignStatus(ctx context.Context, id int, status int8) error {
	result := f.db.WithContext(ctx).
		Model(&model.FormDesign{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		f.logger.Error("更新表单设计状态失败", zap.Error(result.Error), zap.Int("id", id), zap.Int8("status", status))
		return fmt.Errorf("更新表单设计状态失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		f.logger.Warn("表单设计不存在", zap.Int("id", id))
		return ErrFormDesignNotFound
	}

	f.logger.Info("更新表单设计状态成功", zap.Int("id", id), zap.Int8("status", status))
	return nil
}

// CheckFormDesignNameExists 检查表单设计名称是否存在
func (f *formDesignDAO) CheckFormDesignNameExists(ctx context.Context, name string, excludeID ...int) (bool, error) {
	var count int64
	db := f.db.WithContext(ctx).Model(&model.FormDesign{}).Where("name = ?", name)

	if len(excludeID) > 0 && excludeID[0] > 0 {
		db = db.Where("id != ?", excludeID[0])
	}

	if err := db.Count(&count).Error; err != nil {
		f.logger.Error("检查表单设计名称是否存在失败", zap.Error(err), zap.String("name", name))
		return false, fmt.Errorf("检查表单设计名称是否存在失败: %w", err)
	}

	return count > 0, nil
}

// 辅助方法

// buildListQuery 构建列表查询条件
func (f *formDesignDAO) buildListQuery(db *gorm.DB, req *model.ListFormDesignReq) *gorm.DB {
	// 搜索条件
	if req.Search != "" {
		searchPattern := "%" + strings.TrimSpace(req.Search) + "%"
		db = db.Where("name LIKE ? OR description LIKE ?", searchPattern, searchPattern)
	}

	// 状态过滤
	if req.Status != nil && *req.Status != 0 {
		db = db.Where("status = ?", *req.Status)
	}

	// 分类过滤
	if req.CategoryID != nil && *req.CategoryID != 0 {
		db = db.Where("category_id = ?", *req.CategoryID)
	}

	return db
}

// serializeSchema 序列化FormSchema到JSON字符串
func (f *formDesignDAO) serializeSchema(formDesign *model.FormDesign) error {
	formDesign.Schema = string(formDesign.Schema)
	return nil
}

// convertToFormDesignResp 将FormDesign转换为FormDesignResp
func (f *formDesignDAO) convertToFormDesignResp(formDesign *model.FormDesign) (*model.FormDesignResp, error) {
	// 反序列化Schema
	var schema model.FormSchema
	if formDesign.Schema != "" {
		if err := json.Unmarshal([]byte(formDesign.Schema), &schema); err != nil {
			return nil, fmt.Errorf("反序列化schema失败: %w", err)
		}
	}

	return &model.FormDesignResp{
		ID:          formDesign.ID,
		Name:        formDesign.Name,
		Description: formDesign.Description,
		Schema:      schema,
		Version:     formDesign.Version,
		Status:      formDesign.Status,
		CategoryID:  formDesign.CategoryID,
		Category:    formDesign.Category,
		CreatorID:   formDesign.CreatorID,
		CreatorName: formDesign.CreatorName,
		CreatedAt:   formDesign.CreatedAt,
		UpdatedAt:   formDesign.UpdatedAt,
	}, nil
}

// isDuplicateKeyError 判断是否为重复键错误
func (f *formDesignDAO) isDuplicateKeyError(err error) bool {
	return err == gorm.ErrDuplicatedKey ||
		strings.Contains(err.Error(), "UNIQUE constraint failed") ||
		strings.Contains(err.Error(), "Duplicate entry")
}
