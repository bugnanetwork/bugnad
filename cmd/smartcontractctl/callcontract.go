package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/bugnanetwork/bugnad/domain/bvm/abi"
	"github.com/bugnanetwork/bugnad/domain/bvm/vm"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/txscript"
	"github.com/bugnanetwork/bugnad/infrastructure/network/netadapter/server/grpcserver/protowire"
	"github.com/bugnanetwork/bugnad/infrastructure/network/rpcclient"
	"github.com/bugnanetwork/bugnad/util"
	"github.com/kaspanet/go-secp256k1"
	"google.golang.org/protobuf/encoding/protojson"
)

const TrueStr = "true"
const FalseStr = "false"

func callSmartContract(cfg *callContractConfig) error {
	client, addr, myKeyPair, err := newCaller(&cfg.configFlags)
	if err != nil {
		return fmt.Errorf("Error parsing private key: %s", err)
	}

	contractAddress, err := util.DecodeAddress(cfg.ContractAddress, cfg.configFlags.ActiveNetParams.Prefix)
	if err != nil {
		return fmt.Errorf("Error decoding contract address: %s", err)
	}

	inputData, _ := hex.DecodeString(cfg.RawInput)
	if len(inputData) == 0 {
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

		inputData, err = getMethodPayload(ABI, append([]string{cfg.Method}, cfg.Args...))
		if err != nil {
			return fmt.Errorf("Error getting method payload: %s", err)
		}
	}

	input, err := NewCallSmartContractinput(addr, contractAddress, inputData, cfg.configFlags.ActiveNetParams.Prefix)
	if err != nil {
		return fmt.Errorf("Error creating call smart contract input: %s", err)
	}

	return submit(&cfg.configFlags, client, myKeyPair, addr, input)
}

func newCaller(cfg *configFlags) (*rpcclient.RPCClient, util.Address, *secp256k1.SchnorrKeyPair, error) {
	myKeyPair, addr, err := parsePrivateKeyInKeyPair(cfg.PrivateKey, cfg.ActiveNetParams.Prefix)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error parsing private key: %s", err)
	}

	rpcAddress, err := cfg.NetParams().NormalizeRPCServerAddress(cfg.RPCServer)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error parsing RPC server address: %s", err)
	}

	//RPC client activation (to communicate with Kaspad)
	client, err := rpcclient.NewRPCClient(rpcAddress)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error connecting to the RPC server: %s", err)
	}

	client.SetTimeout(5 * time.Minute)

	return client, addr, myKeyPair, nil
}

func submit(
	cfg *configFlags,
	client *rpcclient.RPCClient,
	myKeyPair *secp256k1.SchnorrKeyPair,
	addr util.Address,
	input SmartContractInput,
) error {
	resp, err := client.GetBlockDAGInfo()
	if err != nil {
		return fmt.Errorf("Error getting block DAG info: %s", err)
	}
	currentBlockHash := resp.VirtualParentHashes[0]

	transactionID, err := initiateAddressScript(cfg, client, myKeyPair, addr, input)
	if err != nil {
		return fmt.Errorf("Error initiating address script: %s", err)
	}

	log.Info("Waiting for network to call the contract....")
	log.Infof("Contract P2SH address: %s", input.Address().String())
	log.Infof("Transaction ID: %s", transactionID)

	// Redeem from P2SH contract
	txID, err := redeemContract(client, transactionID, myKeyPair, addr, input)
	if err != nil {
		return fmt.Errorf("Error redeeming contract: %s", err)
	}

	for {
		select {
		case <-time.After(10 * time.Second):
			return fmt.Errorf("Contract not yet called.")
		default:
			blocks, err := client.GetBlocks(currentBlockHash, true, true)
			if err != nil {
				return fmt.Errorf("Error getting blocks: %s", err)
			}
			rpcResponse, err := protowire.FromAppMessage(blocks)
			if err != nil {
				return fmt.Errorf("Error converting to app message: %s", err)
			}

			for _, block := range rpcResponse.GetGetBlocksResponse().Blocks {
				for _, txn := range block.Transactions {
					if txn.VerboseData.TransactionId == txID {
						log.Infof("Contract called successfully. BlockHash: %s Transaction ID: %s", block.VerboseData.Hash, txID)

						marshalOptions := &protojson.MarshalOptions{}
						marshalOptions.Indent = "    "
						marshalOptions.EmitUnpopulated = true

						fmt.Println(marshalOptions.Format(txn))

						return nil
					}
				}
			}
		}
	}
}

type CallSmartContractinput struct {
	addr            util.Address
	contractAddress util.Address
	secretContract  []byte
	inputData       []byte
}

func NewCallSmartContractinput(
	caller, contractAddress util.Address,
	inputData []byte,
	prefix util.Bech32Prefix,
) (*CallSmartContractinput, error) {
	builder := txscript.NewScriptBuilder()

	// Verify their signature is being used to redeem the output
	builder.AddData(caller.ScriptAddress())
	builder.AddOp(txscript.OpCheckSig)

	for i := 0; i < len(inputData); i += txscript.MaxScriptElementSize {
		builder.AddOp(txscript.OpDrop)
	}

	builder.AddOp(txscript.Op2Drop)
	builder.AddOp(txscript.OpDrop)

	secretContract, err := builder.Script()
	if err != nil {
		return nil, err
	}

	contractP2SHaddress, err := util.NewAddressScriptHash(secretContract, prefix)
	if err != nil {
		return nil, err
	}

	return &CallSmartContractinput{
		addr:            contractP2SHaddress,
		contractAddress: contractAddress,
		secretContract:  secretContract,
		inputData:       inputData,
	}, nil
}

func (d *CallSmartContractinput) Address() util.Address {
	return d.addr
}

func (d *CallSmartContractinput) RedeemScript(signature []byte) ([]byte, error) {
	builder := txscript.NewScriptBuilder()

	builder.AddData([]byte("bugna_script"))
	builder.AddData([]byte("interact"))

	scriptPubKey, _ := txscript.PayToAddrScript(d.contractAddress)
	evmAddr := vm.ScriptPubkeyToAddress(scriptPubKey)
	builder.AddData(evmAddr.Bytes())

	for i := 0; i < len(d.inputData); i += txscript.MaxScriptElementSize {
		start := i
		end := i + txscript.MaxScriptElementSize
		if end > len(d.inputData) {
			end = len(d.inputData)
		}

		d := d.inputData[start:end]
		builder.AddData(d)
	}

	builder.AddData(signature)
	builder.AddData(d.secretContract)

	return builder.Script()
}

func getMethodPayload(newAbi abi.ABI, args []string) ([]byte, error) {
	method := newAbi.Methods[args[0]]
	abiArgs := []interface{}{}
	for i, input := range method.Inputs {
		idx := i + 1
		if idx >= len(args) {
			return nil, errors.New("not enough arguments")
		}
		var arg interface{}
		var err error
		switch input.Type.T {
		case abi.IntTy:
			if input.Type.Size > 64 {
				bi, success := new(big.Int).SetString(args[idx], 10)
				if !success {
					return nil, errors.New("invalid big.Int")
				} else {
					arg = bi
				}
			} else {
				val, e := strconv.ParseInt(args[idx], 10, 64)
				err = e
				switch input.Type.Size {
				case 8:
					if val > math.MaxInt8 {
						return nil, errors.New("int8 overflow")
					}
					arg = int8(val)
				case 16:
					if val > math.MaxInt16 {
						return nil, errors.New("int16 overflow")
					}
					arg = int16(val)
				case 32:
					if val > math.MaxInt32 {
						return nil, errors.New("int32 overflow")
					}
					arg = int32(val)
				case 64:
					arg = val
				}
			}
		case abi.UintTy:
			if input.Type.Size > 64 {
				bi, success := new(big.Int).SetString(args[idx], 10)
				if !success {
					return nil, errors.New("invalid big.Int")
				} else {
					arg = bi
				}
			} else {
				val, e := strconv.ParseUint(args[idx], 10, 64)
				err = e
				switch input.Type.Size {
				case 8:
					if val > math.MaxUint8 {
						return nil, errors.New("uint8 overflow")
					}
					arg = uint8(val)
				case 16:
					if val > math.MaxUint16 {
						return nil, errors.New("uint16 overflow")
					}
					arg = uint16(val)
				case 32:
					if val > math.MaxUint32 {
						return nil, errors.New("uint32 overflow")
					}
					arg = uint32(val)
				case 64:
					arg = val
				}
			}
		case abi.BoolTy:
			if args[idx] != TrueStr && args[idx] != FalseStr {
				return nil, fmt.Errorf("boolean argument has to be either \"%s\" or \"%s\"", TrueStr, FalseStr)
			} else {
				arg = args[idx] == TrueStr
			}
		case abi.StringTy:
			arg = args[idx]
		case abi.AddressTy:
			arg = vm.HexToAddress(args[idx])
		default:
			return nil, errors.New("argument type not supported yet")
		}
		if err != nil {
			return nil, err
		}
		abiArgs = append(abiArgs, arg)
	}

	return newAbi.Pack(args[0], abiArgs...)
}
