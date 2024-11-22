package service

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
)

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

type AuthApiService interface {
	GetApiList(ctx context.Context, uid int) ([]*model.Api, error)
	GetApiListAll(ctx context.Context) ([]*model.Api, error)
	DeleteApi(ctx context.Context, apiID string) error
	CreateApi(ctx context.Context, api *model.Api) error
	UpdateApi(ctx context.Context, api *model.Api) error
}

type authApiService struct {
	apiDao  dao.AuthApiDAO
	roleDao dao.AuthRoleDAO
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAuthApiService(apiDao dao.AuthApiDAO, roleDao dao.AuthRoleDAO, l *zap.Logger, userDao userDao.UserDAO) AuthApiService {
	return &authApiService{
		apiDao:  apiDao,
		roleDao: roleDao,
		l:       l,
		userDao: userDao,
	}
}

func (a *authApiService) GetApiList(ctx context.Context, uid int) ([]*model.Api, error) {
	user, err := a.userDao.GetUserByID(ctx, uid)
	if err != nil {
		a.l.Error("GetUserByID failed", zap.Error(err))
		return nil, err
	}

	apis := make([]*model.Api, 0)

	for _, role := range user.Roles {
		roleApis, err := a.roleDao.GetApisByRoleID(ctx, role.ID)
		if err != nil {
			a.l.Error("GetApisByRoleID failed", zap.Error(err))
			return nil, err
		}

		apis = append(apis, roleApis...)
	}

	return apis, nil
}

func (a *authApiService) GetApiListAll(ctx context.Context) ([]*model.Api, error) {
	return a.apiDao.GetAllApis(ctx)
}

func (a *authApiService) DeleteApi(ctx context.Context, apiID string) error {
	return a.apiDao.DeleteApi(ctx, apiID)
}

func (a *authApiService) CreateApi(ctx context.Context, api *model.Api) error {
	return a.apiDao.CreateApi(ctx, api)
}

func (a *authApiService) UpdateApi(ctx context.Context, api *model.Api) error {
	return a.apiDao.UpdateApi(ctx, api)
}