package did

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"sbp-did-chaincode/chaincode/common"
)

// DID信息结构体
type DidInfo struct {
	DidDocument string // DID文档
	Account     string // 注册账户
}

// DIDChaincode 结构体
type DIDChaincode struct {
	contractapi.Contract
}

const didInfoPrefix = "did:info:"

// RegisterDid 注册DID

const (
	defaultSelector       = "RegisterDid"
	defaultSelectorUpdate = "UpdateDidDocument"
)

func (c *DIDChaincode) RegisterDid(ctx contractapi.TransactionContextInterface, did, didDocument string) error {
	if strings.TrimSpace(did) == "" || strings.TrimSpace(didDocument) == "" {
		return errors.New("did and didDocument cannot be empty")
	}
	key := didInfoPrefix + did
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		return err
	}
	if b != nil {
		return errors.New("did already exists")
	}
	// 权限校验（可根据业务调整）
	if !common.CheckPermission(ctx, common.GetCaller(ctx), defaultSelector) {
		return errors.New("no permission to register did")
	}
	info := DidInfo{
		DidDocument: didDocument,
		Account:     common.GetCaller(ctx),
	}
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "DidRegistered", b)
}

// UpdateDidDocument 更新DID文档
func (c *DIDChaincode) UpdateDidDocument(ctx contractapi.TransactionContextInterface, did, didDocument string) error {
	if strings.TrimSpace(did) == "" || strings.TrimSpace(didDocument) == "" {
		return errors.New("did and didDocument cannot be empty")
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
	if !common.CheckPermission(ctx, common.GetCaller(ctx), defaultSelectorUpdate) {
		return errors.New("no permission to update did")
	}
	info.DidDocument = didDocument
	b, _ = json.Marshal(info)
	if err := ctx.GetStub().PutState(key, b); err != nil {
		return err
	}
	return common.EmitEvent(ctx, "DidDocumentUpdated", b)
}

// GetDidInfo 查询DID文档
func (c *DIDChaincode) GetDidInfo(ctx contractapi.TransactionContextInterface, did string) (string, error) {
	if strings.TrimSpace(did) == "" {
		return "", errors.New("did cannot be empty")
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
