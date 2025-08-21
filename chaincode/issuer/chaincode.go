package issuer

import (
	"encoding/json"
	"fmt"
	"sbp-did-chaincode/chaincode/common"
	"sbp-did-chaincode/chaincode/did"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"github.com/pkg/errors"
)

// 发证方信息结构体
type IssuerInfo struct {
	IssuerDid  string `json:"issuerDid"`  // 发证方ID
	Name       string `json:"name"`       // 发证方名称
	IsDisabled bool   `json:"isDisabled"` // 是否禁用
	Account    string `json:"account"`    // 记录链账户信息用于更新
	//MataDate    IssuerInfoMataDate `json:"mataDate"`    // 发证方信息
	VcTemplates map[string]bool `json:"vcTemplates"` // 发证方模板 key为VC模板的ID
}

type IssuerInfoMataDate struct {
	//联系人、联系电话、联系邮箱、业务描述
	ContactPerson string `json:"contactPerson"` // 联系人
	ContactPhone  string `json:"contactPhone"`  // 联系电话
	ContactEmail  string `json:"contactEmail"`  // 联系邮箱
	Description   string `json:"description"`   // 业务描述
}

// VC模板信息结构体
type VcTemplateInfo struct {
	Id             string `json:"id"`             //VC模板的ID
	IssuerDid      string `json:"issuerDid"`      // 发证方ID
	VcTemplateData string `json:"vcTemplateData"` // vc模板序列化数据
	Account        string `json:"account"`        // 记录链账户信息用于更新
	IsDisabled     bool   `json:"isDisabled"`     // 是否禁用
	//MataDate       VcTemplateInfoMataDate `json:"mataDate"`       // 模板信息
}

type VcTemplateInfoMataDate struct {
	Endpoint    string `json:"endpoint"`    // 请求端点
	Version     string `json:"version"`     // 模板版本
	Description string `json:"description"` // 业务描述
}

// IssuerChaincode 结构体
type IssuerChaincode struct {
	contractapi.Contract
	common.PermissionChecker
}

const (
	// 发证方did标识符对应发证方信息映射
	issuerInfoPrefix = "issuer:info:"
	// - 发证方名称对应映射
	issuerNamePrefix = "issuer:name:"
	// - VC模版id对应模版信息映射
	vcTemplateInfoPrefix = "issuer:template:"
)

// ================== 内部合约调用辅助方法 ==================
//
//// getProjectConfig 获取项目配置信息
//func (c *IssuerChaincode) getProjectConfig(ctx contractapi.TransactionContextInterface) (*common.ProjectConfig, error) {
//	permissionChecker := common.GetGlobalPermissionChecker()
//	if permissionChecker == nil {
//		return nil, fmt.Errorf("global permission checker not initialized")
//	}
//
//	return permissionChecker.GetProjectConfig(ctx)
//}
//
//// checkWriteFuncSelectorPermission 调用Permission合约的权限检查方法
//func (c *IssuerChaincode) checkWriteFuncSelectorPermission(ctx contractapi.TransactionContextInterface, caller, funcName string) (bool, error) {
//	//// 获取调用者账户
//	//caller := common.GetCaller(ctx)
//
//	// 通过全局权限检查器调用Permission合约的方法
//	permissionChecker := common.GetGlobalPermissionChecker()
//	if permissionChecker == nil {
//		return false, fmt.Errorf("global permission checker not initialized")
//	}
//
//	return permissionChecker.CheckWriteFuncSelectorPermission(ctx, caller, funcName)
//}
//
//// CheckQueryFuncSelectorPermission 调用Permission合约的查询权限检查
//func (c *IssuerChaincode) checkQueryFuncSelectorPermission(ctx contractapi.TransactionContextInterface, caller, funcName string) (bool, error) {
//	//// 获取调用者账户
//	//caller := common.GetCaller(ctx)
//
//	// 通过全局权限检查器调用Permission合约的方法
//	permissionChecker := common.GetGlobalPermissionChecker()
//	if permissionChecker == nil {
//		return false, fmt.Errorf("global permission checker not initialized")
//	}
//
//	return permissionChecker.CheckQueryFuncSelectorPermission(ctx, caller, funcName)
//}
//
//// checkIssuerVerificationEnabled 检查发证方审核状态是否启用
//// 返回值说明：
//// - true: 账户可以进行颁发者（注册/更新等操作）
//// - false: 账户不可以进行颁发者（注册/更新等操作）
//// - error: 检查过程中发生错误
//func (c *IssuerChaincode) checkIssuerVerificationEnabled(ctx contractapi.TransactionContextInterface, caller string) (bool, error) {
//	// 通过全局权限检查器调用Permission合约的方法
//	permissionChecker := common.GetGlobalPermissionChecker()
//	if permissionChecker == nil {
//		return false, fmt.Errorf("permission checker not initialized, please check system configuration")
//	}
//
//	// 调用Permission合约检查发证方审核状态
//	isEnabled, err := permissionChecker.CheckIssuerVerificationEnabled(ctx, caller)
//	if err != nil {
//		// 提供更详细的错误信息
//		return false, fmt.Errorf("failed to check issuer verification status: %v", err)
//	}
//
//	return isEnabled, nil
//}
//
//func (c *IssuerChaincode) CheckMethod(ctx contractapi.TransactionContextInterface, issuerDid string) error {
//	// 通过全局权限检查器调用Permission合约的方法
//	permissionChecker := common.GetGlobalPermissionChecker()
//	if permissionChecker == nil {
//		return fmt.Errorf("permission checker not initialized, cannot verify DID method")
//	}
//
//	// 验证DID的method部分是否符合项目配置
//	err := permissionChecker.CheckMethod(ctx, issuerDid)
//	if err != nil {
//		return err
//	}
//	return nil
//}

func (c *IssuerChaincode) CheckDid(ctx contractapi.TransactionContextInterface, didId string) (bool, error) {
	return new(did.DIDChaincode).CheckDid(ctx, didId)
}

// RegisterIssuer 注册发证方
func (c *IssuerChaincode) RegisterIssuer(ctx contractapi.TransactionContextInterface, issuerDid, name string) error {
	if strings.TrimSpace(issuerDid) == "" || strings.TrimSpace(name) == "" {
		return errors.New("issuerDid and name cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)
	// 检查发证方审核状态
	// 如果审核已启用，只有管理员可以注册发证方
	// 如果审核未启用，普通用户也可以注册发证方
	issuerVerificationEnabled, err := c.CheckIssuerVerificationEnabled(ctx, caller)
	if err != nil || !issuerVerificationEnabled {
		return fmt.Errorf("failed to check issuer verification status: %v", err)
	}

	// 检查写权限
	hasPermission, err := c.CheckWriteFuncSelectorPermission(ctx, caller, "RegisterIssuer")
	if err != nil {
		return fmt.Errorf("failed to check write permission: %v", err)
	}
	if !hasPermission {
		return errors.New("no permission to register issuer")
	}
	// 4. 调用PermissionChaincode合约checkMethod方法，验证DID的method部分
	err = c.CheckMethod(ctx, issuerDid)
	if err != nil {
		return fmt.Errorf("DID method validation failed: %v", err)
	}
	// 5. 调用DIDChaincode合约的CheckDid方法，验证DID是否已存在
	existDid, err := c.CheckDid(ctx, issuerDid)
	if err != nil {
		return fmt.Errorf("DID check failed: %v", err)
	}
	if !existDid {
		return fmt.Errorf("did %s not found ", issuerDid)
	}
	// 6. 校验name是否唯一，不唯一抛出异常并回滚交易。
	issuerName := issuerNamePrefix + name
	existIssuerName, err := ctx.GetStub().GetState(issuerName)
	if err != nil {
		return err
	}
	if existIssuerName != nil {
		return errors.New("issuer name already exists")
	}
	if err := ctx.GetStub().PutState(issuerName, []byte(issuerDid)); err != nil {
		return err
	}
	// 7. 校验发证方信息是否存在，存在抛出异常并回滚交易。
	key := issuerInfoPrefix + issuerDid
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		return err
	}
	if b != nil {
		return errors.New("issuer already exists")
	}
	info := IssuerInfo{
		Name:       name,
		IssuerDid:  issuerDid,
		IsDisabled: false,
		Account:    caller,
	}
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}
	// 获取项目配置信息，用于事件通知
	cfg, err := c.GetProjectConfig(ctx)
	if err != nil {
		// 如果获取配置失败，仍然发送事件，但不包含项目信息
		return common.EmitEvent(ctx, "IssuerRegistered", b)
	}

	// 构建包含项目信息的事件数据
	eventData := map[string]interface{}{
		"serviceCode": cfg.ServiceCode,
		"projectCode": cfg.ProjectCode,
		"issuerDid":   issuerDid,
		"name":        name,
		"isDisabled":  info.IsDisabled,
		"sender":      caller,
	}
	eventPayload, _ := json.Marshal(eventData)
	return common.EmitEvent(ctx, "IssuerRegistered", eventPayload)
}

// UpdateIssuer 更新发证方
func (c *IssuerChaincode) UpdateIssuer(ctx contractapi.TransactionContextInterface, issuerDid, name string) error {
	if strings.TrimSpace(issuerDid) == "" || strings.TrimSpace(name) == "" {
		return errors.New("issuerDid and name cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)

	// 检查发证方审核状态
	// 如果审核已启用，只有管理员可以注册发证方
	// 如果审核未启用，普通用户也可以注册发证方
	issuerVerificationEnabled, err := c.CheckIssuerVerificationEnabled(ctx, caller)
	if err != nil || !issuerVerificationEnabled {
		return fmt.Errorf("failed to check issuer verification status: %v", err)
	}

	// 2. 检查写权限
	hasPermission, err := c.CheckWriteFuncSelectorPermission(ctx, caller, "UpdateIssuer")
	if err != nil {
		return fmt.Errorf("failed to check write permission: %v", err)
	}
	if !hasPermission {
		return errors.New("no permission to update issuer")
	}

	// 1. 校验发证方信息是否存在
	key := issuerInfoPrefix + issuerDid
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return errors.New("issuer not found")
	}
	var info IssuerInfo
	_ = json.Unmarshal(b, &info)
	//if info.Account != caller {
	//	return errors.New("only creator can update issuer")
	//}
	// 传入的名称不能与当前发证方名称一致
	if info.Name == name {
		return errors.New("the provided name cannot be the same as the current issuer name")
	}
	// 3. 校验name是否唯一，不唯一抛出异常并回滚交易
	issuerName := issuerNamePrefix + name
	existIssuerName, err := ctx.GetStub().GetState(issuerName)
	if err != nil {
		return err
	}
	if existIssuerName != nil && string(existIssuerName) != issuerDid {
		return errors.New("issuer name already exists")
	}

	// 4. 更新发证方名称映射
	if err := ctx.GetStub().PutState(issuerName, []byte(issuerDid)); err != nil {
		return err
	}

	// 5. 更新发证方信息
	info.Name = name
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}

	// 6. 获取项目配置信息，用于事件通知
	cfg, err := c.GetProjectConfig(ctx)
	if err != nil {
		// 如果获取配置失败，仍然发送事件，但不包含项目信息
		return common.EmitEvent(ctx, "IssuerUpdated", b)
	}

	// 7. 构建包含项目信息的事件数据
	eventData := map[string]interface{}{
		"serviceCode": cfg.ServiceCode,
		"projectCode": cfg.ProjectCode,
		"issuerDid":   issuerDid,
		"name":        name,
		"isDisabled":  info.IsDisabled,
		"sender":      caller,
	}
	eventPayload, _ := json.Marshal(eventData)
	return common.EmitEvent(ctx, "IssuerUpdated", eventPayload)
}

// ChangeIssuerStatus 启停发证方
func (c *IssuerChaincode) ChangeIssuerStatus(ctx contractapi.TransactionContextInterface, issuerDid string, isDisabled bool) error {
	if strings.TrimSpace(issuerDid) == "" {
		return errors.New("issuerDid cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)

	// 检查发证方审核状态
	// 如果审核已启用，只有管理员可以注册发证方
	// 如果审核未启用，普通用户也可以注册发证方
	issuerVerificationEnabled, err := c.CheckIssuerVerificationEnabled(ctx, caller)
	if err != nil || !issuerVerificationEnabled {
		return fmt.Errorf("failed to check issuer verification status: %v", err)
	}

	// 2. 检查写权限
	hasPermission, err := c.CheckWriteFuncSelectorPermission(ctx, caller, "ChangeIssuerStatus")
	if err != nil {
		return fmt.Errorf("failed to check write permission: %v", err)
	}
	if !hasPermission {
		return errors.New("no permission to change issuer status")
	}
	// 1. 校验发证方信息是否存在
	key := issuerInfoPrefix + issuerDid
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return errors.New("issuer not found")
	}
	var info IssuerInfo
	_ = json.Unmarshal(b, &info)
	//if info.Account != caller {
	//	return errors.New("only creator can change status")
	//}
	// 传入的名称不能与当前发证方名称一致
	if info.IsDisabled == isDisabled {
		return errors.New("the provided disabled cannot be the same as the current issuer disabled")
	}
	// 3. 更新发证方状态
	info.IsDisabled = isDisabled
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}

	// 4. 获取项目配置信息，用于事件通知
	cfg, err := c.GetProjectConfig(ctx)
	if err != nil {
		// 如果获取配置失败，仍然发送事件，但不包含项目信息
		return common.EmitEvent(ctx, "IssuerStatusChanged", b)
	}

	// 5. 构建包含项目信息的事件数据
	eventData := map[string]interface{}{
		"serviceCode": cfg.ServiceCode,
		"projectCode": cfg.ProjectCode,
		"issuerDid":   issuerDid,
		//"name":        info.Name,
		"isDisabled": isDisabled,
		"sender":     caller,
	}
	eventPayload, _ := json.Marshal(eventData)
	return common.EmitEvent(ctx, "IssuerStatusChanged", eventPayload)
}

// GetIssuerInfo 查询发证方信息
func (c *IssuerChaincode) GetIssuerInfo(ctx contractapi.TransactionContextInterface, issuerDid string) (string, bool, error) {
	if strings.TrimSpace(issuerDid) == "" {
		return "", false, errors.New("issuerDid cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)
	// 检查发证方审核状态
	// 如果审核已启用，只有管理员可以注册发证方
	// 如果审核未启用，普通用户也可以注册发证方
	issuerVerificationEnabled, err := c.CheckIssuerVerificationEnabled(ctx, caller)
	if err != nil || !issuerVerificationEnabled {
		return "", false, fmt.Errorf("failed to check issuer verification status: %v", err)
	}

	// 2. 检查读权限
	hasPermission, err := c.CheckQueryFuncSelectorPermission(ctx, caller, "GetIssuerInfo")
	if err != nil {
		return "", false, fmt.Errorf("failed to check query permission: %v", err)
	}
	if !hasPermission {
		return "", false, errors.New("no permission to get issuer info")
	}
	key := issuerInfoPrefix + issuerDid
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return "", false, errors.New("issuer not found")
	}
	var info IssuerInfo
	_ = json.Unmarshal(b, &info)
	return info.Name, info.IsDisabled, nil
}

// CheckIssuer 校验发证方是否存在,且状态正常
func (c *IssuerChaincode) getIssuer(ctx contractapi.TransactionContextInterface, issuerDid string) (info IssuerInfo, err error) {
	if strings.TrimSpace(issuerDid) == "" {
		return info, errors.New("issuerDid cannot be empty")
	}
	key := issuerInfoPrefix + issuerDid
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return info, errors.New("issuer not found")
	}
	//var info IssuerInfo
	err = json.Unmarshal(b, &info)
	return info, err
}

// CheckIssuer 校验发证方是否存在,且状态正常
func (c *IssuerChaincode) CheckIssuer(ctx contractapi.TransactionContextInterface, issuerDid string) error {
	//key := issuerInfoPrefix + issuerDid
	//b, err := ctx.GetStub().GetState(key)
	//if err != nil || b == nil {
	//	return errors.New("issuer not found")
	//}
	//var info IssuerInfo
	//_ = json.Unmarshal(b, &info)

	info, err := c.getIssuer(ctx, issuerDid)
	if err != nil {
		return err
	}
	if info.IsDisabled == true {
		return errors.New("issuer is disabled")
	}
	return nil
}

// RegisterVCTemplate 注册VC模板
func (c *IssuerChaincode) RegisterVCTemplate(ctx contractapi.TransactionContextInterface, vcTemplateId, vcTemplateData, issuerDid string) error {
	if strings.TrimSpace(vcTemplateId) == "" || strings.TrimSpace(vcTemplateData) == "" || strings.TrimSpace(issuerDid) == "" {
		return errors.New("vcTemplateId and vcTemplateData and issuerDid cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)
	// 4. 获取项目配置信息，用于事件通知
	cfg, err := c.GetProjectConfig(ctx)
	if err != nil {
		// 如果获取配置失败，仍然发送事件，但不包含项目信息
		return err
	}
	vcTemplateVerificationEnabled, err := c.CheckVCTemplateVerificationEnabled(ctx, caller)
	if err != nil || !vcTemplateVerificationEnabled {
		return fmt.Errorf("failed to check vc template verification status: %v", err)
	}

	// 2. 检查写权限
	hasPermission, err := c.CheckWriteFuncSelectorPermission(ctx, caller, "RegisterVCTemplate")
	if err != nil {
		return fmt.Errorf("failed to check write permission: %v", err)
	}
	if !hasPermission {
		return errors.New("no permission to register vc template")
	}
	// 检查当前issuerDid是否已注册为颁发者
	err = c.CheckIssuer(ctx, issuerDid)
	if err != nil {
		return err
	}

	issuer, err := c.getIssuer(ctx, issuerDid)
	if err != nil {
		return err
	}
	if _, exists := issuer.VcTemplates[vcTemplateId]; exists {
		return errors.New("vc template already exists")
	}
	//// 1. 校验VC模板是否已存在
	vcTemplateKey := vcTemplateInfoPrefix + vcTemplateId
	b, err := ctx.GetStub().GetState(vcTemplateKey)
	if err != nil {
		return err
	}
	if b != nil {
		return errors.New("vc template already exists")
	}

	// 3. 创建VC模板信息
	tpl := VcTemplateInfo{
		Id:             vcTemplateId,
		IssuerDid:      issuerDid,
		VcTemplateData: vcTemplateData,
		Account:        caller,
		IsDisabled:     false,
	}
	tplb, _ := json.Marshal(tpl)
	if err := ctx.GetStub().PutState(vcTemplateKey, tplb); err != nil {
		return err
	}

	issuer.VcTemplates[vcTemplateId] = true
	issuerKey := issuerInfoPrefix + issuerDid
	issuerb, _ := json.Marshal(issuer)
	if err := ctx.GetStub().PutState(issuerKey, issuerb); err != nil {
		return err
	}

	// 5. 构建包含项目信息的事件数据
	eventData := map[string]interface{}{
		"serviceCode":    cfg.ServiceCode,
		"projectCode":    cfg.ProjectCode,
		"vcTemplateId":   vcTemplateId,
		"vcTemplateData": vcTemplateData,
		"isDisabled":     false,
		"sender":         caller,
		"issuerDid":      issuerDid,
	}
	eventPayload, _ := json.Marshal(eventData)
	return common.EmitEvent(ctx, "VCTemplateRegistered", eventPayload)
}

// UpdateVCTemplate 更新VC模板
func (c *IssuerChaincode) UpdateVCTemplate(ctx contractapi.TransactionContextInterface, vcTemplateId, vcTemplateData string) error {
	if strings.TrimSpace(vcTemplateId) == "" || strings.TrimSpace(vcTemplateData) == "" {
		return errors.New("vcTemplateId and vcTemplateData cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)
	// 4. 获取项目配置信息，用于事件通知
	cfg, err := c.GetProjectConfig(ctx)
	if err != nil {
		return err
	}
	vcTemplateVerificationEnabled, err := c.CheckVCTemplateVerificationEnabled(ctx, caller)
	if err != nil || !vcTemplateVerificationEnabled {
		return fmt.Errorf("failed to check vc template verification status: %v", err)
	}
	// 2. 检查写权限
	hasPermission, err := c.CheckWriteFuncSelectorPermission(ctx, caller, "UpdateVCTemplate")
	if err != nil {
		return fmt.Errorf("failed to check write permission: %v", err)
	}
	if !hasPermission {
		return errors.New("no permission to update vc template")
	}
	// 1. 校验VC模板是否存在
	key := vcTemplateInfoPrefix + vcTemplateId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return errors.New("vc template not found")
	}
	var tpl VcTemplateInfo
	_ = json.Unmarshal(b, &tpl)
	//if tpl.Account != caller {
	//	return errors.New("only creator can update vc template")
	//}

	// vc模板状态
	// 3. 更新VC模板数据是否正常
	if tpl.IsDisabled {
		return errors.New("vc template is disabled")
	}
	tpl.VcTemplateData = vcTemplateData
	b, _ = json.Marshal(tpl)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}

	// 5. 构建包含项目信息的事件数据
	eventData := map[string]interface{}{
		"serviceCode":    cfg.ServiceCode,
		"projectCode":    cfg.ProjectCode,
		"vcTemplateId":   vcTemplateId,
		"vcTemplateData": vcTemplateData,
		"isDisabled":     tpl.IsDisabled,
		"sender":         caller,
		"issuerDid":      tpl.IssuerDid,
	}
	eventPayload, _ := json.Marshal(eventData)
	return common.EmitEvent(ctx, "VCTemplateUpdated", eventPayload)
}

// ChangeVCTemplateStatus 启停VC模板
func (c *IssuerChaincode) ChangeVCTemplateStatus(ctx contractapi.TransactionContextInterface, vcTemplateId string, isDisabled bool) error {
	if strings.TrimSpace(vcTemplateId) == "" {
		return errors.New("vcTemplateId cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)
	// 4. 获取项目配置信息，用于事件通知
	cfg, err := c.GetProjectConfig(ctx)
	if err != nil {
		return err
	}
	vcTemplateVerificationEnabled, err := c.CheckVCTemplateVerificationEnabled(ctx, caller)
	if err != nil || !vcTemplateVerificationEnabled {
		return fmt.Errorf("failed to check vc template verification status: %v", err)
	}
	// 检查写权限
	hasPermission, err := c.CheckWriteFuncSelectorPermission(ctx, caller, "ChangeVCTemplateStatus")
	if err != nil {
		return fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		return errors.New("no permission to change vc template status")
	}
	//校验VC模版信息是否存在
	key := vcTemplateInfoPrefix + vcTemplateId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return errors.New("vc template not found")
	}
	var tpl VcTemplateInfo
	_ = json.Unmarshal(b, &tpl)
	//if tpl.Account != caller {
	//	return errors.New("only creator can change status")
	//}

	//校验原来数据里的isDisabled比对此次入参isDisabled是否相同
	if tpl.IsDisabled == isDisabled {
		return errors.New("the provided disabled cannot be the same as the current vc template disabled")
	}
	tpl.IsDisabled = isDisabled
	b, _ = json.Marshal(tpl)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}
	// 5. 构建包含项目信息的事件数据
	eventData := map[string]interface{}{
		"serviceCode":  cfg.ServiceCode,
		"projectCode":  cfg.ProjectCode,
		"vcTemplateId": vcTemplateId,
		"isDisabled":   tpl.IsDisabled,
		"sender":       caller,
		"issuerDid":    tpl.IssuerDid,
	}
	eventPayload, _ := json.Marshal(eventData)
	return common.EmitEvent(ctx, "VCTemplateStatusChanged", eventPayload)
}

// GetVCTemplateInfo 查询VC模板信息
func (c *IssuerChaincode) GetVCTemplateInfo(ctx contractapi.TransactionContextInterface, vcTemplateId string) (string, bool, error) {
	if strings.TrimSpace(vcTemplateId) == "" {
		return "", false, errors.New("vcTemplateId cannot be empty")
	}
	// 获取调用者账户
	caller := common.GetCaller(ctx)
	vcTemplateVerificationEnabled, err := c.CheckVCTemplateVerificationEnabled(ctx, caller)
	if err != nil || !vcTemplateVerificationEnabled {
		return "", true, fmt.Errorf("failed to check vc template verification status: %v", err)
	}
	// 检查写权限
	hasPermission, err := c.CheckQueryFuncSelectorPermission(ctx, caller, "GetVCTemplateInfo")
	if err != nil {
		return "", true, fmt.Errorf("permission check failed: %v", err)
	}
	if !hasPermission {
		return "", true, errors.New("no permission to change vc template status")
	}
	key := vcTemplateInfoPrefix + vcTemplateId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return "", false, errors.New("vc template not found")
	}
	var tpl VcTemplateInfo
	_ = json.Unmarshal(b, &tpl)
	return tpl.VcTemplateData, tpl.IsDisabled, nil
}

// CheckVCTemplate 校验VC模板是否存在
func (c *IssuerChaincode) CheckVCTemplate(ctx contractapi.TransactionContextInterface, vcTemplateId string) (bool, error) {
	if strings.TrimSpace(vcTemplateId) == "" {
		return false, errors.New("vcTemplateId cannot be empty")
	}
	key := vcTemplateInfoPrefix + vcTemplateId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return false, nil
	}
	return true, nil
}
