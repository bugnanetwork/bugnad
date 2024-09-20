package appmessage

// GetBvmSmartContractDataRequestMessage is an appmessage corresponding to
// its respective RPC message
type GetBvmSmartContractDataRequestMessage struct {
	baseMessage
	Address   string
	InputData string
}

// Command returns the protocol command string for the message
func (msg *GetBvmSmartContractDataRequestMessage) Command() MessageCommand {
	return CmdGetBvmSmartContractDataRequestMessage
}

// NewGetBvmSmartContractDataRequestMessage returns a instance of the message
func NewGetBvmSmartContractDataRequestMessage(address string, inputData string) *GetBvmSmartContractDataRequestMessage {
	return &GetBvmSmartContractDataRequestMessage{
		Address:   address,
		InputData: inputData,
	}
}

// GetBvmSmartContractDataResponseMessage is an appmessage corresponding to
// its respective RPC message
type GetBvmSmartContractDataResponseMessage struct {
	baseMessage
	Data string

	Error *RPCError
}

// Command returns the protocol command string for the message
func (msg *GetBvmSmartContractDataResponseMessage) Command() MessageCommand {
	return CmdGetBvmSmartContractDataResponseMessage
}

// NewGetBvmSmartContractDataResponseMessage returns a instance of the message
func NewGetBvmSmartContractDataResponseMessage() *GetBvmSmartContractDataResponseMessage {
	return &GetBvmSmartContractDataResponseMessage{}
}
