/*
Package accesscontrol 提供基于Hyperledger Fabric的权限控制功能

该包实现了SBP-DID项目的权限管理系统，包括：
- 项目配置管理（私有/公开、验证开关、写权限等）
- 账户权限管理（函数选择器权限的批量授权/撤销）
- 内部校验方法（权限校验、状态校验等）

主要功能：
1. 项目初始化配置
2. 动态权限管理
3. 项目状态控制（启用/停用）
4. 管理员权限管理

作者: SBP-DID开发团队
版本: v0.1
*/

package accesscontrol

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"sbp-did-chaincode/chaincode/common"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// ProjectConfig 项目配置结构体
// 存储SBP-DID项目的核心配置信息，包括权限控制、验证开关、项目标识等
// 使用common.ProjectConfig类型，避免循环导入

// AccountSelector 账户权限结构体
// 用于批量操作账户权限，包含账户标识和对应的函数权限列表
type AccountSelector struct {
	Account   string   // 链账户SKI（Subject Key Identifier），用于唯一标识账户
	FuncNames []string // 函数名列表，该账户被授权的函数名称集合
}

// PermissionChaincode 权限控制链码结构体
// 继承自contractapi.Contract，实现SBP-DID项目的权限管理功能
// 提供项目配置、权限管理、状态控制等核心功能
type PermissionChaincode struct {
	contractapi.Contract
}

const (
	// projectConfigKey 项目配置在链上的存储键
	// 格式：permission:projectConfig
	projectConfigKey = "permission:projectConfig"

	// selectorPermPrefix 账户权限在链上的存储键前缀
	// 格式：permission:selectorPerm:{account}
	selectorPermPrefix = "permission:selectorPerm:"
)

// ================== 项目配置相关 ==================

// InitProject 初始化项目配置
// 该方法用于首次部署合约时初始化SBP-DID项目的基本配置
// 只能调用一次，后续修改需要通过其他方法进行
//
// 参数说明：
// - ctx: 交易上下文，包含链码存根和调用者信息
// - method: 项目method名称，用于DID标识符验证
// - isPrivate: 项目是否私有，私有项目需要权限验证
// - enableIssuerVerification: 是否启用发证方验证
// - enableVcVerification: 是否启用VC模板验证
// - enableWritePermission: 是否启用写权限控制
// - serviceCode: 服务编码，标识服务实例
// - projectCode: 项目编码，标识具体项目
//
// 返回值：
// - error: 成功返回nil，失败返回错误信息
//
// 权限要求：只有管理员可以调用此方法
func (c *PermissionChaincode) InitProject(
	ctx contractapi.TransactionContextInterface,
	method string,
	isPrivate, enableIssuerVerification, enableVcVerification, enableWritePermission bool,
	serviceCode, projectCode string,
) error {
	//// 权限校验：只有管理员可以初始化项目
	//if !common.IsAdmin(ctx) {
	//	return errors.New("only admin can init project")
	//}

	// 参数校验：关键参数不能为空
	if strings.TrimSpace(method) == "" || strings.TrimSpace(serviceCode) == "" || strings.TrimSpace(projectCode) == "" {
		return errors.New("method, serviceCode, projectCode cannot be empty")
	}

	// 获取调用者身份作为初始管理员
	caller := common.GetCaller(ctx)

	// 构建项目配置对象
	cfg := common.ProjectConfig{
		EnableVCTemplateVerification: enableVcVerification,
		EnableIssuerVerification:     enableIssuerVerification,
		EnableWritePermission:        enableWritePermission,
		Method:                       method,
		Paused:                       false, // 初始状态为启用
		IsProjectPrivate:             isPrivate,
		ServiceCode:                  serviceCode,
		ProjectCode:                  projectCode,
		Admins:                       []string{caller}, // 调用者成为初始管理员
	}

	// 序列化配置并存储到链上
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}
	//  设置全局权限检查器
	common.SetGlobalPermissionChecker(c)
	// 触发项目初始化事件
	eventPayload, _ := json.Marshal(&common.ProjectConfig{
		ServiceCode:                  serviceCode,
		ProjectCode:                  projectCode,
		Method:                       method,
		IsProjectPrivate:             isPrivate,
		EnableVCTemplateVerification: enableVcVerification,
		EnableIssuerVerification:     enableIssuerVerification,
		EnableWritePermission:        enableWritePermission,
		Paused:                       false,
		Admins:                       []string{caller},
	})
	return common.EmitEvent(ctx, "ProjectInitialized", eventPayload)
}

// ChangePrivateStatus 更改项目私有/公开状态
// 该方法用于动态调整项目的可见性，影响后续的权限验证逻辑
//
// 参数说明：
// - ctx: 交易上下文
// - isPrivate: 新的私有状态，true表示私有，false表示公开
//
// 返回值：
// - error: 成功返回nil，失败返回错误信息
//
// 权限要求：只有管理员可以调用此方法
// 业务逻辑：
// - 私有项目：所有操作都需要权限验证
// - 公开项目：查询操作无需权限验证，写操作仍需要权限验证
func (c *PermissionChaincode) ChangePrivateStatus(ctx contractapi.TransactionContextInterface, isPrivate bool) error {
	// 获取当前项目配置
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}

	// 权限校验：只有管理员可以更改项目状态
	if !common.IsAdmin(ctx, cfg.Admins) {
		return errors.New("only admin can change private status")
	}

	// 状态校验：避免重复设置相同状态
	if cfg.IsProjectPrivate == isPrivate {
		return errors.New("private status is already the same")
	}

	// 更新项目配置
	cfg.IsProjectPrivate = isPrivate
	//cfg.IsPrivate = isPrivate

	// 序列化并存储到链上
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}

	// 触发状态变更事件
	//payload, _ := json.Marshal(map[string]interface{}{"serviceCode": cfg.ServiceCode, "projectCode": cfg.ProjectCode, "isPrivate": isPrivate})
	payload, _ := json.Marshal(&common.ProjectConfig{
		ServiceCode:      cfg.ServiceCode,
		ProjectCode:      cfg.ProjectCode,
		IsProjectPrivate: isPrivate,
	})
	return common.EmitEvent(ctx, "PrivateStatusChanged", payload)
}

// ChangeMethod 更改method名称
func (c *PermissionChaincode) ChangeMethod(ctx contractapi.TransactionContextInterface, method string) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx, cfg.Admins) {
		return errors.New("only admin can change method")
	}
	if cfg.Paused {
		return errors.New("project is paused")
	}
	if strings.TrimSpace(method) == "" {
		return errors.New("method cannot be empty")
	}
	if cfg.Method == method {
		return errors.New("method is already the same")
	}

	cfg.Method = method
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}

	//payload, _ := json.Marshal(map[string]interface{}{"method": method})
	payload, _ := json.Marshal(&common.ProjectConfig{
		ServiceCode: cfg.ServiceCode,
		ProjectCode: cfg.ProjectCode,
		Method:      method,
	})
	return common.EmitEvent(ctx, "MethodChanged", payload)
}

// ChangeEnableVCTemplateVerification 更改VC模板验证开关
func (c *PermissionChaincode) ChangeEnableVCTemplateVerification(ctx contractapi.TransactionContextInterface, enableVCTemplateVerification bool) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx, cfg.Admins) {
		return errors.New("only admin can change VC template verification")
	}
	if cfg.Paused {
		return errors.New("project is paused")
	}
	if cfg.EnableVCTemplateVerification == enableVCTemplateVerification {
		return errors.New("VC template verification status is already the same")
	}

	cfg.EnableVCTemplateVerification = enableVCTemplateVerification
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}

	//payload, _ := json.Marshal(map[string]interface{}{"enableVCTemplateVerification": enableVCTemplateVerification})
	payload, _ := json.Marshal(&common.ProjectConfig{
		ServiceCode:                  cfg.ServiceCode,
		ProjectCode:                  cfg.ProjectCode,
		EnableVCTemplateVerification: enableVCTemplateVerification,
	})
	return common.EmitEvent(ctx, "EnableVCTemplateVerificationChanged", payload)
}

// ChangeEnableIssuerVerification 更改Issuer验证开关
func (c *PermissionChaincode) ChangeEnableIssuerVerification(ctx contractapi.TransactionContextInterface, enableIssuerVerification bool) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx, cfg.Admins) {
		return errors.New("only admin can change issuer verification")
	}
	if cfg.Paused {
		return errors.New("project is paused")
	}
	if cfg.EnableIssuerVerification == enableIssuerVerification {
		return errors.New("issuer verification status is already the same")
	}

	cfg.EnableIssuerVerification = enableIssuerVerification
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}

	//payload, _ := json.Marshal(map[string]interface{}{"enableIssuerVerification": enableIssuerVerification})
	payload, _ := json.Marshal(&common.ProjectConfig{
		ServiceCode:              cfg.ServiceCode,
		ProjectCode:              cfg.ProjectCode,
		EnableIssuerVerification: enableIssuerVerification,
	})
	return common.EmitEvent(ctx, "EnableIssuerVerificationChanged", payload)
}

// ChangeEnableWritePermission 更改写权限状态
func (c *PermissionChaincode) ChangeEnableWritePermission(ctx contractapi.TransactionContextInterface, enableWritePermission bool) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx, cfg.Admins) {
		return errors.New("only admin can change write permission")
	}
	if cfg.Paused {
		return errors.New("project is paused")
	}
	if cfg.EnableWritePermission == enableWritePermission {
		return errors.New("write permission status is already the same")
	}

	cfg.EnableWritePermission = enableWritePermission
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}

	//payload, _ := json.Marshal(map[string]interface{}{"enableWritePermission": enableWritePermission})
	payload, _ := json.Marshal(&common.ProjectConfig{
		ServiceCode:           cfg.ServiceCode,
		ProjectCode:           cfg.ProjectCode,
		EnableWritePermission: enableWritePermission,
	})
	return common.EmitEvent(ctx, "EnableWritePermissionChanged", payload)
}

// Pause 项目停用
func (c *PermissionChaincode) Pause(ctx contractapi.TransactionContextInterface) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx, cfg.Admins) {
		return errors.New("only admin can pause project")
	}

	cfg.Paused = true
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}

	// 触发项目停用事件
	eventPayload, _ := json.Marshal(&common.ProjectConfig{
		ServiceCode: cfg.ServiceCode,
		ProjectCode: cfg.ProjectCode,
		Paused:      true,
		Admins:      []string{common.GetCaller(ctx)},
	})
	return common.EmitEvent(ctx, "Paused", eventPayload)
}

// Unpause 项目启用
func (c *PermissionChaincode) Unpause(ctx contractapi.TransactionContextInterface) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx, cfg.Admins) {
		return errors.New("only admin can unpause project")
	}

	cfg.Paused = false
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}

	// 触发项目启用事件
	eventPayload, _ := json.Marshal(&common.ProjectConfig{
		ServiceCode: cfg.ServiceCode,
		ProjectCode: cfg.ProjectCode,
		Paused:      false,
		Admins:      []string{common.GetCaller(ctx)},
	})
	return common.EmitEvent(ctx, "Unpaused", eventPayload)
}

// TransferAdminRole 转移超级管理员权限
func (c *PermissionChaincode) TransferAdminRole(ctx contractapi.TransactionContextInterface, newAdmin string) error {

	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx, cfg.Admins) {
		return errors.New("only admin can transfer admin role")
	}
	if strings.TrimSpace(newAdmin) == "" {
		return errors.New("new admin cannot be empty")
	}

	// 检查新管理员是否已在列表中
	for _, admin := range cfg.Admins {
		if admin == newAdmin {
			return errors.New("new admin is already in admin list")
		}
	}

	cfg.Admins = append(cfg.Admins, newAdmin)
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}

	// 触发管理员权限转移事件
	//eventPayload, _ := json.Marshal(&ProjectConfig{
	//	ServiceCode: cfg.ServiceCode,
	//	ProjectCode: cfg.ProjectCode,
	//	Admins:      cfg.Admins, // 包含新添加的管理员
	//})
	eventPayload, _ := json.Marshal(map[string]interface{}{
		"serviceCode": cfg.ServiceCode,
		"projectCode": cfg.ProjectCode,
		"oldAdmin":    cfg.Admins,
		"newAdmin":    newAdmin})
	return common.EmitEvent(ctx, "AdminRoleTransfered", eventPayload)
}

// ================== 权限管理相关 ==================

// BatchOperateSelectorPermissions 批量授权/撤销账户权限
func (c *PermissionChaincode) BatchOperateSelectorPermissions(ctx contractapi.TransactionContextInterface, accountSelectors []AccountSelector) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx, cfg.Admins) {
		return errors.New("only admin can operate selector permissions")
	}
	if cfg.Paused {
		return errors.New("project is paused")
	}

	for _, perm := range accountSelectors {
		if strings.TrimSpace(perm.Account) == "" {
			return errors.New("account cannot be empty")
		}

		key := selectorPermPrefix + perm.Account
		var selectorMap map[string]bool
		b, err := ctx.GetStub().GetState(key)
		if err != nil || b == nil {
			selectorMap = make(map[string]bool)
		} else {
			_ = json.Unmarshal(b, &selectorMap)
		}

		// 如果FuncNames为空，则删除该账户的所有权限
		if len(perm.FuncNames) == 0 {
			// 删除该账户的所有权限
			ctx.GetStub().DelState(key)
			payload, _ := json.Marshal(map[string]interface{}{
				"serviceCode": cfg.ServiceCode,
				"projectCode": cfg.ProjectCode,
				"account":     perm.Account,
				"action":      "deleteAll",
				"isRevoked":   true})
			_ = common.EmitEvent(ctx, "SelectorPermissionOperated", payload)
		} else {
			// 更新权限
			for _, funcName := range perm.FuncNames {
				selectorMap[funcName] = true
			}
			b, _ = json.Marshal(selectorMap)
			ctx.GetStub().PutState(key, b)

			// 事件通知
			payload, _ := json.Marshal(map[string]interface{}{
				"serviceCode": cfg.ServiceCode,
				"projectCode": cfg.ProjectCode,
				"account":     perm.Account,
				"funcNames":   perm.FuncNames,
				"action":      "update",
				"isRevoked":   false})
			_ = common.EmitEvent(ctx, "SelectorPermissionOperated", payload)
		}
	}
	return nil
}

// HasSelectorPermission 查询账户是否有某函数权限
func (c *PermissionChaincode) HasSelectorPermission(ctx contractapi.TransactionContextInterface, account, funcName string) (bool, error) {
	if strings.TrimSpace(account) == "" || strings.TrimSpace(funcName) == "" {
		return false, errors.New("account and funcName cannot be empty")
	}

	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}
	if cfg.Paused {
		return false, errors.New("project is paused")
	}

	key := selectorPermPrefix + account
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return false, nil
	}

	var selectorMap map[string]bool
	_ = json.Unmarshal(b, &selectorMap)
	return selectorMap[funcName], nil
}

// GetAllSelectorsForUser 查询账户所有函数权限
func (c *PermissionChaincode) GetAllSelectorsForUser(ctx contractapi.TransactionContextInterface, account string) ([]string, error) {
	if strings.TrimSpace(account) == "" {
		return nil, errors.New("account cannot be empty")
	}

	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.Paused {
		return nil, errors.New("project is paused")
	}

	key := selectorPermPrefix + account
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return []string{}, nil
	}

	var selectorMap map[string]bool
	_ = json.Unmarshal(b, &selectorMap)
	selectors := make([]string, 0)
	for sel := range selectorMap {
		selectors = append(selectors, sel)
	}
	sort.Strings(selectors) // 排序
	return selectors, nil
}

// ================== 查询配置相关 ==================

// getProjectConfig 内部方法：获取项目配置
func (c *PermissionChaincode) getProjectConfig(ctx contractapi.TransactionContextInterface) (*common.ProjectConfig, error) {
	b, err := ctx.GetStub().GetState(projectConfigKey)
	if err != nil || b == nil {
		return nil, errors.New("project config not found")
	}
	var cfg common.ProjectConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *PermissionChaincode) IsProjectPrivate(ctx contractapi.TransactionContextInterface) (bool, error) {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}
	if cfg.Paused {
		return false, errors.New("project is paused")
	}
	return cfg.IsProjectPrivate, nil
}

func (c *PermissionChaincode) GetMethod(ctx contractapi.TransactionContextInterface) (string, error) {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return "", err
	}
	if cfg.Paused {
		return "", errors.New("project is paused")
	}
	return cfg.Method, nil
}

func (c *PermissionChaincode) IsIssuerVerificationEnabled(ctx contractapi.TransactionContextInterface) (bool, error) {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}
	if cfg.Paused {
		return false, errors.New("project is paused")
	}
	return cfg.EnableIssuerVerification, nil
}

func (c *PermissionChaincode) IsVCTemplateVerificationEnabled(ctx contractapi.TransactionContextInterface) (bool, error) {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}
	if cfg.Paused {
		return false, errors.New("project is paused")
	}
	return cfg.EnableVCTemplateVerification, nil
}

func (c *PermissionChaincode) IsWritePermissionEnabled(ctx contractapi.TransactionContextInterface) (bool, error) {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}
	if cfg.Paused {
		return false, errors.New("project is paused")
	}
	return cfg.EnableWritePermission, nil
}

// 3.1.3.22 查询项目是否停用
func (c *PermissionChaincode) Paused(ctx contractapi.TransactionContextInterface) (bool, error) {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}
	return cfg.Paused, nil
}

func (c *PermissionChaincode) IsAdminRole(ctx contractapi.TransactionContextInterface, account string) (bool, error) {
	if strings.TrimSpace(account) == "" {
		return false, errors.New("account cannot be empty")
	}

	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}
	if cfg.Paused {
		return false, errors.New("project is paused")
	}
	for _, admin := range cfg.Admins {
		if admin == account {
			return true, nil
		}
	}
	return false, nil
}

// ================== 内部校验方法 ==================

// checkMethod 校验某一个did标识符里的method名称是否和项目内一致
func (c *PermissionChaincode) checkMethod(ctx contractapi.TransactionContextInterface, did string) error {
	if strings.TrimSpace(did) == "" {
		return errors.New("did cannot be empty")
	}

	// 解析DID标识符，格式：did:method:method-specific-id
	parts := strings.Split(did, ":")
	if len(parts) != 3 || parts[0] != "did" {
		return errors.New("invalid did format, expected format: did:method:method-specific-id")
	}

	didMethod := parts[1]
	if strings.TrimSpace(didMethod) == "" {
		return errors.New("did method cannot be empty")
	}

	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}

	// 校验DID的method是否与项目配置中的Method一致
	if didMethod != cfg.Method {
		return fmt.Errorf("did method '%s' does not match project method '%s'", didMethod, cfg.Method)
	}

	return nil
}

// checkWriteFuncSelectorPermission 校验某一个写方法下某一个链账户是否拥有函数选择器权限
func (c *PermissionChaincode) checkWriteFuncSelectorPermission(ctx contractapi.TransactionContextInterface, account, funcName string) (bool, error) {
	if strings.TrimSpace(account) == "" || strings.TrimSpace(funcName) == "" {
		return false, errors.New("account and funcName cannot be empty")
	}

	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}
	if cfg.Paused {
		return false, errors.New("project is paused")
	}

	// 如果是超级管理员直接返回
	isAdmin, err := c.IsAdminRole(ctx, account)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}

	// 校验项目是否开启写权限
	if !cfg.EnableWritePermission {
		return false, errors.New("write permission is not enabled")
	}

	// 校验账户是否有该函数权限
	return c.HasSelectorPermission(ctx, account, funcName)
}

// checkQueryFuncSelectorPermission 校验某一个查方法下某一个链账户是否拥有函数选择器权限
func (c *PermissionChaincode) checkQueryFuncSelectorPermission(ctx contractapi.TransactionContextInterface, account, funcName string) (bool, error) {
	if strings.TrimSpace(account) == "" || strings.TrimSpace(funcName) == "" {
		return false, errors.New("account and funcName cannot be empty")
	}

	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}
	if cfg.Paused {
		return false, errors.New("project is paused")
	}

	// 如果是超级管理员直接返回
	isAdmin, err := c.IsAdminRole(ctx, account)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}

	// 如果项目是公开项目直接返回,公开项目都可以查询
	if !cfg.IsProjectPrivate {
		return true, nil
	}

	// 如果项目是私有项目，则需要校验权限
	return c.HasSelectorPermission(ctx, account, funcName)
}

// checkIssuerVerificationEnabled 检查账户是否可以进行颁发者（注册/更新等操作）
func (c *PermissionChaincode) checkIssuerVerificationEnabled(ctx contractapi.TransactionContextInterface, account string) (bool, error) {
	//若开启审核，则调用方只有管理员账户才可以进行进行注册发证方系列操作
	if strings.TrimSpace(account) == "" {
		return false, errors.New("account cannot be empty")
	}

	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}

	// 如果未启发证方审核，直接返回成功
	if !cfg.EnableIssuerVerification {
		return true, nil
	}
	//c.checkAdminRole(ctx, account)
	// 如果开启发证方审核并且调用者不是超级管理员则抛出异常
	isAdmin, err := c.IsAdminRole(ctx, account)
	if err != nil {
		return false, err
	}
	if !isAdmin {
		return false, errors.New("issuer verification is enabled, only admin can access")
	}

	return true, nil
}

// checkVCTemplateVerificationEnabled 校验某一个链账户是否在开启VC模版审核状态权限下具有访问权限
func (c *PermissionChaincode) checkVCTemplateVerificationEnabled(ctx contractapi.TransactionContextInterface, account string) (bool, error) {
	if strings.TrimSpace(account) == "" {
		return false, errors.New("account cannot be empty")
	}

	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}

	// 如果未启VC模版审核，直接返回成功
	if !cfg.EnableVCTemplateVerification {
		return true, nil
	}

	// 如果开启VC模版审核并且调用者不是超级管理员则抛出异常
	isAdmin, err := c.IsAdminRole(ctx, account)
	if err != nil {
		return false, err
	}
	if !isAdmin {
		return false, errors.New("VC template verification is enabled, only admin can access")
	}

	return true, nil
}

// checkNotPaused 校验项目是否是启动状态
func (c *PermissionChaincode) checkNotPaused(ctx contractapi.TransactionContextInterface) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if cfg.Paused {
		return errors.New("project is paused")
	}
	return nil
}

// checkAdminRole 校验某一个链账户是否是超级管理员
func (c *PermissionChaincode) checkAdminRole(ctx contractapi.TransactionContextInterface, account string) error {
	if strings.TrimSpace(account) == "" {
		return errors.New("account cannot be empty")
	}

	isAdmin, err := c.IsAdminRole(ctx, account)
	if err != nil {
		return err
	}
	if !isAdmin {
		return errors.New("account is not admin")
	}
	return nil
}

// 权限校验工具
func CheckPermission(ctx contractapi.TransactionContextInterface, account, function string) bool {
	// TODO: 实现权限校验逻辑
	acl := new(PermissionChaincode)
	funcB, _ := acl.HasSelectorPermission(ctx, account, function)
	if funcB != true {
		return funcB
	}
	return false
}

// ================== 权限检查接口实现 ==================

// CheckWriteFuncSelectorPermission 实现PermissionChecker接口的写权限检查方法
func (c *PermissionChaincode) CheckWriteFuncSelectorPermission(ctx contractapi.TransactionContextInterface, account, funcName string) (bool, error) {
	return c.checkWriteFuncSelectorPermission(ctx, account, funcName)
}

// CheckQueryFuncSelectorPermission 实现PermissionChecker接口的查询权限检查方法
func (c *PermissionChaincode) CheckQueryFuncSelectorPermission(ctx contractapi.TransactionContextInterface, account, funcName string) (bool, error) {
	return c.checkQueryFuncSelectorPermission(ctx, account, funcName)
}

// CheckMethod 实现PermissionChecker接口的DID方法检查方法
func (c *PermissionChaincode) CheckMethod(ctx contractapi.TransactionContextInterface, did string) error {
	return c.checkMethod(ctx, did)
}

// CheckIssuerVerificationEnabled 实现PermissionChecker接口的发证方验证检查方法
func (c *PermissionChaincode) CheckIssuerVerificationEnabled(ctx contractapi.TransactionContextInterface, account string) (bool, error) {
	return c.checkIssuerVerificationEnabled(ctx, account)
}

// CheckVCTemplateVerificationEnabled 实现PermissionChecker接口的VC模板验证检查方法
func (c *PermissionChaincode) CheckVCTemplateVerificationEnabled(ctx contractapi.TransactionContextInterface, account string) (bool, error) {
	return c.checkVCTemplateVerificationEnabled(ctx, account)
}

// CheckNotPaused 实现PermissionChecker接口的项目状态检查方法
func (c *PermissionChaincode) CheckNotPaused(ctx contractapi.TransactionContextInterface) error {
	return c.checkNotPaused(ctx)
}

// CheckAdminRole 实现PermissionChecker接口的管理员角色检查方法
func (c *PermissionChaincode) CheckAdminRole(ctx contractapi.TransactionContextInterface, account string) error {
	return c.checkAdminRole(ctx, account)
}

// GetProjectConfig 公共方法：获取项目配置，供其他模块调用
func (c *PermissionChaincode) GetProjectConfig(ctx contractapi.TransactionContextInterface) (*common.ProjectConfig, error) {
	return c.getProjectConfig(ctx)
}
