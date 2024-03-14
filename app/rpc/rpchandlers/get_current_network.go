package rpchandlers

import (
	"github.com/bugnanetwork/bugnad/app/appmessage"
	"github.com/bugnanetwork/bugnad/app/rpc/rpccontext"
	"github.com/bugnanetwork/bugnad/infrastructure/network/netadapter/router"
)

// HandleGetCurrentNetwork handles the respectively named RPC command
func HandleGetCurrentNetwork(context *rpccontext.Context, _ *router.Router, _ appmessage.Message) (appmessage.Message, error) {
	response := appmessage.NewGetCurrentNetworkResponseMessage(context.Config.ActiveNetParams.Net.String())
	return response, nil
}
