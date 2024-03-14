package server

import (
	"context"

	"github.com/bugnanetwork/bugnad/cmd/bugnawallet/daemon/pb"
	"github.com/bugnanetwork/bugnad/cmd/bugnawallet/libbugnawallet"
)

func (s *server) Sign(_ context.Context, request *pb.SignRequest) (*pb.SignResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	signedTransactions, err := s.signTransactions(request.UnsignedTransactions, request.Password)
	if err != nil {
		return nil, err
	}
	return &pb.SignResponse{SignedTransactions: signedTransactions}, nil
}

func (s *server) signTransactions(unsignedTransactions [][]byte, password string) ([][]byte, error) {
	mnemonics, err := s.keysFile.DecryptMnemonics(password)
	if err != nil {
		return nil, err
	}
	signedTransactions := make([][]byte, len(unsignedTransactions))
	for i, unsignedTransaction := range unsignedTransactions {
		signedTransaction, err := libbugnawallet.Sign(s.params, mnemonics, unsignedTransaction, s.keysFile.ECDSA)
		if err != nil {
			return nil, err
		}
		signedTransactions[i] = signedTransaction
	}
	return signedTransactions, nil
}
