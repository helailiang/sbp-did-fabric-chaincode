package common

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"sbp-did-chaincode/chaincode/accesscontrol"
)

// 校验是否为管理员
func IsAdmin(ctx contractapi.TransactionContextInterface) bool {
	// TODO: 实现管理员校验逻辑
	return false
}

// 权限校验工具
func CheckPermission(ctx contractapi.TransactionContextInterface, account, function string) bool {
	// TODO: 实现权限校验逻辑
	acl := new(accesscontrol.AccessControlChaincode)
	funcB, _ := acl.HasSelectorPermission(ctx, account, function)
	if funcB != true {
		return funcB
	}
	return false
}

// 事件封装工具
func EmitEvent(ctx contractapi.TransactionContextInterface, eventName string, payload []byte) error {
	return ctx.GetStub().SetEvent(eventName, payload)
}

// GetCaller 获取当前调用者标识（如MSPID或证书ID）
func GetCaller(ctx contractapi.TransactionContextInterface) string {
	// TODO: 实现获取调用者身份标识
	ski, err := GetMsgSenderSKI(ctx.GetStub())
	if err != nil {
		panic(err)
	}
	return ski
}

func GetMsgSenderSKI(stub shim.ChaincodeStubInterface) (string, error) {
	cert, err := cid.GetX509Certificate(stub)
	if err != nil {
		return "", fmt.Errorf("failed to parse CA: %v", err)
	}
	return hex.EncodeToString(cert.SubjectKeyId), nil
}

func GetMsgSenderAddress(stub shim.ChaincodeStubInterface) (common.Address, error) {
	cert, err := cid.GetX509Certificate(stub)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to parse CA: %v", err)
	}
	return GetAddrFromRaw(cert.RawSubjectPublicKeyInfo), nil
}

func GetAddrFromRaw(raw []byte) common.Address {
	hash := sha256.New()
	hash.Write(raw)
	addr := common.BytesToAddress(hash.Sum(nil)[12:])
	return addr
}
