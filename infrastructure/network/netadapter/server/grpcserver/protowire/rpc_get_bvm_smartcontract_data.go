package protowire

import (
	"github.com/bugnanetwork/bugnad/app/appmessage"
	"github.com/pkg/errors"
)

func (x *BugnadMessage_GetBvmSmartContractDataRequest) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "BugnadMessage_GetBvmSmartContractDataRequest is nil")
	}
	return x.GetBvmSmartContractDataRequest.toAppMessage()
}

func (x *BugnadMessage_GetBvmSmartContractDataRequest) fromAppMessage(message *appmessage.GetBvmSmartContractDataRequestMessage) error {
	x.GetBvmSmartContractDataRequest = &GetBvmSmartContractDataRequestMessage{
		Address: message.Address,
		Input:   message.InputData,
	}
	return nil
}

func (x *GetBvmSmartContractDataRequestMessage) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "GetBvmSmartContractDataRequestMessage is nil")
	}
	return &appmessage.GetBvmSmartContractDataRequestMessage{
		Address:   x.Address,
		InputData: x.Input,
	}, nil
}

func (x *BugnadMessage_GetBvmSmartContractDataResponse) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "BugnadMessage_GetBvmSmartContractDataResponse is nil")
	}
	return x.GetBvmSmartContractDataResponse.toAppMessage()
}

func (x *BugnadMessage_GetBvmSmartContractDataResponse) fromAppMessage(message *appmessage.GetBvmSmartContractDataResponseMessage) error {
	var err *RPCError
	if message.Error != nil {
		err = &RPCError{Message: message.Error.Message}
	}

	x.GetBvmSmartContractDataResponse = &GetBvmSmartContractDataResponseMessage{
		Data:  message.Data,
		Error: err,
	}
	return nil
}

func (x *GetBvmSmartContractDataResponseMessage) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "GetBvmSmartContractDataResponseMessage is nil")
	}
	rpcErr, err := x.Error.toAppMessage()
	// Error is an optional field
	if err != nil && !errors.Is(err, errorNil) {
		return nil, err
	}
	return &appmessage.GetBvmSmartContractDataResponseMessage{
		Data:  x.Data,
		Error: rpcErr,
	}, nil
}
