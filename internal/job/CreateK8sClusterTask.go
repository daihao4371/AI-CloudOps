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

package job

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type CreateK8sClusterTask struct {
	client client.K8sClient
	dao    admin.ClusterDAO
	l      *zap.Logger
}

type CreateK8sClusterPayload struct {
	Cluster *model.K8sCluster `json:"cluster"`
}

func NewCreateK8sClusterTask(l *zap.Logger, client client.K8sClient, dao admin.ClusterDAO) *CreateK8sClusterTask {
	return &CreateK8sClusterTask{
		l:      l,
		client: client,
		dao:    dao,
	}
}

// ProcessTask 处理异步任务
func (k *CreateK8sClusterTask) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p CreateK8sClusterPayload

	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		k.l.Error("解析任务载荷失败", zap.Error(err))
		return fmt.Errorf("解析任务载荷失败: %w", err)
	}

	if p.Cluster == nil {
		return fmt.Errorf("无效的集群配置")
	}

	const (
		maxRetries     = 5               // 最大重试次数
		baseRetryDelay = 5 * time.Second // 基础重试延迟
		maxConcurrent  = 5               // 最大并发数
		initTimeout    = 5 * time.Second // 初始化超时时间
	)

	var (
		retryCount int
		lastError  error
	)

	// 验证资源配额格式
	if err := k.validateResourceQuantities(p.Cluster); err != nil {
		k.dao.UpdateClusterStatus(ctx, p.Cluster.ID, "ERROR")
		k.l.Error("资源配额格式验证失败", zap.Error(err))
		return err
	}

	for retryCount < maxRetries {
		select {
		case <-ctx.Done():
			k.dao.UpdateClusterStatus(ctx, p.Cluster.ID, "ERROR")
			return ctx.Err()
		default:
			if err := k.processClusterConfig(ctx, p.Cluster, retryCount, initTimeout, maxConcurrent); err != nil {
				lastError = err
				retryCount++

				if retryCount < maxRetries {
					delay := time.Duration(retryCount) * baseRetryDelay
					k.l.Info("任务重试",
						zap.Int("重试次数", retryCount),
						zap.Duration("延迟时间", delay),
						zap.Error(lastError))
					time.Sleep(delay)
					continue
				}

				k.dao.UpdateClusterStatus(ctx, p.Cluster.ID, "ERROR")
				k.l.Error("达到最大重试次数，任务失败",
					zap.Int("最大重试次数", maxRetries),
					zap.Error(lastError))
				return lastError
			}

			k.dao.UpdateClusterStatus(ctx, p.Cluster.ID, "SUCCESS")
			return nil
		}
	}

	return nil
}

// validateResourceQuantities 验证资源配额格式
func (k *CreateK8sClusterTask) validateResourceQuantities(cluster *model.K8sCluster) error {
	// 检查并设置默认值，避免空字符串导致解析错误
	if cluster.CpuRequest == "" {
		cluster.CpuRequest = "500m" // 修改为小于或等于CpuLimit的值
	}
	if cluster.MemoryRequest == "" {
		cluster.MemoryRequest = "512Mi" // 修改为小于或等于MemoryLimit的值
	}
	if cluster.CpuLimit == "" {
		cluster.CpuLimit = "1000m" // 确保CpuLimit大于或等于CpuRequest
	}
	if cluster.MemoryLimit == "" {
		cluster.MemoryLimit = "1Gi" // 确保MemoryLimit大于或等于MemoryRequest
	}

	// 确保Request不大于Limit
	if cluster.CpuRequest > cluster.CpuLimit {
		cluster.CpuRequest = cluster.CpuLimit
	}
	if cluster.MemoryRequest > cluster.MemoryLimit {
		cluster.MemoryRequest = cluster.MemoryLimit
	}

	return nil
}

// processClusterConfig 处理集群配置
func (k *CreateK8sClusterTask) processClusterConfig(ctx context.Context, cluster *model.K8sCluster, _ int, initTimeout time.Duration, maxConcurrent int) error {
	ctx, cancel := context.WithTimeout(ctx, initTimeout)
	defer cancel()

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		return err
	}

	if err := k.client.InitClient(ctx, cluster.ID, restConfig); err != nil {
		return err
	}

	kubeClient, err := k.client.GetKubeClient(cluster.ID)
	if err != nil {
		return err
	}

	// 确保有命名空间配置
	if cluster.RestrictedNameSpace == nil || len(cluster.RestrictedNameSpace) == 0 {
		cluster.RestrictedNameSpace = []string{"default"}
	}

	return k.processNamespaces(ctx, kubeClient, cluster, maxConcurrent)
}

// processNamespaces 并发处理命名空间配置
func (k *CreateK8sClusterTask) processNamespaces(ctx context.Context, kubeClient *kubernetes.Clientset, cluster *model.K8sCluster, maxConcurrent int) error {
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, maxConcurrent)
	errChan := make(chan error, len(cluster.RestrictedNameSpace))

	ctx, cancel := context.WithTimeout(ctx, time.Duration(cluster.ActionTimeoutSeconds)*time.Second)
	defer cancel()

	for _, ns := range cluster.RestrictedNameSpace {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			wg.Add(1)
			go func(namespace string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				if err := k.configureNamespace(ctx, kubeClient, namespace, cluster); err != nil {
					select {
					case errChan <- err:
					default:
					}
					cancel()
				}
			}(ns)
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		close(errChan)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if err != nil {
			return err
		}
	case <-done:
	}

	return nil
}

// configureNamespace 配置单个命名空间
func (k *CreateK8sClusterTask) configureNamespace(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string, cluster *model.K8sCluster) error {
	if namespace == "" {
		return fmt.Errorf("命名空间名称为空")
	}

	// 确保命名空间存在
	if err := utils.EnsureNamespace(ctx, kubeClient, namespace); err != nil {
		return fmt.Errorf("确保命名空间 %s 存在失败: %w", namespace, err)
	}

	// 应用 LimitRange
	if err := utils.ApplyLimitRange(ctx, kubeClient, namespace, cluster); err != nil {
		return fmt.Errorf("应用 LimitRange 到命名空间 %s 失败: %w", namespace, err)
	}

	// 应用 ResourceQuota
	if err := utils.ApplyResourceQuota(ctx, kubeClient, namespace, cluster); err != nil {
		return fmt.Errorf("应用 ResourceQuota 到命名空间 %s 失败: %w", namespace, err)
	}

	return nil
}
