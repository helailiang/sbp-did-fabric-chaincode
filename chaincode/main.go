package main

import (
	"fmt"
	"sbp-did-chaincode/accesscontrol"
	"sbp-did-chaincode/common"
	"sbp-did-chaincode/did"
	"sbp-did-chaincode/issuer"
	"sbp-did-chaincode/vc"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
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
	if err != nil {
		panic(fmt.Errorf("Error create SBP-DID Chaincode: %s", err))
		return
	}
	chaincode.DefaultContract = didChaincode.GetName()
	if err := chaincode.Start(); err != nil {
		panic(fmt.Errorf("Error starting SBP-DID Chaincode: %s", err))
	}
}
