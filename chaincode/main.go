package main

import (
	"fmt"
	"sbp-did-chaincode/chaincode/accesscontrol"
	"sbp-did-chaincode/chaincode/did"
	"sbp-did-chaincode/chaincode/issuer"
	"sbp-did-chaincode/chaincode/vc"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	chaincode, err := contractapi.NewChaincode(
		new(accesscontrol.AccessControlChaincode),
		new(did.DIDChaincode),
		new(issuer.IssuerChaincode),
		new(vc.VCChaincode),
	)
	if err != nil {
		fmt.Printf("Error create SBP-DID Chaincode: %s", err)
		return
	}
	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting SBP-DID Chaincode: %s", err)
	}
}
