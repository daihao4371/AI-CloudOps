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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type CloudDAO interface {
	ListCloudProviders(ctx context.Context) ([]model.CloudProviderResp, error)
	ListRegions(ctx context.Context, provider model.CloudProvider) ([]model.RegionResp, error)
	ListZones(ctx context.Context, provider model.CloudProvider, region string) ([]model.ZoneResp, error)
	ListInstanceTypes(ctx context.Context, provider model.CloudProvider, region string) ([]model.InstanceTypeResp, error)
	ListImages(ctx context.Context, provider model.CloudProvider, region string) ([]model.ImageResp, error)
	ListVpcs(ctx context.Context, provider model.CloudProvider, region string) ([]model.ResourceVpc, error)
	ListSecurityGroups(ctx context.Context, provider model.CloudProvider, region string) ([]model.SecurityGroupResp, error)
}

type cloudDAO struct {
	db *gorm.DB
}

func NewCloudDAO(db *gorm.DB) CloudDAO {
	return &cloudDAO{
		db: db,
	}
}

// ListCloudProviders 获取云厂商列表
func (c *cloudDAO) ListCloudProviders(ctx context.Context) ([]model.CloudProviderResp, error) {
	panic("unimplemented")
}

// ListImages 获取镜像列表
func (c *cloudDAO) ListImages(ctx context.Context, provider model.CloudProvider, region string) ([]model.ImageResp, error) {
	panic("unimplemented")
}

// ListInstanceTypes 获取实例类型列表
func (c *cloudDAO) ListInstanceTypes(ctx context.Context, provider model.CloudProvider, region string) ([]model.InstanceTypeResp, error) {
	panic("unimplemented")
}

// ListRegions 获取区域列表
func (c *cloudDAO) ListRegions(ctx context.Context, provider model.CloudProvider) ([]model.RegionResp, error) {
	panic("unimplemented")
}

// ListSecurityGroups 获取安全组列表
func (c *cloudDAO) ListSecurityGroups(ctx context.Context, provider model.CloudProvider, region string) ([]model.SecurityGroupResp, error) {
	panic("unimplemented")
}

// ListVpcs 获取VPC列表
func (c *cloudDAO) ListVpcs(ctx context.Context, provider model.CloudProvider, region string) ([]model.ResourceVpc, error) {
	panic("unimplemented")
}

// ListZones 获取可用区列表
func (c *cloudDAO) ListZones(ctx context.Context, provider model.CloudProvider, region string) ([]model.ZoneResp, error) {
	panic("unimplemented")
}
