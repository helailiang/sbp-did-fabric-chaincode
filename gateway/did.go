package main

import (
	"fmt"

	"github.com/hyperledger/fabric-gateway/pkg/client"
)

func createDID(contract *client.Contract) uint64 {
	fmt.Printf("\n--> Submit transaction: CreateDID, %s \n", DIDID)
	didDocument := fmt.Sprintf(`{
    "@context": "https://www.w3.org/ns/did/v1.1",
    "id": "%s",
    "verificationMethod": [
        {
            "id": "%s#key-0",
            "type": "JsonWebKey",
            "controller": "%s",
            "publicKeyJwk": {
                "kty": "OKP",
                "crv": "Ed25519",
                "x": "VCpo2LMLhn6iWku8MKvSLg2ZAoC-nlOyPVQaO3FxVeQ"
            }
        }
    ]
}`, DIDID, DIDID, DIDID)
	_, commit, err := contract.SubmitAsync("RegisterDid", client.WithArguments(DIDID, didDocument))
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	status, err := commit.Status()
	if err != nil {
		panic(fmt.Errorf("failed to get transaction commit status: %w", err))
	}

	if !status.Successful {
		panic(fmt.Errorf("failed to commit transaction with status code %v", status.Code))
	}

	fmt.Println("\n*** RegisterDid committed successfully")

	return status.BlockNumber
}

func updateDID(contract *client.Contract) {
	fmt.Printf("\n--> Submit transaction: UpdateDidDocument, %s update appraised value to 200\n", DIDID)
	didDocument := fmt.Sprintf(`{
    "@context": "https://www.w3.org/ns/did/v1.1",
    "id": "%s",
    "verificationMethod": [
        {
            "id": "%s#key-0",
            "type": "JsonWebKey",
            "controller": "%s",
			"update": true,
            "publicKeyJwk": {
                "kty": "OKP",
                "crv": "Ed25519",
                "x": "VCpo2LMLhn6iWku8MKvSLg2ZAoC-nlOyPVQaO3FxVeQ"
            }
        }
    ]
}`, DIDID, DIDID, DIDID)
	_, err := contract.SubmitTransaction("UpdateDidDocument", DIDID, didDocument)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Println("\n*** UpdateDidDocument committed successfully")
}

func GetDidInfo(contract *client.Contract) {
	fmt.Printf("\n--> Submit transaction: GetDidInfo, %s to did\n", DIDID)

	evaluateResult, err := contract.EvaluateTransaction("GetDidInfo", DIDID)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
	fmt.Println("\n*** GetDidInfo committed successfully")
}
