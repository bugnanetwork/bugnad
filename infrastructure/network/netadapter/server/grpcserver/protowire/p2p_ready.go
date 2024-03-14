package protowire

import (
	"github.com/bugnanetwork/bugnad/app/appmessage"
	"github.com/pkg/errors"
)

func (x *BugnadMessage_Ready) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "BugnadMessage_Ready is nil")
	}
	return &appmessage.MsgReady{}, nil
}

func (x *BugnadMessage_Ready) fromAppMessage(_ *appmessage.MsgReady) error {
	return nil
}
