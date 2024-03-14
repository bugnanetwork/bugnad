package client

import (
	"context"
	"time"

	"github.com/bugnanetwork/bugnad/cmd/bugnawallet/daemon/server"

	"github.com/pkg/errors"

	"github.com/bugnanetwork/bugnad/cmd/bugnawallet/daemon/pb"
	"google.golang.org/grpc"
)

// Connect connects to the bugnawalletd server, and returns the client instance
func Connect(address string) (pb.BugnawalletdClient, func(), error) {
	// Connection is local, so 1 second timeout is sufficient
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(server.MaxDaemonSendMsgSize)))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, nil, errors.New("bugnawallet daemon is not running, start it with `bugnawallet start-daemon`")
		}
		return nil, nil, err
	}

	return pb.NewBugnawalletdClient(conn), func() {
		conn.Close()
	}, nil
}
