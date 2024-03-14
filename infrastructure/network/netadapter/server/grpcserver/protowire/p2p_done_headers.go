package protowire

import (
	"github.com/bugnanetwork/bugnad/app/appmessage"
	"github.com/pkg/errors"
)

func (x *BugnadMessage_DoneHeaders) toAppMessage() (appmessage.Message, error) {
	if x == nil {
		return nil, errors.Wrapf(errorNil, "BugnadMessage_DoneHeaders is nil")
	}
	return &appmessage.MsgDoneHeaders{}, nil
}

func (x *BugnadMessage_DoneHeaders) fromAppMessage(_ *appmessage.MsgDoneHeaders) error {
	return nil
}
