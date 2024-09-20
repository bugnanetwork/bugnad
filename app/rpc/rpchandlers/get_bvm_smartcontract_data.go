package rpchandlers

import (
	"encoding/hex"

	"github.com/bugnanetwork/bugnad/app/appmessage"
	"github.com/bugnanetwork/bugnad/app/rpc/rpccontext"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/txscript"
	"github.com/bugnanetwork/bugnad/infrastructure/network/netadapter/router"
	"github.com/bugnanetwork/bugnad/util"
)

// HandleGetBvmSmartContractData handles the respectively named RPC command
func HandleGetBvmSmartContractData(context *rpccontext.Context, _ *router.Router, request appmessage.Message) (appmessage.Message, error) {
	req := request.(*appmessage.GetBvmSmartContractDataRequestMessage)
	addr, err := util.DecodeAddress(req.Address, context.Config.NetParams().Prefix)

	response := appmessage.NewGetBvmSmartContractDataResponseMessage()
	if err != nil {
		response.Error = &appmessage.RPCError{
			Message: err.Error(),
		}
		return response, nil
	}

	contractAddr, err := txscript.PayToAddrScript(addr)
	if err != nil {
		response.Error = &appmessage.RPCError{
			Message: err.Error(),
		}
		return response, nil
	}

	inputData, err := hex.DecodeString(req.InputData)
	if err != nil {
		response.Error = &appmessage.RPCError{
			Message: err.Error(),
		}
		return response, nil
	}

	ret, err := context.Domain.Consensus().GetBvmSmartContractData(contractAddr, inputData)
	if err != nil {
		response.Error = &appmessage.RPCError{
			Message: err.Error(),
		}
		return response, nil
	}

	response.Data = hex.EncodeToString(ret)
	return response, nil
}
