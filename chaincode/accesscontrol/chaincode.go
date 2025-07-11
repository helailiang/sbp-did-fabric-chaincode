package accesscontrol

import (
	"encoding/json"
	"errors"
	"strings"

	"sbp-did-chaincode/chaincode/common"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// 项目配置结构体
type ProjectConfig struct {
	IsPrivate                    bool     // 项目是否私有
	EnableVCTemplateVerification bool     // 是否启用VC模板验证
	EnableIssuerVerification     bool     // 是否启用Issuer验证
	Method                       string   // 项目method名称
	Paused                       bool     // 项目是否停用
	IsProjectPrivate             bool     // 项目是否私有
	Admins                       []string // Admins的账户地址
}

// 账户权限结构体
type SelectorPermission struct {
	Account   string   // 账户地址
	Selectors []string // 允许的函数名
}

// AccessControlChaincode 结构体
type AccessControlChaincode struct {
	contractapi.Contract
}

const (
	projectConfigKey   = "accesscontrol:projectConfig"
	selectorPermPrefix = "accesscontrol:selectorPerm:"
)

// ================== 项目配置相关 ==================

// InitProject 初始化项目配置
func (c *AccessControlChaincode) InitProject(ctx contractapi.TransactionContextInterface, method string, isPrivate, enableVC, enableIssuer bool) error {
	if !common.IsAdmin(ctx) {
		return errors.New("only admin can init project")
	}
	if strings.TrimSpace(method) == "" {
		return errors.New("method cannot be empty")
	}
	cfg := ProjectConfig{
		IsPrivate:                    isPrivate,
		EnableVCTemplateVerification: enableVC,
		EnableIssuerVerification:     enableIssuer,
		Method:                       method,
		Paused:                       false,
	}
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "ProjectInitialized", b)
}

// ChangePrivateStatus 更改项目私有/公开
func (c *AccessControlChaincode) ChangePrivateStatus(ctx contractapi.TransactionContextInterface, isPrivate bool) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx) {
		return errors.New("only admin can change private status")
	}
	cfg.IsPrivate = isPrivate
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "PrivateStatusChanged", b)
}

// ChangeMethod 更改method名称
func (c *AccessControlChaincode) ChangeMethod(ctx contractapi.TransactionContextInterface, method string) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx) {
		return errors.New("only admin can change method")
	}
	if strings.TrimSpace(method) == "" {
		return errors.New("method cannot be empty")
	}
	cfg.Method = method
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "MethodChanged", b)
}

// ChangeEnableVCTemplateVerification 更改VC模板验证开关
func (c *AccessControlChaincode) ChangeEnableVCTemplateVerification(ctx contractapi.TransactionContextInterface, enable bool) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx) {
		return errors.New("only admin can change VC template verification")
	}
	cfg.EnableVCTemplateVerification = enable
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "EnableVCTemplateVerificationChanged", b)
}

// ChangeEnableIssuerVerification 更改Issuer验证开关
func (c *AccessControlChaincode) ChangeEnableIssuerVerification(ctx contractapi.TransactionContextInterface, enable bool) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx) {
		return errors.New("only admin can change issuer verification")
	}
	cfg.EnableIssuerVerification = enable
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "EnableIssuerVerificationChanged", b)
}

// Pause 项目停用
func (c *AccessControlChaincode) Pause(ctx contractapi.TransactionContextInterface) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx) {
		return errors.New("only admin can pause project")
	}
	cfg.Paused = true
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "Paused", b)
}

// Unpause 项目启用
func (c *AccessControlChaincode) Unpause(ctx contractapi.TransactionContextInterface) error {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return err
	}
	if !common.IsAdmin(ctx) {
		return errors.New("only admin can unpause project")
	}
	cfg.Paused = false
	b, _ := json.Marshal(cfg)
	if err := ctx.GetStub().PutState(projectConfigKey, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "Unpaused", b)
}

// ================== 权限管理相关 ==================

// BatchOperateSelectorPermissions 批量授权/撤销账户权限
func (c *AccessControlChaincode) BatchOperateSelectorPermissions(ctx contractapi.TransactionContextInterface, perms []SelectorPermission, isRevoke bool) error {
	if !common.IsAdmin(ctx) {
		return errors.New("only admin can operate selector permissions")
	}
	for _, perm := range perms {
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
		for _, sel := range perm.Selectors {
			if isRevoke {
				delete(selectorMap, sel)
			} else {
				selectorMap[sel] = true
			}
			// 事件通知
			payload, _ := json.Marshal(map[string]interface{}{"account": perm.Account, "selector": sel, "revoked": isRevoke})
			_ = common.EmitEvent(ctx, "SelectorPermissionOperated", payload)
		}
		b, _ = json.Marshal(selectorMap)
		ctx.GetStub().PutState(key, b)
	}
	return nil
}

// HasSelectorPermission 查询账户是否有某函数权限
func (c *AccessControlChaincode) HasSelectorPermission(ctx contractapi.TransactionContextInterface, account, selector string) (bool, error) {
	if strings.TrimSpace(account) == "" || strings.TrimSpace(selector) == "" {
		return false, errors.New("account and selector cannot be empty")
	}
	key := selectorPermPrefix + account
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return false, nil
	}
	var selectorMap map[string]bool
	_ = json.Unmarshal(b, &selectorMap)
	return selectorMap[selector], nil
}

// GetAllSelectorsForUser 查询账户所有函数权限
func (c *AccessControlChaincode) GetAllSelectorsForUser(ctx contractapi.TransactionContextInterface, account string) ([]string, error) {
	if strings.TrimSpace(account) == "" {
		return nil, errors.New("account cannot be empty")
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
	return selectors, nil
}

// ================== 查询配置相关 ==================

func (c *AccessControlChaincode) getProjectConfig(ctx contractapi.TransactionContextInterface) (*ProjectConfig, error) {
	b, err := ctx.GetStub().GetState(projectConfigKey)
	if err != nil || b == nil {
		return nil, errors.New("project config not found")
	}
	var cfg ProjectConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *AccessControlChaincode) IsProjectPrivate(ctx contractapi.TransactionContextInterface) (bool, error) {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}
	return cfg.IsPrivate, nil
}

func (c *AccessControlChaincode) IsIssuerVerificationEnabled(ctx contractapi.TransactionContextInterface) (bool, error) {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}
	return cfg.EnableIssuerVerification, nil
}

func (c *AccessControlChaincode) IsVCTemplateVerificationEnabled(ctx contractapi.TransactionContextInterface) (bool, error) {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}
	return cfg.EnableVCTemplateVerification, nil
}

func (c *AccessControlChaincode) Paused(ctx contractapi.TransactionContextInterface) (bool, error) {
	cfg, err := c.getProjectConfig(ctx)
	if err != nil {
		return false, err
	}
	return cfg.Paused, nil
}
