package productclient

import (
	"fmt"

	productpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/product"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProductClient struct {
	Service productpb.ProductServiceClient
	Conn    *grpc.ClientConn
}

func NewProductClient(grpcServerAddress string) (*ProductClient, error) {
	conn, err := grpc.NewClient(grpcServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("can't connect to gRPC server: %v", err)
	}

	serviceClient := productpb.NewProductServiceClient(conn)

	return &ProductClient{
		Service: serviceClient,
		Conn:    conn,
	}, nil
}

func (c *ProductClient) Close() {
	c.Conn.Close()
}
