package did

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"sbp-did-chaincode/chaincode/common"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// DID信息结构体
type DidInfo struct {
	DidDocument string `json:"didDocument"` // DID文档
	Account     string `json:"sender"`      // 注册账户
}

// DIDChaincode 结构体
type DIDChaincode struct {
	contractapi.Contract
}

const didInfoPrefix = "did:info:"

// ================== 内部合约调用辅助方法 ==================

// getProjectConfig 获取项目配置信息
func (c *DIDChaincode) getProjectConfig(ctx contractapi.TransactionContextInterface) (*common.ProjectConfig, error) {
	permissionChecker := common.GetGlobalPermissionChecker()
	if permissionChecker == nil {
		return nil, fmt.Errorf("global permission checker not initialized")
	}

	return permissionChecker.GetProjectConfig(ctx)
}

// checkWriteFuncSelectorPermission 调用Permission合约的权限检查方法
func (c *DIDChaincode) checkWriteFuncSelectorPermission(ctx contractapi.TransactionContextInterface, caller, funcName string) (bool, error) {
	// 通过全局权限检查器调用Permission合约的方法
	permissionChecker := common.GetGlobalPermissionChecker()
	if permissionChecker == nil {
		return false, fmt.Errorf("global permission checker not initialized")
	}

	return permissionChecker.CheckWriteFuncSelectorPermission(ctx, caller, funcName)
}

// checkMethod 调用Permission合约的DID方法检查
func (c *DIDChaincode) checkMethod(ctx contractapi.TransactionContextInterface, did string) error {
	// 通过全局权限检查器调用Permission合约的方法
	permissionChecker := common.GetGlobalPermissionChecker()
	if permissionChecker == nil {
		return fmt.Errorf("global permission checker not initialized")
	}

	return permissionChecker.CheckMethod(ctx, did)
}

// checkQueryFuncSelectorPermission 调用Permission合约的查询权限检查方法
func (c *DIDChaincode) checkQueryFuncSelectorPermission(ctx contractapi.TransactionContextInterface, funcName string) (bool, error) {
	// 获取调用者账户
	caller := common.GetCaller(ctx)

	// 通过全局权限检查器调用Permission合约的方法
	permissionChecker := common.GetGlobalPermissionChecker()
	if permissionChecker == nil {
		return false, fmt.Errorf("global permission checker not initialized")
	}

	return permissionChecker.CheckQueryFuncSelectorPermission(ctx, caller, funcName)
}

// ================== 主要业务方法 ==================

// RegisterDid 注册DID
func (c *DIDChaincode) RegisterDid(ctx contractapi.TransactionContextInterface, did, didDocument string) error {
	if strings.TrimSpace(did) == "" || strings.TrimSpace(didDocument) == "" {
		return errors.New("did and didDocument cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)

	// 调用Permission合约的checkWriteFuncSelectorPermission方法
	hasPermission, err := c.checkWriteFuncSelectorPermission(ctx, caller, "RegisterDid")
	if err != nil {
		return fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		return errors.New("no permission to register DID")
	}

	// 调用Permission合约的checkMethod方法
	if err := c.checkMethod(ctx, did); err != nil {
		return fmt.Errorf("method validation failed: %v", err)
	}

	key := didInfoPrefix + did
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		return err
	}
	if b != nil {
		return errors.New("did already exists")
	}

	info := DidInfo{
		DidDocument: didDocument,
		Account:     caller,
	}
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}

	// 获取项目配置信息，用于事件通知
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		// 如果获取配置失败，仍然发送事件，但不包含项目信息
		return common.EmitEvent(ctx, "DidRegistered", b)
	}

	// 构建包含项目信息的事件数据
	eventData := map[string]interface{}{
		"serviceCode": cfg.ServiceCode,
		"projectCode": cfg.ProjectCode,
		"did":         did,
		"didDocument": info,
		"sender":      info.Account,
	}
	eventPayload, _ := json.Marshal(eventData)

	return common.EmitEvent(ctx, "DidRegistered", eventPayload)
}

// UpdateDidDocument 更新DID文档
func (c *DIDChaincode) UpdateDidDocument(ctx contractapi.TransactionContextInterface, did, didDocument string) error {
	if strings.TrimSpace(did) == "" || strings.TrimSpace(didDocument) == "" {
		return errors.New("did and didDocument cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)

	// 调用Permission合约的checkWriteFuncSelectorPermission方法
	hasPermission, err := c.checkWriteFuncSelectorPermission(ctx, caller, "UpdateDidDocument")
	if err != nil {
		return fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		return errors.New("no permission to update DID")
	}

	// 调用Permission合约的checkMethod方法
	if err := c.checkMethod(ctx, did); err != nil {
		return fmt.Errorf("method validation failed: %v", err)
	}

	key := didInfoPrefix + did
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return errors.New("did not found")
	}
	var info DidInfo
	_ = json.Unmarshal(b, &info)
	if info.Account != common.GetCaller(ctx) {
		return errors.New("only creator can update did")
	}

	info.DidDocument = didDocument
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}

	// 获取项目配置信息，用于事件通知
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		// 如果获取配置失败，仍然发送事件，但不包含项目信息
		return common.EmitEvent(ctx, "DidDocumentUpdated", b)
	}

	// 构建包含项目信息的事件数据
	eventData := map[string]interface{}{
		"serviceCode": cfg.ServiceCode,
		"projectCode": cfg.ProjectCode,
		"did":         did,
		"didDocument": info,
		"sender":      common.GetCaller(ctx),
	}
	eventPayload, _ := json.Marshal(eventData)

	return common.EmitEvent(ctx, "DidDocumentUpdated", eventPayload)
}

// GetDidInfo 查询DID文档
func (c *DIDChaincode) GetDidInfo(ctx contractapi.TransactionContextInterface, did string) (string, error) {
	if strings.TrimSpace(did) == "" {
		return "", errors.New("did cannot be empty")
	}

	// 调用Permission合约的checkQueryFuncSelectorPermission方法
	hasPermission, err := c.checkQueryFuncSelectorPermission(ctx, "GetDidInfo")
	if err != nil {
		return "", fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		return "", errors.New("no permission to query DID")
	}

	key := didInfoPrefix + did
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return "", errors.New("did not found")
	}
	var info DidInfo
	_ = json.Unmarshal(b, &info)
	return info.DidDocument, nil
}

// CheckDid 校验DID是否存在
func (c *DIDChaincode) CheckDid(ctx contractapi.TransactionContextInterface, did string) (bool, error) {
	if strings.TrimSpace(did) == "" {
		return false, errors.New("did cannot be empty")
	}
	key := didInfoPrefix + did
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return false, nil
	}
	return true, nil
}
