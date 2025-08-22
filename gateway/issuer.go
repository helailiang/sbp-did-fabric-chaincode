package main

import (
	"fmt"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"google.golang.org/grpc/status"
)

func registerIssuer(contract *client.Contract) uint64 {
	fmt.Printf("\n--> Submit transaction: registerIssuer, %s \n", DIDID)
	_, commit, err := contract.SubmitAsync("RegisterIssuer", client.WithArguments(DIDID, "bsn-first-issuer"))
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			for _, detail := range st.Details() {
				fmt.Printf("endorser detail: %+v\n", detail)
			}
		}
		panic(fmt.Errorf("failed to submit transaction: %s \n", err))
	}

	status, err := commit.Status()
	if err != nil {
		panic(fmt.Errorf("failed to get transaction commit status: %w \n", err))
	}

	if !status.Successful {
		panic(fmt.Errorf("failed to commit transaction with status code %v", status.Code))
	}

	fmt.Println("\n*** RegisterIssuer committed successfully")

	return status.BlockNumber
}

func updateIssuer(contract *client.Contract) {
	fmt.Printf("\n--> Submit transaction: UpdateIssuer, %s update appraised value to 200\n", DIDID)
	_, err := contract.SubmitTransaction("UpdateIssuer", DIDID, "bsn-first-issuer-updated")
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Println("\n*** UpdateIssuerDocument committed successfully")
}

func GetIssuerInfo(contract *client.Contract) {
	fmt.Printf("\n--> Submit transaction: GetIssuerInfo, %s to Issuer\n", DIDID)

	evaluateResult, err := contract.EvaluateTransaction("GetIssuerInfo", DIDID)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
	fmt.Println("\n*** GetIssuerInfo committed successfully")
}
