package rpcclient

import "github.com/bugnanetwork/bugnad/app/appmessage"

// GetBlock sends an RPC request respective to the function's name and returns the RPC server's response
func (c *RPCClient) GetBvmSmartContractData(contractAddr, inputData string) (
	*appmessage.GetBvmSmartContractDataResponseMessage, error) {

	err := c.rpcRouter.outgoingRoute().Enqueue(
		appmessage.NewGetBvmSmartContractDataRequestMessage(contractAddr, inputData))
	if err != nil {
		return nil, err
	}
	response, err := c.route(appmessage.CmdGetBvmSmartContractDataResponseMessage).DequeueWithTimeout(c.timeout)
	if err != nil {
		return nil, err
	}
	resp := response.(*appmessage.GetBvmSmartContractDataResponseMessage)
	if resp.Error != nil {
		return nil, c.convertRPCError(resp.Error)
	}
	return resp, nil
}
