package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/bugnanetwork/bugnad/domain/bvm/abi"
	"github.com/bugnanetwork/bugnad/infrastructure/network/rpcclient"
)

func queryContract(cfg *queryContractConfig) error {
	rpcAddress, err := cfg.NetParams().NormalizeRPCServerAddress(cfg.RPCServer)
	if err != nil {
		return fmt.Errorf("Error parsing RPC server address: %s", err)
	}

	log.Infof("Connecting to RPC server at %s\n", rpcAddress)

	//RPC client activation (to communicate with Kaspad)
	client, err := rpcclient.NewRPCClient(rpcAddress)
	if err != nil {
		return fmt.Errorf("Error connecting to the RPC server: %s", err)
	}

	client.SetTimeout(5 * time.Minute)

	// parse ABI
	abiData, err := os.ReadFile(cfg.AbiFile)
	if err != nil {
		return fmt.Errorf("Error reading ABI file: %s", err)
	}

	ABI, err := abi.JSON(bytes.NewBuffer(abiData))
	if err != nil {
		return fmt.Errorf("Error parsing ABI: %s", err)
	}

	method, ok := ABI.Methods[cfg.Method]
	if !ok {
		return fmt.Errorf("Method `%s` not found in ABI", cfg.Method)
	}

	if len(cfg.Args) != len(method.Inputs) {
		return fmt.Errorf("Method `%s` expects %d arguments, got %d", cfg.Method, len(method.Inputs), len(cfg.Args))
	}

	inputData, err := getMethodPayload(ABI, append([]string{cfg.Method}, cfg.Args...))
	if err != nil {
		return fmt.Errorf("Error getting method payload: %s", err)
	}

	resp, err := client.GetBvmSmartContractData(cfg.ContractAddress, fmt.Sprintf("%x", inputData))
	if err != nil {
		return fmt.Errorf("Error getting contract data: %s", err)
	}

	data, err := hex.DecodeString(resp.Data)
	if err != nil {
		return fmt.Errorf("Error decoding contract data: %s", err)
	}

	values, err := method.Outputs.UnpackValues(data)
	if err != nil {
		return fmt.Errorf("Error unpacking contract data: %s", err)
	}

	log.Infof("Method: %s", method.Name)
	for i, o := range method.Outputs {
		log.Infof("%s (%s): %v\n", o.Name, o.Type, values[i])
	}

	return nil
}
