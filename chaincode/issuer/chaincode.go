package issuer

import (
	"encoding/json"
	"errors"
	"strings"

	"sbp-did-chaincode/chaincode/common"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// 发证方信息结构体
type IssuerInfo struct {
	IssuerDid   string             `json:"issuerDid"`   // 发证方ID
	Name        string             `json:"name"`        // 发证方名称
	IsDisabled  bool               `json:"isDisabled"`  // 是否禁用
	Account     string             `json:"account"`     // 记录链账户信息用于更新
	MataDate    IssuerInfoMataDate `json:"mataDate"`    // 发证方信息
	VcTemplates map[string]bool    `json:"vcTemplates"` // 发证方模板
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
	Id             string                 `json:"id"`             //VC模板的ID
	IssuerDid      string                 `json:"issuerDid"`      // 发证方ID
	VcTemplateData string                 `json:"vcTemplateData"` // vc模板序列化数据
	Account        string                 `json:"account"`        // 记录链账户信息用于更新
	IsDisabled     bool                   `json:"isDisabled"`     // 是否禁用
	MataDate       VcTemplateInfoMataDate `json:"mataDate"`       // 模板信息
}

type VcTemplateInfoMataDate struct {
	Endpoint    string `json:"endpoint"`    // 请求端点
	Version     string `json:"version"`     // 模板版本
	Description string `json:"description"` // 业务描述
}

// IssuerChaincode 结构体
type IssuerChaincode struct {
	contractapi.Contract
}

const (
	issuerInfoPrefix     = "issuer:info:"
	vcTemplateInfoPrefix = "issuer:vctemplate:"
)

// RegisterIssuer 注册发证方
func (c *IssuerChaincode) RegisterIssuer(ctx contractapi.TransactionContextInterface, issuerDid, name string) error {
	if strings.TrimSpace(issuerDid) == "" || strings.TrimSpace(name) == "" {
		return errors.New("issuerDid and name cannot be empty")
	}
	key := issuerInfoPrefix + issuerDid
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		return err
	}
	if b != nil {
		return errors.New("issuer already exists")
	}
	if !common.CheckPermission(ctx, common.GetCaller(ctx), "RegisterIssuer") {
		return errors.New("no permission to register issuer")
	}
	info := IssuerInfo{
		Name:       name,
		IsDisabled: false,
		Account:    common.GetCaller(ctx),
	}
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "IssuerRegistered", b)
}

// UpdateIssuer 更新发证方
func (c *IssuerChaincode) UpdateIssuer(ctx contractapi.TransactionContextInterface, issuerDid, name string) error {
	if strings.TrimSpace(issuerDid) == "" || strings.TrimSpace(name) == "" {
		return errors.New("issuerDid and name cannot be empty")
	}
	key := issuerInfoPrefix + issuerDid
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return errors.New("issuer not found")
	}
	var info IssuerInfo
	_ = json.Unmarshal(b, &info)
	if info.Account != common.GetCaller(ctx) {
		return errors.New("only creator can update issuer")
	}
	if !common.CheckPermission(ctx, common.GetCaller(ctx), "UpdateIssuer") {
		return errors.New("no permission to update issuer")
	}
	info.Name = name
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "IssuerUpdated", b)
}

// ChangeIssuerStatus 启停发证方
func (c *IssuerChaincode) ChangeIssuerStatus(ctx contractapi.TransactionContextInterface, issuerDid string, isDisabled bool) error {
	if strings.TrimSpace(issuerDid) == "" {
		return errors.New("issuerDid cannot be empty")
	}
	key := issuerInfoPrefix + issuerDid
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return errors.New("issuer not found")
	}
	var info IssuerInfo
	_ = json.Unmarshal(b, &info)
	if info.Account != common.GetCaller(ctx) {
		return errors.New("only creator can change status")
	}
	if !common.CheckPermission(ctx, common.GetCaller(ctx), "ChangeIssuerStatus") {
		return errors.New("no permission to change issuer status")
	}
	info.IsDisabled = isDisabled
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "IssuerStatusChanged", b)
}

// GetIssuerInfo 查询发证方信息
func (c *IssuerChaincode) GetIssuerInfo(ctx contractapi.TransactionContextInterface, issuerDid string) (string, bool, error) {
	if strings.TrimSpace(issuerDid) == "" {
		return "", false, errors.New("issuerDid cannot be empty")
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

// CheckIssuer 校验发证方是否存在
func (c *IssuerChaincode) CheckIssuer(ctx contractapi.TransactionContextInterface, issuerDid string) (bool, error) {
	if strings.TrimSpace(issuerDid) == "" {
		return false, errors.New("issuerDid cannot be empty")
	}
	key := issuerInfoPrefix + issuerDid
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return false, nil
	}
	return true, nil
}

// RegisterVCTemplate 注册VC模板
func (c *IssuerChaincode) RegisterVCTemplate(ctx contractapi.TransactionContextInterface, vcTemplateId, vcTemplateData, vcTemplateInfoMataDate string) error {
	if strings.TrimSpace(vcTemplateId) == "" || strings.TrimSpace(vcTemplateData) == "" {
		return errors.New("vcTemplateId and vcTemplateData cannot be empty")
	}
	key := vcTemplateInfoPrefix + vcTemplateId
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		return err
	}
	if b != nil {
		return errors.New("vc template already exists")
	}
	if !common.CheckPermission(ctx, common.GetCaller(ctx), "RegisterVCTemplate") {
		return errors.New("no permission to register vc template")
	}
	tpl := VcTemplateInfo{
		VcTemplateData: vcTemplateData,
		Account:        common.GetCaller(ctx),
		IsDisabled:     false,
	}
	b, _ = json.Marshal(tpl)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "VCTemplateRegistered", b)
}

// UpdateVCTemplate 更新VC模板
func (c *IssuerChaincode) UpdateVCTemplate(ctx contractapi.TransactionContextInterface, vcTemplateId, vcTemplateData, vcTemplateInfoMataDate string) error {
	if strings.TrimSpace(vcTemplateId) == "" || strings.TrimSpace(vcTemplateData) == "" {
		return errors.New("vcTemplateId and vcTemplateData cannot be empty")
	}
	key := vcTemplateInfoPrefix + vcTemplateId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return errors.New("vc template not found")
	}
	var tpl VcTemplateInfo
	_ = json.Unmarshal(b, &tpl)
	if tpl.Account != common.GetCaller(ctx) {
		return errors.New("only creator can update vc template")
	}
	if !common.CheckPermission(ctx, common.GetCaller(ctx), "UpdateVCTemplate") {
		return errors.New("no permission to update vc template")
	}
	tpl.VcTemplateData = vcTemplateData
	b, _ = json.Marshal(tpl)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "VCTemplateUpdated", b)
}

// ChangeVCTemplateStatus 启停VC模板
func (c *IssuerChaincode) ChangeVCTemplateStatus(ctx contractapi.TransactionContextInterface, vcTemplateId string, isDisabled bool) error {
	if strings.TrimSpace(vcTemplateId) == "" {
		return errors.New("vcTemplateId cannot be empty")
	}
	key := vcTemplateInfoPrefix + vcTemplateId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return errors.New("vc template not found")
	}
	var tpl VcTemplateInfo
	_ = json.Unmarshal(b, &tpl)
	if tpl.Account != common.GetCaller(ctx) {
		return errors.New("only creator can change status")
	}
	if !common.CheckPermission(ctx, common.GetCaller(ctx), "ChangeVCTemplateStatus") {
		return errors.New("no permission to change vc template status")
	}
	tpl.IsDisabled = isDisabled
	b, _ = json.Marshal(tpl)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "VCTemplateStatusChanged", b)
}

// GetVCTemplateInfo 查询VC模板信息
func (c *IssuerChaincode) GetVCTemplateInfo(ctx contractapi.TransactionContextInterface, vcTemplateId string) (string, bool, error) {
	if strings.TrimSpace(vcTemplateId) == "" {
		return "", false, errors.New("vcTemplateId cannot be empty")
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
