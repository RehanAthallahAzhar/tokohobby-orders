package account

import (
	"fmt"

	accountpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/account"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AccountClient struct {
	Service accountpb.AccountServiceClient
	Conn    *grpc.ClientConn
}

func NewAccountClient(grpcServerAddress string) (*AccountClient, error) {
	conn, err := grpc.NewClient(grpcServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("can't connect to gRPC server: %v", err)
	}

	serviceClient := accountpb.NewAccountServiceClient(conn)

	return &AccountClient{
		Service: serviceClient,
		Conn:    conn,
	}, nil
}

func (c *AccountClient) Close() {
	c.Conn.Close()
}
