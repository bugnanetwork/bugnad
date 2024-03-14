package protowire

import (
	"github.com/bugnanetwork/bugnad/app/appmessage"
	"github.com/pkg/errors"
)

func (x *BugnadMessage_ShutDownRequest) toAppMessage() (appmessage.Message, error) {
	return &appmessage.ShutDownRequestMessage{}, nil
}

func (x *BugnadMessage_ShutDownRequest) fromAppMessage(_ *appmessage.ShutDownRequestMessage) error {
	x.ShutDownRequest = &ShutDownRequestMessage{}
	return nil
}

func (x *BugnadMessage_ShutDownResponse) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "BugnadMessage_ShutDownResponse is nil")
	}
	return x.ShutDownResponse.toAppMessage()
}

func (x *BugnadMessage_ShutDownResponse) fromAppMessage(message *appmessage.ShutDownResponseMessage) error {
	var err *RPCError
	if message.Error != nil {
		err = &RPCError{Message: message.Error.Message}
	}
	x.ShutDownResponse = &ShutDownResponseMessage{
		Error: err,
	}
	return nil
}

func (x *ShutDownResponseMessage) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "ShutDownResponseMessage is nil")
	}
	rpcErr, err := x.Error.toAppMessage()
	// Error is an optional field
	if err != nil && !errors.Is(err, errorNil) {
		return nil, err
	}
	return &appmessage.ShutDownResponseMessage{
		Error: rpcErr,
	}, nil
}
