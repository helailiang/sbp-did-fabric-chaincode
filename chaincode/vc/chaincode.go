package vc

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sbp-did-chaincode/issuer"
	"strings"

	"sbp-did-chaincode/common"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// VC存证信息结构体
type VCInfo struct {
	VcId string `json:"vcId"` // vc唯一码
	//SubjectDid string `json:"subjectDid,omitempty"` // 持有者DID
	IssuerDid string `json:"issuerDid"` // 发证方DID
	VcHash    string `json:"vcHash"`    // VC内容 hash值
	IsRevoked bool   `json:"isRevoked"` // 是否已吊销 true: 已吊销
	Algorithm string `json:"algorithm"` //哈希算法
}

// VCChaincode 结构体
type VCChaincode struct {
	contractapi.Contract
	common.PermissionChecker
}

const vcInfoPrefix = "vc:info:"

// ================== 内部合约调用辅助方法 ==================

// getProjectConfig 获取项目配置信息
func (c *VCChaincode) getProjectConfig(ctx contractapi.TransactionContextInterface) (*common.ProjectConfig, error) {
	permissionChecker := common.GetGlobalPermissionChecker()
	if permissionChecker == nil {
		return nil, fmt.Errorf("global permission checker not initialized")
	}

	return permissionChecker.GetProjectConfig(ctx)
}

// checkWriteFuncSelectorPermission 调用Permission合约的权限检查方法
func (c *VCChaincode) checkWriteFuncSelectorPermission(ctx contractapi.TransactionContextInterface, caller, funcName string) (bool, error) {
	// 通过全局权限检查器调用Permission合约的方法
	permissionChecker := common.GetGlobalPermissionChecker()
	if permissionChecker == nil {
		return false, fmt.Errorf("global permission checker not initialized")
	}

	return permissionChecker.CheckWriteFuncSelectorPermission(ctx, caller, funcName)
}

func (c *VCChaincode) CheckIssuer(ctx contractapi.TransactionContextInterface, didId string) error {
	return new(issuer.IssuerChaincode).CheckIssuer(ctx, didId)
}

// StoreVCHash 创建VC存证
func (c *VCChaincode) StoreVCHash(ctx contractapi.TransactionContextInterface, vcId, vcInfoStr string) error {
	log.Printf("开始创建VC存证 - VC ID: %s", vcId)
	if strings.TrimSpace(vcId) == "" || strings.TrimSpace(vcInfoStr) == "" {
		log.Printf("参数校验失败 - VC ID或VC信息为空")
		return errors.New("vcId, vcInfo cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)
	log.Printf("VC存证创建 - 调用者: %s", caller)

	// 获取项目配置信息，用于事件通知
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		log.Printf("获取项目配置失败: %v", err)
		return err
	}

	var vcInfo VCInfo
	err = json.Unmarshal([]byte(vcInfoStr), &vcInfo)
	if err != nil {
		log.Printf("VC信息解析失败: %v", err)
		return errors.New(fmt.Sprintf("vcInfo is invalid: %s", err))
	}
	log.Printf("VC信息解析成功 - 发证方DID: %s, 哈希算法: %s", vcInfo.IssuerDid, vcInfo.Algorithm)

	if strings.TrimSpace(vcInfo.IssuerDid) == "" ||
		strings.TrimSpace(vcInfo.VcHash) == "" ||
		strings.TrimSpace(vcInfo.Algorithm) == "" {
		log.Printf("VC信息校验失败 - 发证方DID、哈希值或算法为空")
		return errors.New("IssuerDid, VcHash and Algorithm cannot be empty")
	}
	log.Printf("VC信息校验通过")

	// 检查写权限
	hasPermission, err := c.checkWriteFuncSelectorPermission(ctx, caller, "StoreVCHash")
	if err != nil {
		log.Printf("权限检查失败: %v", err)
		return fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		log.Printf("权限校验失败 - 调用者: %s, 操作: StoreVCHash", caller)
		return errors.New("no permission to create vc")
	}
	log.Printf("权限校验通过 - 调用者: %s, 操作: StoreVCHash", caller)

	key := vcInfoPrefix + vcId
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		log.Printf("查询VC状态失败: %v", err)
		return err
	}
	if b != nil {
		log.Printf("VC存证创建失败 - VC已存在: %s", vcId)
		return errors.New("vc already exists")
	}
	log.Printf("VC状态校验通过 - VC不存在: %s", vcId)

	// 校验发证方是否存在,且状态正常
	err = c.CheckIssuer(ctx, vcInfo.IssuerDid)
	if err != nil {
		log.Printf("发证方校验失败: %v", err)
		return fmt.Errorf("vc issuer check failed: %v", err)
	}
	log.Printf("发证方校验通过 - 发证方DID: %s", vcInfo.IssuerDid)

	vcInfo.VcId = vcId
	b, _ = json.Marshal(vcInfo)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		log.Printf("VC信息存储失败: %v", err)
		return err
	}
	log.Printf("VC信息存储成功 - VC ID: %s, 发证方DID: %s", vcId, vcInfo.IssuerDid)

	// 构建包含项目信息的事件数据
	eventData := map[string]interface{}{
		"serviceCode": cfg.ServiceCode,
		"projectCode": cfg.ProjectCode,
		"vcId":        vcId,
		"vcHash":      vcInfo.VcHash,
		"algorithm":   vcInfo.Algorithm,
		"issuerDid":   vcInfo.IssuerDid,
		"sender":      caller,
	}
	eventPayload, _ := json.Marshal(eventData)
	log.Printf("触发VC存证创建事件 - VC ID: %s", vcId)
	return common.EmitEvent(ctx, "VCHashStored", eventPayload)
}

// GetVCInfo 查询VC哈希
func (c *VCChaincode) GetVCInfo(ctx contractapi.TransactionContextInterface, vcId string) (info VCInfo, err error) {
	log.Printf("开始查询VC信息 - VC ID: %s", vcId)
	if strings.TrimSpace(vcId) == "" {
		log.Printf("参数校验失败 - VC ID为空")
		return info, errors.New("vcId cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)
	log.Printf("VC信息查询 - 调用者: %s", caller)

	// 检查写权限
	hasPermission, err := c.CheckQueryFuncSelectorPermission(ctx, caller, "GetVCInfo")
	if err != nil {
		log.Printf("权限检查失败: %v", err)
		return info, fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		log.Printf("权限校验失败 - 调用者: %s, 操作: GetVCInfo", caller)
		return info, errors.New("no permission to get vc")
	}
	log.Printf("权限校验通过 - 调用者: %s, 操作: GetVCInfo", caller)

	key := vcInfoPrefix + vcId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		log.Printf("VC信息查询失败 - VC不存在: %s", vcId)
		return info, errors.New("vc not found")
	}
	//var info VCInfo
	_ = json.Unmarshal(b, &info)
	log.Printf("VC信息查询成功 - VC ID: %s, 发证方DID: %s, 吊销状态: %t", vcId, info.IssuerDid, info.IsRevoked)
	return info, nil
}

// RevokeVC 吊销VC
func (c *VCChaincode) RevokedVC(ctx contractapi.TransactionContextInterface, vcId string, isRevoked bool) error {
	log.Printf("开始吊销VC - VC ID: %s, 吊销状态: %t", vcId, isRevoked)
	if strings.TrimSpace(vcId) == "" {
		log.Printf("参数校验失败 - VC ID为空")
		return errors.New("vcId cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)
	log.Printf("VC吊销操作 - 调用者: %s", caller)

	// 获取项目配置信息，用于事件通知
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		log.Printf("获取项目配置失败: %v", err)
		return err
	}

	// 检查写权限
	hasPermission, err := c.checkWriteFuncSelectorPermission(ctx, caller, "RevokedVC")
	if err != nil {
		log.Printf("权限检查失败: %v", err)
		return fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		log.Printf("权限校验失败 - 调用者: %s, 操作: RevokedVC", caller)
		return errors.New("no permission to revoke vc")
	}
	log.Printf("权限校验通过 - 调用者: %s, 操作: RevokedVC", caller)

	key := vcInfoPrefix + vcId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		log.Printf("VC吊销失败 - VC不存在: %s", vcId)
		return errors.New("vc not found")
	}
	var info VCInfo
	_ = json.Unmarshal(b, &info)
	log.Printf("获取VC信息成功 - 当前吊销状态: %t", info.IsRevoked)

	info.IsRevoked = isRevoked
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		log.Printf("VC吊销状态更新失败: %v", err)
		return err
	}
	log.Printf("VC吊销状态更新成功 - VC ID: %s, 新吊销状态: %t", vcId, isRevoked)

	// 构建包含项目信息的事件数据
	eventData := map[string]interface{}{
		"serviceCode": cfg.ServiceCode,
		"projectCode": cfg.ProjectCode,
		"vcId":        vcId,
		"isRevoked":   isRevoked,
		"sender":      caller,
	}
	eventPayload, _ := json.Marshal(eventData)
	log.Printf("触发VC吊销事件 - VC ID: %s, 吊销状态: %t", vcId, isRevoked)
	return common.EmitEvent(ctx, "VCRevoked", eventPayload)
}

// GetVCRevokedStatus 查询VC吊销状态
func (c *VCChaincode) GetVCRevokedStatus(ctx contractapi.TransactionContextInterface, vcId string) (bool, error) {
	log.Printf("开始查询VC吊销状态 - VC ID: %s", vcId)
	if strings.TrimSpace(vcId) == "" {
		log.Printf("参数校验失败 - VC ID为空")
		return false, errors.New("vcId cannot be empty")
	}
	key := vcInfoPrefix + vcId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		log.Printf("VC吊销状态查询失败 - VC不存在: %s", vcId)
		return false, errors.New("vc not found")
	}
	var info VCInfo
	_ = json.Unmarshal(b, &info)
	log.Printf("VC吊销状态查询成功 - VC ID: %s, 吊销状态: %t", vcId, info.IsRevoked)
	return info.IsRevoked, nil
}
