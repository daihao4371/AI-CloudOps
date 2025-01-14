package di

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

import (
	"context"
	cn "github.com/GoSimplicity/AI-CloudOps/internal/cron"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"time"
)

// InitAndRefreshK8sClient 初始化并启动定时刷新 Kubernetes 客户端
// 返回 cron 调度器实例以便调用者可以在需要时停止它
func InitAndRefreshK8sClient(K8sClient client.K8sClient, logger *zap.Logger, PromCache cache.MonitorCache, manager cn.CronManager) *cron.Cron {
	stdLogger := zap.NewStdLog(logger)

	// 启用秒级调度，并集成日志记录和恢复中间件
	c := cron.New(
		cron.WithSeconds(),
		cron.WithLogger(cron.VerbosePrintfLogger(stdLogger)), // 集成日志记录
		cron.WithChain(
			cron.Recover(cron.VerbosePrintfLogger(stdLogger)), // 添加恢复中间件，防止任务崩溃调度器
		),
	)

	// 执行初始刷新 Kubernetes 客户端
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		logger.Info("开始初始刷新 Kubernetes 客户端")
		if err := K8sClient.RefreshClients(ctx); err != nil {
			logger.Error("InitAndRefreshK8sClient: 初始刷新 Kubernetes 客户端失败", zap.Error(err))
		} else {
			logger.Info("InitAndRefreshK8sClient: 成功完成初始刷新 Kubernetes 客户端")
		}
	}()

	// 执行初始刷新 Prometheus 缓存
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		logger.Info("开始初始刷新 Prometheus 缓存")
		if err := PromCache.MonitorCacheManager(ctx); err != nil {
			logger.Error("InitAndRefreshPrometheusCache: 初始刷新 Prometheus 缓存失败", zap.Error(err))
		} else {
			logger.Info("InitAndRefreshPrometheusCache: 成功完成初始刷新 Prometheus 缓存")
		}
	}()

	// 启动值班历史记录填充任务
	go func() {
		ctx := context.Background()
		if err := manager.StartOnDutyHistoryManager(ctx); err != nil {
			logger.Error("启动值班历史记录填充任务失败", zap.Error(err))
		} else {
			logger.Info("成功启动值班历史记录填充任务")
		}
	}()

	// 从配置文件中获取 cron 表达式
	k8sRefreshCron := viper.GetString("k8s.refresh_cron")               // 例如 "@every 15s"
	prometheusRefreshCron := viper.GetString("prometheus.refresh_cron") // 例如 "@every 15s"

	// 添加 Kubernetes 客户端定时刷新任务
	if k8sRefreshCron != "" {
		_, err := c.AddFunc(k8sRefreshCron, func() {
			taskCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := K8sClient.RefreshClients(taskCtx); err != nil {
				logger.Error("InitAndRefreshK8sClient: 定时刷新 Kubernetes 客户端失败", zap.Error(err))
			} else {
				logger.Info("InitAndRefreshK8sClient: 成功刷新 Kubernetes 客户端")
			}
		})
		if err != nil {
			logger.Error("InitAndRefreshK8sClient: 添加 Kubernetes 客户端定时刷新任务失败", zap.Error(err))
		}
	} else {
		logger.Warn("InitAndRefreshK8sClient: 未配置 Kubernetes 客户端刷新 cron 表达式")
	}

	// 添加 Prometheus 缓存定时刷新任务
	if prometheusRefreshCron != "" {
		_, err := c.AddFunc(prometheusRefreshCron, func() {
			taskCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			logger.Info("开始定时刷新 Prometheus 缓存")
			if err := PromCache.MonitorCacheManager(taskCtx); err != nil {
				logger.Error("InitAndRefreshPrometheusCache: 定时刷新 Prometheus 缓存失败", zap.Error(err))
			} else {
				logger.Info("InitAndRefreshPrometheusCache: 成功刷新 Prometheus 缓存")
			}
		})
		if err != nil {
			logger.Error("InitAndRefreshK8sClient: 添加 Prometheus 缓存定时刷新任务失败", zap.Error(err))
		}
	} else {
		logger.Warn("InitAndRefreshK8sClient: 未配置 Prometheus 缓存刷新 cron 表达式")
	}

	return c
}
