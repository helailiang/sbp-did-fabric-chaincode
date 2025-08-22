package did

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"sbp-did-chaincode/common"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
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
	log.Printf("开始注册DID - DID: %s", did)
	if strings.TrimSpace(did) == "" || strings.TrimSpace(didDocument) == "" {
		log.Printf("参数校验失败 - DID或DID文档为空")
		return errors.New("did and didDocument cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)
	log.Printf("DID注册 - 调用者: %s", caller)

	// 调用Permission合约的checkWriteFuncSelectorPermission方法
	hasPermission, err := c.checkWriteFuncSelectorPermission(ctx, caller, "RegisterDid")
	if err != nil {
		log.Printf("权限检查失败: %v", err)
		return fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		log.Printf("权限校验失败 - 调用者: %s, 操作: RegisterDid", caller)
		return errors.New("no permission to register DID")
	}
	log.Printf("权限校验通过 - 调用者: %s, 操作: RegisterDid", caller)

	// 调用Permission合约的checkMethod方法
	if err := c.checkMethod(ctx, did); err != nil {
		log.Printf("DID方法校验失败: %v", err)
		return fmt.Errorf("method validation failed: %v", err)
	}
	log.Printf("DID方法校验通过 - DID: %s", did)

	key := didInfoPrefix + did
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		log.Printf("查询DID状态失败: %v", err)
		return err
	}
	if b != nil {
		log.Printf("DID注册失败 - DID已存在: %s", did)
		return errors.New("did already exists")
	}

	info := DidInfo{
		DidDocument: didDocument,
		Account:     caller,
	}
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		log.Printf("DID信息存储失败: %v", err)
		return err
	}
	log.Printf("DID信息存储成功 - DID: %s, 账户: %s", did, caller)

	// 获取项目配置信息，用于事件通知
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		log.Printf("获取项目配置失败: %v", err)
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
	log.Printf("触发DID注册事件 - DID: %s", did)
	return common.EmitEvent(ctx, "DidRegistered", eventPayload)
}

// UpdateDidDocument 更新DID文档
func (c *DIDChaincode) UpdateDidDocument(ctx contractapi.TransactionContextInterface, did, didDocument string) error {
	log.Printf("开始更新DID文档 - DID: %s", did)
	if strings.TrimSpace(did) == "" || strings.TrimSpace(didDocument) == "" {
		log.Printf("参数校验失败 - DID或DID文档为空")
		return errors.New("did and didDocument cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)
	log.Printf("DID文档更新 - 调用者: %s", caller)

	// 调用Permission合约的checkWriteFuncSelectorPermission方法
	hasPermission, err := c.checkWriteFuncSelectorPermission(ctx, caller, "UpdateDidDocument")
	if err != nil {
		log.Printf("权限检查失败: %v", err)
		return fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		log.Printf("权限校验失败 - 调用者: %s, 操作: UpdateDidDocument", caller)
		return errors.New("no permission to update DID")
	}
	log.Printf("权限校验通过 - 调用者: %s, 操作: UpdateDidDocument", caller)

	// 调用Permission合约的checkMethod方法
	if err := c.checkMethod(ctx, did); err != nil {
		log.Printf("DID方法校验失败: %v", err)
		return fmt.Errorf("method validation failed: %v", err)
	}
	log.Printf("DID方法校验通过 - DID: %s", did)

	key := didInfoPrefix + did
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		log.Printf("DID文档更新失败 - DID不存在: %s", did)
		return errors.New("did not found")
	}
	var info DidInfo
	_ = json.Unmarshal(b, &info)
	if info.Account != common.GetCaller(ctx) {
		log.Printf("权限校验失败 - 只有创建者可以更新DID: %s, 创建者: %s, 调用者: %s", did, info.Account, common.GetCaller(ctx))
		return errors.New("only creator can update did")
	}
	log.Printf("权限校验通过 - 调用者是DID创建者")

	info.DidDocument = didDocument
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		log.Printf("DID文档更新存储失败: %v", err)
		return err
	}
	log.Printf("DID文档更新存储成功 - DID: %s", did)

	// 获取项目配置信息，用于事件通知
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		log.Printf("获取项目配置失败: %v", err)
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
	log.Printf("触发DID文档更新事件 - DID: %s", did)
	return common.EmitEvent(ctx, "DidDocumentUpdated", eventPayload)
}

// GetDidInfo 查询DID文档
func (c *DIDChaincode) GetDidInfo(ctx contractapi.TransactionContextInterface, did string) (string, error) {
	log.Printf("开始查询DID信息 - DID: %s", did)
	if strings.TrimSpace(did) == "" {
		log.Printf("参数校验失败 - DID为空")
		return "", errors.New("did cannot be empty")
	}

	// 调用Permission合约的checkQueryFuncSelectorPermission方法
	hasPermission, err := c.checkQueryFuncSelectorPermission(ctx, "GetDidInfo")
	if err != nil {
		log.Printf("权限检查失败: %v", err)
		return "", fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		log.Printf("权限校验失败 - 调用者: %s, 操作: GetDidInfo", common.GetCaller(ctx))
		return "", errors.New("no permission to query DID")
	}
	log.Printf("权限校验通过 - 调用者: %s, 操作: GetDidInfo", common.GetCaller(ctx))

	key := didInfoPrefix + did
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		log.Printf("DID信息查询失败 - DID不存在: %s", did)
		return "", errors.New("did not found")
	}
	var info DidInfo
	_ = json.Unmarshal(b, &info)
	log.Printf("DID信息查询成功 - DID: %s, 账户: %s", did, info.Account)
	return info.DidDocument, nil
}

// CheckDid 校验DID是否存在
func (c *DIDChaincode) CheckDid(ctx contractapi.TransactionContextInterface, did string) (bool, error) {
	log.Printf("开始校验DID是否存在 - DID: %s", did)
	if strings.TrimSpace(did) == "" {
		log.Printf("参数校验失败 - DID为空")
		return false, errors.New("did cannot be empty")
	}
	key := didInfoPrefix + did
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		log.Printf("DID校验结果 - DID不存在: %s", did)
		return false, nil
	}
	log.Printf("DID校验结果 - DID存在: %s", did)
	return true, nil
}
