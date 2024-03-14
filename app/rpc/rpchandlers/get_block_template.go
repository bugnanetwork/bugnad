package rpchandlers

import (
	"time"

	"github.com/bugnanetwork/bugnad/app/appmessage"
	"github.com/bugnanetwork/bugnad/app/rpc/rpccontext"
	"github.com/bugnanetwork/bugnad/domain/consensus/model/externalapi"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/transactionhelper"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/txscript"
	"github.com/bugnanetwork/bugnad/infrastructure/network/netadapter/router"
	"github.com/bugnanetwork/bugnad/util"
	"github.com/bugnanetwork/bugnad/version"
)

// HandleGetBlockTemplate handles the respectively named RPC command
func HandleGetBlockTemplate(context *rpccontext.Context, _ *router.Router, request appmessage.Message) (appmessage.Message, error) {
	if time.Now().Before(context.Config.LaunchDate) {
		errorMessage := &appmessage.GetBlockTemplateResponseMessage{}
		errorMessage.Error = appmessage.RPCErrorf("no block template available before the launch date of the network")
		return errorMessage, nil

	}

	getBlockTemplateRequest := request.(*appmessage.GetBlockTemplateRequestMessage)

	payAddress, err := util.DecodeAddress(getBlockTemplateRequest.PayAddress, context.Config.ActiveNetParams.Prefix)
	if err != nil {
		errorMessage := &appmessage.GetBlockTemplateResponseMessage{}
		errorMessage.Error = appmessage.RPCErrorf("Could not decode address: %s", err)
		return errorMessage, nil
	}

	scriptPublicKey, err := txscript.PayToAddrScript(payAddress)
	if err != nil {
		return nil, err
	}

	coinbaseData := &externalapi.DomainCoinbaseData{ScriptPublicKey: scriptPublicKey, ExtraData: []byte(version.Version() + "/" + getBlockTemplateRequest.ExtraData)}

	templateBlock, isNearlySynced, err := context.Domain.MiningManager().GetBlockTemplate(coinbaseData)
	if err != nil {
		return nil, err
	}

	if uint64(len(templateBlock.Transactions[transactionhelper.CoinbaseTransactionIndex].Payload)) > context.Config.NetParams().MaxCoinbasePayloadLength {
		errorMessage := &appmessage.GetBlockTemplateResponseMessage{}
		errorMessage.Error = appmessage.RPCErrorf("Coinbase payload is above max length (%d). Try to shorten the extra data.", context.Config.NetParams().MaxCoinbasePayloadLength)
		return errorMessage, nil
	}

	rpcBlock := appmessage.DomainBlockToRPCBlock(templateBlock)

	return appmessage.NewGetBlockTemplateResponseMessage(rpcBlock, context.ProtocolManager.Context().HasPeers() && isNearlySynced), nil
}
