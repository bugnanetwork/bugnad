package protowire

import (
	"github.com/bugnanetwork/bugnad/app/appmessage"
	"github.com/pkg/errors"
)

func (x *BugnadMessage_Verack) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "BugnadMessage_Verack is nil")
	}
	return &appmessage.MsgVerAck{}, nil
}

func (x *BugnadMessage_Verack) fromAppMessage(_ *appmessage.MsgVerAck) error {
	return nil
}
