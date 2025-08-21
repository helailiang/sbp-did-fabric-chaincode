package vc

import (
	"encoding/json"
	"errors"
	"fmt"
	"sbp-did-chaincode/chaincode/issuer"
	"strings"

	"sbp-did-chaincode/chaincode/common"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
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
	if strings.TrimSpace(vcId) == "" || strings.TrimSpace(vcInfoStr) == "" {
		return errors.New("vcId, vcInfo cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)

	// 获取项目配置信息，用于事件通知
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	var vcInfo VCInfo
	err = json.Unmarshal([]byte(vcInfoStr), &vcInfo)
	if err != nil {
		return errors.New(fmt.Sprintf("vcInfo is invalid: %s", err))
	}
	if strings.TrimSpace(vcInfo.IssuerDid) == "" ||
		strings.TrimSpace(vcInfo.VcHash) == "" ||
		strings.TrimSpace(vcInfo.Algorithm) == "" {
		return errors.New("IssuerDid, VcHash and Algorithm cannot be empty")
	}
	// 检查写权限
	hasPermission, err := c.checkWriteFuncSelectorPermission(ctx, caller, "StoreVCHash")
	if err != nil {
		return fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		return errors.New("no permission to create vc")
	}
	key := vcInfoPrefix + vcId
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		return err
	}
	if b != nil {
		return errors.New("vc already exists")
	}
	// 校验发证方是否存在,且状态正常

	err = c.CheckIssuer(ctx, vcInfo.IssuerDid)
	if err != nil {
		return fmt.Errorf("vc issuer check failed: %v", err)
	}

	vcInfo.VcId = vcId
	b, _ = json.Marshal(vcInfo)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}

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

	return common.EmitEvent(ctx, "VCHashStored", eventPayload)
}

// GetVCInfo 查询VC哈希
func (c *VCChaincode) GetVCInfo(ctx contractapi.TransactionContextInterface, vcId string) (info VCInfo, err error) {
	if strings.TrimSpace(vcId) == "" {
		return info, errors.New("vcId cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)
	// 检查写权限
	hasPermission, err := c.CheckQueryFuncSelectorPermission(ctx, caller, "GetVCInfo")
	if err != nil {
		return info, fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		return info, errors.New("no permission to get vc")
	}
	key := vcInfoPrefix + vcId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return info, errors.New("vc not found")
	}
	//var info VCInfo
	_ = json.Unmarshal(b, &info)
	return info, nil
}

// RevokeVC 吊销VC
func (c *VCChaincode) RevokedVC(ctx contractapi.TransactionContextInterface, vcId string, isRevoked bool) error {
	if strings.TrimSpace(vcId) == "" {
		return errors.New("vcId cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)
	// 获取项目配置信息，用于事件通知
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	// 检查写权限
	hasPermission, err := c.checkWriteFuncSelectorPermission(ctx, caller, "RevokedVC")
	if err != nil {
		return fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		return errors.New("no permission to revoke vc")
	}
	key := vcInfoPrefix + vcId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return errors.New("vc not found")
	}
	var info VCInfo
	_ = json.Unmarshal(b, &info)

	info.IsRevoked = isRevoked
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}

	// 构建包含项目信息的事件数据
	eventData := map[string]interface{}{
		"serviceCode": cfg.ServiceCode,
		"projectCode": cfg.ProjectCode,
		"vcId":        vcId,
		"isRevoked":   isRevoked,
		"sender":      caller,
	}
	eventPayload, _ := json.Marshal(eventData)

	return common.EmitEvent(ctx, "VCRevoked", eventPayload)
}

// GetVCRevokedStatus 查询VC吊销状态
func (c *VCChaincode) GetVCRevokedStatus(ctx contractapi.TransactionContextInterface, vcId string) (bool, error) {
	if strings.TrimSpace(vcId) == "" {
		return false, errors.New("vcId cannot be empty")
	}
	key := vcInfoPrefix + vcId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return false, errors.New("vc not found")
	}
	var info VCInfo
	_ = json.Unmarshal(b, &info)
	return info.IsRevoked, nil
}
