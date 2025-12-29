package account

import (
	"context"
	"fmt"
	"log"
	"time"

	authpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type AuthClient struct {
	service authpb.AuthServiceClient
	conn    *grpc.ClientConn
}

func NewAuthClient(grpcServerAddress string) (*AuthClient, error) {
	conn, err := grpc.NewClient(grpcServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("can't connect to gRPC server: %v", err)
	}

	serviceClient := authpb.NewAuthServiceClient(conn)
	return &AuthClient{
		service: serviceClient,
		conn:    conn,
	}, nil
}

func NewAuthClientFromService(serviceClient authpb.AuthServiceClient, conn *grpc.ClientConn) *AuthClient {
	return &AuthClient{
		service: serviceClient,
		conn:    conn,
	}
}

func (rc *AuthClient) Close() {
	if rc.conn != nil {
		err := rc.conn.Close()
		if err != nil {
			log.Printf("Failed to close gRPC connection: %v", err)
		}
	}
}

func (c *AuthClient) ValidateToken(token string) (isValid bool, userID string, username string, role string, errorMessage string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &authpb.ValidateTokenRequest{Token: token}
	res, err := c.service.ValidateToken(ctx, req)
	if err != nil {

		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.Unauthenticated {
				return false, "", "", "", st.Message(), st.Err()
			}
		}
		return false, "", "", "", "Kesalahan internal server", err
	}

	if !res.IsValid {
		log.Printf("Token tidak valid: %s", res.GetErrorMessage())
	}

	return res.GetIsValid(), res.GetUserId(), res.GetUsername(), res.GetRole(), res.GetErrorMessage(), nil
}
