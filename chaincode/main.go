package main

import (
	"fmt"
	"sbp-did-chaincode/chaincode/accesscontrol"
	"sbp-did-chaincode/chaincode/common"
	"sbp-did-chaincode/chaincode/did"
	"sbp-did-chaincode/chaincode/issuer"
	"sbp-did-chaincode/chaincode/vc"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

func main() {
	// 创建权限控制合约实例
	permissionChaincode := new(accesscontrol.PermissionChaincode)
	permissionChaincode.Name = "permission"
	// 设置全局权限检查器
	common.SetGlobalPermissionChecker(permissionChaincode)

	didChaincode := new(did.DIDChaincode)
	didChaincode.Name = "did"

	issuerChaincode := new(issuer.IssuerChaincode)
	issuerChaincode.PermissionChecker = permissionChaincode
	issuerChaincode.Name = "issuer"

	vcChaincode := new(vc.VCChaincode)
	vcChaincode.Name = "vc"
	chaincode, err := contractapi.NewChaincode(
		permissionChaincode,
		didChaincode,
		issuerChaincode,
		vcChaincode,
	)
	chaincode.DefaultContract = didChaincode.GetName()
	if err != nil {
		panic(fmt.Errorf("Error create SBP-DID Chaincode: %s", err))
		return
	}
	if err := chaincode.Start(); err != nil {
		panic(fmt.Errorf("Error starting SBP-DID Chaincode: %s", err))
	}
}
