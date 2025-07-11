package vc

import (
	"encoding/json"
	"errors"
	"strings"

	"sbp-did-chaincode/chaincode/common"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// VC存证信息结构体
type VCInfo struct {
	VcId       string `json:"vcId"`                 // vc唯一码
	SubjectDid string `json:"subjectDid,omitempty"` // 持有者DID
	IssuerDid  string `json:"issuerDid"`            // 发证方DID
	VcHash     string `json:"vcHash"`               // VC内容 hash值
	IsRevoked  bool   `json:"isRevoked"`            // 是否已吊销 true: 已吊销
}

// VCChaincode 结构体
type VCChaincode struct {
	contractapi.Contract
}

const vcInfoPrefix = "vc:info:"

// CreateVC 创建VC存证
func (c *VCChaincode) CreateVC(ctx contractapi.TransactionContextInterface, vcId, vcHash, issuerDid string) error {
	if strings.TrimSpace(vcId) == "" || strings.TrimSpace(vcHash) == "" || strings.TrimSpace(issuerDid) == "" {
		return errors.New("vcId, vcHash, issuerDid cannot be empty")
	}
	key := vcInfoPrefix + vcId
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		return err
	}
	if b != nil {
		return errors.New("vc already exists")
	}
	if !common.CheckPermission(ctx, common.GetCaller(ctx), "CreateVC") {
		return errors.New("no permission to create vc")
	}
	info := VCInfo{
		SubjectDid: vcId,
		IssuerDid:  issuerDid,
		VcHash:     vcHash,
		IsRevoked:  false,
	}
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "VCCreated", b)
}

// GetVCHash 查询VC哈希
func (c *VCChaincode) GetVCHash(ctx contractapi.TransactionContextInterface, vcId string) (string, error) {
	if strings.TrimSpace(vcId) == "" {
		return "", errors.New("vcId cannot be empty")
	}
	key := vcInfoPrefix + vcId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return "", errors.New("vc not found")
	}
	var info VCInfo
	_ = json.Unmarshal(b, &info)
	return info.VcHash, nil
}

// RevokeVC 吊销VC
func (c *VCChaincode) RevokeVC(ctx contractapi.TransactionContextInterface, vcId string, isRevoked bool) error {
	if strings.TrimSpace(vcId) == "" {
		return errors.New("vcId cannot be empty")
	}
	key := vcInfoPrefix + vcId
	b, err := ctx.GetStub().GetState(key)
	if err != nil || b == nil {
		return errors.New("vc not found")
	}
	var info VCInfo
	_ = json.Unmarshal(b, &info)
	if !common.CheckPermission(ctx, common.GetCaller(ctx), "RevokeVC") {
		return errors.New("no permission to revoke vc")
	}
	info.IsRevoked = isRevoked
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "VCRevoked", b)
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
