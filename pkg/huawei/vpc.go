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

package huawei

import (
	"context"
	"fmt"
	"time"

	vpc "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3/model"
)

type VpcService struct {
	sdk *SDK
}

func NewVpcService(sdk *SDK) *VpcService {
	return &VpcService{sdk: sdk}
}

type CreateVpcRequest struct {
	Region          string
	VpcName         string
	CidrBlock       string
	Description     string
	ZoneId          string
	SubnetName      string
	SubnetCidrBlock string
}

type CreateVpcResponseBody struct {
	VpcId    string
	SubnetId string
}

func (v *VpcService) CreateVPC(ctx context.Context, req *CreateVpcRequest) (*CreateVpcResponseBody, error) {
	client, err := v.sdk.CreateVpcClient(req.Region, v.sdk.accessKey)
	if err != nil {
		return nil, err
	}

	vpcReq := &model.CreateVpcRequest{
		Body: &model.CreateVpcRequestBody{
			Vpc: &model.CreateVpcOption{
				Name:        req.VpcName,
				Cidr:        req.CidrBlock,
				Description: &req.Description,
			},
		},
	}
	_, err = client.CreateVpc(vpcReq)
	if err != nil {
		return nil, err
	}

	// 创建后通过ListVpcs查询真实VPC ID
	var vpcId string
	listReq := &model.ListVpcsRequest{}
	listResp, err := client.ListVpcs(listReq)
	if err != nil {
		return nil, err
	}
	for _, v := range *listResp.Vpcs {
		if v.Name == req.VpcName && v.Cidr == req.CidrBlock {
			vpcId = v.Id
			break
		}
	}
	if vpcId == "" {
		return nil, fmt.Errorf("未找到刚刚创建的VPC: %s", req.VpcName)
	}

	// 创建子网
	subnetReq := &model.CreateClouddcnSubnetRequest{
		Body: &model.CreateClouddcnSubnetRequestBody{
			ClouddcnSubnet: &model.CreateClouddcnSubnetOption{
				Name:             req.SubnetName,
				Description:      &req.Description,
				Cidr:             req.SubnetCidrBlock,
				VpcId:            vpcId,
				GatewayIp:        "", // 可选: 自动分配或自定义
				AvailabilityZone: &req.ZoneId,
			},
		},
	}
	subnetResp, err := client.CreateClouddcnSubnet(subnetReq)
	if err != nil {
		return nil, err
	}

	subnetId := ""
	if subnetResp.ClouddcnSubnet != nil {
		subnetId = subnetResp.ClouddcnSubnet.Id
	}
	if subnetId == "" {
		return nil, fmt.Errorf("未获取到子网ID")
	}

	return &CreateVpcResponseBody{
		VpcId:    vpcId,
		SubnetId: subnetId,
	}, nil
}

func (v *VpcService) waitForVpcAvailable(client *vpc.VpcClient, region string, vpcId string) error {
	request := &model.ShowVpcRequest{
		VpcId: vpcId,
	}

	for i := 0; i < 10; i++ {
		response, err := client.ShowVpc(request)
		if err != nil {
			return err
		}

		if response.Vpc.Status == "OK" {
			return nil
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("等待VPC可用超时")
}

func (v *VpcService) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	client, err := v.sdk.CreateVpcClient(region, v.sdk.accessKey)
	if err != nil {
		return err
	}

	// 1. 先列出并删除所有子网
	if err := v.deleteSubnets(client, vpcID); err != nil {
		return err
	}

	// 2. 删除VPC
	deleteReq := &model.DeleteVpcRequest{
		VpcId: vpcID,
	}

	_, err = client.DeleteVpc(deleteReq)
	if err != nil {
		return err
	}

	return nil
}

func (v *VpcService) deleteSubnets(client *vpc.VpcClient, vpcID string) error {
	// 列出VPC下的所有子网
	listReq := &model.ListClouddcnSubnetsRequest{
		VpcId: &vpcID,
	}

	response, err := client.ListClouddcnSubnets(listReq)
	if err != nil {
		return fmt.Errorf("列出子网失败: %w", err)
	}

	if response.ClouddcnSubnets == nil || len(*response.ClouddcnSubnets) == 0 {
		return nil
	}

	// 删除所有子网
	for _, subnet := range *response.ClouddcnSubnets {
		subnetID := subnet.Id

		deleteSubnetReq := &model.DeleteClouddcnSubnetRequest{
			ClouddcnSubnetId: subnetID,
		}

		_, err := client.DeleteClouddcnSubnet(deleteSubnetReq)
		if err != nil {
			return fmt.Errorf("删除子网 %s 失败: %w", subnetID, err)
		}
	}

	return nil
}

type ListVpcsRequest struct {
	Region string
	Page   int
	Size   int
}

type ListVpcsResponseBody struct {
	Vpcs []model.Vpc
}

func (v *VpcService) ListVpcs(ctx context.Context, req *ListVpcsRequest) (*ListVpcsResponseBody, int64, error) {
	var allVpcs []model.Vpc
	var totalCount int64 = 0
	
	client, err := v.sdk.CreateVpcClient(req.Region, v.sdk.accessKey)
	if err != nil {
		return nil, 0, err
	}

	limit := int32(req.Size)
	if limit <= 0 {
		limit = 100
	}

	request := &model.ListVpcsRequest{
		Limit: &limit,
	}

	response, err := client.ListVpcs(request)
	if err != nil {
		return nil, 0, err
	}

	if response.Vpcs != nil {
		allVpcs = *response.Vpcs
		totalCount = int64(len(allVpcs))
	}

	startIdx := (req.Page - 1) * req.Size
	endIdx := req.Page * req.Size
	if startIdx >= len(allVpcs) {
		return &ListVpcsResponseBody{
			Vpcs: []model.Vpc{},
		}, totalCount, nil
	}

	if endIdx > len(allVpcs) {
		endIdx = len(allVpcs)
	}

	return &ListVpcsResponseBody{
		Vpcs: allVpcs[startIdx:endIdx],
	}, totalCount, nil
}

func (v *VpcService) GetVpcDetail(ctx context.Context, region string, vpcID string) (*model.Vpc, error) {
	client, err := v.sdk.CreateVpcClient(region, v.sdk.accessKey)
	if err != nil {
		return nil, err
	}

	request := &model.ShowVpcRequest{
		VpcId: vpcID,
	}

	response, err := client.ShowVpc(request)
	if err != nil {
		return nil, err
	}

	return response.Vpc, nil
}
