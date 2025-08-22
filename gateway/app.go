/*
Copyright 2022 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
)

const (
	channelName         = "hll8"
	chaincodeName       = "did"
	issuerChaincodeName = "issuer"
)

var now = time.Now()
var DIDID = fmt.Sprintf("did:bsn:%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)

func main() {
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	gateway, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(clientConnection),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gateway.Close()

	network := gateway.GetNetwork(channelName)
	//contract := network.GetContract(chaincodeName)

	// Context used for event listening
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for events emitted by subsequent transactions
	startChaincodeEventListening(ctx, network)

	//firstBlockNumber := DID(ctx, network)
	firstBlockNumber := Issuer(ctx, network)
	replayChaincodeEvents(ctx, network, firstBlockNumber)
}

func startChaincodeEventListening(ctx context.Context, network *client.Network) {
	fmt.Println("\n*** Start chaincode event listening")

	events, err := network.ChaincodeEvents(ctx, chaincodeName)
	if err != nil {
		panic(fmt.Errorf("failed to start chaincode event listening: %w", err))
	}

	go func() {
		for event := range events {
			Payload := formatJSON(event.Payload)
			fmt.Printf("\n<-- Chaincode event received: [%s] ----- %s \n", event.EventName, Payload)
		}
	}()
}

func formatJSON(data []byte) string {
	var result bytes.Buffer
	if err := json.Indent(&result, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return result.String()
}

func DID(ctx context.Context, network *client.Network) uint64 {
	contract := network.GetContract(chaincodeName)

	firstBlockNumber := createDID(contract)
	//updateDID(contract)
	GetDidInfo(contract)
	// Replay events from the block containing the first transaction
	//replayChaincodeEvents(ctx, network, firstBlockNumber)
	return firstBlockNumber
}

func Issuer(ctx context.Context, network *client.Network) uint64 {
	contract := network.GetContractWithName(chaincodeName, issuerChaincodeName)
	firstBlockNumber := registerIssuer(contract)
	updateIssuer(contract)
	GetIssuerInfo(contract)
	// Replay events from the block containing the first transaction
	//replayChaincodeEvents(ctx, network, firstBlockNumber)
	return firstBlockNumber
}

func replayChaincodeEvents(ctx context.Context, network *client.Network, startBlock uint64) {
	fmt.Println("\n*** Start chaincode event replay")

	events, err := network.ChaincodeEvents(ctx, chaincodeName, client.WithStartBlock(startBlock))
	if err != nil {
		panic(fmt.Errorf("failed to start chaincode event listening: %w", err))
	}

	for {
		select {
		case <-time.After(100 * time.Second):
			panic(errors.New("timeout waiting for event replay"))

		case event := <-events:
			Payload := formatJSON(event.Payload)
			fmt.Printf("\n<-- Chaincode event replayed: [%s] - %s \n", event.EventName, Payload)
		}
	}
}
